package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestToExercisePublicResponse_IncludesSolverFieldsAndStripsHiddenData(t *testing.T) {
	subjectID := "sub-123"
	diff := "medium"
	ex := &domain.Exercise{
		ID:                "ex-456",
		SubjectID:         &subjectID,
		Title:             "Suma Segura",
		Description:       "Suma dos enteros",
		Type:              domain.ExerciseTypeAlgorithm,
		EnvironmentType:   "JUEZ_EFIMERO",
		Difficulty:        &diff,
		Purpose:           "exam",
		PerStudentSeed:    true,
		Language:          "python",
		Boilerplate:       "def solve():\n    pass\n",
		TimeLimitMS:       1200,
		MemoryLimitMB:     256,
		ReferenceSolution: "SECRET_REFERENCE_SOLUTION_DO_NOT_LEAK",
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TimeLimitMS:   1200,
				MemoryLimitMB: 256,
				TestCases: []domain.TestCase{
					{
						Input:          "1 2",
						ExpectedOutput: "3",
						Visibility:     domain.TestCaseVisibilityExample,
					},
					{
						Input:          "999 888",
						ExpectedOutput: "SECRET_HIDDEN_OUTPUT_1887",
						Visibility:     domain.TestCaseVisibilityHidden,
						IsHidden:       true,
					},
				},
			},
		},
	}

	pub := ToExercisePublicResponse(ex)
	if pub == nil {
		t.Fatal("expected non-nil public response")
	}
	if pub.SubjectID == nil || *pub.SubjectID != subjectID {
		t.Errorf("expected SubjectID %q, got %v", subjectID, pub.SubjectID)
	}
	if pub.Language != "python" {
		t.Errorf("expected Language 'python', got %q", pub.Language)
	}
	if pub.Boilerplate != "def solve():\n    pass\n" {
		t.Errorf("expected Boilerplate to be preserved, got %q", pub.Boilerplate)
	}
	if pub.EnvironmentType != "JUEZ_EFIMERO" {
		t.Errorf("expected EnvironmentType 'JUEZ_EFIMERO', got %q", pub.EnvironmentType)
	}
	if !pub.PerStudentSeed {
		t.Errorf("expected PerStudentSeed true, got false")
	}
	if pub.MemoryLimitMB != 256 || pub.MemoryMB != 256 {
		t.Errorf("expected MemoryLimitMB/MemoryMB 256, got %d / %d", pub.MemoryLimitMB, pub.MemoryMB)
	}
	if len(pub.PublicTestCases) != 1 {
		t.Fatalf("expected exactly 1 public example test case, got %d", len(pub.PublicTestCases))
	}

	raw, err := json.Marshal(pub)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(raw)
	if strings.Contains(jsonStr, "SECRET_REFERENCE_SOLUTION_DO_NOT_LEAK") {
		t.Errorf("public response leaked reference_solution: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "SECRET_HIDDEN_OUTPUT_1887") {
		t.Errorf("public response leaked hidden test case: %s", jsonStr)
	}
}
