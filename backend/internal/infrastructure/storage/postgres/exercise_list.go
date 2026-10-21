package postgres

import (
	"context"

	"solv-backend/internal/core/domain"
)

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
		SELECT id, subject_id, title, description, type, COALESCE(environment_type, 'JUEZ_EFIMERO') AS environment_type,
		       template_id, difficulty,
		       COALESCE(tags, '{}') AS tags,
		       COALESCE(purpose, 'class') AS purpose,
		       COALESCE(per_student_seed, false) AS per_student_seed,
		       due_date, 
		       COALESCE(boilerplate, '') AS boilerplate, 
		       COALESCE(status, 'draft') AS status, 
		       COALESCE(language, 'python') AS language, 
		       COALESCE(time_limit_ms, 1000) AS time_limit_ms, 
		       COALESCE(memory_limit_mb, 128) AS memory_limit_mb, 
		       COALESCE(reference_solution, '') AS reference_solution,
		       COALESCE(stale, false) AS stale,
		       last_valid_dry_run_at,
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
