package dto

import (
	"solv-backend/internal/core/domain"
)

type PublicTestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

type ExercisePublicResponse struct {
	ID              string           `json:"id"`
	SubjectID       *string          `json:"subject_id,omitempty"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	EnvironmentType string           `json:"environment_type,omitempty"`
	Difficulty      *string          `json:"difficulty,omitempty"`
	Tags            []string         `json:"tags,omitempty"`
	Purpose         string           `json:"purpose,omitempty"`
	PerStudentSeed  bool             `json:"per_student_seed,omitempty"`
	Language        string           `json:"language,omitempty"`
	Boilerplate     string           `json:"boilerplate,omitempty"`
	TimeLimitMs     int              `json:"time_limit_ms,omitempty"`
	MemoryMB        int              `json:"memory_mb,omitempty"`
	MemoryLimitMB   int              `json:"memory_limit_mb,omitempty"`
	Constraints     []string         `json:"constraints,omitempty"`
	PublicTestCases []PublicTestCase `json:"public_test_cases,omitempty"`
	AllowResubmit   bool             `json:"allow_resubmit,omitempty"`
	AutoCheckpoint  bool             `json:"auto_checkpoint,omitempty"`
}

func ToExercisePublicResponse(ex *domain.Exercise) *ExercisePublicResponse {
	if ex == nil {
		return nil
	}

	resp := &ExercisePublicResponse{
		ID:              ex.ID,
		SubjectID:       ex.SubjectID,
		Title:           ex.Title,
		Description:     ex.Description,
		Type:            string(ex.Type),
		EnvironmentType: ex.EnvironmentType,
		Difficulty:      ex.Difficulty,
		Tags:            ex.Tags,
		Purpose:         ex.Purpose,
		PerStudentSeed:  ex.PerStudentSeed,
		Language:        ex.Language,
		Boilerplate:     ex.Boilerplate,
		TimeLimitMs:     ex.TimeLimitMS,
		MemoryMB:        ex.MemoryLimitMB,
		MemoryLimitMB:   ex.MemoryLimitMB,
	}

	if ex.Config.Algorithm != nil {
		if ex.Config.Algorithm.TimeLimitMS > 0 {
			resp.TimeLimitMs = ex.Config.Algorithm.TimeLimitMS
		}
		if ex.Config.Algorithm.MemoryLimitMB > 0 {
			resp.MemoryMB = ex.Config.Algorithm.MemoryLimitMB
			resp.MemoryLimitMB = ex.Config.Algorithm.MemoryLimitMB
		}

		constraints := make([]string, 0)
		for _, imp := range ex.Config.Algorithm.ASTRules.ForbiddenImports {
			constraints = append(constraints, "Forbidden import: "+imp)
		}
		for _, fn := range ex.Config.Algorithm.ASTRules.ForbiddenFunctions {
			constraints = append(constraints, "Forbidden function: "+fn)
		}
		if len(constraints) > 0 {
			resp.Constraints = constraints
		}

		exampleCases := make([]PublicTestCase, 0)
		for _, tc := range ex.Config.Algorithm.TestCases {
			if tc.Visibility == domain.TestCaseVisibilityExample || (tc.Visibility == "" && !tc.IsHidden) {
				exampleCases = append(exampleCases, PublicTestCase{
					Input:          tc.Input,
					ExpectedOutput: tc.ExpectedOutput,
				})
			}
		}
		if len(exampleCases) > 0 {
			resp.PublicTestCases = exampleCases
		}
	} else if ex.Config.Database != nil {
		if ex.Config.Database.Engine != "" {
			resp.Language = ex.Config.Database.Engine
		}
		if ex.Config.Database.TimeLimitMS > 0 {
			resp.TimeLimitMs = ex.Config.Database.TimeLimitMS
		}
		if ex.Config.Database.MemoryLimitMB > 0 {
			resp.MemoryMB = ex.Config.Database.MemoryLimitMB
			resp.MemoryLimitMB = ex.Config.Database.MemoryLimitMB
		}
	}

	// Metadatos del IDE persistente visibles para el alumno (nunca el db_init_script).
	if ex.Config.IDE != nil {
		resp.AllowResubmit = ex.Config.IDE.AllowResubmit
		resp.AutoCheckpoint = ex.Config.IDE.AutoCheckpoint
	}

	return resp
}
