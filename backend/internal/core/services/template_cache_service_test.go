package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockOrchestratorForCache struct {
	inspectFunc func(ctx context.Context, imageRef string) (bool, int64, error)
	pullFunc    func(ctx context.Context, imageRef string) error
	pullCount   int
}

func (m *mockOrchestratorForCache) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	return nil
}
func (m *mockOrchestratorForCache) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	return nil
}
func (m *mockOrchestratorForCache) StartWorkspaceContainer(ctx context.Context, config domain.WorkspaceContainerConfig) (string, error) {
	return "test-container-id", nil
}
func (m *mockOrchestratorForCache) UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error {
	return nil
}
func (m *mockOrchestratorForCache) GetContainerMetrics(ctx context.Context, containerID string) (*domain.ContainerMetrics, error) {
	return nil, nil
}
func (m *mockOrchestratorForCache) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockOrchestratorForCache) PauseContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockOrchestratorForCache) UnpauseContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockOrchestratorForCache) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockOrchestratorForCache) RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error) {
	return nil, nil
}
func (m *mockOrchestratorForCache) GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error) {
	return "", nil
}
func (m *mockOrchestratorForCache) ExecuteCommandInBackground(ctx context.Context, containerID string, workDir string, cmd []string) error {
	return nil
}
func (m *mockOrchestratorForCache) InspectLocalImage(ctx context.Context, imageRef string) (bool, int64, error) {
	if m.inspectFunc != nil {
		return m.inspectFunc(ctx, imageRef)
	}
	return false, 0, nil
}
func (m *mockOrchestratorForCache) PullImage(ctx context.Context, imageRef string) error {
	m.pullCount++
	if m.pullFunc != nil {
		return m.pullFunc(ctx, imageRef)
	}
	return nil
}

type mockGovRepoForCache struct {
	templates map[string]*domain.AdminTemplateReviewItem
}

func (m *mockGovRepoForCache) ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) GetStudentCourses(ctx context.Context, tenantID, studentID string) ([]*domain.AdminStudentCourseItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) CreateStudent(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) UpdateStudentStatus(ctx context.Context, tenantID, studentID, status, reason string) error {
	return nil
}
func (m *mockGovRepoForCache) ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error) {
	return 0, nil
}
func (m *mockGovRepoForCache) ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error) {
	return true, nil
}
func (m *mockGovRepoForCache) BatchCreateStudents(ctx context.Context, tenantID string, emails []string) (int, error) {
	return 0, nil
}
func (m *mockGovRepoForCache) EnrollStudentsByEmails(ctx context.Context, tenantID, subjectID string, emails []string) (int, int, error) {
	return 0, 0, nil
}
func (m *mockGovRepoForCache) ListTemplates(ctx context.Context, tenantID, status, search string) ([]*domain.AdminTemplateReviewItem, error) {
	list := make([]*domain.AdminTemplateReviewItem, 0, len(m.templates))
	for _, t := range m.templates {
		list = append(list, t)
	}
	return list, nil
}
func (m *mockGovRepoForCache) GetTemplateByID(ctx context.Context, id string) (*domain.AdminTemplateReviewItem, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}
func (m *mockGovRepoForCache) GetTemplateForLanguage(ctx context.Context, envType, lang string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) ReviewTemplate(ctx context.Context, tenantID, templateID, adminID, status, rejectionReason string, baseRamMB *int) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) CreateOfficialTemplate(ctx context.Context, tenantID, adminID string, dto domain.CreateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) ListPendingAuditTemplates(ctx context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) UpdateAuditResults(ctx context.Context, templateID string, smokeStatus, smokeOutput, secStatus string, cveCritical, cveHigh int, secReportJSON []byte, finalStatus string) error {
	return nil
}
func (m *mockGovRepoForCache) DuplicateTemplate(ctx context.Context, tenantID, templateID, adminID string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) UpdateEOLStatus(ctx context.Context, templateID string, status, eolDate, message string) error {
	return nil
}
func (m *mockGovRepoForCache) TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockGovRepoForCache) HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockGovRepoForCache) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error {
	return nil
}
func (m *mockGovRepoForCache) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	return nil
}
func (m *mockGovRepoForCache) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error {
	return nil
}
func (m *mockGovRepoForCache) PromoteTemplateToModel(ctx context.Context, tenantID, templateID, adminID string, dto domain.PromoteTemplateToModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) GetDraftByUser(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockGovRepoForCache) DeleteDraft(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (m *mockGovRepoForCache) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	return nil, nil
}

