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
	tenantID := domain.GetTenantID(ctx)
	return r.GetByIDAndTenant(ctx, id, tenantID)
}

func (r *PostgresExerciseRepository) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	var query string
	var args []interface{}
	if tenantID != "" {
		query = `
			SELECT id, subject_id, title, description, type, due_date, 
			       COALESCE(boilerplate, '') AS boilerplate, 
			       COALESCE(status, 'draft') AS status, 
			       COALESCE(language, 'python') AS language, 
			       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
			       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
			       config, tenant_id, created_at
			FROM exercises
			WHERE id = $1 AND tenant_id = $2
		`
		args = []interface{}{id, tenantID}
	} else {
		query = `
			SELECT id, subject_id, title, description, type, due_date, 
			       COALESCE(boilerplate, '') AS boilerplate, 
			       COALESCE(status, 'draft') AS status, 
			       COALESCE(language, 'python') AS language, 
			       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
			       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
			       config, tenant_id, created_at
			FROM exercises
			WHERE id = $1
		`
		args = []interface{}{id}
	}
	var exercise domain.Exercise
	err := r.db.GetContext(ctx, &exercise, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get exercise by id %s: %w", id, err)
	}
	return &exercise, nil
}

func (r *PostgresExerciseRepository) Create(ctx context.Context, exercise *domain.Exercise) error {
	tenantID := domain.GetTenantID(ctx)
	if exercise.TenantID == "" {
		exercise.TenantID = tenantID
	}
	if exercise.Status == "" {
		exercise.Status = "draft"
	}
	if exercise.Language == "" {
		exercise.Language = "python"
	}
	if exercise.TimeLimitMS == 0 {
		exercise.TimeLimitMS = 1000
	}
	if exercise.MemoryLimitMB == 0 {
		exercise.MemoryLimitMB = 128
	}

	query := `
		INSERT INTO exercises (id, subject_id, title, description, type, due_date, boilerplate, status, language, time_limit_ms, memory_limit_mb, config, tenant_id)
		VALUES (:id, :subject_id, :title, :description, :type, :due_date, :boilerplate, :status, :language, :time_limit_ms, :memory_limit_mb, :config, :tenant_id)
	`
	_, err := r.db.NamedExecContext(ctx, query, exercise)
	if err != nil {
		return fmt.Errorf("failed to create exercise: %w", err)
	}
	return nil
}

func (r *PostgresExerciseRepository) Update(ctx context.Context, exercise *domain.Exercise) error {
	tenantID := domain.GetTenantID(ctx)
	if exercise.TenantID == "" {
		exercise.TenantID = tenantID
	}
	query := `
		UPDATE exercises
		SET title = :title,
		    description = :description,
		    subject_id = :subject_id,
		    due_date = :due_date,
		    boilerplate = :boilerplate,
		    language = :language,
		    time_limit_ms = :time_limit_ms,
		    memory_limit_mb = :memory_limit_mb,
		    config = :config
		WHERE id = :id AND tenant_id = :tenant_id
	`
	res, err := r.db.NamedExecContext(ctx, query, exercise)
	if err != nil {
		return fmt.Errorf("failed to update exercise %s: %w", exercise.ID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("exercise %s not found in tenant", exercise.ID)
	}
	return nil
}

func (r *PostgresExerciseRepository) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	query := `UPDATE exercises SET status = $1 WHERE id = $2 AND tenant_id = $3`
	res, err := r.db.ExecContext(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update status for exercise %s: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("exercise %s not found in tenant", id)
	}
	return nil
}

func (r *PostgresExerciseRepository) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	query := `UPDATE exercises SET config = $1 WHERE id = $2 AND tenant_id = $3`
	res, err := r.db.ExecContext(ctx, query, config, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update config for exercise %s: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("exercise %s not found in tenant", id)
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

func (r *PostgresExerciseRepository) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	query := `
		SELECT id, subject_id, title, description, type, due_date, 
		       COALESCE(boilerplate, '') AS boilerplate, 
		       COALESCE(status, 'draft') AS status, 
		       COALESCE(language, 'python') AS language, 
		       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
		       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
		       config, tenant_id, created_at
		FROM exercises
		WHERE tenant_id = $1 AND subject_id = $2
		ORDER BY created_at DESC
	`
	var exercises []*domain.Exercise
	err := r.db.SelectContext(ctx, &exercises, query, tenantID, subjectID)
	if err != nil {
		return []*domain.Exercise{}, nil
	}
	if exercises == nil {
		exercises = []*domain.Exercise{}
	}
	return exercises, nil
}
