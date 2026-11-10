package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Modos de siembra de un entorno de proyecto. El nombre visible no se deriva
// de estos identificadores: son valores de contrato persistidos.
const (
	WorkspaceSeedModeProject = "project"
	WorkspaceSeedModeNone    = "none"
	WorkspaceSeedModeLegacy  = "legacy"

	WorkspaceInitStatusLegacy        = "legacy"
	WorkspaceInitStatusPending       = "pending"
	WorkspaceInitStatusVolumeCreated = "volume_created"
	WorkspaceInitStatusSetupDone     = "setup_done"
	WorkspaceInitStatusFailed        = "failed"
	WorkspaceInitStatusQuarantined   = "quarantined"
	WorkspaceInitStatusMissing       = "missing"
	WorkspaceInitStatusNotRequired   = "not_required"
)

// Clasificación de la base de comparación de una entrega.
const (
	BaseStatusVersion       = "version"
	BaseStatusNoBase        = "no_base"
	BaseStatusLegacyUnknown = "legacy_unknown"
)

var (
	ErrWorkspaceProjectNotFound           = errors.New("workspace project not found")
	ErrWorkspaceProjectExists             = errors.New("workspace project already exists")
	ErrWorkspaceDraftRevisionConflict     = errors.New("workspace draft revision conflict")
	ErrWorkspaceDraftEmpty                = errors.New("el borrador del proyecto está vacío")
	ErrWorkspacePublishingDisabled        = errors.New("publicación de proyectos deshabilitada")
	ErrWorkspaceProjectManagedBoilerplate = errors.New("el proyecto base de este entorno lo gestiona la publicación de proyectos")
	ErrWorkspaceProjectForbidden          = errors.New("no tiene permisos sobre el proyecto de este ejercicio")
	ErrWorkspaceProjectNotIDEPersistent   = errors.New("el proyecto inicial solo aplica a ejercicios IDE_PERSISTENTE")
	ErrWorkspaceProjectImportInvalid      = errors.New("la importación del proyecto no es válida")
	ErrWorkspaceProjectImportTooLarge     = errors.New("la importación del proyecto excede el tamaño permitido")
	ErrWorkspaceSeedVersionInvalid        = errors.New("la versión publicada del entorno no es válida")
	ErrWorkspaceSeedVolumeNotEmpty        = errors.New("el volumen del entorno ya contiene archivos")
	ErrWorkspaceSeedVolumeConflict        = errors.New("el volumen del entorno pertenece a otra inicialización")
	ErrWorkspaceSeedVolumeAmbiguous       = errors.New("el estado del volumen no permite continuar sin revisión")
	ErrWorkspaceSeedVolumeUnavailable     = errors.New("no se puede operar el volumen del entorno")
	ErrWorkspaceSeedInProgress            = errors.New("la inicialización del entorno ya está en curso")
)

// WorkspaceProject es el contenedor estable de un proyecto inicial. El
// ejercicio es su dueño y solo puede tener un proyecto.
type WorkspaceProject struct {
	ID                        string    `db:"id" json:"id"`
	TenantID                  string    `db:"tenant_id" json:"tenant_id"`
	ExerciseID                string    `db:"exercise_id" json:"exercise_id"`
	CurrentPublishedVersionID *string   `db:"current_published_version_id" json:"current_published_version_id,omitempty"`
	CreatedBy                 *string   `db:"created_by" json:"created_by,omitempty"`
	CreatedAt                 time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time `db:"updated_at" json:"updated_at"`
}

