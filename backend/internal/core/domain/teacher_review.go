package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// SubmissionComment representa un comentario pedagógico docente anclado a una línea de código.
type SubmissionComment struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	SubmissionID string    `json:"submission_id" db:"submission_id"`
	AuthorID     string    `json:"author_id" db:"author_id"`
	AuthorName   string    `json:"author_name" db:"author_name"`
	LineNumber   int       `json:"line_number" db:"line_number"`
	Comment      string    `json:"comment" db:"comment"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SubmissionQueueItem representa un elemento en la cola de revisión del curso.
type SubmissionQueueItem struct {
	ID              string     `json:"id" db:"id"`
	ExerciseID      string     `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle   string     `json:"exercise_title" db:"exercise_title"`
	StudentID       string     `json:"student_id" db:"student_id"`
	StudentName     string     `json:"student_name" db:"student_name"`
	StudentEmail    string     `json:"student_email" db:"student_email"`
	Verdict         string     `json:"verdict" db:"verdict"`
	Score           *int       `json:"score,omitempty" db:"score"`
	ManualOverride  bool       `json:"manual_override" db:"manual_override"`
	ExecutionTimeMS int        `json:"execution_time_ms" db:"execution_time_ms"`
	MemoryUsedMB    int        `json:"memory_used_mb" db:"memory_used_mb"`
	SubmittedAt     time.Time  `json:"submitted_at" db:"submitted_at"`
	CommentsCount   int        `json:"comments_count" db:"comments_count"`
}

// TestCaseReview representa un caso de prueba desenmascarado para la vista docente.
type TestCaseReview struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	IsHidden       bool   `json:"is_hidden"`
	ActualOutput   string `json:"actual_output,omitempty"`
	Passed         bool   `json:"passed"`
}

// TeacherSubmissionReviewDTO representa el DTO completo de la vista SpeedGrader para el docente.
type TeacherSubmissionReviewDTO struct {
	ID               string              `json:"id" db:"id"`
	ExerciseID       string              `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle    string              `json:"exercise_title" db:"exercise_title"`
	SubjectID        string              `json:"subject_id" db:"subject_id"`
	SubjectName      string              `json:"subject_name" db:"subject_name"`
	StudentID        string              `json:"student_id" db:"student_id"`
	StudentName      string              `json:"student_name" db:"student_name"`
	StudentEmail     string              `json:"student_email" db:"student_email"`
	Code             string              `json:"code" db:"code"`
	Verdict          string              `json:"verdict" db:"verdict"`
	Score            *int                `json:"score,omitempty" db:"score"`
	ManualOverride   bool                `json:"manual_override" db:"manual_override"`
	OverrideReason   string              `json:"override_reason,omitempty" db:"override_reason"`
	GradedBy         *string             `json:"graded_by,omitempty" db:"graded_by"`
	GradedByName     string              `json:"graded_by_name,omitempty" db:"graded_by_name"`
	ExecutionTimeMS  int                 `json:"execution_time_ms" db:"execution_time_ms"`
	MemoryUsedMB     int                 `json:"memory_used_mb" db:"memory_used_mb"`
	ASTResult        json.RawMessage     `json:"ast_result" db:"ast_result"`
	TestCases        []TestCaseReview    `json:"test_cases"`
	Comments         []SubmissionComment `json:"comments"`
	NextSubmissionID *string             `json:"next_submission_id,omitempty"`
	PrevSubmissionID *string             `json:"prev_submission_id,omitempty"`
	SubmittedAt      time.Time           `json:"submitted_at" db:"submitted_at"`
}

// OverrideRequestDTO payload para anular o convalidar manualmente una calificación.
type OverrideRequestDTO struct {
	Verdict        string `json:"verdict"`
	OverrideReason string `json:"override_reason"`
	Score          *int   `json:"score,omitempty"`
}

// AddCommentRequestDTO payload para agregar un comentario in-line.
type AddCommentRequestDTO struct {
	LineNumber int    `json:"line_number"`
	Comment    string `json:"comment"`
}

var ErrInvalidOverrideReason = errors.New("override reason must be at least 10 characters")

// EphemeralRunRequestDTO payload para ejecutar código en sandbox efímero sin persistir.
type EphemeralRunRequestDTO struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// EphemeralRunResult resultado en memoria de la ejecución efímera.
type EphemeralRunResult struct {
	SubmissionID    string `json:"submission_id"`
	ExerciseID      string `json:"exercise_id"`
	Verdict         string `json:"verdict"`
	ExecutionTimeMS int    `json:"execution_time_ms"`
	MemoryUsedMB    int    `json:"memory_used_mb"`
	Message         string `json:"message"`
	ActualJSON      string `json:"actual_json,omitempty"`
}

// CourseExerciseHeader cabecera de laboratorio para la matriz de calificaciones.
type CourseExerciseHeader struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// StudentGradesRow fila de notas por estudiante en el curso.
type StudentGradesRow struct {
	StudentID    string         `json:"student_id"`
	StudentName  string         `json:"student_name"`
	StudentEmail string         `json:"student_email"`
	Grades       map[string]int `json:"grades"`
	Average      float64        `json:"average"`
}

// CourseGradesMatrix estructura consolidada para la exportación de notas del curso.
type CourseGradesMatrix struct {
	SubjectID   string                 `json:"subject_id"`
	SubjectName string                 `json:"subject_name"`
	SubjectCode string                 `json:"subject_code"`
	Exercises   []CourseExerciseHeader `json:"exercises"`
	Students    []StudentGradesRow     `json:"students"`
}

// SubmissionForPlagiarism representa la entrega normalizada para el motor anti-plagio AST.
type SubmissionForPlagiarism struct {
	SubmissionID  string    `json:"submission_id" db:"submission_id"`
	ExerciseID    string    `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle string    `json:"exercise_title" db:"exercise_title"`
	StudentID     string    `json:"student_id" db:"student_id"`
	StudentName   string    `json:"student_name" db:"student_name"`
	StudentEmail  string    `json:"student_email" db:"student_email"`
	Language      string    `json:"language" db:"language"`
	Code          string    `json:"code" db:"code"`
	Verdict       string    `json:"verdict" db:"verdict"`
	SubmittedAt   time.Time `json:"submitted_at" db:"submitted_at"`
}

