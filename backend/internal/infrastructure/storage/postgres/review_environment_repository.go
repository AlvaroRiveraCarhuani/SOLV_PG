package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

type PostgresReviewEnvironmentRepository struct {
	db *sqlx.DB
}

func NewPostgresReviewEnvironmentRepository(db *sqlx.DB) *PostgresReviewEnvironmentRepository {
	return &PostgresReviewEnvironmentRepository{db: db}
}

const reviewEnvironmentColumns = `
	id, tenant_id, submission_id, teacher_id, container_id, volume_name,
	access_url, status, error_code, error_message, last_activity_at,
	expires_at, last_rx_bytes, last_tx_bytes, created_at, updated_at
`

func (r *PostgresReviewEnvironmentRepository) Create(ctx context.Context, env *domain.ReviewEnvironment) error {
	query := `
		INSERT INTO review_test_environments (
			id, tenant_id, submission_id, teacher_id, container_id, volume_name,
			access_url, status, error_code, error_message, last_activity_at,
			expires_at, last_rx_bytes, last_tx_bytes, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :submission_id, :teacher_id, :container_id, :volume_name,
			:access_url, :status, :error_code, :error_message, :last_activity_at,
			:expires_at, :last_rx_bytes, :last_tx_bytes, :created_at, :updated_at
		)
	`
	if _, err := r.db.NamedExecContext(ctx, query, env); err != nil {
		return fmt.Errorf("failed to create review environment: %w", err)
	}
	return nil
}

func (r *PostgresReviewEnvironmentRepository) GetActiveBySubmission(ctx context.Context, tenantID, submissionID string) (*domain.ReviewEnvironment, error) {
	query := `SELECT ` + reviewEnvironmentColumns + `
		FROM review_test_environments
		WHERE tenant_id = $1 AND submission_id = $2
		  AND status IN ('starting', 'running')
		ORDER BY created_at DESC
		LIMIT 1`
	var env domain.ReviewEnvironment
	if err := r.db.GetContext(ctx, &env, query, tenantID, submissionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read active review environment: %w", err)
	}
	return &env, nil
}

func (r *PostgresReviewEnvironmentRepository) ListActive(ctx context.Context) ([]*domain.ReviewEnvironment, error) {
	query := `SELECT ` + reviewEnvironmentColumns + `
		FROM review_test_environments
		WHERE status IN ('starting', 'running')
		ORDER BY expires_at ASC`
	rows := make([]*domain.ReviewEnvironment, 0)
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to list active review environments: %w", err)
	}
	return rows, nil
}

func (r *PostgresReviewEnvironmentRepository) SetRunning(ctx context.Context, id, containerID, accessURL string, expiresAt time.Time, rxBytes, txBytes int64) error {
	query := `
		UPDATE review_test_environments
		SET container_id = $2,
		    access_url = $3,
		    status = 'running',
		    error_code = '',
		    error_message = '',
		    last_activity_at = NOW(),
		    expires_at = $4,
		    last_rx_bytes = $5,
		    last_tx_bytes = $6,
		    updated_at = NOW()
		WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, id, containerID, accessURL, expiresAt, rxBytes, txBytes); err != nil {
		return fmt.Errorf("failed to mark review environment running: %w", err)
	}
	return nil
}

func (r *PostgresReviewEnvironmentRepository) Touch(ctx context.Context, id string, lastActivityAt, expiresAt time.Time, rxBytes, txBytes int64) error {
	query := `
		UPDATE review_test_environments
		SET last_activity_at = $2,
		    expires_at = $3,
		    last_rx_bytes = $4,
		    last_tx_bytes = $5,
		    updated_at = NOW()
		WHERE id = $1 AND status IN ('starting', 'running')`
	if _, err := r.db.ExecContext(ctx, query, id, lastActivityAt, expiresAt, rxBytes, txBytes); err != nil {
		return fmt.Errorf("failed to touch review environment: %w", err)
	}
	return nil
}

func (r *PostgresReviewEnvironmentRepository) MarkFailed(ctx context.Context, id, errorCode, errorMessage string) error {
	query := `
		UPDATE review_test_environments
		SET container_id = NULL,
		    access_url = '',
		    status = 'failed',
		    error_code = $2,
		    error_message = $3,
		    updated_at = NOW()
		WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, id, errorCode, errorMessage); err != nil {
		return fmt.Errorf("failed to mark review environment failed: %w", err)
	}
	return nil
}

func (r *PostgresReviewEnvironmentRepository) MarkExpired(ctx context.Context, id string) error {
	query := `
		UPDATE review_test_environments
		SET container_id = NULL,
		    access_url = '',
		    status = 'expired',
		    error_code = '',
		    error_message = '',
		    updated_at = NOW()
		WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("failed to mark review environment expired: %w", err)
	}
	return nil
}
