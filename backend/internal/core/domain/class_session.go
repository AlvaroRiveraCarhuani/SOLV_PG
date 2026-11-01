package domain

import (
	"context"
	"errors"
	"time"
)

// Estados de un aula. Una sesión nace abierta y muere cerrada: no se programan
// clases a futuro, porque el docente abre el aula cuando empieza a dar clase.
const (
	ClassSessionStatusOpen   = "open"
	ClassSessionStatusClosed = "closed"
)

// Estado de un alumno en el aula. Los tres primeros son excepciones con dueño
// docente; "entregado" y "trabajando" son el curso normal de la clase.
const (
	// ParticipantNotStarted: no hay señal del alumno en esta sesión.
	ParticipantNotStarted = "sin_empezar"
	// ParticipantWorking: está dentro y no ha entregado la práctica de la sesión.
	ParticipantWorking = "trabajando"
	// ParticipantStuck: su última entrega falló hace poco; es la fila que el
	// docente necesita ver primero.
	ParticipantStuck = "trabado"
	// ParticipantSubmitted: entregó la práctica de la sesión.
	ParticipantSubmitted = "entregado"
	// ParticipantAbsent: entró al aula pero lleva más de PresenceWindow sin dar
	// señales. No es lo mismo que no haber entrado nunca.
	ParticipantAbsent = "ausente"
)

const (
	// PresenceWindow es la ventana en la que una señal cuenta como presencia.
	// El cliente del alumno late cada minuto, así que 120 s tolera un latido
	// perdido sin marcar ausencias falsas.
	PresenceWindow = 120 * time.Second
	// StuckWindow acota la ventana en la que un veredicto fallido se muestra como
	// "trabado": pasado ese tiempo el alumno pudo resolverlo solo.
	StuckWindow = 10 * time.Minute
)

const (
	// DefaultAulaDurationMinutes es la duración de un aula cuando el docente no
	// elige otra: una clase de hora y media.
	DefaultAulaDurationMinutes = 90
	// MinAulaDurationMinutes y MaxAulaDurationMinutes acotan la duración pedida.
	// Una clase no dura menos de un cuarto de hora ni más de una jornada.
	MinAulaDurationMinutes = 15
	MaxAulaDurationMinutes = 480
)

// Veredictos que cuentan como un bloqueo del alumno dentro de la clase.
var stuckVerdicts = map[string]bool{
	"RE":          true,
	"TLE":         true,
	"AST_BLOCKED": true,
}

var (
	// ErrSessionAlreadyOpen: la materia ya tiene un aula abierta.
	ErrSessionAlreadyOpen = errors.New("la materia ya tiene un aula abierta")
	// ErrSessionNotOpen: el aula existe pero ya se cerró.
	ErrSessionNotOpen = errors.New("el aula no está abierta")
	// ErrSessionNotFound: el aula no existe, no es del tenant o no pertenece al docente.
	ErrSessionNotFound = errors.New("el aula no existe o no pertenece al docente")
	// ErrStudentNotEnrolled: el alumno no cursa la materia del aula.
	ErrStudentNotEnrolled = errors.New("el alumno no está inscrito en la materia del aula")
	// ErrSessionDurationInvalid: la duración pedida para el aula está fuera de rango.
	ErrSessionDurationInvalid = errors.New("la duración del aula debe estar entre 15 y 480 minutos")
)

// ClassSession es un aula abierta o cerrada de una materia. Enrolled y Present
// vienen resueltos del servidor: la vista no cuenta filas por su cuenta.
type ClassSession struct {
	ID            string  `db:"id" json:"id"`
	TenantID      string  `db:"tenant_id" json:"-"`
	SubjectID     string  `db:"subject_id" json:"subject_id"`
	SubjectCode   string  `db:"subject_code" json:"subject_code"`
	SubjectName   string  `db:"subject_name" json:"subject_name"`
	ExerciseID    *string `db:"exercise_id" json:"exercise_id"`
	ExerciseTitle *string `db:"exercise_title" json:"exercise_title,omitempty"`
	// Purpose es el propósito de la práctica de la sesión: "exam" cierra los
	// volúmenes al terminar; "class" no. Sin práctica asignada, "class".
	Purpose string    `db:"purpose" json:"purpose"`
	Status  string    `db:"status" json:"status"`
	OpensAt time.Time `db:"opens_at" json:"opens_at"`
	// EndsAt es el fin previsto: al vencer, el aula se cierra sola.
	EndsAt   time.Time  `db:"ends_at" json:"ends_at"`
	ClosesAt *time.Time `db:"closes_at" json:"closes_at"`
	// FrozenAt es la congelación manual del docente; nulo si la clase no está congelada.
	FrozenAt *time.Time `db:"frozen_at" json:"frozen_at"`
	// GraceUntil es el fin de la ventana de disponibilidad de los avances
	// emitidos: se fija sola al cerrar y el docente puede sumarle minutos,
	// fijar una fecha límite o cerrarla antes.
	GraceUntil *time.Time `db:"grace_until" json:"grace_until"`
	CreatedBy  string     `db:"created_by" json:"-"`
	Enrolled   int        `db:"enrolled" json:"enrolled"`
	Present    int        `db:"present" json:"present"`
}

