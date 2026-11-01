package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// practiceWorkspaceRepo recuerda los entornos del docente por práctica, que es
// la clave con la que el servicio los busca antes de levantar uno nuevo.
type practiceWorkspaceRepo struct {
	*mockWorkspaceRepo
	byPractice map[string]*domain.WorkspaceInstance
}

func newPracticeWorkspaceRepo() *practiceWorkspaceRepo {
	return &practiceWorkspaceRepo{
		mockWorkspaceRepo: newMockWorkspaceRepo(),
		byPractice:        make(map[string]*domain.WorkspaceInstance),
	}
}

func (r *practiceWorkspaceRepo) GetByStudentSubjectExercise(_ context.Context, studentID, subjectID, exerciseID string) (*domain.WorkspaceInstance, error) {
	return r.byPractice[studentID+":"+subjectID+":"+exerciseID], nil
}

func (r *practiceWorkspaceRepo) Create(ctx context.Context, ws *domain.WorkspaceInstance) error {
	if err := r.mockWorkspaceRepo.Create(ctx, ws); err != nil {
		return err
	}
	if ws.ExerciseID != nil {
		r.byPractice[ws.StudentID+":"+ws.SubjectID+":"+*ws.ExerciseID] = ws
	}
	return nil
}

type practiceExerciseMapRepo struct {
	domain.ExerciseRepository
	exercises map[string]*domain.Exercise
}

func (r *practiceExerciseMapRepo) GetByIDAndTenant(_ context.Context, id, _ string) (*domain.Exercise, error) {
	return r.exercises[id], nil
}

func (r *practiceExerciseMapRepo) GetByID(_ context.Context, id string) (*domain.Exercise, error) {
	return r.exercises[id], nil
}

const (
	ideTeacherID = "teacher-ide"
	ideSubjectID = "subj-ide"
	ideOtherID   = "teacher-otro"
	idePracticeA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	idePracticeB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	ideJudgeID   = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
)

