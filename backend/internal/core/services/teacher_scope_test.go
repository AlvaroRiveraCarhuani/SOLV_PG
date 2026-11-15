package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// fakeSubjectOwnership responde por docente y materia, sin base de datos.
type fakeSubjectOwnership struct {
	owners map[string]string // subject_id -> teacher_id
	fail   error
}

func (f fakeSubjectOwnership) TeacherOwnsSubject(_ context.Context, _, subjectID, teacherID string) (bool, error) {
	if f.fail != nil {
		return false, f.fail
	}
	return f.owners[subjectID] == teacherID, nil
}

func ownershipCtx(role, userID string) context.Context {
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, "tenant-1")
	if role != "" {
		ctx = context.WithValue(ctx, domain.UserRoleKey, role)
	}
	if userID != "" {
		ctx = context.WithValue(ctx, domain.UserIDKey, userID)
	}
	return ctx
}

// Un docente solo escribe módulos de las materias que dicta: el id de la materia
// ajena responde igual que uno inexistente.
func TestCourseModuleService_TeacherScopedWrites(t *testing.T) {
	ownership := fakeSubjectOwnership{owners: map[string]string{
		"subject-mine":  "teacher-1",
		"subject-other": "teacher-2",
	}}

	newService := func(t *testing.T) (*services.CourseModuleService, *mockCourseModuleRepo) {
		t.Helper()
		repo := newMockCourseModuleRepo()
		repo.modules["module-other"] = &domain.CourseModule{ID: "module-other", SubjectID: "subject-other", Title: "Ajena"}
		svc := services.NewCourseModuleService(repo, nil)
		svc.SetSubjectOwnership(ownership)
		return svc, repo
	}

	t.Run("docente crea un módulo en su materia", func(t *testing.T) {
		svc, _ := newService(t)
		module, err := svc.CreateModule(ownershipCtx("teacher", "teacher-1"), "subject-mine", "Módulo", "", 1, 60)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if module.SubjectID != "subject-mine" {
			t.Fatalf("módulo creado en la materia equivocada: %q", module.SubjectID)
		}
	})

	t.Run("docente no crea en la materia de otro", func(t *testing.T) {
		svc, repo := newService(t)
		_, err := svc.CreateModule(ownershipCtx("teacher", "teacher-1"), "subject-other", "Intruso", "", 1, 60)
		if !errors.Is(err, domain.ErrModuleNotFound) {
			t.Fatalf("esperado ErrModuleNotFound, obtenido: %v", err)
		}
		if len(repo.modules) != 1 {
			t.Fatalf("no debe crear módulos: hay %d", len(repo.modules))
		}
	})

	t.Run("docente no edita el módulo de otro", func(t *testing.T) {
		svc, repo := newService(t)
		_, err := svc.UpdateModule(ownershipCtx("teacher", "teacher-1"), "module-other", "Secuestrado", "", 2, 70)
		if !errors.Is(err, domain.ErrModuleNotFound) {
			t.Fatalf("esperado ErrModuleNotFound, obtenido: %v", err)
		}
		if repo.modules["module-other"].Title != "Ajena" {
			t.Fatalf("el módulo ajeno se modificó: %q", repo.modules["module-other"].Title)
		}
	})

	t.Run("docente no borra el módulo de otro", func(t *testing.T) {
		svc, repo := newService(t)
		if err := svc.DeleteModule(ownershipCtx("teacher", "teacher-1"), "module-other"); !errors.Is(err, domain.ErrModuleNotFound) {
			t.Fatalf("esperado ErrModuleNotFound, obtenido: %v", err)
		}
		if _, ok := repo.modules["module-other"]; !ok {
			t.Fatal("el módulo ajeno se borró")
		}
	})

	t.Run("docente no reordena dependencias del módulo de otro", func(t *testing.T) {
		svc, _ := newService(t)
		err := svc.SetPrerequisites(ownershipCtx("teacher", "teacher-1"), "module-other", []string{})
		if !errors.Is(err, domain.ErrModuleNotFound) {
			t.Fatalf("esperado ErrModuleNotFound, obtenido: %v", err)
		}
	})

	t.Run("administrador edita cualquier materia de la institución", func(t *testing.T) {
		svc, repo := newService(t)
		if _, err := svc.UpdateModule(ownershipCtx("admin", "admin-1"), "module-other", "Revisado", "", 2, 70); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.modules["module-other"].Title != "Revisado" {
			t.Fatalf("el administrador debería poder editar: %q", repo.modules["module-other"].Title)
		}
	})

	t.Run("estudiante no escribe módulos", func(t *testing.T) {
		svc, _ := newService(t)
		_, err := svc.CreateModule(ownershipCtx("student", "student-1"), "subject-mine", "Módulo", "", 1, 60)
		if !errors.Is(err, domain.ErrModuleNotFound) {
			t.Fatalf("esperado ErrModuleNotFound, obtenido: %v", err)
		}
	})
}

