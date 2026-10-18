package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockCacheGovRepo struct {
	templates map[string]*domain.AdminTemplateReviewItem
}

func (m *mockCacheGovRepo) ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) GetStudentCourses(ctx context.Context, tenantID, studentID string) ([]*domain.AdminStudentCourseItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) CreateStudent(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) UpdateStudentStatus(ctx context.Context, tenantID, studentID, status, reason string) error {
	return nil
}
func (m *mockCacheGovRepo) ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error) {
	return 0, nil
}
func (m *mockCacheGovRepo) ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error) {
	return true, nil
}
func (m *mockCacheGovRepo) ListTemplates(ctx context.Context, tenantID, status, search string) ([]*domain.AdminTemplateReviewItem, error) {
	list := make([]*domain.AdminTemplateReviewItem, 0, len(m.templates))
	for _, t := range m.templates {
		list = append(list, t)
	}
	return list, nil
}
func (m *mockCacheGovRepo) GetTemplateByID(ctx context.Context, id string) (*domain.AdminTemplateReviewItem, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, services.ErrTemplateNotFound
	}
	return t, nil
}
func (m *mockCacheGovRepo) GetTemplateForLanguage(ctx context.Context, envType, lang string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) ReviewTemplate(ctx context.Context, tenantID, templateID, adminID, status, rejectionReason string, baseRamMB *int) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) CreateOfficialTemplate(ctx context.Context, tenantID, adminID string, dto domain.CreateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) ListPendingAuditTemplates(ctx context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) UpdateAuditResults(ctx context.Context, templateID string, smokeStatus, smokeOutput, secStatus string, cveCritical, cveHigh int, secReportJSON []byte, finalStatus string) error {
	return nil
}
func (m *mockCacheGovRepo) DuplicateTemplate(ctx context.Context, tenantID, templateID, adminID string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) UpdateEOLStatus(ctx context.Context, templateID string, status, eolDate, message string) error {
	return nil
}
func (m *mockCacheGovRepo) TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockCacheGovRepo) HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockCacheGovRepo) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error {
	return nil
}
func (m *mockCacheGovRepo) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	return nil
}
func (m *mockCacheGovRepo) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error {
	return nil
}
func (m *mockCacheGovRepo) PromoteTemplateToModel(ctx context.Context, tenantID, templateID, adminID string, dto domain.PromoteTemplateToModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) GetDraftByUser(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockCacheGovRepo) DeleteDraft(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (m *mockCacheGovRepo) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	return nil, nil
}

type mockCacheOrch struct{}

func (m *mockCacheOrch) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	return nil
}
func (m *mockCacheOrch) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	return nil
}
func (m *mockCacheOrch) StartWorkspaceContainer(ctx context.Context, config domain.WorkspaceContainerConfig) (string, error) {
	return "", nil
}
func (m *mockCacheOrch) UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error {
	return nil
}
func (m *mockCacheOrch) GetContainerMetrics(ctx context.Context, containerID string) (*domain.ContainerMetrics, error) {
	return nil, nil
}
func (m *mockCacheOrch) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockCacheOrch) PauseContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockCacheOrch) UnpauseContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockCacheOrch) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockCacheOrch) RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error) {
	return nil, nil
}
func (m *mockCacheOrch) GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error) {
	return "", nil
}
func (m *mockCacheOrch) ExecuteCommandInBackground(ctx context.Context, containerID string, workDir string, cmd []string) error {
	return nil
}
func (m *mockCacheOrch) InspectLocalImage(ctx context.Context, imageRef string) (bool, int64, error) {
	return true, 2048000, nil
}
func (m *mockCacheOrch) PullImage(ctx context.Context, imageRef string) error {
	return nil
}

func TestAdminAcademicHandler_CacheEndpoints(t *testing.T) {
	repo := &mockCacheGovRepo{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tmpl-1": {
				ID:          "tmpl-1",
				Name:        "Test Template",
				DockerImage: "python:3.12-slim",
				Status:      "approved",
			},
		},
	}
	orch := &mockCacheOrch{}
	cacheSvc := services.NewTemplateCacheService(repo, orch)

	handler := NewAdminAcademicHandler(nil, nil, nil).WithCacheService(cacheSvc)

	t.Run("GetTemplatesCacheStatus returns 200 with list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/templates/cache-status", nil)
		w := httptest.NewRecorder()

		handler.GetTemplatesCacheStatus(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Code int                          `json:"code"`
			Data []domain.TemplateCacheStatus `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 status item, got %d", len(resp.Data))
		}
		if !resp.Data[0].IsCached {
			t.Errorf("expected is_cached to be true")
		}
	})

	t.Run("WarmTemplate returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/templates/tmpl-1/warm", nil)
		req.SetPathValue("id", "tmpl-1")
		w := httptest.NewRecorder()

		handler.WarmTemplate(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("WarmAllTemplates returns 200 with summary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/templates/warm-all", nil)
		w := httptest.NewRecorder()

		handler.WarmAllTemplates(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
