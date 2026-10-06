package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/delivery/http/dto"
)

type mockCourseModuleRepoForHTTP struct {
	modules       map[string]*domain.CourseModule
	prerequisites map[string][]string
	exerciseMap   map[string]*string
	curMapData    *domain.CourseCurricularMap
	isLockedMap   map[string]bool
}

func newMockCourseModuleRepoForHTTP() *mockCourseModuleRepoForHTTP {
	return &mockCourseModuleRepoForHTTP{
		modules:       make(map[string]*domain.CourseModule),
		prerequisites: make(map[string][]string),
		exerciseMap:   make(map[string]*string),
		isLockedMap:   make(map[string]bool),
	}
}

func (m *mockCourseModuleRepoForHTTP) CreateModule(ctx context.Context, module *domain.CourseModule) error {
	m.modules[module.ID] = module
	return nil
}

func (m *mockCourseModuleRepoForHTTP) GetModuleByID(ctx context.Context, moduleID string) (*domain.CourseModule, error) {
	mod, exists := m.modules[moduleID]
	if !exists {
		return nil, domain.ErrModuleNotFound
	}
	return mod, nil
}

func (m *mockCourseModuleRepoForHTTP) ListModulesBySubject(ctx context.Context, subjectID string) ([]*domain.CourseModule, error) {
	var list []*domain.CourseModule
	for _, mod := range m.modules {
		if mod.SubjectID == subjectID {
			list = append(list, mod)
		}
	}
	return list, nil
}

func (m *mockCourseModuleRepoForHTTP) UpdateModule(ctx context.Context, module *domain.CourseModule) error {
	m.modules[module.ID] = module
	return nil
}

func (m *mockCourseModuleRepoForHTTP) DeleteModule(ctx context.Context, moduleID string) error {
	delete(m.modules, moduleID)
	return nil
}

func (m *mockCourseModuleRepoForHTTP) SetPrerequisites(ctx context.Context, moduleID string, prerequisiteModuleIDs []string) error {
	m.prerequisites[moduleID] = prerequisiteModuleIDs
	return nil
}

func (m *mockCourseModuleRepoForHTTP) GetPrerequisitesBySubject(ctx context.Context, subjectID string) ([]domain.ModulePrerequisite, error) {
	var list []domain.ModulePrerequisite
	for modID, prereqs := range m.prerequisites {
		if mod, exists := m.modules[modID]; exists && mod.SubjectID == subjectID {
			for _, p := range prereqs {
				list = append(list, domain.ModulePrerequisite{
					ModuleID:             modID,
					PrerequisiteModuleID: p,
				})
			}
		}
	}
	return list, nil
}

func (m *mockCourseModuleRepoForHTTP) GetPrerequisitesForModule(ctx context.Context, moduleID string) ([]string, error) {
	return m.prerequisites[moduleID], nil
}

func (m *mockCourseModuleRepoForHTTP) AssignExerciseModule(ctx context.Context, exerciseID string, moduleID *string) error {
	m.exerciseMap[exerciseID] = moduleID
	return nil
}

func (m *mockCourseModuleRepoForHTTP) GetCourseCurricularMapData(ctx context.Context, tenantID, subjectID, studentID string) (*domain.CourseCurricularMap, error) {
	if m.curMapData != nil {
		return m.curMapData, nil
	}
	return &domain.CourseCurricularMap{
		CourseID: subjectID,
		Modules:  []domain.CurricularModuleMap{},
	}, nil
}

func (m *mockCourseModuleRepoForHTTP) IsModuleLockedForStudent(ctx context.Context, tenantID, moduleID, studentID string) (bool, error) {
	return m.isLockedMap[moduleID], nil
}

