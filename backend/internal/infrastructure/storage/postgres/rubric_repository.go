package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresRubricRepository struct {
	db *sqlx.DB
}

func NewPostgresRubricRepository(db *sqlx.DB) *PostgresRubricRepository {
	return &PostgresRubricRepository{db: db}
}

type rubricDBRow struct {
	ID         uuid.UUID    `db:"id"`
	ExerciseID uuid.UUID    `db:"exercise_id"`
	Criteria   []byte       `db:"criteria"`
	CreatedAt  sql.NullTime `db:"created_at"`
	UpdatedAt  sql.NullTime `db:"updated_at"`
}

func (r *PostgresRubricRepository) GetByExerciseID(ctx context.Context, exerciseID uuid.UUID) (*domain.Rubric, error) {
	query := `
		SELECT id, exercise_id, criteria, created_at, updated_at
		FROM rubrics
		WHERE exercise_id = $1
	`
	var row rubricDBRow
	err := r.db.GetContext(ctx, &row, query, exerciseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRubricNotFound
		}
		return nil, fmt.Errorf("failed to get rubric by exercise_id: %w", err)
	}

	var criteria []domain.RubricCriterion
	if len(row.Criteria) > 0 {
		if err := json.Unmarshal(row.Criteria, &criteria); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rubric criteria: %w", err)
		}
	}

	rubric := &domain.Rubric{
		ID:         row.ID,
		ExerciseID: row.ExerciseID,
		Criteria:   criteria,
	}
	if row.CreatedAt.Valid {
		rubric.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		rubric.UpdatedAt = row.UpdatedAt.Time
	}

	return rubric, nil
}

func (r *PostgresRubricRepository) CreateOrUpdate(ctx context.Context, rubric *domain.Rubric) error {
	criteriaJSON, err := rubric.CriteriaJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal rubric criteria: %w", err)
	}

	query := `
		INSERT INTO rubrics (id, exercise_id, criteria, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (exercise_id) DO UPDATE SET
			criteria = EXCLUDED.criteria,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	if rubric.ID == uuid.Nil {
		rubric.ID = uuid.New()
	}

	var returnedID uuid.UUID
	var createdAt, updatedAt sql.NullTime
	err = r.db.QueryRowContext(ctx, query, rubric.ID, rubric.ExerciseID, criteriaJSON).Scan(&returnedID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert rubric: %w", err)
	}

	rubric.ID = returnedID
	if createdAt.Valid {
		rubric.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		rubric.UpdatedAt = updatedAt.Time
	}

	return nil
}
