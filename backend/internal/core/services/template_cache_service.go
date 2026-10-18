package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

var (
	ErrTemplateNotFound = errors.New("template not found")
)

type TemplateCacheService struct {
	govRepo domain.AdminGovernanceRepository
	docker  domain.WorkspaceOrchestrator
	warmMu  sync.Mutex
}

func NewTemplateCacheService(govRepo domain.AdminGovernanceRepository, docker domain.WorkspaceOrchestrator) *TemplateCacheService {
	return &TemplateCacheService{
		govRepo: govRepo,
		docker:  docker,
	}
}

// GetTemplatesCacheStatus retorna el estado de presencia física en el almacenamiento local del host de todas las plantillas activas
func (s *TemplateCacheService) GetTemplatesCacheStatus(ctx context.Context, tenantID string) ([]domain.TemplateCacheStatus, error) {
	templates, err := s.govRepo.ListTemplates(ctx, tenantID, "approved", "")
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	result := make([]domain.TemplateCacheStatus, 0, len(templates))
	for _, t := range templates {
		if t == nil {
			continue
		}
		status := domain.TemplateCacheStatus{
			TemplateID:   t.ID,
			TemplateName: t.Name,
			DockerImage:  t.DockerImage,
		}

		if s.docker == nil {
			status.Status = "missing"
			status.IsCached = false
			result = append(result, status)
			continue
		}

		isCached, size, inspectErr := s.docker.InspectLocalImage(ctx, t.DockerImage)
		if inspectErr != nil {
			status.IsCached = false
			status.Status = "failed"
			status.ErrorMessage = inspectErr.Error()
		} else if isCached {
			status.IsCached = true
			status.SizeBytes = size
			status.Status = "cached"
		} else {
			status.IsCached = false
			status.SizeBytes = 0
			status.Status = "missing"
		}
		result = append(result, status)
	}

	return result, nil
}

// WarmTemplate precalienta una plantilla individual descargándola al almacenamiento del host si no reside en él
func (s *TemplateCacheService) WarmTemplate(ctx context.Context, tenantID, templateID string) (*domain.TemplateCacheStatus, error) {
	template, err := s.govRepo.GetTemplateByID(ctx, templateID)
	if err != nil || template == nil {
		return nil, ErrTemplateNotFound
	}

	status := &domain.TemplateCacheStatus{
		TemplateID:   template.ID,
		TemplateName: template.Name,
		DockerImage:  template.DockerImage,
	}

	if s.docker == nil {
		return nil, errors.New("docker orchestrator not configured")
	}

	// 1. Si ya está en disco, retornar directamente
	isCached, size, err := s.docker.InspectLocalImage(ctx, template.DockerImage)
	if err == nil && isCached {
		status.IsCached = true
		status.SizeBytes = size
		status.Status = "cached"
		return status, nil
	}

	// 2. Descargar imagen con deduplicación single-flight
	log.Printf("[CacheWarmer] Warming image %q for template %q (%s)...", template.DockerImage, template.Name, template.ID)
	pullCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := s.docker.PullImage(pullCtx, template.DockerImage); err != nil {
		status.IsCached = false
		status.Status = "failed"
		status.ErrorMessage = err.Error()
		return status, fmt.Errorf("failed to warm image %q: %w", template.DockerImage, err)
	}

	// 3. Reinspeccionar imagen para confirmar presencia y tamaño
	isCached, size, err = s.docker.InspectLocalImage(ctx, template.DockerImage)
	if err != nil || !isCached {
		status.IsCached = false
		status.Status = "failed"
		return status, errors.New("image pulled but inspection failed")
	}

	status.IsCached = true
	status.SizeBytes = size
	status.Status = "cached"
	log.Printf("[CacheWarmer] Image %q successfully cached (%d bytes)", template.DockerImage, size)
	return status, nil
}

// WarmAllActiveTemplates ejecuta un calentamiento secuencial de todas las plantillas activas no cacheadas
func (s *TemplateCacheService) WarmAllActiveTemplates(ctx context.Context, tenantID string) (*domain.WarmAllSummary, error) {
	s.warmMu.Lock()
	defer s.warmMu.Unlock()

	templates, err := s.govRepo.ListTemplates(ctx, tenantID, "approved", "")
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	summary := &domain.WarmAllSummary{
		TotalChecked: len(templates),
	}

	for _, t := range templates {
		if t == nil {
			continue
		}

		if s.docker == nil {
			summary.FailedCount++
			continue
		}

		isCached, _, err := s.docker.InspectLocalImage(ctx, t.DockerImage)
		if err == nil && isCached {
			summary.AlreadyWarm++
			continue
		}

		log.Printf("[CacheWarmer] Warming template %q (image: %s)...", t.Name, t.DockerImage)
		pullCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		pullErr := s.docker.PullImage(pullCtx, t.DockerImage)
		cancel()

		if pullErr != nil {
			log.Printf("[CacheWarmer] Failed to warm %s: %v", t.DockerImage, pullErr)
			summary.FailedCount++
		} else {
			summary.WarmedCount++
		}
	}

	return summary, nil
}
