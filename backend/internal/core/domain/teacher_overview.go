package domain

import (
	"context"
	"time"
)

// TeacherOverview es el read model del panel docente: responde una sola pregunta
// ("¿qué me espera hoy?") con los conteos ya resueltos. Ninguna cifra de este
// contrato se recalcula en el cliente.
type TeacherOverview struct {
	Period       *TeacherOverviewPeriod      `json:"period"`
	Session      *TeacherOverviewSession     `json:"session"`
	NextDue      *TeacherOverviewNextDue     `json:"next_due"`
	Queue        TeacherOverviewQueue        `json:"queue"`
	Environments TeacherOverviewEnvironments `json:"environments"`
	Subjects     []TeacherOverviewSubject    `json:"subjects"`
	Signals      []TeacherOverviewSignal     `json:"signals"`
}

// TeacherOverviewNextDue es la práctica publicada con el vencimiento más cercano,
// resuelta en el servidor para que la vista no elija entre filas.
type TeacherOverviewNextDue struct {
	SubjectID     string    `json:"subject_id"`
	SubjectName   string    `json:"subject_name"`
	ExerciseTitle string    `json:"exercise_title"`
	DueAt         time.Time `json:"due_at"`
}

// TeacherOverviewPeriod identifica el período académico que enmarca la pantalla.
type TeacherOverviewPeriod struct {
	ID       string `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	Code     string `json:"code" db:"code"`
	IsActive bool   `json:"is_active" db:"is_active"`
}

// TeacherOverviewSession describe el aula abierta. Es nulo mientras no exista
// una sesión de clase para el docente.
type TeacherOverviewSession struct {
	ID            string     `json:"id"`
	SubjectID     string     `json:"subject_id"`
	SubjectName   string     `json:"subject_name"`
	ExerciseTitle string     `json:"exercise_title,omitempty"`
	OpensAt       time.Time  `json:"opens_at"`
	ClosesAt      *time.Time `json:"closes_at"`
	Enrolled      int        `json:"enrolled"`
	Present       int        `json:"present"`
}

// TeacherOverviewQueue cuenta el trabajo de corrección pendiente.
type TeacherOverviewQueue struct {
	Pending           int        `json:"pending"`
	Practices         int        `json:"practices"`
	OldestSubmittedAt *time.Time `json:"oldest_submitted_at"`
}

// TeacherOverviewEnvironments cuenta los entornos abiertos: contenedores en
// estado running, active o pending. La presencia con heartbeat pertenece a la
// sesión de aula y se resuelve cuando exista.
type TeacherOverviewEnvironments struct {
	OpenNow int `json:"open_now"`
}

// TeacherOverviewSubject es una fila de la tabla de materias. WithoutSubmission
// es nulo cuando la materia no tiene una práctica por vencer, porque en ese
// caso no hay conjunto contra el cual contar.
type TeacherOverviewSubject struct {
	ID                string     `json:"id" db:"id"`
	Code              string     `json:"code" db:"code"`
	Name              string     `json:"name" db:"name"`
	Enrolled          int        `json:"enrolled" db:"enrolled"`
	OpenEnvironments  int        `json:"open_environments" db:"open_environments"`
	PendingReview     int        `json:"pending_review" db:"pending_review"`
	WithoutSubmission *int       `json:"without_submission" db:"without_submission"`
	NextDueAt         *time.Time `json:"next_due_at" db:"next_due_at"`
}

// TeacherOverviewSignal es una excepción con dueño docente. Queda vacío hasta
// que exista la sesión de aula que le da dueño a los avisos de clase.
type TeacherOverviewSignal struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// TeacherOverviewStore declara el read model del panel en el consumidor, para
// que los dobles existentes no tengan que implementarlo.
type TeacherOverviewStore interface {
	GetTeacherOverview(ctx context.Context, tenantID, teacherID, periodID string) (*TeacherOverview, error)
}
