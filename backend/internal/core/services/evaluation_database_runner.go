package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"
)

func (s *EvaluationService) evaluateDatabase(ctx context.Context, exercise *domain.Exercise, solutionSQL string) (*domain.EvaluationResult, error) {
	cfg := exercise.Config.Database
	if cfg == nil {
		return nil, fmt.Errorf("configuración de base de datos faltante para el ejercicio %s", exercise.ID)
	}

	if cfg.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(cfg.MemoryLimitMB, maxAllowed); err != nil {
			return nil, err
		}
	}

	// Si no tiene expected_json aún en DB, realizamos Dry Run previo
	expectedJSON := strings.TrimSpace(cfg.ExpectedJSON)
	if expectedJSON == "" {
		dryRunJSON, err := s.ExecuteDBDryRun(ctx, domain.DBEvaluationRunConfig{
			Engine:          cfg.Engine,
			InitScript:      cfg.InitScript,
			SolutionSQL:     cfg.ReferenceSolution,
			ValidationQuery: cfg.ValidationQuery,
			TimeLimitMS:     cfg.TimeLimitMS,
			MemoryLimitMB:   cfg.MemoryLimitMB,
		})
		if err != nil {
			return nil, fmt.Errorf("falló el Dry Run para generar expected_json: %w", err)
		}
		expectedJSON = strings.TrimSpace(dryRunJSON)
		cfg.ExpectedJSON = expectedJSON
		_ = s.exerciseRepo.UpdateExpectedJSON(ctx, exercise.ID, expectedJSON)
	}

	// Ejecutar solución del alumno en contenedor DB efímero
	dbRunCfg := domain.DBEvaluationRunConfig{
		Engine:          cfg.Engine,
		InitScript:      cfg.InitScript,
		SolutionSQL:     solutionSQL,
		ValidationQuery: cfg.ValidationQuery,
		TimeLimitMS:     cfg.TimeLimitMS,
		MemoryLimitMB:   cfg.MemoryLimitMB,
	}

	res, err := s.runner.RunDBEvaluation(ctx, dbRunCfg)
	if err != nil {
		return nil, fmt.Errorf("error al evaluar solución de BD: %w", err)
	}

	if res.Verdict != domain.VerdictAC {
		return &domain.EvaluationResult{
			Verdict:         res.Verdict,
			ExecutionTimeMS: int(res.ExecutionTime.Milliseconds()),
			Message:         res.ErrorDetails,
			ActualJSON:      res.ResultingJSON,
			ExpectedJSON:    expectedJSON,
		}, nil
	}

	// Comparar JSON resultante contra expected_json
	actualJSONTrim := strings.TrimSpace(res.ResultingJSON)
	if actualJSONTrim == expectedJSON {
		return &domain.EvaluationResult{
			Verdict:         domain.VerdictAC,
			ExecutionTimeMS: int(res.ExecutionTime.Milliseconds()),
			Message:         "¡Solución de Base de Datos Aceptada! El estado resultante coincide exactamente.",
			ActualJSON:      actualJSONTrim,
			ExpectedJSON:    expectedJSON,
		}, nil
	}

	return &domain.EvaluationResult{
		Verdict:         domain.VerdictWA,
		ExecutionTimeMS: int(res.ExecutionTime.Milliseconds()),
		Message:         fmt.Sprintf("Wrong Answer: El estado de la base de datos no coincide. Esperado: %s, Obtenido: %s", expectedJSON, actualJSONTrim),
		ActualJSON:      actualJSONTrim,
		ExpectedJSON:    expectedJSON,
	}, nil
}

// ExecuteDBDryRun ejecuta el dry-run para base de datos validando la memoria contra el host (D-EJ-04).
func (s *EvaluationService) ExecuteDBDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	if config.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(config.MemoryLimitMB, maxAllowed); err != nil {
			return "", err
		}
	}
	return s.runner.RunDBDryRun(ctx, config)
}

type CalculateOutputsRequest struct {
	Language      string   `json:"language"`
	SourceCode    string   `json:"source_code"`
	Inputs        []string `json:"inputs"`
	TimeLimitMS   int      `json:"time_limit_ms,omitempty"`
	MemoryLimitMB int      `json:"memory_limit_mb,omitempty"`
}

type CalculatedOutputItem struct {
	Index           int    `json:"index"`
	Input           string `json:"input"`
	ExpectedOutput  string `json:"expected_output"`
	Status          string `json:"status"` // "ok", "error", "timeout"
	ExecutionTimeMS int    `json:"execution_time_ms"`
	ErrorDetails    string `json:"error_details,omitempty"`
}

