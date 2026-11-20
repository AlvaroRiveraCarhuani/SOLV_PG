package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestCreateOfficialTemplate_NormalizaEjecutoresDeLenguaje(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	dto := domain.CreateOfficialTemplateDTO{
		Name:              "Evaluador Python + C++",
		DockerImage:       "python:3.12-slim",
		BaseRamMB:         256,
		TargetEnvironment: "JUEZ_EFIMERO",
		LanguageRunners: []domain.LanguageRunner{
			{Language: "c++", Entrypoint: "g++ -O2 solution.cpp -o solution && ./solution"},
			{Language: "Python3", Entrypoint: "python3 solution.py"},
		},
	}

	if _, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", dto); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if mockRepo.lastCreatedDTO == nil {
		t.Fatal("el repositorio no recibió el alta")
	}

	runners := mockRepo.lastCreatedDTO.LanguageRunners
	if len(runners) != 2 {
		t.Fatalf("esperaba 2 ejecutores, obtuve %d", len(runners))
	}
	if runners[0].Language != "python" || runners[1].Language != "cpp" {
		t.Errorf("orden canónico inesperado: %q, %q", runners[0].Language, runners[1].Language)
	}
	// El entrypoint de la plantilla se deriva del lenguaje primario cuando el
	// alta no lo envía: es el que usa la comprobación de arranque.
	if mockRepo.lastCreatedDTO.Entrypoint != "python3 solution.py" {
		t.Errorf("entrypoint derivado inesperado: %q", mockRepo.lastCreatedDTO.Entrypoint)
	}
}

func TestCreateOfficialTemplate_RechazaLenguajeDuplicado(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	_, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", domain.CreateOfficialTemplateDTO{
		Name:              "Evaluador Duplicado",
		DockerImage:       "python:3.12-slim",
		BaseRamMB:         256,
		TargetEnvironment: "JUEZ_EFIMERO",
		LanguageRunners: []domain.LanguageRunner{
			{Language: "cpp", Entrypoint: "g++ solution.cpp -o solution && ./solution"},
			{Language: "c++", Entrypoint: "g++ solution.cpp -o solution && ./solution"},
		},
	})
	if !errors.Is(err, domain.ErrLanguageRunnerDuplicate) {
		t.Fatalf("esperaba ErrLanguageRunnerDuplicate, obtuve %v", err)
	}
	if mockRepo.lastCreatedDTO != nil {
		t.Error("un alta rechazada no debe llegar al repositorio")
	}
}

func TestCreateOfficialTemplate_ModoIDEDescartaEjecutores(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	_, err := svc.CreateOfficialTemplate(context.Background(), "tenant-1", "admin-1", domain.CreateOfficialTemplateDTO{
		Name:              "Laboratorio IDE",
		DockerImage:       "python:3.12-slim",
		BaseRamMB:         512,
		TargetEnvironment: "IDE_PERSISTENTE",
		LanguageRunners: []domain.LanguageRunner{
			{Language: "python", Entrypoint: "python3 solution.py"},
		},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if mockRepo.lastCreatedDTO == nil {
		t.Fatal("el repositorio no recibió el alta")
	}
	if len(mockRepo.lastCreatedDTO.LanguageRunners) != 0 {
		t.Errorf("un entorno de desarrollo no debe declarar ejecutores: %+v", mockRepo.lastCreatedDTO.LanguageRunners)
	}
}

func TestUpdateOfficialTemplate_NormalizaEjecutoresDeLenguaje(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	_, err := svc.UpdateOfficialTemplate(context.Background(), "tenant-1", "tpl-1", "admin-1", domain.UpdateOfficialTemplateDTO{
		Name:              "Evaluador Go y Java",
		DockerImage:       "golang:1.22-bookworm",
		BaseRamMB:         512,
		TargetEnvironment: "AMBOS",
		Entrypoint:        "go run main.go",
		LanguageRunners: []domain.LanguageRunner{
			{Language: "golang", Entrypoint: "go run main.go"},
			{Language: "java", Entrypoint: "javac Solution.java && java Solution"},
		},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if mockRepo.lastCreatedDTO != nil {
		t.Fatal("la actualización no debe pasar por el alta")
	}
}
