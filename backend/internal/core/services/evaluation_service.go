package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
)

type EvaluationService struct {
	exerciseRepo    domain.ExerciseRepository
	astAnalyzer     domain.ASTAnalyzer
	codeScanner     domain.CodeScanner
	runner          domain.EvaluationRunner
	metrics         RunMetricsRecorder
	formatValidator FormatValidator
}

// RunMetricsRecorder persiste la telemetria por caso (tabla run_metrics,
// migracion 00012). Es opcional: sin registrador la evaluacion omite el
// registro pero evalua todos los casos igual.
type RunMetricsRecorder interface {
	RecordRunMetric(ctx context.Context, metric domain.RunMetric) error
}

// P95Query calcula el p95 de duration_ms sobre veredictos AC por lenguaje en
// la ventana de language_profiles.p95_window_days (design.md, D-EJ-04).
const P95Query = `SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) FROM run_metrics WHERE language = $1 AND verdict = 'AC' AND created_at > now() - ($2 || ' days')::interval`

// SetMetricsRecorder activa el registro de run_metrics por caso.
func (s *EvaluationService) SetMetricsRecorder(r RunMetricsRecorder) {
	s.metrics = r
}

// CanonicalLanguage normaliza el alias a clave canonica (c++ -> cpp,
// c#/cs -> csharp). Los perfiles y run_metrics rechazan alias (spec 3.1).
func CanonicalLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "c++":
		return "cpp"
	case "c#", "cs":
		return "csharp"
	default:
		return strings.ToLower(strings.TrimSpace(language))
	}
}

// P95OverAC calcula el p95 de duration_ms sobre veredictos AC con
// nearest-rank. Sin muestras AC devuelve 0.
func P95OverAC(metrics []domain.RunMetric) int {
	samples := make([]int, 0, len(metrics))
	for _, m := range metrics {
		if m.Verdict == domain.VerdictAC {
			samples = append(samples, m.DurationMS)
		}
	}
	if len(samples) == 0 {
		return 0
	}
	sort.Ints(samples)
	rank := (95*len(samples) + 99) / 100
	return samples[rank-1]
}

func NewEvaluationService(
	exerciseRepo domain.ExerciseRepository,
	astAnalyzer domain.ASTAnalyzer,
	codeScanner domain.CodeScanner,
	runner domain.EvaluationRunner,
) *EvaluationService {
	return &EvaluationService{
		exerciseRepo:    exerciseRepo,
		astAnalyzer:     astAnalyzer,
		codeScanner:     codeScanner,
		runner:          runner,
		formatValidator: NewFormatValidator(),
	}
}

func (s *EvaluationService) SetFormatValidator(v FormatValidator) {
	s.formatValidator = v
}

func (s *EvaluationService) validateExerciseInputFormat(ex *domain.Exercise) error {
	if ex == nil || s.formatValidator == nil {
		return nil
	}
	var inputFormat json.RawMessage
	if ex.Config.Algorithm != nil && len(ex.Config.Algorithm.InputFormat) > 0 && string(ex.Config.Algorithm.InputFormat) != "null" {
		inputFormat = ex.Config.Algorithm.InputFormat
	} else if len(ex.Config.InputFormat) > 0 && string(ex.Config.InputFormat) != "null" {
		inputFormat = ex.Config.InputFormat
	}

	if len(inputFormat) == 0 || string(inputFormat) == "null" || string(inputFormat) == "{}" {
		return nil
	}

	if err := s.formatValidator.ValidateContract(inputFormat); err != nil {
		return fmt.Errorf("contrato de formato inválido: %w", err)
	}

	if ex.Config.Algorithm != nil {
		for idx, tc := range ex.Config.Algorithm.TestCases {
			ok, msg := s.formatValidator.ValidateCase(inputFormat, tc.Input)
			if !ok {
				return fmt.Errorf("caso %d inválido según contrato de formato: %s", idx+1, msg)
			}
		}
	}
	return nil
}

func (s *EvaluationService) getHostTotalRAM(ctx context.Context) int {
	totalMB := 8192
	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil && v.Total > 0 {
		totalMB = int(v.Total / (1024 * 1024))
	}
	return totalMB
}

var ErrZeroPublicTestCases = fmt.Errorf("cannot publish exercise with 0 public test cases")

func (s *EvaluationService) GetExerciseByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return s.exerciseRepo.GetByID(ctx, id)
}

