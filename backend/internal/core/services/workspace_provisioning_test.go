package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockWorkspaceRepo struct {
	workspaces map[string]*domain.WorkspaceInstance
}

func newMockWorkspaceRepo() *mockWorkspaceRepo {
	return &mockWorkspaceRepo{
		workspaces: make(map[string]*domain.WorkspaceInstance),
	}
}

func (m *mockWorkspaceRepo) GetByStudentAndSubject(ctx context.Context, studentID, subjectID string) (*domain.WorkspaceInstance, error) {
	key := studentID + ":" + subjectID
	return m.workspaces[key], nil
}

func (m *mockWorkspaceRepo) GetByID(ctx context.Context, id string) (*domain.WorkspaceInstance, error) {
	return m.workspaces[id], nil
}

func (m *mockWorkspaceRepo) Create(ctx context.Context, ws *domain.WorkspaceInstance) error {
	key := ws.StudentID + ":" + ws.SubjectID
	m.workspaces[key] = ws
	m.workspaces[ws.ID] = ws
	return nil
}

func (m *mockWorkspaceRepo) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	if ws, ok := m.workspaces[id]; ok {
		ws.ContainerID = &containerID
	}
	return nil
}

func (m *mockWorkspaceRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if ws, ok := m.workspaces[id]; ok {
		ws.Status = status
	}
	return nil
}

func (m *mockWorkspaceRepo) UpdateMemoryLimit(ctx context.Context, id string, newLimitMB int64) error {
	if ws, ok := m.workspaces[id]; ok {
		ws.MemoryLimitMB = newLimitMB
	}
	return nil
}

func (m *mockWorkspaceRepo) RecordHeartbeat(ctx context.Context, id string) error {
	return nil
}

func (m *mockWorkspaceRepo) IncrementOOMStrike(ctx context.Context, id string) error {
	return nil
}

func (m *mockWorkspaceRepo) ResetOOMStrikes(ctx context.Context, id string) error {
	return nil
}

func (m *mockWorkspaceRepo) GetActiveWorkspaces(ctx context.Context) ([]*domain.WorkspaceInstance, error) {
	return nil, nil
}

func (m *mockWorkspaceRepo) GetAllRunningWorkspaces(ctx context.Context) ([]*domain.WorkspaceInstance, error) {
	return nil, nil
}

func (m *mockWorkspaceRepo) GetByType(ctx context.Context, wsType string) ([]*domain.WorkspaceInstance, error) {
	return nil, nil
}

func (m *mockWorkspaceRepo) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	return nil
}

type mockOrchestrator struct {
	lastConfig  *domain.WorkspaceContainerConfig
	lastExecCmd []string
}

func (m *mockOrchestrator) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	return nil
}

func (m *mockOrchestrator) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	return nil
}

func (m *mockOrchestrator) StartWorkspaceContainer(ctx context.Context, config domain.WorkspaceContainerConfig) (string, error) {
	cfgCopy := config
	m.lastConfig = &cfgCopy
	return "container-12345", nil
}

func (m *mockOrchestrator) UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error {
	return nil
}

func (m *mockOrchestrator) GetContainerMetrics(ctx context.Context, containerID string) (*domain.ContainerMetrics, error) {
	return &domain.ContainerMetrics{IsRunning: true}, nil
}

func (m *mockOrchestrator) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockOrchestrator) PauseContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockOrchestrator) UnpauseContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockOrchestrator) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockOrchestrator) RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error) {
	return nil, nil
}

func (m *mockOrchestrator) GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error) {
	return "", nil
}

func (m *mockOrchestrator) ExecuteCommandInBackground(ctx context.Context, containerID string, workDir string, cmd []string) error {
	m.lastExecCmd = cmd
	return nil
}

type mockHostMonitor struct{}

func (m *mockHostMonitor) CanAllocateMemory(mb int64) bool {
	return true
}

func (m *mockHostMonitor) GetHostMemoryStats() (float64, uint64, error) {
	return 50.0, 8192, nil
}

type mockSubjectRepo struct {
	subjects map[string]*domain.Subject
}

func (m *mockSubjectRepo) Create(ctx context.Context, subject *domain.Subject) error {
	return nil
}

func (m *mockSubjectRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Subject, error) {
	return m.subjects[id], nil
}

func (m *mockSubjectRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	return nil, nil
}

func (m *mockSubjectRepo) EnrollStudent(ctx context.Context, enrollment *domain.Enrollment) error {
	return nil
}

