package services

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
)

func templateWithDatabase(engine string) *domain.Template {
	return &domain.Template{
		ID: "tpl-satellite",
		ServicesConfig: domain.ServicesConfig{
			Services: []domain.ServiceRequirement{
				{Category: "database", Engine: engine},
			},
		},
	}
}

func exerciseWithDBInitScript(script string) *domain.Exercise {
	return &domain.Exercise{
		ID: "ex-satellite",
		Config: domain.ExerciseConfig{
			IDE: &domain.IDEConfig{DBInitScript: script},
		},
	}
}

func TestTemplateDeclaresPostgres(t *testing.T) {
	cases := []struct {
		name     string
		template *domain.Template
		want     bool
	}{
		{name: "nil template", template: nil, want: false},
		{name: "postgres service", template: templateWithDatabase("postgres"), want: true},
		{name: "mysql service", template: templateWithDatabase("mysql"), want: false},
		{name: "no services", template: &domain.Template{ID: "tpl-empty"}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := templateDeclaresPostgres(tc.template); got != tc.want {
				t.Fatalf("templateDeclaresPostgres() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnsureExerciseSatelliteDB_SkipsWhenNotNeeded(t *testing.T) {
	service := &WorkspaceService{}
	ctx := context.Background()

	name, err := service.ensureExerciseSatelliteDB(ctx, "std-1", "subj-1", &domain.Template{ID: "tpl-plain"}, nil)
	if err != nil {
		t.Fatalf("unexpected error without satellite requirements: %v", err)
	}
	if name != "" {
		t.Fatalf("expected empty satellite name, got %q", name)
	}
}

func TestEnsureExerciseSatelliteDB_WithoutDatabaseConnectionKeepsLegacyBehaviour(t *testing.T) {
	// Los tests y los despliegues sin PostgreSQL inyectado construyen el
	// servicio con db == nil: no se puede asegurar la BD satélite, pero el
	// arranque del workspace no debe romperse por eso.
	service := &WorkspaceService{}
	ctx := context.Background()

	name, err := service.ensureExerciseSatelliteDB(ctx, "std-1", "subj-1", templateWithDatabase("postgres"), nil)
	if err != nil {
		t.Fatalf("unexpected error without DB connection: %v", err)
	}
	if name != "" {
		t.Fatalf("expected empty satellite name, got %q", name)
	}

	name, err = service.ensureExerciseSatelliteDB(
		ctx, "std-1", "subj-1", nil, exerciseWithDBInitScript("CREATE TABLE t (id INT);"),
	)
	if err != nil {
		t.Fatalf("unexpected error without DB connection and db_init_script: %v", err)
	}
	if name != "" {
		t.Fatalf("expected empty satellite name, got %q", name)
	}
}
