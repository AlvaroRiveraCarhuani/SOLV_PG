package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"solv-backend/internal/core/domain"
)

func (s *PracticeEnvironmentService) reactivatePracticeEnvironment(ctx context.Context, instance *domain.PracticeEnvironment, provision satelliteProvision, linkSession bool) (*domain.PracticeEnvironment, error) {
	// La reanudación nunca deshace el cierre de un examen, ni por un contenedor
	// pausado ni por el reconciliador (ADR-005 §5).
	if instance.FrozenAt != nil || instance.Status == domain.PracticeEnvironmentStatusFrozen {
		return nil, domain.ErrPracticeEnvironmentFrozen
	}

	// Optimización ADR / Memory Governance: intentar despausar el contenedor congelado (instantáneo <50ms)
	//
	// El adapter no considera error que el contenedor ya no exista, así que la
	// comprobación de existencia va antes: sin contenedor en el daemon no hay
	// nada que despausar y el entorno se reconstruye con una URL viva.
	if instance.ContainerID != nil && *instance.ContainerID != "" && s.containerAlive(ctx, instance.ContainerID) {
		if err := s.docker.UnpauseContainer(ctx, *instance.ContainerID); err == nil {
			_ = s.repo.UpdateStatus(ctx, instance.ID, domain.PracticeEnvironmentStatusRunning)
			_ = s.repo.RecordHeartbeat(ctx, instance.ID)
			instance.Status = domain.PracticeEnvironmentStatusRunning
			if linkSession {
				s.linkSessionToAula(ctx, instance)
			}
			return instance, nil
		}
	}

	// La reactivación preserva la plantilla con la que nació el entorno: una
	// práctica abierta antes de un cambio de plantilla no se reconstruye con la
	// imagen de otra práctica de la materia.
	template := s.resolveTemplateForInstance(ctx, instance)
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

	containerName := fmt.Sprintf("solv-environment-%s", instance.ID)
	volumeName := instance.VolumeName()
	networkName := "solv-traefik-net"
	baseDomain := getBaseDomain()

	labels := map[string]string{}
	hostPort, portErr := reservePracticeEnvironmentHostPort()
	if portErr != nil {
		log.Printf("[PracticeEnvironment] WARNING: no se pudo reservar un puerto local para %s: %v", instance.ID, portErr)
	}
	if baseDomain != "" {
		labels["traefik.enable"] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID)] = fmt.Sprintf("Host(`%s.%s`)", instance.ID, baseDomain)
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", instance.ID)] = "3000"
	}
	instance.AccessURL = buildPracticeEnvironmentAccessURL(instance.ID, hostPort)
	if updater, ok := s.repo.(environmentAccessURLUpdater); ok {
		if errURL := updater.UpdateAccessURL(ctx, instance.ID, instance.AccessURL); errURL != nil {
			log.Printf("[PracticeEnvironment] WARNING: no se pudo persistir access_url de %s: %v", instance.ID, errURL)
		}
	}

	// Reinyección de credenciales del motor si aplica (la base ya fue asegurada y el
	// db_init_script del ejercicio aplicado por ensureExerciseSatelliteDB).
	envVars := []string{
		"DONT_PROMPT_WSL_INSTALL=true",
	}
	exerciseIDStr := ""
	if instance.ExerciseID != nil {
		exerciseIDStr = *instance.ExerciseID
	}
	envVars = append(envVars, s.buildCompanionEnvVars(ctx, instance.TenantID, instance.SubjectID, exerciseIDStr, instance.StudentID)...)
	// Un entorno reanudado también tiene que encontrar el almacén preparado: sin
	// estas variables la herramienta vuelve a su ruta por defecto.
	envVars = append(envVars, s.workspaceDependencyEnvVars(ctx, instance.TenantID, exerciseIDStr)...)
	envVars = append(envVars, provision.Env...)

	config := domain.PracticeEnvironmentContainerConfig{
		Image:               imageName,
		ContainerName:       containerName,
		VolumeName:          volumeName,
		MemoryLimitMB:       ramLimitMB,
		NetworkName:         networkName,
		Labels:              labels,
		Env:                 envVars,
		SetupScript:         sanitizePracticeEnvironmentSetupScript(template),
		HostPort:            hostPort,
		AttachSatellitesNet: provision.UsesManagedContainer,
	}

	containerID, err := s.docker.StartPracticeEnvironmentContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, instance.ID, domain.PracticeEnvironmentStatusFailed)
		return nil, fmt.Errorf("failed to reactivate environment container: %w", err)
	}

	_ = s.repo.UpdateContainerID(ctx, instance.ID, containerID)
	_ = s.repo.UpdateStatus(ctx, instance.ID, domain.PracticeEnvironmentStatusRunning)
	_ = s.repo.UpdateMemoryLimit(ctx, instance.ID, ramLimitMB)
	_ = s.repo.RecordHeartbeat(ctx, instance.ID)

	instance.ContainerID = &containerID
	instance.Status = domain.PracticeEnvironmentStatusRunning
	instance.MemoryLimitMB = ramLimitMB
	if linkSession {
		s.linkSessionToAula(ctx, instance)
	}
	return instance, nil
}

