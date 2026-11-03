package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

// ErrPracticeIDENotProject indica que se intentó abrir el IDE de una práctica
// que no usa entorno de proyecto persistente.
var ErrPracticeIDENotProject = errors.New("la práctica no es de IDE persistente")

// workspaceExerciseRepository es una capacidad opcional del repositorio:
// resuelve el entorno de una práctica sin ensanchar la interfaz base que usan
// los dobles de test del ciclo de vida por materia.
type workspaceExerciseRepository interface {
	GetByStudentSubjectExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.WorkspaceInstance, error)
}

type WorkspaceService struct {
	repo            domain.WorkspaceRepository
	docker          domain.WorkspaceOrchestrator
	hostMonitor     domain.HostMonitor
	subjectRepo     domain.SubjectRepository
	templateRepo    domain.TemplateRepository
	exerciseRepo    domain.ExerciseRepository
	submissionRepo  domain.SubmissionRepository
	policiesService *ServerPoliciesService
	// sessionLinker deja constancia de en qué aula nació el entorno. Es opcional:
	// sin él el ciclo de vida del contenedor funciona igual.
	sessionLinker domain.ClassSessionWorkspaceLinker
	// examFreezer cierra el volumen de un examen cuando el alumno entrega antes.
	// Es opcional: sin él, la entrega se registra y el cierre llega con el aula.
	examFreezer ExamWorkspaceFreezer

	// satelliteUseMu protege satelliteLastUse: el último momento en que un
	// laboratorio pidió un motor. Evita que el scale-to-zero apague el satélite en
	// el hueco entre el aprovisionamiento y el registro del laboratorio como activo.
	satelliteUseMu   sync.Mutex
	satelliteLastUse map[string]time.Time
}

func NewWorkspaceService(repo domain.WorkspaceRepository, docker domain.WorkspaceOrchestrator, hostMonitor domain.HostMonitor) *WorkspaceService {
	return &WorkspaceService{
		repo:             repo,
		docker:           docker,
		hostMonitor:      hostMonitor,
		satelliteLastUse: make(map[string]time.Time),
	}
}

