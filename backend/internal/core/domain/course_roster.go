package domain

import (
	"context"
	"errors"
	"time"
)

// Orígenes de una matrícula en `enrollments.source_system`. El origen dice
// quién mandó el dato: la nómina oficial del sistema académico o un alta de
// cátedra hecha por el docente para poder trabajar desde el primer día.
const (
	EnrollmentSourceInstitutional = "institutional_roster"
	EnrollmentSourceTeacherManual = "teacher_manual"
)

// ErrCourseForbidden se emite cuando el docente no dicta la materia sobre la
// que intenta operar. El handler lo traduce a HTTP 403.
var ErrCourseForbidden = errors.New("no tenés permiso sobre esta materia")

// CourseStudentItem es una fila del padrón de una materia: la persona, cuándo
// se matriculó, de dónde vino la matrícula y el estado de su entorno de
// práctica más reciente.
type CourseStudentItem struct {
	StudentID       string    `db:"student_id" json:"student_id"`
	FullName        string    `db:"full_name" json:"full_name"`
	Email           string    `db:"email" json:"email"`
	EnrolledAt      time.Time `db:"enrolled_at" json:"enrolled_at"`
	SourceSystem    string    `db:"source_system" json:"source_system"`
	WorkspaceStatus string    `db:"workspace_status" json:"workspace_status"`
}

// CourseAvailableStudent es un estudiante activo del tenant que todavía no
// pertenece a la materia.
type CourseAvailableStudent struct {
	ID        string `db:"id" json:"id"`
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name" json:"last_name"`
	Email     string `db:"email" json:"email"`
}

// CourseRosterRepository lee y escribe el padrón de una materia. Es la misma
// tabla `enrollments` para administración y docencia: lo que cambia es quién
// autoriza la escritura, no el dato.
type CourseRosterRepository interface {
	ListCourseStudents(ctx context.Context, tenantID, subjectID string) ([]*CourseStudentItem, error)
	ListAvailableStudents(ctx context.Context, tenantID, subjectID, query string, limit int) ([]*CourseAvailableStudent, error)
	TeacherOwnsSubject(ctx context.Context, tenantID, subjectID, teacherID string) (bool, error)
	SubjectExists(ctx context.Context, tenantID, subjectID string) (bool, error)
	EnrollStudentWithSource(ctx context.Context, tenantID, subjectID, studentID, source string) error
	RemoveStudentFromSubject(ctx context.Context, tenantID, subjectID, studentID string) error
}

// RosterSyncer aplica una nómina completa (alta de cuentas faltantes,
// matrícula y reconciliación acotada por origen). Lo implementa el servicio de
// gobernanza académica; el padrón de curso lo reutiliza en vez de duplicar el
// parser ni la transacción.
type RosterSyncer interface {
	RosterEnrollFromSource(ctx context.Context, tenantID, subjectID string, req RosterSyncRequest) (*RosterEnrollResult, error)
}
