package services

import (
	"context"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
)

type SubjectService struct {
	repo domain.SubjectRepository
}

func NewSubjectService(repo domain.SubjectRepository) *SubjectService {
	return &SubjectService{repo: repo}
}

func (s *SubjectService) CreateSubject(ctx context.Context, tenantID, name, code string, classroomCourseID *string) (*domain.Subject, error) {
	if name == "" || code == "" {
		return nil, errors.New("name and code are required")
	}
	subject := &domain.Subject{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Name:              name,
		Code:              code,
		ClassroomCourseID: classroomCourseID,
	}
	if err := s.repo.Create(ctx, subject); err != nil {
		return nil, fmt.Errorf("failed to create subject: %w", err)
	}
	return subject, nil
}

func (s *SubjectService) ListSubjects(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *SubjectService) EnrollStudent(ctx context.Context, tenantID, studentID, subjectID string) (*domain.Enrollment, error) {
	if studentID == "" || subjectID == "" {
		return nil, errors.New("studentID and subjectID are required")
	}
	enrollment := &domain.Enrollment{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		StudentID: studentID,
		SubjectID: subjectID,
	}
	if err := s.repo.EnrollStudent(ctx, enrollment); err != nil {
		return nil, fmt.Errorf("failed to enroll student: %w", err)
	}
	return enrollment, nil
}

func (s *SubjectService) ListStudents(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	return s.repo.ListStudentsBySubject(ctx, tenantID, subjectID)
}
