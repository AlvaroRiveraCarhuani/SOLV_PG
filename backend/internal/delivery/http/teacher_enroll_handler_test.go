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
)

type mockTeacherEnrollRepo struct {
	domain.TeacherRepository
	searchFn func(ctx context.Context, tenantID, teacherID, subjectID, query string, limit int) ([]*domain.AvailableStudentItem, error)
	enrollFn func(ctx context.Context, tenantID, teacherID, subjectID, studentID string) error
}

func (m *mockTeacherEnrollRepo) SearchAvailableStudentsForCourse(
	ctx context.Context,
	tenantID, teacherID, subjectID, query string,
	limit int,
) ([]*domain.AvailableStudentItem, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, tenantID, teacherID, subjectID, query, limit)
	}
	return nil, nil
}

func (m *mockTeacherEnrollRepo) EnrollStudentInCourse(
	ctx context.Context,
	tenantID, teacherID, subjectID, studentID string,
) error {
	if m.enrollFn != nil {
		return m.enrollFn(ctx, tenantID, teacherID, subjectID, studentID)
	}
	return nil
}

func TestTeacherHandler_SearchAvailableStudents(t *testing.T) {
	repo := &mockTeacherEnrollRepo{
		searchFn: func(ctx context.Context, tenantID, teacherID, subjectID, query string, limit int) ([]*domain.AvailableStudentItem, error) {
			return []*domain.AvailableStudentItem{
				{ID: "student-1", FirstName: "Alan", LastName: "Turing", Email: "alan@institucion.edu"},
			}, nil
		},
	}
	service := services.NewTeacherService(repo, nil)
	service.SetEnrollmentStore(repo)
	handler := httpdelivery.NewTeacherHandler(service, nil)

	t.Run("rejects unauthorized role", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/sub-1/available-students", nil)
		req = withTestIdentity(req, "user-1", "tenant-1", "student")
		req.SetPathValue("id", "sub-1")

		w := httptest.NewRecorder()
		handler.SearchAvailableStudents(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("returns available students for teacher", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/sub-1/available-students?q=alan", nil)
		req = withTestIdentity(req, "teacher-1", "tenant-1", "teacher")
		req.SetPathValue("id", "sub-1")

		w := httptest.NewRecorder()
		handler.SearchAvailableStudents(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Data []*domain.AvailableStudentItem `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Data) != 1 || resp.Data[0].FirstName != "Alan" {
			t.Fatalf("unexpected data: %+v", resp.Data)
		}
	})
}

func TestTeacherHandler_EnrollStudent(t *testing.T) {
	enrolledStudent := ""
	repo := &mockTeacherEnrollRepo{
		enrollFn: func(ctx context.Context, tenantID, teacherID, subjectID, studentID string) error {
			enrolledStudent = studentID
			return nil
		},
	}
	service := services.NewTeacherService(repo, nil)
	service.SetEnrollmentStore(repo)
	handler := httpdelivery.NewTeacherHandler(service, nil)

	t.Run("rejects unauthorized role", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"student_id": "stud-1"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/courses/sub-1/students", bytes.NewBuffer(body))
		req = withTestIdentity(req, "user-1", "tenant-1", "student")
		req.SetPathValue("id", "sub-1")

		w := httptest.NewRecorder()
		handler.EnrollStudent(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("rejects missing student id payload", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"student_id": "  "})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/courses/sub-1/students", bytes.NewBuffer(body))
		req = withTestIdentity(req, "teacher-1", "tenant-1", "teacher")
		req.SetPathValue("id", "sub-1")

		w := httptest.NewRecorder()
		handler.EnrollStudent(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("successfully enrolls student", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"student_id": "stud-1"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/courses/sub-1/students", bytes.NewBuffer(body))
		req = withTestIdentity(req, "teacher-1", "tenant-1", "teacher")
		req.SetPathValue("id", "sub-1")

		w := httptest.NewRecorder()
		handler.EnrollStudent(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", w.Code)
		}
		if enrolledStudent != "stud-1" {
			t.Fatalf("expected student-1 to be enrolled, got %q", enrolledStudent)
		}
	})
}
