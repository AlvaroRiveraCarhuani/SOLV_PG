package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockGovRepoForTemplate struct {
	templates map[string]*domain.AdminTemplateReviewItem
	langTpls  map[string]*domain.AdminTemplateReviewItem
}

func (m *mockGovRepoForTemplate) ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) GetStudentCourses(ctx context.Context, tenantID, studentID string) ([]*domain.AdminStudentCourseItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) CreateStudent(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) UpdateStudentStatus(ctx context.Context, tenantID, studentID, status, reason string) error {
	return nil
}
func (m *mockGovRepoForTemplate) ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error) {
	return 0, nil
}
func (m *mockGovRepoForTemplate) ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error) {
	return true, nil
}
func (m *mockGovRepoForTemplate) ListTemplates(ctx context.Context, tenantID, status, search string) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) GetTemplateByID(ctx context.Context, id string) (*domain.AdminTemplateReviewItem, error) {
	if tpl, ok := m.templates[id]; ok {
		return tpl, nil
	}
	return nil, nil
}
func (m *mockGovRepoForTemplate) GetTemplateForLanguage(ctx context.Context, environmentType, language string) (*domain.AdminTemplateReviewItem, error) {
	key := environmentType + ":" + language
	if tpl, ok := m.langTpls[key]; ok {
		return tpl, nil
	}
	return nil, nil
}
func (m *mockGovRepoForTemplate) ReviewTemplate(ctx context.Context, tenantID, templateID, adminID, status, rejectionReason string, baseRamMB *int) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) CreateOfficialTemplate(ctx context.Context, tenantID, adminID string, dto domain.CreateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) ListPendingAuditTemplates(ctx context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) UpdateAuditResults(ctx context.Context, templateID string, smokeStatus, smokeOutput, secStatus string, cveCritical, cveHigh int, secReportJSON []byte, finalStatus string) error {
	return nil
}
func (m *mockGovRepoForTemplate) DuplicateTemplate(ctx context.Context, tenantID, templateID, adminID string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) UpdateEOLStatus(ctx context.Context, templateID string, status, eolDate, message string) error {
	return nil
}
func (m *mockGovRepoForTemplate) TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockGovRepoForTemplate) HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockGovRepoForTemplate) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error {
	return nil
}
func (m *mockGovRepoForTemplate) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	return nil
}
func (m *mockGovRepoForTemplate) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error {
	return nil
}
func (m *mockGovRepoForTemplate) PromoteTemplateToModel(ctx context.Context, tenantID, templateID, adminID string, dto domain.PromoteTemplateToModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) GetDraftByUser(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockGovRepoForTemplate) DeleteDraft(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (m *mockGovRepoForTemplate) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	return nil, nil
}

type mockExerciseRepoGov struct {
	exercises map[string]*domain.Exercise
}

func (m *mockExerciseRepoGov) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return m.exercises[id], nil
}
func (m *mockExerciseRepoGov) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.exercises[id], nil
}
func (m *mockExerciseRepoGov) Create(ctx context.Context, exercise *domain.Exercise) error {
	m.exercises[exercise.ID] = exercise
	return nil
}
func (m *mockExerciseRepoGov) Update(ctx context.Context, exercise *domain.Exercise) error {
	m.exercises[exercise.ID] = exercise
	return nil
}
func (m *mockExerciseRepoGov) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	if ex, ok := m.exercises[id]; ok {
		ex.Status = status
	}
	return nil
}
func (m *mockExerciseRepoGov) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error { return nil }
func (m *mockExerciseRepoGov) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error { return nil }
func (m *mockExerciseRepoGov) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error { return nil }
func (m *mockExerciseRepoGov) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error { return nil }
func (m *mockExerciseRepoGov) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error { return nil }
func (m *mockExerciseRepoGov) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) { return nil, nil }
func (m *mockExerciseRepoGov) UpdateDryRunJobProgress(ctx context.Context, jobID string, status domain.DryRunJobStatus, current, total int, result *domain.EvaluationResult, errMsg string) error { return nil }
func (m *mockExerciseRepoGov) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) { return nil, nil }
func (m *mockExerciseRepoGov) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) { return nil, nil }
func (m *mockExerciseRepoGov) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) { return nil, nil }

