package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

var (
	ErrPracticeTestForbidden   = errors.New("no tienes acceso a esta práctica")
	ErrPracticeTestRequiresIDE = errors.New("Probar Entorno solo aplica a prácticas con IDE persistente")
)

// PracticeEnvTestConfig acota la ejecución de las pruebas de entorno de una práctica.
type PracticeEnvTestConfig struct {
	MaxConcurrentJobs int
	TotalJobTimeout   time.Duration
	SetupTimeout      time.Duration
}

// PracticeEnvTestRequest identifica la práctica y al docente que pide la prueba.
// La identidad y el tenant salen del contexto autenticado, nunca del cuerpo.
type PracticeEnvTestRequest struct {
	TeacherID  string
	TenantID   string
	Role       string
	CourseID   string
	ExerciseID string
}

// PracticeEnvTestService valida una práctica IDE sin crear entregas ni environments:
// aprovisiona sus satélites, ejecuta el setup de la plantilla sobre un volumen temporal
// y reporta el resultado con el vocabulario de EnvTestStatus.
// StudentPracticeDatabaseLister lista las bases de los alumnos que ya abrieron una
// práctica. Es una capacidad opcional: sin ella, el listado responde no disponible.
type StudentPracticeDatabaseLister interface {
	ListStudentDatabasesByExercise(ctx context.Context, tenantID, subjectID, exerciseID string) ([]domain.StudentPracticeDatabase, error)
}

type PracticeEnvTestService struct {
	repo       domain.EnvTestJobRepository
	registry   domain.ImageRegistryPort
	runner     domain.PracticeEnvironmentTestRunnerPort
	satellites PracticeSatelliteProvisioner
	exercises  domain.ExerciseRepository
	subjects   domain.SubjectRepository
	templates  domain.TemplateRepository
	databases  StudentPracticeDatabaseLister
	config     PracticeEnvTestConfig

	sem         chan struct{}
	cancelMu    sync.Mutex
	cancelFuncs map[string]context.CancelFunc
}

