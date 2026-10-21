package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestCreateOfficialTemplate_RamExceedsHost(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.CreateOfficialTemplateDTO{
		Name:        "Template Con Mucha RAM",
		DockerImage: "python:3.12-slim",
		BaseRamMB:   9999999, // Supera cualquier host físico razonable
	}

	_, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dto)
	if err == nil {
		t.Fatal("esperaba error por RAM que excede el host, pero no falló")
	}
	if !errors.Is(err, domain.ErrRamExceedsHostCapacity) {
		t.Fatalf("esperado ErrRamExceedsHostCapacity, obtenido: %v", err)
	}
}

func TestReviewTemplate_RamExceedsHost(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	excessiveRam := 9999999
	dto := domain.ReviewTemplateDTO{
		Status:    "approved",
		BaseRamMB: &excessiveRam,
	}

	_, err := svc.ReviewTemplate(context.Background(), "tenant-1", "tpl-123", "admin-1", dto)
	if err == nil {
		t.Fatal("esperaba error por RAM que excede el host, pero no falló")
	}
	if !errors.Is(err, domain.ErrRamExceedsHostCapacity) {
		t.Fatalf("esperado ErrRamExceedsHostCapacity, obtenido: %v", err)
	}
}

func TestReviewTemplate_RamBelowVitalFloor(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{
		templateByID: map[string]*domain.AdminTemplateReviewItem{
			"tpl-ide": {
				ID:                "tpl-ide",
				Name:              "Python 3.12 IDE",
				TargetEnvironment: "IDE_PERSISTENTE",
				ServicesConfig:    domain.ServicesConfig{},
			},
		},
	}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	lowRam := 128
	dto := domain.ReviewTemplateDTO{
		Status:    "approved",
		BaseRamMB: &lowRam,
	}

	_, err := svc.ReviewTemplate(context.Background(), "tenant-1", "tpl-ide", "admin-1", dto)
	if err == nil {
		t.Fatal("esperaba error por RAM inferior al piso vital (128 MB < 210 MB), pero no falló")
	}
	if !errors.Is(err, services.ErrRamBelowVitalFloor) {
		t.Fatalf("esperado ErrRamBelowVitalFloor, obtenido: %v", err)
	}
}

func TestCreateOfficialTemplate_RamBelowVitalFloor(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	// 1. IDE_PERSISTENTE por debajo de 210 MB (OpenVSCode Server) -> error
	dtoIDE := domain.CreateOfficialTemplateDTO{
		Name:              "IDE Bajo Piso Vital",
		DockerImage:       "python:3.12-slim",
		BaseRamMB:         128,
		TargetEnvironment: "IDE_PERSISTENTE",
	}
	if _, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoIDE); !errors.Is(err, services.ErrRamBelowVitalFloor) {
		t.Fatalf("esperado ErrRamBelowVitalFloor para IDE con 128 MB, obtenido: %v", err)
	}

	// 2. IDE_PERSISTENTE con PostgreSQL (210 + 50 = 260 MB) usando 240 MB -> error
	dtoIDEPg := domain.CreateOfficialTemplateDTO{
		Name:              "IDE Postgres Bajo Piso Vital",
		DockerImage:       "python:3.12-slim",
		BaseRamMB:         240,
		TargetEnvironment: "IDE_PERSISTENTE",
		ServicesConfig: &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{
				{Category: "database", Engine: "postgres"},
			},
		},
	}
	if _, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoIDEPg); !errors.Is(err, services.ErrRamBelowVitalFloor) {
		t.Fatalf("esperado ErrRamBelowVitalFloor para IDE + Postgres con 240 MB (piso 260 MB), obtenido: %v", err)
	}

	// 3. JUEZ_EFIMERO por debajo de 64 MB -> error
	dtoJudge := domain.CreateOfficialTemplateDTO{
		Name:              "Juez Bajo Piso Vital",
		DockerImage:       "gcc:13.2-bookworm",
		BaseRamMB:         32,
		TargetEnvironment: "JUEZ_EFIMERO",
		Entrypoint:        "gcc solution.c && ./a.out",
	}
	if _, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dtoJudge); !errors.Is(err, services.ErrRamBelowVitalFloor) {
		t.Fatalf("esperado ErrRamBelowVitalFloor para Juez con 32 MB (piso 64 MB), obtenido: %v", err)
	}
}

func TestUpdateOfficialTemplate_Guardrails(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	ctx := context.Background()

	// 1. Nombre vacío
	_, err := svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:        "",
		DockerImage: "python:3.12-slim-bookworm",
	})
	if err == nil || !strings.Contains(err.Error(), "nombre de la plantilla es requerido") {
		t.Fatalf("esperado error por nombre vacío, obtenido: %v", err)
	}

	// 2. Imagen vacía
	_, err = svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:        "Python Template",
		DockerImage: "",
	})
	if err == nil || !strings.Contains(err.Error(), "imagen Docker es requerida") {
		t.Fatalf("esperado error por imagen vacía, obtenido: %v", err)
	}

	// 3. Tag :latest prohibido
	_, err = svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:        "Python Template",
		DockerImage: "python:latest",
	})
	if !errors.Is(err, services.ErrLatestTagForbidden) {
		t.Fatalf("esperado ErrLatestTagForbidden, obtenido: %v", err)
	}

	// 4. Formato de imagen inválido
	_, err = svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:        "Python Template",
		DockerImage: "python without tag",
	})
	if !errors.Is(err, services.ErrInvalidDockerImage) {
		t.Fatalf("esperado ErrInvalidDockerImage, obtenido: %v", err)
	}

	// 5. Registro no permitido
	_, err = svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:        "Malicious Template",
		DockerImage: "malicious.evil.com/attacker/image:1.0",
	})
	if err == nil {
		t.Fatalf("esperado error por registro no permitido, obtenido nil")
	}

	// 6. JUEZ_EFIMERO / AMBOS sin entrypoint
	_, err = svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:              "Judge Template",
		DockerImage:       "python:3.12-slim-bookworm",
		TargetEnvironment: "JUEZ_EFIMERO",
		Entrypoint:        "",
	})
	if err == nil || !strings.Contains(err.Error(), "entrypoint) es obligatorio") {
		t.Fatalf("esperado error por entrypoint obligatorio en JUEZ_EFIMERO, obtenido: %v", err)
	}

	_, err = svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:              "Dual Template",
		DockerImage:       "python:3.12-slim-bookworm",
		TargetEnvironment: "AMBOS",
		Entrypoint:        "",
	})
	if err == nil || !strings.Contains(err.Error(), "entrypoint) es obligatorio") {
		t.Fatalf("esperado error por entrypoint obligatorio en AMBOS, obtenido: %v", err)
	}

	// 7. AMBOS con entrypoint y residual services_config -> limpia services_config y pasa
	res, err := svc.UpdateOfficialTemplate(ctx, "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:              "Dual Template",
		DockerImage:       "python:3.12-slim-bookworm",
		TargetEnvironment: "AMBOS",
		Entrypoint:        "python3 solution.py",
		BaseRamMB:         512,
		ServicesConfig: &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{
				{Category: "database", Engine: "postgres"},
			},
		},
	})
	if err != nil {
		t.Fatalf("inesperado error actualizando AMBOS con entrypoint: %v", err)
	}
	if len(res.ServicesConfig.Services) != 0 {
		t.Fatalf("esperado ServicesConfig limpio en AMBOS, obtenido %+v", res.ServicesConfig.Services)
	}
}