// WithProvisioning extiende el servicio para resolver plantillas y bases de datos satélite en tiempo de ejecución
func (s *WorkspaceService) WithProvisioning(subjectRepo domain.SubjectRepository, templateRepo domain.TemplateRepository) *WorkspaceService {
	s.subjectRepo = subjectRepo
	s.templateRepo = templateRepo
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

// SetExamSubmissionFreezer habilita el cierre anticipado por alumno: al entregar
// un examen, solo su volumen pasa a solo lectura (ADR-005 §5.2).
func (s *WorkspaceService) SetExamSubmissionFreezer(freezer ExamWorkspaceFreezer) *WorkspaceService {
	s.examFreezer = freezer
	return s
}

// SetPoliciesService inyecta las políticas QoS del tenant (submódulo 14.6):
// la RAM de la política es el default por alumno y max_containers el techo
// de concurrencia en la admisión. Sin inyección rigen los defaults del dominio.
func (s *WorkspaceService) SetPoliciesService(policies *ServerPoliciesService) *WorkspaceService {
	s.policiesService = policies
	return s
}

// WithClassSessions enlaza el ciclo de vida del entorno con el aula abierta de
// la materia, si la hay. El enlace es best-effort: un fallo al registrar la
// sesión nunca impide que el alumno trabaje.
func (s *WorkspaceService) WithClassSessions(linker domain.ClassSessionWorkspaceLinker) *WorkspaceService {
	s.sessionLinker = linker
	return s
}

// linkSessionToAula registra el entorno en el aula abierta de la materia y marca
// la presencia del alumno. Sin aula abierta no hace nada: el trabajo fuera de
// clase no pertenece a ninguna sesión.
func (s *WorkspaceService) linkSessionToAula(ctx context.Context, instance *domain.WorkspaceInstance) {
	if s.sessionLinker == nil || instance == nil {
		return
	}
	if err := s.sessionLinker.AttachWorkspaceToOpenSession(ctx, instance.TenantID, instance.StudentID, instance.SubjectID, instance.ID); err != nil {
		log.Printf("[Workspace] WARNING: no se pudo enlazar el entorno %s con el aula de la materia %s: %v", instance.ID, instance.SubjectID, err)
	}
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
		return domain.Uint64ToInt(vm.Total / (1024 * 1024))
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
// desde una actividad concreta. Que la práctica no exista es una respuesta
// normal (el llamador devuelve 404), no un fallo del entorno: solo se registra
// cuando la lectura falla de verdad y el servicio no puede distinguir el caso.
func (s *WorkspaceService) resolveExercise(ctx context.Context, exerciseID, tenantID string) *domain.Exercise {
	if s.exerciseRepo == nil || strings.TrimSpace(exerciseID) == "" {
		return nil
	}
	exercise, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err == nil && exercise != nil {
		return exercise
	}

	fallback, fallbackErr := s.exerciseRepo.GetByID(ctx, exerciseID)
	if fallback != nil {
		return fallback
	}

	switch {
	case !isMissingRow(err):
		log.Printf("[Workspace] no se pudo leer el ejercicio %s: %v", exerciseID, err)
	case !isMissingRow(fallbackErr):
		log.Printf("[Workspace] no se pudo leer el ejercicio %s: %v", exerciseID, fallbackErr)
	}
	return nil
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

// resolveTemplateForInstance prefiere la plantilla con la que nació el
// laboratorio. Reconstruir el montaje (congelar o descongelar) tiene que usar la
// misma imagen y los mismos servicios, aunque la materia haya cambiado de
// plantilla después.
func (s *WorkspaceService) resolveTemplateForInstance(ctx context.Context, instance *domain.WorkspaceInstance) *domain.Template {
	if instance != nil && s.templateRepo != nil && instance.TemplateID != nil {
		if templateID := strings.TrimSpace(*instance.TemplateID); templateID != "" {
			if tpl, err := s.templateRepo.GetTemplateByID(ctx, templateID); err == nil && tpl != nil {
				return tpl
			}
		}
	}
	if instance == nil {
		return nil
	}
	return s.resolveTemplate(ctx, instance.SubjectID)
}

// ensureExerciseSatelliteDB asegura todos los servicios que declara la plantilla y
// devuelve el entorno combinado del laboratorio. Cada motor gestionado, incluido
// PostgreSQL de aula, se levanta como satélite del nodo con credenciales por alumno
// (ADR-051); sqlite y 'none' no aprovisionan nada. El db_init_script del ejercicio se
// aplica dentro de la base del alumno (hard-fail: si falla, el inicio o la reactivación
// se aborta).
func (s *WorkspaceService) ensureExerciseSatelliteDB(ctx context.Context, studentID, subjectID string, template *domain.Template, exercise *domain.Exercise) (satelliteProvision, error) {
	bindings := templateServiceBindings(template)

	// Compatibilidad: un ejercicio puede declarar su db_init_script sin que la plantilla
	// enumere servicios. Ese caso conserva el PostgreSQL de aula, ahora como satélite
	// dedicado y no como la base de la plataforma (ADR-051).
	if len(bindings) == 0 {
		if exerciseDBInitScript(exercise) == "" {
			return satelliteProvision{}, nil
		}
		return s.provisionManagedSatellite(ctx, "postgres", studentID, subjectID, exercise)
	}

	merged := satelliteProvision{}
	for _, binding := range bindings {
		engine := domain.CanonicalServiceEngine(binding.Engine)
		if !isManagedSatelliteEngine(engine) {
			continue
		}
		provision, err := s.provisionManagedSatellite(ctx, engine, studentID, subjectID, exercise)
		if err != nil {
			return satelliteProvision{}, err
		}
		merged.Env = append(merged.Env, provision.Env...)
		merged.UsesManagedContainer = merged.UsesManagedContainer || provision.UsesManagedContainer
	}
	return merged, nil
}

func (s *WorkspaceService) StartWorkspace(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	return s.StartWorkspaceForExercise(ctx, studentID, subjectID, "")
}

// sanitizeWorkspaceSetupScript devuelve el script de arranque de la plantilla
// sin espacios sobrantes; una plantilla sin script no arranca con "".
func sanitizeWorkspaceSetupScript(template *domain.Template) string {
	if template == nil {
		return ""
	}
	return strings.TrimSpace(template.SetupScript)
}

// StartWorkspaceForExercise abre el entorno del alumno acotado a la práctica
// cuando llega el ejercicio. Abrir la práctica 2 no debe devolver el entorno de
// la materia (que mezcla el código de todas las prácticas) ni reutilizar su
// volumen: el entorno y el volumen se separan por (alumno, materia, práctica),
// igual que el del docente (ADR-059). Sin práctica se conserva el entorno por
// materia de los laboratorios anteriores.
func (s *WorkspaceService) StartWorkspaceForExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.WorkspaceInstance, error) {
	return s.startWorkspace(ctx, studentID, subjectID, exerciseID, strings.TrimSpace(exerciseID) != "", true)
}

// StartTeacherPracticeWorkspace abre (o reutiliza) el entorno propio del docente
// para una práctica concreta. A diferencia del entorno de materia, el volumen se
// separa por ejercicio: dos prácticas de la misma materia no comparten repo.
func (s *WorkspaceService) StartTeacherPracticeWorkspace(ctx context.Context, teacherID, subjectID, exerciseID, role string) (*domain.WorkspaceInstance, error) {
	teacherID = strings.TrimSpace(teacherID)
	subjectID = strings.TrimSpace(subjectID)
	exerciseID = strings.TrimSpace(exerciseID)
	if teacherID == "" {
		return nil, errors.New("la identidad del docente es requerida")
	}
	if exerciseID == "" {
		return nil, errors.New("la práctica es requerida")
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	exercise := s.resolveExercise(ctx, exerciseID, tenantID)
	if exercise == nil {
		return nil, domain.ErrNotFound
	}
	if exercise.SubjectID != nil && strings.TrimSpace(*exercise.SubjectID) != "" {
		subjectID = strings.TrimSpace(*exercise.SubjectID)
	}
	if subjectID == "" {
		return nil, errors.New("la práctica no tiene materia asociada")
	}
	if exercise.EnvironmentType != string(domain.EnvironmentTypeIDEPersistente) {
		return nil, ErrPracticeIDENotProject
	}
	if s.subjectRepo == nil {
		return nil, errors.New("la materia de la práctica no está disponible")
	}
	subject, err := s.subjectRepo.GetByID(ctx, tenantID, subjectID)
	if err != nil || subject == nil {
		return nil, domain.ErrNotFound
	}
	if role != "admin" {
		if subject.TeacherID == nil || strings.TrimSpace(*subject.TeacherID) != teacherID {
			return nil, ErrPracticeTestForbidden
		}
	}

	// El IDE de la práctica es del docente, no de un alumno: no se inscribe
	// como participante del aula ni pisa la presencia del roster.
	return s.startWorkspace(ctx, teacherID, subjectID, exerciseID, true, false)
}

// containerAlive dice si el contenedor del entorno todavía existe en el daemon.
// Un fallo al listar no se interpreta como ausencia: ante la duda se conserva el
// entorno y se devuelve el registro, para no reconstruir por una caída del daemon.
func (s *WorkspaceService) containerAlive(ctx context.Context, containerID *string) bool {
	if s.docker == nil {
		return true
	}
	id := ""
	if containerID != nil {
		id = strings.TrimSpace(*containerID)
	}
	if id == "" {
		return false
	}
	ids, err := s.docker.ListAllManagedContainers(ctx)
	if err != nil {
		log.Printf("[Workspace] WARNING: no se pudo listar contenedores, se conserva el entorno %s: %v", id, err)
		return true
	}
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func (s *WorkspaceService) startWorkspace(ctx context.Context, userID, subjectID, exerciseID string, scopeByExercise, linkSession bool) (*domain.WorkspaceInstance, error) {
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
	if exercise != nil && exercise.SubjectID != nil && strings.TrimSpace(*exercise.SubjectID) != "" && strings.TrimSpace(*exercise.SubjectID) != subjectID {
		return nil, domain.ErrNotFound
	}

	var exerciseIDPtr *string
	if scopeByExercise {
		trimmed := strings.TrimSpace(exerciseID)
		if trimmed == "" {
			return nil, errors.New("la práctica del entorno es requerida")
		}
		exerciseIDPtr = &trimmed
	}

	// 0. Resolución de plantilla: el ejercicio manda sobre la materia.
	template := s.resolveTemplateForWorkspace(ctx, subjectID, exercise)

	policies := s.effectivePolicies(ctx, tenantID)
	ceilingPct := policies.HostSafetyCeilingPct
	if ceilingPct <= 0 {
		ceilingPct = 85
	}

	active, activeErr := s.repo.GetActiveWorkspaces(ctx)
	if activeErr != nil {
		log.Printf("[Admission] WARNING: no se pudo contar workspaces activos, admitiendo por fail-open: %v", activeErr)
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

	// 2. Verificación de Idempotencia y Estado Previos. El entorno por
	// práctica se busca por ejercicio para no reutilizar el de otra práctica.
	var existing *domain.WorkspaceInstance
	var err error
	if scopeByExercise {
		repo, ok := s.repo.(workspaceExerciseRepository)
		if !ok {
			return nil, errors.New("el repositorio de entornos no soporta entornos por práctica")
		}
		existing, err = repo.GetByStudentSubjectExercise(ctx, userID, subjectID, strings.TrimSpace(exerciseID))
	} else {
		existing, err = s.repo.GetByStudentAndSubject(ctx, userID, subjectID)
	}
	if err == nil && existing != nil {
		// Un examen cerrado no se reabre por recargar la página: el volumen quedó
		// en solo lectura y la escritura vuelve únicamente con la excepción del
		// docente (ADR-005 §5).
		if existing.FrozenAt != nil || existing.Status == domain.WorkspaceStatusFrozen {
			return nil, domain.ErrWorkspaceFrozen
		}

		// El entorno por materia es único por (student_id, subject_id) y el de
		// práctica por (student_id, subject_id, exercise_id). Si ya existía, la
		// base satélite y el db_init_script del ejercicio también deben quedar
		// asegurados antes de devolverlo o reactivarlo. El registro de
		// _solv_exercise_migrations hace la operación idempotente.
		provision, dbErr := s.ensureExerciseSatelliteDB(ctx, userID, subjectID, template, exercise)
		if dbErr != nil {
			return nil, dbErr
		}

		if existing.Status == domain.WorkspaceStatusRunning {
			// El registro puede sobrevivir al contenedor (limpieza de Docker,
			// reinicio del nodo). Antes de devolver la URL se comprueba que el
			// contenedor siga en el daemon; si ya no está, se reconstruye en vez
			// de entregar un enlace que no carga.
			if s.containerAlive(ctx, existing.ContainerID) {
				return existing, nil
			}
			return s.reactivateWorkspace(ctx, existing, provision, linkSession)
		}

		// Verificación de Castigo por OOMKilled (Sistema de 3 Strikes = 5 minutos de bloqueo)
		if existing.OOMStrikeCount >= domain.MaxOOMStrikes && existing.LastOOMKilledAt != nil {
			if time.Since(*existing.LastOOMKilledAt) < domain.OOMCooldownDuration {
				remaining := domain.OOMCooldownDuration - time.Since(*existing.LastOOMKilledAt)
				return nil, fmt.Errorf("%w (%d s remaining)", ErrOOMKilledCooldownPenalty, int(remaining.Seconds()))
			}
			_ = s.repo.ResetOOMStrikes(ctx, existing.ID)
		}

		return s.reactivateWorkspace(ctx, existing, provision, linkSession)
	}

	// 0.1 Techo de concurrencia del tenant (max_containers): hard gate.
	// Solo cuenta workspaces nuevos; reactivaciones de existentes no consumen cupo.
	if activeErr == nil && len(active) >= policies.MaxContainers {
		return nil, fmt.Errorf("%w: tenant %s tiene %d activos (techo %d)", ErrMaxContainersReached, tenantID, len(active), policies.MaxContainers)
	}

	// Control de Admisión por Presupuesto Comprometido (Host Safety Ceiling).
	// Los satélites son contenedores compartidos: su reserva se descuenta una sola
	// vez por motor, no por cada laboratorio que declara ese motor.
	if s.policiesService != nil && hostTotalMB() > 0 {
		var committedMB int64
		for _, ws := range active {
			if ws != nil {
				committedMB += ws.MemoryLimitMB
			}
		}
		reserveMB := s.nodeSatelliteReserveMB(ctx, active, template)
		if admissionBudgetDenied(committedMB, ramLimitMB, reserveMB, int64(hostTotalMB()), int64(ceilingPct)) {
			return nil, ErrHostMemoryExhausted
		}
	}

	provision, err := s.ensureExerciseSatelliteDB(ctx, userID, subjectID, template, exercise)
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
	volumeName := domain.WorkspaceVolumeName(userID, subjectID, exerciseIDPtr)
	networkName := "solv-traefik-net"

	instance := &domain.WorkspaceInstance{
		ID:              workspaceID,
		TenantID:        tenantID,
		StudentID:       userID,
		SubjectID:       subjectID,
		TemplateID:      templateIDStr,
		ExerciseID:      exerciseIDPtr,
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
	envVars = append(envVars, s.buildCompanionEnvVars(ctx, tenantID, subjectID, exerciseID, userID)...)

	// 8. Inyección de las credenciales del motor ya asegurado (ADR-030 / ADR-046):
	// DATABASE_URL y afines del satélite gestionado, o la base del aula en PostgreSQL
	// compartido. El helper ensureExerciseSatelliteDB ya creó la base y aplicó el
	// db_init_script del ejercicio en modo hard-fail.
	envVars = append(envVars, provision.Env...)

	// 9. Configuración del contenedor OpenVSCode Server
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

	// 12. El setup_script ya no se lanza como goroutine legacy: el shell de
	// arranque del contenedor lo ejecuta una sola vez por volumen, sellado con
	// .solv_initialized, por lo que una reactivación no lo duplica.

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning

	// El entorno acaba de nacer: si la materia tiene aula abierta, se registra
	// dentro de ella. El IDE de práctica del docente queda fuera del roster;
	// el del alumno sí entra, porque ese tablero es de alumnos.
	if linkSession {
		s.linkSessionToAula(ctx, instance)
	}
	return instance, nil
}

// buildCompanionEnvVars inyecta las variables de entorno para la extensión SOLV Companion (ADR-062).
func (s *WorkspaceService) buildCompanionEnvVars(ctx context.Context, tenantID, subjectID, exerciseID, studentID string) []string {
	var envs []string
	apiURL := os.Getenv("SOLV_API_URL")
	if apiURL == "" {
		if baseDomain := getBaseDomain(); baseDomain != "" {
			apiURL = fmt.Sprintf("https://api.%s", baseDomain)
		} else {
			apiURL = "http://host.docker.internal:3000"
		}
	}
	envs = append(envs, "SOLV_API_URL="+apiURL)

	if studentToken := domain.GetUserToken(ctx); studentToken != "" {
		envs = append(envs, "SOLV_STUDENT_TOKEN="+studentToken)
	}

	role := strings.TrimSpace(domain.GetUserRole(ctx))
	if role != "" {
		envs = append(envs, "SOLV_USER_ROLE="+role)
	}

	if exerciseID != "" {
		envs = append(envs, "SOLV_EXERCISE_ID="+exerciseID)
		if ex := s.resolveExercise(ctx, exerciseID, tenantID); ex != nil {
			if title := strings.TrimSpace(ex.Title); title != "" {
				envs = append(envs, "SOLV_ASSIGNMENT_TITLE="+title)
			}
			if desc := strings.TrimSpace(ex.Description); desc != "" {
				envs = append(envs, "SOLV_ASSIGNMENT_SUMMARY="+desc)
			}
			if purpose := strings.TrimSpace(ex.Purpose); purpose != "" {
				envs = append(envs, "SOLV_EXERCISE_PURPOSE="+purpose)
			}
			if subjectID == "" && ex.SubjectID != nil {
				subjectID = strings.TrimSpace(*ex.SubjectID)
			}
		}
		if s.submissionRepo != nil {
			if submissions, err := s.submissionRepo.ListByExerciseAndStudent(ctx, tenantID, exerciseID, studentID); err == nil && len(submissions) > 0 {
				envs = append(envs, "SOLV_SUBMISSION_ID="+submissions[0].ID)
			}
		}
	}

	if subjectID != "" {
		envs = append(envs, "SOLV_SUBJECT_ID="+subjectID)
		if s.subjectRepo != nil {
			if sub, err := s.subjectRepo.GetByID(ctx, tenantID, subjectID); err == nil && sub != nil {
				if code := strings.TrimSpace(sub.Code); code != "" {
					envs = append(envs, "SOLV_SUBJECT_CODE="+code)
				}
				if name := strings.TrimSpace(sub.Name); name != "" {
					envs = append(envs, "SOLV_SUBJECT_TITLE="+name)
				}
			}
		}
	}
	return envs
}
