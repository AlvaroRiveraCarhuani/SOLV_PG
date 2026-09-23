package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTemplateNameConflict = errors.New("template with this name already exists")
	ErrCategoryInUse        = errors.New("category is in use and cannot be deleted")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrTemplateNotApproved  = errors.New("only approved templates can be promoted to models")
)

// ServiceRequirement requerimiento de un servicio satélite complementario (BD, cache, message broker, etc.)
type ServiceRequirement struct {
	Category string `json:"category"` // "database", "cache", "message_broker", "storage", etc.
	Engine   string `json:"engine"`   // "postgres", "mysql", "mongodb", "redis", etc.
	Version  string `json:"version,omitempty"`
}

// ServicesConfig configuración agnóstica y extensible de servicios satélite requeridos
type ServicesConfig struct {
	Services []ServiceRequirement `json:"services"`
}

func (sc ServicesConfig) Value() (driver.Value, error) {
	if sc.Services == nil {
		sc.Services = []ServiceRequirement{}
	}
	return json.Marshal(sc)
}

func (sc *ServicesConfig) Scan(value interface{}) error {
	if value == nil {
		*sc = ServicesConfig{Services: []ServiceRequirement{}}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ServicesConfig: expected []byte, got %T", value)
	}
	return json.Unmarshal(b, sc)
}

// TemplateResourceProfile límites de memoria MQoS cgroups v2 para la plantilla
type TemplateResourceProfile struct {
	MinMB  int `json:"min_mb"`
	HighMB int `json:"high_mb"`
	MaxMB  int `json:"max_mb"`
}

func (rp TemplateResourceProfile) Value() (driver.Value, error) {
	return json.Marshal(rp)
}

func (rp *TemplateResourceProfile) Scan(value interface{}) error {
	if value == nil {
		*rp = TemplateResourceProfile{MinMB: 256, HighMB: 768, MaxMB: 1024}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan TemplateResourceProfile: expected []byte, got %T", value)
	}
	return json.Unmarshal(b, rp)
}

// DeriveResourceProfile calcula los límites de memoria cgroups v2 a partir de base_ram_mb:
// min_mb = base / 2, high_mb = base * 1.5, max_mb = base * 2
func DeriveResourceProfile(baseRamMB int) TemplateResourceProfile {
	if baseRamMB <= 0 {
		baseRamMB = 512
	}
	return TemplateResourceProfile{
		MinMB:  baseRamMB / 2,
		HighMB: (baseRamMB * 3) / 2,
		MaxMB:  baseRamMB * 2,
	}
}

// ReassignCourseDTO DTO para reasignar la titularidad de una materia (ADR-036)
type ReassignCourseDTO struct {
	NewTeacherID string `json:"new_teacher_id" validate:"required"`
	Reason       string `json:"reason,omitempty"`
}

// AdminStudentDirectoryItem DTO para el listado institucional de estudiantes (ADR-033)
type AdminStudentDirectoryItem struct {
	ID                    string     `db:"id" json:"id"`
	FirstName             string     `db:"first_name" json:"first_name"`
	LastName              string     `db:"last_name" json:"last_name"`
	Email                 string     `db:"email" json:"email"`
	Role                  string     `db:"role" json:"role"`
	Status                string     `db:"status" json:"status"`
	SuspensionReason      *string    `db:"suspension_reason" json:"suspension_reason,omitempty"`
	AcademicStatus        string     `db:"academic_status" json:"academic_status"`
	EnrolledCoursesCount  int        `db:"enrolled_courses_count" json:"enrolled_courses_count"`
	ActiveWorkspacesCount int        `db:"active_workspaces_count" json:"active_workspaces_count"`
	OOMStrikeCount        int        `db:"oom_strike_count" json:"oom_strike_count"`
	LastOOMKilledAt       *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
}

