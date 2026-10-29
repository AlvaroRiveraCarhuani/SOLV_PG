package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

const localStackImage = "solv-local/stack:abc123def456"

func adminCtx(userID string) context.Context {
	return context.WithValue(context.Background(), domain.UserIDKey, userID)
}

func localTemplateDTO(jobID string) domain.CreateOfficialTemplateDTO {
	return domain.CreateOfficialTemplateDTO{
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
		VerificationJobID: jobID,
	}
}

func localTemplateConfig(dto domain.CreateOfficialTemplateDTO) domain.EnvTestVerificationConfig {
	return domain.EnvTestVerificationConfig{
		Image:             dto.DockerImage,
		Tools:             dto.ToolsDeclared,
		TargetEnvironment: dto.TargetEnvironment,
		Entrypoint:        dto.Entrypoint,
		TimeoutMS:         dto.TimeoutMS,
		SampleInput:       dto.SampleInput,
		BaseRamMB:         dto.BaseRamMB,
		DockerfileContent: dto.StackConfig.DockerfileContent,
		SetupScript:       dto.SetupScript,
	}
}

type verificationJobFixture struct {
	jobID string
}

func seedVerificationJob(
	t *testing.T,
	repo *memory.EnvTestJobMemoryRepository,
	cfg domain.EnvTestVerificationConfig,
	status domain.EnvTestStatus,
	ownerID, tenantID string,
) verificationJobFixture {
	t.Helper()

	job := &domain.EnvTestJob{
		ID:                      "job-" + ownerID + "-" + tenantID,
		Image:                   cfg.Image,
		Tools:                   cfg.Tools,
		TargetEnvironment:       cfg.TargetEnvironment,
		Entrypoint:              cfg.Entrypoint,
		TimeoutMS:               cfg.TimeoutMS,
		SampleInput:             cfg.SampleInput,
		VerificationFingerprint: cfg.Fingerprint(),
		VerificationConfig:      cfg,
		OwnerID:                 ownerID,
		TenantID:                tenantID,
		Status:                  status,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	if err := repo.Save(context.Background(), job); err != nil {
		t.Fatalf("no se pudo guardar el job de verificación: %v", err)
	}
	return verificationJobFixture{jobID: job.ID}
}

// La imagen construida localmente no pasa por la whitelist externa, pero sí por
// la comprobación: sin job, o con uno que no coincida, no se puede publicar.
func TestCreateOfficialTemplate_LocalImageGate(t *testing.T) {
	tests := []struct {
		name      string
		seedJob   bool
		status    domain.EnvTestStatus
		ownerID   string
		tenantID  string
		mutateDTO func(*domain.CreateOfficialTemplateDTO)
		wantErr   bool
	}{
		{name: "sin comprobación previa", seedJob: false, wantErr: true},
		{name: "comprobación exitosa del mismo usuario y tenant", seedJob: true, status: domain.EnvTestStatusSuccess, ownerID: "admin-1", tenantID: "tenant-1"},
		{name: "comprobación fallida", seedJob: true, status: domain.EnvTestStatusFailed, ownerID: "admin-1", tenantID: "tenant-1", wantErr: true},
		{name: "comprobación de otro usuario", seedJob: true, status: domain.EnvTestStatusSuccess, ownerID: "admin-2", tenantID: "tenant-1", wantErr: true},
		{name: "comprobación de otro tenant", seedJob: true, status: domain.EnvTestStatusSuccess, ownerID: "admin-1", tenantID: "tenant-2", wantErr: true},
		{
			name: "entrada de muestra distinta a la probada", seedJob: true, status: domain.EnvTestStatusSuccess,
			ownerID: "admin-1", tenantID: "tenant-1",
			mutateDTO: func(dto *domain.CreateOfficialTemplateDTO) { dto.SampleInput = "99" },
			wantErr:   true,
		},
		{
			name: "dockerfile distinto al probado", seedJob: true, status: domain.EnvTestStatusSuccess,
			ownerID: "admin-1", tenantID: "tenant-1",
			mutateDTO: func(dto *domain.CreateOfficialTemplateDTO) {
				dto.StackConfig.DockerfileContent = "FROM python:3.12-slim-bookworm\nRUN pip install flask"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobs := memory.NewEnvTestJobMemoryRepository(time.Hour)
			mockRepo := &mockAdminGovernanceRepo{}
			svc := services.NewAdminGovernanceService(nil, mockRepo).SetEnvTestJobRepository(jobs)

			dto := localTemplateDTO("")
			if tt.seedJob {
				fixture := seedVerificationJob(t, jobs, localTemplateConfig(dto), tt.status, tt.ownerID, tt.tenantID)
				dto.VerificationJobID = fixture.jobID
			}
			if tt.mutateDTO != nil {
				tt.mutateDTO(&dto)
			}

			item, err := svc.CreateOfficialTemplate(adminCtx("admin-1"), "tenant-1", "admin-1", dto)

			if tt.wantErr {
				if !errors.Is(err, services.ErrTemplateVerificationRequired) {
					t.Fatalf("esperado ErrTemplateVerificationRequired, obtenido: %v", err)
				}
				if item != nil {
					t.Fatalf("la plantilla no debería crearse sin una comprobación válida")
				}
				return
			}

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if item == nil || item.DockerImage != localStackImage {
				t.Fatalf("esperaba la plantilla local creada, obtenido: %+v", item)
			}
		})
	}
}

// La edición de una plantilla con imagen local exige la misma comprobación vigente.
func TestUpdateOfficialTemplate_LocalImageGate(t *testing.T) {
	jobs := memory.NewEnvTestJobMemoryRepository(time.Hour)
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo).SetEnvTestJobRepository(jobs)

	ctx := adminCtx("admin-1")
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

	// Sin comprobación no se permite editar hacia una imagen local.
	if _, err := svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", updateDTO); !errors.Is(err, services.ErrTemplateVerificationRequired) {
		t.Fatalf("esperado ErrTemplateVerificationRequired sin comprobación, obtenido: %v", err)
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

	// Un job exitoso de otro usuario no autoriza la edición.
	foreign := seedVerificationJob(t, jobs, config, domain.EnvTestStatusSuccess, "admin-2", "tenant-1")
	updateDTO.VerificationJobID = foreign.jobID
	if _, err := svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", updateDTO); !errors.Is(err, services.ErrTemplateVerificationRequired) {
		t.Fatalf("esperado ErrTemplateVerificationRequired por autoría, obtenido: %v", err)
	}

	// Con el job del propio usuario y la misma configuración la edición pasa.
	own := seedVerificationJob(t, jobs, config, domain.EnvTestStatusSuccess, "admin-1", "tenant-1")
	updateDTO.VerificationJobID = own.jobID
	item, err := svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", updateDTO)
	if err != nil {
		t.Fatalf("error inesperado editando con comprobación vigente: %v", err)
	}
	if item == nil || item.DockerImage != localStackImage {
		t.Fatalf("esperaba la plantilla local actualizada, obtenido: %+v", item)
	}
}
