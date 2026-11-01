package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"solv-backend/internal/core/domain"
)

// SetWorkspaceMount reconstruye el contenedor del alumno con el volumen del lado
// indicado: :rw para trabajar y :ro cuando el examen se cerró (ADR-005 §5).
//
// Docker no deja cambiar el modo de un montaje en caliente, así que el
// contenedor se detiene, se quita y se vuelve a crear con la misma
// configuración que usa la reanudación. La hora del congelamiento no se toca
// acá: la escribe el servicio del examen en la misma operación.
func (s *WorkspaceService) SetWorkspaceMount(ctx context.Context, instance *domain.WorkspaceInstance, readOnly bool) error {
	if instance == nil {
		return fmt.Errorf("workspace not found")
	}
	if instance.ContainerID != nil && strings.TrimSpace(*instance.ContainerID) != "" {
		if err := s.docker.StopAndRemoveContainer(ctx, *instance.ContainerID); err != nil {
			// Un contenedor ya ausente no impide volver a montar: el volumen es
			// lo que importa y sigue ahí.
			log.Printf("[Workspace Freeze] no se pudo detener el contenedor %s: %v", *instance.ContainerID, err)
		}
	}

	// La reconstrucción conserva la plantilla con la que nació el laboratorio:
	// cambiar de imagen o de servicios en medio de un examen no es parte del
	// cierre.
	template := s.resolveTemplateForInstance(ctx, instance)
	provision, err := s.ensureExerciseSatelliteDB(ctx, instance.StudentID, instance.SubjectID, template, nil)
	if err != nil {
		return err
	}

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

	hostPort, portErr := reserveWorkspaceHostPort()
	if portErr != nil {
		log.Printf("[Workspace Freeze] WARNING: no se pudo reservar un puerto local para %s: %v", instance.ID, portErr)
	}
	instance.AccessURL = buildWorkspaceAccessURL(instance.ID, hostPort)
	if updater, ok := s.repo.(workspaceAccessURLUpdater); ok {
		if errURL := updater.UpdateAccessURL(ctx, instance.ID, instance.AccessURL); errURL != nil {
			log.Printf("[Workspace Freeze] WARNING: no se pudo persistir access_url de %s: %v", instance.ID, errURL)
		}
	}

	labels := map[string]string{}
	if baseDomain := getBaseDomain(); baseDomain != "" {
		labels["traefik.enable"] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID)] = fmt.Sprintf("Host(`%s.%s`)", instance.ID, baseDomain)
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", instance.ID)] = "3000"
	}

	envVars := []string{"DONT_PROMPT_WSL_INSTALL=true"}
	envVars = append(envVars, provision.Env...)

	config := domain.WorkspaceContainerConfig{
		Image:               imageName,
		ContainerName:       fmt.Sprintf("solv-workspace-%s", instance.ID),
		VolumeName:          fmt.Sprintf("solv_workspace_%s_%s", instance.StudentID, instance.SubjectID),
		MemoryLimitMB:       ramLimitMB,
		NetworkName:         "solv-traefik-net",
		Labels:              labels,
		Env:                 envVars,
		SetupScript:         sanitizeWorkspaceSetupScript(template),
		HostPort:            hostPort,
		AttachSatellitesNet: provision.UsesManagedContainer,
		ReadOnly:            readOnly,
	}

	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to rebuild workspace container with readOnly=%t: %w", readOnly, err)
	}

	if err := s.repo.UpdateContainerID(ctx, instance.ID, containerID); err != nil {
		return fmt.Errorf("failed to persist container id after rebuilding workspace %s: %w", instance.ID, err)
	}
	instance.ContainerID = &containerID
	instance.MemoryLimitMB = ramLimitMB
	return nil
}
