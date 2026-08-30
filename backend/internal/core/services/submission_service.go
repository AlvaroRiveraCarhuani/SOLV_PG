package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
)

type SubmissionService struct {
	repo domain.SubmissionRepository
}

func NewSubmissionService(repo domain.SubmissionRepository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

type CreateSubmissionDTO struct {
	ExerciseID      string          `json:"exercise_id"`
	StudentID       string          `json:"student_id"`
	WorkspaceID     *string         `json:"workspace_id,omitempty"`
	Code            string          `json:"code"`
	Verdict         string          `json:"verdict"`
	ASTResult       json.RawMessage `json:"ast_result"`
	ExecutionTimeMS int             `json:"execution_time_ms"`
	MemoryUsedMB    int             `json:"memory_used_mb"`
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, tenantID string, dto CreateSubmissionDTO) (*domain.Submission, error) {
	if dto.ExerciseID == "" || dto.StudentID == "" || dto.Verdict == "" {
		return nil, errors.New("exercise_id, student_id and verdict are required")
	}
	if len(dto.ASTResult) == 0 {
		dto.ASTResult = json.RawMessage("{}")
	}
	sub := &domain.Submission{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		ExerciseID:      dto.ExerciseID,
		StudentID:       dto.StudentID,
		WorkspaceID:     dto.WorkspaceID,
		Code:            dto.Code,
		Verdict:         dto.Verdict,
		ASTResult:       dto.ASTResult,
		ExecutionTimeMS: dto.ExecutionTimeMS,
		MemoryUsedMB:    dto.MemoryUsedMB,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}
	return sub, nil
}

func (s *SubmissionService) GetSubmissionsForExercise(ctx context.Context, tenantID, exerciseID, userID, userRole string) ([]*domain.Submission, error) {
	if userRole == "student" {
		return s.repo.ListByExerciseAndStudent(ctx, tenantID, exerciseID, userID)
	}
	return s.repo.ListByExercise(ctx, tenantID, exerciseID)
}

func (s *SubmissionService) GetSubmissionByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *SubmissionService) OverrideSubmission(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	if verdict == "" {
		return errors.New("verdict is required")
	}
	if reason == "" {
		return errors.New("override_reason is required")
	}
	return s.repo.UpdateOverride(ctx, tenantID, id, verdict, reason, score, gradedBy)
}

