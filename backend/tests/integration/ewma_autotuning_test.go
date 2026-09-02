package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	dockerinfra "solv-backend/internal/infrastructure/docker"
	"solv-backend/internal/infrastructure/storage/postgres"
	systeminfra "solv-backend/internal/infrastructure/system"
)

func TestSlice6ConcurrentEWMAAndAutoLearning(t *testing.T) {
	dsn := "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	dbInstance, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	if err := dbInstance.RunInitialMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	repo := postgres.NewPostgresLabTemplateRepository(dbInstance.GetDB())
	profilerService := services.NewEWMAProfilerService(repo)

	ctx := context.Background()
	baseImage := "python:3.12-slim"
	setupScript := fmt.Sprintf("pip install numpy # test-%d", time.Now().UnixNano())
	sigHash := profilerService.CalculateSignatureHash(baseImage, setupScript)

	// 1. Inicializar perfil
	initialProfile, err := profilerService.RecordSessionPeakAndRecalculate(ctx, baseImage, setupScript, 200.0)
	if err != nil {
		t.Fatalf("Failed to initialize EWMA profile: %v", err)
	}

	if initialProfile.EWMAState.CurrentEWMAMB != 200.0 {
		t.Errorf("Expected initial EWMA 200.0, got %f", initialProfile.EWMAState.CurrentEWMAMB)
	}

	// 2. Prueba de Concurrencia Simulando 10 alumnos finalizando laboratorios simultáneamente
	var wg sync.WaitGroup
	concurrentUpdates := 10
	samplePeakRAM := 500.0

	for i := 0; i < concurrentUpdates; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, updateErr := profilerService.RecordSessionPeakAndRecalculate(ctx, baseImage, setupScript, samplePeakRAM)
			if updateErr != nil {
				t.Errorf("Concurrent EWMA update failed: %v", updateErr)
			}
		}()
	}

	wg.Wait()

	// 3. Obtener resultado final guardado en PostgreSQL
	finalTemplate, err := repo.GetBySignatureHash(ctx, sigHash)
	if err != nil || finalTemplate == nil {
		t.Fatalf("Failed to fetch final lab template profile: %v", err)
	}

	t.Logf("Slice 6 EWMA Test PASSED! Hash: %s... | Final EWMA: %.2f MB | Max Quota (EWMA * 1.25): %d MB | Total Samples: %d",
		sigHash[:8], finalTemplate.ResourceProfile.EWMAState.CurrentEWMAMB, finalTemplate.ResourceProfile.MaxQuotaMB, finalTemplate.ResourceProfile.EWMAState.SampleCount)

	if finalTemplate.ResourceProfile.EWMAState.SampleCount != concurrentUpdates+1 {
		t.Errorf("Expected sample count %d, got %d", concurrentUpdates+1, finalTemplate.ResourceProfile.EWMAState.SampleCount)
	}

	// Hard Quota debe ser EWMA * 1.25
	if finalTemplate.ResourceProfile.MaxQuotaMB <= 256 {
		t.Errorf("Expected recalculated Hard Quota > 256 MB, got %d MB", finalTemplate.ResourceProfile.MaxQuotaMB)
	}
}

func TestSlice6NetworkICCDisabledAndZombieCollector(t *testing.T) {
	dsn := "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	dbInstance, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	dockerClient, err := dockerinfra.NewClient()
	if err != nil {
		t.Skipf("Skipping integration test: Docker daemon not available: %v", err)
	}

	ctx := context.Background()

	// 1. Verificar creación de red con enable_icc=false
	netName := "solv-traefik-net-test"
	if err := dockerClient.EnsureICCDisabledNetworkExists(ctx, netName); err != nil {
		t.Fatalf("Failed to ensure network with enable_icc=false: %v", err)
	}

	// 2. Probar reconciliador de contenedores huérfanos (Zombie Collector)
	workspaceRepo := postgres.NewPostgresWorkspaceRepository(dbInstance.GetDB())
	zombieWorker := services.NewZombieCollectorWorker(workspaceRepo, dockerClient, 1*time.Second)

	// Crear contenedor huérfano simulado
	orphanConfig := domain.WorkspaceContainerConfig{
		Image:         "gitpod/openvscode-server:latest",
		ContainerName: "solv-workspace-orphan-test-12345",
		VolumeName:    "solv_test_vol_orphan",
		MemoryLimitMB: 256,
		NetworkName:   netName,
		Labels: map[string]string{
			"solv.managed": "true",
		},
	}

	_ = dockerClient.EnsureVolumeExists(ctx, orphanConfig.VolumeName)
	orphanID, err := dockerClient.StartWorkspaceContainer(ctx, orphanConfig)
	if err != nil {
		t.Fatalf("Failed to create dummy orphan container: %v", err)
	}

	defer func() {
		_ = dockerClient.StopAndRemoveContainer(ctx, orphanID)
	}()

	// Ejecutar reconciliación de contenedores huérfanos
	zombieWorker.ReconcileOrphanContainers(ctx)

	// Verificar si el contenedor huérfano fue eliminado de Docker
	managedContainers, err := dockerClient.ListAllManagedContainers(ctx)
	if err != nil {
		t.Fatalf("Failed to list managed containers: %v", err)
	}

	for _, cid := range managedContainers {
		if cid == orphanID {
			t.Errorf("Orphan container %s was not reclaimed by Zombie Collector!", orphanID[:12])
		}
	}

	t.Logf("Slice 6 Zombie Collector PASSED! Reclaimed orphan container count: %d", zombieWorker.GetReclaimedCount())
}

func TestSlice6PrometheusMetricsEndpoint(t *testing.T) {
	dsn := "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	dbInstance, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	workspaceRepo := postgres.NewPostgresWorkspaceRepository(dbInstance.GetDB())
	hostMonitor := systeminfra.NewGopsutilHostMonitor(15.0)
	zombieWorker := services.NewZombieCollectorWorker(workspaceRepo, nil, 10*time.Second)

	metricsHandler := httpdelivery.NewMetricsHandler(workspaceRepo, hostMonitor, zombieWorker)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	metricsHandler.HandleMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK from /metrics, got %d", rr.Code)
	}

	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("Failed to read metrics response body: %v", err)
	}

	bodyStr := string(body)
	t.Logf("/metrics Response Body:\n%s", bodyStr)

	requiredMetrics := []string{
		"solv_active_workspaces_total",
		"solv_host_available_memory_bytes",
		"solv_host_oom_guard_bytes",
		"solv_orphan_containers_reclaimed_total",
	}

	for _, metric := range requiredMetrics {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("Metrics endpoint output missing expected metric: %s", metric)
		}
	}
}
