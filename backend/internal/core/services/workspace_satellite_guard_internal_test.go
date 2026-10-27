package services

import (
	"context"
	"strings"
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

	provision, err := service.ensureExerciseSatelliteDB(ctx, "std-1", "subj-1", &domain.Template{ID: "tpl-plain"}, nil)
	if err != nil {
		t.Fatalf("unexpected error without satellite requirements: %v", err)
	}
	if provision.PostgresDBName != "" || len(provision.Env) != 0 || provision.UsesManagedContainer {
		t.Fatalf("expected empty satellite provision, got %+v", provision)
	}
}

func TestEnsureExerciseSatelliteDB_WithoutDatabaseConnectionKeepsLegacyBehaviour(t *testing.T) {
	// Los tests y los despliegues sin PostgreSQL inyectado construyen el
	// servicio con db == nil: no se puede asegurar la BD satélite, pero el
	// arranque del workspace no debe romperse por eso.
	service := &WorkspaceService{}
	ctx := context.Background()

	provision, err := service.ensureExerciseSatelliteDB(ctx, "std-1", "subj-1", templateWithDatabase("postgres"), nil)
	if err != nil {
		t.Fatalf("unexpected error without DB connection: %v", err)
	}
	if provision.PostgresDBName != "" {
		t.Fatalf("expected empty satellite name, got %q", provision.PostgresDBName)
	}

	provision, err = service.ensureExerciseSatelliteDB(
		ctx, "std-1", "subj-1", nil, exerciseWithDBInitScript("CREATE TABLE t (id INT);"),
	)
	if err != nil {
		t.Fatalf("unexpected error without DB connection and db_init_script: %v", err)
	}
	if provision.PostgresDBName != "" {
		t.Fatalf("expected empty satellite name, got %q", provision.PostgresDBName)
	}
}

func TestTemplateManagedEngineDetection(t *testing.T) {
	cases := []struct {
		name     string
		template *domain.Template
		want     string
	}{
		{name: "nil template", template: nil, want: ""},
		{name: "mysql gestionado", template: templateWithDatabase("mysql"), want: "mysql"},
		{name: "mariadb gestionado", template: templateWithDatabase("mariadb"), want: "mariadb"},
		{name: "mongodb gestionado", template: templateWithDatabase("mongodb"), want: "mongodb"},
		{name: "postgres no es contenedor gestionado", template: templateWithDatabase("postgres"), want: ""},
		{name: "sin servicios", template: &domain.Template{ID: "tpl-empty"}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := templateManagedEngine(tc.template); got != tc.want {
				t.Fatalf("templateManagedEngine() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSatelliteIsolationIdentityIsDeterministic(t *testing.T) {
	user, password := satelliteIsolationIdentity("std-1", "subj-1")
	if !strings.HasPrefix(user, "u_") || len(user) != 14 {
		t.Fatalf("usuario aislado inesperado: %q", user)
	}
	if len(password) != 16 {
		t.Fatalf("contraseña derivada inesperada de largo %d", len(password))
	}

	repeatUser, repeatPassword := satelliteIsolationIdentity("std-1", "subj-1")
	if user != repeatUser || password != repeatPassword {
		t.Fatalf("las credenciales deben ser deterministas entre arranques")
	}

	otherUser, otherPassword := satelliteIsolationIdentity("std-2", "subj-1")
	if user == otherUser || password == otherPassword {
		t.Fatalf("cada alumno debe recibir credenciales distintas")
	}

	if satelliteDatabaseName("std-1", "subj-1") == satelliteDatabaseName("std-1", "subj-2") {
		t.Fatal("cada materia debe recibir su propio espacio de datos")
	}
}
