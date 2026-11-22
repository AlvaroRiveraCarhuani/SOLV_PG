package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestWorkspaceProjectArtifactSubmissionUsesLargeLimits comprueba que un
// entorno sembrado con artefacto pueda entregar un árbol que el camino liviano
// rechazaría, y que un proyecto de texto siga cortando en el tope anterior.
func TestWorkspaceProjectArtifactSubmissionUsesLargeLimits(t *testing.T) {
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

	archive := buildPublishArtifact(t, map[string]string{"main.py": "print('base')\n"})
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	key, err := domain.WorkspaceProjectArtifactKey(seed.tenantID, seed.exerciseA, hash)
	if err != nil {
		t.Fatalf("clave del artefacto: %v", err)
	}
	store := &testArtifactStore{objects: map[string][]byte{key: archive}}

	envService := services.NewPracticeEnvironmentService(
		postgres.NewPostgresPracticeEnvironmentRepository(sqlDB),
		volume,
		&mockSlice12HostMonitor{},
	).
		WithProvisioning(subjectRepo, templateRepo).
		WithExerciseSubmission(exerciseRepo, submissionRepo).
		SetWorkspaceProjects(workspaceRepo)
	envService.SetWorkspaceArtifactStore(store)

	// Proyecto grande: una versión con artefacto, sembrada desde el depósito.
	project, err := workspaceRepo.CreateProject(ctx, &domain.WorkspaceProject{
		TenantID:   seed.tenantID,
		ExerciseID: seed.exerciseA,
		CreatedBy:  &seed.teacherID,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	artifactBytes := int64(len(archive))
	artifactFiles := 1
	if _, err := workspaceRepo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:       seed.tenantID,
		ProjectID:      project.ID,
		ContentSHA256:  hash,
		ArtifactKey:    &key,
		ArtifactSHA256: &hash,
		ArtifactBytes:  &artifactBytes,
		ArtifactFiles:  &artifactFiles,
	}, 0); err != nil {
		t.Fatalf("borrador con artefacto: %v", err)
	}
	if _, err := workspaceRepo.Publish(ctx, seed.tenantID, project.ID, seed.teacherID, nil); err != nil {
		t.Fatalf("publicar el artefacto: %v", err)
	}
	environment, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentA, seed.subjectID, seed.exerciseA)
	if err != nil {
		t.Fatalf("abrir el entorno del alumno: %v", err)
	}

	// El alumno trabaja: dos archivos de 3 MiB suman 6 MiB, más que los 4 MiB del
	// camino liviano y menos que el tope por archivo del proyecto grande.
	big := strings.Repeat("x", 3*1024*1024)
	volume.mu.Lock()
	volume.files = map[string]string{"src/a.py": big, "src/b.py": big}
	volume.mu.Unlock()

	submission, err := envService.SubmitPracticeEnvironmentProject(ctx, environment.ID, seed.studentA, seed.exerciseA, nil)
	if err != nil {
		t.Fatalf("la entrega de un proyecto grande no debe cortar en 4 MB: %v", err)
	}
	// La entrega grande no se guarda como texto: la fila referencia el depósito
	// y el objeto contiene el árbol completo.
	if submission.Code != "" {
		t.Fatalf("la entrega grande no debe guardar texto en la base: %d bytes", len(submission.Code))
	}
	artifact := submission.Artifact()
	if artifact == nil {
		t.Fatal("la entrega grande debía quedar referenciada en el depósito")
	}
	if artifact.Files != 2 {
		t.Fatalf("archivos declarados en el artefacto: %d", artifact.Files)
	}
	if _, ok := store.objects[artifact.Key]; !ok {
		t.Fatalf("el objeto de la entrega no está en el depósito: %s", artifact.Key)
	}

	// La revisión del docente lee el árbol desde el depósito y lo devuelve
	// navegable, igual que con una entrega de texto.
	workspaceService := services.NewWorkspaceProjectService(workspaceRepo, true)
	workspaceService.SetArtifactStore(store)
	teacherService := services.NewTeacherService(postgres.NewPostgresTeacherRepository(sqlDB), submissionRepo)
	teacherService.SetWorkspaceBase(workspaceService)
	review, err := teacherService.GetTeacherSubmissionReview(ctx, seed.tenantID, seed.teacherID, submission.ID)
	if err != nil {
		t.Fatalf("revisión de la entrega grande: %v", err)
	}
	if !review.FilesAvailable || len(review.Files) != 2 {
		t.Fatalf("la revisión debía traer los 2 archivos del depósito: %+v", review.Files)
	}
	for _, file := range review.Files {
		if file.Path != "src/a.py" && file.Path != "src/b.py" {
			t.Fatalf("archivo inesperado en la revisión: %s", file.Path)
		}
		if len(file.Content) != 3*1024*1024 {
			t.Fatalf("la revisión devolvió %s con %d bytes", file.Path, len(file.Content))
		}
	}

	// Control: el mismo árbol en un proyecto de texto sigue cortando en 4 MB.
	textProject, err := workspaceRepo.CreateProject(ctx, &domain.WorkspaceProject{
		TenantID:   seed.tenantID,
		ExerciseID: seed.exerciseB,
		CreatedBy:  &seed.teacherID,
	})
	if err != nil {
		t.Fatalf("crear proyecto de texto: %v", err)
	}
	if _, err := workspaceRepo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:     seed.tenantID,
		ProjectID:    textProject.ID,
		TextSnapshot: "# === FILE: main.py ===\nprint('base')\n",
		UpdatedBy:    &seed.teacherID,
	}, 0); err != nil {
		t.Fatalf("borrador de texto: %v", err)
	}
	if _, err := workspaceRepo.Publish(ctx, seed.tenantID, textProject.ID, seed.teacherID, nil); err != nil {
		t.Fatalf("publicar el proyecto de texto: %v", err)
	}
	textEnvironment, err := envService.StartPracticeEnvironmentForExercise(ctx, seed.studentB, seed.subjectID, seed.exerciseB)
	if err != nil {
		t.Fatalf("abrir el entorno del segundo alumno: %v", err)
	}
	volume.mu.Lock()
	volume.files = map[string]string{"src/a.py": big, "src/b.py": big}
	volume.mu.Unlock()

	if _, err := envService.SubmitPracticeEnvironmentProject(ctx, textEnvironment.ID, seed.studentB, seed.exerciseB, nil); err == nil {
		t.Fatal("un proyecto de texto debía seguir cortando en el tope liviano")
	}
}
