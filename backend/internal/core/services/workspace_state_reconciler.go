package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"solv-backend/internal/core/domain"
)

// ReconcilerUnknownAlertThreshold es la cantidad de ciclos consecutivos (de 30 s)
// que un workspace puede permanecer sin verificar antes de emitir la alerta de
// auditoria. 5 ciclos ~= 2.5 minutos de daemon caido.
const ReconcilerUnknownAlertThreshold = 5

// WorkspaceStateReconciler cierra el lazo Docker->DB que faltaba (D2, ADR-049):
// el zombie collector solo detectaba contenedores sin registro en la base; este
// reconcilier detecta lo contrario, registros sin contenedor, y distingue el
// por que con dos estados:
//
//   - unknown: el daemon no respondio durante la verificacion. No se borra nada;
//     se marca el estado para que el panel lo vea, el registro reintenta el
//     siguiente ciclo y el cupo de max_containers se retiene (GetActiveWorkspaces
//     incluye 'unknown').
//   - orphaned: el daemon respondio y el contenedor ya no existe. Libera el cupo
//     y deja el registro para que el alumno lo recree (restart).
//
// Tambien resuelve los contenedores que existen pero salieron del daemon: un
// exit 137/OOM va por la via canonica de IncrementOOMStrike (penalizacion de 3
// strikes); una salida limpia elimina el contenedor y marca hibernated, cosa que
// el worker de QoS (cada 15 min) no hacia con contenedores ya fuera de marcha.
type WorkspaceStateReconciler struct {
	repo         domain.WorkspaceRepository
	orchestrator domain.WorkspaceOrchestrator
	audit        domain.AuditLogRepository

	// failedCycles lleva cuantos ciclos consecutivos lleva cada workspace en
	// unknown para emitir la alerta una sola vez al cruzar el umbral.
	failedCycles map[string]int

	mu sync.Mutex
}

func NewWorkspaceStateReconciler(repo domain.WorkspaceRepository, orchestrator domain.WorkspaceOrchestrator) *WorkspaceStateReconciler {
	return &WorkspaceStateReconciler{
		repo:         repo,
		orchestrator: orchestrator,
		failedCycles: make(map[string]int),
	}
}

// SetAuditRepository inyecta el registro de auditoria. Sin el, las transiciones
// solo se registran por log (no bloquean la reconciliacion).
func (r *WorkspaceStateReconciler) SetAuditRepository(audit domain.AuditLogRepository) {
	r.audit = audit
}

// Reconcile verifica el estado real de cada workspace activo contra el daemon.
// Es seguro ejecutarlo desde el ciclo del zombie collector (30 s): se apoya en
// la mutex de ese worker y no lanza nada en paralelo.
func (r *WorkspaceStateReconciler) Reconcile(ctx context.Context) {
	activeWorkspaces, err := r.repo.GetActiveWorkspaces(ctx)
	if err != nil {
		log.Printf("[State Reconciler] Error reading active workspaces: %v", err)
		return
	}

	// Solo los registros con contenedor declarado se pueden verificar.
	scope := make([]*domain.WorkspaceInstance, 0, len(activeWorkspaces))
	for _, ws := range activeWorkspaces {
		if ws != nil && ws.ContainerID != nil && *ws.ContainerID != "" {
			scope = append(scope, ws)
		}
	}
	if len(scope) == 0 {
		return
	}

	managed, err := r.orchestrator.ListAllManagedContainers(ctx)
	if err != nil {
		// Docker caido: nada destruye. Cada workspace pasa a unknown (si no lo
		// estaba) y suma un ciclo fallido para la alerta de persistencia.
		log.Printf("[State Reconciler] Docker daemon unreachable: marking %d workspaces unknown", len(scope))
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
			r.transitionTo(ctx, ws, domain.WorkspaceStatusOrphaned, "WORKSPACE_RECONCILED_ORPHANED")
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
					log.Printf("[State Reconciler] Workspace %s container exited with OOM (exit %d): oom_killed", ws.ID, metrics.ExitCode)
				}
			} else {
				// Salida limpia: el contenedor quedo fuera de marcha. Se retira y se
				// marca hibernated para que el alumno lo reactive desde la vista.
				if stopErr := r.orchestrator.StopAndRemoveContainer(ctx, cid); stopErr == nil {
					r.transitionTo(ctx, ws, domain.WorkspaceStatusHibernated, "WORKSPACE_RECONCILED_EXITED")
				} else {
					log.Printf("[State Reconciler] Could not remove exited container %s: %v (retrying next cycle)", cid, stopErr)
				}
			}
			continue
		}

		// El contenedor existe y esta operativo. Si el registro venia de unknown,
		// el daemon se recupero: se restaura running y se limpia el contador.
		if ws.Status == domain.WorkspaceStatusUnknown {
			r.transitionTo(ctx, ws, domain.WorkspaceStatusRunning, "WORKSPACE_RECONCILED_RECOVERED")
			r.resetCounter(ws.ID)
		}
	}
}

// markUnknown marca el estado sin verificar (si hace falta), acumula ciclos
// fallidos y alerta por auditoria la primera vez que se cruza el umbral.
func (r *WorkspaceStateReconciler) markUnknown(ctx context.Context, ws *domain.WorkspaceInstance) {
	if ws.Status != domain.WorkspaceStatusUnknown {
		if err := r.repo.UpdateStatus(ctx, ws.ID, domain.WorkspaceStatusUnknown); err != nil {
			log.Printf("[State Reconciler] Error marking workspace %s unknown: %v", ws.ID, err)
			return
		}
	}

	r.mu.Lock()
	r.failedCycles[ws.ID]++
	cycles := r.failedCycles[ws.ID]
	alert := cycles == ReconcilerUnknownAlertThreshold
	r.mu.Unlock()

	if alert {
		log.Printf("[State Reconciler] Workspace %s unknown for %d cycles, notifying audit", ws.ID, cycles)
		r.auditTransition(ctx, ws, domain.WorkspaceStatusUnknown, "WORKSPACE_RECONCILED_UNKNOWN_PERSISTENT")
	}
}

// transitionTo persiste el nuevo estado y registra la transicion en auditoria.
func (r *WorkspaceStateReconciler) transitionTo(ctx context.Context, ws *domain.WorkspaceInstance, status, action string) {
	if ws.Status == status {
		return
	}
	if err := r.repo.UpdateStatus(ctx, ws.ID, status); err != nil {
		log.Printf("[State Reconciler] Error updating workspace %s to %s: %v", ws.ID, status, err)
		return
	}
	log.Printf("[State Reconciler] Workspace %s: %s -> %s", ws.ID, ws.Status, status)
	r.auditTransition(ctx, ws, status, action)
}

func (r *WorkspaceStateReconciler) resetCounter(workspaceID string) {
	r.mu.Lock()
	delete(r.failedCycles, workspaceID)
	r.mu.Unlock()
}

func (r *WorkspaceStateReconciler) auditTransition(ctx context.Context, ws *domain.WorkspaceInstance, status, action string) {
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
		ResourceType: "workspace",
		ResourceID:   &wsID,
		StatusCode:   200,
		Metadata:     meta,
	})
}