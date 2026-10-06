package services

import (
	"context"
	"fmt"
	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
)

type CourseModuleService struct {
	moduleRepo   domain.CourseModuleRepository
	exerciseRepo domain.ExerciseRepository
}

func NewCourseModuleService(
	moduleRepo domain.CourseModuleRepository,
	exerciseRepo domain.ExerciseRepository,
) *CourseModuleService {
	return &CourseModuleService{
		moduleRepo:   moduleRepo,
		exerciseRepo: exerciseRepo,
	}
}

func (s *CourseModuleService) CreateModule(ctx context.Context, subjectID, title, description string, orderIndex, passScore int) (*domain.CourseModule, error) {
	if title == "" {
		return nil, fmt.Errorf("el título del módulo es requerido")
	}
	if passScore <= 0 || passScore > 100 {
		passScore = 60
	}
	module := &domain.CourseModule{
		ID:          uuid.New().String(),
		SubjectID:   subjectID,
		Title:       title,
		Description: description,
		OrderIndex:  orderIndex,
		PassScore:   passScore,
	}
	if err := s.moduleRepo.CreateModule(ctx, module); err != nil {
		return nil, err
	}
	return module, nil
}

func (s *CourseModuleService) UpdateModule(ctx context.Context, moduleID, title, description string, orderIndex, passScore int) (*domain.CourseModule, error) {
	module, err := s.moduleRepo.GetModuleByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if title != "" {
		module.Title = title
	}
	module.Description = description
	module.OrderIndex = orderIndex
	if passScore > 0 && passScore <= 100 {
		module.PassScore = passScore
	}
	if err := s.moduleRepo.UpdateModule(ctx, module); err != nil {
		return nil, err
	}
	return module, nil
}

func (s *CourseModuleService) DeleteModule(ctx context.Context, moduleID string) error {
	return s.moduleRepo.DeleteModule(ctx, moduleID)
}

func (s *CourseModuleService) ListModulesBySubject(ctx context.Context, subjectID string) ([]*domain.TeacherModuleDetails, error) {
	modules, err := s.moduleRepo.ListModulesBySubject(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	prereqs, err := s.moduleRepo.GetPrerequisitesBySubject(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	prereqMap := make(map[string][]string)
	for _, p := range prereqs {
		prereqMap[p.ModuleID] = append(prereqMap[p.ModuleID], p.PrerequisiteModuleID)
	}

	// Obtener ejercicios del curso para agrupar IDs
	var exercises []*domain.Exercise
	if s.exerciseRepo != nil {
		exs, _ := s.exerciseRepo.ListBySubject(ctx, "", subjectID)
		if exs != nil {
			exercises = exs
		}
	}

	exerciseIDsMap := make(map[string][]string)
	for _, ex := range exercises {
		if ex.ModuleID != nil && *ex.ModuleID != "" {
			exerciseIDsMap[*ex.ModuleID] = append(exerciseIDsMap[*ex.ModuleID], ex.ID)
		}
	}

	result := make([]*domain.TeacherModuleDetails, len(modules))
	for i, m := range modules {
		pList := prereqMap[m.ID]
		if pList == nil {
			pList = []string{}
		}
		eList := exerciseIDsMap[m.ID]
		if eList == nil {
			eList = []string{}
		}
		result[i] = &domain.TeacherModuleDetails{
			CourseModule:  *m,
			Prerequisites: pList,
			ExerciseIDs:   eList,
		}
	}

	return result, nil
}

func (s *CourseModuleService) SetPrerequisites(ctx context.Context, moduleID string, prerequisiteModuleIDs []string) error {
	targetModule, err := s.moduleRepo.GetModuleByID(ctx, moduleID)
	if err != nil {
		return err
	}

	// 1. Validar que no contenga a sí mismo
	for _, pID := range prerequisiteModuleIDs {
		if pID == moduleID {
			return domain.ErrSelfPrerequisite
		}
	}

	// 2. Validar que todos los prerrequisitos pertenezcan al mismo curso
	allModules, err := s.moduleRepo.ListModulesBySubject(ctx, targetModule.SubjectID)
	if err != nil {
		return err
	}

	subjectModuleMap := make(map[string]bool)
	for _, m := range allModules {
		subjectModuleMap[m.ID] = true
	}

	for _, pID := range prerequisiteModuleIDs {
		if !subjectModuleMap[pID] {
			return domain.ErrPrerequisiteCrossCourse
		}
	}

	// 3. Validar que no se forme un ciclo (chequeo topológico / DFS)
	allPrereqs, err := s.moduleRepo.GetPrerequisitesBySubject(ctx, targetModule.SubjectID)
	if err != nil {
		return err
	}

	if hasCycle(allModules, allPrereqs, moduleID, prerequisiteModuleIDs) {
		return domain.ErrCyclicPrerequisite
	}

	// 4. Persistir
	return s.moduleRepo.SetPrerequisites(ctx, moduleID, prerequisiteModuleIDs)
}

func hasCycle(allModules []*domain.CourseModule, allPrereqs []domain.ModulePrerequisite, targetModuleID string, newPrereqs []string) bool {
	adj := make(map[string][]string)
	for _, p := range allPrereqs {
		if p.ModuleID != targetModuleID {
			adj[p.ModuleID] = append(adj[p.ModuleID], p.PrerequisiteModuleID)
		}
	}
	adj[targetModuleID] = append([]string{}, newPrereqs...)

	visited := make(map[string]int) // 0 = unvisited, 1 = visiting, 2 = visited

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = 1
		for _, neighbor := range adj[node] {
			if visited[neighbor] == 1 {
				return true
			}
			if visited[neighbor] == 0 {
				if dfs(neighbor) {
					return true
				}
			}
		}
		visited[node] = 2
		return false
	}

	for _, m := range allModules {
		if visited[m.ID] == 0 {
			if dfs(m.ID) {
				return true
			}
		}
	}

	return false
}

func (s *CourseModuleService) AssignExerciseModule(ctx context.Context, exerciseID string, moduleID *string) error {
	if moduleID != nil && *moduleID != "" {
		_, err := s.moduleRepo.GetModuleByID(ctx, *moduleID)
		if err != nil {
			return err
		}
	}
	return s.moduleRepo.AssignExerciseModule(ctx, exerciseID, moduleID)
}

func (s *CourseModuleService) GetStudentCurricularMap(ctx context.Context, tenantID, subjectID, studentID string) (*domain.CourseCurricularMap, error) {
	return s.moduleRepo.GetCourseCurricularMapData(ctx, tenantID, subjectID, studentID)
}

func (s *CourseModuleService) IsModuleLockedForStudent(ctx context.Context, tenantID, moduleID, studentID string) (bool, error) {
	return s.moduleRepo.IsModuleLockedForStudent(ctx, tenantID, moduleID, studentID)
}