func newPracticeIDEService(owner string) (*services.WorkspaceService, *practiceWorkspaceRepo, *mockOrchestrator) {
	subject := &domain.Subject{ID: ideSubjectID, Name: "Estructuras de Datos", TeacherID: &owner}
	subjectID := ideSubjectID

	exercises := map[string]*domain.Exercise{
		idePracticeA: {
			ID:              idePracticeA,
			SubjectID:       &subjectID,
			Title:           "Árboles AVL",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
			TemplateID:      "tpl-a",
		},
		idePracticeB: {
			ID:              idePracticeB,
			SubjectID:       &subjectID,
			Title:           "Grafos",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
			TemplateID:      "tpl-b",
		},
		ideJudgeID: {
			ID:              ideJudgeID,
			SubjectID:       &subjectID,
			Title:           "Ordenamiento",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
	}

	templates := map[string]*domain.Template{
		"tpl-a": {ID: "tpl-a", Name: "Python Estructuras", DockerImage: "solv/python:3.12", BaseRamMB: 512},
		"tpl-b": {ID: "tpl-b", Name: "Python Grafos", DockerImage: "solv/python:3.12-graph", BaseRamMB: 512},
	}

	repo := newPracticeWorkspaceRepo()
	orch := &mockOrchestrator{}
	svc := services.NewWorkspaceService(repo, orch, &mockHostMonitor{}).
		WithProvisioning(&mockSubjectRepo{subjects: map[string]*domain.Subject{ideSubjectID: subject}}, &mockTemplateRepo{templates: templates}).
		WithExerciseSubmission(&practiceExerciseMapRepo{exercises: exercises}, &fakeSubmissionRepo{})
	return svc, repo, orch
}

// Abrir el IDE de una práctica del docente usa el volumen y la plantilla de esa
// práctica: dos prácticas de la misma materia no comparten repositorio.
func TestStartTeacherPracticeWorkspace_ScopesVolumeAndTemplateToPractice(t *testing.T) {
	svc, _, orch := newPracticeIDEService(ideTeacherID)

	first, err := svc.StartTeacherPracticeWorkspace(context.Background(), ideTeacherID, "", idePracticeA, "teacher")
	if err != nil {
		t.Fatalf("la práctica del docente debía abrir: %v", err)
	}
	if first.ExerciseID == nil || *first.ExerciseID != idePracticeA {
		t.Fatalf("el entorno debía quedar acotado a la práctica %s: %v", idePracticeA, first.ExerciseID)
	}
	wantVolume := "solv_workspace_" + ideTeacherID + "_" + ideSubjectID + "_" + idePracticeA
	if first.VolumeName() != wantVolume {
		t.Errorf("volumen inesperado: %s, esperado: %s", first.VolumeName(), wantVolume)
	}
	if orch.lastConfig == nil || orch.lastConfig.VolumeName != wantVolume {
		t.Errorf("el contenedor debía montar el volumen de la práctica: %+v", orch.lastConfig)
	}
	if orch.lastConfig.Image != "solv/python:3.12" {
		t.Errorf("esperada la imagen de la plantilla de la práctica, obtenida: %s", orch.lastConfig.Image)
	}
	if first.AccessURL == "" {
		t.Error("el entorno debía exponer una URL de acceso para abrirlo en otra pestaña")
	}

	second, err := svc.StartTeacherPracticeWorkspace(context.Background(), ideTeacherID, "", idePracticeB, "teacher")
	if err != nil {
		t.Fatalf("la segunda práctica debía abrir: %v", err)
	}
	if second.ID == first.ID {
		t.Error("cada práctica tiene su propio entorno, no el de la práctica anterior")
	}
	if second.VolumeName() == first.VolumeName() {
		t.Errorf("dos prácticas distintas compartieron volumen: %s", second.VolumeName())
	}
	if orch.lastConfig.Image != "solv/python:3.12-graph" {
		t.Errorf("esperada la imagen de la segunda práctica, obtenida: %s", orch.lastConfig.Image)
	}
}

// Volver a abrir la misma práctica devuelve el entorno que ya existe en vez de
// levantar un contenedor nuevo.
func TestStartTeacherPracticeWorkspace_ReusesExistingPracticeEnvironment(t *testing.T) {
	svc, _, _ := newPracticeIDEService(ideTeacherID)

	first, err := svc.StartTeacherPracticeWorkspace(context.Background(), ideTeacherID, "", idePracticeA, "teacher")
	if err != nil {
		t.Fatalf("la práctica debía abrir: %v", err)
	}
	first.Status = domain.WorkspaceStatusRunning

	again, err := svc.StartTeacherPracticeWorkspace(context.Background(), ideTeacherID, "", idePracticeA, "teacher")
	if err != nil {
		t.Fatalf("reabrir la misma práctica no debía fallar: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("esperado el entorno existente %s, obtenido %s", first.ID, again.ID)
	}
}

// Un docente no abre el IDE de una práctica de la materia de otro.
func TestStartTeacherPracticeWorkspace_RejectsForeignTeacher(t *testing.T) {
	svc, _, _ := newPracticeIDEService(ideOtherID)

	_, err := svc.StartTeacherPracticeWorkspace(context.Background(), ideTeacherID, "", idePracticeA, "teacher")
	if !errors.Is(err, services.ErrPracticeTestForbidden) {
		t.Fatalf("esperado ErrPracticeTestForbidden, obtenido: %v", err)
	}
}

// Una práctica del juez no tiene repositorio persistente que abrir.
func TestStartTeacherPracticeWorkspace_RejectsJudgePractice(t *testing.T) {
	svc, _, _ := newPracticeIDEService(ideTeacherID)

	_, err := svc.StartTeacherPracticeWorkspace(context.Background(), ideTeacherID, "", ideJudgeID, "teacher")
	if !errors.Is(err, services.ErrPracticeIDENotProject) {
		t.Fatalf("esperado ErrPracticeIDENotProject, obtenido: %v", err)
	}
}
