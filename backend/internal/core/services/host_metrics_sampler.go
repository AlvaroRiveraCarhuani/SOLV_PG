package services

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

// HostMetricsSampler es el worker que materializa la D1: el historial del panel
// de salud deja de ser estimacion en memoria del handler y pasa a ser filas
// reales en host_metrics_samples. Cada ciclo toma una foto del host (gopsutil +
// daemon de Docker), escribe 1 fila y poda lo anterior a la retencion.
type HostMetricsSampler struct {
	store     domain.HostMetricsSampleStore
	telemetry domain.HostTelemetry
	// activeCounter mide cuantos contenedores de laboratorio estan activos en el
	// nodo. Se inyecta como funcion para no acoplar al repositorio concreto y
	// para poder doblarla en los tests con un valor fijo.
	activeCounter func(ctx context.Context) (int, error)

	mu       sync.Mutex
	tenantID string
	interval time.Duration
	retention time.Duration
}

// NewHostMetricsSampler crea el worker con la frecuencia y retencion por
// defecto (1 minuto / 7 dias). SetTenantID y WithTiming permiten ajustarlos.
func NewHostMetricsSampler(
	store domain.HostMetricsSampleStore,
	telemetry domain.HostTelemetry,
	activeCounter func(ctx context.Context) (int, error),
) *HostMetricsSampler {
	return &HostMetricsSampler{
		store:         store,
		telemetry:     telemetry,
		activeCounter: activeCounter,
		tenantID:      domain.DefaultTenantID,
		interval:      domain.HostMetricsSampleRate,
		retention:     domain.HostMetricsRetentionWindow,
	}
}

// SetTenantID fija bajo que tenant se persisten las muestras del nodo. En
// despliegues single-tenant (el actual) es el tenant por defecto.
func (s *HostMetricsSampler) SetTenantID(tenantID string) {
	if tenantID != "" {
		s.tenantID = tenantID
	}
}

// WithTiming ajusta intervalo y retencion del worker (solo tests o despliegues
// especificos; en produccion se usan los valores del dominio).
func (s *HostMetricsSampler) WithTiming(interval, retention time.Duration) *HostMetricsSampler {
	if interval > 0 {
		s.interval = interval
	}
	if retention > 0 {
		s.retention = retention
	}
	return s
}

// Start corre el ciclo de muestreo hasta que el contexto se cancele. Toma una
// muestra inmediatamente al arrancar (50 ms de espera, no un minuto) para que
// el panel tenga datos reales desde el primer render posterior a un reinicio.
func (s *HostMetricsSampler) Start(ctx context.Context) {
	log.Printf("[Host Metrics Sampler] Worker started (interval=%v, retention=%v)", s.interval, s.retention)
	s.SampleOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Host Metrics Sampler] Worker stopped.")
			return
		case <-ticker.C:
			s.SampleOnce(ctx)
		}
	}
}

// SampleOnce toma, persiste y poda una muestra. Es publico y testeable: el
// worker de Start solo es el loop que lo invoca con el intervalo.
func (s *HostMetricsSampler) SampleOnce(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	snapshot, err := s.telemetry.Snapshot(ctx)
	if err != nil {
		log.Printf("[Host Metrics Sampler] Error sampling host telemetry: %v", err)
		return err
	}

	active := 0
	if s.activeCounter != nil {
		if count, countErr := s.activeCounter(ctx); countErr == nil {
			active = count
		}
	}

	sample := &domain.HostMetricsSample{
		SampledAt:        now,
		RAMPercent:       roundMetric1(snapshot.RAMPercent),
		RAMUsedGB:        roundMetric2(float64(snapshot.RAMUsedBytes) / (1024 * 1024 * 1024)),
		CPUPercent:       roundMetric1(snapshot.CPUPercent),
		ActiveContainers: active,
		TenantID:         s.tenantID,
	}

	if err := s.store.Insert(ctx, sample); err != nil {
		log.Printf("[Host Metrics Sampler] Error inserting sample: %v", err)
		return err
	}

	pruned, err := s.store.PruneBefore(ctx, now.Add(-s.retention))
	if err != nil {
		// La poda fallida no invalida la muestra recien insertada: el siguiente
		// ciclo vuelve a intentarla. Se registra y se sigue.
		log.Printf("[Host Metrics Sampler] Error pruning old samples: %v", err)
		return nil
	}
	if pruned > 0 {
		log.Printf("[Host Metrics Sampler] Pruned %d samples older than %v", pruned, s.retention)
	}
	return nil
}

// roundMetric1 redondea a un decimal; roundMetric2 a dos. Se nombran explícito
// para no colisionar con el redondeo por truncamiento de image_verification
// (round2) y porque ambas redondean de verdad en vez de recortar.
func roundMetric1(v float64) float64 {
	return math.Round(v*10) / 10
}

func roundMetric2(v float64) float64 {
	return math.Round(v*100) / 100
}