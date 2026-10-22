package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

type TeacherService struct {
	repo                          domain.TeacherRepository
	subRepo                       domain.SubmissionRepository
	keystrokeRepo                  domain.KeystrokeRepository
	evalService                   *EvaluationService
	mirrorService                 *TerminalMirrorService
	metricsSnapshotService        *MetricsSnapshotService
}

func NewTeacherService(
	repo domain.TeacherRepository,
	subRepo domain.SubmissionRepository,
) *TeacherService {
	return &TeacherService{
		repo:                          repo,
		subRepo:                       subRepo,
		mirrorService:                 NewTerminalMirrorService(nil),
	}
}

func (s *TeacherService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

func (s *TeacherService) SetWorkspaceOrchestrator(orch domain.WorkspaceOrchestrator) {
	s.mirrorService = NewTerminalMirrorService(orch)
}

func (s *TeacherService) SetMetricsSnapshotService(metricsSnapshotService *MetricsSnapshotService) {
	s.metricsSnapshotService = metricsSnapshotService
}

func (s *TeacherService) SetEvaluationService(evalService *EvaluationService) {
	s.evalService = evalService
}

func (s *TeacherService) GetCoursesSummary(ctx context.Context, tenantID, teacherID, periodID string) ([]*domain.TeacherCourseSummary, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	// Validar period_id si viene vacío se omite el filtro
	// Si viene no vacío, validar que sea un UUID válido
	if periodID != "" {
		if _, err := uuid.Parse(periodID); err != nil {
			return nil, domain.ErrInvalidTenant
		}
	}

	courses, err := s.repo.GetCoursesSummary(ctx, tenantID, teacherID, periodID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener resumen de cursos: %w", err)
	}
	if courses == nil {
		courses = make([]*domain.TeacherCourseSummary, 0)
	}
	return courses, nil
}

func (s *TeacherService) GetAttentionWidget(ctx context.Context, tenantID, teacherID string) (*domain.TeacherAttentionWidget, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	widget, err := s.repo.GetAttentionWidget(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener widget de atencion: %w", err)
	}
	return widget, nil
}

func (s *TeacherService) GetCourseLabsStats(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.TeacherLabStats, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}
	stats, err := s.repo.GetCourseLabsStats(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = make([]*domain.TeacherLabStats, 0)
	}
	return stats, nil
}

func (s *TeacherService) GetLiveWorkspaceSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LiveWorkspaceSession, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	sessions, err := s.repo.ListLiveWorkspaceSessions(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener sesiones en vivo: %w", err)
	}
	if sessions == nil {
		sessions = make([]*domain.LiveWorkspaceSession, 0)
	}
	return sessions, nil
}

