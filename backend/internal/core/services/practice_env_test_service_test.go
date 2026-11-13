package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

type practiceExerciseRepo struct {
	domain.ExerciseRepository
	exercise *domain.Exercise
	err      error
}

func (r *practiceExerciseRepo) GetByIDAndTenant(_ context.Context, _, _ string) (*domain.Exercise, error) {
	return r.exercise, r.err
}

type practiceSubjectRepo struct {
	domain.SubjectRepository
	subject *domain.Subject
	err     error
}

func (r *practiceSubjectRepo) GetByID(_ context.Context, _, _ string) (*domain.Subject, error) {
	return r.subject, r.err
}

type practiceTemplateRepo struct {
	domain.TemplateRepository
	template *domain.Template
	err      error
}

func (r *practiceTemplateRepo) GetTemplateByID(_ context.Context, _ string) (*domain.Template, error) {
	return r.template, r.err
}

type practiceRegistry struct {
	local   bool
	pullErr error
	pulled  bool
}

func (r *practiceRegistry) InspectLocal(context.Context, string) (bool, int64, string, error) {
	return r.local, 0, "", nil
}

func (r *practiceRegistry) InspectRemoteDigest(context.Context, string) (string, error) {
	return "", nil
}

func (r *practiceRegistry) PullImage(context.Context, string, func(int64, int64, int, int, string)) error {
	r.pulled = true
	return r.pullErr
}

type practiceSetupRunner struct {
	spec   domain.PracticeEnvironmentSetupTestSpec
	result domain.PracticeEnvironmentSetupTestResult
	err    error
	calls  int
}

func (r *practiceSetupRunner) RunPracticeEnvironmentSetupTest(_ context.Context, spec domain.PracticeEnvironmentSetupTestSpec) (domain.PracticeEnvironmentSetupTestResult, error) {
	r.spec = spec
	r.calls++
	return r.result, r.err
}

type practiceSatellites struct {
	provision services.PracticeSatelliteProvision
	err       error
	calls     int
	resets    int
	resetID   string
}

func (p *practiceSatellites) ProvisionPracticeSatellites(context.Context, *domain.Exercise, *domain.Template, string) (services.PracticeSatelliteProvision, error) {
	p.calls++
	return p.provision, p.err
}

func (p *practiceSatellites) ResetStudentPracticeDatabase(_ context.Context, studentID, _ string, _ *domain.Template, _ *domain.Exercise) error {
	p.resets++
	p.resetID = studentID
	return p.err
}

type practiceDatabases struct {
	items []domain.StudentPracticeDatabase
	err   error
}

func (p *practiceDatabases) ListStudentDatabasesByExercise(context.Context, string, string, string) ([]domain.StudentPracticeDatabase, error) {
	return p.items, p.err
}

