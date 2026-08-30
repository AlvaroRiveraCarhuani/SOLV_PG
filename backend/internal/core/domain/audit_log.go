package domain

import (
	"context"
	"time"
)

type AuditLog struct {
	ID           string    `db:"id" json:"id"`
	TenantID     string    `db:"tenant_id" json:"tenant_id"`
	ActorID      string    `db:"actor_id" json:"actor_id"`
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
	ListFiltered(ctx context.Context, tenantID, actorID, action string, limit, offset int) ([]*AuditLog, error)
}
