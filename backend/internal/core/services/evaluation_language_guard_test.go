package services

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func exerciseWithLanguages(allowed ...string) *domain.Exercise {
	return &domain.Exercise{
		ID:       "ex-multi",
		Language: "python",
		Config: domain.ExerciseConfig{
			Algorithm:        &domain.AlgorithmConfig{},
			AllowedLanguages: allowed,
		},
	}
}

func TestLanguageAdmittedByExercise_RechazaLenguajeFueraDeLaPractica(t *testing.T) {
	if canonical, admitted := languageAdmittedByExercise(exerciseWithLanguages("python", "cpp"), "go"); admitted {
		t.Fatalf("go no está en la práctica y fue admitido como %q", canonical)
	}
}

func TestLanguageAdmittedByExercise_AceptaAliasDelLenguajeAdmitido(t *testing.T) {
	canonical, admitted := languageAdmittedByExercise(exerciseWithLanguages("python", "cpp"), "c++")
	if !admitted {
		t.Fatal("el alias c++ del lenguaje admitido fue rechazado")
	}
	if canonical != "cpp" {
		t.Fatalf("el alias no se canonizó: %q", canonical)
	}
}

func TestLanguageAdmittedByExercise_SinListaAdmiteElLenguajeDelEntorno(t *testing.T) {
	if _, admitted := languageAdmittedByExercise(exerciseWithLanguages(), "java"); !admitted {
		t.Fatal("sin lista declarada la práctica no debe limitar lenguajes")
	}
}

func TestLanguageAdmittedByExercise_AdmiteElAliasCortoDelCatalogo(t *testing.T) {
	canonical, admitted := languageAdmittedByExercise(exerciseWithLanguages("python", "javascript"), "node")
	if !admitted {
		t.Fatal("el alias node del lenguaje admitido fue rechazado")
	}
	if canonical != "javascript" {
		t.Fatalf("el alias no se canonizó: %q", canonical)
	}
}