func newPracticeTestService(t *testing.T, opts ...func(*practiceTestFixture)) (*services.PracticeEnvTestService, *practiceTestFixture) {
	t.Helper()

	teacherID := "teacher-1"
	fixture := &practiceTestFixture{
		exerciseRepo: &practiceExerciseRepo{exercise: &domain.Exercise{
			ID:              "aaaaaaaa-1111-4111-8111-111111111111",
			SubjectID:       stringPtr("subj-1"),
			TemplateID:      "tpl-1",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		}},
		subjectRepo: &practiceSubjectRepo{subject: &domain.Subject{
			ID:        "subj-1",
			TenantID:  domain.DefaultTenantID,
			TeacherID: &teacherID,
		}},
		templateRepo: &practiceTemplateRepo{template: &domain.Template{
			ID:          "tpl-1",
			DockerImage: "python:3.12-slim",
			BaseRamMB:   1024,
			SetupScript: "mkdir -p /home/workspace/src",
		}},
		registry: &practiceRegistry{local: true},
		runner: &practiceSetupRunner{result: domain.PracticeEnvironmentSetupTestResult{
			ExitCode:       0,
			DurationMs:     1200,
			RepoReady:      true,
			OpenVSCodeSeen: true,
			Output:         "SOLV_REPO:ready\nSOLV_VSCODE:present",
		}},
		satellites: &practiceSatellites{provision: services.PracticeSatelliteProvision{
			Env:         []string{"DATABASE_URL=postgresql://test"},
			NetworkName: "solv-satellites-net",
			Engines:     []string{"postgres"},
		}},
		databases: &practiceDatabases{items: []domain.StudentPracticeDatabase{
			{StudentID: "student-1", StudentName: "Alumna Uno", StudentEmail: "alumna@example.com", Status: "running"},
		}},
		repo: memory.NewEnvTestJobMemoryRepository(time.Hour),
	}
	for _, opt := range opts {
		opt(fixture)
	}

	svc := services.NewPracticeEnvTestService(
		fixture.repo,
		fixture.registry,
		fixture.runner,
		fixture.satellites,
		fixture.exerciseRepo,
		fixture.subjectRepo,
		fixture.templateRepo,
		fixture.databases,
		services.PracticeEnvTestConfig{
			MaxConcurrentJobs: 1,
			TotalJobTimeout:   5 * time.Second,
			SetupTimeout:      2 * time.Second,
		},
	)
	return svc, fixture
}

type practiceTestFixture struct {
	exerciseRepo *practiceExerciseRepo
	subjectRepo  *practiceSubjectRepo
	templateRepo *practiceTemplateRepo
	registry     *practiceRegistry
	runner       *practiceSetupRunner
	satellites   *practiceSatellites
	databases    *practiceDatabases
	repo         *memory.EnvTestJobMemoryRepository
}

func (f *practiceTestFixture) startRequest() services.PracticeEnvTestRequest {
	return services.PracticeEnvTestRequest{
		TeacherID:  "teacher-1",
		TenantID:   domain.DefaultTenantID,
		Role:       "teacher",
		CourseID:   "subj-1",
		ExerciseID: "aaaaaaaa-1111-4111-8111-111111111111",
	}
}

func TestPracticeEnvTest_SuccessValidatesRepoSatellitesAndEditor(t *testing.T) {
	svc, fixture := newPracticeTestService(t)

	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}

	finished := waitPracticeJob(t, fixture.repo, job.ID)
	if finished.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("la prueba debía terminar en success, fue %s (%s)", finished.Status, finished.ErrorMessage)
	}
	if finished.ExerciseID != "aaaaaaaa-1111-4111-8111-111111111111" || finished.SubjectID != "subj-1" {
		t.Fatalf("el job no quedó ligado a la práctica: %+v", finished)
	}
	if fixture.runner.calls != 1 {
		t.Fatalf("el setup debía ejecutarse una vez, se ejecutó %d", fixture.runner.calls)
	}
	if fixture.runner.spec.SetupScript != "mkdir -p /home/workspace/src" {
		t.Fatalf("el setup de la plantilla no llegó al runner: %q", fixture.runner.spec.SetupScript)
	}
	if fixture.runner.spec.NetworkName != "solv-satellites-net" {
		t.Fatalf("la red satélite no llegó al runner: %q", fixture.runner.spec.NetworkName)
	}
	if len(fixture.runner.spec.Env) != 1 || !strings.HasPrefix(fixture.runner.spec.Env[0], "DATABASE_URL=") {
		t.Fatalf("las credenciales satélite no llegaron al runner: %+v", fixture.runner.spec.Env)
	}
	if !strings.HasPrefix(fixture.runner.spec.VolumeName, "solv_envtest_") {
		t.Fatalf("el volumen temporal no es propio del job: %q", fixture.runner.spec.VolumeName)
	}
	if finished.Result == nil || len(finished.Result.Tools) != 3 {
		t.Fatalf("el reporte debía listar repo, editor y satélite: %+v", finished.Result)
	}
	if !finished.Result.Tools[0].Present || !finished.Result.Tools[1].Present || !finished.Result.Tools[2].Present {
		t.Fatalf("las tres comprobaciones debían estar presentes: %+v", finished.Result.Tools)
	}
}