func TestCourseModuleHandler_CreateAndSetPrerequisites(t *testing.T) {
	repo := newMockCourseModuleRepoForHTTP()
	exRepo := &mockExerciseRepoForRecs{}
	svc := services.NewCourseModuleService(repo, exRepo)
	handler := httpdelivery.NewCourseModuleHandler(svc)

	// Create module 1
	body1, _ := json.Marshal(dto.CreateModuleRequest{
		Title:      "Module 1",
		OrderIndex: 1,
		PassScore:  60,
	})
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/course-123/modules", bytes.NewReader(body1))
	req1.SetPathValue("id", "course-123")
	w1 := httptest.NewRecorder()
	handler.CreateModule(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w1.Code, w1.Body.String())
	}

	var res1 struct {
		Data domain.CourseModule `json:"data"`
	}
	if err := json.NewDecoder(w1.Body).Decode(&res1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	created1 := res1.Data

	// Create module 2
	body2, _ := json.Marshal(dto.CreateModuleRequest{
		Title:      "Module 2",
		OrderIndex: 2,
		PassScore:  70,
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/course-123/modules", bytes.NewReader(body2))
	req2.SetPathValue("id", "course-123")
	w2 := httptest.NewRecorder()
	handler.CreateModule(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w2.Code, w2.Body.String())
	}

	var res2 struct {
		Data domain.CourseModule `json:"data"`
	}
	_ = json.NewDecoder(w2.Body).Decode(&res2)
	created2 := res2.Data

	// Set valid prerequisite M2 -> M1
	bodyPrereqValid, _ := json.Marshal(dto.SetPrerequisitesRequest{
		PrerequisiteModuleIDs: []string{created1.ID},
	})
	reqPrereq1 := httptest.NewRequest(http.MethodPut, "/api/v1/courses/course-123/modules/"+created2.ID+"/prerequisites", bytes.NewReader(bodyPrereqValid))
	reqPrereq1.SetPathValue("id", "course-123")
	reqPrereq1.SetPathValue("moduleId", created2.ID)
	wPrereq1 := httptest.NewRecorder()
	handler.SetPrerequisites(wPrereq1, reqPrereq1)

	if wPrereq1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid prerequisite, got %d: %s", wPrereq1.Code, wPrereq1.Body.String())
	}

	// Attempt to create cycle: set M1 -> M2
	bodyPrereqCycle, _ := json.Marshal(dto.SetPrerequisitesRequest{
		PrerequisiteModuleIDs: []string{created2.ID},
	})
	reqPrereqCycle := httptest.NewRequest(http.MethodPut, "/api/v1/courses/course-123/modules/"+created1.ID+"/prerequisites", bytes.NewReader(bodyPrereqCycle))
	reqPrereqCycle.SetPathValue("id", "course-123")
	reqPrereqCycle.SetPathValue("moduleId", created1.ID)
	wPrereqCycle := httptest.NewRecorder()
	handler.SetPrerequisites(wPrereqCycle, reqPrereqCycle)

	if wPrereqCycle.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity for cyclic prerequisite, got %d: %s", wPrereqCycle.Code, wPrereqCycle.Body.String())
	}
}

func TestCourseModuleHandler_StudentCurricularMap(t *testing.T) {
	repo := newMockCourseModuleRepoForHTTP()
	exRepo := &mockExerciseRepoForRecs{}
	repo.curMapData = &domain.CourseCurricularMap{
		CourseID: "course-1",
		Modules: []domain.CurricularModuleMap{
			{
				ID:         "mod-1",
				Title:      "Introducción",
				OrderIndex: 1,
				State:      domain.ModuleStateCompleted,
				PassScore:  60,
			},
			{
				ID:                    "mod-2",
				Title:                 "Avanzado",
				OrderIndex:            2,
				State:                 domain.ModuleStateLocked,
				PassScore:             60,
				PrerequisiteModuleIDs: []string{"mod-1"},
				LockReason:            "Requiere completar los módulos previos",
			},
		},
	}

	svc := services.NewCourseModuleService(repo, exRepo)
	handler := httpdelivery.NewCourseModuleHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/student/courses/course-1/map", nil)
	req.SetPathValue("id", "course-1")
	req.Header.Set("X-Tenant-Id", "tenant-test")
	req.Header.Set("X-User-Id", "student-test")
	w := httptest.NewRecorder()
	handler.GetStudentCurricularMap(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data domain.CourseCurricularMap `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.Data.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(res.Data.Modules))
	}
	if res.Data.Modules[1].State != domain.ModuleStateLocked {
		t.Errorf("expected module 2 state locked, got %s", res.Data.Modules[1].State)
	}
}
