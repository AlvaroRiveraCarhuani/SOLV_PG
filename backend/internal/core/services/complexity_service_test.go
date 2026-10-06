package services_test

import (
	"encoding/json"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"testing"
)

func TestHasSizeParameters(t *testing.T) {
	validFormat := json.RawMessage(`{
		"input": {
			"lines": [
				{"id": "N", "type": "int", "min": "1", "max": "1000"},
				{"id": "arr", "type": "ints", "count": "N"}
			]
		}
	}`)
	if !services.HasSizeParameters(validFormat) {
		t.Errorf("expected HasSizeParameters to return true")
	}

	invalidFormat := json.RawMessage(`{}`)
	if services.HasSizeParameters(invalidFormat) {
		t.Errorf("expected HasSizeParameters to return false for empty format")
	}
}

func TestAnalyzeComplexity(t *testing.T) {
	t.Run("Insufficient measurements fallback to unknown", func(t *testing.T) {
		meas := []domain.ComplexityMeasurement{
			{InputSize: 10, TimeMs: 0.1, MemoryKb: 64},
		}
		res := services.AnalyzeComplexity(meas)
		if res.TimeComplexity != "unknown" || res.Confidence != 0 {
			t.Errorf("expected unknown and 0 confidence, got %s, %f", res.TimeComplexity, res.Confidence)
		}
	})

	t.Run("Detects Linear O(N)", func(t *testing.T) {
		meas := []domain.ComplexityMeasurement{
			{InputSize: 10, TimeMs: 1.0, MemoryKb: 64},
			{InputSize: 100, TimeMs: 10.0, MemoryKb: 64},
			{InputSize: 1000, TimeMs: 100.0, MemoryKb: 64},
			{InputSize: 10000, TimeMs: 1000.0, MemoryKb: 64},
		}
		res := services.AnalyzeComplexity(meas)
		if res.TimeComplexity != "O(N)" {
			t.Errorf("expected O(N), got %s", res.TimeComplexity)
		}
		if res.Confidence < 0.7 {
			t.Errorf("expected confidence >= 0.7, got %f", res.Confidence)
		}
	})

	t.Run("Detects Quadratic O(N²)", func(t *testing.T) {
		meas := []domain.ComplexityMeasurement{
			{InputSize: 10, TimeMs: 1.0, MemoryKb: 64},
			{InputSize: 100, TimeMs: 100.0, MemoryKb: 64},
			{InputSize: 1000, TimeMs: 10000.0, MemoryKb: 64},
		}
		res := services.AnalyzeComplexity(meas)
		if res.TimeComplexity != "O(N²)" {
			t.Errorf("expected O(N²), got %s", res.TimeComplexity)
		}
	})

	t.Run("Detects Constant O(1)", func(t *testing.T) {
		meas := []domain.ComplexityMeasurement{
			{InputSize: 10, TimeMs: 0.5, MemoryKb: 64},
			{InputSize: 100, TimeMs: 0.51, MemoryKb: 64},
			{InputSize: 1000, TimeMs: 0.49, MemoryKb: 64},
			{InputSize: 10000, TimeMs: 0.50, MemoryKb: 64},
		}
		res := services.AnalyzeComplexity(meas)
		if res.TimeComplexity != "O(1)" {
			t.Errorf("expected O(1), got %s", res.TimeComplexity)
		}
	})
}
