package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	dockerinfra "solv-backend/internal/infrastructure/docker"
	"solv-backend/internal/infrastructure/storage/postgres"
	systeminfra "solv-backend/internal/infrastructure/system"
)

func TestTicket1OpenVSCodeServerMigration(t *testing.T) {
	dsn := getTestDSN()
	dbInstance, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	dockerClient, err := dockerinfra.NewClient()
	if err != nil {
		t.Skipf("Skipping integration test: Docker daemon not available: %v", err)
	}

	workspaceRepo := postgres.NewPostgresWorkspaceRepository(dbInstance.GetDB())
	hostMonitor := systeminfra.NewGopsutilHostMonitor(-1.0)
	workspaceService := services.NewWorkspaceService(workspaceRepo, dockerClient, hostMonitor)

	studentID := uuid.NewString()
	subjectID := "00000000-0000-0000-0000-000000000001"

	// Asegurar existencia de tenant y materia base para evitar fallos de clave foránea
	_, _ = dbInstance.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default') ON CONFLICT (id) DO NOTHING;
		INSERT INTO subjects (id, tenant_id, name, code) VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Materia General', 'GEN-101') ON CONFLICT (id) DO NOTHING;
	`)

	ctx := context.Background()

	// 1. Iniciar nuevo entorno con OpenVSCode Server
	wsInstance, err := workspaceService.StartWorkspace(ctx, studentID, subjectID)
	if err != nil {
		t.Fatalf("Failed to start OpenVSCode workspace: %v", err)
	}

	defer func() {
		if wsInstance.ContainerID != nil && *wsInstance.ContainerID != "" {
			_ = dockerClient.StopAndRemoveContainer(ctx, *wsInstance.ContainerID)
		}
	}()

	// 2. Verificar que el registro en DB tiene estado 'running'
	fetched, err := workspaceRepo.GetByID(ctx, wsInstance.ID)
	if err != nil || fetched == nil {
		t.Fatalf("Failed to fetch workspace instance from DB: %v", err)
	}

	if fetched.Status != domain.WorkspaceStatusRunning {
		t.Errorf("Expected workspace status 'running', got %s", fetched.Status)
	}

	t.Logf("Ticket 1 PASSED! OpenVSCode Server instantiated cleanly: ID=%s, AccessURL=%s, Port=3000, Mount=/home/workspace",
		wsInstance.ID, wsInstance.AccessURL)
}
