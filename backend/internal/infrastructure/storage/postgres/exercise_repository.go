package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresExerciseRepository struct {
	db *sqlx.DB
}

func NewPostgresExerciseRepository(db *sqlx.DB) domain.ExerciseRepository {
	return &PostgresExerciseRepository{db: db}
}

func (r *PostgresExerciseRepository) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	query := `
		SELECT id, title, description, type, config, tenant_id, created_at
		FROM exercises
		WHERE id = $1 AND tenant_id = $2
	`
	var exercise domain.Exercise
	tenantID := domain.GetTenantID(ctx)
	err := r.db.GetContext(ctx, &exercise, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise by id: %w", err)
	}
	return &exercise, nil
}

func (r *PostgresExerciseRepository) Create(ctx context.Context, exercise *domain.Exercise) error {
	tenantID := domain.GetTenantID(ctx)
	if exercise.TenantID == "" {
		exercise.TenantID = tenantID
	}
	query := `
		INSERT INTO exercises (id, title, description, type, config, tenant_id)
		VALUES (:id, :title, :description, :type, :config, :tenant_id)
	`
	_, err := r.db.NamedExecContext(ctx, query, exercise)
	if err != nil {
		return fmt.Errorf("failed to create exercise: %w", err)
	}
	return nil
}

func (r *PostgresExerciseRepository) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	tenantID := domain.GetTenantID(ctx)
	query := `
		UPDATE exercises
		SET config = jsonb_set(config, '{database,expected_json}', to_jsonb($2::text))
		WHERE id = $1 AND tenant_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, id, expectedJSON, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update expected_json for exercise %s: %w", id, err)
	}
	return nil
}
