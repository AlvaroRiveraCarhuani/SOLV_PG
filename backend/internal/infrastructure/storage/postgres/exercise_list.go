package postgres

import (
	"context"
	"fmt"

	"solv-backend/internal/core/domain"
)

// dueAssignmentsLimit acota la lista de entregas por vencer. El inicio del alumno
// muestra unas pocas y el resto vive en la materia; sin tope la consulta crecía
// con todo el historial futuro de la carrera.
const dueAssignmentsLimit = 100

func (r *PostgresExerciseRepository) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	// Una entrega pendiente es una práctica publicada, con plazo, de una materia
	// vigente, que el alumno todavía no entregó. Antes la consulta no filtraba
	// nada de eso: contaba borradores, prácticas ya entregadas y materias
	// archivadas, y el inicio las anunciaba como "entregas pendientes".
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
		  AND e.status = 'published'
		  AND s.is_archived = FALSE
		  AND e.due_date > NOW()
		  AND NOT EXISTS (
		      SELECT 1 FROM submissions sub
		      WHERE sub.tenant_id = e.tenant_id
		        AND sub.exercise_id = e.id
		        AND sub.student_id = $2
		  )
		ORDER BY e.due_date ASC
		LIMIT $3
	`
	var assignments []*domain.DueAssignment
	err := r.db.SelectContext(ctx, &assignments, query, tenantID, studentID, dueAssignmentsLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list due assignments: %w", err)
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
