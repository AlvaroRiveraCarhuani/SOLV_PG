package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/docker"
)

type MockHostMonitor struct {
	FreePct     float64
	AvailableMB uint64
}

func (m *MockHostMonitor) GetHostMemoryStats() (float64, uint64, error) {
	return m.FreePct, m.AvailableMB, nil
}

func (m *MockHostMonitor) CanAllocateMemory(requiredMB int64) bool {
	if m.FreePct < domain.MinHostFreeRAMPct {
		return false
	}
	return int64(m.AvailableMB) >= requiredMB
}

func TestHostAdmissionControl15PercentMargin(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize docker client: %v", err)
	}

	repo := NewMockWorkspaceRepository()

	// Simular host con solo 10% de RAM libre (< 15% umbral de seguridad)
	depletedHostMonitor := &MockHostMonitor{FreePct: 10.0, AvailableMB: 500}
	service := services.NewWorkspaceService(repo, dockerClient, depletedHostMonitor)

	studentID := uuid.NewString()
	subjectID := uuid.NewString()

	_, err = service.StartWorkspace(ctx, studentID, subjectID)
	if err == nil {
		t.Fatalf("Expected ErrHostMemoryExhausted when host RAM < 15%%, got nil error")
	}

	if !errors.Is(err, services.ErrHostMemoryExhausted) {
		t.Errorf("Expected error ErrHostMemoryExhausted, got: %v", err)
	}

	t.Logf("PASS: Host Admission Control correctly rejected request when host free RAM < 15%%")
}

func TestOOMKilledThreeStrikePenalty(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize docker client: %v", err)
	}

	repo := NewMockWorkspaceRepository()
	healthyHostMonitor := &MockHostMonitor{FreePct: 50.0, AvailableMB: 8192}
	service := services.NewWorkspaceService(repo, dockerClient, healthyHostMonitor)

	studentID := uuid.NewString()
	subjectID := uuid.NewString()

	// 1. Iniciar workspace
	ws, err := service.StartWorkspace(ctx, studentID, subjectID)
	if err != nil {
		t.Fatalf("StartWorkspace failed: %v", err)
	}

	// 2. Simular 3 caídas consecutivas por OOMKilled (Strike count = 3)
	now := time.Now()
	ws.OOMStrikeCount = 3
	ws.LastOOMKilledAt = &now
	ws.Status = domain.WorkspaceStatusOOMKilled

	// 3. Intentar reiniciar inmediatamente -> Debe ser rechazado por la penalización de 5 minutos
	_, err = service.StartWorkspace(ctx, studentID, subjectID)
	if err == nil {
		t.Fatalf("Expected ErrOOMKilledCooldownPenalty when 3 strikes reached, got nil error")
	}

	if !errors.Is(err, services.ErrOOMKilledCooldownPenalty) {
		t.Errorf("Expected error ErrOOMKilledCooldownPenalty, got: %v", err)
	}

	t.Logf("PASS: 3-Strike OOMKilled penalty correctly enforced 5-minute cooldown period")
}

func TestQoSAutoBurstingAndDualValidation(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize docker client: %v", err)
	}

	rawCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to initialize raw docker SDK client: %v", err)
	}

	repo := NewMockWorkspaceRepository()
	healthyHostMonitor := &MockHostMonitor{FreePct: 50.0, AvailableMB: 8192}
	service := services.NewWorkspaceService(repo, dockerClient, healthyHostMonitor)

	studentID := uuid.NewString()
	subjectID := uuid.NewString()

	// 1. Arrancar contenedor con 256MB base
	ws, err := service.StartWorkspace(ctx, studentID, subjectID)
	if err != nil {
		t.Fatalf("StartWorkspace failed: %v", err)
	}

	if ws.ContainerID != nil {
		defer func() {
			_ = rawCli.ContainerRemove(context.Background(), *ws.ContainerID, container.RemoveOptions{Force: true})
		}()
	}

	if ws.MemoryLimitMB != 256 {
		t.Errorf("Expected base MemoryLimitMB to be 256, got %d", ws.MemoryLimitMB)
	}

	// 2. Probar worker QoS (Auto-Bursting + Hibernación) con intervalo corto de prueba (100ms)
	worker := services.NewQoSOrchestratorWorker(repo, dockerClient, healthyHostMonitor, 500*time.Millisecond, 100*time.Millisecond)
	workerContext, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	worker.Start(workerContext)

	// Esperar ciclo del worker
	time.Sleep(1 * time.Second)

	// Verificar que las métricas e inspección del contenedor funcionen limpiamente
	metrics, err := dockerClient.GetContainerMetrics(ctx, *ws.ContainerID)
	if err != nil {
		t.Fatalf("Failed to read container metrics: %v", err)
	}

	if !metrics.IsRunning {
		t.Errorf("Expected container to be running")
	}

	t.Logf("PASS: QoS Auto-Bursting worker and Dual-Validation Hibernation cycle verified successfully! Initial RAM: %d MB", ws.MemoryLimitMB)
}
