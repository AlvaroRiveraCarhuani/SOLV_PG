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
	return s.CreateSubjectWithDetails(ctx, tenantID, name, code, nil, nil, classroomCourseID, nil)
}

func (s *SubjectService) CreateSubjectWithDetails(ctx context.Context, tenantID, name, code string, teacherID, academicPeriodID, classroomCourseID, templateID *string) (*domain.Subject, error) {
	if name == "" || code == "" {
		return nil, errors.New("name and code are required")
	}
	if templateID != nil && *templateID != "" {
		status, err := s.repo.GetTemplateStatus(ctx, *templateID)
		if err != nil {
			return nil, fmt.Errorf("la plantilla de laboratorio especificada no existe: %w", err)
		}
		if status != "approved" && status != "APROBADA" {
			return nil, fmt.Errorf("la plantilla seleccionada no ha sido aprobada por la auditoría de seguridad (estado actual: %s)", status)
		}
	}
	subject := &domain.Subject{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Name:              name,
		Code:              code,
		TeacherID:         teacherID,
		AcademicPeriodID:  academicPeriodID,
		ClassroomCourseID: classroomCourseID,
		TemplateID:        templateID,
	}
	if err := s.repo.Create(ctx, subject); err != nil {
		return nil, fmt.Errorf("failed to create subject: %w", err)
	}
	return subject, nil
}

func (s *SubjectService) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	return s.repo.ArchiveSubject(ctx, tenantID, subjectID, isArchived)
}

func (s *SubjectService) UpdateSubject(ctx context.Context, tenantID, subjectID, name, code string) error {
	if name == "" || code == "" {
		return errors.New("name and code are required")
	}
	return s.repo.Update(ctx, tenantID, subjectID, name, code)
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