func (s *EvaluationService) GetExerciseByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return s.exerciseRepo.GetByIDAndTenant(ctx, id, tenantID)
}

func (s *EvaluationService) CreateExercise(ctx context.Context, ex *domain.Exercise) error {
	if ex.ID == "" {
		ex.ID = uuid.NewString()
	}
	if err := ex.Validate(); err != nil {
		return err
	}
	if err := s.validateExerciseInputFormat(ex); err != nil {
		return err
	}
	if ex.Config.Algorithm != nil {
		for i := range ex.Config.Algorithm.TestCases {
			ex.Config.Algorithm.TestCases[i].Normalize()
			if err := ex.Config.Algorithm.TestCases[i].Validate(); err != nil {
				return err
			}
		}
	}
	if ex.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(ex.MemoryLimitMB, maxAllowed); err != nil {
			return err
		}
	}
	return s.exerciseRepo.Create(ctx, ex)
}

func (s *EvaluationService) UpdateExercise(ctx context.Context, ex *domain.Exercise) error {
	if err := ex.Validate(); err != nil {
		return err
	}
	if err := s.validateExerciseInputFormat(ex); err != nil {
		return err
	}
	if ex.Config.Algorithm != nil {
		for i := range ex.Config.Algorithm.TestCases {
			ex.Config.Algorithm.TestCases[i].Normalize()
			if err := ex.Config.Algorithm.TestCases[i].Validate(); err != nil {
				return err
			}
		}
	}
	if ex.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(ex.MemoryLimitMB, maxAllowed); err != nil {
			return err
		}
	}
	return s.exerciseRepo.Update(ctx, ex)
}

func (s *EvaluationService) BulkAddTestCases(ctx context.Context, exerciseID, tenantID string, testCases []domain.TestCase) error {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return fmt.Errorf("exercise not found: %w", err)
	}

	if ex.Config.Algorithm == nil {
		ex.Config.Algorithm = &domain.AlgorithmConfig{
			TimeLimitMS:   ex.TimeLimitMS,
			MemoryLimitMB: ex.MemoryLimitMB,
		}
	}
	for i := range testCases {
		testCases[i].Normalize()
		if err := testCases[i].Validate(); err != nil {
			return err
		}
	}
	ex.Config.Algorithm.TestCases = append(ex.Config.Algorithm.TestCases, testCases...)
	if err := s.validateExerciseInputFormat(ex); err != nil {
		return err
	}
	return s.exerciseRepo.Update(ctx, ex)
}

func (s *EvaluationService) PublishExercise(ctx context.Context, exerciseID, tenantID string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}

	if ex.Type == domain.ExerciseTypeAlgorithm {
		publicCount := 0
		if ex.Config.Algorithm != nil {
			for i := range ex.Config.Algorithm.TestCases {
				tc := &ex.Config.Algorithm.TestCases[i]
				tc.Normalize()
				if tc.Visibility != domain.TestCaseVisibilityHidden {
					publicCount++
				}
			}
		}

		if publicCount == 0 {
			return nil, ErrZeroPublicTestCases
		}

		if strings.TrimSpace(ex.ReferenceSolution) == "" {
			return nil, domain.ErrMissingReferenceSolution
		}
		if ex.Stale {
			return nil, domain.ErrExerciseStale
		}
	}

	if err := s.exerciseRepo.UpdateStatus(ctx, exerciseID, tenantID, "published"); err != nil {
		return nil, fmt.Errorf("failed to publish exercise: %w", err)
	}

	ex.Status = "published"
	return ex, nil
}

// GetDryRunJob consulta el estado y progreso de un trabajo de comprobación previa.
func (s *EvaluationService) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	return s.exerciseRepo.GetDryRunJob(ctx, jobID)
}

