package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrUnknownLanguage              = errors.New("lenguaje no soportado por el juez")
	ErrInvalidLanguageProfile       = errors.New("parámetros de perfil de lenguaje inválidos")
	ErrLanguageProfileImageUnpinned = errors.New("la imagen del perfil de lenguaje debe estar fijada por digest sha256")
	ErrLanguageProfileTimeoutRange  = errors.New("el tiempo de ejecución por defecto debe estar entre 100ms y 10000ms")
	ErrLanguageProfileMemoryRange   = errors.New("la memoria por defecto debe estar entre 64MB y 1024MB")
	ErrReasonRequired               = errors.New("el motivo del cambio de perfil es obligatorio para auditoría")
)

// SupportedCanonicalLanguages lista los 6 lenguajes soportados en v1 (D-EJ-04).
var SupportedCanonicalLanguages = map[string]bool{
	"python":     true,
	"javascript": true,
	"cpp":        true,
	"c":          true,
	"csharp":     true,
	"java":       true,
}

// LanguageProfile representa la configuración de ejecución y compilación
// versionada por lenguaje en la tabla language_profiles (migración 00011).
type LanguageProfile struct {
	Language            string    `json:"language" db:"language"`
	DefaultTimeoutMS    int       `json:"default_timeout_ms" db:"default_timeout_ms"`
	DefaultMemoryMB     int       `json:"default_memory_mb" db:"default_memory_mb"`
	Image               string    `json:"image" db:"image"`
	BuildCommand        string    `json:"build_command" db:"build_command"`
	BuildTimeoutMS      int       `json:"build_timeout_ms" db:"build_timeout_ms"`
	BuildMemoryMB       int       `json:"build_memory_mb" db:"build_memory_mb"`
	P95WindowDays       int       `json:"p95_window_days" db:"p95_window_days"`
	CheckerSidecarImage string    `json:"checker_sidecar_image" db:"checker_sidecar_image"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// LanguageProfileAudit registra las modificaciones realizadas por administración
// sobre un perfil de lenguaje (tabla language_profile_audits, migración 00011).
type LanguageProfileAudit struct {
	ID        string          `json:"id" db:"id"`
	Language  string          `json:"language" db:"language"`
	Author    string          `json:"author" db:"author"`
	OldValues json.RawMessage `json:"old_values" db:"old_values"`
	NewValues json.RawMessage `json:"new_values" db:"new_values"`
	Reason    string          `json:"reason" db:"reason"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// UpdateLanguageProfileDTO encapsula los campos modificables de un perfil por el administrador.
type UpdateLanguageProfileDTO struct {
	DefaultTimeoutMS    *int    `json:"default_timeout_ms,omitempty"`
	DefaultMemoryMB     *int    `json:"default_memory_mb,omitempty"`
	Image               *string `json:"image,omitempty"`
	BuildCommand        *string `json:"build_command,omitempty"`
	BuildTimeoutMS      *int    `json:"build_timeout_ms,omitempty"`
	BuildMemoryMB       *int    `json:"build_memory_mb,omitempty"`
	P95WindowDays       *int    `json:"p95_window_days,omitempty"`
	CheckerSidecarImage *string `json:"checker_sidecar_image,omitempty"`
	Reason              string  `json:"reason"`
}
