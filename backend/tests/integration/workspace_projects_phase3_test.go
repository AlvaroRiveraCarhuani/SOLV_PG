package integration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/postgres"
)

type phase3Volume struct {
	domain.PracticeEnvironmentOrchestrator
	mu         sync.Mutex
	files      map[string]string
	prepared   []domain.WorkspaceSeed
	volumes    []string
	prepareErr error
}

func (v *phase3Volume) EnsureVolumeExists(context.Context, string) error { return nil }
func (v *phase3Volume) EnsureICCDisabledNetworkExists(context.Context, string) error {
	return nil
}
func (v *phase3Volume) StartPracticeEnvironmentContainer(context.Context, domain.PracticeEnvironmentContainerConfig) (string, error) {
	return "phase3-container", nil
}
func (v *phase3Volume) ListAllManagedContainers(context.Context) ([]string, error) {
	return []string{}, nil
}
func (v *phase3Volume) PrepareWorkspaceVolume(_ context.Context, volumeName string, seed domain.WorkspaceSeed) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.prepared = append(v.prepared, seed)
	v.volumes = append(v.volumes, volumeName)
	if v.prepareErr != nil {
		return v.prepareErr
	}
	if v.files == nil {
		v.files = make(map[string]string)
	}
	for _, file := range seed.Files {
		v.files[file.Path] = file.Content
	}
	return nil
}
func (v *phase3Volume) SnapshotPracticeEnvironmentVolume(context.Context, string, int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	files := make([]domain.PracticeEnvironmentVolumeFile, 0, len(v.files))
	for path, content := range v.files {
		files = append(files, domain.PracticeEnvironmentVolumeFile{Path: path, Content: []byte(content)})
	}
	return files, nil
}
func (v *phase3Volume) WritePracticeEnvironmentVolumeFiles(context.Context, string, map[string]string) error {
	return nil
}

func phase3Context(tenantID string) context.Context {
	return context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
}

func insertTestExercise(t *testing.T, sqlDB *sqlx.DB, tenantID, subjectID, templateID, environmentType, title string) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := sqlDB.Exec(`
		INSERT INTO exercises (id, tenant_id, subject_id, title, type, status, environment_type, language, boilerplate, template_id)
		VALUES ($1, $2, $3, $4, 'algorithm', 'draft', $5, 'python', '', $6)
	`, id, tenantID, subjectID, title, environmentType, templateID); err != nil {
		t.Fatalf("crear ejercicio %s: %v", environmentType, err)
	}
	return id
}

