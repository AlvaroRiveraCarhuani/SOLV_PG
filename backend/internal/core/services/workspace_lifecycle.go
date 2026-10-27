package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"solv-backend/internal/core/domain"
)

func (s *WorkspaceService) reactivateWorkspace(ctx context.Context, instance *domain.WorkspaceInstance, provision satelliteProvision) (*domain.WorkspaceInstance, error) {
	// Optimización ADR / Memory Governance: intentar despausar el contenedor congelado (instantáneo <50ms)
	if instance.ContainerID != nil && *instance.ContainerID != "" {
		if err := s.docker.UnpauseContainer(ctx, *instance.ContainerID); err == nil {
			_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusRunning)
			_ = s.repo.RecordHeartbeat(ctx, instance.ID)
			instance.Status = domain.WorkspaceStatusRunning
			return instance, nil
		}
	}

	template := s.resolveTemplate(ctx, instance.SubjectID)
	// La reactivación preserva el límite vigente en BD (refleja política,
	// plantilla y auto-bursting acumulados); solo recalcula si no hay valor.
	ramLimitMB := instance.MemoryLimitMB
	if ramLimitMB <= 0 {
		ramLimitMB = int64(s.effectivePolicies(ctx, instance.TenantID).RAMLimitMB)
	}
	imageName := domain.OpenVSCodeImage
	if template != nil {
		if template.BaseRamMB >= 256 && int64(template.BaseRamMB) > ramLimitMB {
			ramLimitMB = int64(template.BaseRamMB)
		}
		if strings.TrimSpace(template.DockerImage) != "" {
			imageName = strings.TrimSpace(template.DockerImage)
		}
	}

	containerName := fmt.Sprintf("solv-workspace-%s", instance.ID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", instance.StudentID, instance.SubjectID)
	networkName := "solv-traefik-net"
	baseDomain := getBaseDomain()

	labels := map[string]string{}
	hostPort, portErr := reserveWorkspaceHostPort()
	if portErr != nil {
		log.Printf("[Workspace] WARNING: no se pudo reservar un puerto local para %s: %v", instance.ID, portErr)
	}
	if baseDomain != "" {
		labels["traefik.enable"] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID)] = fmt.Sprintf("Host(`%s.%s`)", instance.ID, baseDomain)
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", instance.ID)] = "3000"
	}
	instance.AccessURL = buildWorkspaceAccessURL(instance.ID, hostPort)
	if updater, ok := s.repo.(workspaceAccessURLUpdater); ok {
		if errURL := updater.UpdateAccessURL(ctx, instance.ID, instance.AccessURL); errURL != nil {
			log.Printf("[Workspace] WARNING: no se pudo persistir access_url de %s: %v", instance.ID, errURL)
		}
	}

	// Reinyección de credenciales del motor si aplica (la base ya fue asegurada y el
	// db_init_script del ejercicio aplicado por ensureExerciseSatelliteDB).
	envVars := []string{
		"DONT_PROMPT_WSL_INSTALL=true",
	}
	envVars = append(envVars, provision.Env...)
	if provision.PostgresDBName != "" {
		pgHost := getPostgresContainerHost()
		dbURL := fmt.Sprintf("postgresql://postgres:postgres@%s:5432/%s?sslmode=disable", pgHost, provision.PostgresDBName)
		envVars = append(envVars,
			fmt.Sprintf("DATABASE_URL=%s", dbURL),
			fmt.Sprintf("PGHOST=%s", pgHost),
			"PGPORT=5432",
			"PGUSER=postgres",
			"PGPASSWORD=postgres",
			fmt.Sprintf("PGDATABASE=%s", provision.PostgresDBName),
		)
	}

	config := domain.WorkspaceContainerConfig{
		Image:               imageName,
		ContainerName:       containerName,
		VolumeName:          volumeName,
		MemoryLimitMB:       ramLimitMB,
		NetworkName:         networkName,
		Labels:              labels,
		Env:                 envVars,
		SetupScript:         sanitizeWorkspaceSetupScript(template),
		HostPort:            hostPort,
		AttachSatellitesNet: provision.UsesManagedContainer,
	}

	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to reactivate workspace container: %w", err)
	}

	_ = s.repo.UpdateContainerID(ctx, instance.ID, containerID)
	_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusRunning)
	_ = s.repo.UpdateMemoryLimit(ctx, instance.ID, ramLimitMB)
	_ = s.repo.RecordHeartbeat(ctx, instance.ID)

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning
	instance.MemoryLimitMB = ramLimitMB
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

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		_ = s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID)
	}

	_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusPending)

	template := s.resolveTemplate(ctx, ws.SubjectID)
	provision, dbErr := s.ensureExerciseSatelliteDB(ctx, ws.StudentID, ws.SubjectID, template, nil)
	if dbErr != nil {
		return nil, dbErr
	}
	return s.reactivateWorkspace(ctx, ws, provision)
}

func (s *WorkspaceService) TerminateWorkspace(ctx context.Context, workspaceID string) error {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("Warning: failed to stop/remove container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed); err != nil {
		return fmt.Errorf("failed to update workspace status to failed: %w", err)
	}

	// Scale-to-Zero: el motor gestionado se apaga si este era su último laboratorio.
	s.ReconcileIdleManagedSatellites(ctx)
	return nil
}

func (s *WorkspaceService) HibernateWorkspace(ctx context.Context, workspaceID string) error {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.PauseContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("Warning: failed to pause container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusHibernated); err != nil {
		return fmt.Errorf("failed to update workspace status to hibernated: %w", err)
	}

	// Scale-to-Zero: el motor gestionado se apaga si este era su último laboratorio.
	s.ReconcileIdleManagedSatellites(ctx)
	return nil
}

func (s *WorkspaceService) GetWorkspaceByID(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	return s.repo.GetByID(ctx, workspaceID)
}

func (s *WorkspaceService) PauseWorkspace(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.PauseContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("[WorkspaceService] Warning: error pausing container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusHibernated); err != nil {
		return nil, fmt.Errorf("failed to update workspace status to hibernated: %w", err)
	}

	// Scale-to-Zero: el motor gestionado se apaga si este era su último laboratorio.
	s.ReconcileIdleManagedSatellites(ctx)

	ws.Status = domain.WorkspaceStatusHibernated
	return ws, nil
}

func (s *WorkspaceService) GetSemgrepAudit(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}
	return ws, nil
}
