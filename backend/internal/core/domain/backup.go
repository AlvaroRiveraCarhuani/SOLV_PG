package domain

import (
	"time"
)

const (
	StorageTierLocal  = "local"
	StorageTierRemote = "remote"
	StorageTierBoth   = "both"

	BackupStatusInProgress = "in_progress"
	BackupStatusSuccess    = "success"
	BackupStatusFailed     = "failed"
)

// BackupConfig define las políticas de retención y sincronización (ADR-035)
type BackupConfig struct {
	ID                       string    `db:"id" json:"id"`
	TenantID                 string    `db:"tenant_id" json:"tenant_id"`
	LocalFrequencyHours      int       `db:"local_frequency_hours" json:"local_frequency_hours"`
	LocalRetentionDays       int       `db:"local_retention_days" json:"local_retention_days"`
	RemoteEnabled            bool      `db:"remote_enabled" json:"remote_enabled"`
	RemoteProvider           string    `db:"remote_provider" json:"remote_provider"`
	RemoteBucketName         string    `db:"remote_bucket_name" json:"remote_bucket_name"`
	RemoteEndpoint           string    `db:"remote_endpoint" json:"remote_endpoint"`
	RemoteAccessKey          string    `db:"remote_access_key" json:"remote_access_key,omitempty"`
	RemoteSecretKeyEncrypted string    `db:"remote_secret_key_encrypted" json:"-"`
	RemoteRetentionDays      int       `db:"remote_retention_days" json:"remote_retention_days"`
	IsActive                 bool      `db:"is_active" json:"is_active"`
	UpdatedAt                time.Time `db:"updated_at" json:"updated_at"`
}

// UpdateBackupConfigDTO DTO para modificar la configuración de backups
type UpdateBackupConfigDTO struct {
	LocalFrequencyHours int     `json:"local_frequency_hours"`
	LocalRetentionDays  int     `json:"local_retention_days"`
	RemoteEnabled       *bool   `json:"remote_enabled,omitempty"`
	RemoteProvider      *string `json:"remote_provider,omitempty"`
	RemoteBucketName    *string `json:"remote_bucket_name,omitempty"`
	RemoteEndpoint      *string `json:"remote_endpoint,omitempty"`
	RemoteAccessKey     *string `json:"remote_access_key,omitempty"`
	RemoteSecretKey     *string `json:"remote_secret_key,omitempty"`
	RemoteRetentionDays *int    `json:"remote_retention_days,omitempty"`
	IsActive            *bool   `json:"is_active,omitempty"`
}

// BackupExecution registra el historial y estado de cada respaldo
type BackupExecution struct {
	ID             string     `db:"id" json:"id"`
	TenantID       string     `db:"tenant_id" json:"tenant_id"`
	FileName       string     `db:"file_name" json:"file_name"`
	FileSizeBytes  int64      `db:"file_size_bytes" json:"file_size_bytes"`
	SHA256Checksum string     `db:"sha256_checksum" json:"sha256_checksum"`
	StorageTier    string     `db:"storage_tier" json:"storage_tier"`
	Status         string     `db:"status" json:"status"`
	ErrorMessage   string     `db:"error_message" json:"error_message,omitempty"`
	StartedAt      time.Time  `db:"started_at" json:"started_at"`
	CompletedAt    *time.Time `db:"completed_at" json:"completed_at,omitempty"`
}

// VerifyBackupResponse resultado de la verificación de integridad
type VerifyBackupResponse struct {
	ExecutionID      string `json:"execution_id"`
	FileName         string `json:"file_name"`
	DatabaseChecksum string `json:"database_checksum"`
	ComputedChecksum string `json:"computed_checksum"`
	IsValid          bool   `json:"is_valid"`
	Message          string `json:"message"`
}
