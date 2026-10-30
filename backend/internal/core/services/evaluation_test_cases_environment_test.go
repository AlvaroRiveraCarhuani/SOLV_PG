package services_test

import (
	"context"
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
