package domain

import "time"

// TeacherCourseSummary representa una materia del docente con sus contadores de cohorte.
type TeacherCourseSummary struct {
	ID               string  `json:"id" db:"id"`
	Name             string  `json:"name" db:"name"`
	Code             string  `json:"code" db:"code"`
	AcademicPeriodID *string `json:"academic_period_id,omitempty" db:"academic_period_id"`
	StudentsCount    int     `json:"students_count" db:"students_count"`
	ActiveNow        int     `json:"active_now" db:"active_now"`
	PendingReview    int     `json:"pending_review" db:"pending_review"`
	AtRisk           int     `json:"at_risk" db:"at_risk"`
}

// AttentionCriticalAlert representa una alerta crítica de infraestructura o fallo grave.
type AttentionCriticalAlert struct {
	Type        string    `json:"type"` // "oom_killed"
	StudentID   string    `json:"student_id"`
	StudentName string    `json:"student_name"`
	WorkspaceID string    `json:"workspace_id"`
	SubjectID   string    `json:"subject_id"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// AttentionWarningAlert representa una alerta de advertencia académica/AST.
type AttentionWarningAlert struct {
	Type         string    `json:"type"` // "ast_blocked"
	StudentID    string    `json:"student_id"`
	StudentName  string    `json:"student_name"`
	ExerciseID   string    `json:"exercise_id"`
	RuleViolated string    `json:"rule_violated"`
	OccurredAt   time.Time `json:"occurred_at"`
}

// AttentionStandardAlert representa una entrega pendiente de revisión manual docente.
type AttentionStandardAlert struct {
	Type          string    `json:"type"` // "pending_review"
	SubmissionID  string    `json:"submission_id"`
	StudentName   string    `json:"student_name"`
	ExerciseTitle string    `json:"exercise_title"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

// TeacherAttentionWidget agrupa las alertas clasificadas por nivel de severidad.
type TeacherAttentionWidget struct {
	Critical []AttentionCriticalAlert `json:"critical"`
	Warning  []AttentionWarningAlert  `json:"warning"`
	Standard []AttentionStandardAlert `json:"standard"`
}

// TeacherLabStats consolida el estado académico y veredictos por laboratorio de un curso.
type TeacherLabStats struct {
	ID               string         `json:"id" db:"id"`
	Title            string         `json:"title" db:"title"`
	Status           string         `json:"status" db:"status"`
	DueDate          *time.Time     `json:"due_date,omitempty" db:"due_date"`
	SubmissionsCount int            `json:"submissions_count" db:"submissions_count"`
	StudentsCount    int            `json:"students_count" db:"students_count"`
	AutoGraded       int            `json:"auto_graded" db:"auto_graded"`
	PendingReview    int            `json:"pending_review" db:"pending_review"`
	AtRisk           int            `json:"at_risk" db:"at_risk"`
	VerdictsSummary  map[string]int `json:"verdicts_summary"`
}
