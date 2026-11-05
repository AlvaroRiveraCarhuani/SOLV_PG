package services_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// TestBulkAddTestCases_EnvironmentGuard verifica la invariante de modalidad:
// los casos de prueba I/O solo se incorporan a ejercicios de juez virtual
// (JUEZ_EFIMERO); los laboratorios IDE_PERSISTENTE se rechazan porque se
// corrigen con rúbrica en SpeedGrader.
func TestBulkAddTestCases_EnvironmentGuard(t *testing.T) {
	ctx := context.Background()

	run := func(t *testing.T, env string, wantErr bool) {
		t.Helper()
		repo := &mockExerciseRepo{
			exercises: map[string]*domain.Exercise{
				"ex-1": {
					ID:              "ex-1",
					Title:           "Ejercicio",
					EnvironmentType: env,
					Config: domain.ExerciseConfig{
						Algorithm: &domain.AlgorithmConfig{},
					},
				},
			},
		}
		svc := services.NewEvaluationService(repo, nil, nil, &checklistMockRunner{})

		err := svc.BulkAddTestCases(ctx, "ex-1", "tenant-1", []domain.TestCase{
			{Input: "1", ExpectedOutput: "2", IsHidden: false},
		})

		if wantErr {
			if !errors.Is(err, domain.ErrTestCasesRequireJudge) {
				t.Fatalf("esperado ErrTestCasesRequireJudge, got %v", err)
			}
			ex := repo.exercises["ex-1"]
			if len(ex.Config.Algorithm.TestCases) != 0 {
				t.Errorf("los casos no deben persistirse en un laboratorio IDE, got %d", len(ex.Config.Algorithm.TestCases))
			}
			return
		}

		if err != nil {
			t.Fatalf("no se esperaba error en juez, got %v", err)
		}
		ex := repo.exercises["ex-1"]
		if len(ex.Config.Algorithm.TestCases) != 1 {
			t.Errorf("esperado 1 caso persistido en juez, got %d", len(ex.Config.Algorithm.TestCases))
		}
	}

	t.Run("juez acepta casos", func(t *testing.T) { run(t, "JUEZ_EFIMERO", false) })
	t.Run("IDE rechaza casos", func(t *testing.T) { run(t, "IDE_PERSISTENTE", true) })
}

// La modalidad no es solo configuración de la práctica: el endpoint de
// evaluación automática rechaza un ejercicio de IDE antes de intentar ejecutar
// nada, aunque tenga una configuración de algoritmia heredada.
func TestEvaluate_RejectsIDEExercise(t *testing.T) {
	repo := &mockExerciseRepo{
		exercises: map[string]*domain.Exercise{
			"ex-ide": {
				ID:              "ex-ide",
				Title:           "Proyecto",
				EnvironmentType: "IDE_PERSISTENTE",
				Language:        "python",
				Config: domain.ExerciseConfig{
					Algorithm: &domain.AlgorithmConfig{TimeLimitMS: 1000, MemoryLimitMB: 64},
				},
			},
		},
	}
	svc := services.NewEvaluationService(repo, nil, nil, &checklistMockRunner{})

	_, err := svc.Evaluate(context.Background(), "ex-ide", "python", base64.StdEncoding.EncodeToString([]byte("print(1)")))
	if !errors.Is(err, domain.ErrEvaluationRequiresJudge) {
		t.Fatalf("esperado ErrEvaluationRequiresJudge, obtenido %v", err)
	}
}
