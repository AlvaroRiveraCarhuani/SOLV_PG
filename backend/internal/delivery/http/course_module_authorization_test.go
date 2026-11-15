package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Las escrituras de módulos viven bajo /teacher pero no llevaban puerta de rol:
// cualquier alumno autenticado podía editar o borrar módulos con un id.
func TestCourseModuleWritesRejectStudentsBeforeHandlers(t *testing.T) {
	mux := http.NewServeMux()
	SetupRoutes(mux, &Handlers{
		CourseModuleHandler: &CourseModuleHandler{},
		TenantMiddleware:    testIdentityMiddleware,
		AuditMiddleware:     func(next http.Handler) http.Handler { return next },
	})

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/teacher/courses/curso-1/modules"},
		{http.MethodPut, "/api/v1/teacher/modules/modulo-1"},
		{http.MethodDelete, "/api/v1/teacher/modules/modulo-1"},
		{http.MethodPost, "/api/v1/teacher/modules/modulo-1/prerequisites"},
		{http.MethodDelete, "/api/v1/teacher/modules/modulo-1/prerequisites"},
		{http.MethodPut, "/api/v1/teacher/exercises/ejercicio-1/module"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			req.Header.Set("X-Test-Role", "student")
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("un alumno debe recibir 403 antes del handler, se obtuvo %d", rec.Code)
			}
		})
	}
}
