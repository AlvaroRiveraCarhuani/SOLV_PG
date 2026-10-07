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
	repo          domain.SubmissionRepository
	keystrokeRepo domain.KeystrokeRepository
}

func NewSubmissionService(repo domain.SubmissionRepository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

func (s *SubmissionService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

func (s *SubmissionService) SaveKeystrokeEvents(ctx context.Context, tenantID, submissionID string, events []domain.SubmissionKeystrokeEvent) (int, error) {
	if submissionID == "" {
		return 0, errors.New("submission_id is required")
	}
	if len(events) == 0 {
		return 0, nil
	}
	if len(events) > 10000 {
		return 0, errors.New("máximo 10,000 eventos permitidos por lote")
	}
	if s.keystrokeRepo == nil {
		return len(events), nil
	}
	if _, err := s.repo.GetByID(ctx, tenantID, submissionID); err != nil {
		return 0, fmt.Errorf("submission not found: %w", err)
	}
	if err := s.keystrokeRepo.SaveBatch(ctx, submissionID, events); err != nil {
		return 0, fmt.Errorf("failed to save keystroke events: %w", err)
	}
	return len(events), nil
}

type CreateSubmissionDTO struct {
	ExerciseID         string          `json:"exercise_id"`
	StudentID          string          `json:"student_id"`
	WorkspaceID        *string         `json:"workspace_id,omitempty"`
	Code               string          `json:"code"`
	Verdict            string          `json:"verdict"`
	ASTResult          json.RawMessage `json:"ast_result"`
	GeneratedCases     json.RawMessage `json:"generated_cases,omitempty"`
	ExecutionTimeMS    int             `json:"execution_time_ms"`
	MemoryUsedMB       int             `json:"memory_used_mb"`
	ComplexityAnalysis json.RawMessage `json:"complexity_analysis,omitempty"`
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, tenantID string, dto CreateSubmissionDTO) (*domain.Submission, error) {
	if dto.ExerciseID == "" || dto.StudentID == "" || dto.Verdict == "" {
		return nil, errors.New("exercise_id, student_id and verdict are required")
	}
	if len(dto.ASTResult) == 0 {
		dto.ASTResult = json.RawMessage("{}")
	}
	sub := &domain.Submission{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		ExerciseID:         dto.ExerciseID,
		StudentID:          dto.StudentID,
		WorkspaceID:        dto.WorkspaceID,
		Code:               dto.Code,
		Verdict:            dto.Verdict,
		ASTResult:          dto.ASTResult,
		GeneratedCases:     dto.GeneratedCases,
		ComplexityAnalysis: dto.ComplexityAnalysis,
		ExecutionTimeMS:    dto.ExecutionTimeMS,
		MemoryUsedMB:       dto.MemoryUsedMB,
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
