package postgres

import (
	"context"
	"fmt"
	"solv-backend/internal/core/domain"

	"github.com/jmoiron/sqlx"
)

type PostgresSubjectRepository struct {
	db *sqlx.DB
}

func NewPostgresSubjectRepository(db *sqlx.DB) *PostgresSubjectRepository {
	return &PostgresSubjectRepository{db: db}
}

func (r *PostgresSubjectRepository) Create(ctx context.Context, subject *domain.Subject) error {
	query := `
		INSERT INTO subjects (id, tenant_id, name, code, classroom_course_id, created_at, updated_at)
		VALUES (:id, :tenant_id, :name, :code, :classroom_course_id, NOW(), NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, subject)
	if err != nil {
		return fmt.Errorf("failed to create subject: %w", err)
	}
	return nil
}

func (r *PostgresSubjectRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.Subject, error) {
	var s domain.Subject
	query := `SELECT id, tenant_id, name, code, classroom_course_id, created_at, updated_at FROM subjects WHERE tenant_id = $1 AND id = $2`
	err := r.db.GetContext(ctx, &s, query, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("subject not found: %w", err)
	}
	return &s, nil
}

func (r *PostgresSubjectRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	var list []*domain.Subject
	query := `SELECT id, tenant_id, name, code, classroom_course_id, created_at, updated_at FROM subjects WHERE tenant_id = $1 ORDER BY name ASC`
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
		SELECT s.id, s.tenant_id, s.name, s.code, s.classroom_course_id, s.created_at, s.updated_at
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