func NewPracticeEnvTestService(
	repo domain.EnvTestJobRepository,
	registry domain.ImageRegistryPort,
	runner domain.PracticeEnvironmentTestRunnerPort,
	satellites PracticeSatelliteProvisioner,
	exercises domain.ExerciseRepository,
	subjects domain.SubjectRepository,
	templates domain.TemplateRepository,
	databases StudentPracticeDatabaseLister,
	config PracticeEnvTestConfig,
) *PracticeEnvTestService {
	if config.MaxConcurrentJobs <= 0 {
		config.MaxConcurrentJobs = 1
	}
	if config.TotalJobTimeout <= 0 {
		config.TotalJobTimeout = 10 * time.Minute
	}
	if config.SetupTimeout <= 0 {
		config.SetupTimeout = 5 * time.Minute
	}
	return &PracticeEnvTestService{
		repo:        repo,
		registry:    registry,
		runner:      runner,
		satellites:  satellites,
		exercises:   exercises,
		subjects:    subjects,
		templates:   templates,
		databases:   databases,
		config:      config,
		sem:         make(chan struct{}, config.MaxConcurrentJobs),
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// Start valida la práctica y despacha la prueba en segundo plano.
func (s *PracticeEnvTestService) Start(ctx context.Context, req PracticeEnvTestRequest) (*domain.EnvTestJob, error) {
	exerciseID := strings.TrimSpace(req.ExerciseID)
	if exerciseID == "" {
		return nil, errors.New("la práctica es requerida")
	}
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	exercise, subject, subjectID, err := s.resolvePracticeScope(ctx, req.TeacherID, req.Role, tenantID, req.CourseID, exerciseID)
	if err != nil {
		return nil, err
	}

	template, err := s.resolveTemplate(ctx, exercise, subject)
	if err != nil {
		return nil, err
	}
	if template == nil || strings.TrimSpace(template.DockerImage) == "" {
		return nil, errors.New("la práctica no tiene una plantilla con imagen para probar")
	}
	image := strings.TrimSpace(template.DockerImage)
	if strings.HasSuffix(image, ":latest") {
		return nil, errors.New("la imagen de la práctica usa el tag :latest, no permitido por reproducibilidad")
	}

	ramMB := int64(template.BaseRamMB)
	if ramMB < 256 {
		ramMB = 512
	}
	if err := domain.ValidateRamAgainstHost(int(ramMB), domain.CalculateHostMaxAllowedRAM(hostTotalMB())); err != nil {
		return nil, fmt.Errorf("la plantilla de la práctica pide %d MB y excede la capacidad del host: %w", ramMB, err)
	}

	jobID := uuid.NewString()
	job := &domain.EnvTestJob{
		ID:                jobID,
		ExerciseID:        exercise.ID,
		SubjectID:         subjectID,
		Image:             image,
		TargetEnvironment: string(domain.EnvironmentTypeIDEPersistente),
		OwnerID:           req.TeacherID,
		TenantID:          tenantID,
		Status:            domain.EnvTestStatusPending,
		Progress: domain.EnvTestProgress{
			CurrentAction: "Encolado, esperando slot de ejecución...",
		},
	}
	if err := s.repo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("error al registrar la prueba de la práctica: %w", err)
	}

	jobCtx, cancel := context.WithTimeout(context.Background(), s.config.TotalJobTimeout)
	s.cancelMu.Lock()
	s.cancelFuncs[jobID] = cancel
	s.cancelMu.Unlock()

	//nolint:gosec // La prueba sobrevive al request HTTP por diseño.
	go s.execute(jobCtx, jobID, exercise, template, subjectID, ramMB)

	return job, nil
}

// Latest devuelve la última prueba de la práctica para el docente autenticado.
// Sin pruebas devuelve nil, nil: el frontend muestra el estado vacío.
func (s *PracticeEnvTestService) Latest(ctx context.Context, teacherID, tenantID, role, exerciseID string) (*domain.EnvTestJob, error) {
	job, err := s.repo.GetLatestByExercise(ctx, exerciseID)
	if errors.Is(err, domain.ErrEnvTestJobNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if job.TenantID != tenantID {
		return nil, nil
	}
	if role != "admin" && job.OwnerID != teacherID {
		return nil, nil
	}
	return job, nil
}

// CancelJob solicita la cancelación del job y del contexto de ejecución.
func (s *PracticeEnvTestService) CancelJob(ctx context.Context, jobID string) error {
	s.cancelMu.Lock()
	if cancel, exists := s.cancelFuncs[jobID]; exists {
		cancel()
		delete(s.cancelFuncs, jobID)
	}
	s.cancelMu.Unlock()
	return s.repo.Cancel(ctx, jobID)
}

func (s *PracticeEnvTestService) resolveTemplate(ctx context.Context, exercise *domain.Exercise, subject *domain.Subject) (*domain.Template, error) {
	if s.templates == nil {
		return nil, nil
	}
	templateID := strings.TrimSpace(exercise.TemplateID)
	if templateID == "" && subject != nil && subject.TemplateID != nil {
		templateID = strings.TrimSpace(*subject.TemplateID)
	}
	if templateID == "" {
		return nil, nil
	}
	template, err := s.templates.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo resolver la plantilla de la práctica: %w", err)
	}
	return template, nil
}

func (s *PracticeEnvTestService) execute(ctx context.Context, jobID string, exercise *domain.Exercise, template *domain.Template, subjectID string, ramMB int64) {
	defer func() {
		s.cancelMu.Lock()
		delete(s.cancelFuncs, jobID)
		s.cancelMu.Unlock()
	}()

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		_ = s.repo.Cancel(context.Background(), jobID)
		return
	}

	image := template.DockerImage
	_ = s.repo.UpdateProgress(ctx, jobID, domain.EnvTestProgress{
		CurrentAction: "Verificando la imagen de la práctica...",
	})

	isLocal, _, _, err := s.registry.InspectLocal(ctx, image)
	if err != nil {
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrInternal, fmt.Sprintf("No se pudo inspeccionar la imagen: %v", err))
		return
	}
	if !isLocal {
		_ = s.repo.UpdateProgress(ctx, jobID, domain.EnvTestProgress{
			CurrentAction: "Descargando la imagen de la práctica...",
		})
		pullErr := s.registry.PullImage(ctx, image, func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string) {
			percent := float64(0)
			if totalBytes > 0 {
				percent = float64(doneBytes) / float64(totalBytes) * 100
				if percent > 100 {
					percent = 100
				}
			}
			_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
				BytesDone:     doneBytes,
				BytesTotal:    totalBytes,
				LayerCurrent:  currentLayer,
				LayersTotal:   totalLayers,
				Percent:       percent,
				CurrentAction: action,
			})
		})
		if pullErr != nil {
			if errors.Is(pullErr, context.Canceled) {
				_ = s.repo.Cancel(context.Background(), jobID)
				return
			}
			code := domain.EnvTestErrPullTimeout
			if isNotFoundError(pullErr) {
				code = domain.EnvTestErrNotFoundRemote
			} else if isNetworkOrDNSError(pullErr) {
				code = domain.EnvTestErrNetworkFailure
			}
			_ = s.repo.Fail(context.Background(), jobID, code, fmt.Sprintf("No se pudo preparar la imagen de la práctica: %v", pullErr))
			return
		}
	}

	_ = s.repo.MarkTesting(ctx, jobID, domain.EnvTestProgress{
		Percent:       35,
		CurrentAction: "Aprovisionando las bases satélite de la práctica...",
	})

	provision, err := s.satellites.ProvisionPracticeSatellites(ctx, exercise, template, subjectID)
	if err != nil {
		if ctx.Err() != nil {
			_ = s.repo.Cancel(context.Background(), jobID)
			return
		}
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrSatelliteFailed, fmt.Sprintf("No se pudieron preparar los satélites: %v", err))
		return
	}
	if ctx.Err() != nil {
		_ = s.repo.Cancel(context.Background(), jobID)
		return
	}

	_ = s.repo.MarkTesting(ctx, jobID, domain.EnvTestProgress{
		Percent:       60,
		CurrentAction: "Preparando el repositorio base y las dependencias...",
	})

	volumeName := "solv_envtest_" + strings.ReplaceAll(jobID, "-", "")
	networkName := provision.NetworkName
	if networkName == "" {
		// Sin satélites la prueba igual necesita red para el setup de dependencias;
		// el contenedor es efímero y no publica puertos.
		networkName = "bridge"
	}
	result, runErr := s.runner.RunPracticeEnvironmentSetupTest(ctx, domain.PracticeEnvironmentSetupTestSpec{
		Image:         image,
		VolumeName:    volumeName,
		NetworkName:   networkName,
		Env:           provision.Env,
		SetupScript:   template.SetupScript,
		MemoryLimitMB: ramMB,
		TimeoutMS:     int(s.config.SetupTimeout / time.Millisecond),
	})
	if runErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, "La prueba del environment superó el tiempo total de ejecución")
			return
		}
		if errors.Is(runErr, context.Canceled) || ctx.Err() != nil {
			_ = s.repo.Cancel(context.Background(), jobID)
			return
		}
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("La prueba del environment no terminó: %v", runErr))
		return
	}

	tools := []domain.ToolResult{
		{Name: "repositorio base", Present: result.RepoReady, Path: "/home/workspace", Version: "volumen temporal"},
		{Name: "openvscode-server", Present: result.OpenVSCodeSeen, Path: "/home/.openvscode-server/bin/openvscode-server"},
	}
	tools = append(tools, practiceSatelliteTools(provision.Engines)...)
	testResult := &domain.EnvTestResult{
		Tools:      tools,
		ExitCode:   result.ExitCode,
		DurationMs: result.DurationMs,
	}

	if result.ExitCode != 0 {
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrSetupFailed, setupFailureMessage(result))
		s.attachResult(jobID, testResult)
		return
	}
	if !result.RepoReady {
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrSetupFailed, "El volumen de trabajo de la práctica no se puede escribir en /home/workspace")
		s.attachResult(jobID, testResult)
		return
	}
	if !result.OpenVSCodeSeen {
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrSetupFailed, "La imagen no incluye OpenVSCode Server en /home/.openvscode-server/bin/openvscode-server")
		s.attachResult(jobID, testResult)
		return
	}

	_ = s.repo.Complete(context.Background(), jobID, testResult)
}