// stubExerciseRepo solo responde la búsqueda del ejercicio; el resto de la
// interfaz queda embebida porque la rúbrica no la usa.
type stubExerciseRepo struct {
	domain.ExerciseRepository
	exercise *domain.Exercise
}

func (s stubExerciseRepo) GetByID(_ context.Context, _ string) (*domain.Exercise, error) {
	return s.exercise, nil
}

type stubRubricRepo struct {
	domain.RubricRepository
	saved bool
}

func (s *stubRubricRepo) CreateOrUpdate(_ context.Context, _ *domain.Rubric) error {
	s.saved = true
	return nil
}

// La rúbrica califica a los alumnos de la materia: un docente no escribe la de
// otra materia, y el intento responde "no existe".
func TestRubricService_TeacherScopedWrites(t *testing.T) {
	exercise := &domain.Exercise{
		ID:              "exercise-1",
		SubjectID:       stringPtr("subject-other"),
		EnvironmentType: "IDE_PERSISTENTE",
	}

	validRubric := func() *domain.Rubric {
		return &domain.Rubric{Criteria: []domain.RubricCriterion{
			{ID: "c1", Name: "Diseño", Weight: 100, Levels: []domain.RubricLevel{
				{Name: "Logrado", Score: 100},
				{Name: "Parcial", Score: 50},
			}},
		}}
	}

	t.Run("docente no escribe la rúbrica de otra materia", func(t *testing.T) {
		repo := &stubRubricRepo{}
		svc := services.NewDefaultRubricService(repo, stubExerciseRepo{exercise: exercise}, nil)
		svc.SetSubjectOwnership(fakeSubjectOwnership{owners: map[string]string{"subject-mine": "teacher-1", "subject-other": "teacher-2"}})

		err := svc.CreateOrUpdate(ownershipCtx("teacher", "teacher-1"), validRubric())
		if !errors.Is(err, domain.ErrRubricNotFound) {
			t.Fatalf("esperado ErrRubricNotFound, obtenido: %v", err)
		}
		if repo.saved {
			t.Fatal("la rúbrica ajena no se debe guardar")
		}
	})

	t.Run("administrador escribe la rúbrica de cualquier materia", func(t *testing.T) {
		repo := &stubRubricRepo{}
		svc := services.NewDefaultRubricService(repo, stubExerciseRepo{exercise: exercise}, nil)
		svc.SetSubjectOwnership(fakeSubjectOwnership{owners: map[string]string{"subject-other": "teacher-2"}})

		if err := svc.CreateOrUpdate(ownershipCtx("admin", "admin-1"), validRubric()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !repo.saved {
			t.Fatal("el administrador debería poder guardar la rúbrica")
		}
	})

	t.Run("docente sin lector de pertenencia no escribe", func(t *testing.T) {
		repo := &stubRubricRepo{}
		svc := services.NewDefaultRubricService(repo, stubExerciseRepo{exercise: exercise}, nil)

		if err := svc.CreateOrUpdate(ownershipCtx("teacher", "teacher-1"), validRubric()); !errors.Is(err, domain.ErrRubricNotFound) {
			t.Fatalf("esperado rechazo sin forma de comprobar la materia, obtenido: %v", err)
		}
		if repo.saved {
			t.Fatal("sin lector, un docente no debe guardar")
		}
	})
}
