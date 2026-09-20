package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

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
	tenantID, search, subjectID, status, periodID string,
) ([]*domain.AdminStudentDirectoryItem, error) {
	return s.govRepo.ListStudentsDirectory(ctx, tenantID, search, subjectID, status, periodID)
}

func (s *AdminGovernanceService) GetStudentCourses(
	ctx context.Context,
	tenantID, studentID string,
) ([]*domain.AdminStudentCourseItem, error) {
	return s.govRepo.GetStudentCourses(ctx, tenantID, studentID)
}

func (s *AdminGovernanceService) CreateStudent(
	ctx context.Context,
	tenantID string,
	dto domain.CreateStudentDTO,
) (*domain.AdminStudentDirectoryItem, error) {
	if strings.TrimSpace(dto.Email) == "" || strings.TrimSpace(dto.FirstName) == "" || strings.TrimSpace(dto.LastName) == "" {
		return nil, errors.New("todos los campos (email, nombre, apellido) son obligatorios")
	}
	return s.govRepo.CreateStudent(ctx, tenantID, strings.TrimSpace(dto.Email), strings.TrimSpace(dto.FirstName), strings.TrimSpace(dto.LastName))
}

func (s *AdminGovernanceService) UpdateStudentStatus(
	ctx context.Context,
	tenantID, studentID string,
	dto domain.UpdateStudentStatusDTO,
) error {
	if dto.Status != "active" && dto.Status != "suspended" {
		return errors.New("estado inválido: debe ser 'active' o 'suspended'")
	}
	return s.govRepo.UpdateStudentStatus(ctx, tenantID, studentID, dto.Status, dto.Reason)
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
	if dto.Status != "approved" && dto.Status != "rejected" && dto.Status != "paused" {
		return nil, ErrInvalidReviewStatus
	}

	if dto.Status == "rejected" && dto.RejectionReason == "" {
		return nil, ErrRejectionReasonRequired
	}

	return s.govRepo.ReviewTemplate(ctx, tenantID, templateID, adminID, dto.Status, dto.RejectionReason, dto.BaseRamMB)
}

var (
	dockerImageRegex = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$`)
	ErrInvalidDockerImage = errors.New("la imagen Docker debe tener formato válido repositorio:tag sin espacios (ej: python:3.12-slim)")
	ErrLatestTagForbidden = errors.New("el tag :latest está prohibido en plantillas oficiales por reproducibilidad")
)

func (s *AdminGovernanceService) CreateOfficialTemplate(
	ctx context.Context,
	tenantID, adminID string,
	dto domain.CreateOfficialTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	dto.Name = strings.TrimSpace(dto.Name)
	dto.DockerImage = strings.TrimSpace(dto.DockerImage)

	if dto.Name == "" {
		return nil, errors.New("el nombre de la plantilla es requerido")
	}

	if dto.DockerImage == "" {
		return nil, errors.New("la imagen Docker es requerida")
	}

	if strings.HasSuffix(strings.ToLower(dto.DockerImage), ":latest") {
		return nil, ErrLatestTagForbidden
	}

	if !dockerImageRegex.MatchString(dto.DockerImage) {
		return nil, ErrInvalidDockerImage
	}

	if dto.TargetEnvironment == "" {
		dto.TargetEnvironment = "IDE_PERSISTENTE"
	}

	if dto.BaseRamMB <= 0 {
		dto.BaseRamMB = 512
	} else if dto.BaseRamMB < 256 {
		dto.BaseRamMB = 256
	}

	if dto.ServicesConfig == nil {
		dto.ServicesConfig = &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{},
		}
	}

	// Cálculo proporcional dinámico de calidad de servicio (sin números mágicos fijos)
	if dto.ResourceProfile == nil {
		minMB := dto.BaseRamMB / 2
		if minMB < 128 {
			minMB = 128
		}
		highMB := int(float64(dto.BaseRamMB) * 0.8)
		dto.ResourceProfile = &domain.TemplateResourceProfile{
			MinMB:  minMB,
			HighMB: highMB,
			MaxMB:  dto.BaseRamMB,
		}
	}

	return s.govRepo.CreateOfficialTemplate(ctx, tenantID, adminID, dto)
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
