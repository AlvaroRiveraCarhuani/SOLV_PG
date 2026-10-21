package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/logging"
	"solv-backend/internal/infrastructure/metrics"
	"strings"
	"time"
)

func (s *EvaluationService) Evaluate(ctx context.Context, exerciseID string, language string, sourceCodeB64 string) (*domain.EvaluationResult, error) {
	evalStart := time.Now()
	metrics.EvaluatorQueueDepth.Inc()
	defer metrics.EvaluatorQueueDepth.Dec()

	studentID := ""
	if uid, ok := ctx.Value(domain.UserIDKey).(string); ok && uid != "" {
		studentID = uid
	}

	logging.FromContext(ctx).Info("evaluation started",
		slog.String("exercise_id", exerciseID),
		slog.String("student_id", studentID),
		slog.String("language", language),
	)

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

	// 2.5 Verificar si el módulo está bloqueado para el estudiante (fail-closed, salvaguarda de examen)
	if exercise.ModuleID != nil && *exercise.ModuleID != "" && exercise.Purpose != string(domain.ExercisePurposeExam) && s.moduleRepo != nil {
		tenantID := domain.GetTenantID(ctx)
		if studentID != "" {
			isLocked, err := s.moduleRepo.IsModuleLockedForStudent(ctx, tenantID, *exercise.ModuleID, studentID)
			if err == nil && isLocked {
				return nil, domain.ErrModuleLocked
			}
		}
	}

	// 3. Ramificar evaluación según el Tipo de Ejercicio
	var result *domain.EvaluationResult
	var evalErr error
	if exercise.Type == domain.ExerciseTypeDatabase {
		result, evalErr = s.evaluateDatabase(ctx, exercise, sourceCode)
	} else {
		result, evalErr = s.evaluateAlgorithm(ctx, exercise, language, sourceCode)
	}

	evalDuration := time.Since(evalStart)
	if result != nil {
		courseID := ""
		if exercise.SubjectID != nil {
			courseID = *exercise.SubjectID
		}
		verdict := string(result.Verdict)
		metrics.SubmissionsTotal.WithLabelValues(courseID, language, verdict).Inc()
		metrics.SubmissionDurationSeconds.WithLabelValues(language).Observe(evalDuration.Seconds())

		logging.FromContext(ctx).Info("evaluation finished",
			slog.String("exercise_id", exerciseID),
			slog.String("course_id", courseID),
			slog.String("student_id", studentID),
			slog.String("language", language),
			slog.String("verdict", verdict),
			slog.Int("execution_time_ms", result.ExecutionTimeMS),
			slog.Float64("memory_used_mb", result.MemoryUsedMB),
			slog.Float64("duration_ms", float64(evalDuration.Microseconds())/1000.0),
		)
	}

	return result, evalErr
}

// GenerateStudentSeed calcula una semilla pseudoaleatoria de 64 bits determinista por ejercicio y estudiante.
func GenerateStudentSeed(exerciseID, studentID string) int64 {
	h := sha256.Sum256([]byte(exerciseID + ":" + studentID))
	return int64(binary.BigEndian.Uint64(h[:8]))
}

