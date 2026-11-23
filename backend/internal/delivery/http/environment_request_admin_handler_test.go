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

// stubEnvironmentRequestRepo implementa solo GetByID: el handler de detalle no
// usa el resto del contrato, y el repositorio real filtra por tenant en la
// misma consulta.
type stubEnvironmentRequestRepo struct {
	domain.EnvironmentRequestRepository
	byTenant map[string]*domain.EnvironmentRequest
}

func (s *stubEnvironmentRequestRepo) GetByID(_ context.Context, tenantID, id string) (*domain.EnvironmentRequest, error) {
	if request, ok := s.byTenant[tenantID+"/"+id]; ok {
		return request, nil
	}
	return nil, domain.ErrEnvironmentRequestNotFound
}

const (
	adminRequestID    = "d0000000-0000-4000-8000-000000000001"
	adminRequestOther = "d0000000-0000-4000-8000-000000000002"
)

func newAdminRequestHandler() *EnvironmentRequestHandler {
	request := &domain.EnvironmentRequest{
		ID:                adminRequestID,
		TenantID:          "tenant-1",
		Title:             "Kotlin y Android para Programación Móvil",
		Status:            domain.EnvironmentRequestEnRevision,
		TargetEnvironment: domain.TargetEnvironmentIDEPersistente,
		StackConfig:       domain.StackConfig{RuntimeID: "java-21", DatabaseID: "db-none"},
		Components: []domain.EnvironmentRequestComponent{
			{ID: "java-21", Slug: "java-21", Name: "Java 21 LTS", Layer: "runtime"},
		},
		MissingComponents: []domain.EnvironmentRequestMissingComponent{
			{Name: "Kotlin 2.0", Layer: "runtime", Reason: "El catálogo no tiene runtime de Kotlin"},
		},
	}

	repo := &stubEnvironmentRequestRepo{
		byTenant: map[string]*domain.EnvironmentRequest{
			"tenant-1/" + adminRequestID: request,
		},
	}
	return NewEnvironmentRequestHandler(services.NewEnvironmentRequestService(repo, nil))
}

func TestGetAdminRequestReturnsFullDetail(t *testing.T) {
	handler := newAdminRequestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/environment-requests/"+adminRequestID, nil)
	req.SetPathValue("id", adminRequestID)
	req = withTestIdentity(req, "admin-1", "tenant-1", "admin")
	rec := httptest.NewRecorder()

	handler.GetAdminRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba 200, obtuve %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data domain.EnvironmentRequest `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("respuesta ilegible: %v", err)
	}
	if resp.Data.StackConfig.RuntimeID != "java-21" {
		t.Fatalf("la receta debe viajar completa, obtuve %#v", resp.Data.StackConfig)
	}
	if len(resp.Data.Components) != 1 || len(resp.Data.MissingComponents) != 1 {
		t.Fatalf("esperaba piezas reconocidas y faltantes, obtuve %#v / %#v", resp.Data.Components, resp.Data.MissingComponents)
	}
}

func TestGetAdminRequestUnknownIsNotFound(t *testing.T) {
	handler := newAdminRequestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/environment-requests/"+adminRequestOther, nil)
	req.SetPathValue("id", adminRequestOther)
	req = withTestIdentity(req, "admin-1", "tenant-1", "admin")
	rec := httptest.NewRecorder()

	handler.GetAdminRequest(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperaba 404, obtuve %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetAdminRequestDoesNotCrossTenants(t *testing.T) {
	handler := newAdminRequestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/environment-requests/"+adminRequestID, nil)
	req.SetPathValue("id", adminRequestID)
	req = withTestIdentity(req, "admin-2", "tenant-2", "admin")
	rec := httptest.NewRecorder()

	handler.GetAdminRequest(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperaba 404 al cruzar tenants, obtuve %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetAdminRequestRejectsInvalidID(t *testing.T) {
	handler := newAdminRequestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/environment-requests/no-es-uuid", nil)
	req.SetPathValue("id", "no-es-uuid")
	req = withTestIdentity(req, "admin-1", "tenant-1", "admin")
	rec := httptest.NewRecorder()
	handler.GetAdminRequest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperaba 400 con identificador inválido, obtuve %d", rec.Code)
	}

	missing := httptest.NewRequest(http.MethodGet, "/api/v1/admin/environment-requests/", nil)
	missing = withTestIdentity(missing, "admin-1", "tenant-1", "admin")
	rec = httptest.NewRecorder()
	handler.GetAdminRequest(rec, missing)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperaba 400 sin identificador, obtuve %d", rec.Code)
	}
}

func TestGetAdminRequestRejectsTeachers(t *testing.T) {
	handler := newAdminRequestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/environment-requests/"+adminRequestID, nil)
	req.SetPathValue("id", adminRequestID)
	req = withTestIdentity(req, "teacher-1", "tenant-1", "teacher")
	rec := httptest.NewRecorder()

	handler.GetAdminRequest(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperaba 403, obtuve %d", rec.Code)
	}
}
