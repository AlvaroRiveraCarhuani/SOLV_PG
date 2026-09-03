package domain

import (
	"encoding/json"
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
	ID              string          `db:"id" json:"id"`
	TenantID        string          `db:"tenant_id" json:"tenant_id"`
	RecipientUserID string          `db:"recipient_user_id" json:"recipient_user_id"`
	Channel         string          `db:"channel" json:"channel"`
	Severity        string          `db:"severity" json:"severity"`
	Title           string          `db:"title" json:"title"`
	Message         string          `db:"message" json:"message"`
	EventType       string     `db:"event_type" json:"event_type"`
	Metadata        []byte     `db:"metadata" json:"metadata,omitempty"`
	IsRead          bool       `db:"is_read" json:"is_read"`
	ReadAt          *time.Time `db:"read_at" json:"read_at,omitempty"`
	EmailSentAt     *time.Time `db:"email_sent_at" json:"email_sent_at,omitempty"`
	OccurrenceCount int        `db:"occurrence_count" json:"occurrence_count"`
	Link            string     `db:"link" json:"link,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
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
}

// UnreadCountResponse DTO para el contador de la campana UI
type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

// MarkAllReadResult DTO con la cantidad de notificaciones actualizadas
type MarkAllReadResult struct {
	MarkedCount int64 `json:"marked_count"`
}
