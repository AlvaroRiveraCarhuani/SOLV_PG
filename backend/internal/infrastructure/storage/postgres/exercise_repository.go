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
		SELECT id, subject_id, title, description, type, due_date, config, tenant_id, created_at
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
		INSERT INTO exercises (id, subject_id, title, description, type, due_date, config, tenant_id)
		VALUES (:id, :subject_id, :title, :description, :type, :due_date, :config, :tenant_id)
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

func (r *PostgresExerciseRepository) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	query := `
		SELECT 
			e.id AS exercise_id,
			e.title,
			COALESCE(e.description, '') AS description,
			s.id AS subject_id,
			s.name AS subject_name,
			s.code AS subject_code,
			e.due_date,
			e.type
		FROM exercises e
		JOIN subjects s ON s.id = e.subject_id AND s.tenant_id = e.tenant_id
		JOIN enrollments en ON en.subject_id = s.id AND en.student_id = $2 AND en.tenant_id = $1
		WHERE e.tenant_id = $1
		  AND (e.due_date IS NULL OR e.due_date > NOW())
		ORDER BY e.due_date ASC NULLS LAST, e.created_at DESC
	`
	var assignments []*domain.DueAssignment
	err := r.db.SelectContext(ctx, &assignments, query, tenantID, studentID)
	if err != nil {
		return []*domain.DueAssignment{}, nil
	}
	if assignments == nil {
		assignments = []*domain.DueAssignment{}
	}
	return assignments, nil
}
