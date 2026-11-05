package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/docker"
)

type MockPracticeEnvironmentRepository struct {
	environments map[string]*domain.PracticeEnvironment
}

func NewMockPracticeEnvironmentRepository() *MockPracticeEnvironmentRepository {
	return &MockPracticeEnvironmentRepository{
		environments: make(map[string]*domain.PracticeEnvironment),
	}
}

func (m *MockPracticeEnvironmentRepository) GetByID(ctx context.Context, id string) (*domain.PracticeEnvironment, error) {
	if ws, exists := m.environments[id]; exists {
		return ws, nil
	}
	return nil, nil
}

// GetByStudentSubjectExercise satisface la capacidad opcional que el servicio
// usa para reutilizar el entorno de un ejercicio concreto. El doble no necesita
// base de datos: busca sobre lo que ya registró Create.
func (m *MockPracticeEnvironmentRepository) GetByStudentSubjectExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.PracticeEnvironment, error) {
	for _, ws := range m.environments {
		if ws.StudentID != studentID || ws.SubjectID != subjectID {
			continue
		}
		if ws.ExerciseID != nil && *ws.ExerciseID == exerciseID {
			return ws, nil
		}
	}
	return nil, nil
}

func (m *MockPracticeEnvironmentRepository) Create(ctx context.Context, environment *domain.PracticeEnvironment) error {
	key := environment.StudentID + ":" + environment.SubjectID
	if environment.ExerciseID != nil {
		key += ":" + *environment.ExerciseID
	}
	m.environments[key] = environment
	m.environments[environment.ID] = environment
	return nil
}

