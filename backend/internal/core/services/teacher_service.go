package services

import (
	"context"
	"fmt"

	"solv-backend/internal/core/domain"
)

type TeacherService struct {
	repo domain.TeacherRepository
}

func NewTeacherService(repo domain.TeacherRepository) *TeacherService {
	return &TeacherService{repo: repo}
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
