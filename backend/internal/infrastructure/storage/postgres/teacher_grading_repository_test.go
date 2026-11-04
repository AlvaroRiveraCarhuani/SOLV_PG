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
