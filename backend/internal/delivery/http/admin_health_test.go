package httpdelivery

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// counterOnlyRepo embebe la interfaz completa para satisfacer el contrato sin
// reimplementar quince metodos que esta prueba no ejercita.
type counterOnlyRepo struct {
	domain.WorkspaceRepository
	counts map[string]int
	err    error
}

func (r *counterOnlyRepo) CountWorkspacesByStatus(context.Context, string) (map[string]int, error) {
	return r.counts, r.err
}

type plainRepo struct {
	domain.WorkspaceRepository
}

func TestWorkspaceStatusCountsUsesAggregate(t *testing.T) {
	handler := &AdminHandler{workspaceRepo: &counterOnlyRepo{
		counts: map[string]int{"running": 2, "hibernated": 5, "ran": 0},
	}}

	counts := handler.workspaceStatusCounts(context.Background(), "tenant-1")
	if counts["hibernated"] != 5 {
		t.Fatalf("expected 5 hibernated workspaces, got %d", counts["hibernated"])
	}
	if counts["running"] != 2 {
		t.Fatalf("expected 2 running workspaces, got %d", counts["running"])
	}
}

func TestWorkspaceStatusCountsDegradesWhenAggregateFails(t *testing.T) {
	handler := &AdminHandler{workspaceRepo: &counterOnlyRepo{err: errors.New("db down")}}

	if counts := handler.workspaceStatusCounts(context.Background(), "tenant-1"); counts != nil {
		t.Fatalf("expected a nil breakdown on aggregate failure, got %+v", counts)
	}
}

func TestWorkspaceStatusCountsDegradesWhenAggregateMissing(t *testing.T) {
	handler := &AdminHandler{workspaceRepo: &plainRepo{}}

	if counts := handler.workspaceStatusCounts(context.Background(), "tenant-1"); counts != nil {
		t.Fatalf("expected a nil breakdown when the repository lacks the aggregate, got %+v", counts)
	}
}

// ---------------------------------------------------------------------------
// Historial persistido del host (S4): ventana configurable y lectura desde DB
// ---------------------------------------------------------------------------

func TestParseHistoryWindowDefaultsTo60m(t *testing.T) {
	window, limit := parseHistoryWindow("")
	if window != 60*time.Minute {
		t.Fatalf("expected default window 60m, got %v", window)
	}
	if limit != 61 {
		t.Fatalf("expected default limit 61, got %d", limit)
	}

	window, limit = parseHistoryWindow("banana")
	if window != 60*time.Minute {
		t.Fatalf("invalid window must fall back to 60m, got %v", window)
	}
	if limit != 61 {
		t.Fatalf("invalid window must use default limit, got %d", limit)
	}
}

func TestParseHistoryWindow24h(t *testing.T) {
	window, limit := parseHistoryWindow("24h")
	if window != 24*time.Hour {
		t.Fatalf("expected 24h window, got %v", window)
	}
	if limit != 1441 {
		t.Fatalf("expected limit 1441 for 24h, got %d", limit)
	}
}

func TestParseHistoryWindowCapsAtRetention(t *testing.T) {
	window, _ := parseHistoryWindow("30d")
	if window != domain.HostMetricsHistoryWindowMax {
		t.Fatalf("expected window capped at 7 days, got %v", window)
	}

	window, _ = parseHistoryWindow("-1h")
	if window != 60*time.Minute {
		t.Fatalf("negative window must fall back to 60m, got %v", window)
	}
}

// fakeSampleStore devuelve muestras fijas y registra la ventana pedida.
type fakeSampleStore struct {
	samples []domain.HostMetricsSample
	since   time.Time
	tags    []string
}

func (f *fakeSampleStore) Insert(context.Context, *domain.HostMetricsSample) error { return nil }

func (f *fakeSampleStore) ListSince(_ context.Context, tenantID string, since time.Time, _ int) ([]domain.HostMetricsSample, error) {
	f.since = since
	f.tags = append(f.tags, tenantID)
	return f.samples, nil
}

func (f *fakeSampleStore) PruneBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func TestGetHistorySnapshotsReadsFromStore(t *testing.T) {
	now := time.Now()
	store := &fakeSampleStore{samples: []domain.HostMetricsSample{
		{SampledAt: now.Add(-2 * time.Minute), RAMPercent: 12.3, RAMUsedGB: 4.5, CPUPercent: 8.9, ActiveContainers: 2},
		{SampledAt: now.Add(-1 * time.Minute), RAMPercent: 13.4, RAMUsedGB: 4.6, CPUPercent: 9.0, ActiveContainers: 2},
	}}
	handler := &AdminHandler{metricsSamples: store}

	snaps := handler.getHistorySnapshots(context.Background(), "24h", "tenant-42")
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}
	// El timestamp viaja como RFC3339 para que el grafico calcule deltas reales.
	if _, err := time.Parse(time.RFC3339, snaps[0].Timestamp); err != nil {
		t.Fatalf("timestamp must be RFC3339, got %q (%v)", snaps[0].Timestamp, err)
	}
	if snaps[0].RAMPercent != 12.3 || snaps[1].RAMPercent != 13.4 {
		t.Fatalf("unexpected snapshot mapping: %+v", snaps)
	}
	if store.tags[len(store.tags)-1] != "tenant-42" {
		t.Fatalf("expected tenant filter propagated, got %v", store.tags)
	}
	// La ventana 24h se traduce en un since de hace ~24h.
	if d := time.Since(store.since); d < 23*time.Hour || d > 25*time.Hour {
		t.Fatalf("expected since near -24h, got %v ago", d)
	}
}

func TestGetHistorySnapshotsEmptyWithoutStore(t *testing.T) {
	handler := &AdminHandler{}

	snaps := handler.getHistorySnapshots(context.Background(), "", "tenant-42")
	if snaps == nil || len(snaps) != 0 {
		t.Fatalf("expected an empty history list when the store is not injected, got %v", snaps)
	}
}
