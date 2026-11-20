package httpdelivery

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestNormalizeExerciseConfig_CanonizaLenguajesYEsqueletos(t *testing.T) {
	ex := &domain.Exercise{Language: "java", Type: domain.ExerciseTypeAlgorithm}
	normalizeExerciseConfig([]byte(`{
		"allowed_languages": ["Python3", "c++", "python", ""],
		"boilerplates": {
			"Python": "a, b = map(int, input().split())",
			"c++": "#include <iostream>",
			"go": "package main",
			"ruby": "   "
		}
	}`), ex)

	allowed := ex.Config.AllowedLanguages
	if len(allowed) != 2 || allowed[0] != "python" || allowed[1] != "cpp" {
		t.Fatalf("lenguajes canonizados inesperados: %v", allowed)
	}
	if ex.Language != "python" {
		t.Errorf("el lenguaje predeterminado debe ser uno admitido, obtuve %q", ex.Language)
	}
	if len(ex.Config.Boilerplates) != 2 {
		t.Fatalf("solo los lenguajes admitidos conservan esqueleto: %v", ex.Config.Boilerplates)
	}
	if ex.Config.Boilerplates["cpp"] != "#include <iostream>" {
		t.Errorf("falta el esqueleto de cpp: %v", ex.Config.Boilerplates)
	}
	if _, ok := ex.Config.Boilerplates["go"]; ok {
		t.Error("un lenguaje fuera de la lista no debe conservar esqueleto")
	}
}

func TestNormalizeExerciseConfig_SinListaNoTocaElLenguaje(t *testing.T) {
	ex := &domain.Exercise{Language: "java", Type: domain.ExerciseTypeAlgorithm}
	normalizeExerciseConfig([]byte(`{"boilerplates": {"java": "class Main {}"}}`), ex)

	if ex.Language != "java" {
		t.Errorf("sin lista declarada el lenguaje no cambia: %q", ex.Language)
	}
	if len(ex.Config.AllowedLanguages) != 0 {
		t.Errorf("sin lista declarada no se inventan lenguajes: %v", ex.Config.AllowedLanguages)
	}
}
