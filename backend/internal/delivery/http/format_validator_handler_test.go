package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockFormatRepo struct {
	mu        sync.Mutex
	exercises map[string]*domain.Exercise
}

func (m *mockFormatRepo) GetByID(_ context.Context, id string) (*domain.Exercise, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ex, ok := m.exercises[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ex, nil
}

func (m *mockFormatRepo) GetByIDAndTenant(ctx context.Context, id, _ string) (*domain.Exercise, error) {
	return m.GetByID(ctx, id)
}

func (m *mockFormatRepo) Create(_ context.Context, ex *domain.Exercise) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exercises[ex.ID] = ex
	return nil
}

func (m *mockFormatRepo) Update(_ context.Context, ex *domain.Exercise) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exercises[ex.ID] = ex
	return nil
}

func (m *mockFormatRepo) UpdateStatus(_ context.Context, id, _ string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[id]; ok {
		ex.Status = status
	}
	return nil
}

func (m *mockFormatRepo) UpdateConfig(_ context.Context, _, _ string, _ domain.ExerciseConfig) error {
	return nil
}

func (m *mockFormatRepo) UpdateExpectedJSON(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockFormatRepo) MarkExerciseStale(_ context.Context, _, _ string, _ bool) error {
	return nil
}

func (m *mockFormatRepo) UpdateExerciseLastValidDryRun(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

func (m *mockFormatRepo) CreateDryRunJob(_ context.Context, _ *domain.DryRunJob) error {
	return nil
}

func (m *mockFormatRepo) GetDryRunJob(_ context.Context, _ string) (*domain.DryRunJob, error) {
	return nil, nil
}

func (m *mockFormatRepo) UpdateDryRunJobProgress(_ context.Context, _ string, _ domain.DryRunJobStatus, _, _ int, _ *domain.EvaluationResult, _ string) error {
	return nil
}

func (m *mockFormatRepo) ListDueByStudent(_ context.Context, _, _ string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (m *mockFormatRepo) ListBySubject(_ context.Context, _, _ string) ([]*domain.Exercise, error) {
	return nil, nil
}

func TestValidateInputEndpoint(t *testing.T) {
	repo := &mockFormatRepo{exercises: make(map[string]*domain.Exercise)}
	evalSvc := services.NewEvaluationService(repo, nil, nil, nil)
	validate := validator.New()
	handler := httpdelivery.NewEvaluationHandler(evalSvc, validate)

	mux := http.NewServeMux()
	deps := &httpdelivery.Handlers{
		EvaluationHandler: handler,
		TenantMiddleware:  func(next http.Handler) http.Handler { return next },
		AuditMiddleware:   func(next http.Handler) http.Handler { return next },
	}
	httpdelivery.SetupRoutes(mux, deps)

	t.Run("Valid input against contract returns valid=true", func(t *testing.T) {
		payload := map[string]any{
			"contract": map[string]any{
				"version": 1,
				"input": map[string]any{
					"lines": []map[string]any{
						{"id": "n", "type": "int", "min": 1, "max": 10},
						{"id": "a", "type": "ints", "count": "=n", "item": map[string]any{"min": 1, "max": 100}},
					},
				},
			},
			"input": "3\n10 20 30",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/validate-input", bytes.NewReader(body))
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Valid bool   `json:"valid"`
				Error string `json:"error"`
			} `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Data.Valid {
			t.Errorf("expected valid=true, got valid=false, error: %s", resp.Data.Error)
		}
	})

	t.Run("Invalid input returns valid=false with error details in spanish", func(t *testing.T) {
		payload := map[string]any{
			"contract": map[string]any{
				"version": 1,
				"input": map[string]any{
					"lines": []map[string]any{
						{"id": "n", "type": "int", "min": 1, "max": 10},
						{"id": "a", "type": "ints", "count": "=n"},
					},
				},
			},
			"input": "3\n10 20",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises/validate-input", bytes.NewReader(body))
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var resp struct {
			Data struct {
				Valid bool   `json:"valid"`
				Error string `json:"error"`
			} `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Data.Valid {
			t.Errorf("expected valid=false, got valid=true")
		}
		if resp.Data.Error == "" {
			t.Errorf("expected error message in spanish, got empty")
		}
	})

	t.Run("CreateExercise with invalid testcase against contract returns 422", func(t *testing.T) {
		contractJSON := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int", "min": 1, "max": 5 }
				]
			}
		}`)

		ex := domain.Exercise{
			Title:       "Ejercicio con contrato",
			Description: "Desc",
			Type:        domain.ExerciseTypeAlgorithm,
			Config: domain.ExerciseConfig{
				Algorithm: &domain.AlgorithmConfig{
					InputFormat: contractJSON,
					TestCases: []domain.TestCase{
						{
							Input:          "100", // Violará max: 5
							ExpectedOutput: "100",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(ex)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/exercises", bytes.NewReader(body))
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422 Unprocessable Entity, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
