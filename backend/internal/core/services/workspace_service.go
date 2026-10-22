package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

type WorkspaceService struct {
	repo            domain.WorkspaceRepository
	docker          domain.WorkspaceOrchestrator
	hostMonitor     domain.HostMonitor
	subjectRepo     domain.SubjectRepository
	templateRepo    domain.TemplateRepository
	exerciseRepo    domain.ExerciseRepository
	submissionRepo  domain.SubmissionRepository
	db              *sqlx.DB
	policiesService *ServerPoliciesService
}

func NewWorkspaceService(repo domain.WorkspaceRepository, docker domain.WorkspaceOrchestrator, hostMonitor domain.HostMonitor) *WorkspaceService {
	return &WorkspaceService{
		repo:        repo,
		docker:      docker,
		hostMonitor: hostMonitor,
	}
}

// WithProvisioning extiende el servicio para resolver plantillas y bases de datos satélite en tiempo de ejecución
func (s *WorkspaceService) WithProvisioning(subjectRepo domain.SubjectRepository, templateRepo domain.TemplateRepository, db *sqlx.DB) *WorkspaceService {
	s.subjectRepo = subjectRepo
	s.templateRepo = templateRepo
	s.db = db
	return s
}

// WithExerciseSubmission habilita el flujo de IDE Persistente por ejercicio:
// resolución de plantilla desde el ejercicio, db_init_script idempotente y
// entrega del proyecto hacia SpeedGrader. Es fluido para no romper las firmas
// existentes usadas por los tests.
func (s *WorkspaceService) WithExerciseSubmission(exerciseRepo domain.ExerciseRepository, submissionRepo domain.SubmissionRepository) *WorkspaceService {
	s.exerciseRepo = exerciseRepo
	s.submissionRepo = submissionRepo
	return s
}

// SetPoliciesService inyecta las políticas QoS del tenant (submódulo 14.6):
// la RAM de la política es el default por alumno y max_containers el techo
// de concurrencia en la admisión. Sin inyección rigen los defaults del dominio.
func (s *WorkspaceService) SetPoliciesService(policies *ServerPoliciesService) *WorkspaceService {
	s.policiesService = policies
	return s
}

// effectivePolicies resuelve las políticas vigentes del tenant con fallback a
// defaults ante cualquier fallo (fail-open: una política nunca voltea clases;
// la protección dura del host vive en el admission control físico).
func (s *WorkspaceService) effectivePolicies(ctx context.Context, tenantID string) domain.ServerPolicies {
	defaults := domain.DefaultServerPolicies()
	if s.policiesService == nil {
		return defaults
	}
	policies, err := s.policiesService.Get(ctx, tenantID)
	if err != nil || policies == nil {
		return defaults
	}
	return *policies
}

// getBaseDomain devuelve el dominio público configurado, o vacío cuando no hay
// uno. Un valor .local se descarta explícitamente: el acceso local se resuelve
// por 127.0.0.1/localhost y nunca por un dominio inventado.
func getBaseDomain() string {
	d := strings.TrimSpace(os.Getenv("BASE_DOMAIN"))
	if d == "" || strings.HasSuffix(strings.ToLower(d), ".local") {
		return ""
	}
	return d
}

// reserveWorkspaceHostPort pide al kernel un puerto libre en 127.0.0.1 que Docker
// reutiliza en el PortBindings de 3000/tcp.
func reserveWorkspaceHostPort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer func() { _ = listener.Close() }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected listener address %q", listener.Addr().String())
	}
	return strconv.Itoa(addr.Port), nil
}

// buildWorkspaceAccessURL resuelve el acceso al workspace publicado en el host.
func buildWorkspaceAccessURL(workspaceID, hostPort string) string {
	if baseDomain := getBaseDomain(); baseDomain != "" {
		return fmt.Sprintf("https://%s.%s", workspaceID, baseDomain)
	}
	if hostPort != "" {
		return fmt.Sprintf("http://localhost:%s", hostPort)
	}
	return ""
}

// hostTotalMB lee la RAM física total del host con fallback a 8 GB cuando
// gopsutil no responde (mismo contrato que el resto de los workers).
func hostTotalMB() int {
	if vm, err := mem.VirtualMemory(); err == nil && vm.Total > 0 {
		return int(vm.Total / (1024 * 1024))
	}
	return 8192
}

