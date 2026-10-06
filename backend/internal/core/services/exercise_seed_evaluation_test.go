package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type echoRunner struct {
	lastTestCase domain.TestCase
	runCount     int
}

func (e *echoRunner) RunTestCase(_ context.Context, cfg domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	e.lastTestCase = cfg.TestCase
	e.runCount++
	// If sourceCode is "error", return WA
	if cfg.SourceCode == "error" {
		return domain.TestCaseRunResult{
			Verdict:       domain.VerdictWA,
			ActualOutput:  "wrong",
			ExecutionTime: 10 * time.Millisecond,
		}, nil
	}
	// Default: return AC with expected output or echo of input
	return domain.TestCaseRunResult{
		Verdict:       domain.VerdictAC,
		ActualOutput:  "out:" + cfg.TestCase.Input,
		ExecutionTime: 10 * time.Millisecond,
	}, nil
}

func (e *echoRunner) RunDBDryRun(_ context.Context, _ domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func (e *echoRunner) RunDBEvaluation(_ context.Context, _ domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

func TestGenerateStudentSeed_Determinism(t *testing.T) {
	exID := "ex-100"
	studentA := "student-aaa"
	studentB := "student-bbb"

	seedA1 := GenerateStudentSeed(exID, studentA)
	seedA2 := GenerateStudentSeed(exID, studentA)
	seedB := GenerateStudentSeed(exID, studentB)

	if seedA1 != seedA2 {
		t.Errorf("expected seed to be deterministic for student A, got %d != %d", seedA1, seedA2)
	}
	if seedA1 == seedB {
		t.Errorf("expected different seeds for student A and student B, got identical %d", seedA1)
	}
}

func TestEvaluateAlgorithm_PerStudentSeedInExam(t *testing.T) {
	contract := json.RawMessage(`{
		"version": 1,
		"input": {
			"lines": [
				{ "id": "n", "type": "int", "min": 10, "max": 20 }
			]
		}
	}`)

	ex := &domain.Exercise{
		ID:                "ex-exam-1",
		Type:              domain.ExerciseTypeAlgorithm,
		Purpose:           string(domain.ExercisePurposeExam),
		PerStudentSeed:    true,
		Language:          "python",
		ReferenceSolution: "print(int(input()))",
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
				InputFormat:   contract,
				TestCases: []domain.TestCase{
					{OrderIndex: 1, Input: "15", ExpectedOutput: "15", Visibility: domain.TestCaseVisibilityHidden, Weight: 1.0},
					{OrderIndex: 2, Input: "18", ExpectedOutput: "18", Visibility: domain.TestCaseVisibilityHidden, Weight: 2.0},
				},
			},
		},
	}

	repo := &stubExerciseRepo{exercise: ex}
	runner := &echoRunner{}
	svc := NewEvaluationService(repo, allowAnalyzer{}, nil, runner)

	// Evaluate student 1
	ctxStu1 := context.WithValue(context.Background(), domain.UserIDKey, "student-1")
	res1, err := svc.Evaluate(ctxStu1, "ex-exam-1", "python", "cHJpbnQoMSk=") // base64 for "print(1)"
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.Verdict != domain.VerdictAC {
		t.Fatalf("expected AC, got %s", res1.Verdict)
	}
	if len(res1.GeneratedCases) != 2 {
		t.Fatalf("expected 2 generated cases snapshot, got %d", len(res1.GeneratedCases))
	}

	// Evaluate student 1 again -> must match exactly
	res1Repeat, err := svc.Evaluate(ctxStu1, "ex-exam-1", "python", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("unexpected error on repeat: %v", err)
	}
	if res1.GeneratedCases[0].Input != res1Repeat.GeneratedCases[0].Input {
		t.Errorf("repeat evaluation for same student gave different input: %s vs %s",
			res1.GeneratedCases[0].Input, res1Repeat.GeneratedCases[0].Input)
	}

	// Evaluate student 2 -> should have distinct inputs
	ctxStu2 := context.WithValue(context.Background(), domain.UserIDKey, "student-2")
	res2, err := svc.Evaluate(ctxStu2, "ex-exam-1", "python", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("unexpected error for student 2: %v", err)
	}
	if len(res2.GeneratedCases) != 2 {
		t.Fatalf("expected 2 generated cases snapshot for student 2, got %d", len(res2.GeneratedCases))
	}
	if res1.GeneratedCases[0].Input == res2.GeneratedCases[0].Input && res1.GeneratedCases[1].Input == res2.GeneratedCases[1].Input {
		t.Errorf("expected different generated cases for student 1 and 2, got identical inputs: %v", res1.GeneratedCases)
	}
}

func TestEvaluateAlgorithm_ExamCensorshipOnFailure(t *testing.T) {
	contract := json.RawMessage(`{
		"version": 1,
		"input": {
			"lines": [
				{ "id": "n", "type": "int", "min": 1, "max": 100 }
			]
		}
	}`)

	ex := &domain.Exercise{
		ID:                "ex-exam-censorship",
		Type:              domain.ExerciseTypeAlgorithm,
		Purpose:           string(domain.ExercisePurposeExam),
		PerStudentSeed:    true,
		Language:          "python",
		ReferenceSolution: "print(1)",
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
				InputFormat:   contract,
				TestCases: []domain.TestCase{
					{OrderIndex: 1, Input: "10", ExpectedOutput: "10", Visibility: domain.TestCaseVisibilityPublic},
				},
			},
		},
	}

	repo := &stubExerciseRepo{exercise: ex}
	runner := &echoRunner{}
	svc := NewEvaluationService(repo, allowAnalyzer{}, nil, runner)

	// "error" encoded in base64 is "ZXJyb3I="
	ctxStu := context.WithValue(context.Background(), domain.UserIDKey, "student-fail")
	res, err := svc.Evaluate(ctxStu, "ex-exam-censorship", "python", "ZXJyb3I=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Verdict != domain.VerdictWA {
		t.Fatalf("expected WA, got %s", res.Verdict)
	}
	if res.FailedTestCase == nil {
		t.Fatal("expected FailedTestCase to be present")
	}
	if res.FailedTestCase.Input != "[OCULTO POR EXAMEN]" {
		t.Errorf("expected masked input '[OCULTO POR EXAMEN]', got %q", res.FailedTestCase.Input)
	}
	if res.FailedTestCase.ExpectedOutput != "[OCULTO POR EXAMEN]" {
		t.Errorf("expected masked expected_output '[OCULTO POR EXAMEN]', got %q", res.FailedTestCase.ExpectedOutput)
	}
	// But generated_cases snapshot retains real generated values for teacher review
	if len(res.GeneratedCases) != 1 || res.GeneratedCases[0].Input == "[OCULTO POR EXAMEN]" {
		t.Errorf("expected generated_cases snapshot to retain actual generated input, got %v", res.GeneratedCases)
	}
}
