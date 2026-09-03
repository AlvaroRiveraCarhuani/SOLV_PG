package domain

import (
	"time"
)

// AcademicPeriod representa un semestre o periodo académico formal (ADR-029)
type AcademicPeriod struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"tenant_id"`
	Name      string    `db:"name" json:"name"`
	Code      string    `db:"code" json:"code"`
	StartDate time.Time `db:"start_date" json:"start_date"`
	EndDate   time.Time `db:"end_date" json:"end_date"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
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
	Name      string `json:"name" validate:"required"`
	Code      string `json:"code" validate:"required"`
	StartDate string `json:"start_date" validate:"required"`
	EndDate   string `json:"end_date" validate:"required"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

// MaintenanceStatus DTO de estado del modo mantenimiento (ADR-031)
type MaintenanceStatus struct {
	MaintenanceMode   bool       `json:"maintenance_mode"`
	MaintenanceUntil  *time.Time `json:"maintenance_until,omitempty"`
	MaintenanceReason string     `json:"maintenance_reason,omitempty"`
}

// EnableMaintenanceDTO DTO para habilitar mantenimiento
type EnableMaintenanceDTO struct {
	Until  string `json:"until" validate:"required"` // RFC3339 timestamp
	Reason string `json:"reason,omitempty"`
}
