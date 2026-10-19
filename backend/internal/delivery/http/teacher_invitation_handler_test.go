package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockTeacherInvRepo struct {
	teachers []*domain.TeacherListItem
}

func (m *mockTeacherInvRepo) Create(ctx context.Context, inv *domain.TeacherInvitation) error {
	return nil
}
func (m *mockTeacherInvRepo) GetByToken(ctx context.Context, tenantID, token string) (*domain.TeacherInvitation, error) {
	return nil, nil
}
func (m *mockTeacherInvRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.TeacherInvitation, error) {
	return nil, nil
}
func (m *mockTeacherInvRepo) Update(ctx context.Context, inv *domain.TeacherInvitation) error {
	return nil
}
func (m *mockTeacherInvRepo) AcceptInvitationTx(ctx context.Context, tenantID, token, userID, userEmail string) error {
	return nil
}
func (m *mockTeacherInvRepo) ListTeachers(ctx context.Context, tenantID, search, status, origin string) ([]*domain.TeacherListItem, error) {
	return m.teachers, nil
}
func (m *mockTeacherInvRepo) GetTeacherCourses(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherCourseItem, error) {
	return nil, nil
}
func (m *mockTeacherInvRepo) DeleteInvitation(ctx context.Context, tenantID, id string) error {
	return nil
}
func (m *mockTeacherInvRepo) DeleteTeacher(ctx context.Context, tenantID, teacherID string) error {
	return nil
}
func (m *mockTeacherInvRepo) UpsertDirectTeacher(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.TeacherListItem, error) {
	item := &domain.TeacherListItem{
		ID:        "mock-teacher-id",
		FullName:  firstName + " " + lastName,
		Email:     email,
		Origin:    "manual",
		Status:    "active",
		RoleType:  "titular",
		InvitedAt: "Today",
	}
	m.teachers = append(m.teachers, item)
	return item, nil
}
func (m *mockTeacherInvRepo) BatchUpsertDirectTeachers(ctx context.Context, tenantID string, emails []string) (int, error) {
	for _, em := range emails {
		m.teachers = append(m.teachers, &domain.TeacherListItem{
			ID:     "batch-" + em,
			Email:  em,
			Status: "active",
		})
	}
	return len(emails), nil
}

func (m *mockTeacherInvRepo) CheckAndConsumeInvitation(ctx context.Context, tenantID, email string) (bool, error) {
	return true, nil
}

func TestTeacherInvitationHandler_RegisterDirectTeacher(t *testing.T) {
	repo := &mockTeacherInvRepo{}
	svc := services.NewTeacherInvitationService(repo)
	handler := httpdelivery.NewTeacherInvitationHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"email":             "profesor.nuevo@uab.edu.bo",
		"first_name":        "Carlos",
		"last_name":         "Mendoza",
		"send_notification": false,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/teachers/direct", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RegisterDirectTeacher(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if res["notified"] != false {
		t.Errorf("Expected notified: false, got %v", res["notified"])
	}
}

func TestTeacherInvitationHandler_ParseRoster(t *testing.T) {
	repo := &mockTeacherInvRepo{}
	svc := services.NewTeacherInvitationService(repo)
	handler := httpdelivery.NewTeacherInvitationHandler(svc)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("file", "docentes.csv")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	csvData := "Docente,Correo\nProfesor Uno,profesor1@uab.edu.bo\nProfesor Dos,profesor2@uab.edu.bo\n"
	_, _ = part.Write([]byte(csvData))
	_ = w.Close()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/teachers/parse-roster", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()

	handler.ParseRoster(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var result services.ParsedRosterResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode parsed roster: %v", err)
	}
	if len(result.ValidEmails) != 2 {
		t.Errorf("Expected 2 valid emails, got %d", len(result.ValidEmails))
	}
}

func TestTeacherInvitationHandler_BatchRegisterTeachers(t *testing.T) {
	repo := &mockTeacherInvRepo{}
	svc := services.NewTeacherInvitationService(repo)
	handler := httpdelivery.NewTeacherInvitationHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"emails":            []string{"doc1@uab.edu.bo", "doc2@uab.edu.bo"},
		"send_notification": false,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/teachers/direct-batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BatchRegisterTeachers(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res["registered_count"] != float64(2) {
		t.Errorf("Expected 2 registered, got %v", res["registered_count"])
	}
}
