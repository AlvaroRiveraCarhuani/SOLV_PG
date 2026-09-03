package services

import (
	"context"
	"errors"
	"fmt"

	"solv-backend/internal/core/domain"
)

var (
	ErrReasonTooShort       = errors.New("justification reason must have at least 10 characters")
	ErrTeacherNotFoundOrRole = errors.New("assigned user does not exist or does not have teacher role")
)

type AdminGovernanceService struct {
	subjectRepo domain.SubjectRepository
	govRepo     domain.AdminGovernanceRepository
}

func NewAdminGovernanceService(
	subjectRepo domain.SubjectRepository,
	govRepo domain.AdminGovernanceRepository,
) *AdminGovernanceService {
	return &AdminGovernanceService{
		subjectRepo: subjectRepo,
		govRepo:     govRepo,
	}
}

func (s *AdminGovernanceService) ReassignCourse(ctx context.Context, tenantID, subjectID string, dto domain.ReassignCourseDTO) (*domain.Subject, error) {
	if dto.NewTeacherID == "" {
		return nil, fmt.Errorf("new_teacher_id is required")
	}

	// 1. Validar que la materia existe en el tenant
	subject, err := s.subjectRepo.GetByID(ctx, tenantID, subjectID)
	if err != nil {
		return nil, err
	}

	// 2. Validar que el nuevo docente existe y tiene rol teacher o admin
	if s.govRepo != nil {
		isValid, err := s.govRepo.ValidateTeacherRole(ctx, tenantID, dto.NewTeacherID)
		if err != nil || !isValid {
			return nil, ErrTeacherNotFoundOrRole
		}
	}

	// 3. Reasignar materia
	if err := s.subjectRepo.ReassignTeacher(ctx, tenantID, subjectID, dto.NewTeacherID); err != nil {
		return nil, err
	}

	subject.TeacherID = &dto.NewTeacherID
	return subject, nil
}

func (s *AdminGovernanceService) ListStudents(
	ctx context.Context,
	tenantID, search, subjectID, status string,
) ([]*domain.AdminStudentDirectoryItem, error) {
	return s.govRepo.ListStudentsDirectory(ctx, tenantID, search, subjectID, status)
}

func (s *AdminGovernanceService) ResetStudentOOM(
	ctx context.Context,
	tenantID, studentID string,
	dto domain.ResetOOMDTO,
) (*domain.ResetOOMResult, error) {
	if len(dto.Reason) < 10 {
		return nil, ErrReasonTooShort
	}

	affectedRows, err := s.govRepo.ResetStudentOOM(ctx, tenantID, studentID)
	if err != nil {
		return nil, err
	}

	return &domain.ResetOOMResult{
		StudentID:            studentID,
		WorkspacesResetCount: affectedRows,
		Message:              "Penalizaciones OOM reseteadas exitosamente",
	}, nil
}