func TestPracticeEnvTest_WithoutSatellitesUsesBridgeAndNoSatelliteTools(t *testing.T) {
	svc, fixture := newPracticeTestService(t, func(f *practiceTestFixture) {
		f.satellites.provision = services.PracticeSatelliteProvision{}
	})

	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}
	finished := waitPracticeJob(t, fixture.repo, job.ID)

	if finished.Status != domain.EnvTestStatusSuccess {
		t.Fatalf("la prueba sin satélites debía pasar, fue %s", finished.Status)
	}
	if fixture.runner.spec.NetworkName != "bridge" {
		t.Fatalf("sin satélites el setup debía usar bridge, usó %q", fixture.runner.spec.NetworkName)
	}
	if finished.Result == nil || len(finished.Result.Tools) != 2 {
		t.Fatalf("sin satélites solo se reportan repo y editor: %+v", finished.Result)
	}
}

func TestPracticeEnvTest_SetupFailureIsReportedWithoutSubmissions(t *testing.T) {
	svc, fixture := newPracticeTestService(t, func(f *practiceTestFixture) {
		f.runner.result = domain.PracticeEnvironmentSetupTestResult{
			ExitCode: 2,
			Output:   "pip: command not found",
		}
	})

	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}
	finished := waitPracticeJob(t, fixture.repo, job.ID)

	if finished.Status != domain.EnvTestStatusFailed {
		t.Fatalf("el setup roto debía fallar el job, fue %s", finished.Status)
	}
	if finished.ErrorCode != domain.EnvTestErrSetupFailed {
		t.Fatalf("código de error inesperado: %q", finished.ErrorCode)
	}
	if !strings.Contains(finished.ErrorMessage, "pip: command not found") {
		t.Fatalf("el mensaje debía conservar la salida del setup: %q", finished.ErrorMessage)
	}
}

func TestPracticeEnvTest_UnwritableRepositoryFails(t *testing.T) {
	svc, fixture := newPracticeTestService(t, func(f *practiceTestFixture) {
		f.runner.result = domain.PracticeEnvironmentSetupTestResult{
			ExitCode:       0,
			OpenVSCodeSeen: true,
			Output:         "SOLV_REPO:unwritable\nSOLV_VSCODE:present",
		}
	})

	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}
	finished := waitPracticeJob(t, fixture.repo, job.ID)

	if finished.Status != domain.EnvTestStatusFailed {
		t.Fatalf("un repositorio no escribible debía fallar el job, fue %s", finished.Status)
	}
	if finished.ErrorCode != domain.EnvTestErrSetupFailed {
		t.Fatalf("código de error inesperado: %q", finished.ErrorCode)
	}
	if !strings.Contains(finished.ErrorMessage, "/home/workspace") {
		t.Fatalf("el mensaje debía señalar el volumen no escribible: %q", finished.ErrorMessage)
	}
	if finished.Result == nil || len(finished.Result.Tools) == 0 || finished.Result.Tools[0].Present {
		t.Fatalf("el reporte no debía afirmar un repositorio listo: %+v", finished.Result)
	}
}

func TestPracticeEnvTest_NonIDEIsRejected(t *testing.T) {
	svc, fixture := newPracticeTestService(t, func(f *practiceTestFixture) {
		f.exerciseRepo.exercise.EnvironmentType = string(domain.EnvironmentTypeJuezEfimero)
	})

	_, err := svc.Start(context.Background(), fixture.startRequest())
	if !errors.Is(err, services.ErrPracticeTestRequiresIDE) {
		t.Fatalf("se esperaba ErrPracticeTestRequiresIDE, fue: %v", err)
	}
	if fixture.runner.calls != 0 || fixture.satellites.calls != 0 {
		t.Fatal("una práctica de juez no debía aprovisionar ni ejecutar setup")
	}
}

