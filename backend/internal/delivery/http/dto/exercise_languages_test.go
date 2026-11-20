package dto

import (
	"testing"

	"solv-backend/internal/core/domain"
)

// El alumno de un entorno multilingüe recibe la lista de lenguajes y el código
// de partida de cada uno; el lenguaje del ejercicio queda como predeterminado.
func TestToExercisePublicResponse_ExponeLenguajesYEsqueletos(t *testing.T) {
	ex := &domain.Exercise{
		ID:              "ex-multi",
		Title:           "Suma de enteros",
		EnvironmentType: "JUEZ_EFIMERO",
		Language:        "python",
		Boilerplate:     "print(sum(map(int, input().split())))",
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{{Input: "1 2", ExpectedOutput: "3"}},
			},
			AllowedLanguages: []string{"python", "cpp"},
			Boilerplates: map[string]string{
				"python": "a, b = map(int, input().split())",
				"cpp":    "#include <iostream>",
			},
		},
	}

	pub := ToExercisePublicResponse(ex)
	if len(pub.AllowedLanguages) != 2 || pub.AllowedLanguages[0] != "python" || pub.AllowedLanguages[1] != "cpp" {
		t.Fatalf("lenguajes admitidos inesperados: %v", pub.AllowedLanguages)
	}
	if pub.Boilerplates["cpp"] != "#include <iostream>" {
		t.Fatalf("falta el esqueleto de cpp: %v", pub.Boilerplates)
	}
	if pub.Language != "python" {
		t.Errorf("el lenguaje predeterminado cambió: %q", pub.Language)
	}
}

// Sin lenguajes declarados se conserva el contrato anterior: ni lista ni mapas
// adicionales en la respuesta.
func TestToExercisePublicResponse_SinLenguajesDeclaradosNoAgregaCampos(t *testing.T) {
	pub := ToExercisePublicResponse(&domain.Exercise{
		ID:       "ex-simple",
		Language: "python",
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{},
		},
	})
	if len(pub.AllowedLanguages) != 0 {
		t.Errorf("un ejercicio de un lenguaje no debería declarar lista: %v", pub.AllowedLanguages)
	}
	if len(pub.Boilerplates) != 0 {
		t.Errorf("un ejercicio de un lenguaje no debería declarar esqueletos: %v", pub.Boilerplates)
	}
}
