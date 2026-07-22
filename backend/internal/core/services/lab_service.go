package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

type LabService struct {
	repo      domain.LabInstanceRepository
	templates domain.TemplateRepository
	docker    domain.ContainerOrchestrator
}

func NewLabService(repo domain.LabInstanceRepository, templates domain.TemplateRepository, docker domain.ContainerOrchestrator) *LabService {
	return &LabService{
		repo:      repo,
		templates: templates,
		docker:    docker,
	}
}

func (s *LabService) StartLab(ctx context.Context, userID string, templateID string, ramLimitMB int, userEmail string) (*domain.LabInstance, error) {
	// 1. Idempotency Check
	existing, err := s.repo.GetByUserAndTemplate(ctx, userID, templateID)
	if err == nil && existing != nil {
		if existing.Status == "active" {
			return existing, nil
		}
	}

	// 2. Create pending record
	id := uuid.NewString()
	instance := &domain.LabInstance{
		ID:         id,
		UserID:     userID,
		TemplateID: templateID,
		Status:     "pending",
		RAMLimitMB: ramLimitMB,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create pending instance: %w", err)
	}

	// 3. Docker Adapter Call
	// Generate Traefik Labels
	prefix := strings.Split(userEmail, "@")[0]
	subdomain := fmt.Sprintf("%s-lab.solv.uab.edu.bo", prefix)
	containerName := fmt.Sprintf("solv-lab-%s", id)

	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.lab-%s.rule", containerName):                      fmt.Sprintf("Host(`%s`)", subdomain),
		fmt.Sprintf("traefik.http.services.lab-%s.loadbalancer.server.port", containerName): "8080", // Using 8080 as internal default or configurable
	}

	template, err := s.templates.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template for image resolution: %w", err)
	}

	config := domain.LabContainerConfig{
		Image:         template.DockerImage,
		ContainerName: containerName,
		VolumeName:    fmt.Sprintf("solv_vol_%s_%s", userID, templateID),
		MemoryLimitMB: int64(ramLimitMB),
		NetworkMode:   "bridge",
		ReadOnly:      false,
		Labels:        labels,
	}

	if err := s.docker.EnsureVolumeExists(ctx, config.VolumeName); err != nil {
		s.repo.UpdateStatus(ctx, id, "failed")
		return nil, fmt.Errorf("failed to ensure volume: %w", err)
	}

	containerID, err := s.docker.StartContainer(ctx, config)
	if err != nil {
		s.repo.UpdateStatus(ctx, id, "failed")
		return nil, fmt.Errorf("docker start failed: %w", err)
	}

	// 4. Update Database
	if err := s.repo.UpdateContainerID(ctx, id, containerID); err != nil {
		return nil, fmt.Errorf("failed to update container ID: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, id, "active"); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	instance.ContainerID = &containerID
	instance.Status = "active"

	return instance, nil
}

func (s *LabService) DestroyLab(ctx context.Context, id string) error {
	instance, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("lab instance %s not found: %w", id, err)
	}

	if instance.ContainerID != nil && *instance.ContainerID != "" {
		if err := s.docker.StopAndRemoveContainer(ctx, *instance.ContainerID); err != nil {
			return fmt.Errorf("failed to stop and remove container: %w", err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, id, "inactive"); err != nil {
		return fmt.Errorf("failed to update lab status to inactive: %w", err)
	}

	return nil
}
