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
	EnrolledCoursesCount  int        `db:"enrolled_courses_count" json:"enrolled_courses_count"`
	ActiveWorkspacesCount int        `db:"active_workspaces_count" json:"active_workspaces_count"`
	OOMStrikeCount        int        `db:"oom_strike_count" json:"oom_strike_count"`
	LastOOMKilledAt       *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
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
	Status          string `json:"status" validate:"required"` // "approved" | "rejected"
	RejectionReason string `json:"rejection_reason,omitempty"`
	BaseRamMB       *int   `json:"base_ram_mb,omitempty"`
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
	Description     string     `db:"description" json:"description"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}
