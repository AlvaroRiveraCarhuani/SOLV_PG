package services

import (
	"errors"
	"math"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestRubricValidation(t *testing.T) {
	t.Run("Valid rubric criteria summing to 100%", func(t *testing.T) {
		rubric := &domain.Rubric{
			Criteria: []domain.RubricCriterion{
				{
					ID:     "c1",
					Name:   "Arquitectura",
					Weight: 60,
					Levels: []domain.RubricLevel{
						{Name: "Excelente", Score: 100},
						{Name: "Regular", Score: 50},
					},
				},
				{
					ID:     "c2",
					Name:   "Código Limpio",
					Weight: 40,
					Levels: []domain.RubricLevel{
						{Name: "Excelente", Score: 100},
						{Name: "Deficiente", Score: 20},
					},
				},
			},
		}

		if err := rubric.Validate(); err != nil {
			t.Fatalf("expected valid rubric, got err: %v", err)
		}
	})

	t.Run("Rejects weights not summing to 100%", func(t *testing.T) {
		rubric := &domain.Rubric{
			Criteria: []domain.RubricCriterion{
				{
					ID:     "c1",
					Name:   "Arquitectura",
					Weight: 50,
					Levels: []domain.RubricLevel{
						{Name: "Excelente", Score: 100},
						{Name: "Regular", Score: 50},
					},
				},
			},
		}

		err := rubric.Validate()
		if !errors.Is(err, domain.ErrRubricInvalidWeights) {
			t.Fatalf("expected ErrRubricInvalidWeights, got: %v", err)
		}
	})

	t.Run("Rejects criterion with fewer than 2 levels", func(t *testing.T) {
		rubric := &domain.Rubric{
			Criteria: []domain.RubricCriterion{
				{
					ID:     "c1",
					Name:   "Arquitectura",
					Weight: 100,
					Levels: []domain.RubricLevel{
						{Name: "Único", Score: 100},
					},
				},
			},
		}

		err := rubric.Validate()
		if !errors.Is(err, domain.ErrRubricInvalidLevels) {
			t.Fatalf("expected ErrRubricInvalidLevels, got: %v", err)
		}
	})

	t.Run("Rejects level score out of bounds", func(t *testing.T) {
		rubric := &domain.Rubric{
			Criteria: []domain.RubricCriterion{
				{
					ID:     "c1",
					Name:   "Arquitectura",
					Weight: 100,
					Levels: []domain.RubricLevel{
						{Name: "Excelente", Score: 150},
						{Name: "Regular", Score: 50},
					},
				},
			},
		}

		err := rubric.Validate()
		if !errors.Is(err, domain.ErrRubricInvalidScore) {
			t.Fatalf("expected ErrRubricInvalidScore, got: %v", err)
		}
	})

	t.Run("Rejects duplicate level names", func(t *testing.T) {
		rubric := &domain.Rubric{
			Criteria: []domain.RubricCriterion{
				{
					ID:     "c1",
					Name:   "Arquitectura",
					Weight: 100,
					Levels: []domain.RubricLevel{
						{Name: "Bueno", Score: 100},
						{Name: "Bueno", Score: 50},
					},
				},
			},
		}

		err := rubric.Validate()
		if !errors.Is(err, domain.ErrRubricDuplicateLevelName) {
			t.Fatalf("expected ErrRubricDuplicateLevelName, got: %v", err)
		}
	})
}

func TestRubricCalculateScore(t *testing.T) {
	svc := &DefaultRubricService{}

	criteria := []domain.RubricCriterion{
		{
			ID:     "c1",
			Name:   "Endpoints REST",
			Weight: 60,
			Levels: []domain.RubricLevel{
				{Name: "Excelente", Score: 100},
				{Name: "Bueno", Score: 75},
				{Name: "Regular", Score: 50},
			},
		},
		{
			ID:     "c2",
			Name:   "Código Limpio",
			Weight: 40,
			Levels: []domain.RubricLevel{
				{Name: "Excelente", Score: 100},
				{Name: "Regular", Score: 50},
			},
		},
	}

	t.Run("Calculates correct weighted score", func(t *testing.T) {
		selections := map[string]string{
			"c1": "Bueno",     // 60% * 75 = 45
			"c2": "Excelente", // 40% * 100 = 40
		}

		score, breakdown, err := svc.CalculateScore(criteria, selections)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if math.Abs(score-85.0) > 0.01 {
			t.Fatalf("expected score 85.0, got %.2f", score)
		}

		if len(breakdown) != 2 {
			t.Fatalf("expected 2 items in breakdown, got %d", len(breakdown))
		}
		if breakdown[0].Weighted != 45.0 || breakdown[1].Weighted != 40.0 {
			t.Fatalf("unexpected breakdown values: %+v", breakdown)
		}
	})

	t.Run("Fails if selection is missing for a criterion", func(t *testing.T) {
		selections := map[string]string{
			"c1": "Bueno",
		}

		_, _, err := svc.CalculateScore(criteria, selections)
		if !errors.Is(err, domain.ErrRubricSelectionMissing) {
			t.Fatalf("expected ErrRubricSelectionMissing, got: %v", err)
		}
	})
}
