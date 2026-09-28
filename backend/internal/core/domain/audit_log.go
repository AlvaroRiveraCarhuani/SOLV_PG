package domain

import (
	"context"
	"time"
)

type AuditLog struct {
	ID       string `db:"id" json:"id"`
	TenantID string `db:"tenant_id" json:"tenant_id"`
	ActorID  string `db:"actor_id" json:"actor_id"`
	// ActorEmail resuelve la identidad institucional del actor (JOIN a users).
	// Si no hay fila de usuario, contiene el propio actor_id para trazabilidad.
	ActorEmail   string    `db:"actor_email" json:"actor_email"`
	Action       string    `db:"action" json:"action"`
	ResourceType string    `db:"resource_type" json:"resource_type"`
	ResourceID   *string   `db:"resource_id" json:"resource_id,omitempty"`
	StatusCode   int       `db:"status_code" json:"status_code"`
	Metadata     []byte    `db:"metadata" json:"metadata,omitempty"`
	IPAddress    string    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent    string    `db:"user_agent" json:"user_agent,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	ListByTenant(ctx context.Context, tenantID string, limit int) ([]*AuditLog, error)
	ListFiltered(ctx context.Context, tenantID, search, action string, limit, offset int) ([]*AuditLog, error)
	// CountFiltered cuenta las filas de los mismos predicados de ListFiltered
	// para paginación real (el frontend no debe estimar el total).
	CountFiltered(ctx context.Context, tenantID, search, action string) (int, error)
	// ListByActorTimeline devuelve la cronología completa (desc) de un actor
	// dentro del tenant, para el drawer del wireframe AUDIT_LOGS.md.
	ListByActorTimeline(ctx context.Context, tenantID, actorID string, limit int) ([]*AuditLog, error)
}
