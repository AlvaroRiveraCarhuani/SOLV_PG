package integration

import (
	"context"
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

type MockWorkspaceRepository struct {
	workspaces map[string]*domain.WorkspaceInstance
}

func NewMockWorkspaceRepository() *MockWorkspaceRepository {
	return &MockWorkspaceRepository{
		workspaces: make(map[string]*domain.WorkspaceInstance),
	}
}

func (m *MockWorkspaceRepository) GetByStudentAndSubject(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	key := studentID + ":" + subjectID
	if ws, exists := m.workspaces[key]; exists {
		return ws, nil
	}
	return nil, nil
}

func (m *MockWorkspaceRepository) GetByID(ctx context.Context, id string) (*domain.WorkspaceInstance, error) {
	if ws, exists := m.workspaces[id]; exists {
		return ws, nil
	}
	return nil, nil
}

func (m *MockWorkspaceRepository) Create(ctx context.Context, workspace *domain.WorkspaceInstance) error {
	key := workspace.StudentID + ":" + workspace.SubjectID
	m.workspaces[key] = workspace
	m.workspaces[workspace.ID] = workspace
	return nil
}

func (m *MockWorkspaceRepository) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.ContainerID = &containerID
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockWorkspaceRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.Status = status
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockWorkspaceRepository) UpdateMemoryLimit(ctx context.Context, id string, memoryMB int64) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.MemoryLimitMB = memoryMB
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockWorkspaceRepository) RecordHeartbeat(ctx context.Context, id string) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.LastHeartbeatAt = time.Now()
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockWorkspaceRepository) IncrementOOMStrike(ctx context.Context, id string) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.OOMStrikeCount++
		now := time.Now()
		ws.LastOOMKilledAt = &now
		ws.Status = domain.WorkspaceStatusOOMKilled
		ws.UpdatedAt = now
	}
	return nil
}

func (m *MockWorkspaceRepository) ResetOOMStrikes(ctx context.Context, id string) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.OOMStrikeCount = 0
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func (m *MockWorkspaceRepository) GetActiveWorkspaces(ctx context.Context) ([]*domain.WorkspaceInstance, error) {
	var active []*domain.WorkspaceInstance
	for _, ws := range m.workspaces {
		if ws.Status == domain.WorkspaceStatusRunning || ws.Status == domain.WorkspaceStatusPending {
			active = append(active, ws)
		}
	}
	return active, nil
}

func (m *MockWorkspaceRepository) GetAllRunningWorkspaces(ctx context.Context) ([]*domain.WorkspaceInstance, error) {
	var running []*domain.WorkspaceInstance
	for _, ws := range m.workspaces {
		if ws.Status == domain.WorkspaceStatusRunning {
			running = append(running, ws)
		}
	}
	return running, nil
}

func (m *MockWorkspaceRepository) GetByType(ctx context.Context, workspaceType string) ([]*domain.WorkspaceInstance, error) {
	seen := make(map[string]bool)
	var result []*domain.WorkspaceInstance
	for _, ws := range m.workspaces {
		if ws.Type == workspaceType && !seen[ws.ID] {
			seen[ws.ID] = true
			result = append(result, ws)
		}
	}
	return result, nil
}

func (m *MockWorkspaceRepository) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	if ws, exists := m.workspaces[id]; exists {
		ws.SemgrepAudit = auditJSON
		ws.UpdatedAt = time.Now()
	}
	return nil
}

func TestWorkspaceServiceStartAndIdempotency(t *testing.T) {
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

	// 1. Iniciar Entorno de Desarrollo Interactivo (Code-Server)
	ws1, err := service.StartWorkspace(ctx, studentID, subjectID)
	if err != nil {
		t.Fatalf("StartWorkspace failed: %v", err)
	}

	if ws1.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", ws1.Status)
	}

	if ws1.ID == "" {
		t.Errorf("Expected non-empty workspace_id UUID")
	}

	expectedURL := "http://" + ws1.ID + ".solv.local"
	if ws1.AccessURL != expectedURL {
		t.Errorf("Expected AccessURL '%s', got '%s'", expectedURL, ws1.AccessURL)
	}

	// Clean up container and volume
	if ws1.ContainerID != nil {
		defer func() {
			_ = rawCli.ContainerRemove(context.Background(), *ws1.ContainerID, container.RemoveOptions{Force: true})
		}()
	}

	// 2. Verificar Idempotencia (segunda solicitud devuelve la misma instancia)
	ws2, err := service.StartWorkspace(ctx, studentID, subjectID)
	if err != nil {
		t.Fatalf("Second StartWorkspace call failed: %v", err)
	}

	if ws2.ID != ws1.ID {
		t.Errorf("Idempotency violation: expected same workspace ID '%s', got '%s'", ws1.ID, ws2.ID)
	}

	// 3. Inspección del Contenedor de Docker y Etiquetas de Traefik v3
	containerName := "solv-workspace-" + ws1.ID
	inspect, err := rawCli.ContainerInspect(ctx, containerName)
	if err != nil {
		t.Fatalf("Failed to inspect workspace container %s: %v", containerName, err)
	}

	// Verificar etiqueta Traefik router rule
	ruleKey := "traefik.http.routers." + ws1.ID + ".rule"
	expectedRule := "Host(`" + ws1.ID + ".solv.local`)"
	if inspect.Config.Labels[ruleKey] != expectedRule {
		t.Errorf("Expected label %s='%s', got '%s'", ruleKey, expectedRule, inspect.Config.Labels[ruleKey])
	}

	// Verificar puerto del servicio Traefik (3000)
	portKey := "traefik.http.services." + ws1.ID + ".loadbalancer.server.port"
	if inspect.Config.Labels[portKey] != "3000" {
		t.Errorf("Expected label %s='3000', got '%s'", portKey, inspect.Config.Labels[portKey])
	}

	// 4. Inspección de la Red Docker deshabilitada ICC
	netInspect, err := rawCli.NetworkInspect(ctx, "solv-traefik-net", network.InspectOptions{})
	if err != nil {
		t.Fatalf("Failed to inspect network solv-traefik-net: %v", err)
	}

	if netInspect.Options["com.docker.network.bridge.enable_icc"] != "false" {
		t.Errorf("Expected enable_icc='false', got '%s'", netInspect.Options["com.docker.network.bridge.enable_icc"])
	}

	t.Logf("Slice 4 Integration Test PASSED! Workspace ID: %s, Access URL: %s", ws1.ID, ws1.AccessURL)
}
