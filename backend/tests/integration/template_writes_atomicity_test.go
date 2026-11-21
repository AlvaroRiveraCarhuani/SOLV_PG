package integration

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// Las tres escrituras de plantillas que tocan más de una tabla (alta, duplicado
// y promoción) se apoyan en una transacción. Estas pruebas fuerzan un fallo en
// la segunda escritura contra PostgreSQL real y comprueban que la primera no
// quedó guardada. Un repositorio simulado no cubre restricciones ni rollback.
func TestTemplateWritesAreAtomic(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL no disponible para las pruebas de plantillas: %v", err)
		return
	}
	defer func() { _ = db.GetDB().Close() }()

	ctx := context.Background()
	repo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())

	tenantID := uuid.NewString()
	suffix := tenantID[:8]
	adminID := uuid.NewString()
	categoryID := uuid.NewString()
	if _, errTenant := db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'Plantillas atómicas', $2)
	`, tenantID, "it-atomic-"+suffix); errTenant != nil {
		t.Fatalf("no se pudo sembrar el tenant de la prueba: %v", errTenant)
	}
	if _, errUser := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Admin', 'Atómico', $2, 'admin', $3)
	`, adminID, "atomic_"+suffix+"@uab.edu.bo", tenantID); errUser != nil {
		t.Fatalf("no se pudo sembrar el administrador de la prueba: %v", errUser)
	}
	if _, errCategory := db.GetDB().Exec(`
		INSERT INTO template_categories (id, tenant_id, name) VALUES ($1, $2, 'Categoría atómica')
	`, categoryID, tenantID); errCategory != nil {
		t.Fatalf("no se pudo sembrar la categoría de la prueba: %v", errCategory)
	}
	defer func() {
		_, _ = db.GetDB().Exec(`DELETE FROM template_models WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM lab_templates WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM template_categories WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id = $1`, adminID)
		_, _ = db.GetDB().Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	}()

	// dropTrigger deja la base como estaba aunque el subtest falle a mitad.
	dropTrigger := func(name string) {
		_, _ = db.GetDB().Exec(`DROP TRIGGER IF EXISTS ` + name + ` ON lab_templates`)
		_, _ = db.GetDB().Exec(`DROP TRIGGER IF EXISTS ` + name + ` ON template_language_runners`)
		_, _ = db.GetDB().Exec(`DROP FUNCTION IF EXISTS ` + name + `()`)
	}
	t.Run("el alta no deja la plantilla si un ejecutor es inválido", func(t *testing.T) {
		name := "Alta atómica " + suffix
		trigger := "solv_it_block_template_delete"
		defer dropTrigger(trigger)
		// El código anterior compensaba el fallo con un DELETE; se bloquea para
		// comprobar que ahora la reversión la hace la transacción, no un borrado
		// que puede fallar en silencio.
		if _, errTrigger := db.GetDB().Exec(`
			CREATE OR REPLACE FUNCTION ` + trigger + `() RETURNS trigger AS $$
			BEGIN RAISE EXCEPTION 'borrado bloqueado por la prueba'; END;
			$$ LANGUAGE plpgsql
		`); errTrigger != nil {
			t.Fatalf("no se pudo crear el disparador de la prueba: %v", errTrigger)
		}
		if _, errTrigger := db.GetDB().Exec(`
			CREATE TRIGGER ` + trigger + ` BEFORE DELETE ON lab_templates
			FOR EACH ROW EXECUTE FUNCTION ` + trigger + `()
		`); errTrigger != nil {
			t.Fatalf("no se pudo instalar el disparador de la prueba: %v", errTrigger)
		}

		_, errCreate := repo.CreateOfficialTemplate(ctx, tenantID, adminID, domain.CreateOfficialTemplateDTO{
			Name:              name,
			DockerImage:       "python:3.12-slim-bookworm",
			BaseRamMB:         256,
			TargetEnvironment: "JUEZ_EFIMERO",
			Entrypoint:        "python3 solution.py",
			LanguageRunners: []domain.LanguageRunner{
				{Language: "java", Entrypoint: "javac Solution.java && java Solution", TimeoutMS: 90000},
			},
		})
		if errCreate == nil {
			t.Fatal("la base debía rechazar el ejecutor con timeout fuera de rango")
		}

		var templates int
		if errCount := db.GetDB().GetContext(ctx, &templates, `
			SELECT COUNT(*) FROM lab_templates WHERE tenant_id = $1 AND name = $2
		`, tenantID, name); errCount != nil {
			t.Fatalf("no se pudo contar las plantillas: %v", errCount)
		}
		if templates != 0 {
			t.Fatalf("el alta fallida dejó %d plantilla(s) sin ejecutores", templates)
		}
	})

	t.Run("el duplicado no deja la copia si fallan sus ejecutores", func(t *testing.T) {
		original, errCreate := repo.CreateOfficialTemplate(ctx, tenantID, adminID, domain.CreateOfficialTemplateDTO{
			Name:              "Original atómico " + suffix,
			DockerImage:       "python:3.12-slim-bookworm",
			BaseRamMB:         256,
			TargetEnvironment: "JUEZ_EFIMERO",
			Entrypoint:        "python3 solution.py",
			LanguageRunners: []domain.LanguageRunner{
				{Language: "python", Entrypoint: "python3 solution.py"},
			},
		})
		if errCreate != nil {
			t.Fatalf("no se pudo crear la plantilla original: %v", errCreate)
		}

		trigger := "solv_it_block_runner_copy"
		defer dropTrigger(trigger)
		if _, errTrigger := db.GetDB().Exec(`
			CREATE OR REPLACE FUNCTION ` + trigger + `() RETURNS trigger AS $$
			BEGIN RAISE EXCEPTION 'copia de ejecutores bloqueada por la prueba'; END;
			$$ LANGUAGE plpgsql
		`); errTrigger != nil {
			t.Fatalf("no se pudo crear el disparador de la prueba: %v", errTrigger)
		}
		if _, errTrigger := db.GetDB().Exec(`
			CREATE TRIGGER ` + trigger + ` BEFORE INSERT ON template_language_runners
			FOR EACH ROW EXECUTE FUNCTION ` + trigger + `()
		`); errTrigger != nil {
			t.Fatalf("no se pudo instalar el disparador de la prueba: %v", errTrigger)
		}

		_, errDuplicate := repo.DuplicateTemplate(ctx, tenantID, original.ID, adminID)
		if errDuplicate == nil {
			t.Fatal("el duplicado debía fallar al copiar los ejecutores")
		}

		var copies int
		if errCount := db.GetDB().GetContext(ctx, &copies, `
			SELECT COUNT(*) FROM lab_templates WHERE tenant_id = $1 AND name LIKE '(Copia)%'
		`, tenantID); errCount != nil {
			t.Fatalf("no se pudo contar las copias: %v", errCount)
		}
		if copies != 0 {
			t.Fatalf("el duplicado fallido dejó %d copia(s) sin ejecutores", copies)
		}
	})

	t.Run("la promoción no deja el modelo si no puede vincular la plantilla", func(t *testing.T) {
		template, errCreate := repo.CreateOfficialTemplate(ctx, tenantID, adminID, domain.CreateOfficialTemplateDTO{
			Name:              "Promovible atómica " + suffix,
			DockerImage:       "python:3.12-slim-bookworm",
			BaseRamMB:         256,
			TargetEnvironment: "JUEZ_EFIMERO",
			Entrypoint:        "python3 solution.py",
			LanguageRunners: []domain.LanguageRunner{
				{Language: "python", Entrypoint: "python3 solution.py"},
			},
		})
		if errCreate != nil {
			t.Fatalf("no se pudo crear la plantilla a promover: %v", errCreate)
		}
		if _, errStatus := db.GetDB().Exec(`
			UPDATE lab_templates SET status = 'APROBADA' WHERE id = $1
		`, template.ID); errStatus != nil {
			t.Fatalf("no se pudo aprobar la plantilla de la prueba: %v", errStatus)
		}

		trigger := "solv_it_block_template_link"
		if _, errTrigger := db.GetDB().Exec(`
			CREATE OR REPLACE FUNCTION ` + trigger + `() RETURNS trigger AS $$
			BEGIN RAISE EXCEPTION 'vínculo bloqueado por la prueba'; END;
			$$ LANGUAGE plpgsql
		`); errTrigger != nil {
			t.Fatalf("no se pudo crear el disparador de la prueba: %v", errTrigger)
		}
		if _, errTrigger := db.GetDB().Exec(`
			CREATE TRIGGER ` + trigger + ` BEFORE UPDATE ON lab_templates
			FOR EACH ROW EXECUTE FUNCTION ` + trigger + `()
		`); errTrigger != nil {
			t.Fatalf("no se pudo instalar el disparador de la prueba: %v", errTrigger)
		}

		_, errPromote := repo.PromoteTemplateToModel(ctx, tenantID, template.ID, adminID, domain.PromoteTemplateToModelDTO{
			CategoryID: categoryID,
			Title:      "Modelo atómico " + suffix,
		})
		dropTrigger(trigger)
		if errPromote == nil {
			t.Fatal("la promoción debía fallar al vincular la plantilla")
		}

		var models int
		if errCount := db.GetDB().GetContext(ctx, &models, `
			SELECT COUNT(*) FROM template_models WHERE source_template_id = $1
		`, template.ID); errCount != nil {
			t.Fatalf("no se pudo contar los modelos: %v", errCount)
		}
		if models != 0 {
			t.Fatalf("la promoción fallida dejó %d modelo(s) huérfano(s)", models)
		}

		// Sin el disparador, la misma promoción confirma las dos escrituras.
		promoted, errRetry := repo.PromoteTemplateToModel(ctx, tenantID, template.ID, adminID, domain.PromoteTemplateToModelDTO{
			CategoryID: categoryID,
			Title:      "Modelo atómico " + suffix,
		})
		if errRetry != nil {
			t.Fatalf("la promoción válida fue rechazada: %v", errRetry)
		}
		var linkedModelID string
		if errLink := db.GetDB().GetContext(ctx, &linkedModelID, `
			SELECT COALESCE(model_id::text, '') FROM lab_templates WHERE id = $1
		`, template.ID); errLink != nil {
			t.Fatalf("no se pudo leer el vínculo de la plantilla: %v", errLink)
		}
		if linkedModelID != promoted.ID {
			t.Fatalf("la plantilla quedó vinculada a %q, esperaba %q", linkedModelID, promoted.ID)
		}
	})
}