// UpdateStudentStatusDTO DTO para suspender o reactivar una cuenta de estudiante
type UpdateStudentStatusDTO struct {
	Status string `json:"status" validate:"required"` // "active" | "suspended"
	Reason string `json:"reason,omitempty"`
}

// CreateStudentDTO DTO para el alta manual de un estudiante por el administrador
type CreateStudentDTO struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}

// AdminStudentCourseItem DTO para el detalle de materias y espacios de trabajo del estudiante
type AdminStudentCourseItem struct {
	SubjectID       string     `db:"subject_id" json:"subject_id"`
	SubjectCode     string     `db:"subject_code" json:"subject_code"`
	SubjectName     string     `db:"subject_name" json:"subject_name"`
	TeacherName     string     `db:"teacher_name" json:"teacher_name"`
	EnrolledAt      time.Time  `db:"enrolled_at" json:"enrolled_at"`
	WorkspaceID     *string    `db:"workspace_id" json:"workspace_id,omitempty"`
	WorkspaceStatus *string    `db:"workspace_status" json:"workspace_status,omitempty"`
	MemoryLimitMB   *int       `db:"memory_limit_mb" json:"memory_limit_mb,omitempty"`
	OOMStrikeCount  *int       `db:"oom_strike_count" json:"oom_strike_count,omitempty"`
	LastOOMKilledAt *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
}

// ResetOOMDTO DTO para solicitar el reseteo manual de penalizaciones por OOM-Killed (ADR-033)
type ResetOOMDTO struct {
	Reason string `json:"reason" validate:"required,min=10"`
}

// ResetOOMResult resultado tras resetear strikes OOM
type ResetOOMResult struct {
	StudentID            string `json:"student_id"`
	WorkspacesResetCount int64  `json:"workspaces_reset_count"`
	Message              string `json:"message"`
}

// ReviewTemplateDTO DTO para aprobar o rechazar plantillas Docker (ADR-030)
type ReviewTemplateDTO struct {
	Status          string `json:"status" validate:"required"` // "approved" | "rejected" | "paused"
	RejectionReason string `json:"rejection_reason,omitempty"`
	BaseRamMB       *int   `json:"base_ram_mb,omitempty"`
}

// CreateOfficialTemplateDTO DTO para que el administrador registre directamente una plantilla oficial
type CreateOfficialTemplateDTO struct {
	Name              string                   `json:"name" validate:"required"`
	DockerImage       string                   `json:"docker_image" validate:"required"`
	BaseRamMB         int                      `json:"base_ram_mb" validate:"required,gt=0"`
	Description       string                   `json:"description"`
	TargetEnvironment string                   `json:"target_environment"` // "IDE_PERSISTENTE" | "JUEZ_EFIMERO"
	CategoryID        *string                  `json:"category_id,omitempty"`
	ModelID           *string                  `json:"model_id,omitempty"`
	Entrypoint        string                   `json:"entrypoint,omitempty"`
	TimeoutMS         int                      `json:"timeout_ms,omitempty"`
	SampleInput       string                   `json:"sample_input,omitempty"`
	ServicesConfig    *ServicesConfig          `json:"services_config,omitempty"`
	ResourceProfile   *TemplateResourceProfile `json:"resource_profile,omitempty"`
	SetupScript       string                   `json:"setup_script,omitempty"`
	ToolsDeclared     []string                 `json:"tools_declared,omitempty"`
}

