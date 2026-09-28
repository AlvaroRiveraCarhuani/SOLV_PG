package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresBackupRepository struct {
	db *sqlx.DB
}

func NewPostgresBackupRepository(db *sqlx.DB) *PostgresBackupRepository {
	return &PostgresBackupRepository{db: db}
}

func (r *PostgresBackupRepository) GetConfig(ctx context.Context, tenantID string) (*domain.BackupConfig, error) {
	query := `
		SELECT 
			id, tenant_id, local_frequency_hours, local_retention_days,
			remote_enabled, remote_provider, remote_bucket_name, remote_endpoint,
			remote_access_key, remote_secret_key_encrypted, remote_retention_days,
			is_active, updated_at
		FROM backup_configs
		WHERE tenant_id = $1
	`
	var cfg domain.BackupConfig
	err := r.db.GetContext(ctx, &cfg, query, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Retornar configuración por defecto si aún no se ha persistido
			return &domain.BackupConfig{
				ID:                  uuid.NewString(),
				TenantID:            tenantID,
				LocalFrequencyHours: 6,
				LocalRetentionDays:  7,
				RemoteEnabled:       false,
				RemoteProvider:      "backblaze_b2",
				RemoteRetentionDays: 30,
				IsActive:            true,
				UpdatedAt:           time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("error querying backup config: %w", err)
	}
	return &cfg, nil
}

func (r *PostgresBackupRepository) UpsertConfig(ctx context.Context, config *domain.BackupConfig) error {
	query := `
		INSERT INTO backup_configs (
			id, tenant_id, local_frequency_hours, local_retention_days,
			remote_enabled, remote_provider, remote_bucket_name, remote_endpoint,
			remote_access_key, remote_secret_key_encrypted, remote_retention_days,
			is_active, updated_at
		) VALUES (
			:id, :tenant_id, :local_frequency_hours, :local_retention_days,
			:remote_enabled, :remote_provider, :remote_bucket_name, :remote_endpoint,
			:remote_access_key, :remote_secret_key_encrypted, :remote_retention_days,
			:is_active, :updated_at
		)
		ON CONFLICT (tenant_id) DO UPDATE SET
			local_frequency_hours = EXCLUDED.local_frequency_hours,
			local_retention_days = EXCLUDED.local_retention_days,
			remote_enabled = EXCLUDED.remote_enabled,
			remote_provider = EXCLUDED.remote_provider,
			remote_bucket_name = EXCLUDED.remote_bucket_name,
			remote_endpoint = EXCLUDED.remote_endpoint,
			remote_access_key = EXCLUDED.remote_access_key,
			remote_secret_key_encrypted = EXCLUDED.remote_secret_key_encrypted,
			remote_retention_days = EXCLUDED.remote_retention_days,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.NamedExecContext(ctx, query, config)
	if err != nil {
		return fmt.Errorf("error upserting backup config: %w", err)
	}
	return nil
}

func (r *PostgresBackupRepository) CreateExecution(ctx context.Context, execution *domain.BackupExecution) error {
	query := `
		INSERT INTO backup_executions (
			id, tenant_id, file_name, file_size_bytes, sha256_checksum,
			storage_tier, status, error_message, started_at, completed_at,
			last_verify_ok, last_verify_at
		) VALUES (
			:id, :tenant_id, :file_name, :file_size_bytes, :sha256_checksum,
			:storage_tier, :status, :error_message, :started_at, :completed_at,
			:last_verify_ok, :last_verify_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		return fmt.Errorf("error creating backup execution: %w", err)
	}
	return nil
}

func (r *PostgresBackupRepository) UpdateExecution(ctx context.Context, execution *domain.BackupExecution) error {
	query := `
		UPDATE backup_executions
		SET 
			file_size_bytes = :file_size_bytes,
			sha256_checksum = :sha256_checksum,
			status = :status,
			error_message = :error_message,
			completed_at = :completed_at,
			last_verify_ok = :last_verify_ok,
			last_verify_at = :last_verify_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		return fmt.Errorf("error updating backup execution: %w", err)
	}
	return nil
}

func (r *PostgresBackupRepository) GetExecutionByID(ctx context.Context, tenantID, id string) (*domain.BackupExecution, error) {
	query := `
		SELECT 
			id, tenant_id, file_name, file_size_bytes, sha256_checksum,
			storage_tier, status, error_message, started_at, completed_at,
			last_verify_ok, last_verify_at
		FROM backup_executions
		WHERE tenant_id = $1 AND id = $2
	`
	var exec domain.BackupExecution
	err := r.db.GetContext(ctx, &exec, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("error getting backup execution: %w", err)
	}
	return &exec, nil
}

func (r *PostgresBackupRepository) ListExecutions(ctx context.Context, tenantID string, limit, offset int) ([]*domain.BackupExecution, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	countQuery := `SELECT COUNT(*) FROM backup_executions WHERE tenant_id = $1`
	var total int64
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, fmt.Errorf("error counting backup executions: %w", err)
	}

	selectQuery := `
		SELECT 
			id, tenant_id, file_name, file_size_bytes, sha256_checksum,
			storage_tier, status, error_message, started_at, completed_at,
			last_verify_ok, last_verify_at
		FROM backup_executions
		WHERE tenant_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`
	var list []*domain.BackupExecution
	if err := r.db.SelectContext(ctx, &list, selectQuery, tenantID, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("error querying backup executions: %w", err)
	}
	if list == nil {
		list = []*domain.BackupExecution{}
	}

	return list, total, nil
}

func (r *PostgresBackupRepository) GetExpiredExecutions(ctx context.Context, tenantID string, retentionDays int) ([]*domain.BackupExecution, error) {
	if retentionDays <= 0 {
		retentionDays = 7
	}

	query := `
		SELECT 
			id, tenant_id, file_name, file_size_bytes, sha256_checksum,
			storage_tier, status, error_message, started_at, completed_at,
			last_verify_ok, last_verify_at
		FROM backup_executions
		WHERE tenant_id = $1 AND status = 'success' AND started_at < NOW() - ($2 || ' days')::INTERVAL
	`
	var list []*domain.BackupExecution
	if err := r.db.SelectContext(ctx, &list, query, tenantID, retentionDays); err != nil {
		return nil, fmt.Errorf("error querying expired backup executions: %w", err)
	}
	if list == nil {
		list = []*domain.BackupExecution{}
	}
	return list, nil
}

func (r *PostgresBackupRepository) DeleteExecution(ctx context.Context, id string) error {
	query := `DELETE FROM backup_executions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UpdateExecutionVerifyColumns persists ONLY last_verify_ok/last_verify_at.
// Never touches status, checksum, file_size, or other execution fields.
func (r *PostgresBackupRepository) UpdateExecutionVerifyColumns(ctx context.Context, execution *domain.BackupExecution) error {
	query := `
		UPDATE backup_executions
		SET last_verify_ok = :last_verify_ok, last_verify_at = :last_verify_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		return fmt.Errorf("error updating backup verify columns: %w", err)
	}
	return nil
}