func (s *PracticeEnvTestService) attachResult(jobID string, result *domain.EnvTestResult) {
	job, err := s.repo.GetByID(context.Background(), jobID)
	if err != nil || job == nil {
		return
	}
	job.Result = result
	_ = s.repo.Save(context.Background(), job)
}

func setupFailureMessage(result domain.PracticeEnvironmentSetupTestResult) string {
	output := setupLogExcerpt(result.Output)
	if len(output) > 600 {
		output = "…" + output[len(output)-600:]
	}
	if output == "" {
		return fmt.Sprintf("El setup de la práctica terminó con código de salida %d", result.ExitCode)
	}
	return fmt.Sprintf("El setup de la práctica terminó con código de salida %d: %s", result.ExitCode, output)
}

// setupLogExcerpt devuelve el error real del setup, no los marcadores internos
// que el contenedor imprime para reportar repositorio y editor.
func setupLogExcerpt(output string) string {
	const (
		begin = "SOLV_SETUP_LOG_BEGIN"
		end   = "SOLV_SETUP_LOG_END"
	)
	if start := strings.Index(output, begin); start >= 0 {
		excerpt := output[start+len(begin):]
		if stop := strings.Index(excerpt, end); stop >= 0 {
			excerpt = excerpt[:stop]
		}
		return strings.TrimSpace(excerpt)
	}

	lines := make([]string, 0, 8)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SOLV_REPO:") ||
			strings.HasPrefix(trimmed, "SOLV_VSCODE:") ||
			strings.HasPrefix(trimmed, "SOLV_SETUP_EXIT:") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// resolvePracticeScope valida que la práctica exista, sea IDE, pertenezca a la materia
// indicada y que el docente sea su titular. Lo comparten la prueba de entorno y el
// reinicio de bases para que ambas puertas apliquen la misma política.
func (s *PracticeEnvTestService) resolvePracticeScope(
	ctx context.Context,
	teacherID, role, tenantID, courseID, exerciseID string,
) (*domain.Exercise, *domain.Subject, string, error) {
	exerciseID = strings.TrimSpace(exerciseID)
	if exerciseID == "" {
		return nil, nil, "", errors.New("la práctica es requerida")
	}
	exercise, err := s.exercises.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil || exercise == nil {
		return nil, nil, "", domain.ErrNotFound
	}
	if exercise.EnvironmentType != string(domain.EnvironmentTypeIDEPersistente) {
		return nil, nil, "", ErrPracticeTestRequiresIDE
	}

	subjectID := strings.TrimSpace(courseID)
	if subjectID == "" && exercise.SubjectID != nil {
		subjectID = strings.TrimSpace(*exercise.SubjectID)
	}
	if subjectID == "" {
		return nil, nil, "", errors.New("la práctica no tiene materia asociada")
	}
	if exercise.SubjectID != nil && strings.TrimSpace(*exercise.SubjectID) != "" && strings.TrimSpace(*exercise.SubjectID) != subjectID {
		return nil, nil, "", domain.ErrNotFound
	}

	subject, err := s.subjects.GetByID(ctx, tenantID, subjectID)
	if err != nil || subject == nil {
		return nil, nil, "", domain.ErrNotFound
	}
	if role != "admin" {
		if subject.TeacherID == nil || strings.TrimSpace(*subject.TeacherID) != strings.TrimSpace(teacherID) {
			return nil, nil, "", ErrPracticeTestForbidden
		}
	}
	return exercise, subject, subjectID, nil
}

var (
	// ErrPracticeDatabasesUnavailable indica que el servicio no puede listar las bases.
	ErrPracticeDatabasesUnavailable = errors.New("el listado de bases de la práctica no está disponible")
	// ErrStudentPracticeNotStarted indica que el alumno todavía no abrió la práctica,
	// así que no tiene una base propia que reiniciar.
	ErrStudentPracticeNotStarted = errors.New("el alumno todavía no abrió esta práctica")
)

// ListStudentDatabases devuelve quiénes ya abrieron la práctica y tienen base propia.
func (s *PracticeEnvTestService) ListStudentDatabases(
	ctx context.Context,
	teacherID, tenantID, role, courseID, exerciseID string,
) ([]domain.StudentPracticeDatabase, error) {
	exercise, _, subjectID, err := s.resolvePracticeScope(ctx, teacherID, role, tenantID, courseID, exerciseID)
	if err != nil {
		return nil, err
	}
	if s.databases == nil {
		return nil, ErrPracticeDatabasesUnavailable
	}
	return s.databases.ListStudentDatabasesByExercise(ctx, tenantID, subjectID, exercise.ID)
}

// ResetStudentDatabase borra la base de un alumno y la recrea con el script vigente.
// Solo opera sobre alumnos que ya abrieron la práctica: no se crean bases al azar.
func (s *PracticeEnvTestService) ResetStudentDatabase(
	ctx context.Context,
	teacherID, tenantID, role, courseID, exerciseID, studentID string,
) error {
	studentID = strings.TrimSpace(studentID)
	if studentID == "" {
		return errors.New("el alumno es requerido")
	}
	exercise, subject, subjectID, err := s.resolvePracticeScope(ctx, teacherID, role, tenantID, courseID, exerciseID)
	if err != nil {
		return err
	}
	if s.databases == nil {
		return ErrPracticeDatabasesUnavailable
	}
	databases, err := s.databases.ListStudentDatabasesByExercise(ctx, tenantID, subjectID, exercise.ID)
	if err != nil {
		return err
	}
	started := false
	for _, database := range databases {
		if database.StudentID == studentID {
			started = true
			break
		}
	}
	if !started {
		return ErrStudentPracticeNotStarted
	}
	template, err := s.resolveTemplate(ctx, exercise, subject)
	if err != nil {
		return err
	}
	return s.satellites.ResetStudentPracticeDatabase(ctx, studentID, subjectID, template, exercise)
}
