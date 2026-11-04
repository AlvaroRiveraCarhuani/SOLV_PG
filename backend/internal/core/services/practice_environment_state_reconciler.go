package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"solv-backend/internal/core/domain"
)

// ReconcilerUnknownAlertThreshold es la cantidad de ciclos consecutivos (de 30 s)
// que un environment puede permanecer sin verificar antes de emitir la alerta de
// auditoria. 5 ciclos ~= 2.5 minutos de daemon caido.
const ReconcilerUnknownAlertThreshold = 5

// PracticeEnvironmentStateReconciler cierra el lazo Docker->DB que faltaba (D2, ADR-049):
// el zombie collector solo detectaba contenedores sin registro en la base; este
// reconcilier detecta lo contrario, registros sin contenedor, y distingue el
// por que con dos estados:
//
//   - unknown: el daemon no respondio durante la verificacion. No se borra nada;
//     se marca el estado para que el panel lo vea, el registro reintenta el
//     siguiente ciclo y el cupo de max_containers se retiene (GetActivePracticeEnvironments
//     incluye 'unknown').
//   - orphaned: el daemon respondio y el contenedor ya no existe. Libera el cupo
//     y deja el registro para que el alumno lo recree (restart).
//
// Tambien resuelve los contenedores que existen pero salieron del daemon: un
// exit 137/OOM va por la via canonica de IncrementOOMStrike (penalizacion de 3
// strikes); una salida limpia elimina el contenedor y marca hibernated, cosa que
// el worker de QoS (cada 15 min) no hacia con contenedores ya fuera de marcha.
type PracticeEnvironmentStateReconciler struct {
	repo         domain.PracticeEnvironmentRepository
	orchestrator domain.PracticeEnvironmentOrchestrator
	audit        domain.AuditLogRepository

	// failedCycles lleva cuantos ciclos consecutivos lleva cada environment en
	// unknown para emitir la alerta una sola vez al cruzar el umbral.
	failedCycles map[string]int

	mu sync.Mutex
}

func NewPracticeEnvironmentStateReconciler(repo domain.PracticeEnvironmentRepository, orchestrator domain.PracticeEnvironmentOrchestrator) *PracticeEnvironmentStateReconciler {
	return &PracticeEnvironmentStateReconciler{
		repo:         repo,
		orchestrator: orchestrator,
		failedCycles: make(map[string]int),
	}
}

// SetAuditRepository inyecta el registro de auditoria. Sin el, las transiciones
// solo se registran por log (no bloquean la reconciliacion).
func (r *PracticeEnvironmentStateReconciler) SetAuditRepository(audit domain.AuditLogRepository) {
	r.audit = audit
}

// Reconcile verifica el estado real de cada environment activo contra el daemon.
// Es seguro ejecutarlo desde el ciclo del zombie collector (30 s): se apoya en
// la mutex de ese worker y no lanza nada en paralelo.
func (r *PracticeEnvironmentStateReconciler) Reconcile(ctx context.Context) {
	activePracticeEnvironments, err := r.repo.GetActivePracticeEnvironments(ctx)
	if err != nil {
		log.Printf("[State Reconciler] Error reading active environments: %v", err)
		return
	}

	// Solo los registros con contenedor declarado se pueden verificar.
	scope := make([]*domain.PracticeEnvironment, 0, len(activePracticeEnvironments))
	for _, ws := range activePracticeEnvironments {
		if ws != nil && ws.ContainerID != nil && *ws.ContainerID != "" {
			scope = append(scope, ws)
		}
	}
	if len(scope) == 0 {
		return
	}

	managed, err := r.orchestrator.ListAllManagedContainers(ctx)
	if err != nil {
		// Docker caido: nada destruye. Cada environment pasa a unknown (si no lo
		// estaba) y suma un ciclo fallido para la alerta de persistencia.
		log.Printf("[State Reconciler] Docker daemon unreachable: marking %d environments unknown", len(scope))
		for _, ws := range scope {
			r.markUnknown(ctx, ws)
		}
		return
	}

	exists := make(map[string]bool, len(managed))
	for _, cid := range managed {
		exists[cid] = true
	}

	for _, ws := range scope {
		cid := *ws.ContainerID
		if !exists[cid] {
			r.transitionTo(ctx, ws, domain.PracticeEnvironmentStatusOrphaned, "PRACTICE_ENVIRONMENT_RECONCILED_ORPHANED")
			r.resetCounter(ws.ID)
			continue
		}

		metrics, mErr := r.orchestrator.GetContainerMetrics(ctx, cid)
		if mErr != nil {
			// El contenedor existe pero no se pudo inspeccionar: no hay certeza.
			r.markUnknown(ctx, ws)
			continue
		}

		if metrics != nil && !metrics.IsRunning {
			if metrics.OOMKilled || metrics.ExitCode == 137 {
				// Via canonica: el strike y el estado oom_killed los pone el
				// repositorio, igual que en el worker de QoS (3 strikes = penalizacion).
				if oomErr := r.repo.IncrementOOMStrike(ctx, ws.ID); oomErr == nil {
					r.resetCounter(ws.ID)
					log.Printf("[State Reconciler] PracticeEnvironment %s container exited with OOM (exit %d): oom_killed", ws.ID, metrics.ExitCode)
				}
			} else {
				// Salida limpia: el contenedor quedo fuera de marcha. Se retira y se
				// marca hibernated para que el alumno lo reactive desde la vista.
				if stopErr := r.orchestrator.StopAndRemoveContainer(ctx, cid); stopErr == nil {
					r.transitionTo(ctx, ws, domain.PracticeEnvironmentStatusHibernated, "PRACTICE_ENVIRONMENT_RECONCILED_EXITED")
				} else {
					log.Printf("[State Reconciler] Could not remove exited container %s: %v (retrying next cycle)", cid, stopErr)
				}
			}
			continue
		}

		// El contenedor existe y esta operativo. Si el registro venia de unknown,
		// el daemon se recupero: se restaura running y se limpia el contador.
		if ws.Status == domain.PracticeEnvironmentStatusUnknown {
			r.transitionTo(ctx, ws, domain.PracticeEnvironmentStatusRunning, "PRACTICE_ENVIRONMENT_RECONCILED_RECOVERED")
			r.resetCounter(ws.ID)
		}
	}
}

