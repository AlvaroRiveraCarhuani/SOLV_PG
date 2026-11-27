package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type stubIncompatibleComponentRepo struct {
	domain.TemplateComponentRepository
	comps map[string]*domain.TemplateComponent
}

func (s *stubIncompatibleComponentRepo) GetTemplateComponentByIDOrSlug(_ context.Context, _, idOrSlug string) (*domain.TemplateComponent, error) {
	if comp, ok := s.comps[idOrSlug]; ok {
		return comp, nil
	}
	return nil, errors.New("componente no encontrado")
}

type stubIncompatibleRequestRepo struct {
	domain.EnvironmentRequestRepository
}

func (s *stubIncompatibleRequestRepo) Create(_ context.Context, req *domain.EnvironmentRequest) (*domain.EnvironmentRequest, error) {
	return req, nil
}

func (s *stubIncompatibleRequestRepo) FindPendingBySignature(context.Context, string, string) (*domain.EnvironmentRequest, error) {
	return nil, nil
}

func TestCreateEnvironmentRequestIncompatibleAddonReturns422(t *testing.T) {
	compRepo := &stubIncompatibleComponentRepo{comps: map[string]*domain.TemplateComponent{
		"python-312": {
			ID: "python-312", Slug: "python-312", Name: "Python 3.12", Layer: domain.ComponentLayerRuntime,
			BaseImage: "python:3.12-slim-bookworm", IconKey: "python", CompatibleRuntimes: []string{"python"},
		},
		"spring-maven": {
			ID: "spring-maven", Slug: "spring-maven", Name: "Maven + Gestión Java", Layer: domain.ComponentLayerAddon,
			IconKey: "spring", CompatibleRuntimes: []string{"java"},
		},
	}}
	handler := NewEnvironmentRequestHandler(services.NewEnvironmentRequestService(&stubIncompatibleRequestRepo{}, compRepo))

	body := `{"description":"Necesito un entorno para practicar APIs en clase.","uses":["practice"],"stack_config":{"runtime_id":"python-312","addon_ids":["spring-maven"]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/environment-requests", strings.NewReader(body))
	req = withTestIdentity(req, "teacher-1", "tenant-1", "teacher")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperaba 422 ante addon incompatible, obtuve %d. Body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "incompatible") {
		t.Fatalf("el mensaje debe explicar la incompatibilidad, obtuve %s", rec.Body.String())
	}
}
