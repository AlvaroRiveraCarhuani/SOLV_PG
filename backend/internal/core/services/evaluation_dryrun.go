package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"strings"
	"time"
)

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
	s.attachTemplateSummary(ctx, ex)
	// Dry-run asincrono desacoplado del ctx del request para evitar cancelacion prematura.
	//nolint:gosec
	go func(jID, eID, tID string, exerciseCopy domain.Exercise) {
		bgCtx := context.Background()
		if exerciseCopy.Template == nil {
			s.attachTemplateSummary(bgCtx, &exerciseCopy)
		}
		var tplImage, tplEntrypoint string
		if exerciseCopy.Template != nil {
			tplImage = exerciseCopy.Template.DockerImage
			tplEntrypoint = exerciseCopy.Template.Entrypoint
		}
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
				DockerImage:   tplImage,
				Entrypoint:    tplEntrypoint,
			}
			res, err := s.runner.RunTestCase(bgCtx, runCfg)
			if err != nil {
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
