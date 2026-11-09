package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
)

func requestWithRole(role string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), domain.UserRoleKey, role)
	ctx = context.WithValue(ctx, domain.UserIDKey, "user-1")
	ctx = context.WithValue(ctx, domain.TenantIDKey, "tenant-1")
	return req.WithContext(ctx)
}

func TestAdminOnlyRejectsNonAdminRoles(t *testing.T) {
	for _, role := range []string{"", "student", "teacher", "unknown"} {
		t.Run(role, func(t *testing.T) {
			called := false
			handler := adminOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, requestWithRole(role))

			if rec.Code != http.StatusForbidden {
				t.Fatalf("rol %q: esperado 403, obtenido %d", role, rec.Code)
			}
			if called {
				t.Fatal("el handler administrativo no debe ejecutarse para un rol no autorizado")
			}
		})
	}
}

func TestAdminOnlyAllowsAdmin(t *testing.T) {
	called := false
	handler := adminOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithRole("admin"))

	if rec.Code != http.StatusNoContent || !called {
		t.Fatalf("admin debe conservar acceso: status=%d called=%v", rec.Code, called)
	}
}

func TestStaffOnlyAllowsTeacherAndAdmin(t *testing.T) {
	for _, role := range []string{"teacher", "admin"} {
		t.Run(role, func(t *testing.T) {
			called := false
			handler := staffOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, requestWithRole(role))

			if rec.Code != http.StatusNoContent || !called {
				t.Fatalf("rol %q debe conservar acceso: status=%d called=%v", role, rec.Code, called)
			}
		})
	}
}

func TestStaffOnlyRejectsStudentAndUnknownRoles(t *testing.T) {
	for _, role := range []string{"", "student", "unknown"} {
		t.Run(role, func(t *testing.T) {
			called := false
			handler := staffOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, requestWithRole(role))

			if rec.Code != http.StatusForbidden || called {
				t.Fatalf("rol %q: status=%d called=%v", role, rec.Code, called)
			}
		})
	}
}

func testIdentityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Header.Get("X-Test-Role")
		if role == "" {
			http.Error(w, "missing test role", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), domain.UserRoleKey, role)
		ctx = context.WithValue(ctx, domain.UserIDKey, "test-user")
		ctx = context.WithValue(ctx, domain.TenantIDKey, "test-tenant")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestAdminRoutesRejectStudentsAndTeachersBeforeHandlers(t *testing.T) {
	mux := http.NewServeMux()
	SetupRoutes(mux, &Handlers{
		AdminHandler:              &AdminHandler{},
		AdminAcademicHandler:      &AdminAcademicHandler{},
		ServerPoliciesHandler:     &ServerPoliciesHandler{},
		AISettingsHandler:         &AISettingsHandler{},
		AICopilotHandler:          &AICopilotHandler{},
		TenantLogoHandler:         &TenantLogoHandler{},
		EnvironmentRequestHandler: &EnvironmentRequestHandler{},
		BackupHandler:             &BackupHandler{},
		LanguageProfileHandler:    &LanguageProfileHandler{},
		EnvTestHandler:            &EnvTestHandler{},
		TenantMiddleware:          testIdentityMiddleware,
		AuditMiddleware:           func(next http.Handler) http.Handler { return next },
	})

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/audit-logs"},
		{http.MethodGet, "/api/v1/admin/audit-logs/actors/actor-1/timeline"},
		{http.MethodPut, "/api/v1/admin/branding"},
		{http.MethodGet, "/api/v1/admin/metrics/health"},
		{http.MethodGet, "/api/v1/admin/dashboard/courses-load"},
		{http.MethodGet, "/api/v1/admin/dashboard/incidents"},
		{http.MethodGet, "/api/v1/admin/dashboard/containers"},
		{http.MethodGet, "/api/v1/admin/courses/course-1/environments"},
		{http.MethodGet, "/api/v1/admin/environments/environment-1/logs"},
		{http.MethodPost, "/api/v1/admin/environments/environment-1/resolve"},
		{http.MethodGet, "/api/v1/admin/server/policies"},
		{http.MethodPut, "/api/v1/admin/server/policies"},
		{http.MethodPost, "/api/v1/admin/maintenance/enable"},
		{http.MethodPost, "/api/v1/admin/maintenance/disable"},
		{http.MethodGet, "/api/v1/admin/maintenance/status"},
		{http.MethodGet, "/api/v1/admin/academic-periods"},
		{http.MethodGet, "/api/v1/academic-periods"},
		{http.MethodPost, "/api/v1/admin/academic-periods"},
		{http.MethodPut, "/api/v1/admin/academic-periods/period-1"},
		{http.MethodPost, "/api/v1/admin/academic-periods/period-1/archive"},
		{http.MethodDelete, "/api/v1/admin/academic-periods/period-1"},
		{http.MethodPost, "/api/v1/admin/courses/course-1/reassign"},
		{http.MethodGet, "/api/v1/admin/students"},
		{http.MethodPost, "/api/v1/admin/students"},
		{http.MethodGet, "/api/v1/admin/students/student-1/courses"},
		{http.MethodPut, "/api/v1/admin/students/student-1/status"},
		{http.MethodPost, "/api/v1/admin/students/student-1/reset-oom"},
		{http.MethodPost, "/api/v1/admin/students/parse-roster"},
		{http.MethodPost, "/api/v1/admin/students/batch"},
		{http.MethodPost, "/api/v1/admin/subjects/subject-1/enroll-batch"},
		{http.MethodPost, "/api/v1/admin/subjects/subject-1/roster"},
		{http.MethodGet, "/api/v1/admin/administrators"},
		{http.MethodPost, "/api/v1/admin/administrators"},
		{http.MethodPost, "/api/v1/admin/administrators/admin-1/revoke"},
		{http.MethodGet, "/api/v1/admin/templates/capabilities"},
		{http.MethodPost, "/api/v1/admin/templates"},
		{http.MethodPut, "/api/v1/admin/templates/template-1"},
		{http.MethodPost, "/api/v1/admin/templates/compose-preview"},
		{http.MethodPost, "/api/v1/admin/template-components"},
		{http.MethodPut, "/api/v1/admin/template-components/component-1"},
		{http.MethodPatch, "/api/v1/admin/template-components/component-1"},
		{http.MethodPut, "/api/v1/admin/tech-components/component-1"},
		{http.MethodPatch, "/api/v1/admin/tech-components/component-1"},
		{http.MethodPut, "/api/v1/admin/components/component-1"},
		{http.MethodPatch, "/api/v1/admin/components/component-1"},
		{http.MethodDelete, "/api/v1/admin/template-components/component-1"},
		{http.MethodPost, "/api/v1/admin/templates/drafts"},
		{http.MethodGet, "/api/v1/admin/templates/drafts"},
		{http.MethodDelete, "/api/v1/admin/templates/drafts"},
		{http.MethodPut, "/api/v1/admin/templates/template-1/review"},
		{http.MethodGet, "/api/v1/admin/templates/local-images"},
		{http.MethodPost, "/api/v1/admin/templates/verify-image"},
		{http.MethodGet, "/api/v1/admin/templates/verify-image"},
		{http.MethodPost, "/api/v1/admin/templates/template-1/duplicate"},
		{http.MethodPost, "/api/v1/plantillas/template-1/duplicar"},
		{http.MethodPost, "/api/v1/admin/templates/template-1/promote-to-model"},
		{http.MethodGet, "/api/v1/admin/templates/cache-status"},
		{http.MethodPost, "/api/v1/admin/templates/template-1/warm"},
		{http.MethodPost, "/api/v1/admin/templates/warm-all"},
		{http.MethodGet, "/api/v1/admin/template-categories"},
		{http.MethodGet, "/api/v1/admin/manual"},
		{http.MethodPost, "/api/v1/admin/template-categories"},
		{http.MethodPut, "/api/v1/admin/template-categories/reorder"},
		{http.MethodPut, "/api/v1/admin/template-categories/category-1"},
		{http.MethodDelete, "/api/v1/admin/template-categories/category-1"},
		{http.MethodGet, "/api/v1/admin/template-models"},
		{http.MethodPut, "/api/v1/admin/template-models/model-1"},
		{http.MethodPost, "/api/v1/admin/template-models/model-1/deactivate"},
		{http.MethodPost, "/api/v1/admin/template-models/model-1/reactivate"},
		{http.MethodPost, "/api/v1/admin/emergency/action"},
		{http.MethodGet, "/api/v1/admin/environment-requests"},
		{http.MethodPatch, "/api/v1/admin/environment-requests/request-1"},
		{http.MethodPut, "/api/v1/admin/judge/profiles/go"},
		{http.MethodGet, "/api/v1/admin/backups/config"},
		{http.MethodPut, "/api/v1/admin/backups/config"},
		{http.MethodGet, "/api/v1/admin/backups"},
		{http.MethodPost, "/api/v1/admin/backups/trigger"},
		{http.MethodPost, "/api/v1/admin/backups/backup-1/verify"},
		{http.MethodGet, "/api/v1/admin/backups/backup-1/download"},
		{http.MethodGet, "/api/v1/admin/ai/settings"},
		{http.MethodPut, "/api/v1/admin/ai/settings"},
		{http.MethodPost, "/api/v1/admin/ai/test-connection"},
		{http.MethodPost, "/api/v1/admin/ai/models"},
		{http.MethodPost, "/api/v1/admin/ai/suggest-template"},
		{http.MethodPost, "/api/v1/admin/ai/suggest-component"},
		{http.MethodPost, "/api/v1/admin/ai/repair-template"},
		{http.MethodPost, "/api/v1/tenants/logo"},
	}

	for _, role := range []string{"student", "teacher"} {
		for _, route := range routes {
			t.Run(role+" "+route.method+" "+route.path, func(t *testing.T) {
				req := httptest.NewRequest(route.method, route.path, nil)
				req.Header.Set("X-Test-Role", role)
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)

				if rec.Code != http.StatusForbidden {
					t.Fatalf("rol %q: %s %s debía responder 403 antes del handler, obtuvo %d", role, route.method, route.path, rec.Code)
				}
			})
		}
	}
}

func TestEnvTestStartRejectsStudentBeforeHandler(t *testing.T) {
	mux := http.NewServeMux()
	SetupRoutes(mux, &Handlers{
		EnvTestHandler:   &EnvTestHandler{},
		TenantMiddleware: testIdentityMiddleware,
		AuditMiddleware:  func(next http.Handler) http.Handler { return next },
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/env-test", nil)
	req.Header.Set("X-Test-Role", "student")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("un alumno no debe iniciar un trabajo de evaluación de entorno: status=%d", rec.Code)
	}
}