// workspaceAccessURLUpdater es una capacidad opcional del repositorio de
// workspaces: persiste el access_url cuando se recrea el contenedor y cambia el
// puerto publicado. Se resuelve por type-assertion para no ampliar
// domain.WorkspaceRepository ni romper sus mocks.
type workspaceAccessURLUpdater interface {
	UpdateAccessURL(ctx context.Context, id, accessURL string) error
}

func sanitizeIDForDB(id string) string {
	cleaned := strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(cleaned) > 8 {
		return cleaned[:8]
	}
	return cleaned
}

func (s *WorkspaceService) resolveTemplate(ctx context.Context, subjectID string) *domain.Template {
	if s.subjectRepo == nil || s.templateRepo == nil || subjectID == "" {
		return nil
	}
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	subj, err := s.subjectRepo.GetByID(ctx, tenantID, subjectID)
	if err != nil || subj == nil || subj.TemplateID == nil || *subj.TemplateID == "" {
		return nil
	}
	tpl, err := s.templateRepo.GetTemplateByID(ctx, *subj.TemplateID)
	if err != nil || tpl == nil {
		return nil
	}
	return tpl
}

// resolveExercise carga el ejercicio del tenant cuando el workspace se inicia
// desde una actividad concreta.
func (s *WorkspaceService) resolveExercise(ctx context.Context, exerciseID, tenantID string) *domain.Exercise {
	if s.exerciseRepo == nil || strings.TrimSpace(exerciseID) == "" {
		return nil
	}
	exercise, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil || exercise == nil {
		exercise, err = s.exerciseRepo.GetByID(ctx, exerciseID)
		if err != nil || exercise == nil {
			log.Printf("[Workspace] exercise %s not found: %v", exerciseID, err)
			return nil
		}
	}
	return exercise
}

// resolveTemplateForWorkspace prioriza la plantilla del ejercicio sobre la de la
// materia (hallazgo #8): el ejercicio define la imagen real con la que se trabaja.
func (s *WorkspaceService) resolveTemplateForWorkspace(ctx context.Context, subjectID string, exercise *domain.Exercise) *domain.Template {
	if exercise != nil && s.templateRepo != nil {
		if templateID := strings.TrimSpace(exercise.TemplateID); templateID != "" {
			if tpl, err := s.templateRepo.GetTemplateByID(ctx, templateID); err == nil && tpl != nil {
				return tpl
			}
		}
	}
	return s.resolveTemplate(ctx, subjectID)
}

// templateDeclaresPostgres indica si la plantilla declara una base PostgreSQL
// satélite en services_config.services.
func templateDeclaresPostgres(template *domain.Template) bool {
	if template == nil {
		return false
	}
	for _, service := range template.ServicesConfig.Services {
		if service.Category == "database" && service.Engine == "postgres" {
			return true
		}
	}
	return false
}

// ensureExerciseSatelliteDB asegura la base satélite del alumno y aplica el
// db_init_script del ejercicio una sola vez (hard-fail: si el script falla, el
// inicio o la reactivación se aborta). Devuelve el nombre de la BD satélite.
func (s *WorkspaceService) ensureExerciseSatelliteDB(ctx context.Context, studentID, subjectID string, template *domain.Template, exercise *domain.Exercise) (string, error) {
	script := ""
	if exercise != nil && exercise.Config.IDE != nil {
		script = strings.TrimSpace(exercise.Config.IDE.DBInitScript)
	}
	if script == "" && !templateDeclaresPostgres(template) {
		return "", nil
	}
	if s.db == nil {
		if script != "" {
			log.Printf("[Satellite DB Warning] no hay conexión a PostgreSQL configurada: db_init_script del ejercicio %s omitido", exercise.ID)
		}
		return "", nil
	}

	dbName := fmt.Sprintf("solv_std_%s_%s", sanitizeIDForDB(studentID), sanitizeIDForDB(subjectID))
	if err := ensureSatellitePostgresDatabase(ctx, s.db, dbName); err != nil {
		return "", fmt.Errorf("failed to ensure satellite database %s: %w", dbName, err)
	}
	if script != "" {
		if err := applyExerciseDBInitScript(ctx, dbName, exercise.ID, script); err != nil {
			return "", fmt.Errorf("exercise %s db_init_script failed on %s: %w", exercise.ID, dbName, err)
		}
		log.Printf("[Satellite DB] db_init_script del ejercicio %s aplicado/verificado en %s", exercise.ID, dbName)
	}
	return dbName, nil
}

func (s *WorkspaceService) StartWorkspace(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	return s.StartWorkspaceForExercise(ctx, studentID, subjectID, "")
}

