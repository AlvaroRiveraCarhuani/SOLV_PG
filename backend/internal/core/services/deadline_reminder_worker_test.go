package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockDeadlineReminderRepo struct {
	candidates []*domain.DeadlineReminderCandidate
	markedSent map[string]bool
}

func newMockDeadlineReminderRepo() *mockDeadlineReminderRepo {
	return &mockDeadlineReminderRepo{
		markedSent: make(map[string]bool),
	}
}

func (m *mockDeadlineReminderRepo) FindPendingReminders(ctx context.Context, windowStart, windowEnd time.Time) ([]*domain.DeadlineReminderCandidate, error) {
	return m.candidates, nil
}

func (m *mockDeadlineReminderRepo) MarkReminderSent(ctx context.Context, tenantID, exerciseID, studentID string) error {
	key := tenantID + ":" + exerciseID + ":" + studentID
	m.markedSent[key] = true
	return nil
}

func (m *mockDeadlineReminderRepo) IsReminderSent(ctx context.Context, tenantID, exerciseID, studentID string) (bool, error) {
	key := tenantID + ":" + exerciseID + ":" + studentID
	return m.markedSent[key], nil
}

func TestDeadlineReminderWorker_ProcessReminders_Success(t *testing.T) {
	now := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	dueDate := now.Add(11 * time.Hour) // dentro de la ventana de 12 horas

	repo := newMockDeadlineReminderRepo()
	repo.candidates = []*domain.DeadlineReminderCandidate{
		{
			TenantID:            "t-1",
			ExerciseID:          "ex-1",
			ExerciseTitle:       "Algoritmos de Ordenamiento",
			SubjectID:           "sub-1",
			SubjectName:         "Estructuras de Datos I",
			DueDate:             dueDate,
			StudentID:           "stu-1",
			StudentName:         "Lucas Silva",
			StudentEmail:        "lucas.silva@uab.edu.bo",
			HasStartedWorkspace: true,
		},
		{
			TenantID:            "t-1",
			ExerciseID:          "ex-1",
			ExerciseTitle:       "Algoritmos de Ordenamiento",
			SubjectID:           "sub-1",
			SubjectName:         "Estructuras de Datos I",
			DueDate:             dueDate,
			StudentID:           "stu-2",
			StudentName:         "Maria Rodriguez",
			StudentEmail:        "maria.rodriguez@uab.edu.bo",
			HasStartedWorkspace: false,
		},
		{
			TenantID:            "t-1",
			ExerciseID:          "ex-1",
			ExerciseTitle:       "Algoritmos de Ordenamiento",
			SubjectID:           "sub-1",
			SubjectName:         "Estructuras de Datos I",
			DueDate:             dueDate,
			StudentID:           "stu-3",
			StudentName:         "Sin Correo",
			StudentEmail:        "", // Debe ser ignorado limpiamente
			HasStartedWorkspace: false,
		},
	}

	tenantRepo := &mockTenantRepoForTeacher{
		tenant: &domain.Tenant{
			ID:   "t-1",
			Name: "Universidad Adventista de Bolivia",
			Slug: "uab",
			Config: []byte(`{
				"institution_name": "Universidad Adventista de Bolivia",
				"logo_url": "/assets/uab-logo.png",
				"base_domain": "solv.uab.edu.bo"
			}`),
		},
	}

	mailerMock := &mockMailer{}
	worker := services.NewDeadlineReminderWorker(repo, tenantRepo, mailerMock)

	sent, err := worker.ProcessReminders(context.Background(), now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sent != 2 {
		t.Fatalf("expected 2 reminders sent, got %d", sent)
	}

	if len(mailerMock.sentMessages) != 2 {
		t.Fatalf("expected 2 email messages dispatched, got %d", len(mailerMock.sentMessages))
	}

	// Verificar primer correo (estudiante con workspace iniciado)
	msg1 := mailerMock.sentMessages[0]
	if msg1.To != "lucas.silva@uab.edu.bo" {
		t.Errorf("expected recipient lucas.silva@uab.edu.bo, got %s", msg1.To)
	}
	if !strings.Contains(msg1.Subject, "Universidad Adventista de Bolivia") {
		t.Errorf("subject should contain institution name: %s", msg1.Subject)
	}
	if !strings.Contains(msg1.HTMLBody, "Registramos avances guardados") {
		t.Errorf("msg1 should contain progress mention")
	}
	if strings.Contains(msg1.Subject, "SOLV") || strings.Contains(msg1.HTMLBody, "SOLV") {
		t.Errorf("msg1 leaked SOLV internal brand")
	}

	// Verificar segundo correo (estudiante sin workspace iniciado)
	msg2 := mailerMock.sentMessages[1]
	if msg2.To != "maria.rodriguez@uab.edu.bo" {
		t.Errorf("expected recipient maria.rodriguez@uab.edu.bo, got %s", msg2.To)
	}
	if !strings.Contains(msg2.HTMLBody, "no registra actividad ni entrega enviada") {
		t.Errorf("msg2 should notify about no activity")
	}

	// Verificar deduplicación en el repositorio
	if !repo.markedSent["t-1:ex-1:stu-1"] {
		t.Errorf("expected stu-1 to be marked as sent")
	}
	if !repo.markedSent["t-1:ex-1:stu-2"] {
		t.Errorf("expected stu-2 to be marked as sent")
	}
	if repo.markedSent["t-1:ex-1:stu-3"] {
		t.Errorf("stu-3 with empty email should not be marked as sent")
	}
}

func TestDeadlineReminderWorker_NoCandidates(t *testing.T) {
	repo := newMockDeadlineReminderRepo()
	mailerMock := &mockMailer{}
	worker := services.NewDeadlineReminderWorker(repo, nil, mailerMock)

	sent, err := worker.ProcessReminders(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sent != 0 {
		t.Errorf("expected 0 sent, got %d", sent)
	}
	if len(mailerMock.sentMessages) != 0 {
		t.Errorf("expected 0 emails sent, got %d", len(mailerMock.sentMessages))
	}
}

func TestDeadlineReminderWorker_NilSafety(t *testing.T) {
	worker := services.NewDeadlineReminderWorker(nil, nil, nil)
	sent, err := worker.ProcessReminders(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("expected no error on nil deps, got %v", err)
	}
	if sent != 0 {
		t.Errorf("expected 0, got %d", sent)
	}
}
