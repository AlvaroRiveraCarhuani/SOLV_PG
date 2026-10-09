package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRubricNotFound           = errors.New("rubric not found")
	ErrRubricInvalidWeights     = errors.New("rubric criteria weights must sum to 100%")
	ErrRubricInvalidLevels      = errors.New("each criterion must have at least 2 levels")
	ErrRubricInvalidScore       = errors.New("level scores must be between 0 and 100")
	ErrRubricDuplicateLevelName = errors.New("level names must be unique within a criterion")
	ErrRubricNotAllowedForJudge = errors.New("rubrics are only permitted for IDE_PERSISTENTE exercises")
	ErrRubricSelectionMissing   = errors.New("a selection is required for every criterion")
)

type RubricLevel struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

type RubricCriterion struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Weight      float64       `json:"weight"`
	Levels      []RubricLevel `json:"levels"`
}

type Rubric struct {
	ID         uuid.UUID         `db:"id" json:"id"`
	ExerciseID uuid.UUID         `db:"exercise_id" json:"exercise_id"`
	Criteria   []RubricCriterion `json:"criteria"`
	CreatedAt  time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time         `db:"updated_at" json:"updated_at"`
}

type RubricEvaluationRequest struct {
	Selections map[string]string `json:"selections"`
	Comments   string            `json:"comments"`
}

type CriterionBreakdown struct {
	Criterion string  `json:"criterion"`
	Level     string  `json:"level"`
	Score     float64 `json:"score"`
	Weighted  float64 `json:"weighted"`
}

type RubricEvaluationResponse struct {
	Score     float64              `json:"score"`
	Breakdown []CriterionBreakdown `json:"breakdown"`
	Comments  string               `json:"comments,omitempty"`
}

func (r *Rubric) Validate() error {
	if len(r.Criteria) == 0 {
		return nil
	}

	var totalWeight float64
	for _, c := range r.Criteria {
		totalWeight += c.Weight

		if len(c.Levels) < 2 {
			return fmt.Errorf("%w: criterion '%s' has %d levels", ErrRubricInvalidLevels, c.Name, len(c.Levels))
		}

		levelNames := make(map[string]bool)
		for _, lvl := range c.Levels {
			if lvl.Score < 0 || lvl.Score > 100 {
				return fmt.Errorf("%w: level '%s' in '%s' has score %.2f", ErrRubricInvalidScore, lvl.Name, c.Name, lvl.Score)
			}
			if levelNames[lvl.Name] {
				return fmt.Errorf("%w: level '%s' in '%s'", ErrRubricDuplicateLevelName, lvl.Name, c.Name)
			}
			levelNames[lvl.Name] = true
		}
	}

	if math.Abs(totalWeight-100.0) > 0.01 {
		return fmt.Errorf("%w: current sum is %.2f%%", ErrRubricInvalidWeights, totalWeight)
	}

	return nil
}

func (r *Rubric) CriteriaJSON() ([]byte, error) {
	if r.Criteria == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(r.Criteria)
}
