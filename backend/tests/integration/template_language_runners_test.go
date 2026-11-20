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

// La migración 00070 crea template_language_runners y el repositorio debe
// guardar, leer y copiar los ejecutores. Son condiciones de PostgreSQL real:
// claves foráneas, restricciones y cascada no las cubre un repositorio simulado.
func TestTemplateLanguageRunnersPersistence(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL no disponible para la prueba de ejecutores de lenguaje: %v", err)
		return
	}
	defer func() { _ = db.GetDB().Close() }()

	ctx := context.Background()
	repo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())

	t.Run("la migración deja la tabla y su versión aplicada", func(t *testing.T) {
		var version int64
		if errVersion := db.GetDB().GetContext(ctx, &version, `
			SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1
		`); errVersion != nil {
			t.Fatalf("no se pudo leer la versión de migraciones: %v", errVersion)
		}
		if version < 70 {
			t.Fatalf("la migración 00070 no está aplicada: versión actual %d", version)
		}

		var columns int
		if errColumns := db.GetDB().GetContext(ctx, &columns, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_name = 'template_language_runners'
			  AND column_name IN ('template_id', 'language', 'entrypoint', 'docker_image', 'timeout_ms', 'memory_limit_mb')
		`); errColumns != nil {
			t.Fatalf("no se pudo inspeccionar la tabla: %v", errColumns)
		}
		if columns != 6 {
			t.Fatalf("template_language_runners no tiene las 6 columnas esperadas: %d", columns)
		}
	})

	tenantID := uuid.NewString()
	suffix := tenantID[:8]
	adminID := uuid.NewString()
	if _, errTenant := db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'Ejecutores de lenguaje', $2)
	`, tenantID, "it-runners-"+suffix); errTenant != nil {
		t.Fatalf("no se pudo sembrar el tenant de la prueba: %v", errTenant)
	}
	// lab_templates.requested_by referencia users: el alta exige un administrador real.
	if _, errUser := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Admin', 'Ejecutores', $2, 'admin', $3)
	`, adminID, "runners_"+suffix+"@uab.edu.bo", tenantID); errUser != nil {
		t.Fatalf("no se pudo sembrar el administrador de la prueba: %v", errUser)
	}
	defer func() {
		_, _ = db.GetDB().Exec(`DELETE FROM lab_templates WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id = $1`, adminID)
		_, _ = db.GetDB().Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	}()

	templateName := "Entorno multilingüe " + suffix

	created, errCreate := repo.CreateOfficialTemplate(ctx, tenantID, adminID, domain.CreateOfficialTemplateDTO{
		Name:              templateName,
		DockerImage:       "python:3.12-slim-bookworm",
		BaseRamMB:         256,
		TargetEnvironment: "JUEZ_EFIMERO",
		Entrypoint:        "python3 solution.py",
		LanguageRunners: []domain.LanguageRunner{
			{Language: "cpp", Entrypoint: "g++ -O2 solution.cpp -o solution && ./solution", TimeoutMS: 4000, MemoryLimitMB: 128},
			{Language: "python", Entrypoint: "python3 solution.py"},
		},
	})
	if errCreate != nil {
		t.Fatalf("no se pudo crear la plantilla multilingüe: %v", errCreate)
	}
	if len(created.LanguageRunners) != 2 {
		t.Fatalf("el alta devolvió %d ejecutores, esperaba 2", len(created.LanguageRunners))
	}

	t.Run("el alta persiste y la lectura devuelve los ejecutores ordenados", func(t *testing.T) {
		got, errGet := repo.GetTemplateByID(ctx, created.ID)
		if errGet != nil {
			t.Fatalf("no se pudo releer la plantilla: %v", errGet)
		}
		if len(got.LanguageRunners) != 2 {
			t.Fatalf("la lectura devolvió %d ejecutores, esperaba 2", len(got.LanguageRunners))
		}
		if got.LanguageRunners[0].Language != "cpp" || got.LanguageRunners[1].Language != "python" {
			t.Fatalf("orden inesperado de lenguajes: %q, %q", got.LanguageRunners[0].Language, got.LanguageRunners[1].Language)
		}
		cpp := got.LanguageRunners[0]
		if cpp.Entrypoint != "g++ -O2 solution.cpp -o solution && ./solution" {
			t.Errorf("comando de cpp inesperado: %q", cpp.Entrypoint)
		}
		if cpp.TimeoutMS != 4000 || cpp.MemoryLimitMB != 128 {
			t.Errorf("límites de cpp inesperados: timeout=%d memoria=%d", cpp.TimeoutMS, cpp.MemoryLimitMB)
		}
		if cpp.DockerImage != "" {
			t.Errorf("una imagen vacía significa imagen de catálogo, se obtuvo %q", cpp.DockerImage)
		}
	})

	t.Run("el listado administrativo incluye los lenguajes", func(t *testing.T) {
		items, errList := repo.ListTemplates(ctx, tenantID, "", "")
		if errList != nil {
			t.Fatalf("error al listar plantillas: %v", errList)
		}
		var found *domain.AdminTemplateReviewItem
		for _, item := range items {
			if item.ID == created.ID {
				found = item
				break
			}
		}
		if found == nil {
			t.Fatalf("la plantilla %s no apareció en el listado", created.ID)
		}
		if len(found.LanguageRunners) != 2 {
			t.Fatalf("el listado devolvió %d ejecutores, esperaba 2", len(found.LanguageRunners))
		}
	})

	t.Run("duplicar la plantilla copia sus ejecutores", func(t *testing.T) {
		copied, errDup := repo.DuplicateTemplate(ctx, tenantID, created.ID, adminID)
		if errDup != nil {
			t.Fatalf("no se pudo duplicar la plantilla: %v", errDup)
		}
		if copied.ID == created.ID {
			t.Fatal("la copia reutilizó el identificador original")
		}
		got, errGet := repo.GetTemplateByID(ctx, copied.ID)
		if errGet != nil {
			t.Fatalf("no se pudo releer la copia: %v", errGet)
		}
		if len(got.LanguageRunners) != 2 {
			t.Fatalf("la copia tiene %d ejecutores, esperaba 2", len(got.LanguageRunners))
		}
		if got.LanguageRunners[0].Language != "cpp" || got.LanguageRunners[1].Language != "python" {
			t.Fatalf("la copia no conservó los lenguajes: %q, %q", got.LanguageRunners[0].Language, got.LanguageRunners[1].Language)
		}
		if got.LanguageRunners[0].Entrypoint != "g++ -O2 solution.cpp -o solution && ./solution" {
			t.Errorf("la copia no conservó el comando de cpp: %q", got.LanguageRunners[0].Entrypoint)
		}
		if got.LanguageRunners[1].Entrypoint != "python3 solution.py" {
			t.Errorf("la copia no conservó el comando de python: %q", got.LanguageRunners[1].Entrypoint)
		}
	})

	t.Run("la base rechaza un ejecutor fuera de los límites", func(t *testing.T) {
		_, errInsert := db.GetDB().Exec(`
			INSERT INTO template_language_runners (template_id, language, entrypoint, timeout_ms)
			VALUES ($1, 'ruby', 'ruby solution.rb', 60001)
		`, created.ID)
		if errInsert == nil {
			t.Fatal("la restricción de timeout no rechazó 60001 ms")
		}

		_, errEmpty := db.GetDB().Exec(`
			INSERT INTO template_language_runners (template_id, language, entrypoint)
			VALUES ($1, 'go', '')
		`, created.ID)
		if errEmpty == nil {
			t.Fatal("la restricción de entrypoint no rechazó un comando vacío")
		}
	})

	t.Run("borrar la plantilla borra sus ejecutores", func(t *testing.T) {
		if _, errDelete := db.GetDB().Exec(`DELETE FROM lab_templates WHERE id = $1`, created.ID); errDelete != nil {
			t.Fatalf("no se pudo borrar la plantilla: %v", errDelete)
		}
		var remaining int
		if errCount := db.GetDB().GetContext(ctx, &remaining, `
			SELECT COUNT(*) FROM template_language_runners WHERE template_id = $1
		`, created.ID); errCount != nil {
			t.Fatalf("no se pudo contar los ejecutores: %v", errCount)
		}
		if remaining != 0 {
			t.Fatalf("quedaron %d ejecutores huérfanos tras borrar la plantilla", remaining)
		}
	})
}
