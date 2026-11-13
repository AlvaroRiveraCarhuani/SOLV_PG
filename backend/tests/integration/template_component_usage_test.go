package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// El conteo de uso y la propiedad por tenant se verifican contra PostgreSQL:
// son condiciones SQL y una prueba con repositorio simulado no las cubre.
func TestTemplateComponentUsageAndTenantOwnership(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL no disponible para la prueba de uso del catálogo: %v", err)
		return
	}
	defer func() { _ = db.GetDB().Close() }()

	repo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	ctx := context.Background()

	tenantID := uuid.NewString()
	suffix := tenantID[:8]
	slug := "it-usage-" + suffix

	if _, errTenant := db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'Uso de catalogo', $2)
	`, tenantID, slug); errTenant != nil {
		t.Fatalf("no se pudo sembrar el tenant de la prueba: %v", errTenant)
		return
	}

	component, err := repo.CreateTemplateComponent(ctx, tenantID, &domain.TemplateComponent{
		Slug:        slug,
		Layer:       domain.ComponentLayerRuntime,
		SubCategory: "backend",
		Name:        "Runtime de prueba " + suffix,
		BaseImage:   "python:3.12-slim-bookworm",
	})
	if err != nil {
		t.Fatalf("no se pudo registrar la tecnologia de prueba: %v", err)
	}

	templateID := uuid.NewString()
	stackConfig, errJSON := json.Marshal(map[string]string{"runtime_id": slug})
	if errJSON != nil {
		t.Fatalf("no se pudo serializar stack_config: %v", errJSON)
	}
	if _, errSeed := db.GetDB().Exec(`
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, status, stack_config)
		VALUES ($1, $2::uuid, $3, 'python:3.12-slim-bookworm', 512, 'approved', $4::jsonb)
	`, templateID, tenantID, "Plantilla de uso "+suffix, string(stackConfig)); errSeed != nil {
		t.Fatalf("no se pudo sembrar la plantilla que referencia la tecnologia: %v", errSeed)
	}

	cleanup := func() {
		_, _ = db.GetDB().Exec(`DELETE FROM lab_templates WHERE id = $1`, templateID)
		_, _ = db.GetDB().Exec(`DELETE FROM template_components WHERE id = $1`, component.ID)
		_, _ = db.GetDB().Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	}
	defer cleanup()

	t.Run("el listado informa cuantas plantillas usan la tecnologia", func(t *testing.T) {
		items, errList := repo.ListTemplateComponents(ctx, tenantID, "", true)
		if errList != nil {
			t.Fatalf("error al listar el catalogo: %v", errList)
		}
		var found *domain.TemplateComponent
		for _, item := range items {
			if item.ID == component.ID {
				found = item
				break
			}
		}
		if found == nil {
			t.Fatalf("la tecnologia %q no aparecio en el listado", slug)
		}
		if found.UsageCount != 1 {
			t.Fatalf("esperado 1 uso por slug, obtenido %d", found.UsageCount)
		}
	})

	t.Run("borrar una tecnologia en uso se rechaza", func(t *testing.T) {
		if errDel := repo.DeleteTemplateComponent(ctx, tenantID, component.ID); !errors.Is(errDel, domain.ErrComponentInUse) {
			t.Fatalf("esperado ErrComponentInUse, obtenido %v", errDel)
		}
		if _, errStill := repo.GetTemplateComponentByIDOrSlug(ctx, tenantID, component.ID); errStill != nil {
			t.Fatalf("la tecnologia no debia borrarse: %v", errStill)
		}
	})

	t.Run("sin plantillas que la referencien el borrado procede", func(t *testing.T) {
		if _, errDelTemplate := db.GetDB().Exec(`DELETE FROM lab_templates WHERE id = $1`, templateID); errDelTemplate != nil {
			t.Fatalf("no se pudo quitar la plantilla de prueba: %v", errDelTemplate)
		}
		if errDel := repo.DeleteTemplateComponent(ctx, tenantID, component.ID); errDel != nil {
			t.Fatalf("el borrado de una tecnologia sin uso debia proceder, obtuvo %v", errDel)
		}
	})

	t.Run("un tenant ajeno no puede editar el catalogo global", func(t *testing.T) {
		globalSuffix := uuid.NewString()[:8]
		global, errCreate := repo.CreateTemplateComponent(ctx, domain.DefaultTenantID, &domain.TemplateComponent{
			Slug:        "it-global-" + globalSuffix,
			Layer:       domain.ComponentLayerAddon,
			SubCategory: "framework",
			Name:        "Complemento global " + globalSuffix,
			InstallMode: domain.InstallModePip,
			PackageNames: []string{
				"fastapi",
			},
		})
		if errCreate != nil {
			t.Fatalf("no se pudo registrar el complemento global: %v", errCreate)
		}
		defer func() {
			_, _ = db.GetDB().Exec(`DELETE FROM template_components WHERE id = $1`, global.ID)
		}()

		otherTenant := uuid.NewString()
		_, errUpdate := repo.UpdateTemplateComponent(ctx, otherTenant, global.ID, domain.UpdateTemplateComponentDTO{
			Name: "Intento de edicion ajena",
		}, "")
		if !errors.Is(errUpdate, domain.ErrComponentReadOnly) {
			t.Fatalf("esperado ErrComponentReadOnly para un tenant ajeno, obtenido %v", errUpdate)
		}

		unchanged, errGet := repo.GetTemplateComponentByIDOrSlug(ctx, domain.DefaultTenantID, global.ID)
		if errGet != nil {
			t.Fatalf("no se pudo releer el complemento global: %v", errGet)
		}
		if unchanged.Name == "Intento de edicion ajena" {
			t.Fatalf("la edicion ajena modifico el catalogo global")
		}
	})
}
