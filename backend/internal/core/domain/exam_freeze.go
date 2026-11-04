package domain

import (
	"context"
	"errors"
	"time"
)

// Error de dominio para las dos acciones del examen: congelar y reabrir.
var (
	// ErrExamFreezeNotAllowed: la sesión no es de examen, no existe o no es del
	// docente. Emitir y aplicar avances en clase sigue funcionando; el
	// congelamiento es solo de examen (ADR-005 §5).
	ErrExamFreezeNotAllowed = errors.New("solo se congelan los exámenes cerrados del docente")
	// ErrExamSessionOpen: el examen todavía se está rindiendo. Congelar antes de
	// cerrar dejaría al alumno sin poder guardar su trabajo.
	ErrExamSessionOpen = errors.New("el examen sigue abierto; el congelamiento ocurre al cerrarlo")
	// ErrFreezeReasonRequired: reabrir un volumen es una excepción del docente y
	// necesita un motivo escrito para la auditoría.
	ErrFreezeReasonRequired = errors.New("reabrir el entorno de un alumno exige un motivo")
	// ErrFreezeStudentNotInSession: el alumno no participó del examen, así que no
	// tiene un laboratorio que reabrir en esa sesión.
	ErrFreezeStudentNotInSession = errors.New("el alumno no participó de ese examen")
)

const (
	// AuditExamPracticeEnvironmentFrozen y AuditExamPracticeEnvironmentUnfrozen registran el cierre y
	// la excepción individual sobre el aula, igual que el resto de acciones del
	// avance.
	AuditExamPracticeEnvironmentFrozen   = "EXAM_PRACTICE_ENVIRONMENT_FROZEN"
	AuditExamPracticeEnvironmentUnfrozen = "EXAM_PRACTICE_ENVIRONMENT_UNFROZEN"
)

// ExamFreezePracticeEnvironment es el laboratorio de un alumno dentro de una sesión de
// examen. Solo se usa para congelar y reabrir; no es la vista del aula.
type ExamFreezePracticeEnvironment struct {
	EnvironmentID string     `db:"environment_id"`
	StudentID   string     `db:"student_id"`
	Status      string     `db:"status"`
	FrozenAt    *time.Time `db:"frozen_at"`
}

// IsFrozen dice si el laboratorio ya quedó en solo lectura.
func (w ExamFreezePracticeEnvironment) IsFrozen() bool {
	return w.FrozenAt != nil || w.Status == PracticeEnvironmentStatusFrozen
}

// ExamSessionRef identifica un aula cerrada de examen que todavía tiene
// laboratorios sin congelar. El barrido la usa para no recorrer la tabla entera.
type ExamSessionRef struct {
	TenantID  string `db:"tenant_id"`
	SessionID string `db:"session_id"`
}

// ExamFreezeStore declara el almacenamiento del congelamiento de examen.
// Se mantiene aparte de ClassSessionStore y PracticeEnvironmentRepository para no obligar
// a sus dobles a implementar métodos que no usan.
type ExamFreezeStore interface {
	// SessionIsExam dice si la sesión está atada a una práctica con
	// purpose = exam y pertenece a la institución.
	SessionIsExam(ctx context.Context, tenantID, sessionID string) (bool, error)
	// ParticipantPracticeEnvironments devuelve los laboratorios de los alumnos que
	// participaron en la sesión.
	ParticipantPracticeEnvironments(ctx context.Context, tenantID, sessionID string) ([]ExamFreezePracticeEnvironment, error)
	// ClosedExamSessions lista aulas cerradas de examen con al menos un
	// laboratorio sin congelar, de la más vieja a la más nueva.
	ClosedExamSessions(ctx context.Context, limit int) ([]ExamSessionRef, error)
	// SetPracticeEnvironmentFrozen marca o limpia frozen_at y ajusta el estado del
	// laboratorio en la misma sentencia.
	SetPracticeEnvironmentFrozen(ctx context.Context, tenantID, environmentID string, frozen bool, at time.Time) error
}
