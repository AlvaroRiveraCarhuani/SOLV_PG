package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockImportRepoForHTTP struct {
	exercises map[string]*domain.Exercise
}

func newMockRepoHTTP() *mockImportRepoForHTTP {
	return &mockImportRepoForHTTP{
		exercises: make(map[string]*domain.Exercise),
	}
}

func (m *mockImportRepoForHTTP) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	if ex, ok := m.exercises[id]; ok {
		return ex, nil
	}
	return nil, errors.New("exercise not found")
}

func (m *mockImportRepoForHTTP) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.GetByID(ctx, id)
}

func (m *mockImportRepoForHTTP) Create(ctx context.Context, exercise *domain.Exercise) error {
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockImportRepoForHTTP) Update(ctx context.Context, exercise *domain.Exercise) error {
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockImportRepoForHTTP) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	return nil
}

func (m *mockImportRepoForHTTP) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	return nil
}

func (m *mockImportRepoForHTTP) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}

func (m *mockImportRepoForHTTP) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	return nil
}

func (m *mockImportRepoForHTTP) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	return nil
}

func (m *mockImportRepoForHTTP) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	return nil
}

func (m *mockImportRepoForHTTP) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	return nil, nil
}

func (m *mockImportRepoForHTTP) UpdateDryRunJobProgress(ctx context.Context, jobID string, status domain.DryRunJobStatus, current, total int, result *domain.EvaluationResult, errMsg string) error {
	return nil
}

func (m *mockImportRepoForHTTP) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (m *mockImportRepoForHTTP) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	return nil, nil
}

func (m *mockImportRepoForHTTP) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) {
	return nil, nil
}

func TestImportExercisesHTTP(t *testing.T) {
	repo := newMockRepoHTTP()
	evalService := services.NewEvaluationService(repo, nil, nil, nil)
	validate := validator.New()
	handler := httpdelivery.NewEvaluationHandler(evalService, validate)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/teacher/courses/{courseId}/exercises/import", handler.ImportExercises)

	validJSON := `[
		{
			"title": "Ejercicio Importado HTTP",
			"statement": "Enunciado de prueba.",
			"language": "python",
			"config": {
				"algorithm": {
					"test_cases": [{"input": "1\n", "expected_output": "1\n"}]
				}
			}
		}
	]`

	t.Run("Dry-run preview via multipart upload returns 200 OK", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "exercises.json")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write([]byte(validJSON))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/teacher/courses/course-999/exercises/import?dry_run=true", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				CanImport bool   `json:"can_import"`
				Mode      string `json:"mode"`
			} `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Data.CanImport {
			t.Errorf("expected CanImport=true, got false")
		}
		if resp.Data.Mode != "dry_run" {
			t.Errorf("expected mode 'dry_run', got %s", resp.Data.Mode)
		}
	})

	t.Run("Actual import creates exercise and returns 201 Created", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "exercises.json")
		part.Write([]byte(validJSON))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/teacher/courses/course-999/exercises/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		if len(repo.exercises) != 1 {
			t.Errorf("expected 1 exercise created in repository, got %d", len(repo.exercises))
		}
	})
}
