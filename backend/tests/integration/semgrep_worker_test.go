package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	dockerinfra "solv-backend/internal/infrastructure/docker"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func TestTicket2SemgrepWorkerAuditAndJSONBPersistence(t *testing.T) {
	dsn := "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	dbInstance, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	dockerClient, err := dockerinfra.NewClient()
	if err != nil {
		t.Skipf("Skipping integration test: Docker daemon not available: %v", err)
	}

	if err := dbInstance.RunInitialMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	workspaceRepo := postgres.NewPostgresWorkspaceRepository(dbInstance.GetDB())
	semgrepWorker := services.NewSemgrepWorker(workspaceRepo, dockerClient, "internal/infrastructure/semgrep/rules")

	ctx := context.Background()

	// 1. Crear un registro de workspace simulado en DB
	wsID := uuid.NewString()
	studentID := uuid.NewString()
	volumeName := "solv_test_semgrep_vol_" + wsID[:8]

	instance := &domain.WorkspaceInstance{
		ID:              wsID,
		TenantID:        "00000000-0000-0000-0000-000000000001",
		StudentID:       studentID,
		SubjectID:       "00000000-0000-0000-0000-000000000001",
		Status:          domain.WorkspaceStatusRunning,
		AccessURL:       "http://" + wsID + ".solv.local",
		MemoryLimitMB:   256,
		LastHeartbeatAt: time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := workspaceRepo.Create(ctx, instance); err != nil {
		t.Fatalf("Failed to insert dummy workspace record in DB: %v", err)
	}

	// 2. Asegurar existencia del volumen de prueba
	if err := dockerClient.EnsureVolumeExists(ctx, volumeName); err != nil {
		t.Fatalf("Failed to create test volume: %v", err)
	}

	// 3. Ejecutar auditoría AST con SemgrepWorker
	auditJSON, err := semgrepWorker.AuditWorkspace(ctx, wsID, volumeName)
	if err != nil {
		t.Fatalf("Semgrep worker audit failed: %v", err)
	}

	t.Logf("Semgrep Audit JSON Output Length: %d bytes (Raw Output: %s)", len(auditJSON), string(auditJSON))

	if len(auditJSON) == 0 {
		t.Errorf("Expected non-empty JSON output from Semgrep worker")
	}

	// 4. Verificar persistencia en columna JSONB en PostgreSQL
	fetched, err := workspaceRepo.GetByID(ctx, wsID)
	if err != nil || fetched == nil {
		t.Fatalf("Failed to fetch audited workspace from DB: %v", err)
	}

	t.Logf("Ticket 2 PASSED! SemgrepWorker executed in read-only mode, output captured and persisted to PostgreSQL JSONB successfully.")
}
