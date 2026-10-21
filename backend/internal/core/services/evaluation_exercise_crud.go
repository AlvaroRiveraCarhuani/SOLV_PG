package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
	"strings"
)

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

func (s *EvaluationService) resolveAndApplyTemplate(ctx context.Context, ex *domain.Exercise) error {
	if s.adminGovRepo == nil {
		return nil
	}
	if ex.EnvironmentType == "" {
		ex.EnvironmentType = string(domain.EnvironmentTypeJuezEfimero)
	}

	var template *domain.AdminTemplateReviewItem
	var err error

	if ex.TemplateID != "" {
		template, err = s.adminGovRepo.GetTemplateByID(ctx, ex.TemplateID)
		if err != nil {
			return fmt.Errorf("failed to resolve template %s: %w", ex.TemplateID, err)
		}
	} else if ex.Language != "" {
		template, err = s.adminGovRepo.GetTemplateForLanguage(ctx, ex.EnvironmentType, ex.Language)
		if err != nil {
			return fmt.Errorf("failed to resolve default template for language %s: %w", ex.Language, err)
		}
	}

	if template != nil {
		if !strings.EqualFold(template.Status, "approved") && !strings.EqualFold(template.Status, "APROBADA") {
			return domain.ErrTemplateNotApproved
		}
		if !domain.TemplateSupportsEnvironment(template.TargetEnvironment, ex.EnvironmentType) {
			return domain.ErrTemplateEnvironmentMismatch
		}

		ex.TemplateID = template.ID
		ramMB := template.BaseRamMB
		if template.TargetEnvironment == domain.TargetEnvironmentAmbos && ex.EnvironmentType == domain.TargetEnvironmentJuezEfimero {
			ramMB = domain.ResolveJudgeRamMB(template.BaseRamMB)
		}
		ex.MemoryLimitMB = ramMB
		if ex.Config.Algorithm != nil {
			ex.Config.Algorithm.MemoryLimitMB = ramMB
		}
		ex.Template = &domain.TemplateSummary{
			ID:                template.ID,
			Name:              template.Name,
			DockerImage:       template.DockerImage,
			Entrypoint:        template.Entrypoint,
			BaseRamMB:         template.BaseRamMB,
			TargetEnvironment: template.TargetEnvironment,
			ServicesConfig:    template.ServicesConfig,
		}
	}
	return nil
}

func (s *EvaluationService) attachTemplateSummary(ctx context.Context, ex *domain.Exercise) {
	if s.adminGovRepo == nil || ex == nil {
		return
	}
	var tpl *domain.AdminTemplateReviewItem
	var err error
	if ex.TemplateID != "" {
		tpl, err = s.adminGovRepo.GetTemplateByID(ctx, ex.TemplateID)
	} else if ex.Language != "" {
		envType := ex.EnvironmentType
		if envType == "" {
			envType = string(domain.EnvironmentTypeJuezEfimero)
		}
		tpl, err = s.adminGovRepo.GetTemplateForLanguage(ctx, envType, ex.Language)
	}
	if err == nil && tpl != nil {
		ex.Template = &domain.TemplateSummary{
			ID:                tpl.ID,
			Name:              tpl.Name,
			DockerImage:       tpl.DockerImage,
			Entrypoint:        tpl.Entrypoint,
			BaseRamMB:         tpl.BaseRamMB,
			TargetEnvironment: tpl.TargetEnvironment,
			ServicesConfig:    tpl.ServicesConfig,
		}
	}
}

func (s *EvaluationService) GetExerciseByID(ctx context.Context, id string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByID(ctx, id)
	if err != nil || ex == nil {
		return ex, err
	}
	s.attachTemplateSummary(ctx, ex)
	return ex, nil
}

func (s *EvaluationService) GetExerciseByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil || ex == nil {
		return ex, err
	}
	s.attachTemplateSummary(ctx, ex)
	return ex, nil
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
	if err := s.resolveAndApplyTemplate(ctx, ex); err != nil {
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
	if err := s.resolveAndApplyTemplate(ctx, ex); err != nil {
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

	if err := s.resolveAndApplyTemplate(ctx, ex); err != nil {
		return nil, err
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