// StartDryRun inicia un trabajo asíncrono de dry-run con la solución de referencia (D-EJ-05).
func (s *EvaluationService) StartDryRun(ctx context.Context, exerciseID, tenantID string) (*domain.DryRunJob, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}

	if strings.TrimSpace(ex.ReferenceSolution) == "" {
		return nil, domain.ErrMissingReferenceSolution
	}

	totalCases := 0
	if ex.Config.Algorithm != nil {
		totalCases = len(ex.Config.Algorithm.TestCases)
	}
	if totalCases == 0 && ex.Type == domain.ExerciseTypeAlgorithm {
		return nil, errors.New("el ejercicio no contiene casos de prueba para ejecutar dry-run")
	}

	// 0. Frontera ValidateRamAgainstHost para dry-run
	if ex.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(ex.MemoryLimitMB, maxAllowed); err != nil {
			return nil, err
		}
	}

	jobID := uuid.NewString()
	job := &domain.DryRunJob{
		ID:              jobID,
		ExerciseID:      exerciseID,
		Status:          domain.DryRunJobStatusQueued,
		ProgressCurrent: 0,
		ProgressTotal:   totalCases,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.exerciseRepo.CreateDryRunJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create dry run job: %w", err)
	}

	// Ejecución asíncrona del Dry-Run con progreso por caso
	go func(jID, eID, tID string, exerciseCopy domain.Exercise) {
		bgCtx := context.Background()
		_ = s.exerciseRepo.UpdateDryRunJobProgress(bgCtx, jID, domain.DryRunJobStatusRunning, 0, totalCases, nil, "")

		cfg := exerciseCopy.Config.Algorithm
		if cfg == nil {
			_ = s.exerciseRepo.UpdateDryRunJobProgress(bgCtx, jID, domain.DryRunJobStatusFailed, 0, totalCases, nil, "configuración de algoritmia ausente")
			return
		}

		var totalDuration time.Duration
		caseResults := make([]domain.CaseResult, 0, len(cfg.TestCases))
		allAC := true

		for idx, tc := range cfg.TestCases {
			runCfg := domain.EvaluationRunConfig{
				Language:      exerciseCopy.Language,
				SourceCode:    exerciseCopy.ReferenceSolution,
				MemoryLimitMB: cfg.MemoryLimitMB,
				TimeLimitMS:   cfg.TimeLimitMS,
				TestCase:      tc,
				Comparator:    cfg.Comparator,
			}
			res, err := s.runner.RunTestCase(bgCtx, runCfg)
			if err != nil {
				allAC = false
				_ = s.exerciseRepo.UpdateDryRunJobProgress(bgCtx, jID, domain.DryRunJobStatusFailed, idx+1, totalCases, nil, err.Error())
				return
			}

			totalDuration += res.ExecutionTime
			caseResults = append(caseResults, domain.CaseResult{
				Index:      idx,
				Verdict:    res.Verdict,
				DurationMS: int(res.ExecutionTime.Milliseconds()),
				Message:    res.ErrorDetails,
			})

			if res.Verdict != domain.VerdictAC {
				allAC = false
			}

			_ = s.exerciseRepo.UpdateDryRunJobProgress(bgCtx, jID, domain.DryRunJobStatusRunning, idx+1, totalCases, &domain.EvaluationResult{
				Verdict:         res.Verdict,
				ExecutionTimeMS: int(totalDuration.Milliseconds()),
				CaseResults:     caseResults,
			}, "")
		}

		finalVerdict := domain.VerdictAC
		if !allAC {
			finalVerdict = domain.VerdictWA
		}

		evalResult := &domain.EvaluationResult{
			Verdict:         finalVerdict,
			ExecutionTimeMS: int(totalDuration.Milliseconds()),
			CaseResults:     caseResults,
		}

		if allAC {
			_ = s.exerciseRepo.UpdateDryRunJobProgress(bgCtx, jID, domain.DryRunJobStatusDone, totalCases, totalCases, evalResult, "")
			_ = s.exerciseRepo.UpdateExerciseLastValidDryRun(bgCtx, eID, tID, time.Now())
		} else {
			_ = s.exerciseRepo.UpdateDryRunJobProgress(bgCtx, jID, domain.DryRunJobStatusFailed, totalCases, totalCases, evalResult, "la solución de referencia no obtuvo AC en todos los casos")
		}
	}(jobID, exerciseID, tenantID, *ex)

	return job, nil
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

	// 0. Frontera ValidateRamAgainstHost para evaluación
	if cfg.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(cfg.MemoryLimitMB, maxAllowed); err != nil {
			return nil, err
		}
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

	// 2. Ejecución de casos de prueba: todos los casos sin detención
	// temprana (D-EJ-03, corrige DESVÍO-01). El veredicto global es el
	// primer veredicto distinto de AC; el detalle por caso va en CaseResults.
	var totalExecutionTime time.Duration
	var maxMemoryUsedMB float64
	caseResults := make([]domain.CaseResult, 0, len(cfg.TestCases))
	globalVerdict := domain.VerdictAC
	var firstFailed *domain.TestCase
	failCount := 0

	for idx, tc := range cfg.TestCases {
		runConfig := domain.EvaluationRunConfig{
			Language:      language,
			SourceCode:    sourceCode,
			MemoryLimitMB: cfg.MemoryLimitMB,
			TimeLimitMS:   cfg.TimeLimitMS,
			TestCase:      tc,
			Comparator:    cfg.Comparator,
		}

		res, err := s.runner.RunTestCase(ctx, runConfig)
		if err != nil {
			return nil, fmt.Errorf("error del motor de ejecución en caso %d: %w", idx+1, err)
		}

		totalExecutionTime += res.ExecutionTime
		caseMsg := ""
		if res.Verdict != domain.VerdictAC {
			failCount++
			if globalVerdict == domain.VerdictAC {
				globalVerdict = res.Verdict
				failedTC := tc
				if tc.Visibility == domain.TestCaseVisibilityHidden || (tc.Visibility == "" && tc.IsHidden) {
					failedTC.Input = "[OCULTO]"
					failedTC.ExpectedOutput = "[OCULTO]"
				}
				firstFailed = &failedTC
			}
			caseMsg = string(res.Verdict)
			if res.ErrorDetails != "" {
				caseMsg = fmt.Sprintf("%s. Detalle: %s", caseMsg, res.ErrorDetails)
			}
		}
		caseResults = append(caseResults, domain.CaseResult{
			Index:      idx,
			Verdict:    res.Verdict,
			DurationMS: int(res.ExecutionTime.Milliseconds()),
			Message:    caseMsg,
		})

		s.recordRunMetric(ctx, exercise.ID, language, res, idx)
	}

	if globalVerdict != domain.VerdictAC {
		return &domain.EvaluationResult{
			Verdict:         globalVerdict,
			ExecutionTimeMS: int(totalExecutionTime.Milliseconds()),
			MemoryUsedMB:    maxMemoryUsedMB,
			Message:         fmt.Sprintf("Fallaron %d de %d casos de prueba", failCount, len(cfg.TestCases)),
			FailedTestCase:  firstFailed,
			CaseResults:     caseResults,
		}, nil
	}

	return &domain.EvaluationResult{
		Verdict:         domain.VerdictAC,
		ExecutionTimeMS: int(totalExecutionTime.Milliseconds()),
		MemoryUsedMB:    maxMemoryUsedMB,
		Message:         "¡Solución Aceptada! Todos los casos de prueba pasaron exitosamente.",
		CaseResults:     caseResults,
	}, nil
}

