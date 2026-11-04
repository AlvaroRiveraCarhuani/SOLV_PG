package domain

import (
	"context"
	"errors"
	"time"
)

const (
	// ReviewEnvironmentStatusStarting cubre el arranque síncrono del contenedor.
	ReviewEnvironmentStatusStarting = "starting"
	ReviewEnvironmentStatusRunning  = "running"
	ReviewEnvironmentStatusExpired  = "expired"
	ReviewEnvironmentStatusFailed   = "failed"

	// ReviewEnvironmentDefaultPort es el puerto que usa la app cuando la práctica
	// no declara uno propio.
	ReviewEnvironmentDefaultPort = 3000
	// ReviewEnvironmentIdleTimeout hereda el estándar de inactividad del sistema
	// (ADR-011): el preview se apaga solo, sin botón de detener.
	ReviewEnvironmentIdleTimeout = 15 * time.Minute
	// ReviewEnvironmentStartTimeout acota la espera hasta que la app escucha.
	ReviewEnvironmentStartTimeout = 45 * time.Second
	// ReviewEnvironmentMinMemoryMB es el piso de memoria del contenedor de
	// preview. La plantilla puede pedir más; nunca menos.
	ReviewEnvironmentMinMemoryMB int64 = 512
	// ReviewEnvironmentMaxCommandLength evita comandos desmedidos en la base.
	ReviewEnvironmentMaxCommandLength = 500
)

var (
	// ErrReviewEnvironmentUnavailable: el nodo no tiene orquestador o almacén de
	// volúmenes para arrancar previews.
	ErrReviewEnvironmentUnavailable = errors.New("review_environment_unavailable")
	// ErrReviewEnvironmentRequiresIDE: solo los laboratorios IDE tienen una app
	// que se pueda abrir en el navegador.
	ErrReviewEnvironmentRequiresIDE = errors.New("review_environment_requires_ide")
	// ErrReviewEnvironmentCommandMissing: la práctica no declara cómo arrancar
	// su aplicación, así que no hay forma segura de adivinarlo.
	ErrReviewEnvironmentCommandMissing = errors.New("app_run_command_missing")
	// ErrReviewEnvironmentSnapshotUnavailable: la entrega no tiene un snapshot
	// multiarchivo que se pueda volcar al contenedor.
	ErrReviewEnvironmentSnapshotUnavailable = errors.New("project_snapshot_unavailable")
	// ErrReviewEnvironmentTemplateMissing: la práctica no resuelve una imagen.
	ErrReviewEnvironmentTemplateMissing = errors.New("template_image_missing")
	// ErrReviewEnvironmentHostMemory: control de admisión por OOM_GUARD_MB.
	ErrReviewEnvironmentHostMemory = errors.New("oom_guard")
	// ErrReviewEnvironmentStartFailed: Docker no pudo crear o arrancar el
	// contenedor.
	ErrReviewEnvironmentStartFailed = errors.New("app_start_failed")
	// ErrReviewEnvironmentNotReady: el contenedor arrancó pero la app no quedó
	// escuchando en el puerto declarado.
	ErrReviewEnvironmentNotReady = errors.New("app_not_ready")
	// ErrReviewEnvironmentStarting: otro pedido del mismo docente está creando
	// el preview; reintentar cuando termine evita dos contenedores a la vez.
	ErrReviewEnvironmentStarting = errors.New("review_environment_starting")
	// ErrInvalidAppPort: el puerto de ejecución está fuera del rango TCP.
	ErrInvalidAppPort = errors.New("invalid_app_port")
)

