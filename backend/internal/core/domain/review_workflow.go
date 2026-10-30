package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ReviewStatus es el estado explícito de revisión de una entrega. Reemplaza la
// deducción por veredicto: el trabajo manual del docente es un hecho declarado
// y no una consecuencia de que el juez haya devuelto WA, RE o TLE.
type ReviewStatus string

const (
	// ReviewNotRequired: el juez calificó solo y no espera corrección manual.
	ReviewNotRequired ReviewStatus = "not_required"
	// ReviewPending: la entrega espera la corrección del docente.
	ReviewPending ReviewStatus = "pending"
	// ReviewInReview: el docente la abrió y está trabajando en ella.
	ReviewInReview ReviewStatus = "in_review"
	// ReviewGraded: tiene nota y feedback del docente.
	ReviewGraded ReviewStatus = "graded"
	// ReviewReturned: se devolvió al alumno con feedback.
	ReviewReturned ReviewStatus = "returned"
)

var (
	ErrInvalidReviewTransition = errors.New("review status transition is not allowed")
	ErrInvalidReviewStatus     = errors.New("unknown review status")
	ErrGradeReasonRequired     = errors.New("a reason of at least 10 characters is required to change the grade")
	ErrGradeCellNotFound       = errors.New("grade cell not found")
	ErrSubmissionNotInSubject  = errors.New("submission does not belong to the teacher's subject")
	ErrReviewStoreUnavailable  = errors.New("review store is not configured")
	ErrInvalidScoreRange       = errors.New("score must be between 0 and 100")
)

var validReviewStatuses = map[ReviewStatus]bool{
	ReviewNotRequired: true,
	ReviewPending:     true,
	ReviewInReview:    true,
	ReviewGraded:      true,
	ReviewReturned:    true,
}

// reviewTransitions declara las transiciones permitidas. La identidad se acepta
// siempre para que reintentar la misma petición sea idempotente.
var reviewTransitions = map[ReviewStatus]map[ReviewStatus]bool{
	ReviewPending: {
		ReviewInReview: true,
		ReviewGraded:   true,
	},
	ReviewInReview: {
		ReviewPending: true,
		ReviewGraded:  true,
	},
	ReviewGraded: {
		ReviewInReview: true,
		ReviewReturned: true,
	},
	ReviewReturned: {
		ReviewInReview: true,
		ReviewGraded:   true,
	},
}

// ParseReviewStatus normaliza y valida un estado recibido por la API.
func ParseReviewStatus(raw string) (ReviewStatus, error) {
	status := ReviewStatus(strings.TrimSpace(strings.ToLower(raw)))
	if !validReviewStatuses[status] {
		return "", ErrInvalidReviewStatus
	}
	return status, nil
}

// CanTransitionReview indica si el cambio de estado está permitido.
func CanTransitionReview(from, to ReviewStatus) bool {
	if from == to {
		return true
	}
	if from == ReviewNotRequired {
		return false
	}
	return reviewTransitions[from][to]
}

// ValidateReviewTransition devuelve el error tipado si el cambio no es válido.
func ValidateReviewTransition(from, to ReviewStatus) error {
	if !CanTransitionReview(from, to) {
		return ErrInvalidReviewTransition
	}
	return nil
}

// ExerciseGradeCell es la nota declarada de un alumno en una práctica: qué
// entrega cuenta, con qué nota y de dónde salió esa decisión.
type ExerciseGradeCell struct {
	SelectedSubmissionID *string `json:"selected_submission_id,omitempty"`
	Score                *int    `json:"score"`
	Source               string  `json:"source"`
	Reason               string  `json:"reason,omitempty"`
}

// TeacherReviewStore agrupa las operaciones de revisión y libreta. Se declara
// en el consumidor (patrón del proyecto) para no engordar TeacherRepository y
// no romper los dobles de test existentes.
type TeacherReviewStore interface {
	ListPendingReviewGroups(ctx context.Context, tenantID, teacherID, subjectID string) ([]*ReviewQueueGroup, error)
	GetSubmissionReviewState(ctx context.Context, tenantID, teacherID, submissionID string) (ReviewStatus, error)
	UpdateSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string, current, next ReviewStatus, feedback string, score *int, actorID string) error
	RecordGradeFromSubmission(ctx context.Context, tenantID, submissionID string) error
	SetExerciseGrade(ctx context.Context, tenantID, teacherID, studentID, exerciseID, selectedSubmissionID string, requestedScore *int, reason, actorID string) (*int, error)
	GetExerciseGradeCell(ctx context.Context, tenantID, studentID, exerciseID string) (*ExerciseGradeCell, error)
}

// ReviewQueueGroup agrupa las entregas que esperan corrección por práctica.
type ReviewQueueGroup struct {
	ExerciseID        string    `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle     string    `json:"exercise_title" db:"exercise_title"`
	SubjectID         string    `json:"subject_id" db:"subject_id"`
	SubjectName       string    `json:"subject_name" db:"subject_name"`
	EnvironmentType   string    `json:"environment_type" db:"environment_type"`
	Pending           int       `json:"pending" db:"pending"`
	Total             int       `json:"total" db:"total"`
	OldestSubmittedAt time.Time `json:"oldest_submitted_at" db:"oldest_submitted_at"`
}

// UpdateReviewRequest es el cuerpo de PATCH /teacher/submissions/{id}/review.
type UpdateReviewRequest struct {
	Status          string `json:"status"`
	FeedbackSummary string `json:"feedback_summary"`
	Score           *int   `json:"score,omitempty"`
	OverrideReason  string `json:"override_reason,omitempty"`
}

// SetExerciseGradeRequest es el cuerpo de la selección de la entrega que cuenta.
type SetExerciseGradeRequest struct {
	SelectedSubmissionID string `json:"selected_submission_id"`
	Score                *int   `json:"score,omitempty"`
	Reason               string `json:"reason,omitempty"`
}

// ResolvedSubmissionScore aplica la regla única de nota automática: la nota
// explícita manda; si no hay, un AC o una anulación manual valen 100. Devuelve
// nil cuando la entrega no está calificada, para que nadie la confunda con 0.
func ResolvedSubmissionScore(score *int, verdict string, manualOverride bool) *int {
	if score != nil {
		return score
	}
	if strings.EqualFold(strings.TrimSpace(verdict), "AC") || manualOverride {
		full := 100
		return &full
	}
	return nil
}

// NeedsGradeReason exige justificación cuando la nota pedida contradice la
// calificación que ya tenía la entrega elegida.
func NeedsGradeReason(selectedScore *int, requestedScore *int) bool {
	if requestedScore == nil {
		return false
	}
	if selectedScore == nil {
		return true
	}
	return *selectedScore != *requestedScore
}
