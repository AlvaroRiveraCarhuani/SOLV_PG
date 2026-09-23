package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockAdminGovernanceRepo struct {
	lastCreatedDTO      *domain.CreateOfficialTemplateDTO
	lastReviewedStatus  string
	lastRejectionReason string
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
	m.lastReviewedStatus = status
	m.lastRejectionReason = rejectionReason
	ram := 512
	if baseRamMB != nil && *baseRamMB > 0 {
		ram = *baseRamMB
	}
	profile := domain.DeriveResourceProfile(ram)
	return &domain.AdminTemplateReviewItem{
		ID:              templateID,
		Status:          status,
		RejectionReason: rejectionReason,
		BaseRamMB:       ram,
		ResourceProfile: profile,
	}, nil
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
func (m *mockAdminGovernanceRepo) DuplicateTemplate(ctx context.Context, tenantID, templateID, adminID string) (*domain.AdminTemplateReviewItem, error) {
	baseRam := 768
	profile := domain.DeriveResourceProfile(baseRam)
	return &domain.AdminTemplateReviewItem{
		ID:              "dup-tpl-id",
		Name:            "(Copia) Plantilla",
		DockerImage:     "node:20-slim",
		Status:          "PENDIENTE_AUDITORIA",
		BaseRamMB:       baseRam,
		ResourceProfile: profile,
	}, nil
}
func (m *mockAdminGovernanceRepo) UpdateEOLStatus(ctx context.Context, templateID string, status, eolDate, message string) error {
	return nil
}
func (m *mockAdminGovernanceRepo) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	return &domain.TemplateCategory{ID: "cat-1", Name: dto.Name, Description: dto.Description}, nil
}
func (m *mockAdminGovernanceRepo) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	return &domain.TemplateCategory{ID: categoryID, Name: dto.Name, Description: dto.Description}, nil
}
func (m *mockAdminGovernanceRepo) DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error {
	return nil
}
func (m *mockAdminGovernanceRepo) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	return nil
}
func (m *mockAdminGovernanceRepo) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	return &domain.TemplateModelItemDTO{ID: modelID, Title: dto.Title, CategoryID: dto.CategoryID}, nil
}
func (m *mockAdminGovernanceRepo) SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error {
	return nil
}
func (m *mockAdminGovernanceRepo) PromoteTemplateToModel(ctx context.Context, tenantID, templateID, adminID string, dto domain.PromoteTemplateToModelDTO) (*domain.TemplateModelItemDTO, error) {
	return &domain.TemplateModelItemDTO{ID: "model-1", Title: dto.Title, CategoryID: dto.CategoryID}, nil
}
func (m *mockAdminGovernanceRepo) SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	return &domain.TemplateDraft{ID: "draft-1", TenantID: tenantID, UserID: userID, FormData: formData}, nil
}
func (m *mockAdminGovernanceRepo) GetDraftByUser(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	return nil, nil
}
func (m *mockAdminGovernanceRepo) DeleteDraft(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (m *mockAdminGovernanceRepo) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	return map[string]int{
		"python:3.12-slim-bookworm": 4,
		"node:20-bookworm-slim":     2,
	}, nil
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
	// Formula: min = base/2 (256), high = base*1.5 (768), max = base*2 (1024)
	if item.ResourceProfile.HighMB != 768 {
		t.Errorf("esperado HighMB = 768, obtenido: %d", item.ResourceProfile.HighMB)
	}
	if item.ResourceProfile.MaxMB != 1024 {
		t.Errorf("esperado MaxMB = 1024, obtenido: %d", item.ResourceProfile.MaxMB)
	}
	if item.ResourceProfile.MinMB != 256 {
		t.Errorf("esperado MinMB = 256, obtenido: %d", item.ResourceProfile.MinMB)
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
	if item.ResourceProfile.MaxMB != 1024 {
		t.Errorf("esperado MaxMB = 1024, obtenido: %d", item.ResourceProfile.MaxMB)
	}
	if item.ResourceProfile.HighMB != 768 {
		t.Errorf("esperado HighMB = 768, obtenido: %d", item.ResourceProfile.HighMB)
	}
	if item.ResourceProfile.MinMB != 256 {
		t.Errorf("esperado MinMB = 256, obtenido: %d", item.ResourceProfile.MinMB)
	}
}

func TestCreateOfficialTemplate_CgroupsProfileDerivation1024(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.CreateOfficialTemplateDTO{
		Name:              "Go Backend Lab",
		DockerImage:       "golang:1.24-alpine",
		BaseRamMB:         1024,
		TargetEnvironment: "IDE_PERSISTENTE",
	}

	item, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dto)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if item.BaseRamMB != 1024 {
		t.Errorf("esperado BaseRamMB = 1024, obtenido: %d", item.BaseRamMB)
	}
	if item.ResourceProfile.MinMB != 512 {
		t.Errorf("esperado MinMB = 512, obtenido: %d", item.ResourceProfile.MinMB)
	}
	if item.ResourceProfile.HighMB != 1536 {
		t.Errorf("esperado HighMB = 1536, obtenido: %d", item.ResourceProfile.HighMB)
	}
	if item.ResourceProfile.MaxMB != 2048 {
		t.Errorf("esperado MaxMB = 2048, obtenido: %d", item.ResourceProfile.MaxMB)
	}
}

