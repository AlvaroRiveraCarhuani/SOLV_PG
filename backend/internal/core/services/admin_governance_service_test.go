package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockAdminGovernanceRepo struct {
	lastCreatedDTO *domain.CreateOfficialTemplateDTO
}

func (m *mockAdminGovernanceRepo) ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) GetStudentCourses(ctx context.Context, tenantID, studentID string) ([]*domain.AdminStudentCourseItem, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) CreateStudent(ctx context.Context, tenantID, email, firstName, lastName string) (*domain.AdminStudentDirectoryItem, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) UpdateStudentStatus(ctx context.Context, tenantID, studentID, status, reason string) error {
	return nil
}
func (m *mockAdminGovernanceRepo) ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error) {
	return 0, nil
}
func (m *mockAdminGovernanceRepo) ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error) {
	return true, nil
}
func (m *mockAdminGovernanceRepo) ListTemplates(ctx context.Context, tenantID, status, search string) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) ReviewTemplate(ctx context.Context, tenantID, templateID, adminID, status, rejectionReason string, baseRamMB *int) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) CreateOfficialTemplate(ctx context.Context, tenantID, adminID string, dto domain.CreateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	m.lastCreatedDTO = &dto
	return &domain.AdminTemplateReviewItem{
		ID:                "test-tpl-id",
		Name:              dto.Name,
		DockerImage:       dto.DockerImage,
		BaseRamMB:         dto.BaseRamMB,
		TargetEnvironment: dto.TargetEnvironment,
		ServicesConfig:    *dto.ServicesConfig,
		ResourceProfile:   *dto.ResourceProfile,
	}, nil
}
func (m *mockAdminGovernanceRepo) TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockAdminGovernanceRepo) HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (m *mockAdminGovernanceRepo) ListPendingAuditTemplates(ctx context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) UpdateAuditResults(ctx context.Context, templateID string, smokeStatus, smokeOutput, secStatus string, cveCritical, cveHigh int, secReportJSON []byte, finalStatus string) error {
	return nil
}

func TestCreateOfficialTemplate_DynamicProportionalMQoS(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.CreateOfficialTemplateDTO{
		Name:              "Node FullStack + Postgres",
		DockerImage:       "node:20-slim",
		BaseRamMB:         512,
		TargetEnvironment: "IDE_PERSISTENTE",
		ServicesConfig: &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{
				{Category: "database", Engine: "postgres"},
			},
		},
	}

	item, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dto)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if item.BaseRamMB != 512 {
		t.Errorf("esperado BaseRamMB = 512, obtenido: %d", item.BaseRamMB)
	}
	// HighMB debe ser el 80% proporcional (512 * 0.8 = 409)
	if item.ResourceProfile.HighMB != 409 {
		t.Errorf("esperado HighMB = 409 (80%%), obtenido: %d", item.ResourceProfile.HighMB)
	}
	if item.ResourceProfile.MaxMB != 512 {
		t.Errorf("esperado MaxMB = 512 (100%%), obtenido: %d", item.ResourceProfile.MaxMB)
	}
	if item.ResourceProfile.MinMB != 256 {
		t.Errorf("esperado MinMB = 256 (50%%), obtenido: %d", item.ResourceProfile.MinMB)
	}
	if len(item.ServicesConfig.Services) != 1 || item.ServicesConfig.Services[0].Engine != "postgres" {
		t.Errorf("esperado que el servicio postgres esté presente en la lista de servicios")
	}
}

func TestCreateOfficialTemplate_DefaultRAMFallback(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.CreateOfficialTemplateDTO{
		Name:        "Python Base",
		DockerImage: "python:3.12-slim",
		BaseRamMB:   0, // No especificado
	}

	item, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dto)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	// Debe caer al fallback seguro de 512 MB
	if item.BaseRamMB != 512 {
		t.Errorf("esperado fallback BaseRamMB = 512, obtenido: %d", item.BaseRamMB)
	}
	if item.ResourceProfile.MaxMB != 512 {
		t.Errorf("esperado MaxMB = 512, obtenido: %d", item.ResourceProfile.MaxMB)
	}
}

func TestCreateOfficialTemplate_DockerImageValidation(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	// Caso 1: Tag :latest prohibido
	dtoLatest := domain.CreateOfficialTemplateDTO{
		Name:        "Imagen Con Latest",
		DockerImage: "python:latest",
		BaseRamMB:   512,
	}
	_, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoLatest)
	if err == nil || !errors.Is(err, services.ErrLatestTagForbidden) {
		t.Errorf("esperado error ErrLatestTagForbidden, obtenido: %v", err)
	}

	// Caso 2: Formato roto con espacios (ej: asdasd : asdasd)
	dtoInvalid := domain.CreateOfficialTemplateDTO{
		Name:        "Imagen Rota",
		DockerImage: "asdasdas : asdasdas",
		BaseRamMB:   512,
	}
	_, err = svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoInvalid)
	if err == nil || !errors.Is(err, services.ErrInvalidDockerImage) {
		t.Errorf("esperado error ErrInvalidDockerImage, obtenido: %v", err)
	}

	// Caso 3: Imagen sin tag
	dtoNoTag := domain.CreateOfficialTemplateDTO{
		Name:        "Imagen Sin Tag",
		DockerImage: "python",
		BaseRamMB:   512,
	}
	_, err = svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoNoTag)
	if err == nil || !errors.Is(err, services.ErrInvalidDockerImage) {
		t.Errorf("esperado error ErrInvalidDockerImage por falta de tag, obtenido: %v", err)
	}
}
