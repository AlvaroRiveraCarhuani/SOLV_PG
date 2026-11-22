package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type testModeRunner struct{}

func (testModeRunner) RunTestCase(_ context.Context, cfg domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	if cfg.TestCase.Visibility == domain.TestCaseVisibilityExample {
		return domain.TestCaseRunResult{
			Verdict:       domain.VerdictAC,
			ExecutionTime: 5 * time.Millisecond,
			ActualOutput:  "10\n",
		}, nil
	}
	// Si llega a ejecutar un caso oculto en modo test, falla con WA
	return domain.TestCaseRunResult{
		Verdict:       domain.VerdictWA,
		ExecutionTime: 5 * time.Millisecond,
		ActualOutput:  "999\n",
	}, nil
}

func (testModeRunner) RunDBDryRun(_ context.Context, _ domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func (testModeRunner) RunDBEvaluation(_ context.Context, _ domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

func TestEvaluateAlgorithm_TestMode_RunsOnlyExampleCases(t *testing.T) {
	ex := &domain.Exercise{
		ID:   "ex-test-mode-1",
		Type: domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "5", ExpectedOutput: "10\n", Visibility: domain.TestCaseVisibilityExample, Weight: 1},
					{Input: "100", ExpectedOutput: "200\n", Visibility: domain.TestCaseVisibilityHidden, Weight: 1, IsHidden: true},
				},
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
			},
		},
	}

	svc := NewEvaluationService(
		&stubExerciseRepo{exercise: ex},
		allowAnalyzer{},
		nil,
		testModeRunner{},
	)

	// Ejecución en modo "test" (Probar código)
	res, err := svc.Evaluate(context.Background(), ex.ID, "python", "cHJpbnQoMTAp", "test")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	// 1. Solo debe haber corrido el caso de ejemplo
	if len(res.CaseResults) != 1 {
		t.Fatalf("expected 1 case result, got %d", len(res.CaseResults))
	}

	// 2. El caso de ejemplo dio AC
	if res.Verdict != domain.VerdictAC {
		t.Fatalf("verdict = %s, want AC", res.Verdict)
	}

	// 3. No debe tener recibo firmado de entrega
	if res.Receipt != "" {
		t.Fatalf("test mode should not generate signed submission receipt, got %s", res.Receipt)
	}

	// 4. CaseResult debe contener ActualOutput
	if res.CaseResults[0].ActualOutput != "10\n" {
		t.Errorf("ActualOutput = %q, want %q", res.CaseResults[0].ActualOutput, "10\n")
	}
}

func TestEvaluateAlgorithm_DeadlineExpired_Rejects(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	ex := &domain.Exercise{
		ID:      "ex-expired",
		Type:    domain.ExerciseTypeAlgorithm,
		DueDate: &past,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "5", ExpectedOutput: "10\n", Visibility: domain.TestCaseVisibilityExample, Weight: 1},
				},
			},
		},
	}

	svc := NewEvaluationService(
		&stubExerciseRepo{exercise: ex},
		allowAnalyzer{},
		nil,
		testModeRunner{},
	)

	_, err := svc.Evaluate(context.Background(), ex.ID, "python", "cHJpbnQoMTAp")
	if err == nil || err != domain.ErrExerciseDeadlineExpired {
		t.Fatalf("expected ErrExerciseDeadlineExpired, got %v", err)
	}
}

func TestEvaluateAlgorithm_TestMode_NoExampleCases_ReturnsError(t *testing.T) {
	ex := &domain.Exercise{
		ID:   "ex-only-hidden",
		Type: domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "100", ExpectedOutput: "200\n", Visibility: domain.TestCaseVisibilityHidden, Weight: 1, IsHidden: true},
				},
			},
		},
	}

	svc := NewEvaluationService(
		&stubExerciseRepo{exercise: ex},
		allowAnalyzer{},
		nil,
		testModeRunner{},
	)

	_, err := svc.Evaluate(context.Background(), ex.ID, "python", "cHJpbnQoMTAp", "test")
	if err == nil || err != domain.ErrNoExampleCasesForTesting {
		t.Fatalf("expected ErrNoExampleCasesForTesting when no example cases exist, got %v", err)
	}
}

func TestEvaluateAlgorithm_CrossTenant_Rejected(t *testing.T) {
	ex := &domain.Exercise{
		ID:       "ex-tenant-a",
		TenantID: "tenant-a",
		Type:     domain.ExerciseTypeAlgorithm,
	}

	svc := NewEvaluationService(
		&stubExerciseRepo{exercise: ex},
		allowAnalyzer{},
		nil,
		testModeRunner{},
	)

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, "tenant-b")
	_, err := svc.Evaluate(ctx, ex.ID, "python", "cHJpbnQoMTAp")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant access, got %v", err)
	}
}

type evalMockSubmissionRepo struct {
	submissions []*domain.Submission
}

func (m *evalMockSubmissionRepo) Create(_ context.Context, _ *domain.Submission) error {
	return nil
}

func (m *evalMockSubmissionRepo) GetByID(_ context.Context, _, _ string) (*domain.Submission, error) {
	return nil, nil
}

func (m *evalMockSubmissionRepo) ListByExerciseAndStudent(_ context.Context, _, exerciseID, studentID string) ([]*domain.Submission, error) {
	var res []*domain.Submission
	for _, s := range m.submissions {
		if s.ExerciseID == exerciseID && s.StudentID == studentID {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *evalMockSubmissionRepo) ListByExercise(_ context.Context, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (m *evalMockSubmissionRepo) ListByStudent(_ context.Context, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (m *evalMockSubmissionRepo) UpdateOverride(_ context.Context, _, _, _, _ string, _ *int, _ *string) error {
	return nil
}

func TestEvaluateAlgorithm_AlreadySubmitted_Rejects(t *testing.T) {
	ex := &domain.Exercise{
		ID:       "ex-exam-submitted",
		TenantID: "tenant-1",
		Purpose:  string(domain.ExercisePurposeExam),
		Type:     domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "1", ExpectedOutput: "1\n", Visibility: domain.TestCaseVisibilityExample},
				},
			},
		},
	}

	subRepo := &evalMockSubmissionRepo{
		submissions: []*domain.Submission{
			{
				ID:         "sub-1",
				ExerciseID: ex.ID,
				StudentID:  "student-1",
				TenantID:   "tenant-1",
			},
		},
	}

	svc := NewEvaluationService(
		&stubExerciseRepo{exercise: ex},
		allowAnalyzer{},
		nil,
		testModeRunner{},
	)
	svc.SetSubmissionRepository(subRepo)

	ctx := context.WithValue(context.Background(), domain.UserIDKey, "student-1")
	ctx = context.WithValue(ctx, domain.TenantIDKey, "tenant-1")

	_, err := svc.Evaluate(ctx, ex.ID, "python", "cHJpbnQoMTAp", "test")
	if !errors.Is(err, domain.ErrResubmissionNotAllowed) {
		t.Fatalf("expected ErrResubmissionNotAllowed after exam already submitted, got %v", err)
	}
}