func TestWorkspaceProjectPhase3AssignmentAndV1V2LiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)
	ctx := phase3Context(seed.tenantID)

	workspaceRepo := postgres.NewPostgresWorkspaceProjectRepository(sqlDB)
	exerciseRepo := postgres.NewPostgresExerciseRepository(sqlDB)
	subjectRepo := postgres.NewPostgresSubjectRepository(sqlDB)
	templateRepo := postgres.NewPostgresTemplateRepository(sqlDB)
	submissionRepo := postgres.NewPostgresSubmissionRepository(sqlDB)
	volume := &phase3Volume{}

	envService := services.NewPracticeEnvironmentService(
		postgres.NewPostgresPracticeEnvironmentRepository(sqlDB),
		volume,
		&mockSlice12HostMonitor{},
	).
		WithProvisioning(subjectRepo, templateRepo).
		WithExerciseSubmission(exerciseRepo, submissionRepo).
		SetWorkspaceProjects(workspaceRepo)

	project, err := workspaceRepo.CreateProject(ctx, &domain.WorkspaceProject{
		TenantID:   seed.tenantID,
		ExerciseID: seed.exerciseA,
		CreatedBy:  &seed.teacherID,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	v1Text := "# === FILE: src/main.py ===\nprint('v1')\n"
	if _, err := workspaceRepo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID: seed.tenantID, ProjectID: project.ID, TextSnapshot: v1Text, UpdatedBy: &seed.teacherID,
	}, 0); err != nil {
		t.Fatalf("borrador V1: %v", err)
	}
	v1, err := workspaceRepo.Publish(ctx, seed.tenantID, project.ID, seed.teacherID)
	if err != nil {
		t.Fatalf("publicar V1: %v", err)
	}

	first, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, seed.exerciseA)
	if err != nil {
		t.Fatalf("abrir entorno del primer alumno: %v", err)
	}
	if first.WorkspaceProjectVersionID == nil || *first.WorkspaceProjectVersionID != v1.ID {
		t.Fatalf("el primer entorno recibió %v, quería V1 %s", first.WorkspaceProjectVersionID, v1.ID)
	}
	if first.WorkspaceSeedMode != domain.WorkspaceSeedModeProject || first.WorkspaceInitStatus != domain.WorkspaceInitStatusSetupDone {
		t.Fatalf("estado de siembra inesperado: %s/%s", first.WorkspaceSeedMode, first.WorkspaceInitStatus)
	}
	if len(volume.prepared) != 1 || volume.files["src/main.py"] != "print('v1')\n" {
		t.Fatalf("el volumen no recibió V1: prepared=%d files=%+v", len(volume.prepared), volume.files)
	}

	v2Text := "# === FILE: src/main.py ===\nprint('v2')\n"
	if _, err := workspaceRepo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID: seed.tenantID, ProjectID: project.ID, TextSnapshot: v2Text, UpdatedBy: &seed.teacherID,
	}, 1); err != nil {
		t.Fatalf("borrador V2: %v", err)
	}
	v2, err := workspaceRepo.Publish(ctx, seed.tenantID, project.ID, seed.teacherID)
	if err != nil {
		t.Fatalf("publicar V2: %v", err)
	}

	second, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentB, seed.subjectID, seed.exerciseA)
	if err != nil {
		t.Fatalf("abrir entorno del segundo alumno: %v", err)
	}
	if second.WorkspaceProjectVersionID == nil || *second.WorkspaceProjectVersionID != v2.ID {
		t.Fatalf("el entorno nuevo recibió %v, quería V2 %s", second.WorkspaceProjectVersionID, v2.ID)
	}
	if len(volume.prepared) != 2 {
		t.Fatalf("el segundo entorno debía sembrar una vez: prepared=%d", len(volume.prepared))
	}
	if len(volume.volumes) != 2 || volume.volumes[0] == volume.volumes[1] {
		t.Fatalf("cada alumno debía recibir un volumen propio: %v", volume.volumes)
	}

	reopened, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, seed.exerciseA)
	if err != nil {
		t.Fatalf("reabrir entorno del primer alumno: %v", err)
	}
	if reopened.WorkspaceProjectVersionID == nil || *reopened.WorkspaceProjectVersionID != v1.ID {
		t.Fatalf("reabrir movió la versión del primer entorno: %v", reopened.WorkspaceProjectVersionID)
	}
	if len(volume.prepared) != 2 {
		t.Fatalf("reabrir un entorno verificado no debía sembrar: prepared=%d", len(volume.prepared))
	}

	volume.mu.Lock()
	volume.files["src/main.py"] = "print('trabajo del alumno')\n"
	volume.mu.Unlock()
	submission, err := envService.SubmitPracticeEnvironmentProject(ctx, first.ID, seed.studentA, seed.exerciseA, nil)
	if err != nil {
		t.Fatalf("entrega del primer alumno: %v", err)
	}
	if submission.BaseStatus != domain.BaseStatusVersion || submission.WorkspaceProjectVersionID == nil || *submission.WorkspaceProjectVersionID != v1.ID {
		t.Fatalf("la entrega no quedó vinculada a V1: base=%q version=%v", submission.BaseStatus, submission.WorkspaceProjectVersionID)
	}
	if !strings.Contains(submission.Code, "trabajo del alumno") {
		t.Fatalf("la entrega no conserva el trabajo del alumno: %s", submission.Code)
	}

	// Un entorno existente no se puede reasignar a V2 desde la publicación.
	var storedVersion string
	if err := sqlDB.Get(&storedVersion, `SELECT workspace_project_version_id FROM environments WHERE id = $1`, first.ID); err != nil {
		t.Fatalf("leer versión persistida: %v", err)
	}
	if storedVersion != v1.ID {
		t.Fatalf("la versión persistida cambió a %s", storedVersion)
	}
}

