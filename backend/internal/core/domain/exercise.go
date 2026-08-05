package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type ExerciseType string

const (
	ExerciseTypeAlgorithm ExerciseType = "algorithm"
	ExerciseTypeDatabase  ExerciseType = "database"
)

type Verdict string

const (
	VerdictAC           Verdict = "AC"           // Accepted
	VerdictWA           Verdict = "WA"           // Wrong Answer
	VerdictTLE          Verdict = "TLE"          // Time Limit Exceeded
	VerdictRE           Verdict = "RE"           // Runtime Error
	VerdictCE           Verdict = "CE"           // Compilation Error
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
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	IsHidden       bool   `json:"is_hidden"`
}

type TestCases []TestCase

type ASTRules struct {
	ForbiddenImports   []string `json:"forbidden_imports"`
	ForbiddenFunctions []string `json:"forbidden_functions"`
}

type AlgorithmConfig struct {
	TestCases     TestCases `json:"test_cases"`
	ASTRules      ASTRules  `json:"ast_rules"`
	TimeLimitMS   int       `json:"time_limit_ms"`
	MemoryLimitMB int       `json:"memory_limit_mb"`
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
	Algorithm *AlgorithmConfig `json:"algorithm,omitempty"`
	Database  *DatabaseConfig  `json:"database,omitempty"`
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

type Exercise struct {
	ID          string         `json:"id" db:"id"`
	Title       string         `json:"title" db:"title"`
	Description string         `json:"description" db:"description"`
	Type        ExerciseType   `json:"type" db:"type"`
	Config      ExerciseConfig `json:"config" db:"config"`
	TenantID    string         `json:"tenant_id" db:"tenant_id"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

type EvaluationResult struct {
	Verdict         Verdict   `json:"verdict"`
	ExecutionTimeMS int       `json:"execution_time_ms"`
	MemoryUsedMB    float64   `json:"memory_used_mb"`
	Message         string    `json:"message"`
	FailedTestCase  *TestCase `json:"failed_test_case,omitempty"`
	ActualJSON      string    `json:"actual_json,omitempty"`
	ExpectedJSON    string    `json:"expected_json,omitempty"`
}

type EvaluationRunConfig struct {
	Language      string
	SourceCode    string
	MemoryLimitMB int
	TimeLimitMS   int
	TestCase      TestCase
}

type TestCaseRunResult struct {
	Verdict        Verdict
	ExecutionTime  time.Duration
	ActualOutput   string
	StdErr         string
	ErrorDetails   string
}

type DBEvaluationRunConfig struct {
	Engine            string
	InitScript        string
	SolutionSQL       string
	ValidationQuery   string
	TimeLimitMS       int
	MemoryLimitMB     int
}

type DBEvaluationResult struct {
	Verdict        Verdict
	ExecutionTime  time.Duration
	ResultingJSON  string
	ErrorDetails   string
}