func TestExerciseTemplateGovernance_Success(t *testing.T) {
	exRepo := &mockExerciseRepoGov{exercises: make(map[string]*domain.Exercise)}
	govRepo := &mockGovRepoForTemplate{
		templates: make(map[string]*domain.AdminTemplateReviewItem),
		langTpls: map[string]*domain.AdminTemplateReviewItem{
			"JUEZ_EFIMERO:python": {
				ID:                "tpl-py-123",
				Name:              "Python 3 Runner",
				DockerImage:       "python:3.11-slim",
				Status:            "approved",
				TargetEnvironment: "JUEZ_EFIMERO",
				BaseRamMB:         256,
			},
		},
	}

	svc := services.NewEvaluationService(exRepo, nil, nil, nil)
	svc.SetAdminGovernanceRepository(govRepo)

	ex := &domain.Exercise{
		Title:    "Suma de dos números",
		Language: "python",
		Type:     domain.ExerciseTypeAlgorithm,
	}

	err := svc.CreateExercise(context.Background(), ex)
	if err != nil {
		t.Fatalf("unexpected error creating exercise: %v", err)
	}

	if ex.MemoryLimitMB != 256 {
		t.Errorf("expected MemoryLimitMB=256 derived from template, got %d", ex.MemoryLimitMB)
	}

	if ex.TemplateID != "tpl-py-123" {
		t.Errorf("expected TemplateID='tpl-py-123', got %s", ex.TemplateID)
	}

	if ex.Template == nil || ex.Template.BaseRamMB != 256 {
		t.Errorf("expected ex.Template summary attached with BaseRamMB=256")
	}
}

func TestExerciseTemplateGovernance_TemplateNotApproved(t *testing.T) {
	exRepo := &mockExerciseRepoGov{exercises: make(map[string]*domain.Exercise)}
	govRepo := &mockGovRepoForTemplate{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tpl-pending": {
				ID:                "tpl-pending",
				Name:              "Draft Template",
				Status:            "pending",
				TargetEnvironment: "JUEZ_EFIMERO",
				BaseRamMB:         512,
			},
		},
	}

	svc := services.NewEvaluationService(exRepo, nil, nil, nil)
	svc.SetAdminGovernanceRepository(govRepo)

	ex := &domain.Exercise{
		Title:      "Ejercicio invalido",
		Language:   "python",
		TemplateID: "tpl-pending",
	}

	err := svc.CreateExercise(context.Background(), ex)
	if err != domain.ErrTemplateNotApproved {
		t.Fatalf("expected ErrTemplateNotApproved, got %v", err)
	}
}

func TestExerciseTemplateGovernance_EnvironmentMismatch(t *testing.T) {
	exRepo := &mockExerciseRepoGov{exercises: make(map[string]*domain.Exercise)}
	govRepo := &mockGovRepoForTemplate{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tpl-judge": {
				ID:                "tpl-judge",
				Name:              "Judge Runner",
				Status:            "approved",
				TargetEnvironment: "JUEZ_EFIMERO",
				BaseRamMB:         128,
			},
		},
	}

	svc := services.NewEvaluationService(exRepo, nil, nil, nil)
	svc.SetAdminGovernanceRepository(govRepo)

	ex := &domain.Exercise{
		Title:           "Laboratorio Web",
		EnvironmentType: "IDE_PERSISTENTE",
		TemplateID:      "tpl-judge",
	}

	err := svc.CreateExercise(context.Background(), ex)
	if err != domain.ErrTemplateEnvironmentMismatch {
		t.Fatalf("expected ErrTemplateEnvironmentMismatch, got %v", err)
	}
}
