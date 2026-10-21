package httpdelivery

import (
	"encoding/json"
	"errors"
	"strings"

	"solv-backend/internal/core/domain"
)

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrSeedRequiresExamPurpose) ||
		errors.Is(err, domain.ErrInvalidVisibility) ||
		errors.Is(err, domain.ErrInvalidWeight) ||
		errors.Is(err, domain.ErrEmptyExpectedOutput) ||
		errors.Is(err, domain.ErrTestCasesNotArray) ||
		errors.Is(err, domain.ErrMemoryGovernedByTemplate) ||
		errors.Is(err, domain.ErrTemplateEnvironmentMismatch) ||
		errors.Is(err, domain.ErrTemplateNotApproved) ||
		errors.Is(err, domain.ErrPayloadMismatchForEnvironmentType) ||
		strings.Contains(err.Error(), "contrato de formato") ||
		strings.Contains(err.Error(), "inválido según contrato") {
		return true
	}
	return false
}

func validateEnvironmentPayload(bodyBytes []byte, ex *domain.Exercise) error {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return nil
	}

	envType := ex.EnvironmentType
	if envType == "" {
		if val, ok := rawMap["environment_type"].(string); ok {
			envType = val
		}
	}

	if envType == string(domain.EnvironmentTypeIDEPersistente) {
		_, hasContract := rawMap["contract"]
		_, hasCases := rawMap["cases"]
		_, hasTestCases := rawMap["test_cases"]
		refSol, _ := rawMap["reference_solution"].(string)

		if hasContract || hasCases || hasTestCases || strings.TrimSpace(refSol) != "" {
			return domain.ErrPayloadMismatchForEnvironmentType
		}
	}
	return nil
}

func normalizeExerciseConfig(body []byte, ex *domain.Exercise) {
	var raw struct {
		ASTRules       *domain.ASTRules         `json:"ast_rules"`
		Comparator     *domain.ComparatorConfig `json:"comparator"`
		TestCases      []domain.TestCase        `json:"test_cases"`
		DBInitScript   string                   `json:"db_init_script"`
		AllowResubmit  bool                     `json:"allow_resubmit"`
		AutoCheckpoint bool                     `json:"auto_checkpoint"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}
	if ex.EnvironmentType == string(domain.EnvironmentTypeIDEPersistente) || raw.DBInitScript != "" || raw.AllowResubmit || raw.AutoCheckpoint {
		if ex.Config.IDE == nil {
			ex.Config.IDE = &domain.IDEConfig{}
		}
		ex.Config.IDE.AllowResubmit = raw.AllowResubmit
		ex.Config.IDE.AutoCheckpoint = raw.AutoCheckpoint
		if raw.DBInitScript != "" {
			ex.Config.IDE.DBInitScript = raw.DBInitScript
			if ex.Config.Database == nil {
				ex.Config.Database = &domain.DatabaseConfig{}
			}
			ex.Config.Database.InitScript = raw.DBInitScript
		}
	}
	if ex.Type == domain.ExerciseTypeAlgorithm || ex.Type == "" {
		if ex.Config.Algorithm == nil {
			ex.Config.Algorithm = &domain.AlgorithmConfig{
				TimeLimitMS:   ex.TimeLimitMS,
				MemoryLimitMB: ex.MemoryLimitMB,
			}
		}
		if raw.ASTRules != nil {
			ex.Config.Algorithm.ASTRules = *raw.ASTRules
		}
		if raw.Comparator != nil {
			ex.Config.Algorithm.Comparator = raw.Comparator
		}
		if len(raw.TestCases) > 0 && len(ex.Config.Algorithm.TestCases) == 0 {
			for i := range raw.TestCases {
				raw.TestCases[i].Normalize()
			}
			ex.Config.Algorithm.TestCases = raw.TestCases
		}
	}
}
