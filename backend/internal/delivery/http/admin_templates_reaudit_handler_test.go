package httpdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// ResetTemplateAudit del mock reproduce el reinicio de auditoría del
// repositorio real: solo toca las columnas de smoke y escaneo.
func (m *mockCacheGovRepo) ResetTemplateAudit(
	ctx context.Context,
	tenantID, templateID string,
) (*domain.AdminTemplateReviewItem, error) {
	tpl := m.templates[templateID]
	if tpl == nil {
		return nil, fmt.Errorf("template not found")
	}
	updated := *tpl
	updated.SmokeTestStatus = "pending"
	updated.SmokeTestOutput = ""
	updated.SecurityAuditStatus = "pending"
	updated.CVECriticalCount = 0
	updated.CVEHighCount = 0
	return &updated, nil
}

func TestAdminAcademicHandler_ReauditTemplate(t *testing.T) {
	repo := &mockCacheGovRepo{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tmpl-ok": {
				ID:                  "tmpl-ok",
				Name:                "Go Runner Base",
				DockerImage:         "golang:1.24-alpine",
				Status:              "APROBADA",
				SmokeTestStatus:     "skipped",
				SecurityAuditStatus: "failed",
				CVECriticalCount:    2,
			},
			"tmpl-paused": {
				ID:     "tmpl-paused",
				Name:   "Pausada",
				Status: "PAUSADA",
			},
		},
	}
	govService := services.NewAdminGovernanceService(nil, repo)
	handler := NewAdminAcademicHandler(nil, nil, govService)

	t.Run("reencola y responde 202 con la plantilla actualizada", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/admin/templates/tmpl-ok/audit", nil)
		req.SetPathValue("id", "tmpl-ok")
		w := httptest.NewRecorder()

		handler.ReauditTemplate(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("esperaba 202, obtuve %d. Body: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Data domain.AdminTemplateReviewItem `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		if resp.Data.SmokeTestStatus != "pending" || resp.Data.SecurityAuditStatus != "pending" {
			t.Fatalf("esperaba smoke y escaneo en pending, obtuve %q / %q", resp.Data.SmokeTestStatus, resp.Data.SecurityAuditStatus)
		}
		if resp.Data.Status != "APROBADA" {
			t.Fatalf("el estado del catálogo no debía cambiar, obtuve %q", resp.Data.Status)
		}
		if resp.Data.CVECriticalCount != 0 {
			t.Fatalf("los conteos debían reiniciarse, obtuve %d", resp.Data.CVECriticalCount)
		}
	})

	t.Run("plantilla inexistente devuelve 404", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/admin/templates/nope/audit", nil)
		req.SetPathValue("id", "nope")
		w := httptest.NewRecorder()

		handler.ReauditTemplate(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperaba 404, obtuve %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("un estado de revisión humana devuelve 409", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/admin/templates/tmpl-paused/audit", nil)
		req.SetPathValue("id", "tmpl-paused")
		w := httptest.NewRecorder()

		handler.ReauditTemplate(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("esperaba 409, obtuve %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