// AdminTemplateReviewItem modelo para listar y gestionar plantillas Docker institucionales (ADR-030)
type AdminTemplateReviewItem struct {
	ID                  string                  `db:"id" json:"id"`
	TenantID            *string                 `db:"tenant_id" json:"tenant_id,omitempty"`
	Name                string                  `db:"name" json:"name"`
	DockerImage         string                  `db:"docker_image" json:"docker_image"`
	BaseRamMB           int                     `db:"base_ram_mb" json:"base_ram_mb"`
	Status              string                  `db:"status" json:"status"`
	RejectionReason     string                  `db:"rejection_reason" json:"rejection_reason"`
	ReviewedBy          *string                 `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt          *time.Time              `db:"reviewed_at" json:"reviewed_at,omitempty"`
	RequestedBy         *string                 `db:"requested_by" json:"requested_by,omitempty"`
	RequestedByName     *string                 `db:"requested_by_name" json:"requested_by_name,omitempty"`
	Description         string                  `db:"description" json:"description"`
	TargetEnvironment   string                  `db:"target_environment" json:"target_environment"`
	Entrypoint          string                  `db:"entrypoint" json:"entrypoint"`
	TimeoutMS           int                     `db:"timeout_ms" json:"timeout_ms"`
	SampleInput         string                  `db:"sample_input" json:"sample_input"`
	CategoryID          *string                 `db:"category_id" json:"category_id,omitempty"`
	CategoryName        *string                 `db:"category_name" json:"category_name,omitempty"`
	ModelID             *string                 `db:"model_id" json:"model_id,omitempty"`
	ServicesConfig      ServicesConfig          `db:"services_config" json:"services_config"`
	ResourceProfile     TemplateResourceProfile `db:"resource_profile" json:"resource_profile"`
	SetupScript         string                  `db:"setup_script" json:"setup_script"`
	ToolsDeclared       []string                `db:"-" json:"tools_declared"`
	SmokeTestStatus     string                  `db:"smoke_test_status" json:"smoke_test_status"`
	SmokeTestOutput     string                  `db:"smoke_test_output" json:"smoke_test_output"`
	SecurityAuditStatus string                  `db:"security_audit_status" json:"security_audit_status"`
	CVECriticalCount    int                     `db:"cve_critical_count" json:"cve_critical_count"`
	CVEHighCount        int                     `db:"cve_high_count" json:"cve_high_count"`
	SecurityAuditedAt   *time.Time              `db:"security_audited_at" json:"security_audited_at,omitempty"`
	EOLStatus           string                  `db:"eol_status" json:"eol_status"`
	EOLDate             string                  `db:"eol_date" json:"eol_date"`
	EOLMessage          string                  `db:"eol_message" json:"eol_message"`
	EOLCheckedAt        *time.Time              `db:"eol_checked_at" json:"eol_checked_at,omitempty"`
	CreatedAt           time.Time               `db:"created_at" json:"created_at"`
}

// TemplateCategory categoría institucional de entornos
type TemplateCategory struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"tenant_id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	SortOrder   int       `db:"sort_order" json:"sort_order"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CreateCategoryDTO struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateCategoryDTO struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active,omitempty"`
	SortOrder   *int   `json:"sort_order,omitempty"`
}

type ReorderCategoryItemDTO struct {
	ID        string `json:"id" validate:"required"`
	SortOrder int    `json:"sort_order"`
}