// IsOpen dice si el aula todavía admite presencia.
func (s *ClassSession) IsOpen() bool {
	return s != nil && s.Status == ClassSessionStatusOpen
}

// ClassSessionSummary es el marcador del aula. Todas las cifras salen de las
// mismas filas que la tabla, así que no pueden discrepar.
type ClassSessionSummary struct {
	Enrolled          int `json:"enrolled"`
	Present           int `json:"present"`
	NotStarted        int `json:"not_started"`
	Working           int `json:"working"`
	Submitted         int `json:"submitted"`
	Stuck             int `json:"stuck"`
	Absent            int `json:"absent"`
	WithoutSubmission int `json:"without_submission"`
}

// HasExceptions permite escribir "todo en orden" sin que la vista recorra la
// lista de alumnos para decidirlo.
func (s ClassSessionSummary) HasExceptions() bool {
	return s.Stuck > 0 || s.NotStarted > 0 || s.WithoutSubmission > 0
}

// ClassSessionStudent es una fila del aula. Presence y Environment son dos cosas
// distintas: se puede estar en clase sin entorno vivo (juez efímero, o el
// contenedor hibernado) y se puede tener entorno vivo fuera de clase.
type ClassSessionStudent struct {
	StudentID       string     `db:"student_id" json:"student_id"`
	Name            string     `db:"name" json:"name"`
	Email           string     `db:"email" json:"email"`
	State           string     `db:"state" json:"state"`
	Present         bool       `db:"present" json:"present"`
	JoinedAt        *time.Time `db:"joined_at" json:"joined_at"`
	LastSeenAt      *time.Time `db:"last_seen_at" json:"last_seen_at"`
	WorkspaceID     *string    `db:"workspace_id" json:"workspace_id"`
	WorkspaceStatus *string    `db:"workspace_status" json:"workspace_status"`
	SubmittedAt     *time.Time `db:"submitted_at" json:"submitted_at"`
	Verdict         *string    `db:"verdict" json:"verdict"`
	ReviewStatus    *string    `db:"review_status" json:"review_status"`
	Attempts        int        `db:"attempts" json:"attempts"`
}

// IsStuckVerdict expone la misma regla que usa el tablero para que la vista no
// la reinterprete.
func IsStuckVerdict(verdict string) bool {
	return stuckVerdicts[verdict]
}

// ClassSessionBoard es la respuesta completa del aula: la sesión, el marcador y
// los alumnos ordenados por urgencia docente (trabados, sin empezar, ausentes,
// entregados, trabajando).
type ClassSessionBoard struct {
	Session  ClassSession          `json:"session"`
	Summary  ClassSessionSummary   `json:"summary"`
	Students []ClassSessionStudent `json:"students"`
}

// ClassSessionStore declara el almacenamiento del aula en el consumidor.
type ClassSessionStore interface {
	// OpenSession abre el aula de una materia por la duración indicada. Cierra
	// antes las aulas vencidas de la institución, de modo que una sesión olvidada
	// no bloquea la materia. Devuelve ErrSessionAlreadyOpen si todavía hay una
	// abierta y ErrSessionNotFound si la materia no es del docente.
	OpenSession(ctx context.Context, tenantID, teacherID, subjectID string, exerciseID *string, durationMinutes int) (*ClassSession, error)
	// CloseSession cierra el aula. Es idempotente: cerrar dos veces devuelve la
	// misma sesión cerrada.
	CloseSession(ctx context.Context, tenantID, teacherID, sessionID string) (*ClassSession, error)
	// GetOpenSessionForTeacher devuelve el aula abierta del docente, o nil.
	GetOpenSessionForTeacher(ctx context.Context, tenantID, teacherID string) (*ClassSession, error)
	// GetSession devuelve una sesión del docente, abierta o cerrada.
	GetSession(ctx context.Context, tenantID, teacherID, sessionID string) (*ClassSession, error)
	// Board resuelve el tablero del aula sin recorrer alumnos en un bucle.
	Board(ctx context.Context, tenantID, teacherID, sessionID string) (*ClassSessionBoard, error)
	// OpenSessionsForStudent lista las aulas abiertas de las materias del alumno.
	OpenSessionsForStudent(ctx context.Context, tenantID, studentID string) ([]ClassSession, error)
	// TouchParticipant registra o refresca la presencia del alumno en el aula.
	TouchParticipant(ctx context.Context, tenantID, studentID, sessionID string) error
	// AttachWorkspaceToOpenSession enlaza el entorno con el aula abierta de la
	// materia, si existe, y marca la presencia del alumno.
	AttachWorkspaceToOpenSession(ctx context.Context, tenantID, studentID, subjectID, workspaceID string) error
}

// ClassSessionWorkspaceLinker es la capacidad que el servicio de workspaces
// consume para dejar constancia de en qué aula nació un entorno. Se declara
// aparte para no acoplar el ciclo de vida de Docker a la sesión.
type ClassSessionWorkspaceLinker interface {
	AttachWorkspaceToOpenSession(ctx context.Context, tenantID, studentID, subjectID, workspaceID string) error
}
