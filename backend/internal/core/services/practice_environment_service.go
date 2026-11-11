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

// environmentExerciseRepository es una capacidad opcional del repositorio:
// resuelve el entorno de una práctica sin ensanchar la interfaz base que usan
// los dobles de test del ciclo de vida por materia.
type environmentExerciseRepository interface {
	GetByStudentSubjectExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.PracticeEnvironment, error)
}

type PracticeEnvironmentService struct {
	repo            domain.PracticeEnvironmentRepository
	docker          domain.PracticeEnvironmentOrchestrator
	hostMonitor     domain.HostMonitor
	subjectRepo     domain.SubjectRepository
	templateRepo    domain.TemplateRepository
	exerciseRepo    domain.ExerciseRepository
	submissionRepo  domain.SubmissionRepository
	policiesService *ServerPoliciesService
	// workspaceProjects es la fuente de la versión publicada que se asigna al
	// entorno. Sin este repositorio el ciclo anterior sigue funcionando, pero no
	// se siembra ningún proyecto.
	workspaceProjects domain.WorkspaceProjectRepository
	// dependencyPreparation entrega el árbol preparado de una versión para
	// enlazarlo al volumen recién sembrado. Es opcional: sin él el entorno abre
	// igual y el alumno instala al empezar.
	dependencyPreparation workspaceDependencySource
	// sessionLinker deja constancia de en qué aula nació el entorno. Es opcional:
	// sin él el ciclo de vida del contenedor funciona igual.
	sessionLinker domain.ClassSessionPracticeEnvironmentLinker
	// examFreezer cierra el volumen de un examen cuando el alumno entrega antes.
	// Es opcional: sin él, la entrega se registra y el cierre llega con el aula.
	examFreezer ExamPracticeEnvironmentFreezer

	// satelliteUseMu protege satelliteLastUse: el último momento en que un
	// laboratorio pidió un motor. Evita que el scale-to-zero apague el satélite en
	// el hueco entre el aprovisionamiento y el registro del laboratorio como activo.
	satelliteUseMu   sync.Mutex
	satelliteLastUse map[string]time.Time

	// workspaceSeedMu protege el mapa de bloqueos; workspaceSeedLocks serializa
	// la apertura de un mismo entorno dentro del proceso. La exclusión durable
	// entre procesos queda respaldada por el estado persistido del entorno.
	workspaceSeedMu    sync.Mutex
	workspaceSeedLocks map[string]*sync.Mutex
}

func NewPracticeEnvironmentService(repo domain.PracticeEnvironmentRepository, docker domain.PracticeEnvironmentOrchestrator, hostMonitor domain.HostMonitor) *PracticeEnvironmentService {
	return &PracticeEnvironmentService{
		repo:               repo,
		docker:             docker,
		hostMonitor:        hostMonitor,
		satelliteLastUse:   make(map[string]time.Time),
		workspaceSeedLocks: make(map[string]*sync.Mutex),
	}
}

// WithProvisioning extiende el servicio para resolver plantillas y bases de datos satélite en tiempo de ejecución
func (s *PracticeEnvironmentService) WithProvisioning(subjectRepo domain.SubjectRepository, templateRepo domain.TemplateRepository) *PracticeEnvironmentService {
	s.subjectRepo = subjectRepo
	s.templateRepo = templateRepo
	return s
}

// WithExerciseSubmission habilita el flujo de IDE Persistente por ejercicio:
// resolución de plantilla desde el ejercicio, db_init_script idempotente y
// entrega del proyecto hacia SpeedGrader. Es fluido para no romper las firmas
// existentes usadas por los tests.
func (s *PracticeEnvironmentService) WithExerciseSubmission(exerciseRepo domain.ExerciseRepository, submissionRepo domain.SubmissionRepository) *PracticeEnvironmentService {
	s.exerciseRepo = exerciseRepo
	s.submissionRepo = submissionRepo
	return s
}

