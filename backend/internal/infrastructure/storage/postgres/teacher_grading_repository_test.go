package postgres

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestNormalizeReviewEnvironment(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "IDE", raw: "IDE_PERSISTENTE", want: domain.PracticeEnvironmentTypeIDEPersistente},
		{name: "juez", raw: "JUEZ_EFIMERO", want: domain.PracticeEnvironmentTypeJuezEfimero},
		{name: "vacío cae en juez", raw: "", want: domain.PracticeEnvironmentTypeJuezEfimero},
		{name: "desconocido cae en juez", raw: "OTRA_MODALIDAD", want: domain.PracticeEnvironmentTypeJuezEfimero},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeReviewEnvironment(tc.raw); got != tc.want {
				t.Fatalf("normalizeReviewEnvironment(%q) = %q, se esperaba %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParseComplexityAnalysis(t *testing.T) {
	emptyCases := []struct {
		name string
		raw  []byte
	}{
		{name: "nil (SQL NULL)", raw: nil},
		{name: "vacío", raw: []byte{}},
		{name: "literal null", raw: []byte("null")},
		{name: "objeto vacío", raw: []byte("{}")},
		{name: "objeto vacío con espacios", raw: []byte("  { }  ")},
		{name: "sin complejidad ni mediciones", raw: []byte(`{"measurements":[]}`)},
		{name: "json inválido", raw: []byte("{invalid")},
	}

	for _, tc := range emptyCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseComplexityAnalysis(tc.raw); got != nil {
				t.Fatalf("parseComplexityAnalysis(%q) = %+v, se esperaba nil", string(tc.raw), got)
			}
		})
	}

	t.Run("análisis válido", func(t *testing.T) {
		raw := []byte(`{"time_complexity":"O(N)","space_complexity":"O(1)","confidence":0.92,"measurements":[{"input_size":1000,"time_ms":0.42,"memory_kb":8192}]}`)
		got := parseComplexityAnalysis(raw)
		if got == nil {
			t.Fatal("se esperaba análisis no nulo")
		}
		if got.TimeComplexity != "O(N)" || got.SpaceComplexity != "O(1)" || len(got.Measurements) != 1 {
			t.Fatalf("análisis inesperado: %+v", got)
		}
	})
}