func TestTemplateCacheService_GetTemplatesCacheStatus(t *testing.T) {
	ctx := context.Background()
	repo := &mockGovRepoForCache{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tmpl-1": {
				ID:          "tmpl-1",
				Name:        "Python Base",
				DockerImage: "python:3.12-slim",
				Status:      "approved",
			},
			"tmpl-2": {
				ID:          "tmpl-2",
				Name:        "Node Base",
				DockerImage: "node:20-alpine",
				Status:      "approved",
			},
		},
	}

	orch := &mockOrchestratorForCache{
		inspectFunc: func(ctx context.Context, imageRef string) (bool, int64, error) {
			if imageRef == "python:3.12-slim" {
				return true, 125000000, nil // 125 MB
			}
			return false, 0, nil
		},
	}

	svc := services.NewTemplateCacheService(repo, orch)
	statuses, err := svc.GetTemplatesCacheStatus(ctx, "tenant-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}

	for _, s := range statuses {
		switch s.DockerImage {
		case "python:3.12-slim":
			if !s.IsCached || s.Status != "cached" || s.SizeBytes != 125000000 {
				t.Errorf("expected python to be cached with size 125MB, got: %+v", s)
			}
		case "node:20-alpine":
			if s.IsCached || s.Status != "missing" {
				t.Errorf("expected node to be missing, got: %+v", s)
			}
		}
	}
}

func TestTemplateCacheService_WarmTemplate(t *testing.T) {
	ctx := context.Background()
	repo := &mockGovRepoForCache{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tmpl-missing": {
				ID:          "tmpl-missing",
				Name:        "Go Template",
				DockerImage: "golang:1.24-alpine",
				Status:      "approved",
			},
		},
	}

	pulled := false
	orch := &mockOrchestratorForCache{
		inspectFunc: func(ctx context.Context, imageRef string) (bool, int64, error) {
			if pulled {
				return true, 250000000, nil
			}
			return false, 0, nil
		},
		pullFunc: func(ctx context.Context, imageRef string) error {
			pulled = true
			return nil
		},
	}

	svc := services.NewTemplateCacheService(repo, orch)
	status, err := svc.WarmTemplate(ctx, "tenant-test", "tmpl-missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsCached || status.Status != "cached" || status.SizeBytes != 250000000 {
		t.Errorf("expected template to be warmed and cached, got: %+v", status)
	}
	if orch.pullCount != 1 {
		t.Errorf("expected 1 pull, got %d", orch.pullCount)
	}

	// Segundo llamado: ya está en caché, no debe disparar otro pull
	status2, err := svc.WarmTemplate(ctx, "tenant-test", "tmpl-missing")
	if err != nil {
		t.Fatalf("unexpected error on second warm: %v", err)
	}
	if !status2.IsCached {
		t.Errorf("expected still cached on second call")
	}
	if orch.pullCount != 1 {
		t.Errorf("expected pull count to remain 1, got %d", orch.pullCount)
	}
}

func TestTemplateCacheService_WarmAllActiveTemplates(t *testing.T) {
	ctx := context.Background()
	repo := &mockGovRepoForCache{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tmpl-1": {
				ID:          "tmpl-1",
				Name:        "T1",
				DockerImage: "img-1:1.0",
				Status:      "approved",
			},
			"tmpl-2": {
				ID:          "tmpl-2",
				Name:        "T2",
				DockerImage: "img-2:1.0",
				Status:      "approved",
			},
		},
	}

	orch := &mockOrchestratorForCache{
		inspectFunc: func(ctx context.Context, imageRef string) (bool, int64, error) {
			if imageRef == "img-1:1.0" {
				return true, 100000, nil // ya cálido
			}
			return false, 0, nil // img-2 falta
		},
		pullFunc: func(ctx context.Context, imageRef string) error {
			return nil
		},
	}

	svc := services.NewTemplateCacheService(repo, orch)
	summary, err := svc.WarmAllActiveTemplates(ctx, "tenant-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalChecked != 2 {
		t.Errorf("expected 2 checked, got %d", summary.TotalChecked)
	}
	if summary.AlreadyWarm != 1 {
		t.Errorf("expected 1 already warm, got %d", summary.AlreadyWarm)
	}
	if summary.WarmedCount != 1 {
		t.Errorf("expected 1 warmed, got %d", summary.WarmedCount)
	}
	if summary.FailedCount != 0 {
		t.Errorf("expected 0 failed, got %d", summary.FailedCount)
	}
}
