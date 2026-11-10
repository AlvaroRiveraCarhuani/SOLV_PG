package domain

import (
	"encoding/json"
	"time"
)

type Subject struct {
	ID       string `db:"id" json:"id"`
	TenantID string `db:"tenant_id" json:"tenant_id"`
	Name     string `db:"name" json:"name"`
	Code     string `db:"code" json:"code"`
	// CatalogID identifica la materia en el plan de estudios, más allá del
	// periodo en que se dicta. Name y Code son el snapshot de este curso.
	CatalogID *string `db:"subject_catalog_id" json:"subject_catalog_id,omitempty"`
	// Section es el paralelo o comisión dentro del periodo; vacío si hay una
	// sola oferta.
	Section           *string   `db:"section" json:"section,omitempty"`
	TeacherID         *string   `db:"teacher_id" json:"teacher_id,omitempty"`
	TeacherName       *string   `db:"teacher_name" json:"teacher_name,omitempty"`
	AcademicPeriodID  *string   `db:"academic_period_id" json:"academic_period_id,omitempty"`
	IsArchived        bool      `db:"is_archived" json:"is_archived"`
	ClassroomCourseID *string   `db:"classroom_course_id" json:"classroom_course_id,omitempty"`
	TemplateID        *string   `db:"template_id" json:"template_id,omitempty"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type Enrollment struct {
	ID         string    `db:"id" json:"id"`
	TenantID   string    `db:"tenant_id" json:"tenant_id"`
	StudentID  string    `db:"student_id" json:"student_id"`
	SubjectID  string    `db:"subject_id" json:"subject_id"`
	EnrolledAt time.Time `db:"enrolled_at" json:"enrolled_at"`
}

type Submission struct {
	ID            string  `db:"id" json:"id"`
	TenantID      string  `db:"tenant_id" json:"tenant_id"`
	ExerciseID    string  `db:"exercise_id" json:"exercise_id"`
	StudentID     string  `db:"student_id" json:"student_id"`
	EnvironmentID *string `db:"environment_id" json:"environment_id,omitempty"`
	Code          string  `db:"code" json:"code"`
	Verdict       string  `db:"verdict" json:"verdict"`
	// VerdictOrigin declara quién escribió el veredicto: el juez, la rúbrica,
	// un ajuste docente, o nada (fila histórica no atribuible).
	VerdictOrigin      string          `db:"verdict_origin" json:"verdict_origin,omitempty"`
	ReviewStatus       string          `db:"review_status" json:"review_status"`
	ASTResult          json.RawMessage `db:"ast_result" json:"ast_result"`
	GeneratedCases     json.RawMessage `db:"generated_cases" json:"generated_cases,omitempty"`
	ComplexityAnalysis json.RawMessage `db:"complexity_analysis" json:"complexity_analysis,omitempty"`
	ExecutionTimeMS    int             `db:"execution_time_ms" json:"execution_time_ms"`
	MemoryUsedMB       int             `db:"memory_used_mb" json:"memory_used_mb"`
	ManualOverride     bool            `db:"manual_override" json:"manual_override"`
	OverrideReason     string          `db:"override_reason" json:"override_reason"`
	Score              *int            `db:"score" json:"score"`
	GradedBy           *string         `db:"graded_by" json:"graded_by,omitempty"`
	// BaseStatus clasifica la base de comparación de una entrega IDE:
	// version, no_base o legacy_unknown. El juez la deja nula.
	BaseStatus                string    `db:"base_status" json:"base_status,omitempty"`
	WorkspaceProjectVersionID *string   `db:"workspace_project_version_id" json:"workspace_project_version_id,omitempty"`
	SubmittedAt               time.Time `db:"submitted_at" json:"submitted_at"`
}

type TeacherInvitation struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"tenant_id"`
	Token     string    `db:"token" json:"token"`
	Email     string    `db:"email" json:"email"`
	Origin    string    `db:"origin" json:"origin"`
	RoleType  string    `db:"role_type" json:"role_type"`
	Used      bool      `db:"used" json:"used"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type TeacherListItem struct {
	ID            string  `db:"id" json:"id"`
	FullName      string  `db:"full_name" json:"full_name"`
	Email         string  `db:"email" json:"email"`
	Origin        string  `db:"origin" json:"origin"`
	Status        string  `db:"status" json:"status"` // "active", "pending", "expired"
	RoleType      string  `db:"role_type" json:"role_type"`
	InvitedAt     string  `db:"invited_at" json:"invited_at"`
	LastLogin     *string `db:"last_login" json:"last_login,omitempty"`
	ActiveCourses int     `db:"active_courses" json:"active_courses"`
	Token         string  `db:"token" json:"token,omitempty"`
}

type TeacherCourseItem struct {
	ID            string `db:"id" json:"id"`
	Name          string `db:"name" json:"name"`
	Code          string `db:"code" json:"code"`
	StudentsCount int    `db:"students_count" json:"students_count"`
}
