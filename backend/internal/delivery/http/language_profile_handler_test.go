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

type mockLangRepo struct {
	profiles map[string]*domain.LanguageProfile
}

func (m *mockLangRepo) ListProfiles(ctx context.Context) ([]*domain.LanguageProfile, error) {
	list := make([]*domain.LanguageProfile, 0, len(m.profiles))
	for _, p := range m.profiles {
		list = append(list, p)
	}
	return list, nil
}

func (m *mockLangRepo) GetProfileByLanguage(ctx context.Context, language string) (*domain.LanguageProfile, error) {
	p, ok := m.profiles[language]
	if !ok {
		return nil, domain.ErrUnknownLanguage
	}
	copied := *p
	return &copied, nil
}

func (m *mockLangRepo) UpdateProfile(ctx context.Context, profile *domain.LanguageProfile, audit *domain.LanguageProfileAudit) error {
	if _, ok := m.profiles[profile.Language]; !ok {
		return domain.ErrUnknownLanguage
	}
	m.profiles[profile.Language] = profile
	return nil
}

func setupTestMux() (*http.ServeMux, *mockLangRepo) {
	repo := &mockLangRepo{
		profiles: map[string]*domain.LanguageProfile{
			"python": {
				Language:            "python",
				DefaultTimeoutMS:    2000,
				DefaultMemoryMB:     256,
				Image:               "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93",
				BuildCommand:        "",
				BuildTimeoutMS:      10000,
				BuildMemoryMB:       512,
				P95WindowDays:       30,
				CheckerSidecarImage: "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93",
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			},
		},
	}
	svc := services.NewLanguageProfileService(repo)
	handler := httpdelivery.NewLanguageProfileHandler(svc)

	mux := http.NewServeMux()
	deps := &httpdelivery.Handlers{
		LanguageProfileHandler: handler,
		TenantMiddleware:       func(next http.Handler) http.Handler { return next },
		AuditMiddleware:        func(next http.Handler) http.Handler { return next },
	}
	httpdelivery.SetupRoutes(mux, deps)

	return mux, repo
}

func TestLanguageProfileEndpoints(t *testing.T) {
	mux, _ := setupTestMux()

	// 1. GET /api/v1/judge/profiles -> 200 OK
	req := httptest.NewRequest("GET", "/api/v1/judge/profiles", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}

	// 2. GET /api/v1/judge/profiles/c++ -> 400 Bad Request (rechazo de alias no canónico)
	req = httptest.NewRequest("GET", "/api/v1/judge/profiles/c++", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for non-canonical c++, got: %d", rec.Code)
	}

	// 3. PUT /api/v1/admin/judge/profiles/python sin rol admin -> 403 Forbidden
	bodyBytes, _ := json.Marshal(map[string]any{
		"default_memory_mb": 512,
		"reason":            "test",
	})
	req = httptest.NewRequest("PUT", "/api/v1/admin/judge/profiles/python", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-Role", "teacher")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got: %d", rec.Code)
	}

	// 4. PUT /api/v1/admin/judge/profiles/python con admin -> 200 OK
	req = httptest.NewRequest("PUT", "/api/v1/admin/judge/profiles/python", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "admin-user-1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d, body: %s", rec.Code, rec.Body.String())
	}
}
