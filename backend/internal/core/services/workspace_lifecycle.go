package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

func (s *WorkspaceService) reactivateWorkspace(ctx context.Context, instance *domain.WorkspaceInstance, provision satelliteProvision, linkSession bool) (*domain.WorkspaceInstance, error) {
	// La reanudación nunca deshace el cierre de un examen, ni por un contenedor
	// pausado ni por el reconciliador (ADR-005 §5).
	if instance.FrozenAt != nil || instance.Status == domain.WorkspaceStatusFrozen {
		return nil, domain.ErrWorkspaceFrozen
	}

	// Optimización ADR / Memory Governance: intentar despausar el contenedor congelado (instantáneo <50ms)
	//
	// El adapter no considera error que el contenedor ya no exista, así que la
	// comprobación de existencia va antes: sin contenedor en el daemon no hay
	// nada que despausar y el entorno se reconstruye con una URL viva.
	if instance.ContainerID != nil && *instance.ContainerID != "" && s.containerAlive(ctx, instance.ContainerID) {
		if err := s.docker.UnpauseContainer(ctx, *instance.ContainerID); err == nil {
			_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusRunning)
			_ = s.repo.RecordHeartbeat(ctx, instance.ID)
			instance.Status = domain.WorkspaceStatusRunning
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

	containerName := fmt.Sprintf("solv-workspace-%s", instance.ID)
	volumeName := instance.VolumeName()
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
	exerciseIDStr := ""
	if instance.ExerciseID != nil {
		exerciseIDStr = *instance.ExerciseID
	}
	envVars = append(envVars, s.buildCompanionEnvVars(ctx, instance.TenantID, instance.SubjectID, exerciseIDStr, instance.StudentID)...)
	envVars = append(envVars, provision.Env...)

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
	if linkSession {
		s.linkSessionToAula(ctx, instance)
	}
	return instance, nil
}

func (s *WorkspaceService) RecordHeartbeat(ctx context.Context, workspaceID string) error {
	return s.repo.RecordHeartbeat(ctx, workspaceID)
}

// workspaceOrNotFound lee el entorno por id y traduce la ausencia de fila a
// domain.ErrNotFound. Un laboratorio que ya no existe no es un fallo del
// servidor: el transporte responde 404 en lugar de 500.
func (s *WorkspaceService) workspaceOrNotFound(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	instance, err := s.repo.GetByID(ctx, workspaceID)
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

func (s *WorkspaceService) RestartWorkspace(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.workspaceOrNotFound(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		_ = s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID)
	}

	_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusPending)

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
	return s.reactivateWorkspace(ctx, ws, provision, ws.ExerciseID == nil)
}

func (s *WorkspaceService) TerminateWorkspace(ctx context.Context, workspaceID string) error {
	ws, err := s.workspaceOrNotFound(ctx, workspaceID)
	if err != nil {
		return err
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
	ws, err := s.workspaceOrNotFound(ctx, workspaceID)
	if err != nil {
		return err
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

// ErrWorkspaceSubjectRequired indica que falta la materia de la consulta.
var ErrWorkspaceSubjectRequired = errors.New("subject_id is required")

// WorkspaceForSubject devuelve el entorno del usuario en una materia, o nil si
// todavía no abrió ninguno. Es la lectura previa a abrir: la pantalla puede
// mostrar el entorno vigente sin levantar un contenedor que nadie pidió. El
// usuario se resuelve siempre desde la sesión, así que nunca devuelve el de otro.
func (s *WorkspaceService) WorkspaceForSubject(ctx context.Context, userID, subjectID string) (*domain.WorkspaceInstance, error) {
	userID = strings.TrimSpace(userID)
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return nil, ErrWorkspaceSubjectRequired
	}
	if _, err := uuid.Parse(subjectID); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidResourceID, subjectID)
	}

	instance, err := s.repo.GetByStudentAndSubject(ctx, userID, subjectID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *WorkspaceService) PauseWorkspace(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.workspaceOrNotFound(ctx, workspaceID)
	if err != nil {
		return nil, err
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
	return s.workspaceOrNotFound(ctx, workspaceID)
}
