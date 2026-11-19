package domain

import (
	"context"
	"encoding/json"
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

	DefaultBaseMemoryMB int64 = 256
	// HardQuotaMemoryMB es el mayor techo que puede alcanzar un entorno. No es
	// la protección del nodo: esa la ejerce el presupuesto de admisión sobre la
	// RAM total del host. El techo por contenedor es alto para que un perfil
	// Intensivo (Datos & IA) quepa cuando el nodo tiene hueco (ADR-082).
	HardQuotaMemoryMB   int64   = 4096
	MinHostFreeRAMPct   float64 = 15.0 // Porcentaje mínimo de RAM libre en el Host
	MaxOOMStrikes       int     = 3
	OOMCooldownDuration         = 5 * time.Minute

	// Factor de suavizado EWMA (alpha = 0.2) y factor de margen (1.25 -> 25% headroom)
	EWMAAlpha              float64 = 0.2
	EWMASafetyMarginFactor float64 = 1.25
)

// EnvironmentProfile declara las reglas que aplican a un entorno según el
// momento académico. Se deriva del propósito del ejercicio y nunca lo envía el
// cliente: un examen no puede quedar en perfil de práctica por un campo del
// formulario.
type EnvironmentProfile string

const (
	EnvironmentProfilePractice EnvironmentProfile = "practice"
	EnvironmentProfileExam     EnvironmentProfile = "exam"
)

// ProfileForPurpose traduce el propósito del ejercicio al perfil del entorno.
// Un propósito ausente o desconocido cae en práctica: el perfil de examen exige
// que el ejercicio se haya declarado como examen.
func ProfileForPurpose(purpose string) EnvironmentProfile {
	if ExercisePurpose(purpose) == ExercisePurposeExam {
		return EnvironmentProfileExam
	}
	return EnvironmentProfilePractice
}

func (p EnvironmentProfile) IsExam() bool {
	return p == EnvironmentProfileExam
}

// EnvironmentProfileSpec declara qué cambia por momento. No lleva memoria: el
// cupo lo decide la política del servidor y el préstamo de la admisión.
type EnvironmentProfileSpec struct {
	Profile                 EnvironmentProfile
	IdleHibernation         bool
	HostPressureHibernation bool
	KeystrokeCapture        bool
	FreshWorkspace          bool
	FreezeOnSubmit          bool
}

// ProfileSpecFor devuelve el conjunto de reglas de un perfil. Un perfil
// desconocido se comporta como práctica: los privilegios del examen se
// conceden solo con el perfil declarado.
func ProfileSpecFor(profile EnvironmentProfile) EnvironmentProfileSpec {
	if profile == EnvironmentProfileExam {
		return EnvironmentProfileSpec{
			Profile:                 EnvironmentProfileExam,
			HostPressureHibernation: true,
			KeystrokeCapture:        true,
			FreshWorkspace:          true,
			FreezeOnSubmit:          true,
		}
	}
	return EnvironmentProfileSpec{
		Profile:                 EnvironmentProfilePractice,
		IdleHibernation:         true,
		HostPressureHibernation: true,
	}
}

