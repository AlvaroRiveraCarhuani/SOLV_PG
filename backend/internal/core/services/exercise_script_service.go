package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
)

var (
	ErrInputFormatRequired   = errors.New("el ejercicio debe tener un contrato de entrada (input_format) configurado para validar los casos del script")
	ErrTooManyScriptCases    = errors.New("el script generó más de 100 casos de prueba (máximo 100 por ejecución)")
	ErrEmptyScript           = errors.New("el script de generación no puede estar vacío")
	ErrScriptExecutionFailed = errors.New("la ejecución del script falló o excedió el tiempo límite de 30 segundos")
	ErrInvalidScriptOutput   = errors.New("el stdout del script no es un array JSON válido de objetos {input, expected_output}")
)

type ScriptCaseItem struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

type ScriptCaseValidationItem struct {
	Index          int    `json:"index"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Valid          bool   `json:"valid"`
	Error          string `json:"error,omitempty"`
}

type ScriptGenerationResponse struct {
	Mode            string                     `json:"mode"`
	ExecutionTimeMS  int64                      `json:"execution_time_ms,omitempty"`
	CasesCount      int                        `json:"cases_count,omitempty"`
	Cases           []ScriptCaseValidationItem `json:"cases,omitempty"`
	CanImport       bool                       `json:"can_import"`
	TotalValid      int                        `json:"total_valid,omitempty"`
	TotalInvalid    int                        `json:"total_invalid,omitempty"`
	ImportedCount   int                        `json:"imported_count,omitempty"`
	ImportedCaseIDs []string                   `json:"imported_case_ids,omitempty"`
}

func (s *EvaluationService) SetScriptSandboxRunner(r domain.ScriptSandboxRunner) {
	s.scriptRunner = r
}

func (s *EvaluationService) GenerateCasesFromScript(ctx context.Context, exerciseID, tenantID, scriptCode string, dryRun bool) (*ScriptGenerationResponse, error) {
	if strings.TrimSpace(scriptCode) == "" {
		return nil, ErrEmptyScript
	}

	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ejercicio no encontrado: %w", err)
	}

	var inputFormat json.RawMessage
	if ex.Config.Algorithm != nil && len(ex.Config.Algorithm.InputFormat) > 0 && string(ex.Config.Algorithm.InputFormat) != "null" {
		inputFormat = ex.Config.Algorithm.InputFormat
	} else if len(ex.Config.InputFormat) > 0 && string(ex.Config.InputFormat) != "null" {
		inputFormat = ex.Config.InputFormat
	}

	if len(inputFormat) == 0 || string(inputFormat) == "null" || string(inputFormat) == "{}" {
		return nil, ErrInputFormatRequired
	}

	if s.scriptRunner == nil {
		return nil, errors.New("el motor de sandbox para scripts no está inicializado")
	}

	startTime := time.Now()
	stdout, stderr, err := s.scriptRunner.RunPythonScript(ctx, scriptCode, 30)
	execTimeMS := time.Since(startTime).Milliseconds()
	if err != nil {
		errMsg := strings.TrimSpace(stderr)
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("%w: %s", ErrScriptExecutionFailed, errMsg)
	}

	var generatedCases []ScriptCaseItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &generatedCases); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidScriptOutput, strings.TrimSpace(stdout))
	}

	if len(generatedCases) > 100 {
		return nil, ErrTooManyScriptCases
	}

	validationItems := make([]ScriptCaseValidationItem, 0, len(generatedCases))
	totalValid := 0
	totalInvalid := 0

	for i, c := range generatedCases {
		valid, errStr := s.formatValidator.ValidateCase(inputFormat, c.Input)
		if valid {
			totalValid++
		} else {
			totalInvalid++
		}
		validationItems = append(validationItems, ScriptCaseValidationItem{
			Index:          i,
			Input:          c.Input,
			ExpectedOutput: c.ExpectedOutput,
			Valid:          valid,
			Error:          errStr,
		})
	}

	canImport := totalInvalid == 0

	if dryRun {
		return &ScriptGenerationResponse{
			Mode:            "dry_run",
			ExecutionTimeMS:  execTimeMS,
			CasesCount:      len(generatedCases),
			Cases:           validationItems,
			CanImport:       canImport,
			TotalValid:      totalValid,
			TotalInvalid:    totalInvalid,
		}, nil
	}

	if !canImport {
		return &ScriptGenerationResponse{
			Mode:            "import",
			ExecutionTimeMS:  execTimeMS,
			CasesCount:      len(generatedCases),
			Cases:           validationItems,
			CanImport:       false,
			TotalValid:      totalValid,
			TotalInvalid:    totalInvalid,
		}, nil
	}

	testCasesToSave := make([]domain.TestCase, 0, len(generatedCases))
	importedIDs := make([]string, 0, len(generatedCases))

	for _, c := range generatedCases {
		id := uuid.NewString()
		testCasesToSave = append(testCasesToSave, domain.TestCase{
			ID:             id,
			ExerciseID:     exerciseID,
			Input:          c.Input,
			ExpectedOutput: c.ExpectedOutput,
			Visibility:     domain.TestCaseVisibilityHidden,
			Weight:         1.0,
		})
		importedIDs = append(importedIDs, id)
	}

	if err := s.BulkAddTestCases(ctx, exerciseID, tenantID, testCasesToSave); err != nil {
		return nil, fmt.Errorf("error guardando los casos generados: %w", err)
	}

	return &ScriptGenerationResponse{
		Mode:            "import",
		ExecutionTimeMS:  execTimeMS,
		CasesCount:      len(generatedCases),
		CanImport:       true,
		TotalValid:      totalValid,
		TotalInvalid:    0,
		ImportedCount:   len(importedIDs),
		ImportedCaseIDs: importedIDs,
	}, nil
}
