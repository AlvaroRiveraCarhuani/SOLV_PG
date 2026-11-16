package services

import (
	"errors"
	"testing"
)

// Cambiar el propósito mueve a todos los entornos del ejercicio: con entornos
// abiertos el perfil ya aplicado no se puede reescribir, y con entornos
// cerrados el cambio pide confirmación y motivo.
func TestDecidePurposeChange(t *testing.T) {
	cases := []struct {
		nombre       string
		from         string
		to           string
		running      int
		hibernated   int
		confirmation PurposeChangeConfirmation
		wantErr      error
		wantDecision bool
		wantReason   string
	}{
		{
			nombre: "mismo propósito no es un cambio",
			from:   "exam", to: "exam",
		},
		{
			nombre: "propósito vacío no cambia nada",
			from:   "exam", to: "",
		},
		{
			nombre:       "sin entornos el cambio es libre",
			from:         "class",
			to:           "exam",
			wantDecision: true,
		},
		{
			nombre:  "con un entorno abierto no hay cambio",
			from:    "class",
			to:      "exam",
			running: 1,
			wantErr: ErrExercisePurposeChangeBlocked,
		},
		{
			nombre:     "con entornos cerrados pide confirmación",
			from:       "exam",
			to:         "class",
			hibernated: 3,
			wantErr:    ErrExercisePurposeChangeConfirmationNeeded,
		},
		{
			nombre:       "confirmado sin motivo no alcanza",
			from:         "exam",
			to:           "class",
			hibernated:   1,
			confirmation: PurposeChangeConfirmation{Confirmed: true},
			wantErr:      ErrExercisePurposeChangeReasonNeeded,
		},
		{
			nombre:       "confirmado con motivo queda auditado",
			from:         "exam",
			to:           "class",
			hibernated:   2,
			confirmation: PurposeChangeConfirmation{Confirmed: true, Reason: "  se cargó como examen por error "},
			wantDecision: true,
			wantReason:   "se cargó como examen por error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			decision, err := decidePurposeChange(tc.from, tc.to, tc.running, tc.hibernated, tc.confirmation)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("se esperaba %v, se obtuvo %v", tc.wantErr, err)
				}
				if decision != nil {
					t.Fatalf("un cambio rechazado no debía dejar decisión: %+v", decision)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if tc.wantDecision != (decision != nil) {
				t.Fatalf("decisión esperada=%v, obtenida=%+v", tc.wantDecision, decision)
			}
			if decision != nil && decision.reason != tc.wantReason {
				t.Fatalf("motivo esperado %q, obtenido %q", tc.wantReason, decision.reason)
			}
		})
	}
}
