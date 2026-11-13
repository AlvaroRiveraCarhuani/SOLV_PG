package services_test

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestUnpublishExercise_ReturnsToDraftAndKeepsContent(t *testing.T) {
	repo := newMockExerciseRepoForDryRun()
	svc := services.NewEvaluationService(repo, nil, nil, &mockRunnerForDryRun{verdict: domain.VerdictAC})
	ctx := context.Background()

	ex := &domain.Exercise{
		ID:       "ex-unpublish-1",
		Title:    "Práctica publicada",
		Type:     domain.ExerciseTypeAlgorithm,
		Status:   "published",
		Language: "python",
		TenantID: "default",
	}
	if err := repo.Create(ctx, ex); err != nil {
		t.Fatalf("no se pudo crear el ejercicio de prueba: %v", err)
	}

	got, err := svc.UnpublishExercise(ctx, ex.ID, "default")
	if err != nil {
		t.Fatalf("UnpublishExercise devolvió error: %v", err)
	}
	if got.Status != "draft" {
		t.Fatalf("se esperaba estado draft, se obtuvo %q", got.Status)
	}

	stored, err := repo.GetByIDAndTenant(ctx, ex.ID, "default")
	if err != nil {
		t.Fatalf("no se pudo releer el ejercicio: %v", err)
	}
	if stored.Status != "draft" {
		t.Fatalf("el estado persistido debía ser draft, es %q", stored.Status)
	}
	if stored.Title != ex.Title {
		t.Fatalf("el despublicar no debe tocar el contenido: título %q", stored.Title)
	}
}

func TestUnpublishExercise_NotFound(t *testing.T) {
	repo := newMockExerciseRepoForDryRun()
	svc := services.NewEvaluationService(repo, nil, nil, &mockRunnerForDryRun{verdict: domain.VerdictAC})

	_, err := svc.UnpublishExercise(context.Background(), "ex-inexistente", "default")
	if err == nil {
		t.Fatal("se esperaba error para un ejercicio inexistente")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("se esperaba un error de no encontrado, se obtuvo: %v", err)
	}
}