func (s *PracticeEnvironmentService) RecordHeartbeat(ctx context.Context, environmentID string) error {
	return s.repo.RecordHeartbeat(ctx, environmentID)
}

// environmentOrNotFound lee el entorno por id y traduce la ausencia de fila a
// domain.ErrNotFound. Un laboratorio que ya no existe no es un fallo del
// servidor: el transporte responde 404 en lugar de 500.
func (s *PracticeEnvironmentService) environmentOrNotFound(ctx context.Context, environmentID string) (*domain.PracticeEnvironment, error) {
	instance, err := s.repo.GetByID(ctx, environmentID)
	if err != nil {
		if isMissingRow(err) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if instance == nil {
		return nil, domain.ErrNotFound
	}
	return instance, nil
}

func (s *PracticeEnvironmentService) RestartPracticeEnvironment(ctx context.Context, environmentID string) (*domain.PracticeEnvironment, error) {
	ws, err := s.environmentOrNotFound(ctx, environmentID)
	if err != nil {
		return nil, err
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		_ = s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID)
	}

	_ = s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusPending)

	template := s.resolveTemplateForInstance(ctx, ws)
	var exercise *domain.Exercise
	if ws.ExerciseID != nil {
		exercise = s.resolveExercise(ctx, *ws.ExerciseID, ws.TenantID)
	}
	provision, dbErr := s.ensureExerciseSatelliteDB(ctx, ws.StudentID, ws.SubjectID, template, exercise)
	if dbErr != nil {
		return nil, dbErr
	}
	// Un reinicio administrativo no crea presencia: el aula la resuelve el
	// latido del alumno. Solo los entornos de materia se enlazan aquí.
	return s.reactivatePracticeEnvironment(ctx, ws, provision, ws.ExerciseID == nil)
}

func (s *PracticeEnvironmentService) TerminatePracticeEnvironment(ctx context.Context, environmentID string) error {
	ws, err := s.environmentOrNotFound(ctx, environmentID)
	if err != nil {
		return err
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("Warning: failed to stop/remove container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusFailed); err != nil {
		return fmt.Errorf("failed to update environment status to failed: %w", err)
	}

	// Scale-to-Zero: el motor gestionado se apaga si este era su último laboratorio.
	s.ReconcileIdleManagedSatellites(ctx)
	return nil
}

func (s *PracticeEnvironmentService) HibernatePracticeEnvironment(ctx context.Context, environmentID string) error {
	ws, err := s.environmentOrNotFound(ctx, environmentID)
	if err != nil {
		return err
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.PauseContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("Warning: failed to pause container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusHibernated); err != nil {
		return fmt.Errorf("failed to update environment status to hibernated: %w", err)
	}

	// Scale-to-Zero: el motor gestionado se apaga si este era su último laboratorio.
	s.ReconcileIdleManagedSatellites(ctx)
	return nil
}

func (s *PracticeEnvironmentService) GetPracticeEnvironmentByID(ctx context.Context, environmentID string) (*domain.PracticeEnvironment, error) {
	return s.repo.GetByID(ctx, environmentID)
}

func (s *PracticeEnvironmentService) PausePracticeEnvironment(ctx context.Context, environmentID string) (*domain.PracticeEnvironment, error) {
	ws, err := s.environmentOrNotFound(ctx, environmentID)
	if err != nil {
		return nil, err
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.PauseContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("[PracticeEnvironmentService] Warning: error pausing container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusHibernated); err != nil {
		return nil, fmt.Errorf("failed to update environment status to hibernated: %w", err)
	}

	// Scale-to-Zero: el motor gestionado se apaga si este era su último laboratorio.
	s.ReconcileIdleManagedSatellites(ctx)

	ws.Status = domain.PracticeEnvironmentStatusHibernated
	return ws, nil
}

func (s *PracticeEnvironmentService) GetSemgrepAudit(ctx context.Context, environmentID string) (*domain.PracticeEnvironment, error) {
	return s.environmentOrNotFound(ctx, environmentID)
}
