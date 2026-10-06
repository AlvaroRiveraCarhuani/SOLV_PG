package services

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// Fakes for the no-fail-fast cycle: scripted runner verdicts plus call count.

type scriptRunner struct {
	verdicts []domain.Verdict
	calls    int
}

func (f *scriptRunner) RunTestCase(_ context.Context, _ domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	v := domain.VerdictAC
	if f.calls < len(f.verdicts) {
		v = f.verdicts[f.calls]
	}
	f.calls++
	return domain.TestCaseRunResult{Verdict: v, ExecutionTime: 5 * time.Millisecond}, nil
}

func (f *scriptRunner) RunDBDryRun(_ context.Context, _ domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func (f *scriptRunner) RunDBEvaluation(_ context.Context, _ domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

type stubExerciseRepo struct {
	exercise *domain.Exercise
}

func (s *stubExerciseRepo) GetByID(_ context.Context, _ string) (*domain.Exercise, error) {
	return s.exercise, nil
}

func (s *stubExerciseRepo) GetByIDAndTenant(_ context.Context, _, _ string) (*domain.Exercise, error) {
	return s.exercise, nil
}

func (s *stubExerciseRepo) Create(_ context.Context, _ *domain.Exercise) error { return nil }

func (s *stubExerciseRepo) Update(_ context.Context, _ *domain.Exercise) error { return nil }

func (s *stubExerciseRepo) UpdateStatus(_ context.Context, _, _, _ string) error { return nil }

func (s *stubExerciseRepo) UpdateConfig(_ context.Context, _, _ string, _ domain.ExerciseConfig) error {
	return nil
}

func (s *stubExerciseRepo) UpdateExpectedJSON(_ context.Context, _, _ string) error { return nil }

func (s *stubExerciseRepo) MarkExerciseStale(_ context.Context, _, _ string, _ bool) error { return nil }

func (s *stubExerciseRepo) UpdateExerciseLastValidDryRun(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

func (s *stubExerciseRepo) CreateDryRunJob(_ context.Context, _ *domain.DryRunJob) error { return nil }

func (s *stubExerciseRepo) GetDryRunJob(_ context.Context, _ string) (*domain.DryRunJob, error) {
	return nil, nil
}

func (s *stubExerciseRepo) UpdateDryRunJobProgress(_ context.Context, _ string, _ domain.DryRunJobStatus, _, _ int, _ *domain.EvaluationResult, _ string) error {
	return nil
}

func (s *stubExerciseRepo) ListDueByStudent(_ context.Context, _, _ string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (s *stubExerciseRepo) ListBySubject(_ context.Context, _, _ string) ([]*domain.Exercise, error) {
	return nil, nil
}

func (s *stubExerciseRepo) GetStudentRecommendations(_ context.Context, _, _, _ string) (*domain.StudentRecommendations, error) {
	return nil, nil
}

type allowAnalyzer struct{}

func (allowAnalyzer) ValidateCode(_ string, _ string, _ domain.ASTRules) (bool, string) {
	return true, ""
}

func algorithmExercise(cases int) *domain.Exercise {
	tcs := make([]domain.TestCase, 0, cases)
	for i := 0; i < cases; i++ {
		tcs = append(tcs, domain.TestCase{Input: "in", ExpectedOutput: "out"})
	}
	return &domain.Exercise{
		ID:   "ex-1",
		Type: domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases:     tcs,
				TimeLimitMS:   2000,
				MemoryLimitMB: 128,
			},
		},
	}
}

func TestEvaluateAlgorithm_NoFailFast(t *testing.T) {
	runner := &scriptRunner{verdicts: []domain.Verdict{domain.VerdictWA, domain.VerdictAC, domain.VerdictTLE}}
	svc := NewEvaluationService(algorithmExerciseRepo(3), allowAnalyzer{}, nil, runner)

	res, err := svc.Evaluate(context.Background(), "ex-1", "python", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if runner.calls != 3 {
		t.Fatalf("expected 3 runner calls (no fail-fast), got %d", runner.calls)
	}
	if len(res.CaseResults) != 3 {
		t.Fatalf("expected 3 case results, got %d", len(res.CaseResults))
	}
	want := []domain.Verdict{domain.VerdictWA, domain.VerdictAC, domain.VerdictTLE}
	for i, w := range want {
		if res.CaseResults[i].Verdict != w {
			t.Errorf("case %d: verdict = %s, want %s", i, res.CaseResults[i].Verdict, w)
		}
		if res.CaseResults[i].Index != i {
			t.Errorf("case %d: index = %d, want %d", i, res.CaseResults[i].Index, i)
		}
	}
	if res.Verdict != domain.VerdictWA {
		t.Errorf("global verdict = %s, want first non-AC (WA)", res.Verdict)
	}
}

func TestEvaluateAlgorithm_AllAC(t *testing.T) {
	runner := &scriptRunner{verdicts: []domain.Verdict{domain.VerdictAC, domain.VerdictAC}}
	svc := NewEvaluationService(algorithmExerciseRepo(2), allowAnalyzer{}, nil, runner)

	res, err := svc.Evaluate(context.Background(), "ex-1", "python", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 runner calls, got %d", runner.calls)
	}
	if res.Verdict != domain.VerdictAC {
		t.Errorf("global verdict = %s, want AC", res.Verdict)
	}
	if len(res.CaseResults) != 2 {
		t.Fatalf("expected 2 case results, got %d", len(res.CaseResults))
	}
}

func algorithmExerciseRepo(cases int) *stubExerciseRepo {
	return &stubExerciseRepo{exercise: algorithmExercise(cases)}
}

func TestEvaluateAlgorithm_TemplateLock(t *testing.T) {
	// D-EJ-07: Algorithm evaluation executes runner strictly based on language,
	// ignoring any workspace template.
	runner := &scriptRunner{verdicts: []domain.Verdict{domain.VerdictAC}}
	repo := algorithmExerciseRepo(1)
	svc := NewEvaluationService(repo, allowAnalyzer{}, nil, runner)

	res, err := svc.Evaluate(context.Background(), "ex-1", "python", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("unexpected error during evaluation: %v", err)
	}
	if res.Verdict != domain.VerdictAC {
		t.Fatalf("expected verdict AC, got %s", res.Verdict)
	}
}

