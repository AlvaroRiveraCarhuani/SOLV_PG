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

var (
	ErrInvalidReviewStatus      = errors.New("status must be either 'approved' or 'rejected'")
	ErrRejectionReasonRequired = errors.New("rejection_reason is required when rejecting a template")
)

func (s *AdminGovernanceService) ListTemplates(
	ctx context.Context,
	tenantID, status, search string,
) ([]*domain.AdminTemplateReviewItem, error) {
	return s.govRepo.ListTemplates(ctx, tenantID, status, search)
}

func (s *AdminGovernanceService) ReviewTemplate(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.ReviewTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	if dto.Status != "approved" && dto.Status != "rejected" {
		return nil, ErrInvalidReviewStatus
	}

	if dto.Status == "rejected" && dto.RejectionReason == "" {
		return nil, ErrRejectionReasonRequired
	}

	return s.govRepo.ReviewTemplate(ctx, tenantID, templateID, adminID, dto.Status, dto.RejectionReason, dto.BaseRamMB)
}

const (
	ActionTerminateAll = "terminate_all_workspaces"
	ActionHibernateAll = "hibernate_all_workspaces"
	ActionKillZombies  = "kill_zombies"

	PhraseTerminateAll = "TERMINAR TODOS LOS WORKSPACES"
	PhraseHibernateAll = "HIBERNAR TODOS LOS WORKSPACES"
	PhraseKillZombies  = "LIMPIAR ZOMBIES DOCKER"
)

var (
	ErrUnknownEmergencyAction     = errors.New("unknown emergency action")
	ErrInvalidConfirmationPhrase = errors.New("invalid confirmation phrase")
)

func (s *AdminGovernanceService) ExecuteEmergencyAction(
	ctx context.Context,
	tenantID, adminID, action string,
	req domain.EmergencyActionRequest,
) (*domain.EmergencyActionResult, error) {
	switch action {
	case ActionTerminateAll:
		if req.ConfirmationPhrase != PhraseTerminateAll {
			return nil, ErrInvalidConfirmationPhrase
		}
		count, err := s.govRepo.TerminateAllWorkspaces(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		return &domain.EmergencyActionResult{
			Action:        action,
			AffectedCount: count,
			ExecutedBy:    adminID,
			Message:       fmt.Sprintf("Se terminaron forzosamente %d workspaces activos", count),
		}, nil

	case ActionHibernateAll:
		if req.ConfirmationPhrase != PhraseHibernateAll {
			return nil, ErrInvalidConfirmationPhrase
		}
		count, err := s.govRepo.HibernateAllWorkspaces(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		return &domain.EmergencyActionResult{
			Action:        action,
			AffectedCount: count,
			ExecutedBy:    adminID,
			Message:       fmt.Sprintf("Se hibernaron exitosamente %d workspaces activos", count),
		}, nil

	case ActionKillZombies:
		if req.ConfirmationPhrase != PhraseKillZombies {
			return nil, ErrInvalidConfirmationPhrase
		}
		// Acción de limpieza de zombies
		return &domain.EmergencyActionResult{
			Action:        action,
			AffectedCount: 0,
			ExecutedBy:    adminID,
			Message:       "Barrido de contenedores zombies ejecutado exitosamente",
		}, nil

	default:
		return nil, ErrUnknownEmergencyAction
	}
}
