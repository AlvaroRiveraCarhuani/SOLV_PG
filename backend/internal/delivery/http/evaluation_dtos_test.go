package httpdelivery

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestNormalizeExerciseConfigPersistsPreviewRecipe(t *testing.T) {
	ex := &domain.Exercise{
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		Config: domain.ExerciseConfig{
			IDE: &domain.IDEConfig{},
		},
	}

	normalizeExerciseConfig([]byte(`{
		"environment_type": "IDE_PERSISTENTE",
		"app_run_command": "npm run dev",
		"app_port": 5173
	}`), ex)

	if ex.Config.IDE == nil {
		t.Fatal("expected IDE config")
	}
	if ex.Config.IDE.AppRunCommand != "npm run dev" {
		t.Fatalf("expected command, got %q", ex.Config.IDE.AppRunCommand)
	}
	if ex.Config.IDE.AppPort != 5173 {
		t.Fatalf("expected port 5173, got %d", ex.Config.IDE.AppPort)
	}
}

func TestNormalizeExerciseConfigClearsPreviewRecipe(t *testing.T) {
	ex := &domain.Exercise{
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		Config: domain.ExerciseConfig{
			IDE: &domain.IDEConfig{
				AppRunCommand: "npm run dev",
				AppPort:       5173,
			},
		},
	}

	normalizeExerciseConfig([]byte(`{
		"environment_type": "IDE_PERSISTENTE",
		"app_run_command": "",
		"app_port": 0
	}`), ex)

	if ex.Config.IDE.AppRunCommand != "" {
		t.Fatalf("expected cleared command, got %q", ex.Config.IDE.AppRunCommand)
	}
	if ex.Config.IDE.AppPort != 0 {
		t.Fatalf("expected cleared port, got %d", ex.Config.IDE.AppPort)
	}
}
