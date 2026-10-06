package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockCourseModuleRepo struct {
	modules       map[string]*domain.CourseModule
	prerequisites map[string][]string // module_id -> [prereq_id, ...]
	exerciseMap   map[string]*string  // exercise_id -> module_id
	curMapData    *domain.CourseCurricularMap
	isLockedMap   map[string]bool
}

func newMockCourseModuleRepo() *mockCourseModuleRepo {
	return &mockCourseModuleRepo{
		modules:       make(map[string]*domain.CourseModule),
		prerequisites: make(map[string][]string),
		exerciseMap:   make(map[string]*string),
		isLockedMap:   make(map[string]bool),
	}
}

func (m *mockCourseModuleRepo) CreateModule(_ context.Context, module *domain.CourseModule) error {
	m.modules[module.ID] = module
	return nil
}

func (m *mockCourseModuleRepo) GetModuleByID(_ context.Context, moduleID string) (*domain.CourseModule, error) {
	mod, ok := m.modules[moduleID]
	if !ok {
		return nil, domain.ErrModuleNotFound
	}
	return mod, nil
}

func (m *mockCourseModuleRepo) ListModulesBySubject(_ context.Context, subjectID string) ([]*domain.CourseModule, error) {
	var list []*domain.CourseModule
	for _, mod := range m.modules {
		if mod.SubjectID == subjectID {
			list = append(list, mod)
		}
	}
	return list, nil
}

func (m *mockCourseModuleRepo) UpdateModule(_ context.Context, module *domain.CourseModule) error {
	if _, ok := m.modules[module.ID]; !ok {
		return domain.ErrModuleNotFound
	}
	m.modules[module.ID] = module
	return nil
}

func (m *mockCourseModuleRepo) DeleteModule(_ context.Context, moduleID string) error {
	if _, ok := m.modules[moduleID]; !ok {
		return domain.ErrModuleNotFound
	}
	delete(m.modules, moduleID)
	delete(m.prerequisites, moduleID)
	return nil
}

func (m *mockCourseModuleRepo) SetPrerequisites(_ context.Context, moduleID string, prerequisiteModuleIDs []string) error {
	m.prerequisites[moduleID] = prerequisiteModuleIDs
	return nil
}

func (m *mockCourseModuleRepo) GetPrerequisitesBySubject(_ context.Context, subjectID string) ([]domain.ModulePrerequisite, error) {
	var list []domain.ModulePrerequisite
	for modID, prereqs := range m.prerequisites {
		if mod, ok := m.modules[modID]; ok && mod.SubjectID == subjectID {
			for _, pID := range prereqs {
				list = append(list, domain.ModulePrerequisite{
					ModuleID:             modID,
					PrerequisiteModuleID: pID,
				})
			}
		}
	}
	return list, nil
}

func (m *mockCourseModuleRepo) GetPrerequisitesForModule(_ context.Context, moduleID string) ([]string, error) {
	return m.prerequisites[moduleID], nil
}

func (m *mockCourseModuleRepo) AssignExerciseModule(_ context.Context, exerciseID string, moduleID *string) error {
	m.exerciseMap[exerciseID] = moduleID
	return nil
}

func (m *mockCourseModuleRepo) GetCourseCurricularMapData(_ context.Context, tenantID, subjectID, studentID string) (*domain.CourseCurricularMap, error) {
	if m.curMapData != nil {
		return m.curMapData, nil
	}
	return &domain.CourseCurricularMap{CourseID: subjectID}, nil
}

func (m *mockCourseModuleRepo) IsModuleLockedForStudent(_ context.Context, tenantID, moduleID, studentID string) (bool, error) {
	return m.isLockedMap[moduleID], nil
}

func TestCourseModuleService_PrerequisitesValidation(t *testing.T) {
	repo := newMockCourseModuleRepo()
	svc := services.NewCourseModuleService(repo, nil)

	ctx := context.Background()
	subjID := "subj-1"

	// Create 3 modules: M1, M2, M3
	m1 := &domain.CourseModule{ID: "m1", SubjectID: subjID, Title: "Module 1", OrderIndex: 1, PassScore: 60}
	m2 := &domain.CourseModule{ID: "m2", SubjectID: subjID, Title: "Module 2", OrderIndex: 2, PassScore: 60}
	m3 := &domain.CourseModule{ID: "m3", SubjectID: subjID, Title: "Module 3", OrderIndex: 3, PassScore: 60}
	repo.modules["m1"] = m1
	repo.modules["m2"] = m2
	repo.modules["m3"] = m3

	t.Run("Self-prerequisite is rejected", func(t *testing.T) {
		err := svc.SetPrerequisites(ctx, "m1", []string{"m1"})
		if err != domain.ErrSelfPrerequisite {
			t.Fatalf("expected ErrSelfPrerequisite, got %v", err)
		}
	})

	t.Run("Cross-course prerequisite is rejected", func(t *testing.T) {
		// Module in another subject
		repo.modules["m-other"] = &domain.CourseModule{ID: "m-other", SubjectID: "other-subj", Title: "Other", OrderIndex: 1}
		err := svc.SetPrerequisites(ctx, "m1", []string{"m-other"})
		if err != domain.ErrPrerequisiteCrossCourse {
			t.Fatalf("expected ErrPrerequisiteCrossCourse, got %v", err)
		}
	})

	t.Run("Valid chain prerequisite M2 -> M1", func(t *testing.T) {
		err := svc.SetPrerequisites(ctx, "m2", []string{"m1"})
		if err != nil {
			t.Fatalf("unexpected error setting valid prerequisite: %v", err)
		}
	})

	t.Run("Cyclic prerequisite M1 -> M2 when M2 -> M1 is rejected", func(t *testing.T) {
		err := svc.SetPrerequisites(ctx, "m1", []string{"m2"})
		if err != domain.ErrCyclicPrerequisite {
			t.Fatalf("expected ErrCyclicPrerequisite, got %v", err)
		}
	})

	t.Run("Diamond DAG is allowed (M4 depends on M2 and M3, which both depend on M1)", func(t *testing.T) {
		m4 := &domain.CourseModule{ID: "m4", SubjectID: subjID, Title: "Module 4", OrderIndex: 4, PassScore: 60}
		repo.modules["m4"] = m4

		_ = svc.SetPrerequisites(ctx, "m2", []string{"m1"})
		_ = svc.SetPrerequisites(ctx, "m3", []string{"m1"})
		err := svc.SetPrerequisites(ctx, "m4", []string{"m2", "m3"})
		if err != nil {
			t.Fatalf("expected diamond DAG to be accepted, got %v", err)
		}
	})
}
