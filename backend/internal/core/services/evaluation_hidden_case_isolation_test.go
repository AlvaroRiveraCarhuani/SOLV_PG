package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// GOB-SEGURO-01: el rol student nunca debe recibir la salida esperada de los
// casos de evaluación ocultos. El detalle del comparador ("Esperado: ...")
// se enmascara en los resultados por caso de los casos ocultos.

type leakyWaRunner struct{}

func (leakyWaRunner) RunTestCase(_ context.Context, cfg domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	if cfg.TestCase.Visibility == domain.TestCaseVisibilityExample {
		return domain.TestCaseRunResult{Verdict: domain.VerdictAC, ExecutionTime: 3 * time.Millisecond}, nil
	}
	return domain.TestCaseRunResult{
		Verdict:       domain.VerdictWA,
		ExecutionTime: 4 * time.Millisecond,
		ActualOutput:  "424243\n",
		ErrorDetails: fmt.Sprintf(
			"Respuesta incorrecta. Esperado: %q, Obtenido: %q",
			cfg.TestCase.ExpectedOutput,
			"424243\n",
		),
	}, nil
}

func (leakyWaRunner) RunDBDryRun(_ context.Context, _ domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func (leakyWaRunner) RunDBEvaluation(_ context.Context, _ domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

func hiddenCaseExercise() *domain.Exercise {
	return &domain.Exercise{
		ID:   "ex-hidden-1",
		Type: domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "5", ExpectedOutput: "10", Visibility: domain.TestCaseVisibilityExample, Weight: 1},
					{Input: "424242", ExpectedOutput: "848484", Visibility: domain.TestCaseVisibilityHidden, Weight: 1, IsHidden: true},
				},
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
			},
		},
	}
}

func TestEvaluateAlgorithm_HiddenCaseOutputNotExposed(t *testing.T) {
	svc := NewEvaluationService(
		&stubExerciseRepo{exercise: hiddenCaseExercise()},
		allowAnalyzer{},
		nil,
		leakyWaRunner{},
	)

	res, err := svc.Evaluate(context.Background(), "ex-hidden-1", "python", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if res.Verdict != domain.VerdictWA {
		t.Fatalf("verdict = %s, want WA", res.Verdict)
	}
	if len(res.CaseResults) != 2 {
		t.Fatalf("expected 2 case results, got %d", len(res.CaseResults))
	}

	hiddenMessage := res.CaseResults[1].Message
	if strings.Contains(hiddenMessage, "848484") {
		t.Errorf("hidden case message leaks the expected output: %q", hiddenMessage)
	}
	if strings.Contains(hiddenMessage, "424242") {
		t.Errorf("hidden case message leaks the case input: %q", hiddenMessage)
	}
	if !strings.Contains(hiddenMessage, hiddenCaseDetailPlaceholder) {
		t.Errorf("hidden case message = %q, want placeholder %q", hiddenMessage, hiddenCaseDetailPlaceholder)
	}
	if !strings.Contains(hiddenMessage, string(domain.VerdictWA)) {
		t.Errorf("hidden case message = %q, want it to keep the verdict", hiddenMessage)
	}

	if res.FailedTestCase == nil {
		t.Fatal("expected a failed test case in the result")
	}
	if res.FailedTestCase.ExpectedOutput != "[OCULTO POR EXAMEN]" {
		t.Errorf("failed case expected_output = %q, want masked value", res.FailedTestCase.ExpectedOutput)
	}
	if res.FailedTestCase.Input != "[OCULTO POR EXAMEN]" {
		t.Errorf("failed case input = %q, want masked value", res.FailedTestCase.Input)
	}

	visibleMessage := res.CaseResults[0].Message
	if visibleMessage != "" {
		t.Errorf("visible case message = %q, want empty for an accepted case", visibleMessage)
	}
}
