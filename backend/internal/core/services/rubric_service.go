package services

import (
	"context"
	"fmt"
	"math"
	"strings"

	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DefaultRubricService struct {
	rubricRepo   domain.RubricRepository
	exerciseRepo domain.ExerciseRepository
	db           *sqlx.DB
}

func NewDefaultRubricService(
	rubricRepo domain.RubricRepository,
	exerciseRepo domain.ExerciseRepository,
	db *sqlx.DB,
) *DefaultRubricService {
	return &DefaultRubricService{
		rubricRepo:   rubricRepo,
		exerciseRepo: exerciseRepo,
		db:           db,
	}
}

func (s *DefaultRubricService) GetByExerciseID(ctx context.Context, exerciseID uuid.UUID) (*domain.Rubric, error) {
	return s.rubricRepo.GetByExerciseID(ctx, exerciseID)
}

func (s *DefaultRubricService) CreateOrUpdate(ctx context.Context, rubric *domain.Rubric) error {
	exercise, err := s.exerciseRepo.GetByID(ctx, rubric.ExerciseID.String())
	if err != nil {
		return fmt.Errorf("failed to verify exercise: %w", err)
	}

	if exercise.EnvironmentType == "JUEZ_EFIMERO" {
		return domain.ErrRubricNotAllowedForJudge
	}

	for i := range rubric.Criteria {
		if rubric.Criteria[i].ID == "" {
			rubric.Criteria[i].ID = uuid.New().String()
		}
	}

	if err := rubric.Validate(); err != nil {
		return err
	}

	return s.rubricRepo.CreateOrUpdate(ctx, rubric)
}

func (s *DefaultRubricService) CalculateScore(
	criteria []domain.RubricCriterion,
	selections map[string]string,
) (float64, []domain.CriterionBreakdown, error) {
	if len(criteria) == 0 {
		return 0, nil, nil
	}

	var totalScore float64
	var breakdown []domain.CriterionBreakdown

	for _, criterion := range criteria {
		selectedLevelName, ok := selections[criterion.ID]
		if !ok {
			return 0, nil, fmt.Errorf("%w: criterion '%s'", domain.ErrRubricSelectionMissing, criterion.Name)
		}

		var matchedLevel *domain.RubricLevel
		for _, lvl := range criterion.Levels {
			if lvl.Name == selectedLevelName {
				matchedLevel = &lvl
				break
			}
		}

		if matchedLevel == nil {
			return 0, nil, fmt.Errorf("invalid level '%s' for criterion '%s'", selectedLevelName, criterion.Name)
		}

		weighted := (criterion.Weight / 100.0) * matchedLevel.Score
		totalScore += weighted

		breakdown = append(breakdown, domain.CriterionBreakdown{
			Criterion: criterion.Name,
			Level:     matchedLevel.Name,
			Score:     matchedLevel.Score,
			Weighted:  math.Round(weighted*100) / 100,
		})
	}

	finalScore := math.Round(totalScore*100) / 100
	return finalScore, breakdown, nil
}

func (s *DefaultRubricService) EvaluateSubmission(
	ctx context.Context,
	submissionID uuid.UUID,
	req domain.RubricEvaluationRequest,
) (*domain.RubricEvaluationResponse, error) {
	var subMeta struct {
		ExerciseID uuid.UUID `db:"exercise_id"`
		TenantID   uuid.UUID `db:"tenant_id"`
	}
	err := s.db.GetContext(ctx, &subMeta, "SELECT exercise_id, tenant_id FROM submissions WHERE id = $1", submissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	rubric, err := s.rubricRepo.GetByExerciseID(ctx, subMeta.ExerciseID)
	if err != nil {
		return nil, fmt.Errorf("rubric not found for exercise: %w", err)
	}

	score, breakdown, err := s.CalculateScore(rubric.Criteria, req.Selections)
	if err != nil {
		return nil, err
	}

	var authorID any
	if userID := strings.TrimSpace(domain.GetUserID(ctx)); userID != "" {
		authorID = userID
	}

	roundedScoreInt := int(math.Round(score))
	_, err = s.db.ExecContext(ctx, `
		UPDATE submissions
		SET score = $1, verdict = 'AC', manual_override = TRUE, override_reason = 'Evaluado por rúbrica', graded_by = COALESCE($3, graded_by)
		WHERE id = $2
	`, roundedScoreInt, submissionID, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to update submission score: %w", err)
	}

	if strings.TrimSpace(req.Comments) != "" && authorID != nil {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO submission_comments (tenant_id, submission_id, author_id, line_number, comment, created_at)
			VALUES ($1, $2, $3, 1, $4, NOW())
		`, subMeta.TenantID, submissionID, authorID, req.Comments); err != nil {
			return nil, fmt.Errorf("failed to insert rubric comment: %w", err)
		}
	}

	return &domain.RubricEvaluationResponse{
		Score:     score,
		Breakdown: breakdown,
		Comments:  req.Comments,
	}, nil
}
