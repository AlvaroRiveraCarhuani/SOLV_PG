package domain

import "time"

// ReassignCourseDTO DTO para reasignar la titularidad de una materia (ADR-036)
type ReassignCourseDTO struct {
	NewTeacherID string `json:"new_teacher_id" validate:"required"`
	Reason       string `json:"reason,omitempty"`
}

// AdminStudentDirectoryItem DTO para el listado institucional de estudiantes (ADR-033)
type AdminStudentDirectoryItem struct {
	ID                    string     `db:"id" json:"id"`
	FirstName             string     `db:"first_name" json:"first_name"`
	LastName              string     `db:"last_name" json:"last_name"`
	Email                 string     `db:"email" json:"email"`
	Role                  string     `db:"role" json:"role"`
	Status                string     `db:"status" json:"status"`
	SuspensionReason      *string    `db:"suspension_reason" json:"suspension_reason,omitempty"`
	AcademicStatus        string     `db:"academic_status" json:"academic_status"`
	EnrolledCoursesCount  int        `db:"enrolled_courses_count" json:"enrolled_courses_count"`
	ActiveWorkspacesCount int        `db:"active_workspaces_count" json:"active_workspaces_count"`
	OOMStrikeCount        int        `db:"oom_strike_count" json:"oom_strike_count"`
	LastOOMKilledAt       *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
}

// UpdateStudentStatusDTO DTO para suspender o reactivar una cuenta de estudiante
type UpdateStudentStatusDTO struct {
	Status string `json:"status" validate:"required"` // "active" | "suspended"
	Reason string `json:"reason,omitempty"`
}

// CreateStudentDTO DTO para el alta manual de un estudiante por el administrador
type CreateStudentDTO struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}

// AdminStudentCourseItem DTO para el detalle de materias y espacios de trabajo del estudiante
type AdminStudentCourseItem struct {
	SubjectID       string     `db:"subject_id" json:"subject_id"`
	SubjectCode     string     `db:"subject_code" json:"subject_code"`
	SubjectName     string     `db:"subject_name" json:"subject_name"`
	TeacherName     string     `db:"teacher_name" json:"teacher_name"`
	EnrolledAt      time.Time  `db:"enrolled_at" json:"enrolled_at"`
	WorkspaceID     *string    `db:"workspace_id" json:"workspace_id,omitempty"`
	WorkspaceStatus *string    `db:"workspace_status" json:"workspace_status,omitempty"`
	MemoryLimitMB   *int       `db:"memory_limit_mb" json:"memory_limit_mb,omitempty"`
	OOMStrikeCount  *int       `db:"oom_strike_count" json:"oom_strike_count,omitempty"`
	LastOOMKilledAt *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
}

// ResetOOMDTO DTO para solicitar el reseteo manual de penalizaciones por OOM-Killed (ADR-033)
type ResetOOMDTO struct {
	Reason string `json:"reason" validate:"required,min=10"`
}

// ResetOOMResult resultado tras resetear strikes OOM
type ResetOOMResult struct {
	StudentID            string `json:"student_id"`
	WorkspacesResetCount int64  `json:"workspaces_reset_count"`
	Message              string `json:"message"`
}

// ReviewTemplateDTO DTO para aprobar o rechazar plantillas Docker (ADR-030)
type ReviewTemplateDTO struct {
	Status          string `json:"status" validate:"required"` // "approved" | "rejected" | "paused"
	RejectionReason string `json:"rejection_reason,omitempty"`
	BaseRamMB       *int   `json:"base_ram_mb,omitempty"`
}

// CreateOfficialTemplateDTO DTO para que el administrador registre directamente una plantilla oficial
type CreateOfficialTemplateDTO struct {
	Name        string `json:"name" validate:"required"`
	DockerImage string `json:"docker_image" validate:"required"`
	BaseRamMB   int    `json:"base_ram_mb" validate:"required,gt=0"`
	Description string `json:"description"`
}

// AdminTemplateReviewItem modelo para listar y gestionar plantillas Docker institucionales (ADR-030)
type AdminTemplateReviewItem struct {
	ID              string     `db:"id" json:"id"`
	TenantID        *string    `db:"tenant_id" json:"tenant_id,omitempty"`
	Name            string     `db:"name" json:"name"`
	DockerImage     string     `db:"docker_image" json:"docker_image"`
	BaseRamMB       int        `db:"base_ram_mb" json:"base_ram_mb"`
	Status          string     `db:"status" json:"status"`
	RejectionReason string     `db:"rejection_reason" json:"rejection_reason"`
	ReviewedBy      *string    `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `db:"reviewed_at" json:"reviewed_at,omitempty"`
	RequestedBy     *string    `db:"requested_by" json:"requested_by,omitempty"`
	RequestedByName *string    `db:"requested_by_name" json:"requested_by_name,omitempty"`
	Description     string     `db:"description" json:"description"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

// EmergencyActionRequest DTO para solicitar una acción de emergencia administrativa (ADR-032)
type EmergencyActionRequest struct {
	ConfirmationPhrase string `json:"confirmation_phrase" validate:"required"`
	Reason             string `json:"reason,omitempty"`
}

// EmergencyActionResult DTO con el resultado de la acción de emergencia
type EmergencyActionResult struct {
	Action        string `json:"action"`
	AffectedCount int64  `json:"affected_count"`
	ExecutedBy    string `json:"executed_by"`
	Message       string `json:"message"`
}
