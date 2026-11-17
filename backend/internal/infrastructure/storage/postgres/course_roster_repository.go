package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// PostgresCourseRosterRepository resuelve el padrón de una materia sobre
// `enrollments`, con la identidad desde `users` y el estado del entorno desde
// `environments`.
type PostgresCourseRosterRepository struct {
	db *sqlx.DB
}

func NewPostgresCourseRosterRepository(db *sqlx.DB) *PostgresCourseRosterRepository {
	return &PostgresCourseRosterRepository{db: db}
}

func (r *PostgresCourseRosterRepository) ListCourseStudents(
	ctx context.Context,
	tenantID, subjectID string,
) ([]*domain.CourseStudentItem, error) {
	query := `
		SELECT
			e.student_id,
			TRIM(CONCAT(u.first_name, ' ', u.last_name)) AS full_name,
			u.email,
			-- Las matrículas anteriores al alta del padrón pueden tener la fecha
			-- vacía; un NULL no puede escanearse a time.Time.
			COALESCE(e.enrolled_at, u.created_at, NOW()) AS enrolled_at,
			COALESCE(NULLIF(TRIM(e.source_system), ''), 'manual') AS source_system,
			COALESCE((
				SELECT w.status
				FROM environments w
				WHERE w.tenant_id = e.tenant_id
				  AND w.subject_id = e.subject_id
				  AND w.student_id = e.student_id
				ORDER BY w.updated_at DESC NULLS LAST, w.created_at DESC
				LIMIT 1
			), '') AS workspace_status
		FROM enrollments e
		INNER JOIN users u ON u.id = e.student_id AND u.tenant_id = e.tenant_id
		WHERE e.tenant_id = $1 AND e.subject_id = $2
		ORDER BY e.enrolled_at ASC, u.last_name ASC, u.first_name ASC
	`
	var items []*domain.CourseStudentItem
	if err := r.db.SelectContext(ctx, &items, query, tenantID, subjectID); err != nil {
		return nil, fmt.Errorf("failed to list course students: %w", err)
	}
	if items == nil {
		items = []*domain.CourseStudentItem{}
	}
	return items, nil
}

func (r *PostgresCourseRosterRepository) ListAvailableStudents(
	ctx context.Context,
	tenantID, subjectID, query string,
	limit int,
) ([]*domain.CourseAvailableStudent, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 50 {
		limit = 50
	}

	const base = `
		SELECT u.id, u.first_name, u.last_name, u.email
		FROM users u
		WHERE u.tenant_id = $1
		  AND u.role = 'student'
		  AND COALESCE(u.status, 'active') = 'active'
		  AND NOT EXISTS (
			  SELECT 1 FROM enrollments e
			  WHERE e.tenant_id = $1 AND e.subject_id = $2 AND e.student_id = u.id
		  )
	`
	args := []any{tenantID, subjectID}

	trimmed := strings.TrimSpace(query)
	sqlQuery := base
	if trimmed != "" {
		sqlQuery += `
		  AND (
			  LOWER(u.first_name || ' ' || u.last_name) LIKE $3
			  OR LOWER(u.email) LIKE $3
		  )`
		args = append(args, "%"+strings.ToLower(trimmed)+"%")
	}
	sqlQuery += fmt.Sprintf(" ORDER BY u.last_name ASC, u.first_name ASC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	var items []*domain.CourseAvailableStudent
	if err := r.db.SelectContext(ctx, &items, sqlQuery, args...); err != nil {
		return nil, fmt.Errorf("failed to list available students: %w", err)
	}
	if items == nil {
		items = []*domain.CourseAvailableStudent{}
	}
	return items, nil
}

func (r *PostgresCourseRosterRepository) TeacherOwnsSubject(
	ctx context.Context,
	tenantID, subjectID, teacherID string,
) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1 FROM subjects
			WHERE id = $1 AND tenant_id = $2 AND teacher_id::text = $3
		)
	`, subjectID, tenantID, teacherID); err != nil {
		return false, fmt.Errorf("failed to check subject ownership: %w", err)
	}
	return exists, nil
}

func (r *PostgresCourseRosterRepository) SubjectExists(ctx context.Context, tenantID, subjectID string) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS (SELECT 1 FROM subjects WHERE id = $1 AND tenant_id = $2)
	`, subjectID, tenantID); err != nil {
		return false, fmt.Errorf("failed to check subject existence: %w", err)
	}
	return exists, nil
}

func (r *PostgresCourseRosterRepository) EnrollStudentWithSource(
	ctx context.Context,
	tenantID, subjectID, studentID, source string,
) error {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at, source_system)
		SELECT gen_random_uuid(), $1, u.id, $3, NOW(), $4
		FROM users u
		WHERE u.id = $2
		  AND u.tenant_id = $1
		  AND u.role = 'student'
		  AND COALESCE(u.status, 'active') = 'active'
		ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING
	`, tenantID, studentID, subjectID, source)
	if err != nil {
		return fmt.Errorf("failed to enroll student: %w", err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		// Puede ser una matrícula ya existente o un estudiante inexistente: se
		// distingue para no reportar un alta que no ocurrió.
		var enrolled bool
		if errCheck := r.db.GetContext(ctx, &enrolled, `
			SELECT EXISTS (
				SELECT 1 FROM enrollments
				WHERE tenant_id = $1 AND subject_id = $2 AND student_id = $3
			)
		`, tenantID, subjectID, studentID); errCheck == nil && !enrolled {
			return domain.ErrNotFound
		}
	}
	return nil
}

func (r *PostgresCourseRosterRepository) RemoveStudentFromSubject(
	ctx context.Context,
	tenantID, subjectID, studentID string,
) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM enrollments
		WHERE tenant_id = $1 AND subject_id = $2 AND student_id = $3
	`, tenantID, subjectID, studentID)
	if err != nil {
		return fmt.Errorf("failed to remove student from course: %w", err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
