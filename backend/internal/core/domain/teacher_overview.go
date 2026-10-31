package domain

import (
	"context"
	"time"
)

// TeacherOverview es el read model del panel docente: responde una sola pregunta
// ("¿qué me espera hoy?") con los conteos ya resueltos. Ninguna cifra de este
// contrato se recalcula en el cliente.
type TeacherOverview struct {
	Period       *TeacherOverviewPeriod   `json:"period"`
	Session      *TeacherOverviewSession  `json:"session"`
	Queue        TeacherOverviewQueue     `json:"queue"`
	DueSoon       []TeacherOverviewDue     `json:"due_soon"`
	DueSoonTotal  int                      `json:"due_soon_total"`
	DueSoonHidden int                      `json:"due_soon_hidden"`
	Subjects     []TeacherOverviewSubject `json:"subjects"`
	Signals      []TeacherOverviewSignal  `json:"signals"`
}

// TeacherOverviewDue es una práctica publicada que vence dentro de la ventana de
// la columna de pendientes. El servidor la ordena y la limita, y declara el
// total para que la vista no tenga que contar ni recortar.
type TeacherOverviewDue struct {
	SubjectID     string    `json:"subject_id" db:"subject_id"`
	SubjectCode   string    `json:"subject_code" db:"subject_code"`
	SubjectName   string    `json:"subject_name" db:"subject_name"`
	ExerciseID    string    `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle string    `json:"exercise_title" db:"exercise_title"`
	DueAt         time.Time `json:"due_at" db:"due_at"`
	// Urgency es overdue | today | soon | later | none, calculada en el servidor.
	Urgency string `json:"urgency" db:"urgency"`
	// PendingReview cuenta las entregas por calificar de la práctica.
	PendingReview int `json:"pending_review" db:"pending_review"`
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
	SubjectCode   string     `json:"subject_code"`
	SubjectName   string     `json:"subject_name"`
	ExerciseTitle string     `json:"exercise_title,omitempty"`
	OpensAt       time.Time  `json:"opens_at"`
	EndsAt        time.Time  `json:"ends_at"`
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

// TeacherOverviewSubject es una fila de la tabla de materias. WithoutSubmission
// es nulo cuando la materia no tiene una práctica por vencer, porque en ese
// caso no hay conjunto contra el cual contar.
type TeacherOverviewSubject struct {
	ID                string     `json:"id" db:"id"`
	Code              string     `json:"code" db:"code"`
	Name              string     `json:"name" db:"name"`
	Enrolled          int        `json:"enrolled" db:"enrolled"`
	PendingReview     int        `json:"pending_review" db:"pending_review"`
	WithoutSubmission *int       `json:"without_submission" db:"without_submission"`
	NextDueAt         *time.Time `json:"next_due_at" db:"next_due_at"`
	NextDueTitle      string     `json:"next_due_title,omitempty" db:"next_due_title"`
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
