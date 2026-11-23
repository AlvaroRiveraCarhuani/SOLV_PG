package services_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

// ResetTemplateAudit del mock deja el smoke y el escaneo en pending, igual que
// el repositorio real, y conserva el estado del catálogo.
func (m *mockAdminGovernanceRepo) ResetTemplateAudit(
	ctx context.Context,
	tenantID, templateID string,
) (*domain.AdminTemplateReviewItem, error) {
	tpl := m.templateByID[templateID]
	if tpl == nil {
		return nil, fmt.Errorf("template not found")
	}
	m.lastResetAuditID = templateID
	updated := *tpl
	updated.SmokeTestStatus = "pending"
	updated.SmokeTestOutput = ""
	updated.SecurityAuditStatus = "pending"
	updated.CVECriticalCount = 0
	updated.CVEHighCount = 0
	updated.SecurityAuditedAt = nil
	return &updated, nil
}

// El re-escaneo operativo reencola el análisis sin cambiar el estado del
// catálogo: una plantilla publicada sigue publicada.
func TestReauditTemplate_ResetsAuditWithoutTouchingStatus(t *testing.T) {
	tenant := "tenant-1"
	repo := &mockAdminGovernanceRepo{
		templateByID: map[string]*domain.AdminTemplateReviewItem{
			"tpl-1": {
				ID:                  "tpl-1",
				TenantID:            &tenant,
				Status:              "APROBADA",
				SmokeTestStatus:     "passed",
				SecurityAuditStatus: "failed",
				CVECriticalCount:    2,
				CVEHighCount:        478,
			},
		},
	}
	svc := services.NewAdminGovernanceService(nil, repo)

	item, err := svc.ReauditTemplate(context.Background(), tenant, "tpl-1")
	if err != nil {
		t.Fatalf("error inesperado reencolando: %v", err)
	}
	if repo.lastResetAuditID != "tpl-1" {
		t.Fatalf("no se llamó al repositorio para reencolar, got %q", repo.lastResetAuditID)
	}
	if item.Status != "APROBADA" {
		t.Fatalf("el estado del catálogo no debía cambiar, obtuve %q", item.Status)
	}
	if item.SmokeTestStatus != "pending" || item.SecurityAuditStatus != "pending" {
		t.Fatalf("esperaba smoke y escaneo en pending, obtuve %q / %q", item.SmokeTestStatus, item.SecurityAuditStatus)
	}
	if item.CVECriticalCount != 0 || item.CVEHighCount != 0 {
		t.Fatalf("los conteos debían reiniciarse, obtuve %d / %d", item.CVECriticalCount, item.CVEHighCount)
	}
}

// Los estados que decide una persona no se reauditan: el worker los omite y la
// plantilla quedaría en "en análisis" para siempre.
func TestReauditTemplate_RejectsHumanReviewStates(t *testing.T) {
	repo := &mockAdminGovernanceRepo{
		templateByID: map[string]*domain.AdminTemplateReviewItem{
			"tpl-paused": {ID: "tpl-paused", Status: "PAUSADA"},
		},
	}
	svc := services.NewAdminGovernanceService(nil, repo)

	if _, err := svc.ReauditTemplate(context.Background(), "tenant-1", "tpl-paused"); !errors.Is(err, services.ErrTemplateNotAuditable) {
		t.Fatalf("esperaba ErrTemplateNotAuditable, obtuve %v", err)
	}
	if _, err := svc.ReauditTemplate(context.Background(), "tenant-1", "tpl-inexistente"); !errors.Is(err, services.ErrTemplateNotFound) {
		t.Fatalf("esperaba ErrTemplateNotFound, obtuve %v", err)
	}
}

// Otra institución no puede reencolar una plantilla ajena.
func TestReauditTemplate_RejectsForeignTenant(t *testing.T) {
	owner := "tenant-2"
	repo := &mockAdminGovernanceRepo{
		templateByID: map[string]*domain.AdminTemplateReviewItem{
			"tpl-ajena": {ID: "tpl-ajena", TenantID: &owner, Status: "APROBADA"},
		},
	}
	svc := services.NewAdminGovernanceService(nil, repo)

	if _, err := svc.ReauditTemplate(context.Background(), "tenant-1", "tpl-ajena"); !errors.Is(err, services.ErrTemplateNotFound) {
		t.Fatalf("esperaba ErrTemplateNotFound para otra institución, obtuve %v", err)
	}
}