type CalculateOutputsResponse struct {
	Outputs []CalculatedOutputItem `json:"outputs"`
}

func (s *EvaluationService) CalculateOutputs(ctx context.Context, req CalculateOutputsRequest) (*CalculateOutputsResponse, error) {
	if strings.TrimSpace(req.SourceCode) == "" {
		return nil, errors.New("la solución de referencia es obligatoria")
	}
	if req.TimeLimitMS <= 0 {
		req.TimeLimitMS = 2000
	}
	if req.MemoryLimitMB <= 0 {
		req.MemoryLimitMB = 256
	}
	resp := &CalculateOutputsResponse{
		Outputs: make([]CalculatedOutputItem, 0, len(req.Inputs)),
	}
	for i, inp := range req.Inputs {
		runCfg := domain.EvaluationRunConfig{
			Language:      req.Language,
			SourceCode:    req.SourceCode,
			MemoryLimitMB: req.MemoryLimitMB,
			TimeLimitMS:   req.TimeLimitMS,
			TestCase: domain.TestCase{
				Input: inp,
			},
		}
		res, err := s.runner.RunTestCase(ctx, runCfg)
		if err != nil {
			resp.Outputs = append(resp.Outputs, CalculatedOutputItem{
				Index:        i,
				Input:        inp,
				Status:       "error",
				ErrorDetails: err.Error(),
			})
			continue
		}
		status := "ok"
		switch res.Verdict {
		case domain.VerdictTLE:
			status = "timeout"
		case domain.VerdictRE, domain.VerdictCE:
			status = "error"
		}
		resp.Outputs = append(resp.Outputs, CalculatedOutputItem{
			Index:           i,
			Input:           inp,
			ExpectedOutput:  res.ActualOutput,
			Status:          status,
			ExecutionTimeMS: int(res.ExecutionTime.Milliseconds()),
			ErrorDetails:    res.ErrorDetails,
		})
	}
	return resp, nil
}

