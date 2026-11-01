package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

// Alcance de un avance: toda la clase o alumnos elegidos de la sesión.
const (
	AdvanceScopeClass    = "class"
	AdvanceScopeSelected = "selected"
)

// Estado de un destinatario. Se decide una sola vez.
const (
	AdvanceRecipientPending   = "pending"
	AdvanceRecipientApplied   = "applied"
	AdvanceRecipientDismissed = "dismissed"
	AdvanceRecipientFailed    = "failed"
	AdvanceRecipientExpired   = "expired"
)

// Acciones del aula que quedan en el registro de auditoría (ADR-058 §12).
// Comparten el prefijo CLASS_ para leer el historial de una sesión sin mezclar
// las peticiones HTTP genéricas que también guarda el middleware.
const (
	AuditClassAdvanceEmitted       = "CLASS_ADVANCE_EMITTED"
	AuditClassFrozen               = "CLASS_FROZEN"
	AuditClassUnfrozen             = "CLASS_UNFROZEN"
	AuditClassAdvanceGraceExtended = "CLASS_ADVANCE_GRACE_EXTENDED"
	AuditClassAdvanceGraceDeadline = "CLASS_ADVANCE_GRACE_DEADLINE_SET"
	AuditClassAdvanceGraceClosed   = "CLASS_ADVANCE_GRACE_CLOSED"
	AuditClassAdvanceApplied       = "CLASS_ADVANCE_APPLIED"
	AuditClassAdvanceDismissed     = "CLASS_ADVANCE_DISMISSED"
)

// Ventana de rezago por defecto al cerrar el aula, y extensiones válidas.
const (
	DefaultAdvanceGraceMinutes = 30
	MinAdvanceGraceMinutes     = 1
	MaxAdvanceGraceMinutes     = 480
	// MaxAdvanceGraceDays acota la ventana total desde el cierre de la clase.
	MaxAdvanceGraceDays = 7
	// AdvanceSnapshotRetentionDays cubre la última reapertura posible de la
	// ventana más un día de margen. El objeto no se borra antes.
	AdvanceSnapshotRetentionDays = MaxAdvanceGraceDays + 1
	// AdvanceSnapshotPurgeBatch acota cuántos avances revisa un barrido.
	AdvanceSnapshotPurgeBatch = 500
	// AdvanceSnapshotPurgeMaxBatches acota cuántos lotes procesa un ciclo. Si
	// una fila no se deja marcar, el barrido termina en vez de repetirla para
	// siempre; el próximo ciclo la vuelve a intentar.
	AdvanceSnapshotPurgeMaxBatches = 100
)

const (
	// AdvanceSnapshotOrphanAge es la edad mínima de un objeto sin fila antes de
	// borrarlo. Cubre la ventana entre subir el tar.gz y guardar la fila.
	AdvanceSnapshotOrphanAge = 24 * time.Hour
)

