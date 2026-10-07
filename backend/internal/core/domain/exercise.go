package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type ExerciseType string

const (
	ExerciseTypeAlgorithm ExerciseType = "algorithm"
	ExerciseTypeDatabase  ExerciseType = "database"
)

type ExercisePurpose string

const (
	ExercisePurposeClass ExercisePurpose = "class"
	ExercisePurposeExam  ExercisePurpose = "exam"
)

type ExerciseDifficulty string

const (
	ExerciseDifficultyEasy   ExerciseDifficulty = "easy"
	ExerciseDifficultyMedium ExerciseDifficulty = "medium"
	ExerciseDifficultyHard   ExerciseDifficulty = "hard"
)

type TestCaseVisibility string

const (
	TestCaseVisibilityExample TestCaseVisibility = "example"
	TestCaseVisibilityPublic  TestCaseVisibility = "public"
	TestCaseVisibilityHidden  TestCaseVisibility = "hidden"
)

var (
	ErrSeedRequiresExamPurpose = errors.New("per_student_seed is only allowed when purpose is 'exam'")
	ErrInvalidVisibility       = errors.New("invalid test case visibility: must be 'example', 'public', or 'hidden'")
	ErrInvalidWeight           = errors.New("test case weight must be non-negative")
	ErrEmptyExpectedOutput     = errors.New("test case expected_output cannot be empty")
	ErrTestCasesNotArray       = errors.New("test_cases must be an array")
	ErrModuleLocked            = errors.New("MODULE_LOCKED: el módulo curricular se encuentra bloqueado")
)

type Verdict string

const (
	VerdictAC           Verdict = "AC"            // Accepted
	VerdictWA           Verdict = "WA"            // Wrong Answer
	VerdictTLE          Verdict = "TLE"           // Time Limit Exceeded
	VerdictRE           Verdict = "RE"            // Runtime Error
	VerdictCE           Verdict = "CE"            // Compilation Error
	VerdictMLE          Verdict = "MLE"           // Memory Limit Exceeded (OOM kill de cgroups)
	VerdictVE           Verdict = "VE"            // Verdict Error (fallo o timeout del checker docente)
	VerdictASTViolation Verdict = "AST_VIOLATION" // AST Security Violation (regex)
	VerdictASTBlocked   Verdict = "AST_BLOCKED"   // AST Security Violation (Semgrep)
)

// ScanViolation represents a single Semgrep finding
type ScanViolation struct {
	RuleID  string `json:"rule_id"`
	Message string `json:"message"`
	Line    int    `json:"line"`
}

// ScanResult holds the aggregated output of a Semgrep pre-check scan
type ScanResult struct {
	Violations    []ScanViolation `json:"violations"`
	HasViolations bool            `json:"has_violations"`
}

type TestCase struct {
	ID             string             `json:"id,omitempty" db:"id"`
	ExerciseID     string             `json:"exercise_id,omitempty" db:"exercise_id"`
	OrderIndex     int                `json:"order_index" db:"order_index"`
	Input          string             `json:"input" db:"input"`
	ExpectedOutput string             `json:"expected_output" db:"expected_output"`
	Visibility     TestCaseVisibility `json:"visibility" db:"visibility"`
	Weight         float64            `json:"weight" db:"weight"`
	CreatedAt      *time.Time         `json:"created_at,omitempty" db:"created_at"`

	// Derived legacy booleans for transient API compatibility
	IsHidden bool `json:"is_hidden"`
	IsSample bool `json:"is_sample"`
}

func (tc *TestCase) Normalize() {
	if tc.Visibility == "" {
		if tc.IsHidden {
			tc.Visibility = TestCaseVisibilityHidden
		} else if tc.IsSample {
			tc.Visibility = TestCaseVisibilityExample
		} else {
			tc.Visibility = TestCaseVisibilityPublic
		}
	}
	tc.IsHidden = (tc.Visibility == TestCaseVisibilityHidden)
	tc.IsSample = (tc.Visibility == TestCaseVisibilityExample)
	if tc.Weight <= 0 {
		tc.Weight = 1.0
	}
}

func (tc *TestCase) Validate() error {
	if tc.ExpectedOutput == "" {
		return ErrEmptyExpectedOutput
	}
	if tc.Weight < 0 {
		return ErrInvalidWeight
	}
	switch tc.Visibility {
	case TestCaseVisibilityExample, TestCaseVisibilityPublic, TestCaseVisibilityHidden:
		return nil
	default:
		return fmt.Errorf("%w: got %q", ErrInvalidVisibility, tc.Visibility)
	}
}

type TestCases []TestCase

type ASTCustomRule struct {
	Language string `json:"language"`
	Pattern  string `json:"pattern"`
	Type     string `json:"type,omitempty"` // method, function, module
	Message  string `json:"message"`
}