type PracticeEnvironment struct {
	ID            string  `db:"id" json:"environment_id"`
	StudentID     string  `db:"student_id" json:"student_id"`
	SubjectID     string  `db:"subject_id" json:"subject_id"`
	Type          string  `db:"type" json:"type"`
	ContainerID   *string `db:"container_id" json:"-"`
	Status        string  `db:"status" json:"status"`
	AccessURL     string  `db:"access_url" json:"access_url"`
	MemoryLimitMB int64   `db:"memory_limit_mb" json:"memory_limit_mb"`
	// MemoryProfileMB es lo que declaró la plantilla (o la política) antes del
	// préstamo de memoria del nodo. Con MemoryLimitMB se explica la diferencia
	// entre lo pedido y lo concedido.
	MemoryProfileMB int64 `db:"memory_profile_mb" json:"memory_profile_mb,omitempty"`
	// CPUMillis es la cuota absoluta de CPU con la que se creó el contenedor.
	// Cero significa que no se persistió la cuota (registros anteriores).
	CPUMillis int64 `db:"cpu_millis" json:"cpu_millis,omitempty"`
	// DiskUsedMB y DiskState son la última medición del barrido de disco; la
	// pantalla lee el valor guardado y no mide nada. DiskLimitMB sale de la
	// política vigente al responder y no se persiste.
	DiskUsedMB  int64  `db:"disk_used_mb" json:"disk_used_mb"`
	DiskState   string `db:"disk_state" json:"disk_state"`
	DiskLimitMB int64  `db:"-" json:"disk_limit_mb,omitempty"`
	// EnvironmentProfile es el perfil aplicado en la última apertura. Se
	// recalcula desde el propósito del ejercicio; el valor persistido sirve para
	// auditar con qué reglas se ejecutó.
	EnvironmentProfile EnvironmentProfile `db:"environment_profile" json:"environment_profile"`
	// ExamAttempt cuenta los intentos de examen del entorno. Cero en práctica;
	// el primer examen es 1 y forma parte del nombre del volumen del intento.
	ExamAttempt     int        `db:"exam_attempt" json:"exam_attempt"`
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

// NextExamAttempt devuelve el número del intento que sigue a uno cerrado. El
// primero es el 1: el intento 0 es la práctica y nunca cuenta como examen.
func NextExamAttempt(current int) int {
	if current < 1 {
		return 1
	}
	return current + 1
}

// PracticeEnvironmentAttemptVolumeName arma el volumen de un intento. La
// práctica usa el volumen de siempre (intento 0); el examen estrena uno por
// intento, así no hereda lo que hubiera quedado antes.
func PracticeEnvironmentAttemptVolumeName(studentID, subjectID string, exerciseID *string, attempt int) string {
	base := PracticeEnvironmentVolumeName(studentID, subjectID, exerciseID)
	if base == "" || attempt <= 0 {
		return base
	}
	return fmt.Sprintf("%s_exam_%d", base, attempt)
}

// VolumeName devuelve el volumen del entorno según su alcance e intento.
func (w *PracticeEnvironment) VolumeName() string {
	if w == nil {
		return ""
	}
	return PracticeEnvironmentAttemptVolumeName(w.StudentID, w.SubjectID, w.ExerciseID, w.ExamAttempt)
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
	// CPUNanos es la cuota absoluta de CPU. Cero deja el contenedor sin cuota.
	CPUNanos int64
	// PidsLimit acota los procesos del contenedor. Cero usa el tope del runtime.
	PidsLimit int64
	// ExamNoEgress encierra el entorno en una red interna sin puerta de enlace.
	// El editor se publica por un proxy que Traefik sí alcanza, porque una red
	// interna no admite puertos publicados.
	ExamNoEgress bool
	NetworkName  string
	Labels       map[string]string
	Env          []string
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
	// MountDockerSocket monta el socket del daemon en solo lectura. Solo lo pide
	// una plantilla declarada en SOLV_DOCKER_SOCKET_TEMPLATE_IDS: el resto de los
	// laboratorios sigue sin acceso al daemon (ADR-043).
	MountDockerSocket bool
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
// Estados guardados de la última medición de disco.
const (
	DiskStateOK      = "ok"
	DiskStateWarning = "warning"
	DiskStateFull    = "full"
)

// PracticeEnvironmentDiskUsageRecorder guarda la medición del barrido. Se resuelve
// por type-assertion para no ampliar PracticeEnvironmentRepository.
type PracticeEnvironmentDiskUsageRecorder interface {
	RecordDiskUsage(ctx context.Context, environmentID string, usedMB int64, state string) error
}

// ErrPracticeEnvironmentSnapshotTooLarge indica que el contenido del volumen no
// entra en el tope de la lectura. Una entrega lo trata como error para no
// guardar medio proyecto; los avisos de clase siguen recortando a propósito.
var ErrPracticeEnvironmentSnapshotTooLarge = errors.New("el contenido del volumen excede el tope de la lectura")

// PracticeEnvironmentState es el estado que ve el alumno. Se calcula del
// registro y no se guarda: una columna aparte se quedaría vieja en cuanto el
// contenedor cambia sin pasar por el servicio.
type PracticeEnvironmentState string

const (
	// PracticeEnvironmentStatePreparing: el contenedor todavía no responde o el
	// proyecto base todavía se está escribiendo.
	PracticeEnvironmentStatePreparing PracticeEnvironmentState = "preparing"
	// PracticeEnvironmentStateReady: el alumno puede entrar y escribir.
	PracticeEnvironmentStateReady PracticeEnvironmentState = "ready"
	// PracticeEnvironmentStateClosed: el examen terminó y el volumen está en solo
	// lectura.
	PracticeEnvironmentStateClosed PracticeEnvironmentState = "closed"
	// PracticeEnvironmentStateFailed: el arranque falló o el contenedor murió por
	// memoria.
	PracticeEnvironmentStateFailed PracticeEnvironmentState = "failed"
	// PracticeEnvironmentStateDiskWarning: el volumen pasó el umbral de aviso. El
	// editor sigue abierto; solo se le pide al alumno que libere espacio.
	PracticeEnvironmentStateDiskWarning PracticeEnvironmentState = "disk_warning"
	// PracticeEnvironmentStateDiskFull: la entrega se frena hasta que baje del
	// umbral. El editor no se cierra: perder trabajo sin guardar es peor que una
	// entrega rechazada.
	PracticeEnvironmentStateDiskFull PracticeEnvironmentState = "disk_full"
)

// State traduce el registro al estado visible. Un examen cerrado manda sobre
// cualquier otro estado: lo importante para el alumno es que no puede escribir.
func (p PracticeEnvironment) State() PracticeEnvironmentState {
	switch {
	case p.FrozenAt != nil || p.Status == PracticeEnvironmentStatusFrozen:
		return PracticeEnvironmentStateClosed
	case p.Status == PracticeEnvironmentStatusFailed || p.Status == PracticeEnvironmentStatusOOMKilled:
		return PracticeEnvironmentStateFailed
	case p.Status != PracticeEnvironmentStatusRunning:
		return PracticeEnvironmentStatePreparing
	case p.DiskState == DiskStateFull:
		return PracticeEnvironmentStateDiskFull
	case p.DiskState == DiskStateWarning:
		return PracticeEnvironmentStateDiskWarning
	case p.WorkspaceSeedMode == WorkspaceSeedModeProject && failedSeedStatus(p.WorkspaceInitStatus):
		// La siembra no se va a recuperar sola: decir "preparando" deja al alumno
		// esperando un entorno que no va a abrir.
		return PracticeEnvironmentStateFailed
	case p.WorkspaceSeedMode == WorkspaceSeedModeProject && p.WorkspaceInitStatus != WorkspaceInitStatusSetupDone:
		return PracticeEnvironmentStatePreparing
	default:
		return PracticeEnvironmentStateReady
	}
}

// failedSeedStatus dice si la siembra del proyecto quedó en un estado del que no
// se sale solo: cuarentena, fallo o volumen perdido.
func failedSeedStatus(status string) bool {
	switch status {
	case WorkspaceInitStatusFailed, WorkspaceInitStatusQuarantined, WorkspaceInitStatusMissing:
		return true
	default:
		return false
	}
}

// MarshalJSON agrega el estado calculado a cada respuesta del entorno. Los
// campos guardados quedan igual.
func (p PracticeEnvironment) MarshalJSON() ([]byte, error) {
	type plain PracticeEnvironment
	return json.Marshal(struct {
		plain
		State PracticeEnvironmentState `json:"state"`
	}{plain: plain(p), State: p.State()})
}

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
