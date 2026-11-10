package services_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type workspaceSeedProjectRepo struct {
	domain.WorkspaceProjectRepository
	mu       sync.Mutex
	project  *domain.WorkspaceProject
	versions map[string]*domain.WorkspaceProjectVersion
}

func (r *workspaceSeedProjectRepo) GetProjectByExercise(context.Context, string, string) (*domain.WorkspaceProject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.project == nil {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	copy := *r.project
	return &copy, nil
}

func (r *workspaceSeedProjectRepo) GetVersion(_ context.Context, tenantID, versionID string) (*domain.WorkspaceProjectVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	version, ok := r.versions[versionID]
	if !ok || version.TenantID != tenantID {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	copy := *version
	return &copy, nil
}

func (r *workspaceSeedProjectRepo) setPublished(version *domain.WorkspaceProjectVersion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.versions == nil {
		r.versions = make(map[string]*domain.WorkspaceProjectVersion)
	}
	r.versions[version.ID] = version
	if r.project == nil {
		r.project = &domain.WorkspaceProject{ID: version.ProjectID, TenantID: version.TenantID, ExerciseID: version.ExerciseID}
	}
	r.project.CurrentPublishedVersionID = &version.ID
}

type workspaceSeedOrchestrator struct {
	*mockOrchestrator
	mu          sync.Mutex
	prepareErr  error
	prepareCall int
	verifyErr   error
	verifyCall  int
	seeds       []domain.WorkspaceSeed
}

func (o *workspaceSeedOrchestrator) PrepareWorkspaceVolume(_ context.Context, _ string, seed domain.WorkspaceSeed) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.prepareCall++
	o.seeds = append(o.seeds, seed)
	return o.prepareErr
}

func (o *workspaceSeedOrchestrator) VerifyWorkspaceVolume(_ context.Context, _ string, _ domain.WorkspaceSeed) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.verifyCall++
	return o.verifyErr
}

func (m *mockPracticeEnvironmentRepo) UpdateWorkspaceInitStatus(_ context.Context, id, status string) error {
	if ws, ok := m.environments[id]; ok {
		ws.WorkspaceInitStatus = status
	}
	return nil
}

func seedVersionForTest(t *testing.T, tenantID, projectID, exerciseID, versionID, snapshot string) *domain.WorkspaceProjectVersion {
	t.Helper()
	files, ok := domain.ParseProjectSnapshot(snapshot, "")
	if !ok || len(files) == 0 {
		t.Fatalf("snapshot de prueba inválido")
	}
	sum := sha256.Sum256([]byte(snapshot))
	total := 0
	for _, file := range files {
		total += len(file.Content)
	}
	return &domain.WorkspaceProjectVersion{
		ID:            versionID,
		TenantID:      tenantID,
		ProjectID:     projectID,
		ExerciseID:    exerciseID,
		VersionNumber: 1,
		TextSnapshot:  snapshot,
		ContentSHA256: hex.EncodeToString(sum[:]),
		FileCount:     len(files),
		TotalBytes:    total,
		PublishedAt:   time.Now(),
	}
}

func newWorkspaceSeedService(t *testing.T, projectRepo *workspaceSeedProjectRepo, orch *workspaceSeedOrchestrator) (*services.PracticeEnvironmentService, *mockPracticeEnvironmentRepo) {
	t.Helper()
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		teacherID  = "teacher-seed"
	)
	subject := &domain.Subject{ID: subjectID, Name: "Seed", TeacherID: ptrString(teacherID)}
	exercise := &domain.Exercise{
		ID:              exerciseID,
		TenantID:        tenantID,
		SubjectID:       ptrString(subjectID),
		Title:           "Proyecto",
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
	}
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{exerciseID: exercise}}
	repo := newMockPracticeEnvironmentRepo()
	service := services.NewPracticeEnvironmentService(repo, orch, &mockHostMonitor{}).
		WithProvisioning(&mockSubjectRepo{subjects: map[string]*domain.Subject{subjectID: subject}}, &mockTemplateRepo{templates: map[string]*domain.Template{}}).
		WithExerciseSubmission(exerciseRepo, &fakeSubmissionRepo{}).
		SetWorkspaceProjects(projectRepo)
	return service, repo
}

func ptrString(value string) *string { return &value }

func withTenantContext(tenantID string) context.Context {
	return context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
}

