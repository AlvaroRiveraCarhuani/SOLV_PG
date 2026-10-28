package services_test

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// Los motores de base de datos se gobiernan desde el catálogo: el administrador registra
// el motor y el backend completa imagen versionada, clasificación, huella de memoria y
// modo de instalación sin exigirle conocer los detalles del contenedor.
func TestCreateTemplateComponent_GovernsDatabaseEngines(t *testing.T) {
	cases := []struct {
		name            string
		dto             domain.CreateTemplateComponentDTO
		wantEngine      string
		wantImage       string
		wantSubCategory string
		wantRamDeltaMB  int
	}{
		{
			name:            "mysql",
			dto:             domain.CreateTemplateComponentDTO{Name: "MySQL 8.4", Slug: "mysql-lab", Layer: domain.ComponentLayerDatabase},
			wantEngine:      "mysql",
			wantImage:       "mysql:8.4",
			wantSubCategory: "relational",
			wantRamDeltaMB:  128,
		},
		{
			name:            "mariadb",
			dto:             domain.CreateTemplateComponentDTO{Name: "MariaDB 11.4", Slug: "mariadb-lab", Layer: domain.ComponentLayerDatabase, ServiceEngine: "mariadb"},
			wantEngine:      "mariadb",
			wantImage:       "mariadb:11.4",
			wantSubCategory: "relational",
			wantRamDeltaMB:  128,
		},
		{
			name:            "mongodb inferido del nombre",
			dto:             domain.CreateTemplateComponentDTO{Name: "MongoDB 7.0", Slug: "documentos", Layer: domain.ComponentLayerDatabase},
			wantEngine:      "mongodb",
			wantImage:       "mongo:7.0",
			wantSubCategory: "nosql",
			wantRamDeltaMB:  128,
		},
		{
			name:            "redis inferido de la imagen",
			dto:             domain.CreateTemplateComponentDTO{Name: "Caché del laboratorio", Slug: "cache-lab", Layer: domain.ComponentLayerDatabase, BaseImage: "redis:7.2-alpine"},
			wantEngine:      "redis",
			wantImage:       "redis:7.2-alpine",
			wantSubCategory: "nosql",
			wantRamDeltaMB:  64,
		},
		{
			name:            "alias mongo",
			dto:             domain.CreateTemplateComponentDTO{Name: "Documentos del curso", Slug: "docs-curso", Layer: domain.ComponentLayerDatabase, ServiceEngine: "mongo"},
			wantEngine:      "mongodb",
			wantImage:       "mongo:7.0",
			wantSubCategory: "nosql",
			wantRamDeltaMB:  128,
		},
		{
			name:            "alias postgresql",
			dto:             domain.CreateTemplateComponentDTO{Name: "Relacional del aula", Slug: "aula-sql", Layer: domain.ComponentLayerDatabase, ServiceEngine: "postgresql"},
			wantEngine:      "postgres",
			wantImage:       "",
			wantSubCategory: "relational",
			wantRamDeltaMB:  50,
		},
		{
			name:            "sqlite en archivo",
			dto:             domain.CreateTemplateComponentDTO{Name: "SQLite 3", Slug: "sqlite-lab", Layer: domain.ComponentLayerDatabase},
			wantEngine:      "sqlite",
			wantImage:       "",
			wantSubCategory: "relational",
			wantRamDeltaMB:  0,
		},
		{
			name:            "sin base de datos",
			dto:             domain.CreateTemplateComponentDTO{Name: "Sin Base de Datos", Slug: "sin-bd", Layer: domain.ComponentLayerDatabase, ServiceEngine: "none"},
			wantEngine:      "none",
			wantImage:       "",
			wantSubCategory: "none",
			wantRamDeltaMB:  0,
		},
		{
			name:            "motor no reconocible cae en PostgreSQL de aula",
			dto:             domain.CreateTemplateComponentDTO{Name: "Motor genérico del curso", Slug: "motor-generico", Layer: domain.ComponentLayerDatabase, InstallMode: domain.InstallModePip, PackageNames: []string{"psycopg"}},
			wantEngine:      "postgres",
			wantImage:       "",
			wantSubCategory: "relational",
			wantRamDeltaMB:  50,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := services.NewAdminGovernanceService(nil, &mockAdminGovernanceRepo{})

			created, err := svc.CreateTemplateComponent(context.Background(), "tenant-1", "admin-1", tc.dto)
			if err != nil {
				t.Fatalf("error inesperado al registrar el motor: %v", err)
			}

			if created.ServiceEngine != tc.wantEngine {
				t.Errorf("service_engine = %q, esperado %q", created.ServiceEngine, tc.wantEngine)
			}
			if created.BaseImage != tc.wantImage {
				t.Errorf("base_image = %q, esperado %q", created.BaseImage, tc.wantImage)
			}
			if created.SubCategory != tc.wantSubCategory {
				t.Errorf("sub_category = %q, esperado %q", created.SubCategory, tc.wantSubCategory)
			}
			if created.RamDeltaMB != tc.wantRamDeltaMB {
				t.Errorf("ram_delta_mb = %d, esperado %d", created.RamDeltaMB, tc.wantRamDeltaMB)
			}
			if len(created.CompatibleRuntimes) != 1 || created.CompatibleRuntimes[0] != "universal" {
				t.Errorf("compatible_runtimes = %v, esperado [universal]", created.CompatibleRuntimes)
			}
			if created.InstallMode != domain.InstallModeNone {
				t.Errorf("install_mode = %q: un motor se aprovisiona como servicio, no se instala", created.InstallMode)
			}
			if strings.Contains(created.BaseImage, ":latest") {
				t.Errorf("la imagen del motor debe llevar tag fijo: %q", created.BaseImage)
			}
		})
	}
}

func TestUpdateTemplateComponent_RejectsUnknownDatabaseEngine(t *testing.T) {
	svc := services.NewAdminGovernanceService(nil, &mockAdminGovernanceRepo{})
	ctx := context.Background()

	_, err := svc.UpdateTemplateComponent(ctx, "tenant-1", "comp-1", "admin-1", domain.UpdateTemplateComponentDTO{
		Name:          "Motor propietario",
		ServiceEngine: "oracle",
	})
	if err == nil {
		t.Fatal("esperado rechazo de un motor fuera del catálogo soportado")
	}
	if !strings.Contains(err.Error(), "no soportado") {
		t.Errorf("mensaje de error inesperado: %v", err)
	}

	// El alias se canonicaliza antes de persistir.
	if _, err := svc.UpdateTemplateComponent(ctx, "tenant-1", "comp-1", "admin-1", domain.UpdateTemplateComponentDTO{
		Name:          "Documentos",
		ServiceEngine: "mongo",
	}); err != nil {
		t.Fatalf("el alias de un motor soportado debe aceptarse: %v", err)
	}
}
