package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestEnsureScannerPrimesDBAndBacksOff(t *testing.T) {
	worker := NewTemplateAuditWorker(nil, &recordingAuditRepo{})
	worker.cacheDir = t.TempDir()

	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }

	calls := 0
	worker.dbSyncer = func(context.Context) error {
		calls++
		return errors.New("sin red")
	}

	worker.ensureScanner(context.Background())
	if calls != 1 {
		t.Fatalf("esperaba un intento de descarga, obtuve %d", calls)
	}

	// Dentro del backoff no se vuelve a intentar ni se toca la cola.
	worker.ensureScanner(context.Background())
	if calls != 1 {
		t.Fatalf("no debía reintentar dentro del backoff, obtuve %d intentos", calls)
	}

	now = now.Add(trivyDBRetryBase)
	worker.ensureScanner(context.Background())
	if calls != 2 {
		t.Fatalf("esperaba un reintento tras el backoff, obtuve %d", calls)
	}
}

func TestEnsureScannerSkipsDownloadWhenDBIsReady(t *testing.T) {
	worker := NewTemplateAuditWorker(nil, &recordingAuditRepo{})
	worker.cacheDir = t.TempDir()
	writeTrivyDB(t, worker.cacheDir)

	calls := 0
	worker.dbSyncer = func(context.Context) error {
		calls++
		return nil
	}

	worker.ensureScanner(context.Background())
	if calls != 0 {
		t.Fatalf("la base ya existía; no debía descargar, obtuve %d intentos", calls)
	}
}

func TestReconcileSkipsQueueWithoutTrivyDB(t *testing.T) {
	repo := &pendingAuditRepo{
		recordingAuditRepo: &recordingAuditRepo{},
		pending: []*domain.AdminTemplateReviewItem{
			{ID: "tpl-sin-base", Name: "Sin base CVE", DockerImage: "local/audit-test", Status: "PENDIENTE_AUDITORIA"},
		},
	}
	worker := NewTemplateAuditWorker(nil, repo)
	worker.cacheDir = t.TempDir()
	worker.smokeRunner = func(context.Context, *domain.AdminTemplateReviewItem) (string, string) {
		return "passed", ""
	}
	worker.trivyRunner = func(context.Context, string) (string, int, int, []byte) {
		return "passed", 0, 0, []byte(`{"ok":true}`)
	}

	worker.reconcilePending(context.Background())

	if len(repo.calls) != 0 {
		t.Fatalf("sin base CVE no se debe auditar; hubo %d auditorías", len(repo.calls))
	}
}

func TestReconcileBacksOffInfrastructureFailurePerTemplate(t *testing.T) {
	repo := &pendingAuditRepo{
		recordingAuditRepo: &recordingAuditRepo{},
		pending: []*domain.AdminTemplateReviewItem{
			{ID: "tpl-infra", Name: "Falla de infraestructura", DockerImage: "local/audit-test", Status: "PENDIENTE_AUDITORIA"},
		},
	}
	worker := NewTemplateAuditWorker(nil, repo)
	worker.cacheDir = t.TempDir()
	writeTrivyDB(t, worker.cacheDir)

	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }
	worker.smokeRunner = func(context.Context, *domain.AdminTemplateReviewItem) (string, string) {
		return auditStatusError, "el daemon no responde"
	}
	worker.trivyRunner = func(context.Context, string) (string, int, int, []byte) {
		return "passed", 0, 0, []byte(`{"ok":true}`)
	}

	worker.reconcilePending(context.Background())
	if len(repo.calls) != 1 {
		t.Fatalf("esperaba una auditoría, obtuve %d", len(repo.calls))
	}

	// La misma plantilla no vuelve a intentarse dentro del backoff.
	worker.reconcilePending(context.Background())
	if len(repo.calls) != 1 {
		t.Fatalf("no debía reintentar dentro del backoff, obtuve %d auditorías", len(repo.calls))
	}

	now = now.Add(templateRetryBase)
	worker.reconcilePending(context.Background())
	if len(repo.calls) != 2 {
		t.Fatalf("esperaba un reintento tras el backoff, obtuve %d auditorías", len(repo.calls))
	}
}

func TestHasTrivyDBRequiresANonEmptyFile(t *testing.T) {
	worker := NewTemplateAuditWorker(nil, &recordingAuditRepo{})
	worker.cacheDir = t.TempDir()
	if worker.hasTrivyDB() {
		t.Fatal("una caché vacía no debe considerarse lista")
	}

	dbDir := filepath.Join(worker.cacheDir, "db")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dbDir, "trivy.db"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if worker.hasTrivyDB() {
		t.Fatal("un archivo vacío no debe considerarse una base válida")
	}

	if err := os.WriteFile(filepath.Join(dbDir, "trivy.db"), []byte("datos"), 0o600); err != nil {
		t.Fatal(err)
	}
	if worker.hasTrivyDB() {
		t.Fatal("una descarga a medias (sin metadata.json) no debe considerarse lista")
	}

	writeTrivyMetadata(t, dbDir)
	if !worker.hasTrivyDB() {
		t.Fatal("la base completa con metadata debe considerarse lista")
	}
}

func writeTrivyMetadata(t *testing.T, dbDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dbDir, "metadata.json"), []byte(`{"Version":2,"DownloadedAt":"2026-11-17T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeTrivyDB(t *testing.T, cacheDir string) {
	t.Helper()
	dbDir := filepath.Join(cacheDir, "db")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dbDir, "trivy.db"), []byte("datos"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTrivyMetadata(t, dbDir)
}
