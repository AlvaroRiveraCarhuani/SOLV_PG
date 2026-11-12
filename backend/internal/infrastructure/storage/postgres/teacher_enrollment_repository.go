package postgres

import (
	"context"
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
)

func (r *PostgresTeacherRepository) SearchAvailableStudentsForCourse(
	ctx context.Context,
	tenantID, teacherID, subjectID, query string,
	limit int,
) ([]*domain.AvailableStudentItem, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 50 {
		limit = 50
	}

	// 1. Validar que la materia exista y pertenezca al docente en el tenant
	if err := r.assertTeacherOwnsSubject(ctx, tenantID, teacherID, subjectID); err != nil {
		return nil, err
	}

	// 2. Buscar estudiantes del tenant no matriculados en esta materia
	trimmedQuery := strings.TrimSpace(query)
	var items []*domain.AvailableStudentItem

	if trimmedQuery != "" {
		searchPattern := "%" + strings.ToLower(trimmedQuery) + "%"
		sqlQuery := `
			SELECT u.id, u.first_name, u.last_name, u.email
			FROM users u
			WHERE u.tenant_id = $1
			  AND u.role = 'student'
			  AND COALESCE(u.status, 'active') = 'active'
			  AND u.id NOT IN (
				  SELECT e.student_id
				  FROM enrollments e
				  WHERE e.tenant_id = $1 AND e.subject_id = $2
			  )
			  AND (
				  LOWER(u.first_name || ' ' || u.last_name) LIKE $3
				  OR LOWER(u.email) LIKE $3
			  )
			ORDER BY u.last_name ASC, u.first_name ASC
			LIMIT $4
		`
		err := r.db.SelectContext(ctx, &items, sqlQuery, tenantID, subjectID, searchPattern, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to search available students: %w", err)
		}
	} else {
		sqlQuery := `
			SELECT u.id, u.first_name, u.last_name, u.email
			FROM users u
			WHERE u.tenant_id = $1
			  AND u.role = 'student'
			  AND COALESCE(u.status, 'active') = 'active'
			  AND u.id NOT IN (
				  SELECT e.student_id
				  FROM enrollments e
				  WHERE e.tenant_id = $1 AND e.subject_id = $2
			  )
			ORDER BY u.last_name ASC, u.first_name ASC
			LIMIT $3
		`
		err := r.db.SelectContext(ctx, &items, sqlQuery, tenantID, subjectID, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list available students: %w", err)
		}
	}

	if items == nil {
		items = []*domain.AvailableStudentItem{}
	}
	return items, nil
}

func (r *PostgresTeacherRepository) EnrollStudentInCourse(
	ctx context.Context,
	tenantID, teacherID, subjectID, studentID string,
) error {
	// 1. Validar que la materia exista y pertenezca al docente en el tenant
	if err := r.assertTeacherOwnsSubject(ctx, tenantID, teacherID, subjectID); err != nil {
		return err
	}

	// 2. Validar que el estudiante exista en el tenant y esté activo
	var studentExists bool
	studentCheckQuery := `
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE id = $1 AND tenant_id = $2 AND role = 'student' AND COALESCE(status, 'active') = 'active'
		)
	`
	if err := r.db.GetContext(ctx, &studentExists, studentCheckQuery, studentID, tenantID); err != nil {
		return fmt.Errorf("failed to verify student existence: %w", err)
	}
	if !studentExists {
		return domain.ErrNotFound
	}

	// 3. Insertar matrícula de manera idempotente
	enrollQuery := `
		INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING
	`
	if _, err := r.db.ExecContext(ctx, enrollQuery, tenantID, studentID, subjectID); err != nil {
		return fmt.Errorf("failed to enroll student in course: %w", err)
	}

	return nil
}

func (r *PostgresTeacherRepository) assertTeacherOwnsSubject(
	ctx context.Context,
	tenantID, teacherID, subjectID string,
) error {
	var exists bool
	var err error

	if teacherID != "" {
		query := `
			SELECT EXISTS (
				SELECT 1 FROM subjects
				WHERE id = $1 AND tenant_id = $2 AND teacher_id = $3
			)
		`
		err = r.db.GetContext(ctx, &exists, query, subjectID, tenantID, teacherID)
	} else {
		query := `
			SELECT EXISTS (
				SELECT 1 FROM subjects
				WHERE id = $1 AND tenant_id = $2
			)
		`
		err = r.db.GetContext(ctx, &exists, query, subjectID, tenantID)
	}

	if err != nil {
		return fmt.Errorf("failed to verify subject ownership: %w", err)
	}
	if !exists {
		return domain.ErrNotFound
	}
	return nil
}
