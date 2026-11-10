package domain

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// Constantes de canal y severidad para notificaciones (ADR-034)
const (
	NotificationChannelInApp = "in_app"
	NotificationChannelEmail = "email"
	NotificationChannelBoth  = "both"

	NotificationSeverityInfo     = "info"
	NotificationSeverityWarning  = "warning"
	NotificationSeverityError    = "error"
	NotificationSeverityCritical = "critical"
)

// Notification representa el modelo de datos de una notificación proactiva (ADR-034)
type Notification struct {
	ID              string     `db:"id" json:"id"`
	TenantID        string     `db:"tenant_id" json:"tenant_id"`
	RecipientUserID string     `db:"recipient_user_id" json:"recipient_user_id"`
	Channel         string     `db:"channel" json:"channel"`
	Severity        string     `db:"severity" json:"severity"`
	Title           string     `db:"title" json:"title"`
	Message         string     `db:"message" json:"message"`
	EventType       string     `db:"event_type" json:"event_type"`
	Metadata        []byte     `db:"metadata" json:"metadata,omitempty"`
	IsRead          bool       `db:"is_read" json:"is_read"`
	ReadAt          *time.Time `db:"read_at" json:"read_at,omitempty"`
	EmailSentAt     *time.Time `db:"email_sent_at" json:"email_sent_at,omitempty"`
	OccurrenceCount int        `db:"occurrence_count" json:"occurrence_count"`
	Link            string     `db:"link" json:"link,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	// Consolidate no se persiste: indica al servicio que agrupe ocurrencias
	// repetidas dentro de la ventana anti-fatiga.
	Consolidate bool `db:"-" json:"-"`
}

// CreateNotificationDTO DTO para emitir una notificación
type CreateNotificationDTO struct {
	RecipientUserID string          `json:"recipient_user_id" validate:"required"`
	Title           string          `json:"title" validate:"required"`
	Message         string          `json:"message" validate:"required"`
	Severity        string          `json:"severity,omitempty"`
	Channel         string          `json:"channel,omitempty"`
	EventType       string          `json:"event_type,omitempty"`
	Link            string          `json:"link,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	// Consolidate agrupa ocurrencias del mismo evento en un solo aviso dentro
	// de la ventana anti-fatiga. Se reserva para eventos de infraestructura: dos
	// calificaciones distintas nunca deben fundirse en una.
	Consolidate bool `json:"-"`
}

// NotificationListFilter acota el listado del usuario autenticado.
type NotificationListFilter struct {
	UnreadOnly bool
	Severity   string
	EventType  string
}

// NotificationConsolidationWindow es la ventana anti-fatiga de ADR-034.
const NotificationConsolidationWindow = 30 * time.Minute

var notificationEventTypePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// IsValidNotificationSeverity valida el catálogo de severidades.
func IsValidNotificationSeverity(severity string) bool {
	switch severity {
	case NotificationSeverityInfo, NotificationSeverityWarning, NotificationSeverityError, NotificationSeverityCritical:
		return true
	default:
		return false
	}
}

// IsValidNotificationEventType acota los identificadores que el frontend usa
// para agrupar y para resolver la etiqueta visible.
func IsValidNotificationEventType(eventType string) bool {
	return notificationEventTypePattern.MatchString(eventType)
}

// SanitizeNotificationLink deja pasar solo rutas internas relativas. Un aviso
// nunca debe convertirse en una redirección abierta ni en un enlace externo.
func SanitizeNotificationLink(link string) string {
	trimmed := strings.TrimSpace(link)
	if trimmed == "" || len(trimmed) > 255 {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return ""
	}
	if strings.Contains(trimmed, "\\") {
		return ""
	}
	for _, r := range trimmed {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == ".." || segment == "." {
			return ""
		}
	}
	if strings.Contains(trimmed, "://") {
		return ""
	}
	return trimmed
}

// UnreadCountResponse DTO para el contador de la campana UI
type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

// MarkAllReadResult DTO con la cantidad de notificaciones actualizadas
type MarkAllReadResult struct {
	MarkedCount int64 `json:"marked_count"`
}