func TestPracticeEnvTest_ForeignTeacherIsRejected(t *testing.T) {
	svc, fixture := newPracticeTestService(t)
	req := fixture.startRequest()
	req.TeacherID = "teacher-2"

	_, err := svc.Start(context.Background(), req)
	if !errors.Is(err, services.ErrPracticeTestForbidden) {
		t.Fatalf("se esperaba ErrPracticeTestForbidden, fue: %v", err)
	}
}

func TestPracticeEnvTest_SatelliteFailureIsTerminal(t *testing.T) {
	svc, fixture := newPracticeTestService(t, func(f *practiceTestFixture) {
		f.satellites.err = errors.New("postgres no responde")
	})

	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}
	finished := waitPracticeJob(t, fixture.repo, job.ID)

	if finished.Status != domain.EnvTestStatusFailed {
		t.Fatalf("sin satélites la prueba debía fallar, fue %s", finished.Status)
	}
	if finished.ErrorCode != domain.EnvTestErrSatelliteFailed {
		t.Fatalf("código de error inesperado: %q", finished.ErrorCode)
	}
	if fixture.runner.calls != 0 {
		t.Fatal("sin satélites no se debía ejecutar el setup")
	}
}

func TestPracticeEnvTest_PullFailureSkipsSetup(t *testing.T) {
	svc, fixture := newPracticeTestService(t, func(f *practiceTestFixture) {
		f.registry.local = false
		f.registry.pullErr = errors.New("manifest unknown: not found")
	})

	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}
	finished := waitPracticeJob(t, fixture.repo, job.ID)

	if finished.Status != domain.EnvTestStatusFailed {
		t.Fatalf("sin imagen la prueba debía fallar, fue %s", finished.Status)
	}
	if finished.ErrorCode != domain.EnvTestErrNotFoundRemote {
		t.Fatalf("código de error inesperado: %q", finished.ErrorCode)
	}
	if fixture.runner.calls != 0 {
		t.Fatal("sin imagen no se debía ejecutar el setup")
	}
}

func TestPracticeEnvTest_LatestIsScopedToTheTeacher(t *testing.T) {
	svc, fixture := newPracticeTestService(t)
	job, err := svc.Start(context.Background(), fixture.startRequest())
	if err != nil {
		t.Fatalf("inicio inesperado: %v", err)
	}
	waitPracticeJob(t, fixture.repo, job.ID)

	latest, err := svc.Latest(context.Background(), "teacher-1", domain.DefaultTenantID, "teacher", job.ExerciseID)
	if err != nil {
		t.Fatalf("consulta inesperada: %v", err)
	}
	if latest == nil || latest.ID != job.ID {
		t.Fatalf("el docente debía ver su última prueba: %+v", latest)
	}

	foreign, err := svc.Latest(context.Background(), "teacher-2", domain.DefaultTenantID, "teacher", job.ExerciseID)
	if err != nil {
		t.Fatalf("consulta inesperada: %v", err)
	}
	if foreign != nil {
		t.Fatalf("otro docente no debía ver la prueba: %+v", foreign)
	}

	none, err := svc.Latest(context.Background(), "teacher-1", domain.DefaultTenantID, "teacher", "otra-practica")
	if err != nil {
		t.Fatalf("consulta inesperada: %v", err)
	}
	if none != nil {
		t.Fatalf("sin pruebas debía devolver nil: %+v", none)
	}
}

func waitPracticeJob(t *testing.T, repo *memory.EnvTestJobMemoryRepository, id string) *domain.EnvTestJob {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, err := repo.GetByID(context.Background(), id)
		if err == nil && job.IsTerminal() {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("el job %s no terminó a tiempo", id)
	return nil
}
