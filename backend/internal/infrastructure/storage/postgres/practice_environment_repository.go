package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresPracticeEnvironmentRepository struct {
	db *sqlx.DB
}

func NewPostgresPracticeEnvironmentRepository(db *sqlx.DB) domain.PracticeEnvironmentRepository {
	return &PostgresPracticeEnvironmentRepository{db: db}
}

// GetByStudentSubjectExercise devuelve el entorno de una práctica concreta de
// la persona (alumno o docente). Es una capacidad opcional del repositorio: el
// ciclo de vida la usa solo para entornos acotados a un ejercicio, sin
// ensanchar la interfaz base.
func (r *PostgresPracticeEnvironmentRepository) GetByStudentSubjectExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.PracticeEnvironment, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at
		FROM environments
		WHERE student_id = $1 AND subject_id = $2 AND exercise_id = $3 AND tenant_id = $4
		LIMIT 1
	`
	var ws domain.PracticeEnvironment
	err := r.db.GetContext(ctx, &ws, query, studentID, subjectID, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get practice environment %s for %s/%s: %w", exerciseID, studentID, subjectID, err)
	}
	return &ws, nil
}

func (r *PostgresPracticeEnvironmentRepository) GetByID(ctx context.Context, id string) (*domain.PracticeEnvironment, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at
		FROM environments
		WHERE id = $1 AND tenant_id = $2
	`
	var ws domain.PracticeEnvironment
	err := r.db.GetContext(ctx, &ws, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment by id %s: %w", id, err)
	}
	return &ws, nil
}

func (r *PostgresPracticeEnvironmentRepository) Create(ctx context.Context, environment *domain.PracticeEnvironment) error {
	tenantID := domain.GetTenantID(ctx)
	if environment.TenantID == "" {
		environment.TenantID = tenantID
	}
	if environment.TenantID == "" {
		environment.TenantID = domain.DefaultTenantID
	}
	if environment.SubjectID == "" {
		environment.SubjectID = domain.DefaultTenantID
	}
	if environment.MemoryLimitMB <= 0 {
		environment.MemoryLimitMB = domain.DefaultBaseMemoryMB
	}
	if environment.Type == "" {
		environment.Type = domain.PracticeEnvironmentTypeIDEPersistente
	}
	if environment.LastHeartbeatAt.IsZero() {
		environment.LastHeartbeatAt = time.Now()
	}
	query := `
		INSERT INTO environments (id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at)
		VALUES (:id, :student_id, :subject_id, :template_id, :type, :container_id, :status, :access_url, :memory_limit_mb, :last_heartbeat_at, :last_oom_killed_at, :oom_strike_count, :semgrep_audit, :tenant_id, :exercise_id, :frozen_at, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, environment)
	if err != nil {
		return fmt.Errorf("failed to create environment record via sqlx: %w", err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET container_id = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, containerID, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update container_id for environment %s: %w", id, err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET status = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, status, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update status for environment %s: %w", id, err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) UpdateMemoryLimit(ctx context.Context, id string, memoryMB int64) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET memory_limit_mb = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, memoryMB, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update memory_limit_mb for environment %s: %w", id, err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) RecordHeartbeat(ctx context.Context, id string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET last_heartbeat_at = $2, updated_at = $2
		WHERE id = $1 AND tenant_id = $3
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, now, tenantID)
	if err != nil {
		return fmt.Errorf("failed to record heartbeat for environment %s: %w", id, err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) IncrementOOMStrike(ctx context.Context, id string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET oom_strike_count = oom_strike_count + 1, last_oom_killed_at = $2, status = $3, updated_at = $2
		WHERE id = $1 AND tenant_id = $4
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, now, domain.PracticeEnvironmentStatusOOMKilled, tenantID)
	if err != nil {
		return fmt.Errorf("failed to increment oom strike for environment %s: %w", id, err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) ResetOOMStrikes(ctx context.Context, id string) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET oom_strike_count = 0, updated_at = $2
		WHERE id = $1 AND tenant_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, id, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to reset oom strikes for environment %s: %w", id, err)
	}
	return nil
}

func (r *PostgresPracticeEnvironmentRepository) GetActivePracticeEnvironments(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at
		FROM environments
		WHERE status IN ('running', 'pending', 'unknown') AND tenant_id = $1
	`
	var environments []*domain.PracticeEnvironment
	err := r.db.SelectContext(ctx, &environments, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active environments: %w", err)
	}
	return environments, nil
}

func (r *PostgresPracticeEnvironmentRepository) CountPracticeEnvironmentsByStatus(ctx context.Context, tenantID string) (map[string]int, error) {
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	const query = `
		SELECT status, COUNT(*) AS total
		FROM environments
		WHERE tenant_id = $1
		GROUP BY status
	`

	var rows []struct {
		Status string `db:"status"`
		Total  int    `db:"total"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed to count environments by status: %w", err)
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Total
	}
	return counts, nil
}

func (r *PostgresPracticeEnvironmentRepository) GetAllRunningPracticeEnvironments(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
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
			SELECT id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at
			FROM environments
			WHERE status IN ('running', 'frozen')
		`
	} else {
		query = `
			SELECT id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at
			FROM environments
			WHERE status IN ('running', 'frozen') AND tenant_id = $1
		`
		args = append(args, tenantID)
	}
	var environments []*domain.PracticeEnvironment
	err := r.db.SelectContext(ctx, &environments, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get running environments: %w", err)
	}
	return environments, nil
}

func (r *PostgresPracticeEnvironmentRepository) GetByType(ctx context.Context, environmentType string) ([]*domain.PracticeEnvironment, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT id, student_id, subject_id, template_id, type, container_id, status, access_url, memory_limit_mb, last_heartbeat_at, last_oom_killed_at, oom_strike_count, COALESCE(semgrep_audit, '{}'::jsonb) AS semgrep_audit, tenant_id, exercise_id, frozen_at, created_at, updated_at
		FROM environments
		WHERE type = $1 AND tenant_id = $2
	`
	var environments []*domain.PracticeEnvironment
	err := r.db.SelectContext(ctx, &environments, query, environmentType, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environments by type %s: %w", environmentType, err)
	}
	return environments, nil
}

func (r *PostgresPracticeEnvironmentRepository) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		UPDATE environments
		SET semgrep_audit = $2, updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, id, auditJSON, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to save semgrep audit for environment %s: %w", id, err)
	}
	return nil
}
