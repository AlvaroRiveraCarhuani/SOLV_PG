package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	PracticeEnvironmentStatusPending    = "pending"
	PracticeEnvironmentStatusRunning    = "running"
	PracticeEnvironmentStatusHibernated = "hibernated"
	PracticeEnvironmentStatusOOMKilled  = "oom_killed"
	PracticeEnvironmentStatusFailed     = "failed"
	// PracticeEnvironmentStatusUnknown marca un laboratorio cuyo estado no pudo verificarse
	// (el daemon de Docker no respondio durante la reconciliacion). No destruye el
	// registro: reintenta el siguiente ciclo y retiene el cupo de max_containers.
	PracticeEnvironmentStatusUnknown = "unknown"
	// PracticeEnvironmentStatusOrphaned marca un laboratorio cuyo contenedor ya no existe en
	// el daemon. Libera el cupo y deja el registro para que el alumno lo recree.
	PracticeEnvironmentStatusOrphaned = "orphaned"
	// PracticeEnvironmentStatusFrozen marca el laboratorio de un examen ya cerrado: el
	// volumen quedó montado en solo lectura y la edición no se reabre sola
	// (ADR-005 §5). Es un estado buscado, no un fallo.
	PracticeEnvironmentStatusFrozen = "frozen"

	PracticeEnvironmentTypeIDEPersistente = "IDE_PERSISTENTE"
	PracticeEnvironmentTypeJuezEfimero    = "JUEZ_EFIMERO"

	DefaultBaseMemoryMB int64   = 256
	HardQuotaMemoryMB   int64   = 2048
	MinHostFreeRAMPct   float64 = 15.0 // Porcentaje mínimo de RAM libre en el Host
	MaxOOMStrikes       int     = 3
	OOMCooldownDuration         = 5 * time.Minute

	// Factor de suavizado EWMA (alpha = 0.2) y factor de margen (1.25 -> 25% headroom)
	EWMAAlpha              float64 = 0.2
	EWMASafetyMarginFactor float64 = 1.25
)

type PracticeEnvironment struct {
	ID              string     `db:"id" json:"environment_id"`
	StudentID       string     `db:"student_id" json:"student_id"`
	SubjectID       string     `db:"subject_id" json:"subject_id"`
	Type            string     `db:"type" json:"type"`
	ContainerID     *string    `db:"container_id" json:"-"`
	Status          string     `db:"status" json:"status"`
	AccessURL       string     `db:"access_url" json:"access_url"`
	MemoryLimitMB   int64      `db:"memory_limit_mb" json:"memory_limit_mb"`
	LastHeartbeatAt time.Time  `db:"last_heartbeat_at" json:"last_heartbeat_at"`
	LastOOMKilledAt *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
	OOMStrikeCount  int        `db:"oom_strike_count" json:"oom_strike_count"`
	SemgrepAudit    []byte     `db:"semgrep_audit" json:"semgrep_audit,omitempty"`
	TenantID        string     `db:"tenant_id" json:"tenant_id"`
	TemplateID      *string    `db:"template_id" json:"template_id,omitempty"`
	// ExerciseID acota el entorno a una práctica concreta, con su propio
	// volumen. Todo entorno pertenece a una práctica: docentes y alumnos tienen
	// uno por cada práctica que abren, nunca uno compartido por materia.
	ExerciseID *string `db:"exercise_id" json:"exercise_id,omitempty"`
	// FrozenAt es la hora en que el examen se cerró y el volumen pasó a solo
	// lectura. Nulo mientras el alumno puede editar.
	FrozenAt *time.Time `db:"frozen_at" json:"frozen_at,omitempty"`
	// WorkspaceProjectVersionID es la versión exacta sembrada en el volumen.
	// Solo existe cuando WorkspaceSeedMode=project.
	WorkspaceProjectVersionID *string   `db:"workspace_project_version_id" json:"workspace_project_version_id,omitempty"`
	WorkspaceSeedMode         string    `db:"workspace_seed_mode" json:"workspace_seed_mode"`
	WorkspaceInitStatus       string    `db:"workspace_init_status" json:"workspace_init_status"`
	CreatedAt                 time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time `db:"updated_at" json:"updated_at"`
}

// PracticeEnvironmentVolumeName arma el volumen nombrado de un entorno. Todo entorno se
// acota a una práctica y lleva el ejercicio en el nombre, para no mezclar el
// código de dos prácticas de la misma materia. Sin práctica no hay volumen: el
// modelo por materia se retiró y no se recrea.
func PracticeEnvironmentVolumeName(studentID, subjectID string, exerciseID *string) string {
	if exerciseID == nil {
		return ""
	}
	trimmed := strings.TrimSpace(*exerciseID)
	if trimmed == "" {
		return ""
	}
	return fmt.Sprintf("solv_workspace_%s_%s_%s", studentID, subjectID, trimmed)
}

