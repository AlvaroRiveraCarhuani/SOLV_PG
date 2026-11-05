package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// SubmissionComment representa un comentario pedagógico docente anclado a una línea de código.
// EndLineNumber es opcional: null comenta una sola línea; con valor comenta el
// bloque [LineNumber, EndLineNumber]. La línea sigue siendo global sobre `code`.
// En modalidad IDE, FilePath y LocalLineNumber guardan el ancla por archivo que
// consume el visor y SOLV Companion; en juez ambos son nil.
type SubmissionComment struct {
	ID              string  `json:"id" db:"id"`
	TenantID        string  `json:"tenant_id" db:"tenant_id"`
	SubmissionID    string  `json:"submission_id" db:"submission_id"`
	AuthorID        string  `json:"author_id" db:"author_id"`
	AuthorName      string  `json:"author_name" db:"author_name"`
	LineNumber      int     `json:"line_number" db:"line_number"`
	EndLineNumber   *int    `json:"end_line_number,omitempty" db:"end_line_number"`
	FilePath        *string `json:"file_path,omitempty" db:"file_path"`
	LocalLineNumber *int    `json:"local_line_number,omitempty" db:"local_line_number"`
	// LocalEndLineNumber solo se usa al crear: el servicio lo traduce a la línea
	// global de EndLineNumber y no se persiste.
	LocalEndLineNumber *int      `json:"-" db:"-"`
	Comment            string    `json:"comment" db:"comment"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// RangeEnd devuelve la última línea del comentario. Un comentario de una sola
// línea devuelve su propia línea, también cuando el puntero es nil o inválido.
func (c SubmissionComment) RangeEnd() int {
	if c.EndLineNumber != nil && *c.EndLineNumber >= c.LineNumber {
		return *c.EndLineNumber
	}
	return c.LineNumber
}

// SubmissionQueueItem representa un elemento en la cola de revisión del curso.
type SubmissionQueueItem struct {
	ID              string    `json:"id" db:"id"`
	ExerciseID      string    `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle   string    `json:"exercise_title" db:"exercise_title"`
	StudentID       string    `json:"student_id" db:"student_id"`
	StudentName     string    `json:"student_name" db:"student_name"`
	StudentEmail    string    `json:"student_email" db:"student_email"`
	Verdict         string    `json:"verdict" db:"verdict"`
	VerdictOrigin   string    `json:"verdict_origin,omitempty" db:"verdict_origin"`
	Score           *int      `json:"score,omitempty" db:"score"`
	ManualOverride  bool      `json:"manual_override" db:"manual_override"`
	ExecutionTimeMS int       `json:"execution_time_ms" db:"execution_time_ms"`
	MemoryUsedMB    int       `json:"memory_used_mb" db:"memory_used_mb"`
	SubmittedAt     time.Time `json:"submitted_at" db:"submitted_at"`
	CommentsCount   int       `json:"comments_count" db:"comments_count"`
	ReviewStatus    string    `json:"review_status" db:"review_status"`
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
	ID                 string              `json:"id" db:"id"`
	ExerciseID         string              `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle      string              `json:"exercise_title" db:"exercise_title"`
	SubjectID          string              `json:"subject_id" db:"subject_id"`
	SubjectName        string              `json:"subject_name" db:"subject_name"`
	StudentID          string              `json:"student_id" db:"student_id"`
	StudentName        string              `json:"student_name" db:"student_name"`
	StudentEmail       string              `json:"student_email" db:"student_email"`
	EnvironmentType    string              `json:"environment_type" db:"environment_type"`
	Language           string              `json:"language" db:"language"`
	Boilerplate        string              `json:"boilerplate,omitempty" db:"boilerplate"`
	Code               string              `json:"code,omitempty" db:"code"`
	Verdict            string              `json:"verdict" db:"verdict"`
	VerdictOrigin      string              `json:"verdict_origin,omitempty" db:"verdict_origin"`
	Score              *int                `json:"score,omitempty" db:"score"`
	ManualOverride     bool                `json:"manual_override" db:"manual_override"`
	OverrideReason     string              `json:"override_reason,omitempty" db:"override_reason"`
	GradedBy           *string             `json:"graded_by,omitempty" db:"graded_by"`
	GradedByName       string              `json:"graded_by_name,omitempty" db:"graded_by_name"`
	ExecutionTimeMS    int                 `json:"execution_time_ms" db:"execution_time_ms"`
	MemoryUsedMB       int                 `json:"memory_used_mb" db:"memory_used_mb"`
	ASTResult          json.RawMessage     `json:"ast_result" db:"ast_result"`
	GeneratedCases     json.RawMessage     `json:"generated_cases,omitempty" db:"generated_cases"`
	ExpectedComplexity string              `json:"expected_complexity,omitempty" db:"expected_complexity"`
	ComplexityAnalysis *ComplexityAnalysis `json:"complexity_analysis,omitempty" db:"complexity_analysis"`
	TestCases          []TestCaseReview    `json:"test_cases"`
	FilesAvailable     bool                `json:"files_available"`
	MetricsAvailable   bool                `json:"metrics_available"`
	Files              []ProjectFile       `json:"files"`
	// BaseFiles es el proyecto base del ejercicio (IDE) derivado de
	// `exercises.boilerplate` con el mismo parser del snapshot. Es lo que usa
	// el diff por archivo; no se persiste ni se toca el disco.
	BaseFiles          []ProjectFile       `json:"base_files"`
	BaseFilesAvailable bool                `json:"base_files_available"`
	Comments           []SubmissionComment `json:"comments"`
	NextSubmissionID   *string             `json:"next_submission_id,omitempty"`
	PrevSubmissionID   *string             `json:"prev_submission_id,omitempty"`
	SubmittedAt        time.Time           `json:"submitted_at" db:"submitted_at"`
}

// OverrideRequestDTO payload para anular o convalidar manualmente una calificación.
type OverrideRequestDTO struct {
	Verdict        string `json:"verdict"`
	OverrideReason string `json:"override_reason"`
	Score          *int   `json:"score,omitempty"`
}

// AddCommentRequestDTO payload para agregar un comentario in-line o de bloque.
// Acepta dos formas: coordenadas globales (clientes juez y versiones previas) o
// ancla por archivo (modalidad IDE). Si llegan ambas, deben coincidir.
type AddCommentRequestDTO struct {
	LineNumber         int     `json:"line_number"`
	EndLineNumber      *int    `json:"end_line_number,omitempty"`
	FilePath           *string `json:"file_path,omitempty"`
	LocalLineNumber    *int    `json:"local_line_number,omitempty"`
	LocalEndLineNumber *int    `json:"local_end_line_number,omitempty"`
	Comment            string  `json:"comment"`
}

var ErrInvalidOverrideReason = errors.New("override reason must be at least 10 characters")

// ErrLineOutOfRange: el comentario apunta fuera de las líneas de la entrega.
// La vista traduce líneas locales a globales, pero el backend valida el rango
// real de `code` para no aceptar anclas inexistentes.
var ErrLineOutOfRange = errors.New("line_out_of_range")

// ErrLineRangeInvalid: el rango está invertido o la línea inicial no es válida.
var ErrLineRangeInvalid = errors.New("line_range_invalid")

// ErrCommentFileUnknown: el comentario apunta a un archivo que no existe en el
// snapshot de la entrega, o la práctica no tiene archivos navegables.
var ErrCommentFileUnknown = errors.New("comment_file_unknown")

// ErrLineRangeSpansFiles: en IDE el bloque debe vivir dentro de un solo archivo;
// un rango que cruza cabeceras o archivos no se puede anclar en el árbol.
var ErrLineRangeSpansFiles = errors.New("line_range_spans_files")

// ErrEphemeralNotSupportedForIDE: el runner efímero es una capacidad del juez.
// Un proyecto IDE se revisa con la rúbrica, no re-ejecutando casos I/O.
var ErrEphemeralNotSupportedForIDE = errors.New("ephemeral run is only available for JUEZ_EFIMERO")

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

// StudentGradesRow fila de notas por estudiante en el curso. Una celda nula
// significa "sin corregir" y nunca se presenta como 0.
type StudentGradesRow struct {
	StudentID    string          `json:"student_id"`
	StudentName  string          `json:"student_name"`
	StudentEmail string          `json:"student_email"`
	Grades       map[string]*int `json:"grades"`
	Average      float64         `json:"average"`
	GradedCount  int             `json:"graded_count"`
	PendingCount int             `json:"pending_count"`
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
	SubmissionID    string    `json:"submission_id" db:"submission_id"`
	ExerciseID      string    `json:"exercise_id" db:"exercise_id"`
	ExerciseTitle   string    `json:"exercise_title" db:"exercise_title"`
	StudentID       string    `json:"student_id" db:"student_id"`
	StudentName     string    `json:"student_name" db:"student_name"`
	StudentEmail    string    `json:"student_email" db:"student_email"`
	Language        string    `json:"language" db:"language"`
	Code            string    `json:"code" db:"code"`
	Boilerplate     string    `json:"boilerplate" db:"boilerplate"`
	EnvironmentType string    `json:"environment_type" db:"environment_type"`
	Verdict         string    `json:"verdict" db:"verdict"`
	SubmittedAt     time.Time `json:"submitted_at" db:"submitted_at"`
}

// PlagiarismRangeSource es un tramo de líneas de una de las dos entregas. En
// modalidad IDE FilePath identifica el archivo y las líneas son locales a ese
// archivo; en el evaluador automático el tramo es del código completo y las
// líneas son las del envío.
type PlagiarismRangeSource struct {
	FilePath  string `json:"file_path,omitempty"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// PlagiarismRange es un tramo de tokens que las dos entregas comparten. Las
// líneas apuntan al fuente, no a tokens normalizados del compilador.
type PlagiarismRange struct {
	SourceA    PlagiarismRangeSource `json:"source_a"`
	SourceB    PlagiarismRangeSource `json:"source_b"`
	TokenCount int                   `json:"token_count"`
	ExactMatch bool                  `json:"exact_match"`
}

// PlagiarismMatch representa un par de entregas con similitud estructural AST por encima del umbral.
type PlagiarismMatch struct {
	SubmissionIDA  string            `json:"submission_id_a"`
	StudentIDA     string            `json:"student_id_a"`
	StudentNameA   string            `json:"student_name_a"`
	SubmissionIDB  string            `json:"submission_id_b"`
	StudentIDB     string            `json:"student_id_b"`
	StudentNameB   string            `json:"student_name_b"`
	ExerciseID     string            `json:"exercise_id"`
	ExerciseTitle  string            `json:"exercise_title"`
	Similarity     float64           `json:"similarity"`
	RiskLevel      string            `json:"risk_level"`
	MatchingTokens int               `json:"matching_tokens"`
	TotalTokensA   int               `json:"total_tokens_a"`
	TotalTokensB   int               `json:"total_tokens_b"`
	Ranges         []PlagiarismRange `json:"ranges"`
}

// Errores del visor de un par de entregas.
var (
	// ErrPlagiarismPairForbidden: alguna de las dos entregas no pertenece a una
	// materia del docente. No se distingue de una entrega inexistente.
	ErrPlagiarismPairForbidden = errors.New("plagiarism_pair_forbidden")
	// ErrPlagiarismPairDifferentExercise: las entregas son de prácticas distintas.
	ErrPlagiarismPairDifferentExercise = errors.New("plagiarism_pair_different_exercise")
	// ErrPlagiarismPairSameSubmission: el par apunta dos veces a la misma entrega.
	ErrPlagiarismPairSameSubmission = errors.New("plagiarism_pair_same_submission")
)

// PlagiarismPairSubmission es la lectura de una entrega para el visor del par.
// Vive fuera de SubmissionForPlagiarism porque agrega la metadata de revisión
// que el motor no necesita.
type PlagiarismPairSubmission struct {
	SubmissionID       string              `db:"submission_id"`
	ExerciseID         string              `db:"exercise_id"`
	ExerciseTitle      string              `db:"exercise_title"`
	SubjectID          string              `db:"subject_id"`
	SubjectName        string              `db:"subject_name"`
	StudentID          string              `db:"student_id"`
	StudentName        string              `db:"student_name"`
	StudentEmail       string              `db:"student_email"`
	EnvironmentType    string              `db:"environment_type"`
	Language           string              `db:"language"`
	Code               string              `db:"code"`
	Boilerplate        string              `db:"boilerplate"`
	Verdict            string              `db:"verdict"`
	ExecutionTimeMS    int                 `db:"execution_time_ms"`
	MemoryUsedMB       int                 `db:"memory_used_mb"`
	ExpectedComplexity string              `db:"expected_complexity"`
	ComplexityAnalysis *ComplexityAnalysis `db:"complexity_analysis"`
	SubmittedAt        time.Time           `db:"submitted_at"`
}

// PlagiarismPairMetrics resume la telemetría de escritura de una entrega. No
// incluye los eventos: el visor solo necesita los totales.
type PlagiarismPairMetrics struct {
	TotalTimeMS     int64   `json:"total_time_ms"`
	TotalKeystrokes int     `json:"total_keystrokes"`
	PasteEvents     int     `json:"paste_events_count"`
	PastePercentage float64 `json:"paste_percentage"`
	EventsAvailable bool    `json:"events_available"`
}

// PlagiarismPairSide es una de las dos entregas enfrentadas. En IDE se envían
// los archivos; en el evaluador automático, el código completo.
type PlagiarismPairSide struct {
	SubmissionID    string                `json:"submission_id"`
	StudentID       string                `json:"student_id"`
	StudentName     string                `json:"student_name"`
	StudentEmail    string                `json:"student_email"`
	Verdict         string                `json:"verdict"`
	Language        string                `json:"language"`
	Complexity      string                `json:"complexity,omitempty"`
	ExecutionTimeMS int                   `json:"execution_time_ms"`
	MemoryUsedMB    int                   `json:"memory_used_mb"`
	Code            string                `json:"code,omitempty"`
	Files           []ProjectFile         `json:"files,omitempty"`
	FilesAvailable  bool                  `json:"files_available"`
	Metrics         PlagiarismPairMetrics `json:"metrics"`
}

// PlagiarismPairExercise es el contexto compartido por las dos entregas.
type PlagiarismPairExercise struct {
	ID                 string        `json:"id"`
	Title              string        `json:"title"`
	SubjectID          string        `json:"subject_id"`
	SubjectName        string        `json:"subject_name"`
	EnvironmentType    string        `json:"environment_type"`
	ExpectedComplexity string        `json:"expected_complexity,omitempty"`
	BaseFiles          []ProjectFile `json:"base_files,omitempty"`
	BaseFilesAvailable bool          `json:"base_files_available"`
}

// PlagiarismPair es el documento del visor: contexto, las dos entregas, sus
// tramos compartidos y las métricas forenses resumidas.
type PlagiarismPair struct {
	Exercise   PlagiarismPairExercise `json:"exercise"`
	A          PlagiarismPairSide     `json:"a"`
	B          PlagiarismPairSide     `json:"b"`
	Similarity float64                `json:"similarity"`
	RiskLevel  string                 `json:"risk_level"`
	Ranges     []PlagiarismRange      `json:"ranges"`
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

// LivePracticeEnvironmentSession representa una sesion de contenedor activa en vivo para Shadow Mode y monitoreo en tiempo real.
type LivePracticeEnvironmentSession struct {
	EnvironmentID   string    `db:"environment_id" json:"environment_id"`
	ContainerID   string    `db:"container_id" json:"container_id"`
	StudentID     string    `db:"student_id" json:"student_id"`
	StudentName   string    `db:"student_name" json:"student_name"`
	StudentEmail  string    `db:"student_email" json:"student_email"`
	SubjectID     string    `db:"subject_id" json:"subject_id"`
	SubjectName   string    `db:"subject_name" json:"subject_name"`
	Status        string    `db:"status" json:"status"`
	MemoryLimitMB int64     `db:"memory_limit_mb" json:"memory_limit_mb"`
	OOMStrikes    int       `db:"oom_strikes" json:"oom_strikes"`
	LastHeartbeat time.Time `db:"last_heartbeat" json:"last_heartbeat"`
	IsAttached    bool      `json:"is_attached"`

	// New real-time metrics from Docker
	MemoryUsedMB     int64     `json:"memory_used_mb,omitempty"`
	CPUPercent       float64   `json:"cpu_percent,omitempty"`
	MetricsAvailable bool      `json:"metrics_available,omitempty"`
	MetricsUpdatedAt time.Time `json:"metrics_updated_at,omitempty"`

	// Legacy fields for backward compatibility
	MemoryHistory []int64   `json:"memory_history,omitempty"`
	CPUHistory    []float64 `json:"cpu_history,omitempty"`
	ActivityState string    `json:"activity_state,omitempty"`
	WPM           int       `json:"wpm,omitempty"`
}

// TutorCommandRequest comando emitido por el docente para ejecutar diagnostico o ayuda interactiva.
type TutorCommandRequest struct {
	Command string `json:"command"`
}

// TutorCommandResponse resultado de la ejecucion del comando en el contenedor del estudiante.
type TutorCommandResponse struct {
	ContainerID string `json:"container_id"`
	Command     string `json:"command"`
	Output      string `json:"output"`
	ExitCode    int    `json:"exit_code"`
	ExecutedAt  string `json:"executed_at"`
}

// FuzzGenerationRequest parametros para la generacion automatizada de casos de prueba y condiciones de borde.
type FuzzGenerationRequest struct {
	ExerciseID     string   `json:"exercise_id,omitempty"`
	TargetLanguage string   `json:"target_language"`
	ParameterTypes []string `json:"parameter_types"` // "integer", "string", "array", "matrix", "float", "sql"
	Categories     []string `json:"categories"`      // "numeric_bounds", "empty_structures", "unicode_special", "injection_escape", "large_input"
	ReferenceCode  string   `json:"reference_code,omitempty"`
	Count          int      `json:"count"`
}

// GeneratedFuzzCase caso de prueba individual generado para evaluar robustez.
type GeneratedFuzzCase struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Input       string `json:"input"`
	Expected    string `json:"expected"`
	IsPublic    bool   `json:"is_public"`
	Weight      int    `json:"weight"`
	RiskLevel   string `json:"risk_level"`
}

// FuzzGenerationReport reporte consolidado de casos generados.
type FuzzGenerationReport struct {
	TotalGenerated int                 `json:"total_generated"`
	Categories     map[string]int      `json:"categories"`
	Cases          []GeneratedFuzzCase `json:"cases"`
}

// ApplyFuzzCasesRequest solicitud para persistir los casos seleccionados en el ejercicio.
type ApplyFuzzCasesRequest struct {
	Cases []GeneratedFuzzCase `json:"cases"`
}
