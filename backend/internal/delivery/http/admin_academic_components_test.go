package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type stubComponentGovRepo struct {
	mockCacheGovRepo
	components map[string]*domain.TemplateComponent
}

func newStubComponentGovRepo() *stubComponentGovRepo {
	r := &stubComponentGovRepo{
		components: make(map[string]*domain.TemplateComponent),
	}
	py := &domain.TemplateComponent{
		ID:                 "rt-python",
		Slug:               "python-312",
		Layer:              domain.ComponentLayerRuntime,
		SubCategory:        "backend",
		Name:               "Python 3.12",
		IconKey:            "python",
		BaseImage:          "python:3.12-slim-bookworm",
		CompatibleRuntimes: []string{"python"},
		ToolsProvided:      []string{"python3", "pip"},
		RamDeltaMB:         512,
		IsSystem:           true,
		IsActive:           true,
	}
	r.components[py.ID] = py
	r.components[py.Slug] = py
	return r
}

func (s *stubComponentGovRepo) ListTemplateComponents(ctx context.Context, tenantID, layer string, includeInactive bool) ([]*domain.TemplateComponent, error) {
	seen := make(map[string]bool)
	var list []*domain.TemplateComponent
	for _, it := range s.components {
		if seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		if layer != "" && string(it.Layer) != layer {
			continue
		}
		list = append(list, it)
	}
	return list, nil
}

func (s *stubComponentGovRepo) GetTemplateComponentByIDOrSlug(ctx context.Context, tenantID, idOrSlug string) (*domain.TemplateComponent, error) {
	if it, ok := s.components[idOrSlug]; ok {
		return it, nil
	}
	return nil, domain.ErrComponentNotFound
}

func (s *stubComponentGovRepo) CreateTemplateComponent(ctx context.Context, tenantID string, comp *domain.TemplateComponent) (*domain.TemplateComponent, error) {
	cp := *comp
	cp.ID = "comp-new-1"
	if cp.Slug == "" {
		cp.Slug = "comp-new-1"
	}
	cp.IsActive = true
	s.components[cp.ID] = &cp
	s.components[cp.Slug] = &cp
	return &cp, nil
}

func (s *stubComponentGovRepo) UpdateTemplateComponent(ctx context.Context, tenantID, id string, dto domain.UpdateTemplateComponentDTO, synthesizedLayer string) (*domain.TemplateComponent, error) {
	it, ok := s.components[id]
	if !ok {
		return nil, domain.ErrComponentNotFound
	}
	it.Name = dto.Name
	if synthesizedLayer != "" {
		it.BuildLayerSnippet = synthesizedLayer
	}
	return it, nil
}

func (s *stubComponentGovRepo) DeleteTemplateComponent(ctx context.Context, tenantID, id string) error {
	it, ok := s.components[id]
	if !ok {
		return domain.ErrComponentNotFound
	}
	if it.IsSystem {
		return domain.ErrComponentSystemLocked
	}
	delete(s.components, id)
	return nil
}

func (s *stubComponentGovRepo) UpdateOfficialTemplate(ctx context.Context, tenantID, templateID, adminID string, dto domain.UpdateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	return &domain.AdminTemplateReviewItem{
		ID:          templateID,
		Name:        dto.Name,
		DockerImage: dto.DockerImage,
		BaseRamMB:   dto.BaseRamMB,
	}, nil
}

func TestAdminAcademicHandler_ComponentsAndComposePreview(t *testing.T) {
	repo := newStubComponentGovRepo()
	govSvc := services.NewAdminGovernanceService(nil, repo)
	handler := NewAdminAcademicHandler(nil, nil, govSvc)

	// 1. POST componente inválido (intento de inyección en nombre de paquete) -> 422
	badBody := `{"name":"Malicious","layer":"addon","install_mode":"pip","package_names":["flask; rm -rf /"]}`
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/admin/template-components", bytes.NewBufferString(badBody))
	recBad := httptest.NewRecorder()
	handler.CreateTemplateComponent(recBad, reqBad)
	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on invalid package name, got %d: %s", recBad.Code, recBad.Body.String())
	}

	// 2. POST componente asistido válido -> 201
	goodBody := `{"name":"FastAPI","layer":"addon","install_mode":"pip","package_names":["fastapi","uvicorn"],"tools_provided":["python3"],"ram_delta_mb":64}`
	reqGood := httptest.NewRequest(http.MethodPost, "/api/v1/admin/template-components", bytes.NewBufferString(goodBody))
	recGood := httptest.NewRecorder()
	handler.CreateTemplateComponent(recGood, reqGood)
	if recGood.Code != http.StatusCreated {
		t.Fatalf("expected 201 on valid component, got %d: %s", recGood.Code, recGood.Body.String())
	}

	// 3. POST compose-preview -> 200 con plan compilado
	composeBody := `{"target_environment":"IDE_PERSISTENTE","stack_config":{"mode":"visual","runtime_id":"rt-python","addon_ids":["comp-new-1"],"workspace_folders":["backend"]}}`
	reqCompose := httptest.NewRequest(http.MethodPost, "/api/v1/admin/templates/compose-preview", bytes.NewBufferString(composeBody))
	recCompose := httptest.NewRecorder()
	handler.ComposeTemplatePreview(recCompose, reqCompose)
	if recCompose.Code != http.StatusOK {
		t.Fatalf("expected 200 on compose-preview, got %d: %s", recCompose.Code, recCompose.Body.String())
	}

	var resp struct {
		Data domain.ComposedTemplatePlan `json:"data"`
	}
	if err := json.Unmarshal(recCompose.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal compose response: %v", err)
	}
	if !resp.Data.NeedsLocalBuild {
		t.Fatalf("expected NeedsLocalBuild=true when addon has pip build layer")
	}
}
