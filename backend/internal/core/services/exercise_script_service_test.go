package services_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockScriptSandboxRunner struct {
	runFunc func(ctx context.Context, scriptCode string, timeoutSec int) (string, string, error)
}

func (m *mockScriptSandboxRunner) RunPythonScript(ctx context.Context, scriptCode string, timeoutSec int) (string, string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, scriptCode, timeoutSec)
	}
	return `[{"input": "5\n", "expected_output": "15\n"}]`, "", nil
}

func TestGenerateCasesFromScript(t *testing.T) {
	ctx := context.Background()
	validContract := json.RawMessage(`{
		"version": 1,
		"input": {
			"lines": [
				{ "id": "n", "type": "int", "min": 1, "max": 100 }
			]
		}
	}`)

	t.Run("dry-run generates valid preview", func(t *testing.T) {
		repo := newMockImportRepo()
		exID := "ex-script-1"
		repo.exercises[exID] = &domain.Exercise{
			ID:        exID,
			SubjectID: strPtr("course-1"),
			Config: domain.ExerciseConfig{
				Algorithm: &domain.AlgorithmConfig{
					InputFormat: validContract,
				},
			},
		}

		svc := services.NewEvaluationService(repo, nil, nil, nil)
		runner := &mockScriptSandboxRunner{
			runFunc: func(ctx context.Context, scriptCode string, timeoutSec int) (string, string, error) {
				return `[
					{"input": "5\n", "expected_output": "15\n"},
					{"input": "10\n", "expected_output": "55\n"}
				]`, "", nil
			},
		}
		svc.SetScriptSandboxRunner(runner)

		res, err := svc.GenerateCasesFromScript(ctx, exID, "tenant-1", "import json; print('ok')", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Mode != "dry_run" {
			t.Errorf("expected mode 'dry_run', got %s", res.Mode)
		}
		if !res.CanImport {
			t.Errorf("expected CanImport=true, got false")
		}
		if res.TotalValid != 2 {
			t.Errorf("expected 2 valid cases, got %d", res.TotalValid)
		}
		if len(res.Cases) != 2 {
			t.Errorf("expected 2 cases preview, got %d", len(res.Cases))
		}
	})

	t.Run("missing input_format returns ErrInputFormatRequired", func(t *testing.T) {
		repo := newMockImportRepo()
		exID := "ex-no-format"
		repo.exercises[exID] = &domain.Exercise{
			ID: exID,
		}

		svc := services.NewEvaluationService(repo, nil, nil, nil)
		svc.SetScriptSandboxRunner(&mockScriptSandboxRunner{})

		_, err := svc.GenerateCasesFromScript(ctx, exID, "tenant-1", "print('[]')", true)
		if err == nil || !strings.Contains(err.Error(), "input_format") {
			t.Errorf("expected ErrInputFormatRequired, got: %v", err)
		}
	})

	t.Run("more than 100 cases returns ErrTooManyScriptCases", func(t *testing.T) {
		repo := newMockImportRepo()
		exID := "ex-too-many"
		repo.exercises[exID] = &domain.Exercise{
			ID: exID,
			Config: domain.ExerciseConfig{
				Algorithm: &domain.AlgorithmConfig{
					InputFormat: validContract,
				},
			},
		}

		var items []string
		for i := 0; i < 101; i++ {
			items = append(items, `{"input": "1\n", "expected_output": "1\n"}`)
		}
		jsonOutput := "[" + strings.Join(items, ",") + "]"

		svc := services.NewEvaluationService(repo, nil, nil, nil)
		svc.SetScriptSandboxRunner(&mockScriptSandboxRunner{
			runFunc: func(ctx context.Context, scriptCode string, timeoutSec int) (string, string, error) {
				return jsonOutput, "", nil
			},
		})

		_, err := svc.GenerateCasesFromScript(ctx, exID, "tenant-1", "script", true)
		if err == nil || !strings.Contains(err.Error(), "100 casos") {
			t.Errorf("expected ErrTooManyScriptCases, got: %v", err)
		}
	})

	t.Run("actual mode appends hidden test cases", func(t *testing.T) {
		repo := newMockImportRepo()
		exID := "ex-actual"
		repo.exercises[exID] = &domain.Exercise{
			ID: exID,
			Config: domain.ExerciseConfig{
				Algorithm: &domain.AlgorithmConfig{
					InputFormat: validContract,
					TestCases:   domain.TestCases{},
				},
			},
		}

		svc := services.NewEvaluationService(repo, nil, nil, nil)
		svc.SetScriptSandboxRunner(&mockScriptSandboxRunner{
			runFunc: func(ctx context.Context, scriptCode string, timeoutSec int) (string, string, error) {
				return `[{"input": "5\n", "expected_output": "15\n"}]`, "", nil
			},
		})

		res, err := svc.GenerateCasesFromScript(ctx, exID, "tenant-1", "script", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Mode != "import" {
			t.Errorf("expected mode 'import', got %s", res.Mode)
		}
		if res.ImportedCount != 1 {
			t.Errorf("expected 1 imported case, got %d", res.ImportedCount)
		}

		ex := repo.exercises[exID]
		if len(ex.Config.Algorithm.TestCases) != 1 {
			t.Fatalf("expected 1 test case stored in exercise, got %d", len(ex.Config.Algorithm.TestCases))
		}
		if ex.Config.Algorithm.TestCases[0].Visibility != domain.TestCaseVisibilityHidden {
			t.Errorf("expected test case to be hidden, got %s", ex.Config.Algorithm.TestCases[0].Visibility)
		}
	})
}

func strPtr(s string) *string {
	return &s
}
