package domain

import (
	"errors"
	"testing"
)

func TestExerciseValidateTimeLimitRange(t *testing.T) {
	cases := []struct {
		name string
		ms   int
		want error
	}{
		{name: "0 se permite (el repo normaliza a 1000)", ms: 0, want: nil},
		{name: "mínimo del rango", ms: 100, want: nil},
		{name: "máximo del rango", ms: 30000, want: nil},
		{name: "valor típico docente", ms: 1500, want: nil},
		{name: "negativo rechazado", ms: -1, want: ErrTimeLimitOutOfRange},
		{name: "bajo el mínimo rechazado", ms: 99, want: ErrTimeLimitOutOfRange},
		{name: "sobre el máximo rechazado", ms: 30001, want: ErrTimeLimitOutOfRange},
		{name: "valor absurdo rechazado", ms: 999999, want: ErrTimeLimitOutOfRange},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ex := Exercise{
				TimeLimitMS: tc.ms,
				Purpose:     string(ExercisePurposeClass),
				Tags:        []string{},
			}
			err := ex.Validate()
			if !errors.Is(err, tc.want) {
				t.Fatalf("Validate() con time_limit_ms=%d: esperado %v, obtenido %v", tc.ms, tc.want, err)
			}
		})
	}
}