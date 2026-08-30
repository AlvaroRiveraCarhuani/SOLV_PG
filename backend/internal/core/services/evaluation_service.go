package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

type EvaluationService struct {
	exerciseRepo domain.ExerciseRepository
	astAnalyzer  domain.ASTAnalyzer
	codeScanner  domain.CodeScanner
	runner       domain.EvaluationRunner
}

func NewEvaluationService(
	exerciseRepo domain.ExerciseRepository,
	astAnalyzer domain.ASTAnalyzer,
	codeScanner domain.CodeScanner,
	runner domain.EvaluationRunner,
) *EvaluationService {
	return &EvaluationService{
		exerciseRepo: exerciseRepo,
		astAnalyzer:  astAnalyzer,
		codeScanner:  codeScanner,
		runner:       runner,
	}
}


func (s *EvaluationService) GetExerciseByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return s.exerciseRepo.GetByID(ctx, id)
}

func (s *EvaluationService) CreateExercise(ctx context.Context, ex *domain.Exercise) error {
	if ex.ID == "" {
		ex.ID = fmt.Sprintf("%s", time.Now().Format("20060102150405"))
	}
	return s.exerciseRepo.Create(ctx, ex)
}

func (s *EvaluationService) Evaluate(ctx context.Context, exerciseID string, language string, sourceCodeB64 string) (*domain.EvaluationResult, error) {
	// 1. Decodificar Base64
	decodedBytes, err := base64.StdEncoding.DecodeString(sourceCodeB64)
	if err != nil {
		decodedBytes, err = base64.URLEncoding.DecodeString(sourceCodeB64)
		if err != nil {
			return nil, fmt.Errorf("código fuente en Base64 inválido: %w", err)
		}
	}
	sourceCode := string(decodedBytes)

	// 2. Obtener Ejercicio
	exercise, err := s.exerciseRepo.GetByID(ctx, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("ejercicio no encontrado (ID: %s): %w", exerciseID, err)
	}

	// 3. Ramificar evaluación según el Tipo de Ejercicio
	if exercise.Type == domain.ExerciseTypeDatabase {
		return s.evaluateDatabase(ctx, exercise, sourceCode)
	}

	return s.evaluateAlgorithm(ctx, exercise, language, sourceCode)
}

func (s *EvaluationService) evaluateAlgorithm(ctx context.Context, exercise *domain.Exercise, language string, sourceCode string) (*domain.EvaluationResult, error) {
	cfg := exercise.Config.Algorithm
	if cfg == nil {
		return nil, fmt.Errorf("configuración de algoritmia faltante para el ejercicio %s", exercise.ID)
	}

	// 1. Filtro AST Estático Rápido (Regex)
	if ok, violationMsg := s.astAnalyzer.ValidateCode(language, sourceCode, cfg.ASTRules); !ok {
		return &domain.EvaluationResult{
			Verdict:         domain.VerdictASTViolation,
			ExecutionTimeMS: 0,
			MemoryUsedMB:    0,
			Message:         violationMsg,
		}, nil
	}

	// 2. Pre-chequeo AST Semántico (Semgrep CLI)
	if s.codeScanner != nil {
		scanRes, err := s.codeScanner.ScanCode(sourceCode, language)
		if err != nil {
			return nil, fmt.Errorf("error en pre-chequeo AST Semgrep: %w", err)
		}
		if scanRes != nil && scanRes.HasViolations {
			msg := "Violación de seguridad AST (Semgrep)"
			if len(scanRes.Violations) > 0 {
				msg = fmt.Sprintf("Violación de seguridad AST (Semgrep): %s (Línea %d)", scanRes.Violations[0].Message, scanRes.Violations[0].Line)
			}
			jsonBytes, _ := json.Marshal(scanRes)
			return &domain.EvaluationResult{
				Verdict:         domain.VerdictASTBlocked,
				ExecutionTimeMS: 0,
				MemoryUsedMB:    0,
				Message:         msg,
				ActualJSON:      string(jsonBytes),
			}, nil
		}
	}

	// 2. Ejecución de casos de prueba
	var totalExecutionTime time.Duration
	var maxMemoryUsedMB float64

	for idx, tc := range cfg.TestCases {
		runConfig := domain.EvaluationRunConfig{
			Language:      language,
			SourceCode:    sourceCode,
			MemoryLimitMB: cfg.MemoryLimitMB,
			TimeLimitMS:   cfg.TimeLimitMS,
			TestCase:      tc,
		}

		res, err := s.runner.RunTestCase(ctx, runConfig)
		if err != nil {
			return nil, fmt.Errorf("error del motor de ejecución en caso %d: %w", idx+1, err)
		}

		totalExecutionTime += res.ExecutionTime
		if res.Verdict != domain.VerdictAC {
			failedTC := tc
			if tc.IsHidden {
				failedTC.Input = "[OCULTO]"
				failedTC.ExpectedOutput = "[OCULTO]"
			}

			msg := fmt.Sprintf("Falló en el caso de prueba %d: %s", idx+1, res.Verdict)
			if res.ErrorDetails != "" {
				msg = fmt.Sprintf("%s. Detalle: %s", msg, res.ErrorDetails)
			}

			return &domain.EvaluationResult{
				Verdict:         res.Verdict,
				ExecutionTimeMS: int(res.ExecutionTime.Milliseconds()),
				MemoryUsedMB:    maxMemoryUsedMB,
				Message:         msg,
				FailedTestCase:  &failedTC,
			}, nil
		}
	}

	return &domain.EvaluationResult{
		Verdict:         domain.VerdictAC,
		ExecutionTimeMS: int(totalExecutionTime.Milliseconds()),
		MemoryUsedMB:    maxMemoryUsedMB,
		Message:         "¡Solución Aceptada! Todos los casos de prueba pasaron exitosamente.",
	}, nil
}

func (s *EvaluationService) evaluateDatabase(ctx context.Context, exercise *domain.Exercise, solutionSQL string) (*domain.EvaluationResult, error) {
	cfg := exercise.Config.Database
	if cfg == nil {
		return nil, fmt.Errorf("configuración de base de datos faltante para el ejercicio %s", exercise.ID)
	}

	// Si no tiene expected_json aún en DB, realizamos Dry Run previo
	expectedJSON := strings.TrimSpace(cfg.ExpectedJSON)
	if expectedJSON == "" {
		dryRunJSON, err := s.runner.RunDBDryRun(ctx, domain.DBEvaluationRunConfig{
			Engine:            cfg.Engine,
			InitScript:        cfg.InitScript,
			SolutionSQL:       cfg.ReferenceSolution,
			ValidationQuery:   cfg.ValidationQuery,
			TimeLimitMS:       cfg.TimeLimitMS,
			MemoryLimitMB:     cfg.MemoryLimitMB,
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