// SetWorkspaceProjects inyecta la fuente de versiones publicadas. Es opcional
// para no romper los dobles de las fases anteriores; en producción se conecta
// al mismo repositorio que usa el editor docente.
// SetDependencyPreparation engancha la preparación de dependencias. El enlace
// solo ocurre al sembrar un volumen: un volumen ya inicializado no se toca.
func (s *PracticeEnvironmentService) SetDependencyPreparation(source workspaceDependencySource) *PracticeEnvironmentService {
	if s == nil {
		return s
	}
	s.dependencyPreparation = source
	return s
}

func (s *PracticeEnvironmentService) SetWorkspaceProjects(repo domain.WorkspaceProjectRepository) *PracticeEnvironmentService {
	if s == nil {
		return s
	}
	s.workspaceProjects = repo
	return s
}

// SetExamSubmissionFreezer habilita el cierre anticipado por alumno: al entregar
// un examen, solo su volumen pasa a solo lectura (ADR-005 §5.2).
func (s *PracticeEnvironmentService) SetExamSubmissionFreezer(freezer ExamPracticeEnvironmentFreezer) *PracticeEnvironmentService {
	s.examFreezer = freezer
	return s
}

// SetPoliciesService inyecta las políticas QoS del tenant (submódulo 14.6):
// la RAM de la política es el default por alumno y max_containers el techo
// de concurrencia en la admisión. Sin inyección rigen los defaults del dominio.
func (s *PracticeEnvironmentService) SetPoliciesService(policies *ServerPoliciesService) *PracticeEnvironmentService {
	s.policiesService = policies
	return s
}

// WithClassSessions enlaza el ciclo de vida del entorno con el aula abierta de
// la materia, si la hay. El enlace es best-effort: un fallo al registrar la
// sesión nunca impide que el alumno trabaje.
func (s *PracticeEnvironmentService) WithClassSessions(linker domain.ClassSessionPracticeEnvironmentLinker) *PracticeEnvironmentService {
	s.sessionLinker = linker
	return s
}

// linkSessionToAula registra el entorno en el aula abierta de la materia y marca
// la presencia del alumno. Sin aula abierta no hace nada: el trabajo fuera de
// clase no pertenece a ninguna sesión.
func (s *PracticeEnvironmentService) linkSessionToAula(ctx context.Context, instance *domain.PracticeEnvironment) {
	if s.sessionLinker == nil || instance == nil {
		return
	}
	if err := s.sessionLinker.AttachPracticeEnvironmentToOpenSession(ctx, instance.TenantID, instance.StudentID, instance.SubjectID, instance.ID); err != nil {
		log.Printf("[PracticeEnvironment] WARNING: no se pudo enlazar el entorno %s con el aula de la materia %s: %v", instance.ID, instance.SubjectID, err)
	}
}

