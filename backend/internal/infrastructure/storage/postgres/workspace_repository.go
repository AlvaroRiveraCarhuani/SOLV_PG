package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresWorkspaceRepository struct {
	db *sqlx.DB
}

func NewPostgresWorkspaceRepository(db *sqlx.DB) domain.WorkspaceRepository {
	return &PostgresWorkspaceRepository{db: db}
}

func (r *PostgresWorkspaceRepository) GetByStudentAndSubject(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, created_at, updated_at
		FROM workspaces
		WHERE student_id = $1 AND subject_id = $2 AND tenant_id = $3
	`
	var ws domain.WorkspaceInstance
	err := r.db.GetContext(ctx, &ws, query, studentID, subjectID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace for student %s and subject %s: %w", studentID, subjectID, err)
	}
	return &ws, nil
}

func (r *PostgresWorkspaceRepository) GetByID(ctx context.Context, id string) (*domain.WorkspaceInstance, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, created_at, updated_at
		FROM workspaces
		WHERE id = $1 AND tenant_id = $2
	`
	var ws domain.WorkspaceInstance
	err := r.db.GetContext(ctx, &ws, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace by id %s: %w", id, err)
	}
	return &ws, nil
}

func (r *PostgresWorkspaceRepository) Create(ctx context.Context, workspace *domain.WorkspaceInstance) error {
	tenantID := domain.GetTenantID(ctx)
	if workspace.TenantID == "" {
		workspace.TenantID = tenantID
	}
	if workspace.TenantID == "" {
		workspace.TenantID = domain.DefaultTenantID
	}
	if workspace.SubjectID == "" {
		workspace.SubjectID = "00000000-0000-0000-0000-000000000001"
	}
	if workspace.MemoryLimitMB <= 0 {
		workspace.MemoryLimitMB = domain.DefaultBaseMemoryMB
	}
	if workspace.Type == "" {
		workspace.Type = domain.WorkspaceTypeIDEPersistente
	}
	if workspace.LastHeartbeatAt.IsZero() {
		workspace.LastHeartbeatAt = time.Now()
	}
	query := `
		INSERT INTO workspaces (id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, semgrep_audit, tenant_id, created_at, updated_at)
		VALUES (:id, :student_id, :subject_id, :type, :container_id, :status, :access_url, :memory_limit_mb, :last_heartbeat_at, :last_oom_killed_at, :oom_strike_count, :semgrep_audit, :tenant_id, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, workspace)
	if err != nil {
		return fmt.Errorf("failed to create workspace record via sqlx: %w", err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET container_id = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, containerID, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update container_id for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET status = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, status, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update status for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) UpdateMemoryLimit(ctx context.Context, id string, memoryMB int64) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET memory_limit_mb = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, memoryMB, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update memory_limit_mb for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) RecordHeartbeat(ctx context.Context, id string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET last_heartbeat_at = $2, updated_at = $2
		WHERE id = $1 AND tenant_id = $3
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, now, tenantID)
	if err != nil {
		return fmt.Errorf("failed to record heartbeat for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) IncrementOOMStrike(ctx context.Context, id string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET oom_strike_count = oom_strike_count + 1, last_oom_killed_at = $2, status = $3, updated_at = $2
		WHERE id = $1 AND tenant_id = $4
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, now, domain.WorkspaceStatusOOMKilled, tenantID)
	if err != nil {
		return fmt.Errorf("failed to increment oom strike for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) ResetOOMStrikes(ctx context.Context, id string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET oom_strike_count = 0, updated_at = $2
		WHERE id = $1 AND tenant_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, id, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to reset oom strikes for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) GetActiveWorkspaces(ctx context.Context) ([]*domain.WorkspaceInstance, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, created_at, updated_at
		FROM workspaces
		WHERE status IN ('running', 'pending') AND tenant_id = $1
	`
	var workspaces []*domain.WorkspaceInstance
	err := r.db.SelectContext(ctx, &workspaces, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active workspaces: %w", err)
	}
	return workspaces, nil
}

func (r *PostgresWorkspaceRepository) GetAllRunningWorkspaces(ctx context.Context) ([]*domain.WorkspaceInstance, error) {
	// A veces las tareas de limpieza global corren sin tenant_id en background.
	// Si no hay tenant_id, consultamos todos.
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	var query string
	var args []interface{}
	if tenantID == "" {
		query = `
			SELECT id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, created_at, updated_at
			FROM workspaces
			WHERE status = 'running'
		`
	} else {
		query = `
			SELECT id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, created_at, updated_at
			FROM workspaces
			WHERE status = 'running' AND tenant_id = $1
		`
		args = append(args, tenantID)
	}
	var workspaces []*domain.WorkspaceInstance
	err := r.db.SelectContext(ctx, &workspaces, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get running workspaces: %w", err)
	}
	return workspaces, nil
}

func (r *PostgresWorkspaceRepository) GetByType(ctx context.Context, workspaceType string) ([]*domain.WorkspaceInstance, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, created_at, updated_at
		FROM workspaces
		WHERE type = $1 AND tenant_id = $2
	`
	var workspaces []*domain.WorkspaceInstance
	err := r.db.SelectContext(ctx, &workspaces, query, workspaceType, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspaces by type %s: %w", workspaceType, err)
	}
	return workspaces, nil
}

func (r *PostgresWorkspaceRepository) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE workspaces
		SET semgrep_audit = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, auditJSON, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to save semgrep audit for workspace %s: %w", id, err)
	}
	return nil
}