func TestWorkspaceProjectPhase3NoProjectLegacyAndJudgeLiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)
	ctx := phase3Context(seed.tenantID)
	volume := &phase3Volume{}
	envRepo := postgres.NewPostgresPracticeEnvironmentRepository(sqlDB)
	envService := services.NewPracticeEnvironmentService(envRepo, volume, &mockSlice12HostMonitor{}).
		WithProvisioning(
			postgres.NewPostgresSubjectRepository(sqlDB),
			postgres.NewPostgresTemplateRepository(sqlDB),
		).
		WithExerciseSubmission(
			postgres.NewPostgresExerciseRepository(sqlDB),
			postgres.NewPostgresSubmissionRepository(sqlDB),
		).
		SetWorkspaceProjects(postgres.NewPostgresWorkspaceProjectRepository(sqlDB))

	withoutProject, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, seed.exerciseA)
	if err != nil {
		t.Fatalf("entorno sin proyecto: %v", err)
	}
	if withoutProject.WorkspaceSeedMode != domain.WorkspaceSeedModeNone || withoutProject.WorkspaceInitStatus != domain.WorkspaceInitStatusNotRequired || len(volume.prepared) != 0 {
		t.Fatalf("entorno sin proyecto inesperado: %+v prepared=%d", withoutProject, len(volume.prepared))
	}

	otherTenant := uuid.NewString()
	if _, err := sqlDB.Exec(`INSERT INTO tenants (id, name, slug) VALUES ($1, 'F3 Other Tenant', $2)`, otherTenant, "f3-"+otherTenant[:8]); err != nil {
		t.Fatalf("crear tenant aislado: %v", err)
	}
	if _, err := envService.StartPracticeEnvironmentForExercise(phase3Context(otherTenant), seed.studentA, seed.subjectID, seed.exerciseA); err == nil {
		t.Fatal("un tenant distinto no debía abrir el ejercicio de otro tenant")
	}
	if len(volume.prepared) != 0 {
		t.Fatalf("el intento entre tenants no debía sembrar: %d", len(volume.prepared))
	}

	legacy := &domain.PracticeEnvironment{
		ID: uuid.NewString(), TenantID: seed.tenantID, StudentID: seed.studentB, SubjectID: seed.subjectID,
		ExerciseID: &seed.exerciseB, Status: domain.PracticeEnvironmentStatusHibernated,
		WorkspaceSeedMode: domain.WorkspaceSeedModeLegacy, WorkspaceInitStatus: domain.WorkspaceInitStatusLegacy,
	}
	if err := envRepo.Create(ctx, legacy); err != nil {
		t.Fatalf("crear legacy: %v", err)
	}
	legacyStarted, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentB, seed.subjectID, seed.exerciseB)
	if err != nil {
		t.Fatalf("reabrir legacy: %v", err)
	}
	if legacyStarted.WorkspaceSeedMode != domain.WorkspaceSeedModeLegacy || legacyStarted.WorkspaceInitStatus != domain.WorkspaceInitStatusLegacy || len(volume.prepared) != 0 {
		t.Fatalf("legacy fue reinterpretado: %+v prepared=%d", legacyStarted, len(volume.prepared))
	}

	templateID := ""
	if err := sqlDB.Get(&templateID, `SELECT id FROM lab_templates WHERE tenant_id = $1 LIMIT 1`, seed.tenantID); err != nil {
		t.Fatalf("leer plantilla: %v", err)
	}
	judgeID := insertTestExercise(t, sqlDB, seed.tenantID, seed.subjectID, templateID, "JUEZ_EFIMERO", "phase3-judge")
	judge, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, judgeID)
	if err != nil {
		t.Fatalf("entorno del juez: %v", err)
	}
	if judge.WorkspaceSeedMode != domain.WorkspaceSeedModeNone || len(volume.prepared) != 0 {
		t.Fatalf("el juez no debe sembrar un proyecto: %+v", judge)
	}
}

func TestWorkspaceProjectPhase3QuarantineSurvivesReopenLiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)
	ctx := phase3Context(seed.tenantID)
	workspaceRepo := postgres.NewPostgresWorkspaceProjectRepository(sqlDB)
	project, err := workspaceRepo.CreateProject(ctx, &domain.WorkspaceProject{TenantID: seed.tenantID, ExerciseID: seed.exerciseA, CreatedBy: &seed.teacherID})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	if _, err := workspaceRepo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{TenantID: seed.tenantID, ProjectID: project.ID, TextSnapshot: "# === FILE: main.py ===\nprint(1)\n"}, 0); err != nil {
		t.Fatalf("borrador: %v", err)
	}
	if _, err := workspaceRepo.Publish(ctx, seed.tenantID, project.ID, seed.teacherID); err != nil {
		t.Fatalf("publicar: %v", err)
	}
	volume := &phase3Volume{prepareErr: domain.ErrWorkspaceSeedVolumeAmbiguous}
	envService := services.NewPracticeEnvironmentService(
		postgres.NewPostgresPracticeEnvironmentRepository(sqlDB), volume, &mockSlice12HostMonitor{},
	).WithProvisioning(
		postgres.NewPostgresSubjectRepository(sqlDB), postgres.NewPostgresTemplateRepository(sqlDB),
	).WithExerciseSubmission(
		postgres.NewPostgresExerciseRepository(sqlDB), postgres.NewPostgresSubmissionRepository(sqlDB),
	).SetWorkspaceProjects(workspaceRepo)

	if _, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, seed.exerciseA); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("esperado error de cuarentena, obtuvo %v", err)
	}
	var initStatus string
	if err := sqlDB.Get(&initStatus, `SELECT workspace_init_status FROM environments WHERE student_id = $1 AND exercise_id = $2`, seed.studentA, seed.exerciseA); err != nil {
		t.Fatalf("leer estado: %v", err)
	}
	if initStatus != domain.WorkspaceInitStatusQuarantined {
		t.Fatalf("estado de inicialización = %q", initStatus)
	}
	if _, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, seed.exerciseA); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("reabrir cuarentena debía rechazarse, obtuvo %v", err)
	}
	if len(volume.prepared) != 1 {
		t.Fatalf("la cuarentena no debía volver a sembrar: %d", len(volume.prepared))
	}
}
