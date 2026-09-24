package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetAdminManual(t *testing.T) {
	handler := &AdminAcademicHandler{}

	t.Run("GET method returns embedded markdown with 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/manual", nil)
		w := httptest.NewRecorder()

		handler.GetAdminManual(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/markdown") {
			t.Errorf("expected Content-Type to contain text/markdown, got %s", contentType)
		}

		body := w.Body.String()
		if len(body) == 0 {
			t.Fatal("expected non-empty manual body")
		}

		if !strings.Contains(body, "Manual de Administración") && !strings.Contains(body, "#") {
			t.Errorf("expected body to contain manual title or markdown headers, got:\n%s", body[:min(100, len(body))])
		}
	})

	t.Run("Non-GET method returns 405 Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/manual", nil)
		w := httptest.NewRecorder()

		handler.GetAdminManual(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected status 405, got %d", w.Code)
		}
	})
}