// ReviewEnvironment es una ejecución inmutable del snapshot de una entrega. No
// comparte volumen con el environment vivo del alumno ni con otra revisión.
type ReviewEnvironment struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"tenant_id"`
	SubmissionID   string    `db:"submission_id" json:"submission_id"`
	TeacherID      string    `db:"teacher_id" json:"teacher_id"`
	ContainerID    *string   `db:"container_id" json:"-"`
	VolumeName     string    `db:"volume_name" json:"-"`
	AccessURL      string    `db:"access_url" json:"url"`
	Status         string    `db:"status" json:"status"`
	ErrorCode      string    `db:"error_code" json:"error_code,omitempty"`
	ErrorMessage   string    `db:"error_message" json:"error_message,omitempty"`
	LastActivityAt time.Time `db:"last_activity_at" json:"last_activity_at"`
	ExpiresAt      time.Time `db:"expires_at" json:"expires_at"`
	LastRxBytes    int64     `db:"last_rx_bytes" json:"-"`
	LastTxBytes    int64     `db:"last_tx_bytes" json:"-"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// IsActive indica si el preview todavía puede servirse sin recrearlo.
func (e *ReviewEnvironment) IsActive() bool {
	if e == nil {
		return false
	}
	return e.Status == ReviewEnvironmentStatusStarting || e.Status == ReviewEnvironmentStatusRunning
}

// ReviewEnvironmentAccess es la respuesta pública del contrato: la URL que el
// docente abre en una pestaña nueva y el instante en que el QoS la apagará si
// nadie la usa.
type ReviewEnvironmentAccess struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ReviewEnvironmentRepository persiste la vida del preview. La unicidad del
// preview activo por entrega la garantiza un índice parcial en PostgreSQL.
type ReviewEnvironmentRepository interface {
	Create(ctx context.Context, env *ReviewEnvironment) error
	GetActiveBySubmission(ctx context.Context, tenantID, submissionID string) (*ReviewEnvironment, error)
	ListActive(ctx context.Context) ([]*ReviewEnvironment, error)
	SetRunning(ctx context.Context, id, containerID, accessURL string, expiresAt time.Time, rxBytes, txBytes int64) error
	Touch(ctx context.Context, id string, lastActivityAt, expiresAt time.Time, rxBytes, txBytes int64) error
	MarkFailed(ctx context.Context, id, errorCode, errorMessage string) error
	MarkExpired(ctx context.Context, id string) error
}

// ReviewEnvironmentContext es lo que SpeedGrader necesita para ejecutar la
// entrega: el snapshot, la imagen de la práctica y la receta explícita de
// arranque. No incluye el environment vivo del alumno.
type ReviewEnvironmentContext struct {
	SubmissionID     string
	ExerciseID       string
	EnvironmentType  string
	Code             string
	Language         string
	AppRunCommand    string
	AppPort          int
	TemplateImage    string
	TemplateSetup    string
	TemplateMemoryMB int64
}

// ReviewEnvironmentSource lee el contexto del preview con la misma autorización
// de tenant que la revisión docente. Lo implementa el repositorio de SpeedGrader.
type ReviewEnvironmentSource interface {
	GetReviewEnvironmentContext(ctx context.Context, tenantID, teacherID, submissionID string) (*ReviewEnvironmentContext, error)
}

// ReviewEnvironmentContainerConfig describe el contenedor efímero que ejecuta
// la app del alumno desde una copia del snapshot.
type ReviewEnvironmentContainerConfig struct {
	Image               string
	ContainerName       string
	VolumeName          string
	NetworkName         string
	AppPort             int
	Command             string
	SetupScript         string
	Env                 []string
	Labels              map[string]string
	MemoryLimitMB       int64
	HostPort            string
	AttachSatellitesNet bool
}

// ReviewEnvironmentOrchestrator es la capacidad Docker del preview. Se declara
// separada de PracticeEnvironmentOrchestrator para no romper sus dobles de test.
type ReviewEnvironmentOrchestrator interface {
	EnsureVolumeExists(ctx context.Context, volumeName string) error
	EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error
	WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error
	StartReviewEnvironmentContainer(ctx context.Context, config ReviewEnvironmentContainerConfig) (string, error)
	GetContainerMetrics(ctx context.Context, containerID string) (*ContainerMetrics, error)
	GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error)
	StopAndRemoveContainer(ctx context.Context, containerID string) error
	RemoveVolume(ctx context.Context, volumeName string) error
}
