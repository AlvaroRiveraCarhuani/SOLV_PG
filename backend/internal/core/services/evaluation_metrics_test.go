package services

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
)

// Fake recorder: captures one metric per evaluated case.

type captureRecorder struct {
	metrics []domain.RunMetric
}

func (c *captureRecorder) RecordRunMetric(_ context.Context, m domain.RunMetric) error {
	c.metrics = append(c.metrics, m)
	return nil
}

func TestEvaluateAlgorithm_RecordsMetricsPerCase(t *testing.T) {
	runner := &scriptRunner{verdicts: []domain.Verdict{domain.VerdictAC, domain.VerdictWA}}
	rec := &captureRecorder{}
	svc := NewEvaluationService(algorithmExerciseRepo(2), allowAnalyzer{}, nil, runner)
	svc.SetMetricsRecorder(rec)

	_, err := svc.Evaluate(context.Background(), "ex-1", "c++", "cHJpbnQoMSk=")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(rec.metrics) != 2 {
		t.Fatalf("expected 2 recorded metrics, got %d", len(rec.metrics))
	}
	for i, m := range rec.metrics {
		if m.Language != "cpp" {
			t.Errorf("metric %d: language = %q, want canonical cpp", i, m.Language)
		}
		if m.CaseIndex != i {
			t.Errorf("metric %d: case index = %d, want %d", i, m.CaseIndex, i)
		}
		if m.ExerciseID != "ex-1" {
			t.Errorf("metric %d: exercise = %q, want ex-1", i, m.ExerciseID)
		}
	}
	if rec.metrics[0].Verdict != domain.VerdictAC || rec.metrics[1].Verdict != domain.VerdictWA {
		t.Errorf("metric verdicts = %v, want [AC WA]", rec.metrics)
	}
}

func TestEvaluateAlgorithm_BuildFailureIsCEPerCase(t *testing.T) {
	runner := &scriptRunner{verdicts: []domain.Verdict{domain.VerdictCE, domain.VerdictCE}}
	svc := NewEvaluationService(algorithmExerciseRepo(2), allowAnalyzer{}, nil, runner)

	res, err := svc.Evaluate(context.Background(), "ex-1", "cpp", "aW52YWxpZA==")
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 runner calls on CE, got %d", runner.calls)
	}
	if res.Verdict != domain.VerdictCE {
		t.Errorf("global verdict = %s, want CE", res.Verdict)
	}
	if len(res.CaseResults) != 2 {
		t.Fatalf("expected 2 case results, got %d", len(res.CaseResults))
	}
}
