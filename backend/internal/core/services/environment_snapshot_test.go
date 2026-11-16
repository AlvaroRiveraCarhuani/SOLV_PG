package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestPublishExercise_PersistsEnvironmentSnapshot(t *testing.T) {
	exRepo := &mockExerciseRepoGov{exercises: make(map[string]*domain.Exercise)}
	govRepo := &mockGovRepoForTemplate{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tpl-pin": {
				ID:                "tpl-pin",
				Name:              "Pin IDE",
				DockerImage:       "solv/ide:1.2.3",
				Entrypoint:        "python3 main.py",
				TimeoutMS:         7000,
				BaseRamMB:         512,
				TargetEnvironment: "IDE_PERSISTENTE",
				Status:            "approved",
				SetupScript:       "echo setup",
				ServicesConfig: domain.ServicesConfig{Services: []domain.ServiceRequirement{
					{Engine: "postgres", Version: "16", EnvAlias: "db"},
					{Engine: "redis", Version: "7", EnvAlias: "cache"},
				}},
			},
		},
	}
	svc := services.NewEvaluationService(exRepo, nil, nil, nil)
	svc.SetAdminGovernanceRepository(govRepo)

	ex := &domain.Exercise{
		ID:              "ex-pin",
		Title:           "Práctica con pin",
		Type:            domain.ExerciseTypeProject,
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		TemplateID:      "tpl-pin",
	}
	if err := svc.CreateExercise(context.Background(), ex); err != nil {
		t.Fatalf("no se pudo crear el ejercicio: %v", err)
	}
	if _, err := svc.PublishExercise(context.Background(), ex.ID, ex.TenantID); err != nil {
		t.Fatalf("no se pudo publicar: %v", err)
	}

	snapshot := exRepo.exercises[ex.ID].EnvironmentSnapshot
	if snapshot == nil {
		t.Fatal("la publicación no persistió environment_snapshot")
	}
	if snapshot.TemplateID != "tpl-pin" || snapshot.DockerImage != "solv/ide:1.2.3" {
		t.Fatalf("snapshot incompleto: %+v", snapshot)
	}
	if snapshot.Entrypoint != "python3 main.py" || snapshot.TimeoutMS != 7000 || snapshot.BaseRAMMB != 512 {
		t.Fatalf("snapshot no conservó entrypoint/timeout/RAM: %+v", snapshot)
	}
	if snapshot.SetupScript != "echo setup" {
		t.Fatalf("snapshot no conservó el setup: %+v", snapshot)
	}
	if len(snapshot.Bindings) != 2 || snapshot.Bindings[0].Engine != "postgres" || snapshot.Bindings[0].EnvAlias != "db" {
		t.Fatalf("bindings resueltos inesperados: %+v", snapshot.Bindings)
	}
	if len(snapshot.InitContract) != 2 || snapshot.InitContract[0].Kind != domain.ServiceInitSQL {
		t.Fatalf("contrato de init inesperado: %+v", snapshot.InitContract)
	}
	if snapshot.VerificationFingerprint == "" {
		t.Fatal("la publicación no calculó verification_fingerprint")
	}
	if snapshot.VerificationFingerprint != snapshot.ComputeVerificationFingerprint() {
		t.Fatal("verification_fingerprint no es reproducible")
	}
}
