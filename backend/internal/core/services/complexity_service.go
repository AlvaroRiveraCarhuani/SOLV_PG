package services

import (
	"encoding/json"
	"math"
	"solv-backend/internal/core/domain"
	"strings"
)

// HasSizeParameters verifica si el contrato de formato de entrada incluye parámetros de tamaño.
func HasSizeParameters(inputFormat json.RawMessage) bool {
	if len(inputFormat) == 0 || string(inputFormat) == "null" || string(inputFormat) == "{}" {
		return false
	}
	var contract InputFormatContract
	if err := json.Unmarshal(inputFormat, &contract); err != nil {
		return false
	}
	if len(contract.Input.Lines) == 0 {
		return false
	}
	for _, line := range contract.Input.Lines {
		idLower := strings.ToLower(line.ID)
		if idLower == "n" || idLower == "size" || idLower == "len" || idLower == "count" || idLower == "rows" || idLower == "cols" {
			return true
		}
		if line.Type == "ints" || line.Type == "matrix" || line.Type == "strings" {
			return true
		}
	}
	return true
}

// AnalyzeComplexity analiza un conjunto de mediciones empíricas de tiempo/memoria y calcula la complejidad asintótica O(f(N)).
func AnalyzeComplexity(measurements []domain.ComplexityMeasurement) *domain.ComplexityAnalysis {
	if len(measurements) < 3 {
		return &domain.ComplexityAnalysis{
			TimeComplexity:  "unknown",
			SpaceComplexity: "unknown",
			Measurements:    measurements,
			Confidence:      0,
		}
	}

	n := make([]float64, len(measurements))
	t := make([]float64, len(measurements))
	m := make([]float64, len(measurements))

	for i, meas := range measurements {
		n[i] = float64(meas.InputSize)
		t[i] = float64(meas.TimeMs)
		m[i] = float64(meas.MemoryKb)
	}

	curves := []struct {
		name string
		fn   func(x float64) float64
	}{
		{"O(1)", func(x float64) float64 { return 1.0 }},
		{"O(N)", func(x float64) float64 { return x }},
		{"O(N log N)", func(x float64) float64 { return x * math.Log2(math.Max(1.0, x)) }},
		{"O(N²)", func(x float64) float64 { return x * x }},
	}

	var sumT float64
	for _, val := range t {
		sumT += val
	}
	meanT := sumT / float64(len(t))

	var ssTot float64
	for _, val := range t {
		ssTot += (val - meanT) * (val - meanT)
	}

	bestComplexity := "unknown"
	bestR2 := -1.0

	if ssTot < 1e-6 {
		bestComplexity = "O(1)"
		bestR2 = 1.0
	} else {
		for _, c := range curves {
			var r2 float64
			if c.name == "O(1)" {
				var ssRes float64
				for _, val := range t {
					ssRes += (val - meanT) * (val - meanT)
				}
				stdDev := math.Sqrt(ssTot / float64(len(t)))
				if stdDev < 2.0 {
					r2 = 0.95
				} else {
					r2 = 0.1
				}
			} else {
				var num, den float64
				for i := 0; i < len(n); i++ {
					fx := c.fn(n[i])
					num += fx * t[i]
					den += fx * fx
				}
				a := 0.0
				if den > 1e-12 {
					a = num / den
				}

				var ssRes float64
				for i := 0; i < len(n); i++ {
					yHat := a * c.fn(n[i])
					diff := t[i] - yHat
					ssRes += diff * diff
				}

				r2 = 1.0 - (ssRes / ssTot)
			}

			if r2 > 1.0 {
				r2 = 1.0
			}
			if r2 < 0.0 {
				r2 = 0.0
			}

			if r2 > bestR2 {
				bestR2 = r2
				bestComplexity = c.name
			}
		}
	}

	if bestR2 < 0.3 {
		bestComplexity = "unknown"
	}

	spaceComplexity := "O(1)"
	var minM, maxM float64 = m[0], m[0]
	for _, val := range m {
		if val < minM {
			minM = val
		}
		if val > maxM {
			maxM = val
		}
	}
	if maxM-minM > 5120.0 {
		spaceComplexity = "O(N)"
	}

	return &domain.ComplexityAnalysis{
		TimeComplexity:  bestComplexity,
		SpaceComplexity: spaceComplexity,
		Measurements:    measurements,
		Confidence:      math.Round(bestR2*100) / 100,
	}
}
