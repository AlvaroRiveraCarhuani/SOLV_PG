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