// TemplateModelItemDTO modelo oficial preconfigurado para creación rápida
type TemplateModelItemDTO struct {
	ID                string    `db:"id" json:"id"`
	TenantID          string    `db:"tenant_id" json:"tenant_id"`
	CategoryID        string    `db:"category_id" json:"category_id"`
	CategoryName      string    `db:"category_name" json:"category_name"`
	SourceTemplateID  *string   `db:"source_template_id" json:"source_template_id,omitempty"`
	Title             string    `db:"title" json:"title"`
	Description       string    `db:"description" json:"description"`
	DockerImage       string    `db:"docker_image" json:"docker_image"`
	BaseRamMB         int       `db:"base_ram_mb" json:"base_ram_mb"`
	Tools             []string  `db:"-" json:"tools"`
	TargetEnvironment string    `db:"target_environment" json:"target_environment"`
	Entrypoint        string    `db:"entrypoint" json:"entrypoint"`
	TimeoutMS         int       `db:"timeout_ms" json:"timeout_ms"`
	SampleInput       string    `db:"sample_input" json:"sample_input"`
	UsageCount        int       `db:"usage_count" json:"usage_count"`
	IsActive          bool      `db:"is_active" json:"is_active"`
	SortOrder         int       `db:"sort_order" json:"sort_order"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

type UpdateTemplateModelDTO struct {
	Title       string `json:"title" validate:"required,min=2,max=150"`
	Description string `json:"description"`
	CategoryID  string `json:"category_id" validate:"required"`
}

type PromoteTemplateToModelDTO struct {
	CategoryID  string `json:"category_id" validate:"required"`
	Title       string `json:"title" validate:"required,min=3,max=150"`
	Description string `json:"description"`
}

// EmergencyActionRequest DTO para solicitar una acción de emergencia administrativa (ADR-032)
type EmergencyActionRequest struct {
	ConfirmationPhrase string `json:"confirmation_phrase" validate:"required"`
	Reason             string `json:"reason,omitempty"`
}

// EmergencyActionResult DTO con el resultado de la acción de emergencia
type EmergencyActionResult struct {
	Action        string `json:"action"`
	AffectedCount int64  `json:"affected_count"`
	ExecutedBy    string `json:"executed_by"`
	Message       string `json:"message"`
}

// HostCapacityInfo métricas físicas reales del host de ejecución
type HostCapacityInfo struct {
	TotalRAMMB     int64 `json:"total_ram_mb"`
	AvailableRAMMB int64 `json:"available_ram_mb"`
	UsedRAMMB      int64 `json:"used_ram_mb"`
	CPUCores       int   `json:"cpu_cores"`
}

// SatelliteServiceCapability especificación de un servicio satélite soportado por la plataforma
type SatelliteServiceCapability struct {
	Category    string `json:"category"`
	Engine      string `json:"engine"`
	Label       string `json:"label"`
	Version     string `json:"version"`
	Description string `json:"description"`
	EnvVar      string `json:"env_var"`
	IsAvailable bool   `json:"is_available"`
}

// RamPresetSuggestion preset de memoria contextual sugerido según capacidades del nodo
type RamPresetSuggestion struct {
	MB    int    `json:"mb"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

// RuntimeCapabilities capacidades de ejecución dinámicas del entorno activo (hardware y servicios)
type RuntimeCapabilities struct {
	HostMemory        HostCapacityInfo             `json:"host_memory"`
	SatelliteServices []SatelliteServiceCapability `json:"satellite_services"`
	IDEPresets        []RamPresetSuggestion        `json:"ide_presets"`
	JudgePresets      []RamPresetSuggestion        `json:"judge_presets"`
	MaxAllowedRamMB   int                          `json:"max_allowed_ram_mb"`
	// EditorBaseMB es la RAM mínima que consume el proceso del editor (OpenVSCode Server).
	// El frontend lo usa para calcular la RAM disponible para el programa del alumno.
	EditorBaseMB  int `json:"editor_base_mb"`
	// RuntimeBaseMB es la RAM mínima reservada para el runtime del Juez (kernel + sandbox).
	RuntimeBaseMB int `json:"runtime_base_mb"`
}

var (
	ErrDraftNotFound = errors.New("template draft not found")
)

// TemplateDraft borrador de configuración de plantilla persistido en BD
type TemplateDraft struct {
	ID         string          `json:"id" db:"id"`
	TenantID   string          `json:"tenant_id" db:"tenant_id"`
	UserID     string          `json:"user_id" db:"user_id"`
	FormData   json.RawMessage `json:"form_data" db:"form_data"`
	TemplateID *string         `json:"template_id,omitempty" db:"template_id"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// CreateDraftDTO DTO de creación o actualización de borrador
type CreateDraftDTO struct {
	FormData   json.RawMessage `json:"form_data"`
	TemplateID *string         `json:"template_id,omitempty"`
}

// DraftResponse respuesta pública de borrador
type DraftResponse struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	FormData  json.RawMessage `json:"form_data"`
	UpdatedAt time.Time       `json:"updated_at"`
}