func (s *EvaluationService) evaluateAlgorithm(ctx context.Context, exercise *domain.Exercise, language string, sourceCode string) (*domain.EvaluationResult, error) {
	cfg := exercise.Config.Algorithm
	if cfg == nil {
		return nil, fmt.Errorf("configuración de algoritmia faltante para el ejercicio %s", exercise.ID)
	}
	if exercise.Template == nil {
		s.attachTemplateSummary(ctx, exercise)
	}
	var tplImage, tplEntrypoint string
	if exercise.Template != nil {
		tplImage = exercise.Template.DockerImage
		tplEntrypoint = exercise.Template.Entrypoint
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

	// 2.5 Generación de casos deterministas por estudiante en exámenes
	testCasesToRun := cfg.TestCases
	var generatedCases []domain.TestCase
	isExamWithSeed := exercise.PerStudentSeed && exercise.Purpose == string(domain.ExercisePurposeExam)

	if isExamWithSeed {
		studentID := "anonymous"
		if uid, ok := ctx.Value(domain.UserIDKey).(string); ok && uid != "" {
			studentID = uid
		}
		seed := GenerateStudentSeed(exercise.ID, studentID)

		var inputFormat json.RawMessage
		if len(cfg.InputFormat) > 0 && string(cfg.InputFormat) != "null" {
			inputFormat = cfg.InputFormat
		} else if len(exercise.Config.InputFormat) > 0 && string(exercise.Config.InputFormat) != "null" {
			inputFormat = exercise.Config.InputFormat
		}

		if len(inputFormat) > 0 && s.formatValidator != nil {
			count := len(cfg.TestCases)
			if count <= 0 {
				count = 5
			}
			generatedCases = make([]domain.TestCase, 0, count)
			for i := 0; i < count; i++ {
				caseSeed := seed + int64(i*10007)
				genInput, err := s.formatValidator.GenerateCase(inputFormat, caseSeed)
				if err != nil {
					return nil, fmt.Errorf("error generando caso determinista #%d: %w", i+1, err)
				}

				expectedOut := ""
				if strings.TrimSpace(exercise.ReferenceSolution) != "" {
					refRunConfig := domain.EvaluationRunConfig{
						Language:      exercise.Language,
						SourceCode:    exercise.ReferenceSolution,
						MemoryLimitMB: cfg.MemoryLimitMB,
						TimeLimitMS:   cfg.TimeLimitMS,
						TestCase: domain.TestCase{
							Input: genInput,
						},
						Comparator:  cfg.Comparator,
						DockerImage: tplImage,
						Entrypoint:  tplEntrypoint,
					}
					refRes, refErr := s.runner.RunTestCase(ctx, refRunConfig)
					if refErr == nil {
						expectedOut = refRes.ActualOutput
					}
				}

				tc := domain.TestCase{
					ID:             fmt.Sprintf("gen-%d", i+1),
					ExerciseID:     exercise.ID,
					OrderIndex:     i + 1,
					Input:          genInput,
					ExpectedOutput: expectedOut,
					Visibility:     domain.TestCaseVisibilityHidden,
					Weight:         1.0,
					IsHidden:       true,
				}
				if i < len(cfg.TestCases) {
					tc.Weight = cfg.TestCases[i].Weight
					tc.Visibility = cfg.TestCases[i].Visibility
					tc.IsHidden = (tc.Visibility == domain.TestCaseVisibilityHidden || cfg.TestCases[i].IsHidden)
				}
				generatedCases = append(generatedCases, tc)
			}
			testCasesToRun = generatedCases
		}
	}

	// 3. Ejecución de casos de prueba: todos los casos sin detención
	// temprana (D-EJ-03, corrige DESVÍO-01). El veredicto global es el
	// primer veredicto distinto de AC; el detalle por caso va en CaseResults.
	var totalExecutionTime time.Duration
	var maxMemoryUsedMB float64
	caseResults := make([]domain.CaseResult, 0, len(testCasesToRun))
	globalVerdict := domain.VerdictAC
	var firstFailed *domain.TestCase
	failCount := 0

	for idx, tc := range testCasesToRun {
		runConfig := domain.EvaluationRunConfig{
			Language:      language,
			SourceCode:    sourceCode,
			MemoryLimitMB: cfg.MemoryLimitMB,
			TimeLimitMS:   cfg.TimeLimitMS,
			TestCase:      tc,
			Comparator:    cfg.Comparator,
			DockerImage:   tplImage,
			Entrypoint:    tplEntrypoint,
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
				if isExamWithSeed || tc.Visibility == domain.TestCaseVisibilityHidden || (tc.Visibility == "" && tc.IsHidden) {
					failedTC.Input = "[OCULTO POR EXAMEN]"
					failedTC.ExpectedOutput = "[OCULTO POR EXAMEN]"
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
			Message:         fmt.Sprintf("Fallaron %d de %d casos de prueba", failCount, len(testCasesToRun)),
			FailedTestCase:  firstFailed,
			CaseResults:     caseResults,
			GeneratedCases:  generatedCases,
		}, nil
	}

	return &domain.EvaluationResult{
		Verdict:         domain.VerdictAC,
		ExecutionTimeMS: int(totalExecutionTime.Milliseconds()),
		MemoryUsedMB:    maxMemoryUsedMB,
		Message:         "¡Solución Aceptada! Todos los casos de prueba pasaron exitosamente.",
		CaseResults:     caseResults,
		GeneratedCases:  generatedCases,
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
