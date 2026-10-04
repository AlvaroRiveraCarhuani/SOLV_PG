package services_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockExerciseRepoForDryRun struct {
	mu        sync.Mutex
	exercises map[string]*domain.Exercise
	jobs      map[string]*domain.DryRunJob
}

func newMockExerciseRepoForDryRun() *mockExerciseRepoForDryRun {
	return &mockExerciseRepoForDryRun{
		exercises: make(map[string]*domain.Exercise),
		jobs:      make(map[string]*domain.DryRunJob),
	}
}

func (m *mockExerciseRepoForDryRun) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ex, ok := m.exercises[id]
	if !ok {
		return nil, errors.New("exercise not found")
	}
	return ex, nil
}

func (m *mockExerciseRepoForDryRun) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.GetByID(ctx, id)
}

func (m *mockExerciseRepoForDryRun) Create(ctx context.Context, exercise *domain.Exercise) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockExerciseRepoForDryRun) Update(ctx context.Context, exercise *domain.Exercise) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockExerciseRepoForDryRun) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[id]; ok {
		ex.Status = status
		return nil
	}
	return errors.New("not found")
}

func (m *mockExerciseRepoForDryRun) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[id]; ok {
		ex.Config = config
		ex.Stale = true
		return nil
	}
	return errors.New("not found")
}

func (m *mockExerciseRepoForDryRun) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}

func (m *mockExerciseRepoForDryRun) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[exerciseID]; ok {
		ex.Stale = stale
		return nil
	}
	return errors.New("not found")
}

func (m *mockExerciseRepoForDryRun) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ex, ok := m.exercises[exerciseID]; ok {
		ex.Stale = false
		ex.LastValidDryRunAt = &dryRunAt
		return nil
	}
	return errors.New("not found")
}

func (m *mockExerciseRepoForDryRun) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *mockExerciseRepoForDryRun) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return nil, errors.New("job not found")
	}
	return j, nil
}

func (m *mockExerciseRepoForDryRun) UpdateDryRunJobProgress(
	ctx context.Context,
	jobID string,
	status domain.DryRunJobStatus,
	current, total int,
	result *domain.EvaluationResult,
	errMsg string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return errors.New("job not found")
	}
	j.Status = status
	j.ProgressCurrent = current
	j.ProgressTotal = total
	j.Result = result
	j.Error = errMsg
	j.UpdatedAt = time.Now()
	return nil
}

func (m *mockExerciseRepoForDryRun) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (m *mockExerciseRepoForDryRun) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	return nil, nil
}

type mockRunnerForDryRun struct {
	verdict domain.Verdict
}

func (r *mockRunnerForDryRun) RunTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return domain.TestCaseRunResult{
		Verdict:       r.verdict,
		ExecutionTime: 10 * time.Millisecond,
		ImageDigest:   "python@sha256:dummy",
	}, nil
}

func (r *mockRunnerForDryRun) RunDBDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	return "[]", nil
}

func (r *mockRunnerForDryRun) RunDBEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{Verdict: domain.VerdictAC}, nil
}

