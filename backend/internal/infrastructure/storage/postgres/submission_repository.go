package postgres

import (
	"context"
	"fmt"
	"solv-backend/internal/core/domain"

	"github.com/jmoiron/sqlx"
)

type PostgresSubmissionRepository struct {
	db *sqlx.DB
}

func NewPostgresSubmissionRepository(db *sqlx.DB) *PostgresSubmissionRepository {
	return &PostgresSubmissionRepository{db: db}
}

func (r *PostgresSubmissionRepository) Create(ctx context.Context, sub *domain.Submission) error {
	query := `
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, ast_result, execution_time_ms, memory_used_mb, submitted_at)
		VALUES (:id, :tenant_id, :exercise_id, :student_id, :workspace_id, :code, :verdict, :ast_result, :execution_time_ms, :memory_used_mb, NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, sub)
	if err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}
	return nil
}

func (r *PostgresSubmissionRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	var sub domain.Submission
	query := `
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, ast_result, execution_time_ms, memory_used_mb,
		       manual_override, override_reason, score, graded_by, submitted_at
		FROM submissions
		WHERE tenant_id = $1 AND id = $2
	`
	err := r.db.GetContext(ctx, &sub, query, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}
	return &sub, nil
}

func (r *PostgresSubmissionRepository) ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*domain.Submission, error) {
	var list []*domain.Submission
	query := `
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, ast_result, execution_time_ms, memory_used_mb,
		       manual_override, override_reason, score, graded_by, submitted_at
		FROM submissions
		WHERE tenant_id = $1 AND exercise_id = $2 AND student_id = $3
		ORDER BY submitted_at DESC
	`
	err := r.db.SelectContext(ctx, &list, query, tenantID, exerciseID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list student submissions: %w", err)
	}
	return list, nil
}

func (r *PostgresSubmissionRepository) ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*domain.Submission, error) {
	var list []*domain.Submission
	query := `
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, ast_result, execution_time_ms, memory_used_mb,
		       manual_override, override_reason, score, graded_by, submitted_at
		FROM submissions
		WHERE tenant_id = $1 AND exercise_id = $2
		ORDER BY submitted_at DESC
	`
	err := r.db.SelectContext(ctx, &list, query, tenantID, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list exercise submissions: %w", err)
	}
	return list, nil
}

func (r *PostgresSubmissionRepository) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Submission, error) {
	var list []*domain.Submission
	query := `
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, ast_result, execution_time_ms, memory_used_mb,
		       manual_override, override_reason, score, graded_by, submitted_at
		FROM submissions
		WHERE tenant_id = $1 AND student_id = $2
		ORDER BY submitted_at DESC
	`
	err := r.db.SelectContext(ctx, &list, query, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions by student: %w", err)
	}
	return list, nil
}

func (r *PostgresSubmissionRepository) UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	query := `
		UPDATE submissions
		SET verdict = $1, manual_override = TRUE, override_reason = $2, score = $3, graded_by = $4
		WHERE tenant_id = $5 AND id = $6
	`
	res, err := r.db.ExecContext(ctx, query, verdict, reason, score, gradedBy, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to update submission override: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("submission not found")
	}
	return nil
}

