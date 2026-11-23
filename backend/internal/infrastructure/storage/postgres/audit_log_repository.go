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

// auditLogFilterWhere predicados compartidos de ListFiltered/CountFiltered:
//   - action soporta categorías semánticas (CREATE/POST, UPDATE/PUT, DELETE)
//     además de prefijo exacto como EMERGENCY_ o verbos HTTP específicos.
//   - search matchea globalmente email/id del actor, acción y recurso.
const auditLogFilterWhere = `
	WHERE a.tenant_id = $1
	  AND ($2 = '' OR a.actor_id::text ILIKE '%' || $2 || '%'
	       OR u.email ILIKE '%' || $2 || '%'
	       OR a.action ILIKE '%' || $2 || '%'
	       OR a.resource_type ILIKE '%' || $2 || '%'
	       OR COALESCE(a.resource_id::text, '') ILIKE '%' || $2 || '%')
	  AND (
	       $3 = ''
	       OR ($3 IN ('CREATE', 'POST') AND (
	            a.action LIKE 'POST %'
	            OR a.action LIKE 'CREATE_%'
	            OR a.action LIKE '%_CREATED'
	            OR a.action LIKE '%_REGISTERED'
	            OR a.action LIKE '%_EMITTED'
	            OR a.action LIKE '%_DUPLICATED'
	            OR a.action LIKE '%_GRANTED'
	       ))
	       OR ($3 IN ('UPDATE', 'PUT') AND (
	            a.action LIKE 'PUT %'
	            OR a.action LIKE 'PATCH %'
	            OR a.action LIKE '%_UPDATED'
	            OR a.action LIKE '%_SET'
	            OR a.action LIKE '%_CLOSED'
	            OR a.action LIKE '%_FROZEN'
	            OR a.action LIKE '%_UNFROZEN'
	            OR a.action LIKE '%_APPLIED'
	            OR a.action LIKE '%_DISMISSED'
	            OR a.action LIKE '%_ROLLBACK'
	            OR a.action LIKE '%_REVIEWED'
	            OR a.action LIKE '%_PROMOTED_TO_MODEL'
	            OR a.action LIKE '%_ENABLED'
	            OR a.action LIKE '%_DISABLED'
	            OR a.action LIKE '%_CHANGED'
	            OR a.action LIKE '%_SELECTED'
	            OR a.action LIKE '%_RECONCILED_%'
	            OR a.action LIKE '%_REACTIVATED'
	            OR a.action LIKE '%_DEACTIVATED'
	            OR a.action = 'IMAGE_VERIFICATION_FORCE_BYPASS'
	       ))
	       OR ($3 = 'DELETE' AND (
	            a.action LIKE 'DELETE %'
	            OR a.action LIKE '%_DELETED'
	            OR a.action LIKE '%_REVOKED'
	            OR a.action LIKE '%_RELEASED'
	            OR a.action LIKE 'EMERGENCY_%'
	       ))
	       OR ($3 NOT IN ('CREATE', 'POST', 'UPDATE', 'PUT', 'DELETE') AND a.action LIKE $3 || '%')
	  )
`

// MatchesActionFilter comprueba en memoria si una acción técnica cumple con la categoría o prefijo solicitado.
func MatchesActionFilter(filter, action string) bool {
	if filter == "" {
		return true
	}
	switch filter {
	case "CREATE", "POST":
		return (len(action) > 5 && action[:5] == "POST ") ||
			(len(action) > 7 && action[:7] == "CREATE_") ||
			hasSuffix(action, "_CREATED") ||
			hasSuffix(action, "_REGISTERED") ||
			hasSuffix(action, "_EMITTED") ||
			hasSuffix(action, "_DUPLICATED") ||
			hasSuffix(action, "_GRANTED")
	case "UPDATE", "PUT":
		return (len(action) > 4 && action[:4] == "PUT ") ||
			(len(action) > 6 && action[:6] == "PATCH ") ||
			hasSuffix(action, "_UPDATED") ||
			hasSuffix(action, "_SET") ||
			hasSuffix(action, "_CLOSED") ||
			hasSuffix(action, "_FROZEN") ||
			hasSuffix(action, "_UNFROZEN") ||
			hasSuffix(action, "_APPLIED") ||
			hasSuffix(action, "_DISMISSED") ||
			hasSuffix(action, "_ROLLBACK") ||
			hasSuffix(action, "_REVIEWED") ||
			hasSuffix(action, "_PROMOTED_TO_MODEL") ||
			hasSuffix(action, "_ENABLED") ||
			hasSuffix(action, "_DISABLED") ||
			hasSuffix(action, "_CHANGED") ||
			hasSuffix(action, "_SELECTED") ||
			contains(action, "_RECONCILED_") ||
			hasSuffix(action, "_REACTIVATED") ||
			hasSuffix(action, "_DEACTIVATED") ||
			action == "IMAGE_VERIFICATION_FORCE_BYPASS"
	case "DELETE":
		return (len(action) > 7 && action[:7] == "DELETE ") ||
			hasSuffix(action, "_DELETED") ||
			hasSuffix(action, "_REVOKED") ||
			hasSuffix(action, "_RELEASED") ||
			(len(action) >= 10 && action[:10] == "EMERGENCY_")
	default:
		return len(action) >= len(filter) && action[:len(filter)] == filter
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (r *AuditLogRepository) ListFiltered(ctx context.Context, tenantID, search, action string, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := auditLogSelectColumns + auditLogFilterWhere + `
		ORDER BY a.created_at DESC
		LIMIT $4 OFFSET $5
	`
	logs := make([]*domain.AuditLog, 0)
	err := r.db.SelectContext(ctx, &logs, query, tenantID, search, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list filtered audit logs: %w", err)
	}
	return logs, nil
}

func (r *AuditLogRepository) CountFiltered(ctx context.Context, tenantID, search, action string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
	` + auditLogFilterWhere
	var total int
	if err := r.db.GetContext(ctx, &total, query, tenantID, search, action); err != nil {
		return 0, fmt.Errorf("failed to count filtered audit logs: %w", err)
	}
	return total, nil
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
	logs := make([]*domain.AuditLog, 0)
	err := r.db.SelectContext(ctx, &logs, query, tenantID, actorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list actor timeline: %w", err)
	}
	return logs, nil
}
