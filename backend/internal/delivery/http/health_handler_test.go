package httpdelivery_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpdelivery "solv-backend/internal/delivery/http"
)

func TestHealthHandler_OK(t *testing.T) {
	handler := httpdelivery.NewHealthHandler(func(ctx context.Context) error { return nil })

	rec := httptest.NewRecorder()
	handler.GetHealth(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("el chequeo no debe cachearse: %q", rec.Header().Get("Cache-Control"))
	}

	var body struct {
		Data struct {
			Status   string `json:"status"`
			Database string `json:"database"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON: %v", err)
	}
	if body.Data.Status != "ok" || body.Data.Database != "ok" {
		t.Errorf("estado inesperado: %+v", body.Data)
	}
}

// El vigilante externo necesita un código distinto de 2xx para avisar: si la
// base no responde, el sitio está degradado aunque el proceso viva.
func TestHealthHandler_DatabaseUnavailable(t *testing.T) {
	handler := httpdelivery.NewHealthHandler(func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	rec := httptest.NewRecorder()
	handler.GetHealth(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("se esperaba 503, se obtuvo %d", rec.Code)
	}
}
