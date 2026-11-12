package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockTeacherEnrollmentStore struct {
	searchFn func(ctx context.Context, tenantID, teacherID, subjectID, query string, limit int) ([]*domain.AvailableStudentItem, error)
	enrollFn func(ctx context.Context, tenantID, teacherID, subjectID, studentID string) error
}

func (m *mockTeacherEnrollmentStore) SearchAvailableStudentsForCourse(
	ctx context.Context,
	tenantID, teacherID, subjectID, query string,
	limit int,
) ([]*domain.AvailableStudentItem, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, tenantID, teacherID, subjectID, query, limit)
	}
	return nil, nil
}

func (m *mockTeacherEnrollmentStore) EnrollStudentInCourse(
	ctx context.Context,
	tenantID, teacherID, subjectID, studentID string,
) error {
	if m.enrollFn != nil {
		return m.enrollFn(ctx, tenantID, teacherID, subjectID, studentID)
	}
	return nil
}

type dummyTeacherRepo struct {
	domain.TeacherRepository
}

func TestTeacherService_SearchAvailableStudents(t *testing.T) {
	t.Run("fails when store not configured", func(t *testing.T) {
		svc := services.NewTeacherService(&dummyTeacherRepo{}, nil)
		_, err := svc.SearchAvailableStudents(context.Background(), "t1", "teach1", "sub1", "query")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when tenant or subject is missing", func(t *testing.T) {
		mockStore := &mockTeacherEnrollmentStore{}
		svc := services.NewTeacherService(&dummyTeacherRepo{}, nil)
		svc.SetEnrollmentStore(mockStore)

		_, err := svc.SearchAvailableStudents(context.Background(), "", "teach1", "sub1", "query")
		if err == nil {
			t.Fatal("expected error on empty tenant, got nil")
		}

		_, err = svc.SearchAvailableStudents(context.Background(), "t1", "teach1", "", "query")
		if err == nil {
			t.Fatal("expected error on empty subject, got nil")
		}
	})

	t.Run("delegates to store with query and limit", func(t *testing.T) {
		called := false
		mockStore := &mockTeacherEnrollmentStore{
			searchFn: func(ctx context.Context, tenantID, teacherID, subjectID, query string, limit int) ([]*domain.AvailableStudentItem, error) {
				called = true
				if tenantID != "t1" || teacherID != "teach1" || subjectID != "sub1" || query != "algo" || limit != 20 {
					t.Fatalf("unexpected args: %v, %v, %v, %v, %v", tenantID, teacherID, subjectID, query, limit)
				}
				return []*domain.AvailableStudentItem{
					{ID: "s1", FirstName: "Ada", LastName: "Lovelace", Email: "ada@u.edu"},
				}, nil
			},
		}
		svc := services.NewTeacherService(&dummyTeacherRepo{}, nil)
		svc.SetEnrollmentStore(mockStore)

		results, err := svc.SearchAvailableStudents(context.Background(), "t1", "teach1", "sub1", "algo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called || len(results) != 1 || results[0].ID != "s1" {
			t.Fatalf("unexpected results: %v", results)
		}
	})
}

func TestTeacherService_EnrollStudentInCourse(t *testing.T) {
	t.Run("fails when store not configured", func(t *testing.T) {
		svc := services.NewTeacherService(&dummyTeacherRepo{}, nil)
		err := svc.EnrollStudentInCourse(context.Background(), "t1", "teach1", "sub1", "s1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when required fields missing", func(t *testing.T) {
		mockStore := &mockTeacherEnrollmentStore{}
		svc := services.NewTeacherService(&dummyTeacherRepo{}, nil)
		svc.SetEnrollmentStore(mockStore)

		if err := svc.EnrollStudentInCourse(context.Background(), "", "teach1", "sub1", "s1"); err == nil {
			t.Fatal("expected error on empty tenant")
		}
		if err := svc.EnrollStudentInCourse(context.Background(), "t1", "teach1", "", "s1"); err == nil {
			t.Fatal("expected error on empty subject")
		}
		if err := svc.EnrollStudentInCourse(context.Background(), "t1", "teach1", "sub1", ""); err == nil {
			t.Fatal("expected error on empty student")
		}
	})

	t.Run("delegates to store", func(t *testing.T) {
		called := false
		mockStore := &mockTeacherEnrollmentStore{
			enrollFn: func(ctx context.Context, tenantID, teacherID, subjectID, studentID string) error {
				called = true
				if tenantID != "t1" || teacherID != "teach1" || subjectID != "sub1" || studentID != "s1" {
					t.Fatalf("unexpected args: %v, %v, %v, %v", tenantID, teacherID, subjectID, studentID)
				}
				return nil
			},
		}
		svc := services.NewTeacherService(&dummyTeacherRepo{}, nil)
		svc.SetEnrollmentStore(mockStore)

		err := svc.EnrollStudentInCourse(context.Background(), "t1", "teach1", "sub1", "s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("store was not called")
		}
	})
}