func (s *EvaluationService) GenerateChecklist(ctx context.Context, exerciseID string, refInput *domain.ReferenceSolutionInput) (*domain.ChecklistReport, error) {
	exercise, err := s.exerciseRepo.GetByID(ctx, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener ejercicio: %w", err)
	}

	report := &domain.ChecklistReport{
		Blockers:   make([]string, 0),
		Warnings:   make([]string, 0),
		Info:       make([]string, 0),
		CanPublish: true,
	}

	if exercise.Config.Algorithm == nil {
		report.Blockers = append(report.Blockers, "El ejercicio no tiene configuración de algoritmo válida.")
		report.CanPublish = false
		return report, nil
	}

	testCases := exercise.Config.Algorithm.TestCases
	if len(testCases) == 0 {
		report.Blockers = append(report.Blockers, "Debes incluir al menos un caso de prueba.")
	}

	for i, tc := range testCases {
		if strings.TrimSpace(tc.ExpectedOutput) == "" {
			report.Blockers = append(report.Blockers, fmt.Sprintf("El caso #%d tiene la salida esperada vacía.", i+1))
		}
	}

	// Validar input_format si está presente
	var inputFormat json.RawMessage
	if len(exercise.Config.Algorithm.InputFormat) > 0 && string(exercise.Config.Algorithm.InputFormat) != "null" {
		inputFormat = exercise.Config.Algorithm.InputFormat
	} else if len(exercise.Config.InputFormat) > 0 && string(exercise.Config.InputFormat) != "null" {
		inputFormat = exercise.Config.InputFormat
	}

	if len(inputFormat) > 0 && s.formatValidator != nil {
		if err := s.formatValidator.ValidateContract(inputFormat); err != nil {
			report.Blockers = append(report.Blockers, fmt.Sprintf("El contrato input_format tiene errores de estructura: %s", err.Error()))
		} else {
			for i, tc := range testCases {
				valid, errMsg := s.formatValidator.ValidateCase(inputFormat, tc.Input)
				if !valid {
					report.Blockers = append(report.Blockers, fmt.Sprintf("Caso #%d no cumple con input_format: %s", i+1, errMsg))
				}
			}
		}
	}

	if strings.TrimSpace(exercise.Language) == "" {
		report.Blockers = append(report.Blockers, "No se ha definido el lenguaje del ejercicio.")
	}

	// Solución de referencia y dry-run
	refCode := exercise.ReferenceSolution
	refLang := exercise.Language
	if refInput != nil && strings.TrimSpace(refInput.Code) != "" {
		refCode = refInput.Code
		if strings.TrimSpace(refInput.Language) != "" {
			refLang = refInput.Language
		}
	}

	var refExecutionTimeMS = 0
	if strings.TrimSpace(refCode) == "" {
		report.Warnings = append(report.Warnings, "No se ha configurado la solución de referencia del docente.")
	} else if len(testCases) > 0 && len(report.Blockers) == 0 && s.runner != nil {
		dryRunPassed := true
		var maxDurationMS = 0
		for i, tc := range testCases {
			timeLimit := exercise.TimeLimitMS
			if timeLimit <= 0 {
				timeLimit = 2000
			}
			memLimit := exercise.MemoryLimitMB
			if memLimit <= 0 {
				memLimit = 256
			}

			runCfg := domain.EvaluationRunConfig{
				Language:      refLang,
				SourceCode:    refCode,
				MemoryLimitMB: memLimit,
				TimeLimitMS:   timeLimit,
				TestCase: domain.TestCase{
					Input:          tc.Input,
					ExpectedOutput: tc.ExpectedOutput,
				},
			}
			res, runErr := s.runner.RunTestCase(ctx, runCfg)
			if runErr != nil || res.Verdict != domain.VerdictAC {
				dryRunPassed = false
				v := string(res.Verdict)
				if v == "" {
					v = "Error de ejecución"
				}
				if res.ErrorDetails != "" {
					v += " (" + res.ErrorDetails + ")"
				}
				report.Blockers = append(report.Blockers, fmt.Sprintf("La solución de referencia falló en el caso #%d (Resultado: %s).", i+1, v))
				break
			}
			dur := int(res.ExecutionTime.Milliseconds())
			if dur > maxDurationMS {
				maxDurationMS = dur
			}
		}
		if dryRunPassed {
			refExecutionTimeMS = maxDurationMS
			report.Info = append(report.Info, fmt.Sprintf("Solución de referencia verificada con éxito en todos los casos (%d ms máx).", refExecutionTimeMS))
		}
	}

	// Estadísticas y advertencias
	exampleCount := 0
	publicCount := 0
	hiddenCount := 0
	visibleWeight := 0.0
	hiddenWeight := 0.0

	for _, tc := range testCases {
		tc.Normalize()
		switch tc.Visibility {
		case domain.TestCaseVisibilityExample:
			exampleCount++
			visibleWeight += tc.Weight
		case domain.TestCaseVisibilityPublic:
			publicCount++
			visibleWeight += tc.Weight
		case domain.TestCaseVisibilityHidden:
			hiddenCount++
			hiddenWeight += tc.Weight
		}
	}

	if hiddenCount == 0 {
		report.Warnings = append(report.Warnings, "Sin casos ocultos: un alumno podría aprobar mostrando salidas fijas en consola.")
	}

	hasBranchingTags := false
	for _, t := range exercise.Tags {
		tl := strings.ToLower(t)
		if strings.Contains(tl, "condicional") || strings.Contains(tl, "ciclo") || strings.Contains(tl, "bucle") || strings.Contains(tl, "loop") {
			hasBranchingTags = true
			break
		}
	}
	if hasBranchingTags && hiddenCount < 3 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("El ejercicio tiene tags de lógica de control pero solo %d caso(s) oculto(s) (se recomiendan al menos 3).", hiddenCount))
	}

	if exampleCount == 0 {
		report.Warnings = append(report.Warnings, "0 casos de ejemplo: el estudiante no verá casos ilustrativos en el enunciado.")
	}

	if exercise.PerStudentSeed && exercise.Purpose != string(domain.ExercisePurposeExam) {
		report.Warnings = append(report.Warnings, "La semilla por estudiante está activa para una práctica regular (recomendada solo en exámenes).")
	}

	if visibleWeight > hiddenWeight && hiddenCount > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("El peso de casos visibles (%.1f) supera al de casos ocultos (%.1f).", visibleWeight, hiddenWeight))
	}

	if refExecutionTimeMS > 0 && exercise.TimeLimitMS > 0 && exercise.TimeLimitMS < (2*refExecutionTimeMS) {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Calibración de tiempo: el límite configurado (%d ms) es muy ajustado (< 2× el tiempo de referencia de %d ms).", exercise.TimeLimitMS, refExecutionTimeMS))
	}

	report.Info = append(report.Info, fmt.Sprintf("Distribución de casos: %d total (%d ejemplos, %d públicos, %d ocultos).", len(testCases), exampleCount, publicCount, hiddenCount))
	report.CanPublish = len(report.Blockers) == 0

	return report, nil
}
