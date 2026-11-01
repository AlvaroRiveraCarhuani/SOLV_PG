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

// TeacherExerciseBankItem es el read model del banco de ejercicios del docente:
// consolida datos del laboratorio y métricas de entrega a través de todas sus
// materias. La modalidad se deriva de EnvironmentType (JUEZ_EFIMERO |
// IDE_PERSISTENTE); ReviewPending usa la semántica de cada entorno (veredictos
// por revisar en el juez, entregas PENDING en la cola de SpeedGrader en IDE).
type TeacherExerciseBankItem struct {
	ID               string     `json:"id" db:"id"`
	Title            string     `json:"title" db:"title"`
	Status           string     `json:"status" db:"status"`
	Type             string     `json:"type" db:"type"`
	EnvironmentType  string     `json:"environment_type" db:"environment_type"`
	Language         string     `json:"language" db:"language"`
	MemoryLimitMB    int        `json:"memory_limit_mb" db:"memory_limit_mb"`
	TimeLimitMS      int        `json:"time_limit_ms" db:"time_limit_ms"`
	DueDate          *time.Time `json:"due_date,omitempty" db:"due_date"`
	SubjectID        string     `json:"subject_id" db:"subject_id"`
	SubjectCode      string     `json:"subject_code" db:"subject_code"`
	SubjectName      string     `json:"subject_name" db:"subject_name"`
	SubmissionsCount int        `json:"submissions_count" db:"submissions_count"`
	StudentsCount    int        `json:"students_count" db:"students_count"`
	ReviewPending    int        `json:"review_pending" db:"review_pending"`
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