func TestPublishExercise_StaleAndReferenceEnforcement(t *testing.T) {
	repo := newMockExerciseRepoForDryRun()
	runner := &mockRunnerForDryRun{verdict: domain.VerdictAC}
	svc := services.NewEvaluationService(repo, nil, nil, runner)
	ctx := context.Background()

	// 1. Ejercicio sin reference_solution -> Error 409 ErrMissingReferenceSolution
	ex1 := &domain.Exercise{
		ID:                "ex-1",
		Title:             "Sum",
		Type:              domain.ExerciseTypeAlgorithm,
		Status:            "draft",
		Language:          "python",
		ReferenceSolution: "",
		Stale:             false,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{{Input: "1 2", ExpectedOutput: "3", IsHidden: false}},
			},
		},
		TenantID: "default",
	}
	_ = repo.Create(ctx, ex1)

	_, err := svc.PublishExercise(ctx, "ex-1", "default")
	if !errors.Is(err, domain.ErrMissingReferenceSolution) {
		t.Fatalf("expected ErrMissingReferenceSolution, got: %v", err)
	}

	// 2. Ejercicio con reference_solution pero Stale -> Error 409 ErrExerciseStale
	ex2 := &domain.Exercise{
		ID:                "ex-2",
		Title:             "Sum",
		Type:              domain.ExerciseTypeAlgorithm,
		Status:            "draft",
		Language:          "python",
		ReferenceSolution: "print(3)",
		Stale:             true,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{{Input: "1 2", ExpectedOutput: "3", IsHidden: false}},
			},
		},
		TenantID: "default",
	}
	_ = repo.Create(ctx, ex2)

	_, err = svc.PublishExercise(ctx, "ex-2", "default")
	if !errors.Is(err, domain.ErrExerciseStale) {
		t.Fatalf("expected ErrExerciseStale, got: %v", err)
	}

	// 3. Ejercicio con reference_solution, stale=false pero sin casos públicos -> ErrZeroPublicTestCases
	ex3 := &domain.Exercise{
		ID:                "ex-3",
		Title:             "Sum",
		Type:              domain.ExerciseTypeAlgorithm,
		Status:            "draft",
		Language:          "python",
		ReferenceSolution: "print(3)",
		Stale:             false,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{{Input: "1 2", ExpectedOutput: "3", IsHidden: true}},
			},
		},
		TenantID: "default",
	}
	_ = repo.Create(ctx, ex3)

	_, err = svc.PublishExercise(ctx, "ex-3", "default")
	if !errors.Is(err, services.ErrZeroPublicTestCases) {
		t.Fatalf("expected ErrZeroPublicTestCases, got: %v", err)
	}

	// 4. Ejercicio válido -> Publicado exitosamente
	ex4 := &domain.Exercise{
		ID:                "ex-4",
		Title:             "Sum",
		Type:              domain.ExerciseTypeAlgorithm,
		Status:            "draft",
		Language:          "python",
		ReferenceSolution: "print(3)",
		Stale:             false,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{{Input: "1 2", ExpectedOutput: "3", IsHidden: false}},
			},
		},
		TenantID: "default",
	}
	_ = repo.Create(ctx, ex4)

	pub, err := svc.PublishExercise(ctx, "ex-4", "default")
	if err != nil {
		t.Fatalf("unexpected error publishing valid exercise: %v", err)
	}
	if pub.Status != "published" {
		t.Errorf("expected status 'published', got: %s", pub.Status)
	}
}

func TestStartDryRun_LifecycleAndStaleReset(t *testing.T) {
	repo := newMockExerciseRepoForDryRun()
	runner := &mockRunnerForDryRun{verdict: domain.VerdictAC}
	svc := services.NewEvaluationService(repo, nil, nil, runner)
	ctx := context.Background()

	ex := &domain.Exercise{
		ID:                "ex-dry-1",
		Title:             "Fibonacci",
		Type:              domain.ExerciseTypeAlgorithm,
		Status:            "draft",
		Language:          "python",
		ReferenceSolution: "print(5)",
		Stale:             true,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "5", ExpectedOutput: "5", IsHidden: false},
					{Input: "6", ExpectedOutput: "8", IsHidden: true},
				},
				MemoryLimitMB: 256,
				TimeLimitMS:   1000,
			},
		},
		TenantID: "default",
	}
	_ = repo.Create(ctx, ex)

	// 1. Iniciar Dry-Run
	job, err := svc.StartDryRun(ctx, "ex-dry-1", "default")
	if err != nil {
		t.Fatalf("unexpected error starting dry run: %v", err)
	}
	if job.Status != domain.DryRunJobStatusQueued {
		t.Errorf("expected job status 'queued', got: %s", job.Status)
	}
	if job.ProgressTotal != 2 {
		t.Errorf("expected progress_total 2, got: %d", job.ProgressTotal)
	}

	// 2. Esperar finalización del job asíncrono
	time.Sleep(50 * time.Millisecond)

	finalJob, err := svc.GetDryRunJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("error retrieving job: %v", err)
	}
	if finalJob.Status != domain.DryRunJobStatusDone {
		t.Errorf("expected job status 'done', got: %s", finalJob.Status)
	}
	if finalJob.ProgressCurrent != 2 {
		t.Errorf("expected progress_current 2, got: %d", finalJob.ProgressCurrent)
	}

	// 3. Verificar que el ejercicio ya no está stale y tiene timestamp de dry-run
	updatedEx, _ := repo.GetByID(ctx, "ex-dry-1")
	if updatedEx.Stale {
		t.Errorf("expected stale to be false after successful dry run")
	}
	if updatedEx.LastValidDryRunAt == nil {
		t.Errorf("expected LastValidDryRunAt to be set")
	}
}
