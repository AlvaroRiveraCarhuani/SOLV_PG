package services_test

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// appPortFixture arma un laboratorio cuya práctica declara el puerto de su
// aplicación y devuelve el orquestador que captura la configuración del
// contenedor.
func appPortFixture(appPort int) (*mockOrchestrator, *services.PracticeEnvironmentService) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	templateID := "tpl-app"
	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		"subj-app": {ID: "subj-app", Name: "Aplicaciones Web"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		templateID: {
			ID:          templateID,
			Name:        "React",
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-app": {
			ID:         "ex-app",
			TemplateID: templateID,
			Config: domain.ExerciseConfig{IDE: &domain.IDEConfig{
				AppRunCommand: "npm run dev",
				AppPort:       appPort,
			}},
		},
	}}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})
	return orch, svc
}

func TestStartPracticeEnvironmentForExercise_PublishesAppPort(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "")
	orch, svc := appPortFixture(5173)

	ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "std-app", "subj-app", "ex-app")
	if err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
	}
	if orch.lastConfig == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if orch.lastConfig.AppPort != 5173 {
		t.Errorf("esperado AppPort = 5173, obtenido %d", orch.lastConfig.AppPort)
	}
	if orch.lastConfig.AppHostPort == "" {
		t.Error("el puerto de la aplicación debía publicarse en 127.0.0.1")
	}
	if !strings.HasPrefix(ws.AppURL, "http://localhost:"+orch.lastConfig.AppHostPort) {
		t.Errorf("app_url inesperada: %q (puerto %q)", ws.AppURL, orch.lastConfig.AppHostPort)
	}
}

func TestStartPracticeEnvironmentForExercise_WithoutAppPortPublishesNothing(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "")
	for _, appPort := range []int{0, 3000, 70000} {
		orch, svc := appPortFixture(appPort)

		ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "std-app", "subj-app", "ex-app")
		if err != nil {
			t.Fatalf("app_port %d: error inesperado al iniciar el laboratorio: %v", appPort, err)
		}
		if orch.lastConfig == nil {
			t.Fatal("no se envió configuración de contenedor al orquestador")
		}
		if orch.lastConfig.AppPort != 0 || orch.lastConfig.AppHostPort != "" {
			t.Errorf("app_port %d: no debía publicarse la aplicación (port=%d host=%q)", appPort, orch.lastConfig.AppPort, orch.lastConfig.AppHostPort)
		}
		if ws.AppURL != "" {
			t.Errorf("app_port %d: app_url debía quedar vacía, obtuvo %q", appPort, ws.AppURL)
		}
	}
}
