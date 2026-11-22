package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"solv-backend/internal/core/domain"
)

// DiskUsageSweepInterval es cada cuánto se mide el disco de los entornos
// abiertos. Cinco minutos alcanzan para avisar antes de que un volumen llene el
// nodo, y el barrido mide un volumen por entorno abierto, no todos los guardados.
const DiskUsageSweepInterval = 5 * time.Minute

// SweepDiskUsage mide el volumen de cada entorno abierto y guarda el resultado.
// Es el único camino que consulta el daemon por espacio: la pantalla lee lo que
// dejó guardado, así que refrescar no cuesta un contenedor.
func (s *PracticeEnvironmentService) SweepDiskUsage(ctx context.Context) {
	measurer, ok := s.docker.(volumeUsageMeasurer)
	if !ok {
		return
	}
	recorder, ok := s.repo.(domain.PracticeEnvironmentDiskUsageRecorder)
	if !ok {
		return
	}

	environments, err := s.repo.GetAllRunningPracticeEnvironments(ctx)
	if err != nil {
		log.Printf("[Disk Sweep] no se pudieron listar los entornos abiertos: %v", err)
		return
	}

	const mb = int64(1024 * 1024)
	for _, environment := range environments {
		tenantID := environment.TenantID
		if tenantID == "" {
			tenantID = domain.DefaultTenantID
		}
		policies := s.effectivePolicies(ctx, tenantID)
		quotaMB := policies.EffectiveDiskQuotaMB()
		if quotaMB <= 0 {
			continue
		}

		usedMB, measureErr := measurer.MeasurePracticeEnvironmentVolumeMB(ctx, environment.VolumeName())
		if measureErr != nil {
			// Un volumen ilegible se reintenta en el próximo barrido; el entorno no
			// se bloquea por una medición que no se pudo hacer.
			log.Printf("[Disk Sweep] entorno %s: no se pudo medir el volumen: %v", environment.ID, measureErr)
			continue
		}

		verdict := domain.EvaluateDiskQuota(usedMB*mb, quotaMB, policies.EffectiveDiskWarningPct(), policies.EffectiveDiskBlockPct())
		state := domain.DiskStateOK
		switch verdict.Status {
		case domain.DiskQuotaWarning:
			state = domain.DiskStateWarning
		case domain.DiskQuotaFull:
			state = domain.DiskStateFull
		}
		if state == environment.DiskState && verdict.UsedMB == environment.DiskUsedMB {
			continue
		}
		if recordErr := recorder.RecordDiskUsage(ctx, environment.ID, verdict.UsedMB, state); recordErr != nil {
			log.Printf("[Disk Sweep] entorno %s: no se pudo guardar la medición: %v", environment.ID, recordErr)
			continue
		}
		if state != domain.DiskStateOK {
			log.Printf("[Disk Sweep] entorno %s en %s: %s", environment.ID, state, verdict.Message)
		}
	}
}

// cachePurger es la capacidad opcional de borrar rutas de caché del volumen.
type cachePurger interface {
	PurgePracticeEnvironmentCaches(ctx context.Context, volumeName string, paths []string) (int64, error)
}

// CachePurgeResult es lo que ve la pantalla después de limpiar.
type CachePurgeResult struct {
	FreedMB int64  `json:"freed_mb"`
	UsedMB  int64  `json:"used_mb"`
	LimitMB int64  `json:"limit_mb"`
	State   string `json:"state"`
}

// PurgeEnvironmentCaches borra las cachés declaradas del entorno y devuelve el
// espacio liberado con la medición nueva. No toca el código del alumno ni el
// volumen congelado de un examen cerrado.
func (s *PracticeEnvironmentService) PurgeEnvironmentCaches(ctx context.Context, environmentID string, paths []string) (*CachePurgeResult, error) {
	environment, err := s.repo.GetByID(ctx, environmentID)
	if err != nil || environment == nil {
		return nil, fmt.Errorf("%w: environment %s", domain.ErrNotFound, environmentID)
	}
	if environment.FrozenAt != nil || environment.Status == domain.PracticeEnvironmentStatusFrozen {
		return nil, s.frozenErrorFor(ctx, environment)
	}
	purger, ok := s.docker.(cachePurger)
	if !ok {
		return nil, ErrPracticeEnvironmentVolumeUnavailable
	}

	freedMB, err := purger.PurgePracticeEnvironmentCaches(ctx, environment.VolumeName(), paths)
	if err != nil {
		return nil, fmt.Errorf("failed to purge environment caches: %w", err)
	}

	environment, err = s.repo.GetByID(ctx, environmentID)
	if err != nil || environment == nil {
		return nil, fmt.Errorf("%w: environment %s", domain.ErrNotFound, environmentID)
	}
	s.annotateDiskLimit(ctx, environment)

	result := &CachePurgeResult{FreedMB: freedMB, LimitMB: environment.DiskLimitMB, State: domain.DiskStateOK}
	if measurer, ok := s.docker.(volumeUsageMeasurer); ok {
		if usedMB, measureErr := measurer.MeasurePracticeEnvironmentVolumeMB(ctx, environment.VolumeName()); measureErr == nil {
			result.UsedMB = usedMB
			if recorder, ok := s.repo.(domain.PracticeEnvironmentDiskUsageRecorder); ok {
				state := domain.DiskStateOK
				if result.LimitMB > 0 && usedMB*100 >= result.LimitMB*100 {
					state = domain.DiskStateFull
				} else if result.LimitMB > 0 && usedMB*100 >= result.LimitMB*80 {
					state = domain.DiskStateWarning
				}
				result.State = state
				_ = recorder.RecordDiskUsage(ctx, environment.ID, usedMB, state)
			}
		}
	}
	return result, nil
}

// DiskUsageWorker corre el barrido cada intervalo sin bloquear el arranque.
type DiskUsageWorker struct {
	sweep    func(ctx context.Context)
	interval time.Duration
	stopChan chan struct{}
}

func NewDiskUsageWorker(sweep func(ctx context.Context), interval time.Duration) *DiskUsageWorker {
	if interval <= 0 {
		interval = DiskUsageSweepInterval
	}
	return &DiskUsageWorker{sweep: sweep, interval: interval, stopChan: make(chan struct{})}
}

func (w *DiskUsageWorker) Start(ctx context.Context) {
	if w.sweep == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.sweep(ctx)
			case <-w.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	log.Printf("[Disk Sweep] medición de disco iniciada (intervalo: %v)", w.interval)
}

func (w *DiskUsageWorker) Stop() {
	close(w.stopChan)
}
