package postgres

import (
	"context"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"

	"github.com/jmoiron/sqlx"
)

type PostgresSubjectRepository struct {
	db *sqlx.DB
}

func NewPostgresSubjectRepository(db *sqlx.DB) *PostgresSubjectRepository {
	return &PostgresSubjectRepository{db: db}
}

func (r *PostgresSubjectRepository) Create(ctx context.Context, subject *domain.Subject) error {
	// El curso hereda su identidad del catálogo y su código visible se compone con
	// la sección: la materia "SIS-211" con sección "1" da el curso "SIS-211-1". El
	// catálogo guarda siempre el código de la materia, sin sección, porque dos
	// paralelos son la misma materia dictada dos veces.
	//
	// Si la materia ya se dictó antes, manda el nombre vigente del catálogo y no el
	// que llegó en el alta, para que dos cursos del mismo código no terminen
	// llamándose distinto.
	if baseCode := strings.ToUpper(strings.TrimSpace(subject.Code)); baseCode != "" {
		if subject.CatalogID == nil {
			catalogID, canonicalName, err := r.resolveCatalog(ctx, subject.TenantID, baseCode, subject.Name)
			if err != nil {
				return err
			}
			subject.CatalogID = &catalogID
			subject.Name = canonicalName
		}
		subject.Code = baseCode
		if subject.Section != nil {
			if section := strings.ToUpper(strings.TrimSpace(*subject.Section)); section != "" {
				subject.Code = baseCode + "-" + section
			}
		}
	}

	query := `
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id, academic_period_id, classroom_course_id, template_id, subject_catalog_id, section, created_at, updated_at)
		VALUES (:id, :tenant_id, :name, :code, :teacher_id, :academic_period_id, :classroom_course_id, :template_id, :subject_catalog_id, :section, NOW(), NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, subject)
	if err != nil {
		return fmt.Errorf("failed to create subject: %w", err)
	}
	return nil
}

// resolveCatalog devuelve la materia del catálogo, creándola si es la primera vez
// que se dicta. El código se guarda en mayúsculas porque es la forma en que se
// muestra en toda la interfaz y la que usa el índice único del catálogo.
func (r *PostgresSubjectRepository) resolveCatalog(ctx context.Context, tenantID, code, name string) (string, string, error) {
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO subject_catalog (tenant_id, code, name)
		VALUES ($1, upper($2), $3)
		ON CONFLICT (tenant_id, upper(code)) DO NOTHING
	`, tenantID, code, name); err != nil {
		return "", "", fmt.Errorf("failed to register subject catalog: %w", err)
	}

	var catalog struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	if err := r.db.GetContext(ctx, &catalog, `
		SELECT id, name FROM subject_catalog
		WHERE tenant_id = $1 AND upper(code) = upper($2)
	`, tenantID, code); err != nil {
		return "", "", fmt.Errorf("failed to read subject catalog: %w", err)
	}
	return catalog.ID, catalog.Name, nil
}

func (r *PostgresSubjectRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.Subject, error) {
	var s domain.Subject
	query := `
		SELECT 
			s.id, s.tenant_id, s.name, s.code, s.subject_catalog_id, s.section, s.teacher_id, 
			COALESCE(NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''), '') as teacher_name,
			s.academic_period_id, s.is_archived, s.classroom_course_id, s.template_id, s.created_at, s.updated_at 
		FROM subjects s
		LEFT JOIN users u ON s.teacher_id = u.id
		WHERE s.tenant_id = $1 AND s.id = $2
	`
	err := r.db.GetContext(ctx, &s, query, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("subject not found: %w", err)
	}
	return &s, nil
}

func (r *PostgresSubjectRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	var list []*domain.Subject
	query := `
		SELECT 
			s.id, s.tenant_id, s.name, s.code, s.subject_catalog_id, s.section, s.teacher_id, 
			COALESCE(NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''), '') as teacher_name,
			s.academic_period_id, s.is_archived, s.classroom_course_id, s.template_id, s.created_at, s.updated_at 
		FROM subjects s
		LEFT JOIN users u ON s.teacher_id = u.id
		WHERE s.tenant_id = $1 
		ORDER BY s.created_at DESC
	`
	err := r.db.SelectContext(ctx, &list, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subjects: %w", err)
	}
	return list, nil
}

func (r *PostgresSubjectRepository) EnrollStudent(ctx context.Context, enrollment *domain.Enrollment) error {
	query := `
		INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at)
		VALUES (:id, :tenant_id, :student_id, :subject_id, NOW())
		ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING
	`
	_, err := r.db.NamedExecContext(ctx, query, enrollment)
	if err != nil {
		return fmt.Errorf("failed to enroll student: %w", err)
	}
	return nil
}

func (r *PostgresSubjectRepository) ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	var studentIDs []string
	query := `SELECT student_id FROM enrollments WHERE tenant_id = $1 AND subject_id = $2 ORDER BY enrolled_at ASC`
	err := r.db.SelectContext(ctx, &studentIDs, query, tenantID, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list enrolled students: %w", err)
	}
	return studentIDs, nil
}

func (r *PostgresSubjectRepository) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Subject, error) {
	var list []*domain.Subject
	query := `
		SELECT s.id, s.tenant_id, s.name, s.code, s.subject_catalog_id, s.section, s.teacher_id, s.academic_period_id, s.is_archived, s.classroom_course_id, s.template_id, s.created_at, s.updated_at
		FROM subjects s
		INNER JOIN enrollments e ON s.id = e.subject_id AND s.tenant_id = e.tenant_id
		WHERE s.tenant_id = $1 AND e.student_id = $2
		ORDER BY s.name ASC
	`
	err := r.db.SelectContext(ctx, &list, query, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list student subjects: %w", err)
	}
	return list, nil
}

func (r *PostgresSubjectRepository) ReassignTeacher(ctx context.Context, tenantID, subjectID, newTeacherID string) error {
	query := `
		UPDATE subjects
		SET teacher_id = $1
		WHERE tenant_id = $2 AND id = $3
	`
	res, err := r.db.ExecContext(ctx, query, newTeacherID, tenantID, subjectID)
	if err != nil {
		return fmt.Errorf("failed to reassign subject teacher: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("subject not found")
	}
	return nil
}

func (r *PostgresSubjectRepository) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	query := `
		UPDATE subjects
		SET is_archived = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
	`
	res, err := r.db.ExecContext(ctx, query, isArchived, tenantID, subjectID)
	if err != nil {
		return fmt.Errorf("failed to update subject archive status: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("subject not found")
	}
	return nil
}

func (r *PostgresSubjectRepository) Update(ctx context.Context, tenantID, subjectID, name, code string) error {
	query := `
		UPDATE subjects
		SET name = $1, code = $2, updated_at = NOW()
		WHERE tenant_id = $3 AND id = $4
	`
	res, err := r.db.ExecContext(ctx, query, name, code, tenantID, subjectID)
	if err != nil {
		return fmt.Errorf("failed to update subject: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("subject not found")
	}
	return nil
}

func (r *PostgresSubjectRepository) GetTemplateStatus(ctx context.Context, templateID string) (string, error) {
	var status string
	query := `SELECT COALESCE(status, 'pending') FROM lab_templates WHERE id = $1`
	err := r.db.GetContext(ctx, &status, query, templateID)
	if err != nil {
		return "", fmt.Errorf("error querying template status: %w", err)
	}
	return status, nil
}
