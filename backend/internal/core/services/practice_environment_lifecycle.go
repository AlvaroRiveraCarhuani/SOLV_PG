package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"solv-backend/internal/core/domain"
)

func (s *PracticeEnvironmentService) reactivatePracticeEnvironment(ctx context.Context, instance *domain.PracticeEnvironment, provision satelliteProvision, linkSession bool) (*domain.PracticeEnvironment, error) {
	// La reanudación nunca deshace el cierre de un examen, ni por un contenedor
	// pausado ni por el reconciliador (ADR-005 §5).
	if instance.FrozenAt != nil || instance.Status == domain.PracticeEnvironmentStatusFrozen {
		return nil, domain.ErrPracticeEnvironmentFrozen
	}

	// La pausa del aula se lee antes de decidir cómo despertar: un entorno que
	// vuelve con la clase pausada se reconstruye con el volumen en solo lectura
	// y no se despausa en escritura.
	exerciseIDStr := ""
	if instance.ExerciseID != nil {
		exerciseIDStr = *instance.ExerciseID
	}
	aulaPaused := s.aulaPausedForStart(ctx, linkSession, instance.TenantID, instance.StudentID, instance.SubjectID, exerciseIDStr)

	// Optimización ADR / Memory Governance: intentar despausar el contenedor congelado (instantáneo <50ms)
	//
	// El adapter no considera error que el contenedor ya no exista, así que la
	// comprobación de existencia va antes: sin contenedor en el daemon no hay
	// nada que despausar y el entorno se reconstruye con una URL viva. Con el
	// aula pausada se salta el atajo: despausar devolvería la escritura.
	if !aulaPaused && instance.ContainerID != nil && *instance.ContainerID != "" && s.containerAlive(ctx, instance.ContainerID) {
		if err := s.docker.UnpauseContainer(ctx, *instance.ContainerID); err == nil {
			// El contenedor ya estaba en marcha y conserva sus puertos: la URL de la
			// aplicación se recupera del propio contenedor.
			s.attachPracticeAppURL(ctx, instance)
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
	if template != nil && strings.TrimSpace(template.ID) != "" {
		if instance.TemplateID == nil || *instance.TemplateID != template.ID {
			if updater, ok := s.repo.(environmentTemplateUpdater); ok {
				if err := updater.UpdateTemplateID(ctx, instance.ID, template.ID); err != nil {
					log.Printf("[PracticeEnvironment] WARNING: no se pudo persistir template_id de %s: %v", instance.ID, err)
				} else {
					templateID := template.ID
					instance.TemplateID = &templateID
				}
			}
		}
	}

	// La cuota al despertar se decide otra vez: el préstamo no sobrevive a la
	// hibernación. La base la fijan la política y la plantilla; el hueco del
	// nodo decide cuánto se presta en este momento (ADR-082).
	baseRAMMB := int64(s.effectivePolicies(ctx, instance.TenantID).RAMLimitMB)
	if template != nil && template.BaseRamMB >= 256 && int64(template.BaseRamMB) > baseRAMMB {
		baseRAMMB = int64(template.BaseRamMB)
	}
	ramLimitMB := baseRAMMB
	// El perfil declarado se conserva aunque el préstamo conceda otra cosa.
	declaredRAMMB := baseRAMMB
	if s.policiesService == nil && instance.MemoryLimitMB > 0 {
		// Sin políticas inyectadas (dobles de test, ciclos anteriores) se
		// conserva el límite del registro: no hay presupuesto que recalcular.
		ramLimitMB = instance.MemoryLimitMB
	} else {
		policies := s.effectivePolicies(ctx, instance.TenantID)
		ceilingPct := policies.HostSafetyCeilingPct
		if ceilingPct <= 0 {
			ceilingPct = 85
		}
		if s.repo != nil {
			if active, activeErr := s.repo.GetActivePracticeEnvironments(ctx); activeErr == nil {
				ramLimitMB = s.loanQuotaForReactivation(ctx, instance, baseRAMMB, active, ceilingPct)
			}
		}
	}

	imageName := domain.OpenVSCodeImage
	if template != nil && strings.TrimSpace(template.DockerImage) != "" {
		imageName = strings.TrimSpace(template.DockerImage)
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
		// Sin esta etiqueta Traefik puede tomar la IP de la red de satélites, que
		// no tiene ruta hacia el editor, y la petición muere en 504.
		labels["traefik.docker.network"] = networkName
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID)] = fmt.Sprintf("Host(`%s.%s`)", instance.ID, baseDomain)
		// websecure no trae TLS por defecto: el router del editor necesita
		// declararlo para servirse por 443 con el certificado comodín.
		labels[fmt.Sprintf("traefik.http.routers.%s.tls", instance.ID)] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", instance.ID)] = "letsencrypt"
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
	envVars = append(envVars, s.buildCompanionEnvVars(ctx, instance.TenantID, instance.SubjectID, exerciseIDStr, instance.StudentID)...)
	// Un entorno reanudado también tiene que encontrar el almacén preparado: sin
	// estas variables la herramienta vuelve a su ruta por defecto.
	envVars = append(envVars, s.workspaceDependencyEnvVars(ctx, instance.TenantID, exerciseIDStr)...)
	envVars = append(envVars, provision.Env...)

	// La aplicación del alumno se vuelve a publicar al despertar: el puerto local
	// lo reserva el nodo y no sobrevive a la recreación del contenedor.
	var appExercise *domain.Exercise
	if exerciseIDStr != "" {
		appExercise = s.resolveExercise(ctx, exerciseIDStr, instance.TenantID)
	}

	runtimePolicies := s.effectivePolicies(ctx, instance.TenantID)
	config := domain.PracticeEnvironmentContainerConfig{
		Image:               imageName,
		ContainerName:       containerName,
		VolumeName:          volumeName,
		MemoryLimitMB:       ramLimitMB,
		CPUNanos:            domain.CPUQuotaNanos(runtimePolicies.EffectivePracticeCPUMillis()),
		PidsLimit:           runtimePolicies.EffectivePidsLimit(),
		NetworkName:         networkName,
		Labels:              labels,
		Env:                 envVars,
		SetupScript:         sanitizePracticeEnvironmentSetupScript(template),
		HostPort:            hostPort,
		AttachSatellitesNet: provision.UsesManagedContainer,
		ExamNoEgress:        instance.EnvironmentProfile.IsExam() && s.examNetworkPolicyFor(ctx, instance.TenantID, instance.ExerciseID) == domain.ExamNetworkNone,
		// Si la clase está pausada, el entorno despierta en solo lectura.
		ReadOnly:          aulaPaused,
		MountDockerSocket: dockerSocketMountForTemplate(template),
	}

	if appPort := practiceAppPort(appExercise); appPort > 0 && !config.ExamNoEgress {
		if appHostPort, portErr := reservePracticeEnvironmentHostPort(); portErr == nil {
			config.AppPort = appPort
			config.AppHostPort = appHostPort
			instance.AppURL = buildPracticeAppAccessURL(instance.ID, appHostPort)
			if baseDomain != "" {
				router := "app-" + instance.ID
				// Dos routers sobre el mismo contenedor: se declara a qué servicio
				// pertenece cada uno, porque Traefik no los vincula solo.
				labels[fmt.Sprintf("traefik.http.routers.%s.service", instance.ID)] = instance.ID
				labels[fmt.Sprintf("traefik.http.routers.%s.service", router)] = router
				labels[fmt.Sprintf("traefik.http.routers.%s.rule", router)] = fmt.Sprintf("Host(`%s-app.%s`)", instance.ID, baseDomain)
				labels[fmt.Sprintf("traefik.http.routers.%s.tls", router)] = "true"
				labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", router)] = "letsencrypt"
				labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", router)] = strconv.Itoa(appPort)
				// El servidor de desarrollo del proyecto rechaza hosts que no conoce;
				// sin el host del cliente el servicio queda como `ip:puerto`.
				labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.passhostheader", router)] = "false"
			}
		}
	}

	containerID, err := s.docker.StartPracticeEnvironmentContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, instance.ID, domain.PracticeEnvironmentStatusFailed)
		return nil, fmt.Errorf("failed to reactivate environment container: %w", err)
	}

	_ = s.repo.UpdateContainerID(ctx, instance.ID, containerID)
	_ = s.repo.UpdateStatus(ctx, instance.ID, domain.PracticeEnvironmentStatusRunning)
	// La memoria concedida, el perfil declarado y la CPU viajan juntos: la ficha
	// necesita los tres para explicar la cuota real con la que despertó.
	if recorder, ok := s.repo.(resourceQuotaRecorder); ok {
		_ = recorder.RecordResourceQuota(ctx, instance.ID, ramLimitMB, declaredRAMMB, runtimePolicies.EffectivePracticeCPUMillis())
	} else {
		_ = s.repo.UpdateMemoryLimit(ctx, instance.ID, ramLimitMB)
	}
	_ = s.repo.RecordHeartbeat(ctx, instance.ID)

	instance.ContainerID = &containerID
	instance.Status = domain.PracticeEnvironmentStatusRunning
	instance.MemoryLimitMB = ramLimitMB
	instance.MemoryProfileMB = declaredRAMMB
	instance.CPUMillis = runtimePolicies.EffectivePracticeCPUMillis()
	if linkSession {
		s.linkSessionToAula(ctx, instance)
	}
	if aulaPaused {
		s.freezeEnvironmentForPausedAula(ctx, instance)
	}
	// Un entorno que vuelve de la hibernación tarda lo mismo que uno nuevo en
	// servir el editor: se espera antes de entregar la URL para no mandar la
	// pestaña a un 504 del proxy.
	if err := waitForEditorReady(ctx, instance.AccessURL); err != nil {
		return nil, err
	}
	return instance, nil
}