func TestWorkspaceSeed_NewEnvironmentAssignsExactPublishedVersion(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	snapshot := "# === FILE: src/main.py ===\nprint('v1')\n\n# === FILE: README.md ===\n# V1\n"
	version := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", snapshot)
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(version)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, _ := newWorkspaceSeedService(t, projectRepo, orch)

	ctx := withTenantContext(tenantID)
	instance, err := service.StartPracticeEnvironmentForExercise(ctx, "student-a", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("inicio del entorno: %v", err)
	}
	if instance.WorkspaceProjectVersionID == nil || *instance.WorkspaceProjectVersionID != version.ID {
		t.Fatalf("versión asignada = %v, quería %s", instance.WorkspaceProjectVersionID, version.ID)
	}
	if instance.WorkspaceSeedMode != domain.WorkspaceSeedModeProject || instance.WorkspaceInitStatus != domain.WorkspaceInitStatusSetupDone {
		t.Fatalf("estado de siembra inesperado: %s/%s", instance.WorkspaceSeedMode, instance.WorkspaceInitStatus)
	}
	if orch.prepareCall != 1 || len(orch.seeds) != 1 {
		t.Fatalf("esperado un sembrado, obtenido %d", orch.prepareCall)
	}
	if got := orch.seeds[0].VersionID; got != version.ID {
		t.Fatalf("el sembrado usó la versión %q", got)
	}
	if len(orch.seeds[0].Files) != 2 || orch.seeds[0].Files[0].Path != "README.md" || orch.seeds[0].Files[1].Path != "src/main.py" {
		t.Fatalf("archivos sembrados inesperados: %+v", orch.seeds[0].Files)
	}
	if !strings.Contains(orch.seeds[0].Files[1].Content, "v1") {
		t.Fatalf("el contenido sembrado no es V1: %q", orch.seeds[0].Files[1].Content)
	}
}

func TestWorkspaceSeed_NewEnvironmentWithoutPublishedProjectUsesNone(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
	)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, _ := newWorkspaceSeedService(t, &workspaceSeedProjectRepo{}, orch)

	instance, err := service.StartPracticeEnvironmentForExercise(withTenantContext(tenantID), "student-a", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("inicio del entorno sin proyecto: %v", err)
	}
	if instance.WorkspaceSeedMode != domain.WorkspaceSeedModeNone || instance.WorkspaceInitStatus != domain.WorkspaceInitStatusNotRequired {
		t.Fatalf("estado sin proyecto inesperado: %s/%s", instance.WorkspaceSeedMode, instance.WorkspaceInitStatus)
	}
	if instance.WorkspaceProjectVersionID != nil || orch.prepareCall != 0 {
		t.Fatalf("un entorno sin proyecto no debe sembrar: version=%v calls=%d", instance.WorkspaceProjectVersionID, orch.prepareCall)
	}
}

func TestWorkspaceSeed_LegacyEnvironmentIsNeverReseeded(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
	)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, repo := newWorkspaceSeedService(t, &workspaceSeedProjectRepo{}, orch)
	legacy := &domain.PracticeEnvironment{
		ID:                  "env-legacy",
		TenantID:            tenantID,
		StudentID:           "student-a",
		SubjectID:           subjectID,
		ExerciseID:          ptrString(exerciseID),
		Status:              domain.PracticeEnvironmentStatusHibernated,
		WorkspaceSeedMode:   domain.WorkspaceSeedModeLegacy,
		WorkspaceInitStatus: domain.WorkspaceInitStatusLegacy,
	}
	if err := repo.Create(withTenantContext(tenantID), legacy); err != nil {
		t.Fatalf("preparar legacy: %v", err)
	}

	instance, err := service.StartPracticeEnvironmentForExercise(withTenantContext(tenantID), "student-a", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("reabrir legacy: %v", err)
	}
	if instance.WorkspaceSeedMode != domain.WorkspaceSeedModeLegacy || instance.WorkspaceInitStatus != domain.WorkspaceInitStatusLegacy {
		t.Fatalf("legacy fue reinterpretado: %+v", instance)
	}
	if orch.prepareCall != 0 {
		t.Fatalf("un legacy no debe sembrarse, calls=%d", orch.prepareCall)
	}
}

