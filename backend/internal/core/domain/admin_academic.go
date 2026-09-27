package domain

import (
	"time"
)

// AcademicPeriod representa un semestre o periodo académico formal (ADR-029).
// IsArchived es el congelamiento institucional formal e irreversible: solo se
// establece vía ArchivePeriod (confirmación fuerte) o por expiración automática.
type AcademicPeriod struct {
	ID         string     `db:"id" json:"id"`
	TenantID   string     `db:"tenant_id" json:"tenant_id"`
	Name       string     `db:"name" json:"name"`
	Code       string     `db:"code" json:"code"`
	StartDate  time.Time  `db:"start_date" json:"start_date"`
	EndDate    time.Time  `db:"end_date" json:"end_date"`
	IsActive   bool       `db:"is_active" json:"is_active"`
	IsArchived bool       `db:"is_archived" json:"is_archived"`
	ArchivedAt *time.Time `db:"archived_at" json:"archived_at,omitempty"`
	ArchivedBy *string    `db:"archived_by" json:"archived_by,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// ArchiveAcademicPeriodDTO DTO para el archivo formal con confirmación fuerte (ADR-029)
type ArchiveAcademicPeriodDTO struct {
	ConfirmationCode string `json:"confirmation_code" validate:"required"`
}

// CreateAcademicPeriodDTO DTO para crear periodos
type CreateAcademicPeriodDTO struct {
	Name      string `json:"name" validate:"required"`
	Code      string `json:"code" validate:"required"`
	StartDate string `json:"start_date" validate:"required"` // Formato YYYY-MM-DD o RFC3339
	EndDate   string `json:"end_date" validate:"required"`   // Formato YYYY-MM-DD o RFC3339
	IsActive  *bool  `json:"is_active,omitempty"`
}

// UpdateAcademicPeriodDTO DTO para actualizar periodos
type UpdateAcademicPeriodDTO struct {
	Name      string `json:"name,omitempty"`
	Code      string `json:"code,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

// MaintenanceStatus DTO de estado del modo mantenimiento (ADR-031)
type MaintenanceStatus struct {
	MaintenanceMode   bool       `json:"maintenance_mode"`
	MaintenanceUntil  *time.Time `json:"maintenance_until,omitempty"`
	MaintenanceReason string     `json:"maintenance_reason,omitempty"`
}

// EnableMaintenanceDTO DTO para habilitar mantenimiento.
// Until is optional RFC3339 ("" = indefinite); Reason needs >= 10 chars;
// ConfirmPhrase must match MANTENIMIENTO exactly (type-to-confirm).
type EnableMaintenanceDTO struct {
	Until         string `json:"until"`
	Reason        string `json:"reason,omitempty"`
	ConfirmPhrase string `json:"confirm_phrase"`
}
