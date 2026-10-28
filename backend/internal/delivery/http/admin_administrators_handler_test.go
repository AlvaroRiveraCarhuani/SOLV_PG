package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockAdminGovRepoForHTTP struct {
	admins          []*domain.AdminUserItem
	activeCount     int
	createAdminFunc func(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminUserItem, error)
	revokeAdminFunc func(ctx context.Context, tenantID, targetUserID string) error
	countActiveFunc func(ctx context.Context, tenantID string) (int, error)
	listAdminFunc   func(ctx context.Context, tenantID string) ([]*domain.AdminUserItem, error)
}

func (m *mockAdminGovRepoForHTTP) ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) GetStudentCourses(ctx context.Context, tenantID, studentID string) ([]*domain.AdminStudentCourseItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) CreateStudent(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) UpdateStudentStatus(ctx context.Context, tenantID, studentID, status, reason string) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error) {
	return 0, nil
}
func (m *mockAdminGovRepoForHTTP) ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error) {
	return true, nil
}
func (m *mockAdminGovRepoForHTTP) BatchCreateStudents(ctx context.Context, tenantID string, emails []string) (int, error) {
	return 0, nil
}
func (m *mockAdminGovRepoForHTTP) EnrollStudentsByEmails(ctx context.Context, tenantID, subjectID string, emails []string) (int, int, error) {
	return 0, 0, nil
}
func (m *mockAdminGovRepoForHTTP) ListTemplates(ctx context.Context, tenantID, status, search string) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) GetTemplateByID(ctx context.Context, id string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) GetTemplateForLanguage(ctx context.Context, environmentType, language string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) ReviewTemplate(ctx context.Context, tenantID, templateID, adminID, status, rejectionReason string, baseRamMB *int) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) CreateOfficialTemplate(ctx context.Context, tenantID, adminID string, dto domain.CreateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) ListPendingAuditTemplates(ctx context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) UpdateAuditResults(ctx context.Context, templateID string, smokeStatus, smokeOutput, secStatus string, cveCritical, cveHigh int, secReportJSON []byte, finalStatus string) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) DuplicateTemplate(ctx context.Context, tenantID, templateID, adminID string) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) UpdateEOLStatus(ctx context.Context, templateID string, status, eolDate, message string) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockAdminGovRepoForHTTP) HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockAdminGovRepoForHTTP) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) PromoteTemplateToModel(ctx context.Context, tenantID, templateID, adminID string, dto domain.PromoteTemplateToModelDTO) (*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) GetDraftByUser(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockAdminGovRepoForHTTP) DeleteDraft(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (m *mockAdminGovRepoForHTTP) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	return nil, nil
}

func (m *mockAdminGovRepoForHTTP) ListAdministrators(ctx context.Context, tenantID string) ([]*domain.AdminUserItem, error) {
	if m.listAdminFunc != nil {
		return m.listAdminFunc(ctx, tenantID)
	}
	return m.admins, nil
}

func (m *mockAdminGovRepoForHTTP) CountActiveAdministrators(ctx context.Context, tenantID string) (int, error) {
	if m.countActiveFunc != nil {
		return m.countActiveFunc(ctx, tenantID)
	}
	return m.activeCount, nil
}

func (m *mockAdminGovRepoForHTTP) CreateAdministrator(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminUserItem, error) {
	if m.createAdminFunc != nil {
		return m.createAdminFunc(ctx, tenantID, email, firstName, lastName)
	}
	item := &domain.AdminUserItem{
		ID:        "admin-created-uuid",
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      "admin",
		Status:    "active",
		Origin:    "manual",
		CreatedAt: time.Now(),
	}
	m.admins = append(m.admins, item)
	return item, nil
}

func (m *mockAdminGovRepoForHTTP) RevokeAdministrator(ctx context.Context, tenantID, targetUserID string) error {
	if m.revokeAdminFunc != nil {
		return m.revokeAdminFunc(ctx, tenantID, targetUserID)
	}
	return nil
}

func setupAdminHandler(repo *mockAdminGovRepoForHTTP) (*httpdelivery.AdminAcademicHandler, *http.ServeMux) {
	govSvc := services.NewAdminGovernanceService(nil, repo)
	handler := httpdelivery.NewAdminAcademicHandler(nil, nil, govSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/admin/administrators", handler.ListAdministrators)
	mux.HandleFunc("POST /api/v1/admin/administrators", handler.CreateAdministrator)
	mux.HandleFunc("POST /api/v1/admin/administrators/{id}/revoke", handler.RevokeAdministrator)

	return handler, mux
}

func TestHTTP_ListAdministrators_ForbiddenForNonAdmin(t *testing.T) {
	_, mux := setupAdminHandler(&mockAdminGovRepoForHTTP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/administrators", nil)
	req.Header.Set("X-User-Role", "student")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rec.Code)
	}
}

func TestHTTP_ListAdministrators_Success(t *testing.T) {
	repo := &mockAdminGovRepoForHTTP{
		admins: []*domain.AdminUserItem{
			{ID: "admin-1", Email: "admin1@uab.edu.bo", FirstName: "Admin", LastName: "Uno", Role: "admin", Status: "active"},
		},
	}
	_, mux := setupAdminHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/administrators", nil)
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Tenant-Id", "tenant-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp httpdelivery.GlobalResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if resp.Message != "Administradores obtenidos exitosamente" {
		t.Errorf("unexpected message: %q", resp.Message)
	}
}

func TestHTTP_CreateAdministrator_ValidationErrors(t *testing.T) {
	repo := &mockAdminGovRepoForHTTP{}
	_, mux := setupAdminHandler(repo)

	// Email inválido
	body, _ := json.Marshal(domain.CreateAdminDTO{
		FirstName: "Juan",
		LastName:  "Perez",
		Email:     "no-valid-email",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators", bytes.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "admin-super")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid email, got %d", rec.Code)
	}

	// Nombre o apellido faltante
	bodyEmptyName, _ := json.Marshal(domain.CreateAdminDTO{
		FirstName: "",
		LastName:  "Perez",
		Email:     "juan@uab.edu.bo",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators", bytes.NewReader(bodyEmptyName))
	req2.Header.Set("X-User-Role", "admin")
	req2.Header.Set("X-User-Id", "admin-super")
	rec2 := httptest.NewRecorder()

	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty name, got %d", rec2.Code)
	}
}

func TestHTTP_CreateAdministrator_Success(t *testing.T) {
	repo := &mockAdminGovRepoForHTTP{}
	_, mux := setupAdminHandler(repo)

	body, _ := json.Marshal(domain.CreateAdminDTO{
		FirstName: "Ana",
		LastName:  "Rios",
		Email:     "ana.rios@uab.edu.bo",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators", bytes.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "admin-super")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTTP_RevokeAdministrator_CannotRevokeSelf(t *testing.T) {
	repo := &mockAdminGovRepoForHTTP{activeCount: 3}
	_, mux := setupAdminHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators/admin-self/revoke", nil)
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "admin-self")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", rec.Code)
	}
}

func TestHTTP_RevokeAdministrator_LastAdministratorBlocked(t *testing.T) {
	repo := &mockAdminGovRepoForHTTP{activeCount: 1}
	_, mux := setupAdminHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators/admin-target/revoke", nil)
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "admin-super")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", rec.Code)
	}
}

func TestHTTP_RevokeAdministrator_Success(t *testing.T) {
	repo := &mockAdminGovRepoForHTTP{activeCount: 2}
	_, mux := setupAdminHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators/admin-target/revoke", nil)
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "admin-super")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
}
