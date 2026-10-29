package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrTemplateNameConflict = errors.New("template with this name already exists")
	ErrCategoryInUse        = errors.New("category is in use and cannot be deleted")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrTemplateNotApproved  = errors.New("only approved templates can be promoted to models")
	ErrCannotRevokeSelf     = errors.New("cannot_revoke_self: no es posible revocar los propios privilegios de administrador")
	ErrLastAdministrator    = errors.New("last_administrator: debe existir al menos un administrador activo en la institución")
	ErrAdminNotFound        = errors.New("admin_not_found: el administrador no existe o no tiene rol administrativo")
	ErrInvalidAdminEmail    = errors.New("invalid_admin_email: el correo del administrador es inválido")
	ErrAdminNamesRequired   = errors.New("admin_names_required: el nombre y apellido del administrador son obligatorios")

	ErrUnsupportedServiceEngine = errors.New("unsupported_service_engine: el motor no pertenece al dominio soportado")
	ErrServiceRoleMismatch      = errors.New("service_role_mismatch: el rol no corresponde al motor")
	ErrMultiplePrimaryDatabase  = errors.New("multiple_primary_database: solo se admite una base de datos principal")
	ErrDuplicateServiceEngine   = errors.New("duplicate_service_engine: el motor está declarado más de una vez")
	ErrDuplicateServiceAlias    = errors.New("duplicate_service_alias: el alias de entorno está repetido")
	ErrInvalidServiceAlias      = errors.New("invalid_service_alias: el alias debe empezar con letra y usar minúsculas, números o guion bajo")
)