func (m *mockSubjectRepo) ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	return nil, nil
}

func (m *mockSubjectRepo) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Subject, error) {
	return nil, nil
}

func (m *mockSubjectRepo) ReassignTeacher(ctx context.Context, tenantID, subjectID, newTeacherID string) error {
	return nil
}

func (m *mockSubjectRepo) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	return nil
}

func (m *mockSubjectRepo) Update(ctx context.Context, tenantID, subjectID, name, code string) error {
	return nil
}

type mockTemplateRepo struct {
	templates map[string]*domain.Template
}

func (m *mockTemplateRepo) GetTemplateByID(ctx context.Context, id string) (*domain.Template, error) {
	return m.templates[id], nil
}

func TestWorkspaceService_TemplateProvisioning(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	monitor := &mockHostMonitor{}

	tplID := "tpl-python-ds"
	subjRepo := &mockSubjectRepo{
		subjects: map[string]*domain.Subject{
			"subj-101": {
				ID:         "subj-101",
				Name:       "Ciencia de Datos",
				TemplateID: &tplID,
			},
		},
	}

	tplRepo := &mockTemplateRepo{
		templates: map[string]*domain.Template{
			tplID: {
				ID:          tplID,
				Name:        "Python Data Science",
				DockerImage: "solv/python:3.12-ds",
				BaseRamMB:   1024,
				SetupScript: "pip install pandas numpy",
				ServicesConfig: domain.ServicesConfig{
					Services: []domain.ServiceRequirement{
						{Category: "database", Engine: "postgres"},
					},
				},
			},
		},
	}

	svc := services.NewWorkspaceService(wsRepo, orch, monitor).
		WithProvisioning(subjRepo, tplRepo, nil)

	ws, err := svc.StartWorkspace(context.Background(), "student-1", "subj-101")
	if err != nil {
		t.Fatalf("error inesperado al iniciar workspace con plantilla: %v", err)
	}

	// 1. Debe haber adoptado la memoria de la plantilla (1024 MB)
	if ws.MemoryLimitMB != 1024 {
		t.Errorf("esperado MemoryLimitMB = 1024, obtenido: %d", ws.MemoryLimitMB)
	}

	// 2. Debe haber registrado el TemplateID en la instancia
	if ws.TemplateID == nil || *ws.TemplateID != tplID {
		t.Errorf("esperado TemplateID = %s, obtenido: %v", tplID, ws.TemplateID)
	}

	// 3. El contenedor debe haber arrancado con la imagen especificada
	if orch.lastConfig == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if orch.lastConfig.Image != "solv/python:3.12-ds" {
		t.Errorf("esperado imagen solv/python:3.12-ds, obtenido: %s", orch.lastConfig.Image)
	}
	if orch.lastConfig.MemoryLimitMB != 1024 {
		t.Errorf("esperado container MemoryLimitMB = 1024, obtenido: %d", orch.lastConfig.MemoryLimitMB)
	}

	// 4. Esperar brevemente a que el goroutine dispare el setup script
	time.Sleep(1200 * time.Millisecond)
	if orch.lastExecCmd == nil {
		t.Errorf("esperado comando de setup_script ejecutado vía Docker Exec")
	} else {
		cmdStr := strings.Join(orch.lastExecCmd, " ")
		if !strings.Contains(cmdStr, "pip install pandas numpy") {
			t.Errorf("esperado script de setup en comando exec, obtenido: %s", cmdStr)
		}
	}
}

func TestWorkspaceService_DefaultFallbackWithoutTemplate(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	monitor := &mockHostMonitor{}

	// Sin materia registrada o sin template asignado
	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{}}

	svc := services.NewWorkspaceService(wsRepo, orch, monitor).
		WithProvisioning(subjRepo, tplRepo, nil)

	ws, err := svc.StartWorkspace(context.Background(), "student-2", "subj-general")
	if err != nil {
		t.Fatalf("error inesperado en fallback: %v", err)
	}

	// Debe caer a 256 MB y OpenVSCodeImage por defecto
	if ws.MemoryLimitMB != domain.DefaultBaseMemoryMB {
		t.Errorf("esperado DefaultBaseMemoryMB (%d), obtenido: %d", domain.DefaultBaseMemoryMB, ws.MemoryLimitMB)
	}
	if orch.lastConfig.Image != domain.OpenVSCodeImage {
		t.Errorf("esperado imagen default %s, obtenido: %s", domain.OpenVSCodeImage, orch.lastConfig.Image)
	}
}
