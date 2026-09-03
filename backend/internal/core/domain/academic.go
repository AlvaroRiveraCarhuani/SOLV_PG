package domain

import (
	"encoding/json"
	"time"
)

type Subject struct {
	ID                string    `db:"id" json:"id"`
	TenantID          string    `db:"tenant_id" json:"tenant_id"`
	Name              string    `db:"name" json:"name"`
	Code              string    `db:"code" json:"code"`
	TeacherID         *string   `db:"teacher_id" json:"teacher_id,omitempty"`
	AcademicPeriodID  *string   `db:"academic_period_id" json:"academic_period_id,omitempty"`
	IsArchived        bool      `db:"is_archived" json:"is_archived"`
	ClassroomCourseID *string   `db:"classroom_course_id" json:"classroom_course_id,omitempty"`
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
	ID              string          `db:"id" json:"id"`
	TenantID        string          `db:"tenant_id" json:"tenant_id"`
	ExerciseID      string          `db:"exercise_id" json:"exercise_id"`
	StudentID       string          `db:"student_id" json:"student_id"`
	WorkspaceID     *string         `db:"workspace_id" json:"workspace_id,omitempty"`
	Code            string          `db:"code" json:"code"`
	Verdict         string          `db:"verdict" json:"verdict"`
	ASTResult       json.RawMessage `db:"ast_result" json:"ast_result"`
	ExecutionTimeMS int             `db:"execution_time_ms" json:"execution_time_ms"`
	MemoryUsedMB    int             `db:"memory_used_mb" json:"memory_used_mb"`
	ManualOverride  bool            `db:"manual_override" json:"manual_override"`
	OverrideReason  string          `db:"override_reason" json:"override_reason"`
	Score           *int            `db:"score" json:"score"`
	GradedBy        *string         `db:"graded_by" json:"graded_by,omitempty"`
	SubmittedAt     time.Time       `db:"submitted_at" json:"submitted_at"`
}

type TeacherInvitation struct {
	ID        string    `db:"id" json:"id"`
	TenantID  string    `db:"tenant_id" json:"tenant_id"`
	Token     string    `db:"token" json:"token"`
	Email     string    `db:"email" json:"email"`
	Used      bool      `db:"used" json:"used"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
