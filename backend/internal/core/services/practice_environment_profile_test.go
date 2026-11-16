package services

import (
	"testing"

	"solv-backend/internal/core/domain"
)

// El perfil del entorno sale del ejercicio, no de la petición. Sin catálogo o
// sin ejercicio el arranque no puede afirmar que sea un examen: cae en práctica.
func TestResolveEnvironmentProfile(t *testing.T) {
	svc := &PracticeEnvironmentService{}

	cases := []struct {
		nombre   string
		exercise *domain.Exercise
		esperado domain.EnvironmentProfile
	}{
		{"sin ejercicio", nil, domain.EnvironmentProfilePractice},
		{"examen", &domain.Exercise{Purpose: string(domain.ExercisePurposeExam)}, domain.EnvironmentProfileExam},
		{"clase", &domain.Exercise{Purpose: string(domain.ExercisePurposeClass)}, domain.EnvironmentProfilePractice},
		{"propósito vacío", &domain.Exercise{}, domain.EnvironmentProfilePractice},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			if obtenido := svc.resolveEnvironmentProfile(tc.exercise); obtenido != tc.esperado {
				t.Fatalf("esperado %q, obtenido %q", tc.esperado, obtenido)
			}
		})
	}
}

// La práctica no tiene intentos: su volumen es el de siempre. El primer intento
// de examen es el 1, para que su volumen no coincida con el de práctica.
func TestInitialExamAttempt(t *testing.T) {
	cases := []struct {
		profile  domain.EnvironmentProfile
		esperado int
	}{
		{domain.EnvironmentProfilePractice, 0},
		{domain.EnvironmentProfileExam, 1},
		{"", 0},
	}

	for _, tc := range cases {
		t.Run(string(tc.profile), func(t *testing.T) {
			if obtenido := initialExamAttempt(tc.profile); obtenido != tc.esperado {
				t.Fatalf("esperado %d, obtenido %d", tc.esperado, obtenido)
			}
		})
	}
}
