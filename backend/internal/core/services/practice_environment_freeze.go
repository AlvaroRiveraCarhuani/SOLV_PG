package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"solv-backend/internal/core/domain"
)

// companionEnvReader lee las variables SOLV_* del contenedor vigente. Es
// opcional: un orquestador sin esta capacidad reconstruye el entorno sin ellas,
// como antes.
type companionEnvReader interface {
	ReadCompanionEnv(ctx context.Context, containerID string) ([]string, error)
}

// preserveCompanionEnv copia la configuración del Companion del contenedor que
// se va a reemplazar. La pausa corre con el contexto del docente: reconstruir
// con buildCompanionEnvVars inyectaría su rol y su token en el entorno del
// alumno, así que se conserva lo que ya tenía.
func preserveCompanionEnv(ctx context.Context, docker domain.PracticeEnvironmentOrchestrator, containerID string) []string {
	reader, ok := docker.(companionEnvReader)
	if !ok || strings.TrimSpace(containerID) == "" {
		return nil
	}
	env, err := reader.ReadCompanionEnv(ctx, containerID)
	if err != nil {
		log.Printf("[PracticeEnvironment Freeze] no se pudo leer la configuración del Companion de %s: %v", containerID, err)
		return nil
	}
	return env
}

// lockEnvironmentMount serializa la reconstrucción de un laboratorio dentro del
// proceso. El mapa se crea por demanda, igual que el de la apertura del
// workspace, y la clave es el id del entorno.
func (s *PracticeEnvironmentService) lockEnvironmentMount(environmentID string) func() {
	if s == nil {
		return func() {}
	}
	key := strings.TrimSpace(environmentID)
	s.environmentMountMu.Lock()
	if s.environmentMountLocks == nil {
		s.environmentMountLocks = make(map[string]*sync.Mutex)
	}
	lock, ok := s.environmentMountLocks[key]
	if !ok {
		lock = &sync.Mutex{}
		s.environmentMountLocks[key] = lock
	}
	s.environmentMountMu.Unlock()

	lock.Lock()
	return lock.Unlock
}

// SetPracticeEnvironmentMount reconstruye el contenedor del alumno con el volumen del lado
// indicado: :rw para trabajar y :ro cuando el examen se cerró (ADR-005 §5).
//
// Docker no deja cambiar el modo de un montaje en caliente, así que el
// contenedor se detiene, se quita y se vuelve a crear con la misma
// configuración que usa la reanudación. La hora del congelamiento no se toca
// acá: la escribe el servicio del examen en la misma operación.
func (s *PracticeEnvironmentService) SetPracticeEnvironmentMount(ctx context.Context, instance *domain.PracticeEnvironment, readOnly bool) error {
	if instance == nil {
		return domain.ErrNotFound
	}

	// Dos clics de pausa seguidos llegan con el mismo laboratorio: el segundo
	// espera a que termine la reconstrucción del primero y retoma el contenedor
	// vigente. Sin esto, ambos intentaban crear el mismo nombre y uno fallaba.
	unlock := s.lockEnvironmentMount(instance.ID)
	defer unlock()
	if current, err := s.repo.GetByID(ctx, instance.ID); err == nil && current != nil {
		instance = current
	}
	companionEnv := []string{}
	if instance.ContainerID != nil && strings.TrimSpace(*instance.ContainerID) != "" {
		companionEnv = preserveCompanionEnv(ctx, s.docker, *instance.ContainerID)
		if err := s.docker.StopAndRemoveContainer(ctx, *instance.ContainerID); err != nil {
			// Un contenedor ya ausente no impide volver a montar: el volumen es
			// lo que importa y sigue ahí.
			log.Printf("[PracticeEnvironment Freeze] no se pudo detener el contenedor %s: %v", *instance.ContainerID, err)
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

	hostPort, portErr := reservePracticeEnvironmentHostPort()
	if portErr != nil {
		log.Printf("[PracticeEnvironment Freeze] WARNING: no se pudo reservar un puerto local para %s: %v", instance.ID, portErr)
	}
	instance.AccessURL = buildPracticeEnvironmentAccessURL(instance.ID, hostPort)
	if updater, ok := s.repo.(environmentAccessURLUpdater); ok {
		if errURL := updater.UpdateAccessURL(ctx, instance.ID, instance.AccessURL); errURL != nil {
			log.Printf("[PracticeEnvironment Freeze] WARNING: no se pudo persistir access_url de %s: %v", instance.ID, errURL)
		}
	}

	labels := map[string]string{}
	if baseDomain := getBaseDomain(); baseDomain != "" {
		labels["traefik.enable"] = "true"
		// El contenedor también está en la red de satélites: Traefik tiene que
		// saber en cuál mirar o la petición termina en 504.
		labels["traefik.docker.network"] = "solv-traefik-net"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID)] = fmt.Sprintf("Host(`%s.%s`)", instance.ID, baseDomain)
		// El entorno congelado se sirve igual por 443: websecure no declara TLS
		// por defecto, así que el router lo pide explícito con el comodín.
		labels[fmt.Sprintf("traefik.http.routers.%s.tls", instance.ID)] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", instance.ID)] = "letsencrypt"
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", instance.ID)] = "3000"
	}

	envVars := []string{"DONT_PROMPT_WSL_INSTALL=true"}
	// La pausa reconstruye el contenedor: sin estas variables el Companion del
	// alumno despertaba sin URL, sin token y sin aula.
	envVars = append(envVars, companionEnv...)
	// El contenedor congelado monta el volumen en solo lectura, pero el almacén
	// sigue en la misma ruta: las variables tienen que viajar igual.
	envVars = append(envVars, s.workspaceDependencyEnvVars(ctx, instance.TenantID, valueOrEmpty(instance.ExerciseID))...)
	envVars = append(envVars, provision.Env...)

	runtimePolicies := s.effectivePolicies(ctx, instance.TenantID)
	config := domain.PracticeEnvironmentContainerConfig{
		Image:               imageName,
		ContainerName:       fmt.Sprintf("solv-environment-%s", instance.ID),
		VolumeName:          instance.VolumeName(),
		MemoryLimitMB:       ramLimitMB,
		CPUNanos:            domain.CPUQuotaNanos(runtimePolicies.EffectivePracticeCPUMillis()),
		PidsLimit:           runtimePolicies.EffectivePidsLimit(),
		NetworkName:         "solv-traefik-net",
		Labels:              labels,
		Env:                 envVars,
		SetupScript:         sanitizePracticeEnvironmentSetupScript(template),
		HostPort:            hostPort,
		AttachSatellitesNet: provision.UsesManagedContainer,
		ReadOnly:            readOnly,
		ExamNoEgress:        instance.EnvironmentProfile.IsExam() && s.examNetworkPolicyFor(ctx, instance.TenantID, instance.ExerciseID) == domain.ExamNetworkNone,
	}

	containerID, err := s.docker.StartPracticeEnvironmentContainer(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to rebuild environment container with readOnly=%t: %w", readOnly, err)
	}

	if err := s.repo.UpdateContainerID(ctx, instance.ID, containerID); err != nil {
		return fmt.Errorf("failed to persist container id after rebuilding environment %s: %w", instance.ID, err)
	}
	instance.ContainerID = &containerID
	instance.MemoryLimitMB = ramLimitMB
	instance.CPUMillis = runtimePolicies.EffectivePracticeCPUMillis()
	return nil
}
