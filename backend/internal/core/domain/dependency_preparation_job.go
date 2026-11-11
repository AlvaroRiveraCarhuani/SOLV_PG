package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DependencyPreparationJobStatus es el estado de la preparación de una versión.
// `pending` es la única cola; `running` marca el trabajo que el nodo tiene
// tomado; `prepared`, `failed` y `skipped` son finales hasta un reintento manual.
type DependencyPreparationJobStatus string

const (
	DependencyPreparationJobPending  DependencyPreparationJobStatus = "pending"
	DependencyPreparationJobRunning  DependencyPreparationJobStatus = "running"
	DependencyPreparationJobPrepared DependencyPreparationJobStatus = "prepared"
	DependencyPreparationJobFailed   DependencyPreparationJobStatus = "failed"
	DependencyPreparationJobSkipped  DependencyPreparationJobStatus = "skipped"
)

var (
	ErrDependencyPreparationJobNotFound  = errors.New("trabajo de preparación no encontrado")
	ErrInvalidDependencyPreparationJob   = errors.New("trabajo de preparación inválido")
	ErrDependencyPreparationNotRetryable = errors.New("la preparación no se puede reintentar")
	ErrDependencyPreparationDisabled     = errors.New("la preparación de dependencias no está activa en este nodo")
)

// DependencyPreparationJob es la solicitud persistida de una versión. El trabajo
// no reemplaza el resultado del worker: guarda a quién pertenece, qué clave se
// preparó, en qué estado quedó y dónde mirar cuando falla.
type DependencyPreparationJob struct {
	ID          string                         `db:"id" json:"id"`
	TenantID    string                         `db:"tenant_id" json:"tenant_id"`
	VersionID   string                         `db:"version_id" json:"version_id"`
	ProjectID   string                         `db:"project_id" json:"project_id"`
	ComponentID string                         `db:"component_id" json:"component_id,omitempty"`
	Key         string                         `db:"preparation_key" json:"preparation_key"`
	Status      DependencyPreparationJobStatus `db:"status" json:"status"`
	Attempts    int                            `db:"attempts" json:"attempts"`
	LogPath     string                         `db:"log_path" json:"log_path,omitempty"`
	Message     string                         `db:"message" json:"message,omitempty"`
	TreePath    string                         `db:"tree_path" json:"tree_path,omitempty"`
	Bytes       int64                          `db:"bytes" json:"bytes"`
	CreatedAt   time.Time                      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time                      `db:"updated_at" json:"updated_at"`
}

// Validate comprueba lo mínimo para persistir el trabajo.
func (j DependencyPreparationJob) Validate() error {
	if strings.TrimSpace(j.TenantID) == "" {
		return fmt.Errorf("%w: falta el tenant", ErrInvalidDependencyPreparationJob)
	}
	if strings.TrimSpace(j.VersionID) == "" {
		return fmt.Errorf("%w: falta la versión", ErrInvalidDependencyPreparationJob)
	}
	if strings.TrimSpace(j.ProjectID) == "" {
		return fmt.Errorf("%w: falta el proyecto", ErrInvalidDependencyPreparationJob)
	}
	switch j.Status {
	case DependencyPreparationJobPending,
		DependencyPreparationJobRunning,
		DependencyPreparationJobPrepared,
		DependencyPreparationJobFailed,
		DependencyPreparationJobSkipped:
	default:
		return fmt.Errorf("%w: estado desconocido %q", ErrInvalidDependencyPreparationJob, j.Status)
	}
	if key := strings.TrimSpace(j.Key); key != "" && len(key) != 64 {
		return fmt.Errorf("%w: la clave debe tener 64 caracteres hex", ErrInvalidDependencyPreparationJob)
	}
	return nil
}

// Retryable dice si el trabajo puede volver a la cola. `failed` y `skipped`
// sí: el catálogo o el proyecto pudieron cambiar después. `prepared` solo
// cuenta como reintentable cuando el árbol ya no está, y eso lo decide quien
// conoce el disco.
func (j DependencyPreparationJob) Retryable() bool {
	switch j.Status {
	case DependencyPreparationJobFailed, DependencyPreparationJobSkipped:
		return true
	default:
		return false
	}
}

// DependencyPreparationJobCandidate es una versión publicada sin trabajo. La
// usa el reconciliador al arrancar para cubrir una publicación cuyo encolado no
// llegó a persistirse.
type DependencyPreparationJobCandidate struct {
	TenantID   string `db:"tenant_id"`
	VersionID  string `db:"version_id"`
	ProjectID  string `db:"project_id"`
	ExerciseID string `db:"exercise_id"`
}

// DependencyPreparationTarget es lo que el catálogo aporta para preparar una
// versión: la imagen donde correr la instalación y el perfil que la describe.
type DependencyPreparationTarget struct {
	ComponentID   string
	ComponentSlug string
	BaseImage     string
	Profile       *DependencyProfile
	User          string
	Home          string
}

// DependencyPreparationTargetResolver resuelve el entorno de un ejercicio desde
// el catálogo. Devuelve nil sin error cuando el ejercicio no declara una
// tecnología de ejecución: ese caso termina en `skipped`, no en fallo.
type DependencyPreparationTargetResolver interface {
	ResolveDependencyPreparationTarget(ctx context.Context, tenantID, exerciseID string) (*DependencyPreparationTarget, error)
}

// DependencyPreparationJobRepository es el contrato de la cola. Enqueue es
// idempotente por versión; ClaimNext toma un solo trabajo y lo marca en curso
// con bloqueo, de modo que dos nodos no preparen lo mismo.
type DependencyPreparationJobRepository interface {
	Enqueue(ctx context.Context, job *DependencyPreparationJob) (*DependencyPreparationJob, error)
	ClaimNext(ctx context.Context) (*DependencyPreparationJob, error)
	UpdateResult(ctx context.Context, id string, status DependencyPreparationJobStatus, result DependencyPreparationResult) error
	// ResetForRetry devuelve el trabajo a la cola con la clave recalculada. Un
	// reintento es una ejecución nueva: el desenlace anterior no se conserva,
	// y la clave se reemplaza porque el perfil pudo cambiar.
	ResetForRetry(ctx context.Context, id, key string) error
	GetByVersion(ctx context.Context, tenantID, versionID string) (*DependencyPreparationJob, error)
	ListActiveKeys(ctx context.Context) (map[string]bool, error)
	ListVersionsWithoutJob(ctx context.Context, limit int) ([]DependencyPreparationJobCandidate, error)
	// ReclaimStale devuelve a la cola los trabajos tomados por un nodo que no
	// terminó. Sin esto, una caída a mitad deja la versión en `running` para
	// siempre y el reconciliador no la mira, porque solo busca versiones sin
	// trabajo.
	ReclaimStale(ctx context.Context, olderThan time.Duration) (int, error)
}