func TestWorkspaceSeed_VerifiedEnvironmentReopensWithoutCallingPublishedVersion(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	version := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", "# === FILE: main.py ===\nprint(1)\n")
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(version)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, repo := newWorkspaceSeedService(t, projectRepo, orch)
	instance := &domain.PracticeEnvironment{
		ID:                        "env-verified",
		TenantID:                  tenantID,
		StudentID:                 "student-a",
		SubjectID:                 subjectID,
		ExerciseID:                ptrString(exerciseID),
		Status:                    domain.PracticeEnvironmentStatusHibernated,
		WorkspaceSeedMode:         domain.WorkspaceSeedModeProject,
		WorkspaceInitStatus:       domain.WorkspaceInitStatusSetupDone,
		WorkspaceProjectVersionID: &version.ID,
	}
	if err := repo.Create(withTenantContext(tenantID), instance); err != nil {
		t.Fatalf("preparar verificado: %v", err)
	}
	// La publicación actual cambia, pero el entorno ya tiene su versión.
	projectRepo.setPublished(seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v2", "# === FILE: main.py ===\nprint(2)\n"))

	_, err := service.StartPracticeEnvironmentForExercise(withTenantContext(tenantID), "student-a", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("reabrir verificado: %v", err)
	}
	if orch.prepareCall != 0 {
		t.Fatalf("un entorno verificado no debe volver a sembrar, calls=%d", orch.prepareCall)
	}
	if orch.verifyCall != 1 {
		t.Fatalf("un entorno verificado debe comprobar su marcador, calls=%d", orch.verifyCall)
	}
}

func TestWorkspaceSeed_SetupDoneWithLostMarkerIsQuarantined(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	version := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", "# === FILE: main.py ===\nprint(1)\n")
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(version)
	orch := &workspaceSeedOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		verifyErr:        domain.ErrWorkspaceSeedVolumeAmbiguous,
	}
	service, repo := newWorkspaceSeedService(t, projectRepo, orch)
	instance := &domain.PracticeEnvironment{
		ID:                        "env-verified",
		TenantID:                  tenantID,
		StudentID:                 "student-a",
		SubjectID:                 subjectID,
		ExerciseID:                ptrString(exerciseID),
		Status:                    domain.PracticeEnvironmentStatusHibernated,
		WorkspaceSeedMode:         domain.WorkspaceSeedModeProject,
		WorkspaceInitStatus:       domain.WorkspaceInitStatusSetupDone,
		WorkspaceProjectVersionID: &version.ID,
	}
	if err := repo.Create(withTenantContext(tenantID), instance); err != nil {
		t.Fatalf("preparar verificado: %v", err)
	}

	if _, err := service.StartPracticeEnvironmentForExercise(withTenantContext(tenantID), "student-a", subjectID, exerciseID); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("un marcador perdido debía requerir revisión, obtuvo %v", err)
	}
	if instance.WorkspaceInitStatus != domain.WorkspaceInitStatusQuarantined {
		t.Fatalf("estado inesperado: %s", instance.WorkspaceInitStatus)
	}
	if orch.prepareCall != 0 {
		t.Fatalf("no debía re-sembrar, calls=%d", orch.prepareCall)
	}
}

func TestWorkspaceSeed_MissingStatusIsQuarantinedWithoutSeeding(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	version := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", "# === FILE: main.py ===\nprint(1)\n")
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(version)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, repo := newWorkspaceSeedService(t, projectRepo, orch)
	instance := &domain.PracticeEnvironment{
		ID:                        "env-missing",
		TenantID:                  tenantID,
		StudentID:                 "student-a",
		SubjectID:                 subjectID,
		ExerciseID:                ptrString(exerciseID),
		Status:                    domain.PracticeEnvironmentStatusHibernated,
		WorkspaceSeedMode:         domain.WorkspaceSeedModeProject,
		WorkspaceInitStatus:       domain.WorkspaceInitStatusMissing,
		WorkspaceProjectVersionID: &version.ID,
	}
	if err := repo.Create(withTenantContext(tenantID), instance); err != nil {
		t.Fatalf("preparar missing: %v", err)
	}

	if _, err := service.StartPracticeEnvironmentForExercise(withTenantContext(tenantID), "student-a", subjectID, exerciseID); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("un volumen missing debía requerir revisión, obtuvo %v", err)
	}
	if instance.WorkspaceInitStatus != domain.WorkspaceInitStatusQuarantined || orch.prepareCall != 0 {
		t.Fatalf("missing no debía sembrar: status=%s calls=%d", instance.WorkspaceInitStatus, orch.prepareCall)
	}
}