func (m *MockPracticeEnvironmentRepository) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	if ws, exists := m.environments[id]; exists {
		ws.ContainerID = &containerID
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockPracticeEnvironmentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	if ws, exists := m.environments[id]; exists {
		ws.Status = status
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockPracticeEnvironmentRepository) UpdateMemoryLimit(ctx context.Context, id string, memoryMB int64) error {
	if ws, exists := m.environments[id]; exists {
		ws.MemoryLimitMB = memoryMB
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockPracticeEnvironmentRepository) RecordHeartbeat(ctx context.Context, id string) error {
	if ws, exists := m.environments[id]; exists {
		ws.LastHeartbeatAt = time.Now()
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockPracticeEnvironmentRepository) IncrementOOMStrike(ctx context.Context, id string) error {
	if ws, exists := m.environments[id]; exists {
		ws.OOMStrikeCount++
		now := time.Now()
		ws.LastOOMKilledAt = &now
		ws.Status = domain.PracticeEnvironmentStatusOOMKilled
		ws.UpdatedAt = now
	}
	return nil
}

func (m *MockPracticeEnvironmentRepository) ResetOOMStrikes(ctx context.Context, id string) error {
	if ws, exists := m.environments[id]; exists {
		ws.OOMStrikeCount = 0
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockPracticeEnvironmentRepository) GetActivePracticeEnvironments(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
	var active []*domain.PracticeEnvironment
	for _, ws := range m.environments {
		if ws.Status == domain.PracticeEnvironmentStatusRunning || ws.Status == domain.PracticeEnvironmentStatusPending {
			active = append(active, ws)
		}
	}
	return active, nil
}

func (m *MockPracticeEnvironmentRepository) GetAllRunningPracticeEnvironments(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
	var running []*domain.PracticeEnvironment
	for _, ws := range m.environments {
		if ws.Status == domain.PracticeEnvironmentStatusRunning {
			running = append(running, ws)
		}
	}
	return running, nil
}

func (m *MockPracticeEnvironmentRepository) GetByType(ctx context.Context, environmentType string) ([]*domain.PracticeEnvironment, error) {
	seen := make(map[string]bool)
	var result []*domain.PracticeEnvironment
	for _, ws := range m.environments {
		if ws.Type == environmentType && !seen[ws.ID] {
			seen[ws.ID] = true
			result = append(result, ws)
		}
	}
	return result, nil
}

func (m *MockPracticeEnvironmentRepository) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	if ws, exists := m.environments[id]; exists {
		ws.SemgrepAudit = auditJSON
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func TestSlice04PracticeEnvironmentLifecycle(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize docker client: %v", err)
	}

	rawCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to initialize raw docker SDK client: %v", err)
	}

	repo := NewMockPracticeEnvironmentRepository()
	healthyHostMonitor := &MockHostMonitor{FreePct: 50.0, AvailableMB: 8192}
	service := services.NewPracticeEnvironmentService(repo, dockerClient, healthyHostMonitor)

	studentID := uuid.NewString()
	subjectID := uuid.NewString()

	// 1. Iniciar Entorno de Desarrollo Interactivo (Code-Server)
	ws1, err := service.StartPracticeEnvironmentForExercise(ctx, studentID, subjectID, testExerciseID)
	if err != nil {
		t.Fatalf("StartPracticeEnvironment failed: %v", err)
	}

	if ws1.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", ws1.Status)
	}

	if ws1.ID == "" {
		t.Errorf("Expected non-empty environment_id UUID")
	}

	// Sin BASE_DOMAIN el acceso se publica en 127.0.0.1 con puerto dinámico.
	if !strings.HasPrefix(ws1.AccessURL, "http://localhost:") {
		t.Errorf("Expected AccessURL 'http://localhost:<port>', got '%s'", ws1.AccessURL)
	}

	// Clean up container and volume
	if ws1.ContainerID != nil {
		defer func() {
			_ = rawCli.ContainerRemove(context.Background(), *ws1.ContainerID, container.RemoveOptions{Force: true})
		}()
	}

	// 2. Verificar Idempotencia (segunda solicitud devuelve la misma instancia)
	ws2, err := service.StartPracticeEnvironmentForExercise(ctx, studentID, subjectID, testExerciseID)
	if err != nil {
		t.Fatalf("Second StartPracticeEnvironment call failed: %v", err)
	}

	if ws2.ID != ws1.ID {
		t.Errorf("Idempotency violation: expected same environment ID '%s', got '%s'", ws1.ID, ws2.ID)
	}

	// 3. Inspección del Contenedor de Docker: puerto 3000 publicado en 127.0.0.1
	containerName := "solv-environment-" + ws1.ID
	inspect, err := rawCli.ContainerInspect(ctx, containerName)
	if err != nil {
		t.Fatalf("Failed to inspect environment container %s: %v", containerName, err)
	}

	bindings := inspect.HostConfig.PortBindings["3000/tcp"]
	if len(bindings) == 0 {
		t.Fatalf("Expected container port 3000/tcp to be published, got %+v", inspect.HostConfig.PortBindings)
	}
	if bindings[0].HostIP != "127.0.0.1" {
		t.Errorf("Expected HostIP 127.0.0.1, got '%s'", bindings[0].HostIP)
	}
	if !strings.HasSuffix(ws1.AccessURL, ":"+bindings[0].HostPort) {
		t.Errorf("Expected AccessURL '%s' to use host port '%s'", ws1.AccessURL, bindings[0].HostPort)
	}

	// 4. Inspección de la Red Docker deshabilitada ICC
	netInspect, err := rawCli.NetworkInspect(ctx, "solv-traefik-net", network.InspectOptions{})
	if err != nil {
		t.Fatalf("Failed to inspect network solv-traefik-net: %v", err)
	}

	if netInspect.Options["com.docker.network.bridge.enable_icc"] != "false" {
		t.Errorf("Expected enable_icc='false', got '%s'", netInspect.Options["com.docker.network.bridge.enable_icc"])
	}

	t.Logf("Slice 4 Integration Test PASSED! PracticeEnvironment ID: %s, Access URL: %s", ws1.ID, ws1.AccessURL)
}