type ASTRules struct {
	BlockNativeSort    bool            `json:"block_native_sort,omitempty"`
	BlockSystemModules bool            `json:"block_system_modules,omitempty"`
	ForbiddenImports   []string        `json:"forbidden_imports"`
	ForbiddenFunctions []string        `json:"forbidden_functions"`
	CustomRules        []ASTCustomRule `json:"custom_rules,omitempty"`
}

type ChecklistReport struct {
	Blockers   []string `json:"blockers"`
	Warnings   []string `json:"warnings"`
	Info       []string `json:"info"`
	CanPublish bool     `json:"can_publish"`
}

type ReferenceSolutionInput struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type ComparatorConfig struct {
	ID     string         `json:"id"`
	Params map[string]any `json:"params,omitempty"`
}

type AlgorithmConfig struct {
	TestCases     TestCases         `json:"test_cases"`
	ASTRules      ASTRules          `json:"ast_rules"`
	Comparator    *ComparatorConfig `json:"comparator,omitempty"`
	TimeLimitMS   int               `json:"time_limit_ms"`
	MemoryLimitMB int               `json:"memory_limit_mb"`
	InputFormat   json.RawMessage   `json:"input_format,omitempty"`
}

type DatabaseConfig struct {
	Engine            string `json:"engine"`             // e.g. "postgres"
	InitScript        string `json:"init_script"`        // DDL / DML previo
	ReferenceSolution string `json:"reference_solution"` // Solución de referencia del docente
	ValidationQuery   string `json:"validation_query"`   // Query para extraer el estado resultante
	ExpectedJSON      string `json:"expected_json"`      // JSON de referencia (autogenerado vía Dry Run)
	TimeLimitMS       int    `json:"time_limit_ms"`
	MemoryLimitMB     int    `json:"memory_limit_mb"`
}

type ExerciseConfig struct {
	Algorithm   *AlgorithmConfig `json:"algorithm,omitempty"`
	Database    *DatabaseConfig  `json:"database,omitempty"`
	InputFormat json.RawMessage  `json:"input_format,omitempty"`
}

func (ec ExerciseConfig) Value() (driver.Value, error) {
	return json.Marshal(ec)
}

func (ec *ExerciseConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ExerciseConfig value: %v", value)
	}
	return json.Unmarshal(bytes, ec)
}

var (
	ErrMissingReferenceSolution = errors.New("el ejercicio requiere una solución de referencia antes de ser publicado")
	ErrExerciseStale            = errors.New("el ejercicio tiene cambios pendientes y requiere un dry-run exitoso antes de ser publicado")
)

type DryRunJobStatus string

const (
	DryRunJobStatusQueued  DryRunJobStatus = "queued"
	DryRunJobStatusRunning DryRunJobStatus = "running"
	DryRunJobStatusDone    DryRunJobStatus = "done"
	DryRunJobStatusFailed  DryRunJobStatus = "failed"
)

type DryRunJob struct {
	ID              string            `json:"id" db:"id"`
	ExerciseID      string            `json:"exercise_id" db:"exercise_id"`
	Status          DryRunJobStatus   `json:"status" db:"status"`
	ProgressCurrent int               `json:"progress_current" db:"progress_current"`
	ProgressTotal   int               `json:"progress_total" db:"progress_total"`
	Result          *EvaluationResult `json:"result,omitempty" db:"result"`
	Error           string            `json:"error,omitempty" db:"error"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

type Exercise struct {
	ID                 string         `json:"id" db:"id"`
	SubjectID          *string        `json:"subject_id,omitempty" db:"subject_id"`
	Title              string         `json:"title" db:"title"`
	Description        string         `json:"description" db:"description"`
	Type               ExerciseType   `json:"type" db:"type"`
	Difficulty         *string        `json:"difficulty,omitempty" db:"difficulty"`
	Tags               pq.StringArray `json:"tags" db:"tags"`
	Purpose            string         `json:"purpose" db:"purpose"` // class, exam
	PerStudentSeed     bool           `json:"per_student_seed" db:"per_student_seed"`
	DueDate            *time.Time     `json:"due_date,omitempty" db:"due_date"`
	Boilerplate        string         `json:"boilerplate" db:"boilerplate"`
	Status             string         `json:"status" db:"status"` // draft, published, closed
	Language           string         `json:"language" db:"language"`
	TimeLimitMS        int            `json:"time_limit_ms" db:"time_limit_ms"`
	MemoryLimitMB      int            `json:"memory_limit_mb" db:"memory_limit_mb"`
	ReferenceSolution  string         `json:"reference_solution" db:"reference_solution"`
	Stale              bool           `json:"stale" db:"stale"`
	LastValidDryRunAt  *time.Time     `json:"last_valid_dry_run_at,omitempty" db:"last_valid_dry_run_at"`
	ModuleID           *string        `json:"module_id,omitempty" db:"module_id"`
	ExpectedComplexity *string        `json:"expected_complexity,omitempty" db:"expected_complexity"`
	Config             ExerciseConfig `json:"config" db:"config"`
	TenantID           string         `json:"tenant_id" db:"tenant_id"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
}