// VolumeName devuelve el volumen del entorno según su alcance.
func (w *PracticeEnvironment) VolumeName() string {
	if w == nil {
		return ""
	}
	return PracticeEnvironmentVolumeName(w.StudentID, w.SubjectID, w.ExerciseID)
}

var (
	// ErrPracticeEnvironmentFrozen: el examen terminó y el entorno quedó en solo lectura.
	// Se devuelve al intentar abrir o reanudar el laboratorio, para que el
	// alumno no reciba un contenedor en escritura por recargar la página.
	ErrPracticeEnvironmentFrozen = errors.New("el examen terminó: el entorno quedó en solo lectura")
	// ErrPracticeEnvironmentPracticeRequired: todo entorno pertenece a una práctica. Sin
	// ejercicio no hay volumen, porque el modelo por materia se retiró.
	ErrPracticeEnvironmentPracticeRequired = errors.New("la práctica del entorno es requerida")
)

type PracticeEnvironmentContainerConfig struct {
	Image         string
	ContainerName string
	VolumeName    string
	MemoryLimitMB int64
	NetworkName   string
	Labels        map[string]string
	Env           []string
	// SetupScript es el script de inicialización de la plantilla compuesta.
	// Cuando está presente, el runtime lo ejecuta una sola vez por sesión.
	SetupScript string
	// HostPort es el puerto ya reservado en 127.0.0.1 que se publicará contra el
	// 3000/tcp del contenedor. Vacío delega la elección al daemon de Docker.
	HostPort string
	// AttachSatellitesNet conecta además el contenedor del laboratorio a la red
	// interna de satélites, donde resuelve por DNS el motor de base de datos que
	// su plantilla declara (solv-sat-<engine>). Falso cuando el laboratorio no usa
	// un satélite gestionado.
	AttachSatellitesNet bool
	// ReadOnly monta el volumen del alumno sin permiso de escritura. Se usa al
	// cerrar un examen: el bloqueo lo aplica el sistema de archivos, no la
	// pantalla (ADR-005 §5).
	ReadOnly bool
}

// PracticeEnvironmentVolumeFile es un archivo leído del volumen de un environment.
type PracticeEnvironmentVolumeFile struct {
	Path    string
	Content []byte
}

// PracticeEnvironmentVolumeStore es una capacidad opcional del orquestador: permite leer y
// escribir el volumen nombrado del environment sin depender del estado del
// contenedor. Se resuelve por type-assertion para no ampliar
// PracticeEnvironmentOrchestrator (y no romper sus mocks).
type PracticeEnvironmentVolumeStore interface {
	SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]PracticeEnvironmentVolumeFile, error)
	WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error
}

// PracticeEnvironmentBackupPruner es una capacidad opcional del almacén de volúmenes:
// permite listar las carpetas de respaldo de un volumen y borrar las que ya no
// se conservan. Se resuelve por type-assertion para no ampliar
// PracticeEnvironmentVolumeStore ni romper sus dobles.
type PracticeEnvironmentBackupPruner interface {
	// ListPracticeEnvironmentBackupDirs devuelve las carpetas de root, de la más nueva a
	// la más vieja. root es una ruta relativa dentro del volumen.
	ListPracticeEnvironmentBackupDirs(ctx context.Context, volumeName, root string) ([]string, error)
	// DeletePracticeEnvironmentVolumePaths borra rutas relativas dentro del volumen. La
	// implementación solo acepta carpetas de respaldo.
	DeletePracticeEnvironmentVolumePaths(ctx context.Context, volumeName string, paths []string) error
}

type ContainerMetrics struct {
	MemoryUsageBytes int64
	MemoryLimitBytes int64
	CPUPercent       float64
	RxBytes          uint64
	TxBytes          uint64
	IsRunning        bool
	OOMKilled        bool
	ExitCode         int
}

type EWMAState struct {
	CurrentEWMAMB float64   `json:"current_ewma_mb"`
	SampleCount   int       `json:"sample_count"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

type ResourceProfile struct {
	SignatureHash string    `json:"signature_hash"`
	BaseMemoryMB  int64     `json:"base_memory_mb"`
	MaxQuotaMB    int64     `json:"max_quota_mb"`
	EWMAState     EWMAState `json:"ewma_state"`
}

type LabTemplate struct {
	ID              string          `db:"id" json:"id"`
	Name            string          `db:"name" json:"name"`
	BaseImage       string          `db:"base_image" json:"base_image"`
	SetupScript     string          `db:"setup_script" json:"setup_script"`
	SignatureHash   string          `db:"signature_hash" json:"signature_hash"`
	ResourceProfile ResourceProfile `json:"resource_profile"`
	TenantID        string          `db:"tenant_id" json:"tenant_id"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
}
