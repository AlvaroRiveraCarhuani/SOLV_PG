package httpdelivery_test

import (
	"context"
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

type mockExerciseRepoHTTP struct {
	mu        sync.Mutex
	exercises map[string]*domain.Exercise
	jobs      map[string]*domain.DryRunJob
}

func (m *mockExerciseRepoHTTP) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ex, ok := m.exercises[id]
	if !ok {
		return nil, domain.ErrUnknownLanguage
	}
	return ex, nil
}

func (m *mockExerciseRepoHTTP) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.GetByID(ctx, id)
}

func (m *mockExerciseRepoHTTP) Create(ctx context.Context, exercise *domain.Exercise) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockExerciseRepoHTTP) Update(ctx context.Context, exercise *domain.Exercise) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockExerciseRepoHTTP) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[id]; ok {
		ex.Status = status
		return nil
	}
	return nil
}

func (m *mockExerciseRepoHTTP) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	return nil
}

func (m *mockExerciseRepoHTTP) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}

func (m *mockExerciseRepoHTTP) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	return nil
}

func (m *mockExerciseRepoHTTP) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[exerciseID]; ok {
		ex.Stale = false
		ex.LastValidDryRunAt = &dryRunAt
	}
	return nil
}

func (m *mockExerciseRepoHTTP) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *mockExerciseRepoHTTP) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return nil, domain.ErrUnknownLanguage
	}
	return j, nil
}

func (m *mockExerciseRepoHTTP) UpdateDryRunJobProgress(ctx context.Context, jobID string, status domain.DryRunJobStatus, current, total int, result *domain.EvaluationResult, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = status
		j.ProgressCurrent = current
		j.ProgressTotal = total
	}
	return nil
}

func (m *mockExerciseRepoHTTP) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (m *mockExerciseRepoHTTP) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	return nil, nil
}

func (m *mockExerciseRepoHTTP) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) {
	return nil, nil
}

type mockRunnerHTTP struct{}

func (r *mockRunnerHTTP) RunTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return domain.TestCaseRunResult{
		Verdict:       domain.VerdictAC,
		ExecutionTime: 5 * time.Millisecond,
		ImageDigest:   "python@sha256:dummy",
	}, nil
}
func (r *mockRunnerHTTP) RunDBDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	return "[]", nil
}
func (r *mockRunnerHTTP) RunDBEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{Verdict: domain.VerdictAC}, nil
}

func TestDryRunAndPublishHTTPContracts(t *testing.T) {
	repo := &mockExerciseRepoHTTP{
		exercises: map[string]*domain.Exercise{
			"ex-stale": {
				ID:                "ex-stale",
				Title:             "Stale Exercise",
				Type:              domain.ExerciseTypeAlgorithm,
				Status:            "draft",
				Language:          "python",
				ReferenceSolution: "print(1)",
				Stale:             true,
				Config: domain.ExerciseConfig{
					Algorithm: &domain.AlgorithmConfig{
						TestCases: []domain.TestCase{{Input: "1", ExpectedOutput: "1", IsHidden: false}},
					},
				},
				TenantID: "default",
			},
			"ex-no-ref": {
				ID:                "ex-no-ref",
				Title:             "No Reference Exercise",
				Type:              domain.ExerciseTypeAlgorithm,
				Status:            "draft",
				Language:          "python",
				ReferenceSolution: "",
				Stale:             false,
				Config: domain.ExerciseConfig{
					Algorithm: &domain.AlgorithmConfig{
						TestCases: []domain.TestCase{{Input: "1", ExpectedOutput: "1", IsHidden: false}},
					},
				},
				TenantID: "default",
			},
		},
		jobs: make(map[string]*domain.DryRunJob),
	}

	evalService := services.NewEvaluationService(repo, nil, nil, &mockRunnerHTTP{})
	evalHandler := httpdelivery.NewEvaluationHandler(evalService, validator.New())

	mux := http.NewServeMux()
	deps := &httpdelivery.Handlers{
		EvaluationHandler: evalHandler,
		TenantMiddleware:  func(next http.Handler) http.Handler { return next },
		AuditMiddleware:   func(next http.Handler) http.Handler { return next },
	}
	httpdelivery.SetupRoutes(mux, deps)

	// 1. Publicar ejercicio stale -> 409 Conflict (EXERCISE_STALE)
	req := httptest.NewRequest("POST", "/api/v1/exercises/ex-stale/publish", nil)
	req.Header.Set("X-User-Role", "teacher")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409 for stale publish, got: %d", rec.Code)
	}

	// 2. Publicar ejercicio sin referencia -> 409 Conflict (REFERENCE_REQUIRED)
	req = httptest.NewRequest("POST", "/api/v1/exercises/ex-no-ref/publish", nil)
	req.Header.Set("X-User-Role", "teacher")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409 for missing reference, got: %d", rec.Code)
	}

	// 3. Iniciar Dry-Run -> 202 Accepted
	req = httptest.NewRequest("POST", "/api/v1/exercises/ex-stale/dry-run", nil)
	req.Header.Set("X-User-Role", "teacher")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 for dry-run start, got: %d, body: %s", rec.Code, rec.Body.String())
	}
}