// WorkspaceProjectDraft es el único borrador mutable del proyecto. base_version_id
// identifica la versión sobre la que se está editando; nil en el primer borrador.
type WorkspaceProjectDraft struct {
	ID            string    `db:"id" json:"id"`
	TenantID      string    `db:"tenant_id" json:"tenant_id"`
	ProjectID     string    `db:"project_id" json:"project_id"`
	BaseVersionID *string   `db:"base_version_id" json:"base_version_id,omitempty"`
	TextSnapshot  string    `db:"text_snapshot" json:"text_snapshot"`
	ContentSHA256 string    `db:"content_sha256" json:"content_sha256"`
	Revision      int64     `db:"revision" json:"revision"`
	UpdatedBy     *string   `db:"updated_by" json:"updated_by,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// WorkspaceProjectManifestEntry describe un archivo de una versión sin duplicar
// el contenido, que ya vive en text_snapshot.
type WorkspaceProjectManifestEntry struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
	Lines int    `json:"lines"`
}

// WorkspaceProjectVersion es una publicación inmutable. Una vez creada no se
// actualiza ni se borra: es la base exacta de las entregas que la referencian.
type WorkspaceProjectVersion struct {
	ID            string          `db:"id" json:"id"`
	TenantID      string          `db:"tenant_id" json:"tenant_id"`
	ProjectID     string          `db:"project_id" json:"project_id"`
	ExerciseID    string          `db:"exercise_id" json:"exercise_id"`
	VersionNumber int             `db:"version_number" json:"version_number"`
	TextSnapshot  string          `db:"text_snapshot" json:"text_snapshot"`
	Manifest      json.RawMessage `db:"manifest" json:"manifest"`
	ContentSHA256 string          `db:"content_sha256" json:"content_sha256"`
	FileCount     int             `db:"file_count" json:"file_count"`
	TotalBytes    int             `db:"total_bytes" json:"total_bytes"`
	CreatedBy     *string         `db:"created_by" json:"created_by,omitempty"`
	PublishedAt   time.Time       `db:"published_at" json:"published_at"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

// WorkspaceLegacyProjectReference conserva el proyecto base histórico sin
// convertirlo en versión ni vincularlo a entregas.
type WorkspaceLegacyProjectReference struct {
	ID            string    `db:"id" json:"id"`
	TenantID      string    `db:"tenant_id" json:"tenant_id"`
	ExerciseID    string    `db:"exercise_id" json:"exercise_id"`
	TextSnapshot  string    `db:"text_snapshot" json:"text_snapshot"`
	ContentSHA256 string    `db:"content_sha256" json:"content_sha256"`
	Source        string    `db:"source" json:"source"`
	CapturedAt    time.Time `db:"captured_at" json:"captured_at"`
}

// WorkspaceSeedFile es un archivo textual de la versión que se escribe en el
// volumen del entorno. No contiene rutas resueltas ni permisos: la
// infraestructura vuelve a validar cada ruta antes de escribir.
type WorkspaceSeedFile struct {
	Path    string
	Content string
}

// WorkspaceSeed es la unidad de inicialización de un volumen. VersionID y
// ContentSHA256 vienen de la versión inmutable publicada y se usan para
// detectar reanudaciones sobre otra versión.
type WorkspaceSeed struct {
	VersionID     string
	ContentSHA256 string
	Files         []WorkspaceSeedFile
}

// WorkspaceSeedVolumeStore es una capacidad opcional del orquestador. La
// implementación debe ser conservadora: solo escribe un volumen vacío o una
// inicialización propia pendiente, y nunca reemplaza contenido del alumno.
type WorkspaceSeedVolumeStore interface {
	PrepareWorkspaceVolume(ctx context.Context, volumeName string, seed WorkspaceSeed) error
}

// WorkspaceSeedVolumeVerifier comprueba, sin escribir, que un volumen sigue
// conservando el marcador de la inicialización asignada. Se usa al reabrir un
// entorno ya inicializado para no confiar únicamente en el estado de la base.
type WorkspaceSeedVolumeVerifier interface {
	VerifyWorkspaceVolume(ctx context.Context, volumeName string, seed WorkspaceSeed) error
}

// WorkspaceProjectRepository es el contrato de persistencia de proyectos. Las
// operaciones de publicación son transaccionales: versión, espejo del ejercicio
// y puntero del proyecto se escriben juntos o no se escribe ninguno.
type WorkspaceProjectRepository interface {
	GetProjectByExercise(ctx context.Context, tenantID, exerciseID string) (*WorkspaceProject, error)
	CreateProject(ctx context.Context, project *WorkspaceProject) (*WorkspaceProject, error)
	GetDraftByProject(ctx context.Context, tenantID, projectID string) (*WorkspaceProjectDraft, error)
	SaveDraft(ctx context.Context, draft *WorkspaceProjectDraft, expectedRevision int64) (*WorkspaceProjectDraft, error)
	Publish(ctx context.Context, tenantID, projectID, userID string) (*WorkspaceProjectVersion, error)
	ListVersions(ctx context.Context, tenantID, projectID string) ([]*WorkspaceProjectVersion, error)
	GetVersion(ctx context.Context, tenantID, versionID string) (*WorkspaceProjectVersion, error)
	GetLegacyReference(ctx context.Context, tenantID, exerciseID string) (*WorkspaceLegacyProjectReference, error)
}
