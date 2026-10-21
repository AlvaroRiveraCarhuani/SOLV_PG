package services

import (
	"context"
	"fmt"

	"solv-backend/internal/core/domain"
)

func (s *TeacherService) GetCourseAnalytics(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseAnalytics, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}

	analytics, err := s.repo.GetCourseAnalytics(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, fmt.Errorf("error al calcular analitica del curso: %w", err)
	}
	if analytics == nil {
		analytics = &domain.CourseAnalytics{
			DifficultyDistribution:        make(map[string]domain.DifficultyMetric),
			TopTags:                       make([]domain.TagMetric, 0),
			MostFailedCases:               make([]domain.FailedCaseMetric, 0),
			AvgResolutionTimeByDifficulty: make(map[string]int),
			SubmissionsTimeline:           make([]domain.TimelineMetric, 0),
		}
	}
	return analytics, nil
}
