package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func quotaRepoWith(status, targetEnv string) (*mockAdminGovernanceRepo, string) {
	tenant := "tenant-1"
	return &mockAdminGovernanceRepo{templateByID: map[string]*domain.AdminTemplateReviewItem{
		"tpl-local": {
			ID:                "tpl-local",
			TenantID:          &tenant,
			DockerImage:       "solv-local/stack:abc123",
			BaseRamMB:         768,
			Status:            status,
			TargetEnvironment: targetEnv,
		},
	}}, tenant
}

func TestUpdateTemplateQuotaAllowsLocalImageWithoutBuildVerification(t *testing.T) {
	repo, tenant := quotaRepoWith("APROBADA", domain.TargetEnvironmentIDEPersistente)
	svc := services.NewAdminGovernanceService(nil, repo)
	description := "Cuota ajustada para el curso"

	item, err := svc.UpdateTemplateQuota(context.Background(), tenant, "tpl-local", "admin-1", domain.UpdateTemplateQuotaDTO{
		BaseRamMB:   1024,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("una plantilla local no debe exigir job de build para cambiar la cuota: %v", err)
	}
	if item.BaseRamMB != 1024 {
		t.Fatalf("esperaba 1024 MB, obtuve %d", item.BaseRamMB)
	}
	if repo.lastQuotaDTO == nil || !repo.lastQuotaDTO.ResetSmoke {
		t.Fatal("una plantilla aprobada debe repetir la comprobación de arranque")
	}
}

func TestUpdateTemplateQuotaDoesNotReactivatePausedTemplate(t *testing.T) {
	repo, tenant := quotaRepoWith("paused", domain.TargetEnvironmentIDEPersistente)
	svc := services.NewAdminGovernanceService(nil, repo)

	if _, err := svc.UpdateTemplateQuota(context.Background(), tenant, "tpl-local", "admin-1", domain.UpdateTemplateQuotaDTO{
		BaseRamMB: 1024,
	}); err != nil {
		t.Fatalf("ajustar una plantilla pausada no debe fallar: %v", err)
	}
	if repo.lastQuotaDTO == nil || repo.lastQuotaDTO.ResetSmoke {
		t.Fatal("una plantilla pausada no debe reencolarse para auditoría")
	}
}

func TestUpdateTemplateQuotaRejectsBelowVitalFloor(t *testing.T) {
	repo, tenant := quotaRepoWith("APROBADA", domain.TargetEnvironmentIDEPersistente)
	svc := services.NewAdminGovernanceService(nil, repo)

	_, err := svc.UpdateTemplateQuota(context.Background(), tenant, "tpl-local", "admin-1", domain.UpdateTemplateQuotaDTO{
		BaseRamMB: 64,
	})
	if !errors.Is(err, services.ErrRamBelowVitalFloor) {
		t.Fatalf("esperaba ErrRamBelowVitalFloor, obtuve %v", err)
	}
}

func TestUpdateTemplateQuotaRejectsHostOverflow(t *testing.T) {
	repo, tenant := quotaRepoWith("APROBADA", domain.TargetEnvironmentIDEPersistente)
	svc := services.NewAdminGovernanceService(nil, repo)

	_, err := svc.UpdateTemplateQuota(context.Background(), tenant, "tpl-local", "admin-1", domain.UpdateTemplateQuotaDTO{
		BaseRamMB: 1 << 20,
	})
	if !errors.Is(err, domain.ErrRamExceedsHostCapacity) {
		t.Fatalf("esperaba ErrRamExceedsHostCapacity, obtuve %v", err)
	}
}

func TestUpdateTemplateQuotaRejectsForeignTenant(t *testing.T) {
	repo, _ := quotaRepoWith("APROBADA", domain.TargetEnvironmentIDEPersistente)
	svc := services.NewAdminGovernanceService(nil, repo)

	_, err := svc.UpdateTemplateQuota(context.Background(), "tenant-2", "tpl-local", "admin-1", domain.UpdateTemplateQuotaDTO{
		BaseRamMB: 1024,
	})
	if !errors.Is(err, services.ErrTemplateNotFound) {
		t.Fatalf("esperaba ErrTemplateNotFound para otro tenant, obtuve %v", err)
	}
}

func TestUpdateTemplateQuotaRejectsRejectedTemplate(t *testing.T) {
	repo, tenant := quotaRepoWith("RECHAZADA", domain.TargetEnvironmentIDEPersistente)
	svc := services.NewAdminGovernanceService(nil, repo)

	_, err := svc.UpdateTemplateQuota(context.Background(), tenant, "tpl-local", "admin-1", domain.UpdateTemplateQuotaDTO{
		BaseRamMB: 1024,
	})
	if !errors.Is(err, services.ErrTemplateQuotaNotEditable) {
		t.Fatalf("esperaba ErrTemplateQuotaNotEditable, obtuve %v", err)
	}
}