// examNetworkPolicyFor devuelve la política que declaró el ejercicio. Si no se
// puede leer (ejercicio borrado, repositorio ausente), conserva la salida: es
// el comportamiento previo y no corta la red de un examen en curso por un
// error de lectura. La salida cerrada se aplica cuando el ejercicio la declara.
func (s *PracticeEnvironmentService) examNetworkPolicyFor(ctx context.Context, tenantID string, exerciseID *string) string {
	if exerciseID == nil || strings.TrimSpace(*exerciseID) == "" {
		return domain.ExamNetworkEgress
	}
	exercise := s.resolveExercise(ctx, *exerciseID, tenantID)
	if exercise == nil {
		return domain.ExamNetworkEgress
	}
	return domain.NormalizeExamNetwork(exercise.ExamNetwork)
}

// resourceQuotaRecorder es la capacidad opcional del repositorio de guardar la
// cuota concedida, el perfil declarado y la CPU en una sola escritura. Sin ella
// se conserva el camino anterior, que solo actualiza la memoria.
type resourceQuotaRecorder interface {
	RecordResourceQuota(ctx context.Context, id string, memoryMB, profileMB, cpuMillis int64) error
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

// ErrExamPauseForbidden impide apagar a mano un entorno de examen. El apagado
// por inactividad ya está exento por perfil; esto cierra la otra puerta, la del
// docente o el administrador que pausa a mitad del examen y deja al alumno sin
// su espacio.
var ErrExamPauseForbidden = errors.New("un entorno de examen no se puede pausar a mano")

func (s *PracticeEnvironmentService) HibernatePracticeEnvironment(ctx context.Context, environmentID string) error {
	ws, err := s.environmentOrNotFound(ctx, environmentID)
	if err != nil {
		return err
	}
	if ws.EnvironmentProfile.IsExam() {
		return ErrExamPauseForbidden
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
	environment, err := s.repo.GetByID(ctx, environmentID)
	if err != nil || environment == nil {
		return environment, err
	}
	s.annotateDiskLimit(ctx, environment)
	return environment, nil
}

func (s *PracticeEnvironmentService) PausePracticeEnvironment(ctx context.Context, environmentID string) (*domain.PracticeEnvironment, error) {
	ws, err := s.environmentOrNotFound(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	if ws.EnvironmentProfile.IsExam() {
		return nil, ErrExamPauseForbidden
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
