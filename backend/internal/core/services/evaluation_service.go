package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
)

type EvaluationService struct {
	exerciseRepo domain.ExerciseRepository
	astAnalyzer  domain.ASTAnalyzer
	codeScanner  domain.CodeScanner
	runner       domain.EvaluationRunner
	metrics      RunMetricsRecorder
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
		exerciseRepo: exerciseRepo,
		astAnalyzer:  astAnalyzer,
		codeScanner:  codeScanner,
		runner:       runner,
	}
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
	if ex.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(ex.MemoryLimitMB, maxAllowed); err != nil {
			return err
		}
	}
	return s.exerciseRepo.Create(ctx, ex)
}

func (s *EvaluationService) UpdateExercise(ctx context.Context, ex *domain.Exercise) error {
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
		ex.Config.Algorithm = &domain.AlgorithmConfig{}
	}
	ex.Config.Algorithm.TestCases = append(ex.Config.Algorithm.TestCases, testCases...)
	return s.exerciseRepo.UpdateConfig(ctx, exerciseID, tenantID, ex.Config)
}

func (s *EvaluationService) PublishExercise(ctx context.Context, exerciseID, tenantID string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}

	publicCount := 0
	if ex.Config.Algorithm != nil {
		for _, tc := range ex.Config.Algorithm.TestCases {
			if !tc.IsHidden {
				publicCount++
			}
		}
	}

	if publicCount == 0 {
		return nil, ErrZeroPublicTestCases
	}

	if err := s.exerciseRepo.UpdateStatus(ctx, exerciseID, tenantID, "published"); err != nil {
		return nil, fmt.Errorf("failed to publish exercise: %w", err)
	}

	ex.Status = "published"
	return ex, nil
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
				if tc.IsHidden {
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

