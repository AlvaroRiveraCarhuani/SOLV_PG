package httpdelivery_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

func TestGenerateCasesEndpoint(t *testing.T) {
	evalService := services.NewEvaluationService(nil, nil, nil, nil)
	validate := validator.New()
	handler := httpdelivery.NewEvaluationHandler(evalService, validate)

	validContract := `{
		"version": 1,
		"input": {
			"lines": [
				{ "id": "n", "type": "int", "min": 1, "max": 100 },
				{ "id": "a", "type": "ints", "count": "=n", "item": { "min": -50, "max": 50 } }
			]
		}
	}`

	t.Run("Generate 5 cases returns 5 structured inputs", func(t *testing.T) {
		payload := map[string]any{
			"contract": json.RawMessage(validContract),
			"count":    5,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/exercises/generate-cases", bytes.NewBuffer(body))
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.GenerateCases(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				Cases []struct {
					Input  string  `json:"input"`
					Output *string `json:"output"`
				} `json:"cases"`
			} `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data.Cases) != 5 {
			t.Fatalf("expected 5 cases, got %d", len(resp.Data.Cases))
		}

		for i, c := range resp.Data.Cases {
			if c.Input == "" {
				t.Errorf("case #%d has empty input", i+1)
			}
		}
	})

	t.Run("Deterministic generation with specified seed", func(t *testing.T) {
		seedVal := int64(98765)
		payload := map[string]any{
			"contract": json.RawMessage(validContract),
			"count":    3,
			"seed":     seedVal,
		}
		body, _ := json.Marshal(payload)

		req1 := httptest.NewRequest("POST", "/api/v1/exercises/generate-cases", bytes.NewBuffer(body))
		req1.Header.Set("X-User-Role", "teacher")
		w1 := httptest.NewRecorder()
		handler.GenerateCases(w1, req1)

		req2 := httptest.NewRequest("POST", "/api/v1/exercises/generate-cases", bytes.NewBuffer(body))
		req2.Header.Set("X-User-Role", "teacher")
		w2 := httptest.NewRecorder()
		handler.GenerateCases(w2, req2)

		if w1.Body.String() != w2.Body.String() {
			t.Errorf("expected deterministic outputs for same seed, got:\n1: %s\n2: %s", w1.Body.String(), w2.Body.String())
		}
	})

	t.Run("Invalid contract returns 400", func(t *testing.T) {
		invalidContract := `{"version": 1, "input": { "lines": [ { "id": "n", "type": "ints", "count": "=x" } ] } }`
		payload := map[string]any{
			"contract": json.RawMessage(invalidContract),
			"count":    5,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/exercises/generate-cases", bytes.NewBuffer(body))
		req.Header.Set("X-User-Role", "teacher")

		w := httptest.NewRecorder()
		handler.GenerateCases(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for invalid contract, got %d", w.Code)
		}
	})
}