// markUnknown marca el estado sin verificar (si hace falta), acumula ciclos
// fallidos y alerta por auditoria la primera vez que se cruza el umbral.
func (r *PracticeEnvironmentStateReconciler) markUnknown(ctx context.Context, ws *domain.PracticeEnvironment) {
	if ws.Status != domain.PracticeEnvironmentStatusUnknown {
		if err := r.repo.UpdateStatus(ctx, ws.ID, domain.PracticeEnvironmentStatusUnknown); err != nil {
			log.Printf("[State Reconciler] Error marking environment %s unknown: %v", ws.ID, err)
			return
		}
	}

	r.mu.Lock()
	r.failedCycles[ws.ID]++
	cycles := r.failedCycles[ws.ID]
	alert := cycles == ReconcilerUnknownAlertThreshold
	r.mu.Unlock()

	if alert {
		log.Printf("[State Reconciler] PracticeEnvironment %s unknown for %d cycles, notifying audit", ws.ID, cycles)
		r.auditTransition(ctx, ws, domain.PracticeEnvironmentStatusUnknown, "PRACTICE_ENVIRONMENT_RECONCILED_UNKNOWN_PERSISTENT")
	}
}

// transitionTo persiste el nuevo estado y registra la transicion en auditoria.
func (r *PracticeEnvironmentStateReconciler) transitionTo(ctx context.Context, ws *domain.PracticeEnvironment, status, action string) {
	if ws.Status == status {
		return
	}
	if err := r.repo.UpdateStatus(ctx, ws.ID, status); err != nil {
		log.Printf("[State Reconciler] Error updating environment %s to %s: %v", ws.ID, status, err)
		return
	}
	log.Printf("[State Reconciler] PracticeEnvironment %s: %s -> %s", ws.ID, ws.Status, status)
	r.auditTransition(ctx, ws, status, action)
}

func (r *PracticeEnvironmentStateReconciler) resetCounter(environmentID string) {
	r.mu.Lock()
	delete(r.failedCycles, environmentID)
	r.mu.Unlock()
}

func (r *PracticeEnvironmentStateReconciler) auditTransition(ctx context.Context, ws *domain.PracticeEnvironment, status, action string) {
	if r.audit == nil || ws == nil {
		return
	}
	meta, _ := json.Marshal(map[string]interface{}{
		"from_status": ws.Status,
		"to_status":   status,
	})
	wsID := ws.ID
	_ = r.audit.Create(ctx, &domain.AuditLog{
		TenantID:     ws.TenantID,
		ActorID:      "system",
		Action:       action,
		ResourceType: "environment",
		ResourceID:   &wsID,
		StatusCode:   200,
		Metadata:     meta,
	})
}
