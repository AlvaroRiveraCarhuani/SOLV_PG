package postgres

import (
	"context"
	"fmt"

	"solv-backend/internal/core/domain"
	"github.com/jmoiron/sqlx"
)

type AuditLogRepository struct {
	db *sqlx.DB
}

func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, resource_id, status_code, metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, '')::inet, $9)
		RETURNING id, created_at
	`
	meta := log.Metadata
	if len(meta) == 0 {
		meta = []byte("{}")
	}

	err := r.db.QueryRowContext(
		ctx, query,
		log.TenantID, log.ActorID, log.Action, log.ResourceType, log.ResourceID, log.StatusCode, meta, log.IPAddress, log.UserAgent,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

func (r *AuditLogRepository) ListByTenant(ctx context.Context, tenantID string, limit int) ([]*domain.AuditLog, error) {
	return r.ListFiltered(ctx, tenantID, "", "", limit, 0)
}

func (r *AuditLogRepository) ListFiltered(ctx context.Context, tenantID, actorID, action string, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, tenant_id, actor_id, action, resource_type, resource_id, status_code, metadata, COALESCE(ip_address::text, '') as ip_address, COALESCE(user_agent, '') as user_agent, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		  AND ($2 = '' OR actor_id::text = $2)
		  AND ($3 = '' OR action = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	var logs []*domain.AuditLog
	err := r.db.SelectContext(ctx, &logs, query, tenantID, actorID, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list filtered audit logs: %w", err)
	}
	return logs, nil
}

