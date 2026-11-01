package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"

	"github.com/google/uuid"
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
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, review_status, ast_result, generated_cases, complexity_analysis, score, execution_time_ms, memory_used_mb, submitted_at)
		VALUES (:id, :tenant_id, :exercise_id, :student_id, :workspace_id, :code, :verdict, COALESCE(NULLIF(:review_status, ''), 'not_required'), :ast_result, :generated_cases, :complexity_analysis, :score, :execution_time_ms, :memory_used_mb, NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, sub)
	if err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}
	return nil
}

func (r *PostgresSubmissionRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	// Leer con un id mal formado es validación, no una entrega ausente ni un
	// fallo del servidor; el transporte puede responder 400 con este error.
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidResourceID, id)
	}

	var sub domain.Submission
	query := `
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, review_status,
		       COALESCE(ast_result, '{}'::jsonb) AS ast_result,
		       COALESCE(generated_cases, '{}'::jsonb) AS generated_cases,
		       COALESCE(complexity_analysis, '{}'::jsonb) AS complexity_analysis,
		       execution_time_ms, memory_used_mb,
		       manual_override, override_reason, score, graded_by, submitted_at
		FROM submissions
		WHERE tenant_id = $1 AND id = $2
	`
	err := r.db.GetContext(ctx, &sub, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Una entrega que no existe (o es de otra institución) es un 404,
			// no un fallo interno del servidor.
			return nil, fmt.Errorf("%w: la entrega %s no existe en esta institución", domain.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to load submission: %w", err)
	}
	return &sub, nil
}

func (r *PostgresSubmissionRepository) ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*domain.Submission, error) {
	var list []*domain.Submission
	query := `
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, review_status,
		       COALESCE(ast_result, '{}'::jsonb) AS ast_result,
		       COALESCE(generated_cases, '{}'::jsonb) AS generated_cases,
		       COALESCE(complexity_analysis, '{}'::jsonb) AS complexity_analysis,
		       execution_time_ms, memory_used_mb,
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
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, review_status,
		       COALESCE(ast_result, '{}'::jsonb) AS ast_result,
		       COALESCE(generated_cases, '{}'::jsonb) AS generated_cases,
		       COALESCE(complexity_analysis, '{}'::jsonb) AS complexity_analysis,
		       execution_time_ms, memory_used_mb,
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
		SELECT id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, review_status,
		       COALESCE(ast_result, '{}'::jsonb) AS ast_result,
		       COALESCE(generated_cases, '{}'::jsonb) AS generated_cases,
		       COALESCE(complexity_analysis, '{}'::jsonb) AS complexity_analysis,
		       execution_time_ms, memory_used_mb,
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
	// Un id mal formado no puede alcanzar ninguna fila: es validación, no una
	// entrega ausente ni un fallo del servidor.
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return fmt.Errorf("%w: %s", domain.ErrInvalidResourceID, id)
	}

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
		// Sin fila no hubo override: el servicio no debe notificar ni
		// responder como si la nota se hubiera guardado.
		return fmt.Errorf("%w: la entrega %s no existe en esta institución", domain.ErrNotFound, id)
	}
	return nil
}
