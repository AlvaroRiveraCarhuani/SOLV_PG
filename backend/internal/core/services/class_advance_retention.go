package services

import (
	"context"
	"log"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

const advanceSnapshotPrefix = "class-advances/"

// AdvancePurgeResult resume un barrido. Los fallos no abortan el resto: la fila
// queda sin marcar y el próximo barrido vuelve a intentar ese objeto.
type AdvancePurgeResult struct {
	SnapshotsDeleted int
	SnapshotFailures int
	OrphansDeleted   int
	OrphanFailures   int
}

// PurgeExpiredSnapshots borra los snapshots que ya no pueden aplicarse y los
// objetos que quedaron sin fila por una caída entre la subida y el registro.
// El corte se mide desde el cierre del aula, no desde la emisión: la ventana
// puede reabrirse hasta el máximo y el objeto tiene que seguir ahí.
func (s *ClassAdvanceService) PurgeExpiredSnapshots(ctx context.Context, now time.Time) (AdvancePurgeResult, error) {
	var result AdvancePurgeResult
	if s.snapshots == nil || s.retention == nil {
		return result, ErrAdvanceStorageUnavailable
	}

	cutoff := now.Add(-time.Duration(domain.AdvanceSnapshotRetentionDays) * 24 * time.Hour)
	for batch := 0; batch < domain.AdvanceSnapshotPurgeMaxBatches; batch++ {
		candidates, err := s.retention.ExpiredAdvanceSnapshots(ctx, cutoff, domain.AdvanceSnapshotPurgeBatch)
		if err != nil {
			return result, err
		}
		if len(candidates) == 0 {
			break
		}

		batchFailures := 0
		for _, advance := range candidates {
			// La clave lleva el tenant. Si una fila apunta fuera de su prefijo, no
			// se borra: una inconsistencia no puede tocar el código de otra
			// institución.
			if !isAdvanceSnapshotKeyForTenant(advance.TenantID, advance.SnapshotKey) {
				result.SnapshotFailures++
				batchFailures++
				log.Printf("[Advance Retention] la clave %q no pertenece al tenant del avance %s", advance.SnapshotKey, advance.ID)
				continue
			}
			if err := s.snapshots.Delete(ctx, advance.SnapshotKey); err != nil {
				result.SnapshotFailures++
				batchFailures++
				log.Printf("[Advance Retention] no se pudo borrar el snapshot del avance %s: %v", advance.ID, err)
				continue
			}
			if err := s.retention.MarkSnapshotPurged(ctx, advance.TenantID, advance.ID, now); err != nil {
				result.SnapshotFailures++
				batchFailures++
				log.Printf("[Advance Retention] el snapshot del avance %s se borró pero no se pudo marcar: %v", advance.ID, err)
				continue
			}
			result.SnapshotsDeleted++
		}

		// Si el lote venía lleno y todo salió bien, puede haber más candidatos.
		// Con un fallo se corta para no repetir las mismas filas en el mismo ciclo.
		if len(candidates) < domain.AdvanceSnapshotPurgeBatch || batchFailures > 0 {
			break
		}
	}

	// Los objetos huérfanos no tienen fila que los reclame. Se borran recién a
	// las 24 horas para no tocar una emisión que todavía está en curso.
	lister, ok := s.snapshots.(domain.AdvanceSnapshotLister)
	if !ok {
		return result, nil
	}
	objects, err := lister.List(ctx, advanceSnapshotPrefix)
	if err != nil {
		return result, err
	}
	referenced, err := s.retention.AdvanceSnapshotKeys(ctx)
	if err != nil {
		return result, err
	}
	known := make(map[string]struct{}, len(referenced))
	for _, key := range referenced {
		known[key] = struct{}{}
	}

	orphanCutoff := now.Add(-domain.AdvanceSnapshotOrphanAge)
	for _, object := range objects {
		if object.Key == "" || object.LastModified.IsZero() || !object.LastModified.Before(orphanCutoff) {
			continue
		}
		if _, exists := known[object.Key]; exists {
			continue
		}
		if err := s.snapshots.Delete(ctx, object.Key); err != nil {
			result.OrphanFailures++
			log.Printf("[Advance Retention] no se pudo borrar el snapshot huérfano %s: %v", object.Key, err)
			continue
		}
		result.OrphansDeleted++
	}
	return result, nil
}

// isAdvanceSnapshotKeyForTenant comprueba que la clave viva bajo el prefijo del
// tenant de la fila. La emisión siempre la arma así; la validación protege el
// borrado de una fila corrupta o migrada a mano.
func isAdvanceSnapshotKeyForTenant(tenantID, key string) bool {
	if tenantID == "" {
		return false
	}
	return strings.HasPrefix(key, advanceSnapshotPrefix+tenantID+"/")
}

// AdvanceRetentionWorker corre el barrido al arrancar y después cada 24 horas.
// El primer ciclo recupera lo que venció mientras el backend estaba detenido.
type AdvanceRetentionWorker struct {
	service  *ClassAdvanceService
	interval time.Duration
	now      func() time.Time
}

func NewAdvanceRetentionWorker(service *ClassAdvanceService) *AdvanceRetentionWorker {
	return &AdvanceRetentionWorker{
		service:  service,
		interval: 24 * time.Hour,
		now:      time.Now,
	}
}

// WithInterval ajusta el ciclo del worker para pruebas o despliegues puntuales.
func (w *AdvanceRetentionWorker) WithInterval(interval time.Duration) *AdvanceRetentionWorker {
	if interval > 0 {
		w.interval = interval
	}
	return w
}

// Start ejecuta la purga al arrancar y la repite con el intervalo configurado.
func (w *AdvanceRetentionWorker) Start(ctx context.Context) {
	if w.service == nil {
		return
	}
	log.Printf("[Advance Retention] Worker started (interval=%v)", w.interval)
	w.PurgeOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[Advance Retention] Worker stopped.")
			return
		case <-ticker.C:
			w.PurgeOnce(ctx)
		}
	}
}

// PurgeOnce ejecuta un ciclo y deja el resultado en el log. Un error de consulta
// no tumba al worker: el próximo ciclo vuelve a intentarlo.
func (w *AdvanceRetentionWorker) PurgeOnce(ctx context.Context) {
	if w.service == nil {
		return
	}
	result, err := w.service.PurgeExpiredSnapshots(ctx, w.now())
	if err != nil {
		log.Printf("[Advance Retention] Error en el barrido: %v", err)
		return
	}
	if result.SnapshotsDeleted > 0 || result.SnapshotFailures > 0 || result.OrphansDeleted > 0 || result.OrphanFailures > 0 {
		log.Printf("[Advance Retention] snapshots borrados=%d fallos=%d huérfanos borrados=%d fallos=%d",
			result.SnapshotsDeleted, result.SnapshotFailures, result.OrphansDeleted, result.OrphanFailures)
	}
}