var (
	// ErrAdvanceFrozen: la clase está congelada; no se emiten avances nuevos.
	ErrAdvanceFrozen = errors.New("la clase está congelada, no se emiten avances")
	// ErrAdvanceSessionClosed: el aula no está abierta; no se emite.
	ErrAdvanceSessionClosed = errors.New("el aula no está abierta")
	// ErrAdvanceExamBlocked: el ejercicio es de examen; no se emite ni se aplica.
	ErrAdvanceExamBlocked = errors.New("los avances no se emiten en examen")
	// ErrAdvanceGraceExpired: la ventana de rezago venció.
	ErrAdvanceGraceExpired = errors.New("la ventana para aplicar el avance venció")
	// ErrAdvanceNotCloseable: la extensión de ventana solo aplica con la sesión cerrada.
	ErrAdvanceNotCloseable = errors.New("la ventana solo se extiende con el aula cerrada")
	// ErrAdvanceRecipientNotInSession: un destinatario no participa de la sesión.
	ErrAdvanceRecipientNotInSession = errors.New("el alumno no participa de la sesión")
	// ErrAdvanceNoRecipients: el alcance no tiene destinatarios válidos.
	ErrAdvanceNoRecipients = errors.New("el avance no tiene destinatarios")
	// ErrAdvanceScopeInvalid: alcance desconocido o selección vacía.
	ErrAdvanceScopeInvalid = errors.New("alcance de avance inválido")
	// ErrAdvanceSnapshotTooLarge: el snapshot supera el tope de bytes o de archivos.
	ErrAdvanceSnapshotTooLarge = errors.New("el snapshot supera el tamaño permitido")
	// ErrAdvanceGraceInvalid: la extensión de ventana está fuera de rango.
	ErrAdvanceGraceInvalid = errors.New("la ventana debe estar entre 1 y 480 minutos")
	// ErrAdvanceGraceExceeded: la suma supera el máximo desde el cierre.
	ErrAdvanceGraceExceeded = errors.New("la ventana no puede superar los 7 días desde el cierre")
	// ErrAdvanceGraceDeadline: la fecha límite no es futura o supera el máximo.
	ErrAdvanceGraceDeadline = errors.New("la fecha límite debe ser futura y no superar los 7 días desde el cierre")
	// ErrAdvanceNotFound: el avance o el destinatario no existen para este alumno.
	ErrAdvanceNotFound = errors.New("avance no encontrado")
	// ErrAdvanceNotPending: el alumno ya aplicó o descartó el avance.
	ErrAdvanceNotPending = errors.New("el avance ya fue decidido")
	// ErrAdvanceWorkspaceStopped: aplicar exige el laboratorio del alumno en ejecución.
	ErrAdvanceWorkspaceStopped = errors.New("el laboratorio debe estar en ejecución para aplicar el avance")
	// ErrAdvanceSnapshotInvalid: el snapshot no se pudo leer o no coincide con su hash.
	ErrAdvanceSnapshotInvalid = errors.New("el snapshot del avance está corrupto o incompleto")
	// ErrAdvanceSourceMissing: el docente no tiene entorno de clase abierto en la
	// materia, así que no hay código que emitir.
	ErrAdvanceSourceMissing = errors.New("el entorno de clase del docente no está disponible o no tiene código")
)

// ClassAdvance es una emisión del docente: snapshot del código de un ejercicio.
type ClassAdvance struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"-"`
	SessionID      string    `db:"session_id" json:"session_id"`
	ExerciseID     string    `db:"exercise_id" json:"exercise_id"`
	Scope          string    `db:"scope" json:"scope"`
	SnapshotKey    string    `db:"snapshot_key" json:"-"`
	SnapshotSHA256 string    `db:"snapshot_sha256" json:"snapshot_sha256"`
	SnapshotSize   int64     `db:"snapshot_size_bytes" json:"snapshot_size_bytes"`
	FileCount      int       `db:"file_count" json:"file_count"`
	CreatedBy      *string   `db:"created_by" json:"created_by"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	// SnapshotPurgedAt marca que el objeto ya se borró. La fila y el historial
	// se conservan; solo desaparece el código.
	SnapshotPurgedAt *time.Time `db:"snapshot_purged_at" json:"-"`
}

// ClassAdvanceRecipient es el destinatario de un avance. Una fila por alumno.
type ClassAdvanceRecipient struct {
	AdvanceID  string     `db:"advance_id" json:"advance_id"`
	StudentID  string     `db:"student_id" json:"student_id"`
	TenantID   string     `db:"tenant_id" json:"-"`
	Status     string     `db:"status" json:"status"`
	DecidedAt  *time.Time `db:"decided_at" json:"decided_at"`
	BackupPath *string    `db:"backup_path" json:"-"`
}