// PlagiarismMatch representa un par de entregas con similitud estructural AST por encima del umbral.
type PlagiarismMatch struct {
	SubmissionIDA    string   `json:"submission_id_a"`
	StudentIDA       string   `json:"student_id_a"`
	StudentNameA     string   `json:"student_name_a"`
	SubmissionIDB    string   `json:"submission_id_b"`
	StudentIDB       string   `json:"student_id_b"`
	StudentNameB     string   `json:"student_name_b"`
	ExerciseID       string   `json:"exercise_id"`
	ExerciseTitle    string   `json:"exercise_title"`
	Similarity       float64  `json:"similarity"`
	RiskLevel        string   `json:"risk_level"`
	MatchingTokens   int      `json:"matching_tokens"`
	TotalTokensA     int      `json:"total_tokens_a"`
	TotalTokensB     int      `json:"total_tokens_b"`
	CommonStructures []string `json:"common_structures"`
}

// PlagiarismReport reporte consolidado de similitud estructural por ejercicio o curso.
type PlagiarismReport struct {
	SubjectID         string            `json:"subject_id"`
	SubjectName       string            `json:"subject_name,omitempty"`
	ExerciseID        string            `json:"exercise_id,omitempty"`
	ExerciseTitle     string            `json:"exercise_title,omitempty"`
	AnalyzedAt        time.Time         `json:"analyzed_at"`
	TotalSubmissions  int               `json:"total_submissions"`
	SuspectPairsCount int               `json:"suspect_pairs_count"`
	Matches           []PlagiarismMatch `json:"matches"`
}

// TimelineKeyframe representa un snapshot o delta en la línea de tiempo de escritura del código.
type TimelineKeyframe struct {
	OffsetMS   int    `json:"offset_ms"`
	Action     string `json:"action"` // "insert", "delete", "paste", "checkpoint"
	Content    string `json:"content"`
	CursorLine int    `json:"cursor_line"`
	IsPaste    bool   `json:"is_paste"`
	CharCount  int    `json:"char_count"`
}

// SubmissionTimeline telemetría completa de time-travel replay para auditar el proceso de desarrollo.
type SubmissionTimeline struct {
	SubmissionID         string             `json:"submission_id"`
	StudentID            string             `json:"student_id"`
	StudentName          string             `json:"student_name"`
	TotalDurationSeconds int                `json:"total_duration_seconds"`
	TotalKeystrokes      int                `json:"total_keystrokes"`
	PasteEventsCount     int                `json:"paste_events_count"`
	PastePercentage      float64            `json:"paste_percentage"`
	SuspiciousPasteFlag  bool               `json:"suspicious_paste_flag"`
	Keyframes            []TimelineKeyframe `json:"keyframes"`
}


