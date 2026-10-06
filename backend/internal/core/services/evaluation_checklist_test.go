package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type checklistMockRunner struct {
	runFunc func(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error)
}

func (m *checklistMockRunner) RunTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, config)
	}
	return domain.TestCaseRunResult{Verdict: domain.VerdictAC, ExecutionTime: 10 * time.Millisecond}, nil
}

func (m *checklistMockRunner) RunDBEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

func (m *checklistMockRunner) RunDBDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func TestGenerateChecklist_BlockersAndWarnings(t *testing.T) {
	ctx := context.Background()

	t.Run("exercise with 0 test cases has blocker", func(t *testing.T) {
		repo := &mockExerciseRepo{
			exercises: map[string]*domain.Exercise{
				"ex-1": {
					ID:       "ex-1",
					Title:    "Test Exercise",
					Language: "python",
					Config: domain.ExerciseConfig{
						Algorithm: &domain.AlgorithmConfig{
							TestCases: []domain.TestCase{},
						},
					},
				},
			},
		}
		runner := &checklistMockRunner{}
		svc := services.NewEvaluationService(repo, nil, nil, runner)

		report, err := svc.GenerateChecklist(ctx, "ex-1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.CanPublish {
			t.Errorf("expected CanPublish=false, got true")
		}
		if len(report.Blockers) == 0 {
			t.Errorf("expected at least 1 blocker, got 0")
		} else if !strings.Contains(report.Blockers[0], "Debes incluir al menos un caso de prueba") {
			t.Errorf("expected blocker message to contain 'Debes incluir al menos un caso de prueba', got %q", report.Blockers[0])
		}
	})

	t.Run("exercise with 0 hidden cases triggers warning", func(t *testing.T) {
		repo := &mockExerciseRepo{
			exercises: map[string]*domain.Exercise{
				"ex-2": {
					ID:                "ex-2",
					Title:             "Test Exercise 2",
					Language:          "python",
					ReferenceSolution: "print(int(input())*2)",
					TimeLimitMS:       1000,
					Config: domain.ExerciseConfig{
						Algorithm: &domain.AlgorithmConfig{
							TestCases: []domain.TestCase{
								{Input: "2", ExpectedOutput: "4", Visibility: domain.TestCaseVisibilityExample, Weight: 1.0},
								{Input: "3", ExpectedOutput: "6", Visibility: domain.TestCaseVisibilityPublic, Weight: 1.0},
							},
						},
					},
				},
			},
		}
		runner := &checklistMockRunner{
			runFunc: func(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
				return domain.TestCaseRunResult{
					Verdict:       domain.VerdictAC,
					ExecutionTime: 50 * time.Millisecond,
				}, nil
			},
		}
		svc := services.NewEvaluationService(repo, nil, nil, runner)

		report, err := svc.GenerateChecklist(ctx, "ex-2", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.CanPublish {
			t.Errorf("expected CanPublish=true with only warnings, got false (blockers: %v)", report.Blockers)
		}
		if len(report.Warnings) == 0 {
			t.Errorf("expected at least 1 warning, got 0")
		} else if !strings.Contains(report.Warnings[0], "0 casos ocultos") {
			t.Errorf("expected warning message to contain '0 casos ocultos', got %q", report.Warnings[0])
		}
	})

	t.Run("exercise with failing reference solution creates blocker", func(t *testing.T) {
		repo := &mockExerciseRepo{
			exercises: map[string]*domain.Exercise{
				"ex-3": {
					ID:                "ex-3",
					Title:             "Test Exercise 3",
					Language:          "python",
					ReferenceSolution: "print(0)",
					Config: domain.ExerciseConfig{
						Algorithm: &domain.AlgorithmConfig{
							TestCases: []domain.TestCase{
								{Input: "5", ExpectedOutput: "10", Visibility: domain.TestCaseVisibilityHidden, Weight: 1.0},
							},
						},
					},
				},
			},
		}
		runner := &checklistMockRunner{
			runFunc: func(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
				return domain.TestCaseRunResult{
					Verdict:       domain.VerdictWA,
					ActualOutput:  "0",
					ExecutionTime: 10 * time.Millisecond,
				}, nil
			},
		}
		svc := services.NewEvaluationService(repo, nil, nil, runner)

		report, err := svc.GenerateChecklist(ctx, "ex-3", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.CanPublish {
			t.Errorf("expected CanPublish=false, got true")
		}
		if len(report.Blockers) == 0 {
			t.Errorf("expected at least 1 blocker, got 0")
		} else if !strings.Contains(report.Blockers[0], "La solución de referencia no obtuvo AC") {
			t.Errorf("expected blocker message to contain 'La solución de referencia no obtuvo AC', got %q", report.Blockers[0])
		}
	})
}