// effectivePolicies resuelve las políticas vigentes del tenant con fallback a
// defaults ante cualquier fallo (fail-open: una política nunca voltea clases;
// la protección dura del host vive en el admission control físico).
func (s *PracticeEnvironmentService) effectivePolicies(ctx context.Context, tenantID string) domain.ServerPolicies {
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

// reservePracticeEnvironmentHostPort pide al kernel un puerto libre en 127.0.0.1 que Docker
// reutiliza en el PortBindings de 3000/tcp.
func reservePracticeEnvironmentHostPort() (string, error) {
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

// buildPracticeEnvironmentAccessURL resuelve el acceso al environment publicado en el host.
func buildPracticeEnvironmentAccessURL(environmentID, hostPort string) string {
	if baseDomain := getBaseDomain(); baseDomain != "" {
		return fmt.Sprintf("https://%s.%s", environmentID, baseDomain)
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

// environmentAccessURLUpdater es una capacidad opcional del repositorio de
// environments: persiste el access_url cuando se recrea el contenedor y cambia el
// puerto publicado. Se resuelve por type-assertion para no ampliar
// domain.PracticeEnvironmentRepository ni romper sus mocks.
type environmentAccessURLUpdater interface {
	UpdateAccessURL(ctx context.Context, id, accessURL string) error
}

func sanitizeIDForDB(id string) string {
	cleaned := strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(cleaned) > 8 {
		return cleaned[:8]
	}
	return cleaned
}

func (s *PracticeEnvironmentService) resolveTemplate(ctx context.Context, subjectID string) *domain.Template {
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

// resolveExercise carga el ejercicio del tenant cuando el environment se inicia
// desde una actividad concreta. Que la práctica no exista es una respuesta
// normal (el llamador devuelve 404), no un fallo del entorno: solo se registra
// cuando la lectura falla de verdad y el servicio no puede distinguir el caso.
func (s *PracticeEnvironmentService) resolveExercise(ctx context.Context, exerciseID, tenantID string) *domain.Exercise {
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
		log.Printf("[PracticeEnvironment] no se pudo leer el ejercicio %s: %v", exerciseID, err)
	case !isMissingRow(fallbackErr):
		log.Printf("[PracticeEnvironment] no se pudo leer el ejercicio %s: %v", exerciseID, fallbackErr)
	}
	return nil
}

// resolveTemplateForPracticeEnvironment prioriza la plantilla del ejercicio sobre la de la
// materia (hallazgo #8): el ejercicio define la imagen real con la que se trabaja.
func (s *PracticeEnvironmentService) resolveTemplateForPracticeEnvironment(ctx context.Context, subjectID string, exercise *domain.Exercise) *domain.Template {
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
func (s *PracticeEnvironmentService) resolveTemplateForInstance(ctx context.Context, instance *domain.PracticeEnvironment) *domain.Template {
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
func (s *PracticeEnvironmentService) ensureExerciseSatelliteDB(ctx context.Context, studentID, subjectID string, template *domain.Template, exercise *domain.Exercise) (satelliteProvision, error) {
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

// sanitizePracticeEnvironmentSetupScript devuelve el script de arranque de la plantilla
// sin espacios sobrantes; una plantilla sin script no arranca con "".
func sanitizePracticeEnvironmentSetupScript(template *domain.Template) string {
	if template == nil {
		return ""
	}
	return strings.TrimSpace(template.SetupScript)
}

// StartPracticeEnvironmentForExercise abre el entorno del alumno acotado a la práctica.
// Abrir la práctica 2 no debe devolver el entorno de la materia (que mezclaba el
// código de todas las prácticas) ni reutilizar su volumen: el entorno y el
// volumen se separan por (alumno, materia, práctica), igual que el del docente
// (ADR-059). La práctica es obligatoria.
func (s *PracticeEnvironmentService) StartPracticeEnvironmentForExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.PracticeEnvironment, error) {
	return s.startPracticeEnvironment(ctx, studentID, subjectID, exerciseID, true)
}

// StartTeacherPracticePracticeEnvironment abre (o reutiliza) el entorno propio del docente
// para una práctica concreta. A diferencia del entorno de materia, el volumen se
// separa por ejercicio: dos prácticas de la misma materia no comparten repo.
func (s *PracticeEnvironmentService) StartTeacherPracticePracticeEnvironment(ctx context.Context, teacherID, subjectID, exerciseID, role string) (*domain.PracticeEnvironment, error) {
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
	return s.startPracticeEnvironment(ctx, teacherID, subjectID, exerciseID, false)
}

// containerAlive dice si el contenedor del entorno todavía existe en el daemon.
// Un fallo al listar no se interpreta como ausencia: ante la duda se conserva el
// entorno y se devuelve el registro, para no reconstruir por una caída del daemon.
func (s *PracticeEnvironmentService) containerAlive(ctx context.Context, containerID *string) bool {
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
		log.Printf("[PracticeEnvironment] WARNING: no se pudo listar contenedores, se conserva el entorno %s: %v", id, err)
		return true
	}
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

// startPracticeEnvironment abre (o reutiliza) el entorno de una práctica. El alcance es
// siempre un ejercicio: sin práctica no hay entorno, porque un entorno por
// materia mezclaría el código de todas las prácticas en un mismo volumen.
func (s *PracticeEnvironmentService) startPracticeEnvironment(ctx context.Context, userID, subjectID, exerciseID string, linkSession bool) (*domain.PracticeEnvironment, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	exercise := s.resolveExercise(ctx, exerciseID, tenantID)
	// Sin catálogo de ejercicios el servicio no puede afirmar que la práctica no
	// existe. En producción el catálogo siempre está inyectado, así que la
	// validación sigue vigente.
	if strings.TrimSpace(exerciseID) != "" && exercise == nil && s.exerciseRepo != nil {
		return nil, fmt.Errorf("exercise %s not found", exerciseID)
	}
	if strings.TrimSpace(subjectID) == "" && exercise != nil && exercise.SubjectID != nil {
		subjectID = strings.TrimSpace(*exercise.SubjectID)
	}
	if exercise != nil && exercise.SubjectID != nil && strings.TrimSpace(*exercise.SubjectID) != "" && strings.TrimSpace(*exercise.SubjectID) != subjectID {
		return nil, domain.ErrNotFound
	}

	trimmedExerciseID := strings.TrimSpace(exerciseID)
	if trimmedExerciseID == "" {
		return nil, domain.ErrPracticeEnvironmentPracticeRequired
	}
	exerciseIDPtr := &trimmedExerciseID

	// El lock se toma después de normalizar materia y práctica: dos llamadas
	// equivalentes deben compartir la misma clave aunque una llegue sin
	// subject_id y la otra con el valor que resuelve el ejercicio.
	unlock := s.lockWorkspaceStart(tenantID, userID, subjectID, trimmedExerciseID)
	defer unlock()

	// 0. Resolución de plantilla: el ejercicio manda sobre la materia.
	template := s.resolveTemplateForPracticeEnvironment(ctx, subjectID, exercise)

	policies := s.effectivePolicies(ctx, tenantID)
	ceilingPct := policies.HostSafetyCeilingPct
	if ceilingPct <= 0 {
		ceilingPct = 85
	}

	active, activeErr := s.repo.GetActivePracticeEnvironments(ctx)
	if activeErr != nil {
		log.Printf("[Admission] WARNING: no se pudo contar environments activos, admitiendo por fail-open: %v", activeErr)
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

	// 2. Verificación de Idempotencia y Estado Previos. El entorno se busca por
	// ejercicio para no reutilizar el de otra práctica de la misma materia.
	repo, ok := s.repo.(environmentExerciseRepository)
	if !ok {
		return nil, errors.New("el repositorio de entornos no soporta entornos por práctica")
	}
	existing, err := repo.GetByStudentSubjectExercise(ctx, userID, subjectID, trimmedExerciseID)
	if err == nil && existing != nil {
		// Un examen cerrado no se reabre por recargar la página: el volumen quedó
		// en solo lectura y la escritura vuelve únicamente con la excepción del
		// docente (ADR-005 §5).
		if existing.FrozenAt != nil || existing.Status == domain.PracticeEnvironmentStatusFrozen {
			return nil, domain.ErrPracticeEnvironmentFrozen
		}

		// El entorno es único por (student_id, subject_id, exercise_id). Si ya
		// existía, la base satélite y el db_init_script del ejercicio también
		// deben quedar asegurados antes de devolverlo o reactivarlo. El registro
		// de _solv_exercise_migrations hace la operación idempotente.
		provision, dbErr := s.ensureExerciseSatelliteDB(ctx, userID, subjectID, template, exercise)
		if dbErr != nil {
			return nil, dbErr
		}
		if seedErr := s.ensureExistingWorkspaceSeed(ctx, existing); seedErr != nil {
			_ = s.repo.UpdateStatus(ctx, existing.ID, domain.PracticeEnvironmentStatusFailed)
			return nil, seedErr
		}

		if existing.Status == domain.PracticeEnvironmentStatusRunning {
			// El registro puede sobrevivir al contenedor (limpieza de Docker,
			// reinicio del nodo). Antes de devolver la URL se comprueba que el
			// contenedor siga en el daemon; si ya no está, se reconstruye en vez
			// de entregar un enlace que no carga.
			if s.containerAlive(ctx, existing.ContainerID) {
				return existing, nil
			}
			return s.reactivatePracticeEnvironment(ctx, existing, provision, linkSession)
		}

		// Verificación de Castigo por OOMKilled (Sistema de 3 Strikes = 5 minutos de bloqueo)
		if existing.OOMStrikeCount >= domain.MaxOOMStrikes && existing.LastOOMKilledAt != nil {
			if time.Since(*existing.LastOOMKilledAt) < domain.OOMCooldownDuration {
				remaining := domain.OOMCooldownDuration - time.Since(*existing.LastOOMKilledAt)
				return nil, fmt.Errorf("%w (%d s remaining)", ErrOOMKilledCooldownPenalty, int(remaining.Seconds()))
			}
			_ = s.repo.ResetOOMStrikes(ctx, existing.ID)
		}

		return s.reactivatePracticeEnvironment(ctx, existing, provision, linkSession)
	}

	// 0.1 Techo de concurrencia del tenant (max_containers): hard gate.
	// Solo cuenta environments nuevos; reactivaciones de existentes no consumen cupo.
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

	// 3. Generación de UUID opaco para el environment_id y construcción de access_url
	environmentID := uuid.NewString()
	baseDomain := getBaseDomain()
	hostPort, portErr := reservePracticeEnvironmentHostPort()
	if portErr != nil {
		log.Printf("[PracticeEnvironment] WARNING: no se pudo reservar un puerto local para %s: %v", environmentID, portErr)
	}
	accessURL := buildPracticeEnvironmentAccessURL(environmentID, hostPort)
	containerName := fmt.Sprintf("solv-environment-%s", environmentID)
	volumeName := domain.PracticeEnvironmentVolumeName(userID, subjectID, exerciseIDPtr)
	networkName := "solv-traefik-net"

	seedVersion, err := s.resolveWorkspaceSeed(ctx, tenantID, exercise)
	if err != nil {
		return nil, err
	}
	seedMode := domain.WorkspaceSeedModeNone
	seedStatus := domain.WorkspaceInitStatusNotRequired
	var seedVersionID *string
	if seedVersion != nil {
		seedMode = domain.WorkspaceSeedModeProject
		seedStatus = domain.WorkspaceInitStatusPending
		seedVersionID = &seedVersion.ID
	}

	instance := &domain.PracticeEnvironment{
		ID:                        environmentID,
		TenantID:                  tenantID,
		StudentID:                 userID,
		SubjectID:                 subjectID,
		TemplateID:                templateIDStr,
		ExerciseID:                exerciseIDPtr,
		Status:                    domain.PracticeEnvironmentStatusPending,
		AccessURL:                 accessURL,
		MemoryLimitMB:             ramLimitMB,
		LastHeartbeatAt:           time.Now(),
		WorkspaceProjectVersionID: seedVersionID,
		WorkspaceSeedMode:         seedMode,
		WorkspaceInitStatus:       seedStatus,
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}

	// 4. Crear registro pendiente en PostgreSQL mediante sqlx
	if err := s.repo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create pending environment: %w", err)
	}

	// 5. Asegurar la existencia del Volumen Nombrado (ADR-001) para la materia y estudiante
	if err := s.docker.EnsureVolumeExists(ctx, volumeName); err != nil {
		_ = s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusFailed)
		return nil, fmt.Errorf("failed to ensure named volume %s: %w", volumeName, err)
	}

	// El proyecto se escribe antes de arrancar el IDE. Si la inicialización no
	// termina, no se entrega un contenedor vacío: el entorno queda fallido y el
	// volumen conserva lo que haya para una revisión segura.
	if seedVersion != nil {
		seed, seedErr := workspaceSeedFromVersion(seedVersion)
		if seedErr != nil {
			_ = s.setWorkspaceInitStatus(ctx, instance, domain.WorkspaceInitStatusQuarantined)
			_ = s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusFailed)
			return nil, seedErr
		}
		if seedErr := s.seedWorkspaceVolume(ctx, instance, seed); seedErr != nil {
			_ = s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusFailed)
			return nil, seedErr
		}
	}

	// 6. Asegurar la existencia de la red Docker compartida con Traefik deshabilitando ICC (enable_icc=false)
	if err := s.docker.EnsureICCDisabledNetworkExists(ctx, networkName); err != nil {
		_ = s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusFailed)
		return nil, fmt.Errorf("failed to ensure ICC-disabled docker network %s: %w", networkName, err)
	}

	// 7. Dynamic Labels de Traefik v3 (Puerto 3000 para OpenVSCode Server). Solo se
	// publican cuando existe un dominio público configurado; el acceso local no
	// depende de Traefik ni de .local.
	labels := map[string]string{}
	if baseDomain != "" {
		labels["traefik.enable"] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", environmentID)] = fmt.Sprintf("Host(`%s.%s`)", environmentID, baseDomain)
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", environmentID)] = "3000"
	}

	envVars := []string{
		"DONT_PROMPT_WSL_INSTALL=true",
	}
	envVars = append(envVars, s.buildCompanionEnvVars(ctx, tenantID, subjectID, exerciseID, userID)...)
	// El almacén de dependencias se preparó en /home/workspace: las variables del
	// perfil son las que hacen que el alumno lo encuentre en esa misma ruta.
	envVars = append(envVars, s.workspaceDependencyEnvVars(ctx, tenantID, exerciseID)...)

	// 8. Inyección de las credenciales del motor ya asegurado (ADR-030 / ADR-046):
	// DATABASE_URL y afines del satélite gestionado, o la base del aula en PostgreSQL
	// compartido. El helper ensureExerciseSatelliteDB ya creó la base y aplicó el
	// db_init_script del ejercicio en modo hard-fail.
	envVars = append(envVars, provision.Env...)

	// 9. Configuración del contenedor OpenVSCode Server
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

	// 10. Instanciación e inicio del contenedor OpenVSCode Server
	containerID, err := s.docker.StartPracticeEnvironmentContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusFailed)
		return nil, fmt.Errorf("failed to start openvscode-server container: %w", err)
	}

	// 11. Actualización del registro en PostgreSQL a estado 'running'
	if err := s.repo.UpdateContainerID(ctx, environmentID, containerID); err != nil {
		return nil, fmt.Errorf("failed to update container_id: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, environmentID, domain.PracticeEnvironmentStatusRunning); err != nil {
		return nil, fmt.Errorf("failed to update environment status: %w", err)
	}

	// 12. El setup_script ya no se lanza como goroutine legacy: el shell de
	// arranque del contenedor lo ejecuta una sola vez por volumen, sellado con
	// .solv_initialized, por lo que una reactivación no lo duplica.

	instance.ContainerID = &containerID
	instance.Status = domain.PracticeEnvironmentStatusRunning

	// El entorno acaba de nacer: si la materia tiene aula abierta, se registra
	// dentro de ella. El IDE de práctica del docente queda fuera del roster;
	// el del alumno sí entra, porque ese tablero es de alumnos.
	if linkSession {
		s.linkSessionToAula(ctx, instance)
	}
	return instance, nil
}

// buildCompanionEnvVars inyecta las variables de entorno para la extensión SOLV Companion (ADR-062).
func (s *PracticeEnvironmentService) buildCompanionEnvVars(ctx context.Context, tenantID, subjectID, exerciseID, studentID string) []string {
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