// AdvanceDelivery es un avance visto por un destinatario. La sesión y el
// propósito del ejercicio vienen resueltos de la misma consulta: al aplicar no
// se puede decidir con datos de otra sesión.
type AdvanceDelivery struct {
	ClassAdvance
	// Status es el estado del destinatario (pending, applied, dismissed, failed, expired).
	Status    string     `db:"recipient_status" json:"status"`
	DecidedAt *time.Time `db:"decided_at" json:"decided_at,omitempty"`
	SubjectID string     `db:"subject_id" json:"subject_id"`
	// SessionStatus es open o closed; con el aula cerrada solo vale la ventana de rezago.
	SessionStatus string     `db:"session_status" json:"session_status"`
	GraceUntil    *time.Time `db:"grace_until" json:"grace_until,omitempty"`
	Purpose       string     `db:"purpose" json:"purpose"`
}

// Appliable dice si el avance todavía se puede aplicar: nunca en examen, y con
// el aula cerrada solo mientras la ventana de rezago siga abierta.
func (d AdvanceDelivery) Appliable(now time.Time) bool {
	if d.Purpose == "exam" {
		return false
	}
	if d.SessionStatus == ClassSessionStatusOpen {
		return true
	}
	return d.GraceUntil != nil && now.Before(*d.GraceUntil)
}

// AdvanceRecipientSummary es un destinatario con el nombre del alumno. La
// lectura del docente no debería cruzar identificadores por su cuenta.
type AdvanceRecipientSummary struct {
	StudentID string     `json:"student_id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Status    string     `json:"status"`
	DecidedAt *time.Time `json:"decided_at"`
}

// AdvanceRecipientCounts resume el estado de una emisión sin que la vista
// recorra la lista para contar.
type AdvanceRecipientCounts struct {
	Total     int `json:"total"`
	Applied   int `json:"applied"`
	Dismissed int `json:"dismissed"`
	Pending   int `json:"pending"`
	Failed    int `json:"failed"`
	Expired   int `json:"expired"`
}

// AdvanceWithRecipients es una emisión del aula con el estado de cada alumno.
type AdvanceWithRecipients struct {
	ID            string                    `json:"id"`
	ExerciseID    string                    `json:"exercise_id"`
	ExerciseTitle string                    `json:"exercise_title"`
	Scope         string                    `json:"scope"`
	FileCount     int                       `json:"file_count"`
	CreatedAt     time.Time                 `json:"created_at"`
	Counts        AdvanceRecipientCounts    `json:"counts"`
	Recipients    []AdvanceRecipientSummary `json:"recipients"`
}

// ClassHistoryEvent es un movimiento del aula ya resuelto para la vista: quién
// lo hizo, cuándo y qué cambió. El detalle viene del metadata del audit log.
type ClassHistoryEvent struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	ActorID    string    `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	ActorEmail string    `json:"actor_email"`
	ActorRole  string    `json:"actor_role"`
	CreatedAt  time.Time `json:"created_at"`

	AdvanceID    *string `json:"advance_id,omitempty"`
	ExerciseID   *string `json:"exercise_id,omitempty"`
	Scope        string  `json:"scope,omitempty"`
	Recipients   int     `json:"recipients,omitempty"`
	FileCount    int     `json:"file_count,omitempty"`
	FilesWritten int     `json:"files_written,omitempty"`
	Minutes      int     `json:"minutes,omitempty"`
	// PreviousUntil y Until muestran el cambio de la ventana: de qué hora a
	// cuál. En un cierre, Until es el momento en que se cerró.
	PreviousUntil *time.Time `json:"previous_until,omitempty"`
	Until         *time.Time `json:"until,omitempty"`
}

// ClassAuditRecorder es la capacidad mínima que el servicio necesita del
// registro de auditoría: escribir una entrada. La lectura del historial va por
// ClassHistoryStore para no acoplar el servicio al repositorio completo.
type ClassAuditRecorder interface {
	Create(ctx context.Context, log *AuditLog) error
}

// ClassHistoryStore lee los movimientos de una sesión desde audit_logs.
type ClassHistoryStore interface {
	SessionHistory(ctx context.Context, tenantID, sessionID string, limit int) ([]ClassHistoryEvent, error)
}

// AdvanceSnapshotObject es un objeto del bucket de avances. LastModified sale
// del almacén y se usa para no borrar un objeto recién subido.
type AdvanceSnapshotObject struct {
	Key          string
	LastModified time.Time
}