func TestReviewTemplate_CgroupsProfileRegeneration2048(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	newRam := 2048
	dto := domain.ReviewTemplateDTO{
		Status:    "approved",
		BaseRamMB: &newRam,
	}

	item, err := svc.ReviewTemplate(context.Background(), "tenant-1", "tpl-1", "admin-1", dto)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if item.BaseRamMB != 2048 {
		t.Errorf("esperado BaseRamMB = 2048, obtenido: %d", item.BaseRamMB)
	}
	if item.ResourceProfile.MinMB != 1024 {
		t.Errorf("esperado MinMB = 1024, obtenido: %d", item.ResourceProfile.MinMB)
	}
	if item.ResourceProfile.HighMB != 3072 {
		t.Errorf("esperado HighMB = 3072, obtenido: %d", item.ResourceProfile.HighMB)
	}
	if item.ResourceProfile.MaxMB != 4096 {
		t.Errorf("esperado MaxMB = 4096, obtenido: %d", item.ResourceProfile.MaxMB)
	}
}

func TestDuplicateTemplate_CgroupsProfileDerivedFromCopy(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	item, err := svc.DuplicateTemplate(context.Background(), "tenant-1", "tpl-1", "admin-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	expectedProfile := domain.DeriveResourceProfile(item.BaseRamMB)
	if item.ResourceProfile != expectedProfile {
		t.Errorf("esperado ResourceProfile %+v, obtenido: %+v", expectedProfile, item.ResourceProfile)
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

func TestDuplicateTemplate_Success(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	item, err := svc.DuplicateTemplate(context.Background(), "tenant-1", "orig-tpl-1", "admin-1")
	if err != nil {
		t.Fatalf("error inesperado duplicando plantilla: %v", err)
	}
	if item.Name != "(Copia) Plantilla" {
		t.Errorf("esperado nombre con prefijo (Copia), obtenido: %s", item.Name)
	}
	if item.Status != "PENDIENTE_AUDITORIA" {
		t.Errorf("esperado estado PENDIENTE_AUDITORIA, obtenido: %s", item.Status)
	}
}

func TestCreateOfficialTemplate_RegistryWhitelist(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	// Registry no permitido (ej. untrusted.evil.com)
	dtoUntrusted := domain.CreateOfficialTemplateDTO{
		Name:        "Imagen Maliciosa",
		DockerImage: "untrusted.evil.com/malware:1.0",
		BaseRamMB:   512,
	}
	_, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoUntrusted)
	if err == nil {
		t.Fatal("esperaba error por registro no permitido, se obtuvo nil")
	}

	// Registry permitido estándar (ghcr.io)
	dtoAllowed := domain.CreateOfficialTemplateDTO{
		Name:        "Imagen Confiable GHCR",
		DockerImage: "ghcr.io/academic-org/lab-c:v1.0",
		BaseRamMB:   512,
	}
	item, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoAllowed)
	if err != nil {
		t.Fatalf("error inesperado para imagen en registro permitido: %v", err)
	}
	if item == nil {
		t.Fatal("esperaba plantilla creada, se obtuvo nil")
	}
}

func TestReviewTemplate_ApproveTransitionsToPendingAudit(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.ReviewTemplateDTO{
		Status: "pending_audit",
	}

	item, err := svc.ReviewTemplate(context.Background(), "tenant-1", "tpl-123", "admin-1", dto)
	if err != nil {
		t.Fatalf("error inesperado al revisar plantilla: %v", err)
	}

	if mockRepo.lastReviewedStatus != "PENDIENTE_AUDITORIA" {
		t.Errorf("esperado estado PENDIENTE_AUDITORIA, obtenido: %s", mockRepo.lastReviewedStatus)
	}
	if item.Status != "PENDIENTE_AUDITORIA" {
		t.Errorf("esperado item con status PENDIENTE_AUDITORIA, obtenido: %s", item.Status)
	}
}

func TestReviewTemplate_SuspendedRequiresReason(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dtoWithoutReason := domain.ReviewTemplateDTO{
		Status:          "suspended",
		RejectionReason: "   ",
	}

	_, err := svc.ReviewTemplate(context.Background(), "tenant-1", "tpl-123", "admin-1", dtoWithoutReason)
	if err == nil || !errors.Is(err, services.ErrRejectionReasonRequired) {
		t.Fatalf("esperado ErrRejectionReasonRequired al suspender sin motivo, obtenido: %v", err)
	}
}

func TestReviewTemplate_SuspendedSuccess(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.ReviewTemplateDTO{
		Status:          "suspended",
		RejectionReason: "Tag :latest prohibido por reproducibilidad",
	}

	item, err := svc.ReviewTemplate(context.Background(), "tenant-1", "tpl-123", "admin-1", dto)
	if err != nil {
		t.Fatalf("error inesperado al suspender plantilla: %v", err)
	}

	if mockRepo.lastReviewedStatus != "SUSPENDIDA" {
		t.Errorf("esperado estado SUSPENDIDA en el repositorio, obtenido: %s", mockRepo.lastReviewedStatus)
	}
	if item.Status != "SUSPENDIDA" {
		t.Errorf("esperado status SUSPENDIDA en el resultado, obtenido: %s", item.Status)
	}
	if mockRepo.lastRejectionReason != "Tag :latest prohibido por reproducibilidad" {
		t.Errorf("motivo no coincide: %s", mockRepo.lastRejectionReason)
	}
}

func TestGetRuntimeCapabilities(t *testing.T) {
	svc := services.NewAdminGovernanceService(nil, nil)

	caps, err := svc.GetRuntimeCapabilities(context.Background())
	if err != nil {
		t.Fatalf("error inesperado al obtener capacidades de ejecución: %v", err)
	}

	if caps == nil {
		t.Fatal("las capacidades no deben ser nulas")
	}

	if caps.HostMemory.TotalRAMMB <= 0 {
		t.Errorf("total_ram_mb debe ser mayor a 0, obtenido: %d", caps.HostMemory.TotalRAMMB)
	}

	if len(caps.SatelliteServices) == 0 {
		t.Error("debe reportar catálogo de servicios satélite")
	}

	foundPostgres := false
	for _, s := range caps.SatelliteServices {
		if s.Engine == "postgres" && s.IsAvailable {
			foundPostgres = true
			if s.Description == "" {
				t.Error("descripción de postgres no debe estar vacía")
			}
		}
	}
	if !foundPostgres {
		t.Error("PostgreSQL debe estar disponible en las capacidades")
	}

	if len(caps.IDEPresets) == 0 || len(caps.JudgePresets) == 0 {
		t.Error("deben existir presets tanto para IDE como para Juez")
	}

	if caps.MaxAllowedRamMB <= 0 {
		t.Errorf("max_allowed_ram_mb debe ser positivo, obtenido: %d", caps.MaxAllowedRamMB)
	}
}

func TestAdminGovernanceService_GetImageUsageCounts(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	usageMap, err := svc.GetImageUsageCounts(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if usageMap["python:3.12-slim-bookworm"] != 4 {
		t.Errorf("esperado 4 usos para python, obtenido: %d", usageMap["python:3.12-slim-bookworm"])
	}
	if usageMap["node:20-bookworm-slim"] != 2 {
		t.Errorf("esperado 2 usos para node, obtenido: %d", usageMap["node:20-bookworm-slim"])
	}
}