// El Constructor ya ejecutó el smoke con las herramientas declaradas: el alta
// debe persistir ese resultado en lugar de un 'pending' sin sustento.
func TestCreateOfficialTemplate_PersistsConstructorSmoke(t *testing.T) {
	jobs := memory.NewEnvTestJobMemoryRepository(time.Hour)
	repo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, repo).SetEnvTestJobRepository(jobs)

	dto := localTemplateDTO("")
	fixture := seedVerificationJob(t, jobs, localTemplateConfig(dto), domain.EnvTestStatusSuccess, "admin-1", "tenant-1")
	dto.VerificationJobID = fixture.jobID

	if _, err := svc.CreateOfficialTemplate(adminCtx("admin-1"), "tenant-1", "admin-1", dto); err != nil {
		t.Fatalf("error inesperado creando la plantilla: %v", err)
	}
	if repo.lastCreatedDTO == nil {
		t.Fatal("no se llamó al repositorio con el alta")
	}
	if repo.lastCreatedDTO.SmokeTestStatus != "passed" {
		t.Fatalf("smoke = %q, quiero 'passed' tras la comprobación del Constructor", repo.lastCreatedDTO.SmokeTestStatus)
	}
	if repo.lastCreatedDTO.SmokeTestOutput == "" {
		t.Fatal("el smoke aprobado debe registrar de dónde viene")
	}
}

// Una imagen pública no pasó por el Constructor: sigue esperando el smoke real.
func TestCreateOfficialTemplate_PublicImageStaysPending(t *testing.T) {
	repo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, repo)

	dto := domain.CreateOfficialTemplateDTO{
		Name:              "Python público",
		DockerImage:       "python:3.12-slim",
		BaseRamMB:         512,
		TargetEnvironment: "IDE_PERSISTENTE",
		ToolsDeclared:     []string{"python3"},
	}
	if _, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dto); err != nil {
		t.Fatalf("error inesperado creando la plantilla pública: %v", err)
	}
	if repo.lastCreatedDTO.SmokeTestStatus != "" {
		t.Fatalf("un alta pública no debe declarar smoke, obtuve %q", repo.lastCreatedDTO.SmokeTestStatus)
	}
}

// La edición verificada en el Constructor también conserva el smoke aprobado.
func TestUpdateOfficialTemplate_PersistsConstructorSmoke(t *testing.T) {
	jobs := memory.NewEnvTestJobMemoryRepository(time.Hour)
	repo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, repo).SetEnvTestJobRepository(jobs)

	updateDTO := domain.UpdateOfficialTemplateDTO{
		Name:              "Stack Local Python",
		DockerImage:       localStackImage,
		BaseRamMB:         768,
		TargetEnvironment: "IDE_PERSISTENTE",
		TimeoutMS:         5000,
		SampleInput:       "42",
		SetupScript:       "echo setup",
		ToolsDeclared:     []string{"python3", "pip"},
		StackConfig: &domain.StackConfig{
			DockerfileContent: "FROM python:3.12-slim-bookworm",
		},
	}
	config := domain.EnvTestVerificationConfig{
		Image:             updateDTO.DockerImage,
		Tools:             updateDTO.ToolsDeclared,
		TargetEnvironment: updateDTO.TargetEnvironment,
		Entrypoint:        updateDTO.Entrypoint,
		TimeoutMS:         updateDTO.TimeoutMS,
		SampleInput:       updateDTO.SampleInput,
		BaseRamMB:         updateDTO.BaseRamMB,
		DockerfileContent: updateDTO.StackConfig.DockerfileContent,
		SetupScript:       updateDTO.SetupScript,
	}
	fixture := seedVerificationJob(t, jobs, config, domain.EnvTestStatusSuccess, "admin-1", "tenant-1")
	updateDTO.VerificationJobID = fixture.jobID

	if _, err := svc.UpdateOfficialTemplate(adminCtx("admin-1"), "tenant-1", "tpl-1", "admin-1", updateDTO); err != nil {
		t.Fatalf("error inesperado editando la plantilla: %v", err)
	}
	if repo.lastUpdatedDTO == nil || repo.lastUpdatedDTO.SmokeTestStatus != "passed" {
		t.Fatalf("la edición verificada debía persistir smoke='passed', obtuve %+v", repo.lastUpdatedDTO)
	}
}
