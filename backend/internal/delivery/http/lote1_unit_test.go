package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	commondomain "solv-backend/internal/domain"
)

type mockUserRepo struct {
	user commondomain.UserResponseDTO
	err  error
}

func (m *mockUserRepo) CreateUser(ctx context.Context, dto commondomain.CreateUserDTO) (string, error) {
	return "user-123", nil
}
func (m *mockUserRepo) GetUserByID(ctx context.Context, id string) (commondomain.UserResponseDTO, error) {
	if m.err != nil {
		return commondomain.UserResponseDTO{}, m.err
	}
	return m.user, nil
}
func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*commondomain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) CreateUserFromSSO(ctx context.Context, user *commondomain.User) (string, error) {
	return "user-123", nil
}

func TestUserHandler_GetMe(t *testing.T) {
	t.Run("401 Unauthorized when X-User-Id is missing", func(t *testing.T) {
		h := NewUserHandler(&mockUserRepo{}, nil)
		req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		w := httptest.NewRecorder()

		h.GetMe(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("404 Not Found when user does not exist", func(t *testing.T) {
		h := NewUserHandler(&mockUserRepo{err: errors.New("not found")}, nil)
		req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		req.Header.Set("X-User-Id", "non-existent")
		w := httptest.NewRecorder()

		h.GetMe(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("200 OK with complete user profile", func(t *testing.T) {
		mockUser := commondomain.UserResponseDTO{
			ID:        "user-ada-101",
			FirstName: "Ada",
			LastName:  "Lovelace",
			Email:     "alovelace@uab.edu.bo",
			Role:      "student",
			TenantID:  "tenant-uab-001",
		}
		h := NewUserHandler(&mockUserRepo{user: mockUser}, nil)
		req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		req.Header.Set("X-User-Id", "user-ada-101")
		w := httptest.NewRecorder()

		h.GetMe(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].(map[string]interface{})

		if data["email"] != "alovelace@uab.edu.bo" {
			t.Errorf("Expected email alovelace@uab.edu.bo, got %v", data["email"])
		}
		if data["full_name"] != "Ada Lovelace" {
			t.Errorf("Expected full_name 'Ada Lovelace', got %v", data["full_name"])
		}
		if data["role"] != "student" {
			t.Errorf("Expected role 'student', got %v", data["role"])
		}
		if data["tenant_id"] != "tenant-uab-001" {
			t.Errorf("Expected tenant_id 'tenant-uab-001', got %v", data["tenant_id"])
		}
	})
}

type mockExerciseRepoForDue struct {
	assignments []*domain.DueAssignment
}

func (m *mockExerciseRepoForDue) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return nil, nil
}
func (m *mockExerciseRepoForDue) Create(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}
func (m *mockExerciseRepoForDue) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}
func (m *mockExerciseRepoForDue) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return m.assignments, nil
}

func TestStudentHandler_GetDueAssignments(t *testing.T) {
	t.Run("401 when tenant or user missing", func(t *testing.T) {
		h := NewStudentHandler(nil, nil, nil, nil)
		req := httptest.NewRequest("GET", "/api/v1/student/assignments/due", nil)
		w := httptest.NewRecorder()

		h.GetDueAssignments(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("200 OK with due assignments list", func(t *testing.T) {
		dueTime := time.Now().Add(24 * time.Hour)
		mockAssignments := []*domain.DueAssignment{
			{
				ExerciseID:  "ex-avl-01",
				Title:       "Árboles AVL",
				SubjectID:   "subj-ed-101",
				SubjectName: "Estructuras de Datos",
				SubjectCode: "ED-101",
				DueDate:     &dueTime,
				Type:        "algorithm",
			},
		}

		h := NewStudentHandler(nil, nil, nil, &mockExerciseRepoForDue{assignments: mockAssignments})
		req := httptest.NewRequest("GET", "/api/v1/student/assignments/due", nil)
		ctx := context.WithValue(req.Context(), domain.TenantIDKey, "00000000-0000-0000-0000-000000000001")
		req = req.WithContext(ctx)
		req.Header.Set("X-User-Id", "student-123")
		w := httptest.NewRecorder()

		h.GetDueAssignments(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		dataList := resp["data"].([]interface{})
		if len(dataList) != 1 {
			t.Fatalf("Expected 1 assignment, got %d", len(dataList))
		}
		item := dataList[0].(map[string]interface{})
		if item["title"] != "Árboles AVL" {
			t.Errorf("Expected title 'Árboles AVL', got %v", item["title"])
		}
	})
}

func TestWebSocketHub_Basics(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	client := &WebSocketClient{
		hub:      hub,
		conn:     nil,
		send:     make(chan []byte, 10),
		userID:   "user-ws-test",
		tenantID: "tenant-01",
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.EmitToUser("user-ws-test", WebSocketMessage{
		Event: "TEST_EVENT",
		Stage: "QUEUED",
	})

	select {
	case msg := <-client.send:
		var parsed WebSocketMessage
		json.Unmarshal(msg, &parsed)
		if parsed.Event != "TEST_EVENT" {
			t.Errorf("Expected event TEST_EVENT, got %v", parsed.Event)
		}
		if parsed.Stage != "QUEUED" {
			t.Errorf("Expected stage QUEUED, got %v", parsed.Stage)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timed out waiting for WebSocket message from hub")
	}

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)
}
