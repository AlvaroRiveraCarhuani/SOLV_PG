package services

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestFuzzingEngine_GenerateTestCases_Default(t *testing.T) {
	engine := NewFuzzingEngine()
	req := domain.FuzzGenerationRequest{
		ParameterTypes: []string{"integer", "string", "array"},
		Categories:     []string{"numeric_bounds", "empty_structures", "unicode_special"},
	}

	report := engine.GenerateTestCases(req)
	if report.TotalGenerated == 0 {
		t.Fatalf("expected test cases to be generated")
	}

	if len(report.Cases) != report.TotalGenerated {
		t.Fatalf("mismatch in total generated and cases slice")
	}

	// Verificar que existan casos de numeric_bounds
	foundNumeric := false
	for _, c := range report.Cases {
		if c.Category == "numeric_bounds" && c.Input == "0" {
			foundNumeric = true
			break
		}
	}
	if !foundNumeric {
		t.Fatalf("expected to find numeric bound case with input '0'")
	}
}

func TestFuzzingEngine_GenerateTestCases_LimitCount(t *testing.T) {
	engine := NewFuzzingEngine()
	req := domain.FuzzGenerationRequest{
		ParameterTypes: []string{"integer", "string", "array", "matrix", "sql"},
		Categories:     []string{"numeric_bounds", "empty_structures", "unicode_special", "injection_escape", "large_input"},
		Count:          3,
	}

	report := engine.GenerateTestCases(req)
	if report.TotalGenerated != 3 {
		t.Fatalf("expected exactly 3 cases, got %d", report.TotalGenerated)
	}
}

func TestFuzzingEngine_GenerateTestCases_UnicodeAndEscape(t *testing.T) {
	engine := NewFuzzingEngine()
	req := domain.FuzzGenerationRequest{
		ParameterTypes: []string{"string", "sql"},
		Categories:     []string{"unicode_special", "injection_escape"},
	}

	report := engine.GenerateTestCases(req)
	foundUnicode := false
	foundSQL := false
	for _, c := range report.Cases {
		if c.Category == "unicode_special" {
			foundUnicode = true
		}
		if c.Category == "injection_escape" {
			foundSQL = true
		}
	}

	if !foundUnicode || !foundSQL {
		t.Fatalf("expected both unicode and sql escape cases to be generated")
	}
}
