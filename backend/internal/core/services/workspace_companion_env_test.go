package services

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

type mockSubmissionRepoForEnv struct {
	submissions []*domain.Submission
}

func (m *mockSubmissionRepoForEnv) Create(ctx context.Context, sub *domain.Submission) error {
	return nil
}

func (m *mockSubmissionRepoForEnv) GetByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	return nil, nil
}

func (m *mockSubmissionRepoForEnv) ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*domain.Submission, error) {
	return m.submissions, nil
}

func (m *mockSubmissionRepoForEnv) ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (m *mockSubmissionRepoForEnv) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (m *mockSubmissionRepoForEnv) UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	return nil
}

func TestBuildCompanionEnvVars_InjectsExpectedValues(t *testing.T) {
	mockRepo := &mockSubmissionRepoForEnv{
		submissions: []*domain.Submission{
			{ID: "sub-12345"},
		},
	}

	service := &WorkspaceService{
		submissionRepo: mockRepo,
	}

	ctx := context.WithValue(context.Background(), domain.UserTokenKey, "jwt-token-valeria")

	t.Setenv("SOLV_API_URL", "http://test-host:3000")

	envs := service.buildCompanionEnvVars(ctx, "tenant-1", "sub-1", "ex-1", "student-1")

	var foundAPI, foundToken, foundSub, foundSubject, foundExercise bool
	for _, env := range envs {
		if env == "SOLV_API_URL=http://test-host:3000" {
			foundAPI = true
		}
		if env == "SOLV_STUDENT_TOKEN=jwt-token-valeria" {
			foundToken = true
		}
		if env == "SOLV_SUBMISSION_ID=sub-12345" {
			foundSub = true
		}
		if env == "SOLV_SUBJECT_ID=sub-1" {
			foundSubject = true
		}
		if env == "SOLV_EXERCISE_ID=ex-1" {
			foundExercise = true
		}
	}

	if !foundAPI {
		t.Errorf("esperado SOLV_API_URL inyectado en envVars: %+v", envs)
	}
	if !foundToken {
		t.Errorf("esperado SOLV_STUDENT_TOKEN inyectado en envVars: %+v", envs)
	}
	if !foundSub {
		t.Errorf("esperado SOLV_SUBMISSION_ID inyectado en envVars: %+v", envs)
	}
	if !foundSubject {
		t.Errorf("esperado SOLV_SUBJECT_ID inyectado en envVars: %+v", envs)
	}
	if !foundExercise {
		t.Errorf("esperado SOLV_EXERCISE_ID inyectado en envVars: %+v", envs)
	}
}

func TestBuildCompanionEnvVars_FallbackWithoutSubmission(t *testing.T) {
	mockRepo := &mockSubmissionRepoForEnv{
		submissions: []*domain.Submission{},
	}

	service := &WorkspaceService{
		submissionRepo: mockRepo,
	}

	ctx := context.Background()
	t.Setenv("SOLV_API_URL", "")

	envs := service.buildCompanionEnvVars(ctx, "tenant-1", "", "ex-1", "student-1")

	var foundAPI bool
	for _, env := range envs {
		if strings.HasPrefix(env, "SOLV_API_URL=") {
			foundAPI = true
		}
		if strings.HasPrefix(env, "SOLV_SUBMISSION_ID=") {
			t.Errorf("no se esperaba SOLV_SUBMISSION_ID sin entregas: %s", env)
		}
		if strings.HasPrefix(env, "SOLV_STUDENT_TOKEN=") {
			t.Errorf("no se esperaba SOLV_STUDENT_TOKEN sin token en context: %s", env)
		}
	}

	if !foundAPI {
		t.Errorf("esperado SOLV_API_URL por defecto")
	}
}
