package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

type WorkspaceService struct {
	repo   domain.WorkspaceRepository
	docker domain.WorkspaceOrchestrator
}

func NewWorkspaceService(repo domain.WorkspaceRepository, docker domain.WorkspaceOrchestrator) *WorkspaceService {
	return &WorkspaceService{
		repo:   repo,
		docker: docker,
	}
}

func (s *WorkspaceService) StartWorkspace(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	// 1. Verificación de Idempotencia
	existing, err := s.repo.GetByStudentAndSubject(ctx, studentID, subjectID)
	if err == nil && existing != nil {
		if existing.Status == "running" {
			return existing, nil
		}
	}

	// 2. Generación de UUID opaco para el workspace_id y construcción de access_url
	workspaceID := uuid.NewString()
	accessURL := fmt.Sprintf("http://%s.solv.local", workspaceID)
	containerName := fmt.Sprintf("solv-workspace-%s", workspaceID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", studentID, subjectID)
	networkName := "solv-traefik-net"

	instance := &domain.WorkspaceInstance{
		ID:        workspaceID,
		StudentID: studentID,
		SubjectID: subjectID,
		Status:    "pending",
		AccessURL: accessURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 3. Crear registro pendiente en PostgreSQL mediante sqlx
	if err := s.repo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create pending workspace: %w", err)
	}

	// 4. Asegurar la existencia del Volumen Nombrado (ADR-001) para la materia y estudiante
	if err := s.docker.EnsureVolumeExists(ctx, volumeName); err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, "failed")
		return nil, fmt.Errorf("failed to ensure named volume %s: %w", volumeName, err)
	}

	// 5. Asegurar la existencia de la red Docker compartida con Traefik deshabilitando ICC (enable_icc=false)
	if err := s.docker.EnsureICCDisabledNetworkExists(ctx, networkName); err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, "failed")
		return nil, fmt.Errorf("failed to ensure ICC-disabled docker network %s: %w", networkName, err)
	}

	// 6. Inyección de Dynamic Labels de Traefik v3
	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", workspaceID):                      fmt.Sprintf("Host(`%s.solv.local`)", workspaceID),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", workspaceID): "8443",
	}

	// 7. Configuración de seguridad del contenedor (linuxserver/code-server con AUTH=none)
	config := domain.WorkspaceContainerConfig{
		Image:         "linuxserver/code-server:latest",
		ContainerName: containerName,
		VolumeName:    volumeName,
		MemoryLimitMB: 512,
		NetworkName:   networkName,
		Labels:        labels,
		Env: []string{
			"AUTH=none",
			"PASSWORD=",
			"SUDO_PASSWORD=",
			"DEFAULT_WORKSPACE=/workspace",
			"PUID=1000",
			"PGID=1000",
			"TZ=Etc/UTC",
		},
	}

	// 8. Instanciación e inicio del contenedor code-server
	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, "failed")
		return nil, fmt.Errorf("failed to start code-server container: %w", err)
	}

	// 9. Actualización del registro en PostgreSQL a estado 'running' con container_id opaco guardado internamente
	if err := s.repo.UpdateContainerID(ctx, workspaceID, containerID); err != nil {
		return nil, fmt.Errorf("failed to update container_id: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, workspaceID, "running"); err != nil {
		return nil, fmt.Errorf("failed to update workspace status: %w", err)
	}

	instance.ContainerID = &containerID
	instance.Status = "running"
	return instance, nil
}