// AdminUserItem representa a un operador con rol administrativo en el tenant.
type AdminUserItem struct {
	ID          string     `json:"id" db:"id"`
	Email       string     `json:"email" db:"email"`
	FirstName   string     `json:"first_name" db:"first_name"`
	LastName    string     `json:"last_name" db:"last_name"`
	Role        string     `json:"role" db:"role"`
	Status      string     `json:"status" db:"status"`
	Origin      string     `json:"origin" db:"origin"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// CreateAdminDTO encapsula los datos requeridos para pre-aprovisionar un administrador.
type CreateAdminDTO struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

const (
	TargetEnvironmentIDEPersistente = "IDE_PERSISTENTE"
	TargetEnvironmentJuezEfimero    = "JUEZ_EFIMERO"
	TargetEnvironmentAmbos          = "AMBOS"
)

// TemplateSupportsEnvironment verifica si la plantilla puede satisfacer el entorno solicitado
func TemplateSupportsEnvironment(templateTargetEnv, requestedEnv string) bool {
	if templateTargetEnv == TargetEnvironmentAmbos {
		return requestedEnv == TargetEnvironmentIDEPersistente || requestedEnv == TargetEnvironmentJuezEfimero
	}
	return templateTargetEnv == requestedEnv
}

// ResolveJudgeRamMB resuelve un límite seguro de memoria para ejecuciones de Juez Efímero sobre plantillas duales.
func ResolveJudgeRamMB(baseRamMB int) int {
	if baseRamMB >= 1024 {
		return 256
	}
	return 128
}

// ServiceRole es el rol del servicio dentro de la plantilla. El rol es el contrato con
// el laboratorio; el motor es la implementación.
type ServiceRole string

const (
	ServiceRolePrimaryDatabase ServiceRole = "primary_database"
	ServiceRoleCache           ServiceRole = "cache"
)

// ServiceInitKind declara cómo se aplica el db_init_script del ejercicio en un motor.
// La ejecución de los init NoSQL llega con los adaptadores; el contrato ya los nombra.
type ServiceInitKind string

const (
	ServiceInitNone      ServiceInitKind = "none"
	ServiceInitSQL       ServiceInitKind = "sql"
	ServiceInitMongoJS   ServiceInitKind = "mongo-js"
	ServiceInitRedisCmds ServiceInitKind = "redis-cmds"
)

// ServiceRequirement requerimiento de un servicio satélite complementario (BD, cache, message broker, etc.)
type ServiceRequirement struct {
	Role        ServiceRole `json:"role,omitempty"`         // "primary_database" | "cache"
	Category    string      `json:"category,omitempty"`     // alias heredado: "database" | "cache"
	Engine      string      `json:"engine"`                 // "postgres", "mysql", "mongodb", "redis", etc.
	ComponentID string      `json:"component_id,omitempty"` // trazabilidad al catálogo
	Version     string      `json:"version,omitempty"`
	EnvAlias    string      `json:"env_alias,omitempty"`
}

// ServicesConfig configuración agnóstica y extensible de servicios satélite requeridos
type ServicesConfig struct {
	Services []ServiceRequirement `json:"services"`
}

var serviceEnvAliasRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// CanonicalServiceEngine expone la resolución de alias de motor del catálogo.
func CanonicalServiceEngine(engine string) string {
	return canonicalServiceEngine(engine)
}

// ServiceRoleForEngine resuelve el rol que corresponde a un motor.
func ServiceRoleForEngine(engine string) ServiceRole {
	switch canonicalServiceEngine(engine) {
	case "sqlite", "postgres", "mysql", "mariadb", "mongodb":
		return ServiceRolePrimaryDatabase
	case "redis":
		return ServiceRoleCache
	default:
		return ""
	}
}

// EffectiveRole resuelve el rol declarado o el heredado de category/engine.
func (sr ServiceRequirement) EffectiveRole() ServiceRole {
	if role := ServiceRole(strings.ToLower(strings.TrimSpace(string(sr.Role)))); role != "" {
		return role
	}
	if strings.EqualFold(strings.TrimSpace(sr.Category), string(ServiceRoleCache)) {
		return ServiceRoleCache
	}
	return ServiceRoleForEngine(sr.Engine)
}

// EffectiveEnvAlias resuelve el alias estable del binding para el laboratorio.
func (sr ServiceRequirement) EffectiveEnvAlias() string {
	if alias := strings.TrimSpace(sr.EnvAlias); alias != "" {
		return alias
	}
	switch sr.EffectiveRole() {
	case ServiceRolePrimaryDatabase:
		return "db"
	case ServiceRoleCache:
		return "cache"
	default:
		return ""
	}
}

// InitKind declara el contrato de inicialización del motor.
func (sr ServiceRequirement) InitKind() ServiceInitKind {
	switch canonicalServiceEngine(sr.Engine) {
	case "postgres", "mysql", "mariadb":
		return ServiceInitSQL
	case "mongodb":
		return ServiceInitMongoJS
	case "redis":
		return ServiceInitRedisCmds
	default:
		return ServiceInitNone
	}
}

// NormalizeServiceBindings valida la cardinalidad por rol, canoniza motores y alias, y
// deja la lista en su forma persistible. Es la única puerta de validación del contrato.
func NormalizeServiceBindings(cfg *ServicesConfig) error {
	if cfg == nil {
		return nil
	}

	normalized := make([]ServiceRequirement, 0, len(cfg.Services))
	seenEngines := make(map[string]bool, len(cfg.Services))
	seenAliases := make(map[string]bool, len(cfg.Services))
	primaryCount := 0

	for _, raw := range cfg.Services {
		engine := canonicalServiceEngine(raw.Engine)
		if engine == "" {
			return fmt.Errorf("%w: %q", ErrUnsupportedServiceEngine, raw.Engine)
		}
		if engine == "none" {
			continue
		}

		role := raw.EffectiveRole()
		expectedRole := ServiceRoleForEngine(engine)
		if role == "" || role != expectedRole {
			return fmt.Errorf("%w: engine=%q role=%q", ErrServiceRoleMismatch, engine, role)
		}
		if seenEngines[engine] {
			return fmt.Errorf("%w: %q", ErrDuplicateServiceEngine, engine)
		}
		seenEngines[engine] = true

		if role == ServiceRolePrimaryDatabase {
			primaryCount++
			if primaryCount > 1 {
				return ErrMultiplePrimaryDatabase
			}
		}

		alias := raw.EffectiveEnvAlias()
		if !serviceEnvAliasRegex.MatchString(alias) {
			return fmt.Errorf("%w: %q", ErrInvalidServiceAlias, alias)
		}
		if seenAliases[alias] {
			return fmt.Errorf("%w: %q", ErrDuplicateServiceAlias, alias)
		}
		seenAliases[alias] = true

		// `category` es solo el alias de lectura heredado de `role`: se deriva para que no
		// pueda contradecirlo (un Redis enviado como "database" sigue siendo caché).
		category := "database"
		if role == ServiceRoleCache {
			category = string(ServiceRoleCache)
		}

		normalized = append(normalized, ServiceRequirement{
			Role:        role,
			Category:    category,
			Engine:      engine,
			ComponentID: strings.TrimSpace(raw.ComponentID),
			Version:     strings.TrimSpace(raw.Version),
			EnvAlias:    alias,
		})
	}

	cfg.Services = normalized
	return nil
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

var (
	ErrRamExceedsHostCapacity = errors.New("la memoria asignada supera el techo permitido por el host")
)

// CalculateHostMaxAllowedRAM deriva el techo estructural de RAM asignable (75% de la RAM física total, mín 512 MB).
func CalculateHostMaxAllowedRAM(totalHostMB int) int {
	maxAllowed := int(float64(totalHostMB) * 0.75)
	if maxAllowed < 512 {
		return 512
	}
	return maxAllowed
}

// ValidateRamAgainstHost valida que la memoria solicitada no supere el techo máximo del host.
func ValidateRamAgainstHost(baseRamMB, maxAllowedMB int) error {
	if maxAllowedMB > 0 && baseRamMB > maxAllowedMB {
		return fmt.Errorf("%w: base_ram_mb=%d supera max_allowed_ram_mb=%d", ErrRamExceedsHostCapacity, baseRamMB, maxAllowedMB)
	}
	return nil
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

// BatchCreateStudentsResult DTO con el resultado del alta masiva de estudiantes
type BatchCreateStudentsResult struct {
	TotalProcessed int `json:"total_processed"`
	CreatedCount   int `json:"created_count"`
}

// RosterEnrollResult DTO con el resumen de la nómina de curso: cuentas creadas,
// matriculaciones nuevas y correos inválidos/duplicados detectados.
type RosterEnrollResult struct {
	CreatedStudents  int      `json:"created_students"`
	EnrolledStudents int      `json:"enrolled_students"`
	DuplicateEmails  []string `json:"duplicate_emails"`
	InvalidEmails    []string `json:"invalid_emails"`
	TotalDetected    int      `json:"total_detected"`
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
	StackConfig       *StackConfig             `json:"stack_config,omitempty"`
	VerificationJobID string                   `json:"verification_job_id,omitempty"`
}

// UpdateOfficialTemplateDTO DTO para actualizar una plantilla existente con versionado sin corte
type UpdateOfficialTemplateDTO struct {
	Name              string          `json:"name" validate:"required"`
	DockerImage       string          `json:"docker_image" validate:"required"`
	BaseRamMB         int             `json:"base_ram_mb" validate:"required,gt=0"`
	Description       string          `json:"description"`
	TargetEnvironment string          `json:"target_environment"`
	CategoryID        *string         `json:"category_id,omitempty"`
	Entrypoint        string          `json:"entrypoint,omitempty"`
	TimeoutMS         int             `json:"timeout_ms,omitempty"`
	SampleInput       string          `json:"sample_input,omitempty"`
	ServicesConfig    *ServicesConfig `json:"services_config,omitempty"`
	SetupScript       string          `json:"setup_script,omitempty"`
	ToolsDeclared     []string        `json:"tools_declared,omitempty"`
	StackConfig       *StackConfig    `json:"stack_config,omitempty"`
	VerificationJobID string          `json:"verification_job_id,omitempty"`
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
	StackConfig         StackConfig             `db:"stack_config" json:"stack_config"`
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
	ID                string      `db:"id" json:"id"`
	TenantID          string      `db:"tenant_id" json:"tenant_id"`
	CategoryID        string      `db:"category_id" json:"category_id"`
	CategoryName      string      `db:"category_name" json:"category_name"`
	SourceTemplateID  *string     `db:"source_template_id" json:"source_template_id,omitempty"`
	Title             string      `db:"title" json:"title"`
	Description       string      `db:"description" json:"description"`
	DockerImage       string      `db:"docker_image" json:"docker_image"`
	BaseRamMB         int         `db:"base_ram_mb" json:"base_ram_mb"`
	Tools             []string    `db:"-" json:"tools"`
	TargetEnvironment string      `db:"target_environment" json:"target_environment"`
	Entrypoint        string      `db:"entrypoint" json:"entrypoint"`
	TimeoutMS         int         `db:"timeout_ms" json:"timeout_ms"`
	SampleInput       string      `db:"sample_input" json:"sample_input"`
	StackConfig       StackConfig `db:"stack_config" json:"stack_config"`
	UsageCount        int         `db:"usage_count" json:"usage_count"`
	IsActive          bool        `db:"is_active" json:"is_active"`
	SortOrder         int         `db:"sort_order" json:"sort_order"`
	CreatedAt         time.Time   `db:"created_at" json:"created_at"`
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

// Modos de aprovisionamiento de un motor de base de datos en el nodo.
const (
	// ProvisioningEmbedded: el motor vive dentro del propio laboratorio (archivo local).
	ProvisioningEmbedded = "embedded"
	// ProvisioningSharedHost: el motor es un servicio compartido del nodo (aula).
	ProvisioningSharedHost = "shared_host"
	// ProvisioningManagedContainer: el motor se levanta bajo demanda como contenedor
	// satélite y se detiene cuando ningún laboratorio lo usa (Scale-to-Zero).
	ProvisioningManagedContainer = "managed_container"
)

// SatelliteServiceCapability especificación de un servicio satélite soportado por la plataforma
type SatelliteServiceCapability struct {
	Category    string `json:"category"`
	Engine      string `json:"engine"`
	Label       string `json:"label"`
	Version     string `json:"version"`
	Description string `json:"description"`
	EnvVar      string `json:"env_var"`
	IsAvailable bool   `json:"is_available"`
	BaseRAMMB   int    `json:"base_ram_mb"`
	// ProvisioningMode indica cómo se aprovisiona el motor: 'embedded', 'shared_host'
	// o 'managed_container'. El frontend lo usa para explicar el costo de memoria y
	// si el motor se levanta bajo demanda.
	ProvisioningMode string `json:"provisioning_mode"`
	// DefaultImage es la imagen versionada con la que se levanta el motor cuando se
	// aprovisiona como contenedor gestionado (vacío en motores embebidos o de host).
	DefaultImage string `json:"default_image,omitempty"`
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
	EditorBaseMB int `json:"editor_base_mb"`
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

// TemplateCacheStatus representa el estado de presencia física de la imagen en el host Docker
type TemplateCacheStatus struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	DockerImage  string `json:"docker_image"`
	IsCached     bool   `json:"is_cached"`
	SizeBytes    int64  `json:"size_bytes"`
	Status       string `json:"status"` // "cached", "missing", "warming", "failed"
	ErrorMessage string `json:"error_message,omitempty"`
}

// WarmAllSummary resume el resultado de calentamiento de caché masivo
type WarmAllSummary struct {
	TotalChecked int `json:"total_checked"`
	AlreadyWarm  int `json:"already_warm"`
	WarmedCount  int `json:"warmed_count"`
	FailedCount  int `json:"failed_count"`
}