// AdvanceSnapshotLister es una capacidad opcional del almacén: lista los
// objetos de un prefijo. El adaptador S3 la implementa para limpiar huérfanos.
type AdvanceSnapshotLister interface {
	List(ctx context.Context, prefix string) ([]AdvanceSnapshotObject, error)
}

// AdvanceSnapshotRetentionStore es la parte del almacenamiento que necesita el
// barrido: candidatos vencidos, marca de purgado y claves todavía referenciadas.
type AdvanceSnapshotRetentionStore interface {
	ExpiredAdvanceSnapshots(ctx context.Context, cutoff time.Time, limit int) ([]ClassAdvance, error)
	MarkSnapshotPurged(ctx context.Context, tenantID, advanceID string, purgedAt time.Time) error
	AdvanceSnapshotKeys(ctx context.Context) ([]string, error)
}

// AdvanceSnapshotStore guarda y lee el tar.gz de un snapshot. La implementación
// concreta es un adaptador S3 contra SeaweedFS (ADR-058 §6).
type AdvanceSnapshotStore interface {
	// Put escribe el objeto y devuelve su clave, hash y tamaño.
	Put(ctx context.Context, key string, body io.Reader) (sha256Hex string, size int64, err error)
	// PutFiles empaqueta un conjunto de archivos y lo escribe como snapshot,
	// aplicando las exclusiones y el tope de archivos de la configuración.
	PutFiles(ctx context.Context, key string, files map[string]string) (sha256Hex string, size int64, fileCount int, err error)
	// Files verifica el hash del objeto y devuelve sus archivos. El adaptador
	// nunca entrega un archivo que no coincida con lo emitido.
	Files(ctx context.Context, key, expectedSHA256 string, maxFiles int) ([]WorkspaceVolumeFile, error)
	// Delete elimina el objeto. Es idempotente.
	Delete(ctx context.Context, key string) error
}

// ClassAdvanceStore declara el almacenamiento de avances y destinatarios.
type ClassAdvanceStore interface {
	// CreateAdvance inserta el avance y sus destinatarios en una transacción.
	CreateAdvance(ctx context.Context, advance ClassAdvance, studentIDs []string) error
	// SessionParticipants devuelve los alumnos con participación en la sesión.
	SessionParticipants(ctx context.Context, tenantID, sessionID string) ([]string, error)
	// ExercisePurpose devuelve el propósito del ejercicio (practice o exam).
	ExercisePurpose(ctx context.Context, tenantID, exerciseID string) (string, error)
	// SetFrozen fija o limpia frozen_at de la sesión del docente.
	SetFrozen(ctx context.Context, tenantID, teacherID, sessionID string, frozen bool) (*ClassSession, error)
	// SetGrace fija grace_until de una sesión cerrada del docente.
	SetGrace(ctx context.Context, tenantID, teacherID, sessionID string, graceUntil time.Time) (*ClassSession, error)
	// PendingAdvances lista los avisos que el alumno todavía no decidió, con la
	// sesión y el propósito resueltos.
	PendingAdvances(ctx context.Context, tenantID, studentID string) ([]AdvanceDelivery, error)
	// AdvanceForStudent devuelve un avance donde el alumno es destinatario, o
	// ErrAdvanceNotFound. Otro alumno nunca lo ve.
	AdvanceForStudent(ctx context.Context, tenantID, studentID, advanceID string) (*AdvanceDelivery, error)
	// DecideRecipient fija el estado final del destinatario una sola vez. Si ya
	// estaba decidido devuelve ErrAdvanceNotPending.
	DecideRecipient(ctx context.Context, tenantID, studentID, advanceID, status string, backupPath *string) error
	// SessionAdvances lista las emisiones del aula con el estado de cada
	// destinatario, del más nuevo al más viejo, filtradas por el docente dueño
	// de la materia. Un teacherID vacío (administrador) no filtra.
	SessionAdvances(ctx context.Context, tenantID, teacherID, sessionID string) ([]AdvanceWithRecipients, error)
}
