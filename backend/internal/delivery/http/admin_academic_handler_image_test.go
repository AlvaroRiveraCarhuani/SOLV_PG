package httpdelivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestAdminAcademicHandler_VerifyImage_Validation(t *testing.T) {
	imageService := services.NewImageVerificationService(nil)
	handler := NewAdminAcademicHandler(nil, nil, nil).WithImageService(imageService)

	t.Run("Rejects latest tag with 422", func(t *testing.T) {
		body := `{"image": "python:latest", "force": false}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/templates/verify-image", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.VerifyImage(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Rejects invalid format with 422", func(t *testing.T) {
		body := `{"image": "invalid format with spaces", "force": false}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/templates/verify-image", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.VerifyImage(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Valid image returns 200 with result", func(t *testing.T) {
		body := `{"image": "python:3.11-slim", "force": false}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/templates/verify-image", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.VerifyImage(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Code int                            `json:"code"`
			Data domain.ImageVerificationResult `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Data.ImageRef != "python:3.11-slim" {
			t.Errorf("expected ImageRef python:3.11-slim, got %s", resp.Data.ImageRef)
		}
	})
}

func TestAdminAcademicHandler_ListLocalImages(t *testing.T) {
	imageService := services.NewImageVerificationService(nil)
	handler := NewAdminAcademicHandler(nil, nil, nil).WithImageService(imageService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/templates/local-images", nil)
	w := httptest.NewRecorder()

	handler.ListLocalImages(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}
