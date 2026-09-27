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

// auditLogSelectColumns proyecta las columnas de audit_logs resolviendo la
// identidad institucional del actor: email real vía LEFT JOIN a users, con
// fallback al propio actor_id cuando no existe fila de usuario.
const auditLogSelectColumns = `
	SELECT a.id, a.tenant_id, a.actor_id,
	       COALESCE(u.email, a.actor_id::text) AS actor_email,
	       a.action, a.resource_type, a.resource_id, a.status_code, a.metadata,
	       COALESCE(a.ip_address::text, '') as ip_address,
	       COALESCE(a.user_agent, '') as user_agent, a.created_at
	FROM audit_logs a
	LEFT JOIN users u ON u.id = a.actor_id
`

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

	query := auditLogSelectColumns + `
		WHERE a.tenant_id = $1
		  AND ($2 = '' OR a.actor_id::text = $2)
		  AND ($3 = '' OR a.action = $3)
		ORDER BY a.created_at DESC
		LIMIT $4 OFFSET $5
	`
	var logs []*domain.AuditLog
	err := r.db.SelectContext(ctx, &logs, query, tenantID, actorID, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list filtered audit logs: %w", err)
	}
	return logs, nil
}

// ListByActorTimeline devuelve la cronología completa de un actor (desc) para
// el drawer de timeline del wireframe AUDIT_LOGS.md. Tope duro de 200 filas.
func (r *AuditLogRepository) ListByActorTimeline(ctx context.Context, tenantID, actorID string, limit int) ([]*domain.AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}

	query := auditLogSelectColumns + `
		WHERE a.tenant_id = $1
		  AND a.actor_id::text = $2
		ORDER BY a.created_at DESC
		LIMIT $3
	`
	var logs []*domain.AuditLog
	err := r.db.SelectContext(ctx, &logs, query, tenantID, actorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list actor timeline: %w", err)
	}
	return logs, nil
}