func TestWorkspaceSeed_QuarantinedEnvironmentIsNotOverwritten(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	version := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", "# === FILE: main.py ===\nprint(1)\n")
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(version)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}, prepareErr: domain.ErrWorkspaceSeedVolumeAmbiguous}
	service, repo := newWorkspaceSeedService(t, projectRepo, orch)

	ctx := withTenantContext(tenantID)
	if _, err := service.StartPracticeEnvironmentForExercise(ctx, "student-a", subjectID, exerciseID); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("el error de ambigüedad debía propagarse, obtuvo %v", err)
	}
	env, ok := repo.environments["student-a:"+subjectID+":"+exerciseID]
	if !ok {
		t.Fatal("el entorno fallido debía conservarse")
	}
	if env.WorkspaceInitStatus != domain.WorkspaceInitStatusQuarantined || env.Status != domain.PracticeEnvironmentStatusFailed {
		t.Fatalf("estado inesperado: %s/%s", env.WorkspaceInitStatus, env.Status)
	}

	if _, err := service.StartPracticeEnvironmentForExercise(ctx, "student-a", subjectID, exerciseID); !errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
		t.Fatalf("un entorno en cuarentena no debe reabrirse, obtuvo %v", err)
	}
	if orch.prepareCall != 1 {
		t.Fatalf("un entorno en cuarentena no debe volver a preparar el volumen, calls=%d", orch.prepareCall)
	}
}

func TestWorkspaceSeed_ConcurrentOpensSeedOnce(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	version := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", "# === FILE: main.go ===\npackage main\n")
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(version)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, _ := newWorkspaceSeedService(t, projectRepo, orch)

	ctx := withTenantContext(tenantID)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.StartPracticeEnvironmentForExercise(ctx, "student-a", subjectID, exerciseID)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("apertura concurrente: %v", err)
		}
	}
	if orch.prepareCall != 1 {
		t.Fatalf("dos aperturas concurrentes debían sembrar una vez, calls=%d", orch.prepareCall)
	}
}

func TestWorkspaceSeed_NewStudentAfterV2ReceivesV2WhileExistingKeepsV1(t *testing.T) {
	const (
		tenantID   = "tenant-seed"
		subjectID  = "subject-seed"
		exerciseID = "exercise-seed"
		projectID  = "project-seed"
	)
	v1 := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v1", "# === FILE: main.py ===\nprint('v1')\n")
	v2 := seedVersionForTest(t, tenantID, projectID, exerciseID, "version-v2", "# === FILE: main.py ===\nprint('v2')\n")
	projectRepo := &workspaceSeedProjectRepo{}
	projectRepo.setPublished(v1)
	orch := &workspaceSeedOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	service, _ := newWorkspaceSeedService(t, projectRepo, orch)
	ctx := withTenantContext(tenantID)

	first, err := service.StartPracticeEnvironmentForExercise(ctx, "student-a", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("V1: %v", err)
	}
	projectRepo.setPublished(v2)
	second, err := service.StartPracticeEnvironmentForExercise(ctx, "student-b", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("V2: %v", err)
	}
	reopened, err := service.StartPracticeEnvironmentForExercise(ctx, "student-a", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("reabrir V1: %v", err)
	}

	if first.WorkspaceProjectVersionID == nil || *first.WorkspaceProjectVersionID != v1.ID {
		t.Fatalf("el primer entorno debía conservar V1: %v", first.WorkspaceProjectVersionID)
	}
	if second.WorkspaceProjectVersionID == nil || *second.WorkspaceProjectVersionID != v2.ID {
		t.Fatalf("el entorno nuevo debía recibir V2: %v", second.WorkspaceProjectVersionID)
	}
	if reopened.WorkspaceProjectVersionID == nil || *reopened.WorkspaceProjectVersionID != v1.ID {
		t.Fatalf("reabrir no debía mover la versión: %v", reopened.WorkspaceProjectVersionID)
	}
	if orch.prepareCall != 2 {
		t.Fatalf("solo debían sembrarse los dos entornos nuevos, calls=%d", orch.prepareCall)
	}
}
