package services

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestTemplateServiceBindings(t *testing.T) {
	cases := []struct {
		name     string
		template *domain.Template
		want     []string
	}{
		{name: "nil template", template: nil, want: nil},
		{name: "postgres service", template: templateWithDatabase("postgres"), want: []string{"postgres"}},
		{name: "mysql service", template: templateWithDatabase("mysql"), want: []string{"mysql"}},
		{name: "no services", template: &domain.Template{ID: "tpl-empty"}, want: nil},
		{name: "unknown engine skipped", template: &domain.Template{ServicesConfig: domain.ServicesConfig{Services: []domain.ServiceRequirement{{Engine: "oracle"}}}}, want: nil},
		{
			name: "primary first",
			template: &domain.Template{ServicesConfig: domain.ServicesConfig{Services: []domain.ServiceRequirement{
				{Category: "cache", Engine: "redis"},
				{Category: "database", Engine: "postgres"},
			}}},
			want: []string{"postgres", "redis"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := templateServiceBindings(tc.template)
			var engines []string
			for _, binding := range got {
				engines = append(engines, binding.Engine)
			}
			if !reflect.DeepEqual(engines, tc.want) {
				t.Fatalf("templateServiceBindings() = %v, want %v", engines, tc.want)
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
	if len(provision.Env) != 0 || provision.UsesManagedContainer {
		t.Fatalf("expected empty satellite provision, got %+v", provision)
	}
}

// Un despliegue sin orquestador de satélites (tests, mocks) no rompe el arranque del
// laboratorio: el motor se omite y queda registrado, nunca se cae a la base de la
// plataforma.
func TestEnsureExerciseSatelliteDB_WithoutOrchestratorOmitsEngine(t *testing.T) {
	service := &WorkspaceService{}
	ctx := context.Background()

	for _, tc := range []struct {
		name     string
		template *domain.Template
		exercise *domain.Exercise
	}{
		{name: "plantilla con postgres", template: templateWithDatabase("postgres")},
		{name: "ejercicio con db_init_script", exercise: exerciseWithDBInitScript("CREATE TABLE t (id INT);")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provision, err := service.ensureExerciseSatelliteDB(ctx, "std-1", "subj-1", tc.template, tc.exercise)
			if err != nil {
				t.Fatalf("unexpected error without satellite orchestrator: %v", err)
			}
			if len(provision.Env) != 0 || provision.UsesManagedContainer {
				t.Fatalf("expected empty satellite provision, got %+v", provision)
			}
		})
	}
}

func TestTemplateManagedEnginesDetection(t *testing.T) {
	cases := []struct {
		name     string
		template *domain.Template
		want     []string
	}{
		{name: "nil template", template: nil, want: nil},
		{name: "mysql gestionado", template: templateWithDatabase("mysql"), want: []string{"mysql"}},
		{name: "mariadb gestionado", template: templateWithDatabase("mariadb"), want: []string{"mariadb"}},
		{name: "mongodb gestionado", template: templateWithDatabase("mongodb"), want: []string{"mongodb"}},
		{name: "postgres gestionado", template: templateWithDatabase("postgres"), want: []string{"postgres"}},
		{name: "sin servicios", template: &domain.Template{ID: "tpl-empty"}, want: nil},
		{
			name: "dos gestionados",
			template: &domain.Template{ServicesConfig: domain.ServicesConfig{Services: []domain.ServiceRequirement{
				{Category: "cache", Engine: "redis"},
				{Category: "database", Engine: "mongodb"},
			}}},
			want: []string{"mongodb", "redis"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := templateManagedEngines(tc.template); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("templateManagedEngines() = %v, want %v", got, tc.want)
			}
		})
	}
}

// El scale-to-zero no puede apagar un motor que un laboratorio acaba de pedir: mientras
// el laboratorio se aprovisiona todavía no figura como activo en la base de datos.
func TestManagedSatelliteUseGrace(t *testing.T) {
	service := NewWorkspaceService(nil, nil, nil)

	if service.managedSatelliteRecentlyUsed("postgres") {
		t.Fatal("un motor nunca usado no está dentro del margen de gracia")
	}

	service.noteManagedSatelliteUse("postgres")
	if !service.managedSatelliteRecentlyUsed("postgres") {
		t.Fatal("un motor recién usado debe quedar protegido")
	}
	if service.managedSatelliteRecentlyUsed("redis") {
		t.Fatal("el margen es por motor: lo usado en uno no protege a los demás")
	}

	service.satelliteUseMu.Lock()
	service.satelliteLastUse["postgres"] = time.Now().Add(-managedSatelliteIdleGrace - time.Second)
	service.satelliteUseMu.Unlock()
	if service.managedSatelliteRecentlyUsed("postgres") {
		t.Fatal("pasado el margen de gracia el motor vuelve a ser apagable")
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
