package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

var (
	ErrHostMemoryExhausted     = errors.New("host physical memory depleted: admission control denied request (HTTP 503)")
	ErrOOMKilledCooldownPenalty = errors.New("workspace reached 3 consecutive OOMKilled strikes: 5-minute cooldown penalty enforced")
)

type WorkspaceService struct {
	repo        domain.WorkspaceRepository
	docker      domain.WorkspaceOrchestrator
	hostMonitor domain.HostMonitor
}

func NewWorkspaceService(repo domain.WorkspaceRepository, docker domain.WorkspaceOrchestrator, hostMonitor domain.HostMonitor) *WorkspaceService {
	return &WorkspaceService{
		repo:        repo,
		docker:      docker,
		hostMonitor: hostMonitor,
	}
}

func (s *WorkspaceService) StartWorkspace(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	// 1. Verificación de Admisión del Host (Regla del 15% de RAM libre en Host)
	if !s.hostMonitor.CanAllocateMemory(domain.DefaultBaseMemoryMB) {
		return nil, ErrHostMemoryExhausted
	}

	// 2. Verificación de Idempotencia y Estado Previos
	existing, err := s.repo.GetByStudentAndSubject(ctx, studentID, subjectID)
	if err == nil && existing != nil {
		if existing.Status == domain.WorkspaceStatusRunning {
			return existing, nil
		}

		// Verificación de Castigo por OOMKilled (Sistema de 3 Strikes = 5 minutos de bloqueo)
		if existing.OOMStrikeCount >= domain.MaxOOMStrikes && existing.LastOOMKilledAt != nil {
			if time.Since(*existing.LastOOMKilledAt) < domain.OOMCooldownDuration {
				remaining := domain.OOMCooldownDuration - time.Since(*existing.LastOOMKilledAt)
				return nil, fmt.Errorf("%w (%d s remaining)", ErrOOMKilledCooldownPenalty, int(remaining.Seconds()))
			}
			// Pasado el tiempo de penalización, reiniciamos el contador de strikes
			_ = s.repo.ResetOOMStrikes(ctx, existing.ID)
		}

		// Si estaba hibernado u OOMKilled, reactivamos la misma instancia
		return s.reactivateWorkspace(ctx, existing)
	}

	// 3. Generación de UUID opaco para el workspace_id y construcción de access_url
	workspaceID := uuid.NewString()
	accessURL := fmt.Sprintf("http://%s.solv.local", workspaceID)
	containerName := fmt.Sprintf("solv-workspace-%s", workspaceID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", studentID, subjectID)
	networkName := "solv-traefik-net"

	instance := &domain.WorkspaceInstance{
		ID:              workspaceID,
		StudentID:       studentID,
		SubjectID:       subjectID,
		Status:          domain.WorkspaceStatusPending,
		AccessURL:       accessURL,
		MemoryLimitMB:   domain.DefaultBaseMemoryMB,
		LastHeartbeatAt: time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 4. Crear registro pendiente en PostgreSQL mediante sqlx
	if err := s.repo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create pending workspace: %w", err)
	}

	// 5. Asegurar la existencia del Volumen Nombrado (ADR-001) para la materia y estudiante
	if err := s.docker.EnsureVolumeExists(ctx, volumeName); err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to ensure named volume %s: %w", volumeName, err)
	}

	// 6. Asegurar la existencia de la red Docker compartida con Traefik deshabilitando ICC (enable_icc=false)
	if err := s.docker.EnsureICCDisabledNetworkExists(ctx, networkName); err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to ensure ICC-disabled docker network %s: %w", networkName, err)
	}

	// 7. Inyección de Dynamic Labels de Traefik v3
	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", workspaceID):                      fmt.Sprintf("Host(`%s.solv.local`)", workspaceID),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", workspaceID): "8443",
	}

	// 8. Configuración de seguridad del contenedor (linuxserver/code-server con AUTH=none y 256MB base)
	config := domain.WorkspaceContainerConfig{
		Image:         "linuxserver/code-server:latest",
		ContainerName: containerName,
		VolumeName:    volumeName,
		MemoryLimitMB: domain.DefaultBaseMemoryMB,
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

	// 9. Instanciación e inicio del contenedor code-server
	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to start code-server container: %w", err)
	}

	// 10. Actualización del registro en PostgreSQL a estado 'running'
	if err := s.repo.UpdateContainerID(ctx, workspaceID, containerID); err != nil {
		return nil, fmt.Errorf("failed to update container_id: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusRunning); err != nil {
		return nil, fmt.Errorf("failed to update workspace status: %w", err)
	}

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning
	return instance, nil
}

func (s *WorkspaceService) reactivateWorkspace(ctx context.Context, instance *domain.WorkspaceInstance) (*domain.WorkspaceInstance, error) {
	containerName := fmt.Sprintf("solv-workspace-%s", instance.ID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", instance.StudentID, instance.SubjectID)
	networkName := "solv-traefik-net"

	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID):                      fmt.Sprintf("Host(`%s.solv.local`)", instance.ID),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", instance.ID): "8443",
	}

	config := domain.WorkspaceContainerConfig{
		Image:         "linuxserver/code-server:latest",
		ContainerName: containerName,
		VolumeName:    volumeName,
		MemoryLimitMB: domain.DefaultBaseMemoryMB,
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

	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to reactivate workspace container: %w", err)
	}

	_ = s.repo.UpdateContainerID(ctx, instance.ID, containerID)
	_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusRunning)
	_ = s.repo.UpdateMemoryLimit(ctx, instance.ID, domain.DefaultBaseMemoryMB)
	_ = s.repo.RecordHeartbeat(ctx, instance.ID)

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning
	instance.MemoryLimitMB = domain.DefaultBaseMemoryMB
	return instance, nil
}

func (s *WorkspaceService) RecordHeartbeat(ctx context.Context, workspaceID string) error {
	return s.repo.RecordHeartbeat(ctx, workspaceID)
}

func (s *WorkspaceService) RestartWorkspace(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	return s.StartWorkspace(ctx, ws.StudentID, ws.SubjectID)
}
