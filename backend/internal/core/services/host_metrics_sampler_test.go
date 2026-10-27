package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// fakeMetricsStore registra lo que se inserta y pide podar, para verificar que
// el sampler persiste una fila real por ciclo y conserva la retencion.
type fakeMetricsStore struct {
	inserted   []domain.HostMetricsSample
	prunedCall []time.Time
}

func (f *fakeMetricsStore) Insert(_ context.Context, sample *domain.HostMetricsSample) error {
	f.inserted = append(f.inserted, *sample)
	return nil
}

func (f *fakeMetricsStore) ListSince(context.Context, string, time.Time, int) ([]domain.HostMetricsSample, error) {
	return nil, nil
}

func (f *fakeMetricsStore) PruneBefore(_ context.Context, olderThan time.Time) (int64, error) {
	f.prunedCall = append(f.prunedCall, olderThan)
	return 0, nil
}

// storeConFalloDeInsercion embebe el fake por puntero (metodos con receptor
// puntero) y sobreescribe Insert para simular un fallo de persistencia.
type storeConFalloDeInsercion struct {
	*fakeMetricsStore
	insertErr error
}

func (s *storeConFalloDeInsercion) Insert(_ context.Context, _ *domain.HostMetricsSample) error {
	return s.insertErr
}

func TestSamplerPersistsSampleOnce(t *testing.T) {
	store := &fakeMetricsStore{}
	sampler := NewHostMetricsSampler(store, &fixedTelemetry{
		snapshot: domain.HostTelemetrySnapshot{
			RAMUsedBytes:  8 * 1024 * 1024 * 1024, // 8 GiB
			RAMAvailableBytes: 6 * 1024 * 1024 * 1024,
			RAMPercent:    42.34,
			CPUPercent:    17.77,
			CPUCores:      8,
		},
	}, func(context.Context) (int, error) { return 3, nil })

	if err := sampler.SampleOnce(context.Background()); err != nil {
		t.Fatalf("SampleOnce failed: %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("expected exactly one inserted sample, got %d", len(store.inserted))
	}
	sample := store.inserted[0]
	if sample.RAMPercent != 42.3 {
		t.Errorf("expected RAMPercent 42.3, got %v", sample.RAMPercent)
	}
	if sample.RAMUsedGB != 8.0 {
		t.Errorf("expected RAMUsedGB 8.0, got %v", sample.RAMUsedGB)
	}
	if sample.CPUPercent != 17.8 {
		t.Errorf("expected CPUPercent 17.8, got %v", sample.CPUPercent)
	}
	if sample.ActiveContainers != 3 {
		t.Errorf("expected ActiveContainers 3, got %d", sample.ActiveContainers)
	}
	if sample.SampledAt.IsZero() {
		t.Error("expected a sampled_at timestamp")
	}

	// La poda se pidio con la ventana de retencion: ahora - 7 dias (mismo minuto).
	if len(store.prunedCall) != 1 {
		t.Fatalf("expected one prune call, got %d", len(store.prunedCall))
	}
	expectedPrune := time.Now().Add(-domain.HostMetricsRetentionWindow)
	delta := store.prunedCall[0].Sub(expectedPrune)
	if delta < 0 {
		delta = -delta
	}
	if delta > time.Minute {
		t.Errorf("prune window off by %v (retention %v)", delta, domain.HostMetricsRetentionWindow)
	}
}

// TestSamplerActiveCounterFailOpen: si el contador de activos falla, la muestra
// se persiste igual con cero activos (la telemetria del host no debe depender
// de un conteo accesorio).
func TestSamplerActiveCounterFailOpen(t *testing.T) {
	store := &fakeMetricsStore{}
	sampler := NewHostMetricsSampler(store, &fixedTelemetry{
		snapshot: domain.HostTelemetrySnapshot{
			RAMUsedBytes:  4 * 1024 * 1024 * 1024,
			RAMTotalBytes: 16 * 1024 * 1024 * 1024,
			RAMPercent:    25,
		},
	}, func(context.Context) (int, error) { return 0, errors.New("db down") })

	if err := sampler.SampleOnce(context.Background()); err != nil {
		t.Fatalf("SampleOnce should tolerate counter failure: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("expected the sample persisted despite counter failure, got %d", len(store.inserted))
	}
	if store.inserted[0].ActiveContainers != 0 {
		t.Errorf("expected 0 active containers on fail-open, got %d", store.inserted[0].ActiveContainers)
	}
}

// TestSamplerTelemetryFailureNoInsert: si no se puede leer la memoria del host
// no hay muestra que escribir: falla hacia arriba para que el panel quede vacio
// (estado "Recopilando telemetria") en vez de inventar filas.
func TestSamplerTelemetryFailureNoInsert(t *testing.T) {
	store := &fakeMetricsStore{}
	sampler := NewHostMetricsSampler(store, &fixedTelemetry{err: errors.New("host down")}, nil)

	if err := sampler.SampleOnce(context.Background()); err == nil {
		t.Fatal("expected SampleOnce to fail when telemetry fails")
	}
	if len(store.inserted) != 0 {
		t.Fatalf("expected no insert on telemetry failure, got %d", len(store.inserted))
	}
}

func TestSamplerInsertFailureSurfaces(t *testing.T) {
	store := &storeConFalloDeInsercion{
		fakeMetricsStore: &fakeMetricsStore{},
		insertErr:        errors.New("insert failed"),
	}
	sampler := NewHostMetricsSampler(store, &fixedTelemetry{
		snapshot: domain.HostTelemetrySnapshot{RAMPercent: 10},
	}, nil)

	if err := sampler.SampleOnce(context.Background()); err == nil {
		t.Fatal("expected the insert error to surface")
	}
}

// fixedTelemetry es un doble de HostTelemetry con snapshot fijo o error.
type fixedTelemetry struct {
	snapshot domain.HostTelemetrySnapshot
	err      error
}

func (f *fixedTelemetry) Snapshot(context.Context) (domain.HostTelemetrySnapshot, error) {
	if f.err != nil {
		return domain.HostTelemetrySnapshot{}, f.err
	}
	return f.snapshot, nil
}