func (s *EvaluationService) recordRunMetric(ctx context.Context, exerciseID, language string, res domain.TestCaseRunResult, caseIdx int) {
	if s.metrics == nil {
		return
	}
	_ = s.metrics.RecordRunMetric(ctx, domain.RunMetric{
		ExerciseID:  exerciseID,
		Language:    CanonicalLanguage(language),
		ImageDigest: res.ImageDigest,
		DurationMS:  int(res.ExecutionTime.Milliseconds()),
		Verdict:     res.Verdict,
		CaseIndex:   caseIdx,
	})
}

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
		if res.Verdict == domain.VerdictTLE {
			status = "timeout"
		} else if res.Verdict == domain.VerdictRE || res.Verdict == domain.VerdictCE {
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

	var refExecutionTimeMS int = 0
	if strings.TrimSpace(refCode) == "" {
		report.Warnings = append(report.Warnings, "No se ha configurado la solución de referencia del docente.")
	} else if len(testCases) > 0 && len(report.Blockers) == 0 && s.runner != nil {
		dryRunPassed := true
		var maxDurationMS int = 0
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
				report.Blockers = append(report.Blockers, fmt.Sprintf("La solución de referencia no obtuvo AC en el caso #%d (Veredicto: %s).", i+1, v))
				break
			}
			dur := int(res.ExecutionTime.Milliseconds())
			if dur > maxDurationMS {
				maxDurationMS = dur
			}
		}
		if dryRunPassed {
			refExecutionTimeMS = maxDurationMS
			report.Info = append(report.Info, fmt.Sprintf("Solución de referencia validada con AC en todos los casos (%d ms máx).", refExecutionTimeMS))
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
		report.Warnings = append(report.Warnings, "0 casos ocultos: un print fijo podría aprobar este ejercicio.")
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

