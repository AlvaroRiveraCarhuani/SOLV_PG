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
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	Language        string           `json:"language,omitempty"`
	TimeLimitMs     int              `json:"time_limit_ms,omitempty"`
	MemoryMB        int              `json:"memory_mb,omitempty"`
	Constraints     []string         `json:"constraints,omitempty"`
	PublicTestCases []PublicTestCase `json:"public_test_cases,omitempty"`
}

func ToExercisePublicResponse(ex *domain.Exercise) *ExercisePublicResponse {
	if ex == nil {
		return nil
	}

	resp := &ExercisePublicResponse{
		ID:          ex.ID,
		Title:       ex.Title,
		Description: ex.Description,
		Type:        string(ex.Type),
	}

	if ex.Config.Algorithm != nil {
		resp.TimeLimitMs = ex.Config.Algorithm.TimeLimitMS
		resp.MemoryMB = ex.Config.Algorithm.MemoryLimitMB

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

		publicCases := make([]PublicTestCase, 0)
		for _, tc := range ex.Config.Algorithm.TestCases {
			if !tc.IsHidden {
				publicCases = append(publicCases, PublicTestCase{
					Input:          tc.Input,
					ExpectedOutput: tc.ExpectedOutput,
				})
			}
		}
		if len(publicCases) > 0 {
			resp.PublicTestCases = publicCases
		}
	} else if ex.Config.Database != nil {
		resp.Language = ex.Config.Database.Engine
		resp.TimeLimitMs = ex.Config.Database.TimeLimitMS
		resp.MemoryMB = ex.Config.Database.MemoryLimitMB
	}

	return resp
}

