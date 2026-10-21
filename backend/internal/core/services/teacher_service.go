package services

import (
	"context"
	"fmt"

	"solv-backend/internal/core/domain"
)

type TeacherService struct {
	repo          domain.TeacherRepository
	subRepo       domain.SubmissionRepository
	keystrokeRepo domain.KeystrokeRepository
	evalService   *EvaluationService
	mirrorService *TerminalMirrorService
}

func NewTeacherService(repo domain.TeacherRepository, subRepo ...domain.SubmissionRepository) *TeacherService {
	var sRepo domain.SubmissionRepository
	if len(subRepo) > 0 {
		sRepo = subRepo[0]
	}
	return &TeacherService{
		repo:          repo,
		subRepo:       sRepo,
		mirrorService: NewTerminalMirrorService(nil),
	}
}

func (s *TeacherService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

func (s *TeacherService) SetWorkspaceOrchestrator(orch domain.WorkspaceOrchestrator) {
	s.mirrorService = NewTerminalMirrorService(orch)
}

func (s *TeacherService) SetSubmissionRepository(subRepo domain.SubmissionRepository) {
	s.subRepo = subRepo
}

func (s *TeacherService) SetEvaluationService(evalService *EvaluationService) {
	s.evalService = evalService
}

func (s *TeacherService) GetCoursesSummary(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherCourseSummary, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	courses, err := s.repo.GetCoursesSummary(ctx, tenantID, teacherID)
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
