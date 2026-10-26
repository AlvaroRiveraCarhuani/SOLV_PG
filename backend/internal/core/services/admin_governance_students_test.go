package services

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
)

type mockStudentBatchGovRepo struct {
	domain.AdminGovernanceRepository
	createdCalled       int
	enrolledCalled      int
	skippedCalled       int
	lastCreateEmails    []string
	lastEnrollEmails    []string
	lastEnrollSubjectID string
	createResult        int
	enrollResult        int
	skippedResult       int
}

func (m *mockStudentBatchGovRepo) BatchCreateStudents(ctx context.Context, tenantID string, emails []string) (int, error) {
	m.createdCalled++
	m.lastCreateEmails = emails
	return m.createResult, nil
}

func (m *mockStudentBatchGovRepo) EnrollStudentsByEmails(ctx context.Context, tenantID, subjectID string, emails []string) (int, int, error) {
	m.enrolledCalled++
	m.lastEnrollSubjectID = subjectID
	m.lastEnrollEmails = emails
	return m.enrollResult, m.skippedResult, nil
}

func TestCleanEmails(t *testing.T) {
	cleaned := cleanEmails([]string{
		"  Student1@UAB.edu.bo ",
		"student1@uab.edu.bo", // duplicado tras normalizar
		"not-an-email",
		"",
		"student2@uab.edu.bo",
	})
	if len(cleaned) != 2 {
		t.Fatalf("expected 2 valid emails, got %d: %v", len(cleaned), cleaned)
	}
	if cleaned[0] != "student1@uab.edu.bo" {
		t.Errorf("expected normalized student1 email, got %q", cleaned[0])
	}
	if cleaned[1] != "student2@uab.edu.bo" {
		t.Errorf("expected student2 email, got %q", cleaned[1])
	}
}

func TestBatchCreateStudentsDelegatesCleaned(t *testing.T) {
	repo := &mockStudentBatchGovRepo{createResult: 2}
	svc := NewAdminGovernanceService(nil, repo)

	count, err := svc.BatchCreateStudents(context.Background(), "tenant-1", []string{
		"A@uab.edu.bo",
		"a@uab.edu.bo",
		"bad-email",
		"B@uab.edu.bo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 created, got %d", count)
	}
	if repo.createdCalled != 1 {
		t.Errorf("expected repo to be called once, got %d", repo.createdCalled)
	}
	if len(repo.lastCreateEmails) != 2 {
		t.Errorf("expected 2 cleaned emails passed to repo, got %v", repo.lastCreateEmails)
	}
}

func TestEnrollStudentsByEmailsRequiresSubject(t *testing.T) {
	repo := &mockStudentBatchGovRepo{}
	svc := NewAdminGovernanceService(nil, repo)

	_, _, err := svc.EnrollStudentsByEmails(context.Background(), "tenant-1", "", []string{"a@uab.edu.bo"})
	if err == nil {
		t.Fatal("expected error when subject id is empty")
	}
	if repo.enrolledCalled != 0 {
		t.Error("repo should not be called with empty subject id")
	}
}

func TestEnrollStudentsByEmailsDelegates(t *testing.T) {
	repo := &mockStudentBatchGovRepo{enrollResult: 3, skippedResult: 1}
	svc := NewAdminGovernanceService(nil, repo)

	enrolled, skipped, err := svc.EnrollStudentsByEmails(context.Background(), "tenant-1", "subject-9", []string{
		"a@uab.edu.bo",
		"b@uab.edu.bo",
		"c@uab.edu.bo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enrolled != 3 || skipped != 1 {
		t.Errorf("expected 3 enrolled and 1 skipped, got %d/%d", enrolled, skipped)
	}
	if repo.lastEnrollSubjectID != "subject-9" {
		t.Errorf("unexpected subject id: %q", repo.lastEnrollSubjectID)
	}
}

func TestRosterEnrollCombines(t *testing.T) {
	repo := &mockStudentBatchGovRepo{createResult: 2, enrollResult: 5}
	svc := NewAdminGovernanceService(nil, repo)

	result, err := svc.RosterEnroll(context.Background(), "tenant-1", "subject-9", []string{
		"a@uab.edu.bo",
		"b@uab.edu.bo",
		"c@uab.edu.bo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CreatedStudents != 2 {
		t.Errorf("expected 2 created, got %d", result.CreatedStudents)
	}
	if result.EnrolledStudents != 5 {
		t.Errorf("expected 5 enrolled, got %d", result.EnrolledStudents)
	}
	if repo.createdCalled != 1 || repo.enrolledCalled != 1 {
		t.Errorf("expected both repo operations called once, got create=%d enroll=%d", repo.createdCalled, repo.enrolledCalled)
	}
}

var _ domain.AdminGovernanceRepository = (*mockStudentBatchGovRepo)(nil)