type ComplexityMeasurement struct {
	InputSize int     `json:"input_size"`
	TimeMs    float64 `json:"time_ms"`
	MemoryKb  float64 `json:"memory_kb"`
}

type ComplexityAnalysis struct {
	TimeComplexity  string                  `json:"time_complexity"`
	SpaceComplexity string                  `json:"space_complexity"`
	Measurements    []ComplexityMeasurement `json:"measurements"`
	Confidence      float64                 `json:"confidence"`
}

func (ex *Exercise) Validate() error {
	if ex.PerStudentSeed && ex.Purpose != string(ExercisePurposeExam) {
		return ErrSeedRequiresExamPurpose
	}
	if ex.Difficulty != nil && *ex.Difficulty != "" {
		diff := *ex.Difficulty
		if diff != string(ExerciseDifficultyEasy) && diff != string(ExerciseDifficultyMedium) && diff != string(ExerciseDifficultyHard) {
			return fmt.Errorf("invalid difficulty %q: must be 'easy', 'medium', or 'hard'", diff)
		}
	}
	if ex.Purpose == "" {
		ex.Purpose = string(ExercisePurposeClass)
	}
	if ex.Purpose != string(ExercisePurposeClass) && ex.Purpose != string(ExercisePurposeExam) {
		return fmt.Errorf("invalid purpose %q: must be 'class' or 'exam'", ex.Purpose)
	}
	if ex.Tags == nil {
		ex.Tags = pq.StringArray{}
	}
	return nil
}

type DueAssignment struct {
	ExerciseID  string     `json:"exercise_id" db:"exercise_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	SubjectID   string     `json:"subject_id" db:"subject_id"`
	SubjectName string     `json:"subject_name" db:"subject_name"`
	SubjectCode string     `json:"subject_code" db:"subject_code"`
	DueDate     *time.Time `json:"due_date" db:"due_date"`
	Type        string     `json:"type" db:"type"`
}

type EvaluationResult struct {
	Verdict         Verdict   `json:"verdict"`
	ExecutionTimeMS int       `json:"execution_time_ms"`
	MemoryUsedMB    float64   `json:"memory_used_mb"`
	Message         string    `json:"message"`
	FailedTestCase  *TestCase `json:"failed_test_case,omitempty"`
	// CaseResults agrega el veredicto de cada caso evaluado. La evaluacion
	// recorre todos los casos sin detencion temprana (D-EJ-03).
	CaseResults        []CaseResult        `json:"case_results,omitempty"`
	ActualJSON         string              `json:"actual_json,omitempty"`
	ExpectedJSON       string              `json:"expected_json,omitempty"`
	GeneratedCases     []TestCase          `json:"generated_cases,omitempty"`
	ComplexityAnalysis *ComplexityAnalysis `json:"complexity_analysis,omitempty"`
}

// CaseResult es el resultado de un unico caso de prueba.
type CaseResult struct {
	Index      int     `json:"index"`
	Verdict    Verdict `json:"verdict"`
	DurationMS int     `json:"duration_ms"`
	Message    string  `json:"message,omitempty"`
}

// RunMetric registra la telemetria de un caso ejecutado. Language usa la
// clave canonica (c++ -> cpp, c#/cs -> csharp). Ver design.md y 00012.
type RunMetric struct {
	ExerciseID  string  `json:"exercise_id" db:"exercise_id"`
	Language    string  `json:"language" db:"language"`
	ImageDigest string  `json:"image_digest" db:"image_digest"`
	DurationMS  int     `json:"duration_ms" db:"duration_ms"`
	Verdict     Verdict `json:"verdict" db:"verdict"`
	CaseIndex   int     `json:"case_index" db:"case_index"`
}

type EvaluationRunConfig struct {
	Language      string
	SourceCode    string
	MemoryLimitMB int
	TimeLimitMS   int
	TestCase      TestCase
	Comparator    *ComparatorConfig
}

type TestCaseRunResult struct {
	Verdict       Verdict
	ExecutionTime time.Duration
	ActualOutput  string
	StdErr        string
	ErrorDetails  string
	// ImageDigest identifica la imagen del runner (forma repo@digest) y
	// alimenta run_metrics. Lo informa la estrategia de lenguaje.
	ImageDigest string
}

type DBEvaluationRunConfig struct {
	Engine          string
	InitScript      string
	SolutionSQL     string
	ValidationQuery string
	TimeLimitMS     int
	MemoryLimitMB   int
}

type DBEvaluationResult struct {
	Verdict       Verdict
	ExecutionTime time.Duration
	ResultingJSON string
	ErrorDetails  string
}
