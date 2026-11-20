package httpdelivery

import (
	"encoding/json"
	"errors"
	"slices"
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
		errors.Is(err, domain.ErrTimeLimitOutOfRange) ||
		errors.Is(err, domain.ErrInvalidAppPort) ||
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
		AppRunCommand  *string                  `json:"app_run_command"`
		AppPort        *int                     `json:"app_port"`
		// Los lenguajes elegibles y sus códigos de partida viven en config:
		// el alumno los recibe con el ejercicio, nunca el enunciado oculto.
		AllowedLanguages *[]string         `json:"allowed_languages"`
		Boilerplates     map[string]string `json:"boilerplates"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}
	if raw.AllowedLanguages != nil {
		allowed := make([]string, 0, len(*raw.AllowedLanguages))
		seen := make(map[string]bool, len(*raw.AllowedLanguages))
		for _, lang := range *raw.AllowedLanguages {
			canonical := domain.CanonicalJudgeLanguage(lang)
			if canonical == "" || seen[canonical] {
				continue
			}
			seen[canonical] = true
			allowed = append(allowed, canonical)
		}
		ex.Config.AllowedLanguages = allowed
		if len(allowed) > 0 && !slices.Contains(allowed, domain.CanonicalJudgeLanguage(ex.Language)) {
			// El lenguaje por defecto debe estar entre los ofrecidos: el primero
			// de la lista es el que se muestra al alumno al abrir la práctica.
			ex.Language = allowed[0]
		}
	}
	if raw.Boilerplates != nil {
		normalized := make(map[string]string, len(raw.Boilerplates))
		for lang, code := range raw.Boilerplates {
			canonical := domain.CanonicalJudgeLanguage(lang)
			if canonical == "" || strings.TrimSpace(code) == "" {
				continue
			}
			if len(ex.Config.AllowedLanguages) > 0 && !slices.Contains(ex.Config.AllowedLanguages, canonical) {
				continue
			}
			normalized[canonical] = code
		}
		if len(normalized) > 0 {
			ex.Config.Boilerplates = normalized
		}
	}
	if ex.EnvironmentType == string(domain.EnvironmentTypeIDEPersistente) || raw.DBInitScript != "" || raw.AllowResubmit || raw.AutoCheckpoint || raw.AppRunCommand != nil || raw.AppPort != nil {
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
		if raw.AppRunCommand != nil {
			// Un comando vacío borra la receta anterior: el docente puede volver
			// a dejar la práctica sin preview sin que quede un valor viejo.
			ex.Config.IDE.AppRunCommand = strings.TrimSpace(*raw.AppRunCommand)
		}
		if raw.AppPort != nil {
			ex.Config.IDE.AppPort = *raw.AppPort
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