// StartWorkspaceForExercise inicia (o reactiva) el IDE persistente resolviendo
// plantilla y db_init_script desde el ejercicio cuando se provee exerciseID.
func (s *WorkspaceService) StartWorkspaceForExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.WorkspaceInstance, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	exercise := s.resolveExercise(ctx, exerciseID, tenantID)
	if strings.TrimSpace(exerciseID) != "" && exercise == nil {
		return nil, fmt.Errorf("exercise %s not found", exerciseID)
	}
	if strings.TrimSpace(subjectID) == "" && exercise != nil && exercise.SubjectID != nil {
		subjectID = strings.TrimSpace(*exercise.SubjectID)
	}

	// 0. Resolución de plantilla: el ejercicio manda sobre la materia.
	template := s.resolveTemplateForWorkspace(ctx, subjectID, exercise)

	policies := s.effectivePolicies(ctx, tenantID)

	// 0.1 Techo de concurrencia del tenant (max_containers): hard gate.
	// Solo cuenta workspaces nuevos; reactivaciones de existentes no consumen cupo.
	if active, err := s.repo.GetActiveWorkspaces(ctx); err != nil {
		log.Printf("[Admission] WARNING: no se pudo contar workspaces activos, admitiendo por fail-open: %v", err)
	} else if len(active) >= policies.MaxContainers {
		return nil, fmt.Errorf("%w: tenant %s tiene %d activos (techo %d)", ErrMaxContainersReached, tenantID, len(active), policies.MaxContainers)
	}

	// 0.2 RAM efectiva: la política es el default por alumno; la plantilla
	// prevalece cuando declara un requisito (BaseRamMB >= 256) y el exceso
	// sobre la política queda registrado para visibilidad del admin.
	ramLimitMB := int64(policies.RAMLimitMB)
	imageName := domain.OpenVSCodeImage // gitpod/openvscode-server:latest
	var templateIDStr *string
	if template != nil {
		templateIDStr = &template.ID
		if template.BaseRamMB >= 256 {
			ramLimitMB = int64(template.BaseRamMB)
			if int(ramLimitMB) > policies.RAMLimitMB {
				log.Printf("[Admission PolicyOverride] plantilla %s requiere %d MB sobre la política del tenant (%d MB): prevalece la plantilla",
					template.ID, ramLimitMB, policies.RAMLimitMB)
			}
		}
		if err := domain.ValidateRamAgainstHost(int(ramLimitMB), domain.CalculateHostMaxAllowedRAM(hostTotalMB())); err != nil {
			return nil, fmt.Errorf("solicitud rechazada: la memoria de la plantilla (%d MB) excede la capacidad estructural del host", ramLimitMB)
		}
		if strings.TrimSpace(template.DockerImage) != "" {
			imageName = strings.TrimSpace(template.DockerImage)
		}
	}

	// 1. Verificación de Admisión del Host con la memoria real calculada
	if !s.hostMonitor.CanAllocateMemory(ramLimitMB) {
		return nil, ErrHostMemoryExhausted
	}

	// 2. Verificación de Idempotencia y Estado Previos
	existing, err := s.repo.GetByStudentAndSubject(ctx, studentID, subjectID)
	if err == nil && existing != nil {
		// El workspace es único por (student_id, subject_id): si ya existía, la
		// base satélite y el db_init_script del ejercicio también deben quedar
		// asegurados antes de devolverlo o reactivarlo. El registro de
		// _solv_exercise_migrations hace la operación idempotente.
		satelliteDBName, dbErr := s.ensureExerciseSatelliteDB(ctx, studentID, subjectID, template, exercise)
		if dbErr != nil {
			return nil, dbErr
		}

		if existing.Status == domain.WorkspaceStatusRunning {
			return existing, nil
		}

		// Verificación de Castigo por OOMKilled (Sistema de 3 Strikes = 5 minutos de bloqueo)
		if existing.OOMStrikeCount >= domain.MaxOOMStrikes && existing.LastOOMKilledAt != nil {
			if time.Since(*existing.LastOOMKilledAt) < domain.OOMCooldownDuration {
				remaining := domain.OOMCooldownDuration - time.Since(*existing.LastOOMKilledAt)
				return nil, fmt.Errorf("%w (%d s remaining)", ErrOOMKilledCooldownPenalty, int(remaining.Seconds()))
			}
			_ = s.repo.ResetOOMStrikes(ctx, existing.ID)
		}

		return s.reactivateWorkspace(ctx, existing, satelliteDBName)
	}

	satelliteDBName, err := s.ensureExerciseSatelliteDB(ctx, studentID, subjectID, template, exercise)
	if err != nil {
		return nil, err
	}

	// 3. Generación de UUID opaco para el workspace_id y construcción de access_url
	workspaceID := uuid.NewString()
	baseDomain := getBaseDomain()
	hostPort, portErr := reserveWorkspaceHostPort()
	if portErr != nil {
		log.Printf("[Workspace] WARNING: no se pudo reservar un puerto local para %s: %v", workspaceID, portErr)
	}
	accessURL := buildWorkspaceAccessURL(workspaceID, hostPort)
	containerName := fmt.Sprintf("solv-workspace-%s", workspaceID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", studentID, subjectID)
	networkName := "solv-traefik-net"

	instance := &domain.WorkspaceInstance{
		ID:              workspaceID,
		TenantID:        tenantID,
		StudentID:       studentID,
		SubjectID:       subjectID,
		TemplateID:      templateIDStr,
		Status:          domain.WorkspaceStatusPending,
		AccessURL:       accessURL,
		MemoryLimitMB:   ramLimitMB,
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

	// 7. Dynamic Labels de Traefik v3 (Puerto 3000 para OpenVSCode Server). Solo se
	// publican cuando existe un dominio público configurado; el acceso local no
	// depende de Traefik ni de .local.
	labels := map[string]string{}
	if baseDomain != "" {
		labels["traefik.enable"] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", workspaceID)] = fmt.Sprintf("Host(`%s.%s`)", workspaceID, baseDomain)
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", workspaceID)] = "3000"
	}

	envVars := []string{
		"DONT_PROMPT_WSL_INSTALL=true",
	}

	// 8. Inyección de credenciales de la base satélite ya asegurada (ADR-030).
	// El helper ensureExerciseSatelliteDB ya creó la BD y aplicó el
	// db_init_script del ejercicio en modo hard-fail.
	if satelliteDBName != "" && s.db != nil {
		pgHost := getPostgresContainerHost()
		dbURL := fmt.Sprintf("postgresql://postgres:postgres@%s:5432/%s?sslmode=disable", pgHost, satelliteDBName)
		envVars = append(envVars,
			fmt.Sprintf("DATABASE_URL=%s", dbURL),
			fmt.Sprintf("PGHOST=%s", pgHost),
			"PGPORT=5432",
			"PGUSER=postgres",
			"PGPASSWORD=postgres",
			fmt.Sprintf("PGDATABASE=%s", satelliteDBName),
		)
		log.Printf("[Satellite DB] Injected DATABASE_URL for workspace %s (db: %s)", workspaceID, satelliteDBName)
	}

	// 9. Configuración del contenedor OpenVSCode Server
	config := domain.WorkspaceContainerConfig{
		Image:         imageName,
		ContainerName: containerName,
		VolumeName:    volumeName,
		MemoryLimitMB: ramLimitMB,
		NetworkName:   networkName,
		Labels:        labels,
		Env:           envVars,
		HostPort:      hostPort,
	}

	// 10. Instanciación e inicio del contenedor OpenVSCode Server
	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to start openvscode-server container: %w", err)
	}

	// 11. Actualización del registro en PostgreSQL a estado 'running'
	if err := s.repo.UpdateContainerID(ctx, workspaceID, containerID); err != nil {
		return nil, fmt.Errorf("failed to update container_id: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusRunning); err != nil {
		return nil, fmt.Errorf("failed to update workspace status: %w", err)
	}

	// 12. Ejecución no bloqueante del setup_script si la plantilla lo define
	if template != nil && strings.TrimSpace(template.SetupScript) != "" {
		setupScript := template.SetupScript
		setupCmd := []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("cd /home/workspace && (%s) > /home/workspace/.solv_setup.log 2>&1", setupScript),
		}
		go func(cid, wid string) {
			time.Sleep(1 * time.Second)
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if errExec := s.docker.ExecuteCommandInBackground(bgCtx, cid, "/home/workspace", setupCmd); errExec != nil {
				log.Printf("[Setup Script Warning] Failed to execute setup_script in workspace %s: %v", wid, errExec)
			} else {
				log.Printf("[Setup Script] Successfully triggered setup_script for workspace %s", wid)
			}
		}(containerID, workspaceID)
	}

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning
	return instance, nil
}
