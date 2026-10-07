package services_test

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockSubjectTestRepo struct {
	createdSubjects  []*domain.Subject
	templateStatuses map[string]string
}

func (m *mockSubjectTestRepo) Create(ctx context.Context, s *domain.Subject) error {
	m.createdSubjects = append(m.createdSubjects, s)
	return nil
}
func (m *mockSubjectTestRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Subject, error) {
	return nil, nil
}
func (m *mockSubjectTestRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	return nil, nil
}
func (m *mockSubjectTestRepo) EnrollStudent(ctx context.Context, e *domain.Enrollment) error {
	return nil
}
func (m *mockSubjectTestRepo) ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	return nil, nil
}
func (m *mockSubjectTestRepo) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Subject, error) {
	return nil, nil
}
func (m *mockSubjectTestRepo) ReassignTeacher(ctx context.Context, tenantID, subjectID, newTeacherID string) error {
	return nil
}
func (m *mockSubjectTestRepo) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	return nil
}
func (m *mockSubjectTestRepo) Update(ctx context.Context, tenantID, subjectID, name, code string) error {
	return nil
}
func (m *mockSubjectTestRepo) GetTemplateStatus(ctx context.Context, templateID string) (string, error) {
	status, ok := m.templateStatuses[templateID]
	if !ok {
		return "pending", nil
	}
	return status, nil
}

func TestSubjectService_TemplateIntegrityEnforcement(t *testing.T) {
	repo := &mockSubjectTestRepo{
		templateStatuses: map[string]string{
			"tpl-pending":  "PENDIENTE_AUDITORIA",
			"tpl-rejected": "RECHAZADA",
			"tpl-approved": "APROBADA",
		},
	}
	svc := services.NewSubjectService(repo)

	// 1. Asignar plantilla en PENDIENTE_AUDITORIA debe fallar
	pendingTpl := "tpl-pending"
	_, err := svc.CreateSubjectWithDetails(context.Background(), "t1", "Algoritmos", "CS101", nil, nil, nil, &pendingTpl)
	if err == nil {
		t.Fatal("esperado error al asignar plantilla en auditoría pendiente, pero no ocurrió ninguno")
	}
	if !strings.Contains(err.Error(), "no ha sido aprobada por la auditoría de seguridad") {
		t.Fatalf("mensaje de error inesperado: %v", err)
	}

	// 2. Asignar plantilla en RECHAZADA debe fallar
	rejectedTpl := "tpl-rejected"
	_, err = svc.CreateSubjectWithDetails(context.Background(), "t1", "Estructuras", "CS102", nil, nil, nil, &rejectedTpl)
	if err == nil {
		t.Fatal("esperado error al asignar plantilla rechazada, pero no ocurrió ninguno")
	}

	// 3. Asignar plantilla en APROBADA debe pasar
	approvedTpl := "tpl-approved"
	subj, err := svc.CreateSubjectWithDetails(context.Background(), "t1", "Sistemas Operativos", "CS201", nil, nil, nil, &approvedTpl)
	if err != nil {
		t.Fatalf("error inesperado al crear materia con plantilla aprobada: %v", err)
	}
	if subj.TemplateID == nil || *subj.TemplateID != "tpl-approved" {
		t.Fatalf("esperado template_id tpl-approved, obtenido %v", subj.TemplateID)
	}

	// 4. Crear sin plantilla debe pasar
	subjNoTpl, err := svc.CreateSubjectWithDetails(context.Background(), "t1", "Matemática", "MAT101", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("error inesperado al crear materia sin plantilla: %v", err)
	}
	if subjNoTpl.TemplateID != nil {
		t.Fatalf("esperado template_id nil, obtenido %v", subjNoTpl.TemplateID)
	}
}
