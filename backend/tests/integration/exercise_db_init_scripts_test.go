package integration

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

const (
	dbScriptMigrationsDir = "../../migrations"
	// dbScriptVersionBefore es la última migración sin la tabla del script. El test
	// necesita detenerse justo antes para sembrar ejercicios con el formato viejo.
	dbScriptVersionBefore int64 = 56
)

// newIsolatedTestDB crea una base descartable en el mismo servidor de tests y la elimina
// al terminar. El aislamiento es por base y no por esquema porque la cadena de migraciones
// está atada al esquema public: la migración 00048 busca `workspaces` con
// `table_schema = 'public'` y en un esquema propio el renombre nunca ocurre.
func newIsolatedTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	base, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL de tests no disponible: %v", err)
		return nil
	}
	baseHandle := base.GetDB()
	t.Cleanup(func() { _ = baseHandle.Close() })

	name := "solv_test_dbscript_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if _, err := baseHandle.Exec(`CREATE DATABASE ` + name); err != nil {
		skipOrFailInfra(t, "no se pudo crear una base descartable de tests: %v", err)
		return nil
	}
	t.Cleanup(func() { _, _ = baseHandle.Exec(`DROP DATABASE IF EXISTS ` + name + ` WITH (FORCE)`) })

	parsed, err := url.Parse(getTestDSN())
	if err != nil {
		t.Fatalf("DSN de tests inválido: %v", err)
	}
	parsed.Path = "/" + name

	db, err := database.NewPostgresDB(parsed.String())
	if err != nil {
		t.Fatalf("conectar a la base descartable %s: %v", name, err)
	}
	handle := db.GetDB()
	t.Cleanup(func() { _ = handle.Close() })
	return handle
}

func dbScriptConfigText(t *testing.T, db *sqlx.DB, exerciseID string) string {
	t.Helper()
	var out string
	if err := db.QueryRow(`SELECT config::text FROM exercises WHERE id = $1`, exerciseID).Scan(&out); err != nil {
		t.Fatalf("leer config de %s: %v", exerciseID, err)
	}
	return out
}

func dbScriptRow(t *testing.T, db *sqlx.DB, exerciseID string) (string, int, bool) {
	t.Helper()
	var content string
	var size int
	err := db.QueryRow(`SELECT content, byte_size FROM exercise_db_init_scripts WHERE exercise_id = $1`, exerciseID).
		Scan(&content, &size)
	if err == sql.ErrNoRows {
		return "", 0, false
	}
	if err != nil {
		t.Fatalf("leer script de %s: %v", exerciseID, err)
	}
	return content, size, true
}

func dbScriptHasKey(t *testing.T, db *sqlx.DB, exerciseID, path string) bool {
	t.Helper()
	var found bool
	if err := db.QueryRow(`SELECT `+path+` IS NOT NULL FROM exercises WHERE id = $1`, exerciseID).Scan(&found); err != nil {
		t.Fatalf("leer %s de %s: %v", path, exerciseID, err)
	}
	return found
}

func TestExerciseDBInitScriptMigration_BackfillYReversion(t *testing.T) {
	db := newIsolatedTestDB(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialecto de goose: %v", err)
	}
	if err := goose.UpTo(db.DB, dbScriptMigrationsDir, dbScriptVersionBefore); err != nil {
		t.Fatalf("aplicar migraciones hasta %d: %v", dbScriptVersionBefore, err)
	}

	tenantID := uuid.NewString()
	subjectID := uuid.NewString()
	if _, err := db.ExecContext(ctx, `INSERT INTO tenants (id, name, slug) VALUES ($1, 'Integración', $2)`,
		tenantID, "mig-"+strings.ReplaceAll(tenantID, "-", "")[:8]); err != nil {
		t.Fatalf("sembrar tenant: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO subjects (id, tenant_id, name, code) VALUES ($1, $2, 'Bases', 'INT1')`,
		subjectID, tenantID); err != nil {
		t.Fatalf("sembrar materia: %v", err)
	}
	var templateID string
	if err := db.QueryRow(`SELECT id FROM lab_templates ORDER BY created_at LIMIT 1`).Scan(&templateID); err != nil {
		t.Fatalf("no hay plantilla sembrada por las migraciones: %v", err)
	}

	// Ejercicios con el script en el formato viejo: la clave canónica, el espejo y
	// ninguno de los dos.
	legacyIDE := uuid.NewString()
	legacyMirror := uuid.NewString()
	legacyNone := uuid.NewString()
	fixtures := []struct{ id, title, config string }{
		{legacyIDE, "legacy-ide", `{"ide":{"db_init_script":"CREATE TABLE a (id INT);","allow_resubmit":true}}`},
		{legacyMirror, "legacy-database", `{"database":{"engine":"postgres","init_script":"CREATE TABLE b (id INT);"}}`},
		{legacyNone, "sin-script", `{"ide":{"auto_checkpoint":true}}`},
	}
	for _, f := range fixtures {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO exercises (id, tenant_id, subject_id, template_id, title, config)
			 VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
			f.id, tenantID, subjectID, templateID, f.title, f.config); err != nil {
			t.Fatalf("sembrar ejercicio %s: %v", f.title, err)
		}
	}

	if err := goose.Up(db.DB, dbScriptMigrationsDir); err != nil {
		t.Fatalf("aplicar la migración del script: %v", err)
	}

	const scriptIDE = "CREATE TABLE a (id INT);"
	const scriptMirror = "CREATE TABLE b (id INT);"

	t.Run("backfill desde la clave canónica y el espejo", func(t *testing.T) {
		content, size, ok := dbScriptRow(t, db, legacyIDE)
		if !ok || content != scriptIDE {
			t.Fatalf("backfill de config.ide.db_init_script: ok=%v content=%q", ok, content)
		}
		if size != len(scriptIDE) {
			t.Errorf("byte_size = %d, esperado %d", size, len(scriptIDE))
		}

		content, _, ok = dbScriptRow(t, db, legacyMirror)
		if !ok || content != scriptMirror {
			t.Fatalf("backfill de config.database.init_script: ok=%v content=%q", ok, content)
		}

		if _, _, ok := dbScriptRow(t, db, legacyNone); ok {
			t.Error("un ejercicio sin script no debe generar fila")
		}
	})

	t.Run("el script sale del JSONB y el resto del config se conserva", func(t *testing.T) {
		if dbScriptHasKey(t, db, legacyIDE, `config->'ide'->'db_init_script'`) {
			t.Error("config.ide.db_init_script debe retirarse del JSONB")
		}
		if dbScriptHasKey(t, db, legacyMirror, `config->'database'->'init_script'`) {
			t.Error("config.database.init_script debe retirarse del JSONB")
		}
		if !dbScriptHasKey(t, db, legacyIDE, `config->'ide'->'allow_resubmit'`) {
			t.Error("el config debe conservar allow_resubmit")
		}
		if !dbScriptHasKey(t, db, legacyNone, `config->'ide'->'auto_checkpoint'`) {
			t.Error("el config debe conservar auto_checkpoint")
		}
		if strings.Contains(dbScriptConfigText(t, db, legacyIDE), "CREATE TABLE a") {
			t.Error("el JSONB no debe contener el texto del script")
		}
	})

	t.Run("el repositorio guarda, reconstruye, reemplaza y borra", func(t *testing.T) {
		repo := postgres.NewPostgresExerciseRepository(db)
		exerciseID := uuid.NewString()
		const firstScript = "CREATE TABLE repo_t (id INT);"

		ex := &domain.Exercise{
			ID:         exerciseID,
			TenantID:   tenantID,
			SubjectID:  &subjectID,
			TemplateID: templateID,
			Title:      "Con script",
			Type:       domain.ExerciseTypeDatabase,
			Config: domain.ExerciseConfig{
				IDE: &domain.IDEConfig{DBInitScript: firstScript, AllowResubmit: true},
			},
		}
		if err := repo.Create(ctx, ex); err != nil {
			t.Fatalf("Create: %v", err)
		}

		content, _, ok := dbScriptRow(t, db, exerciseID)
		if !ok || content != firstScript {
			t.Fatalf("Create debe guardar el script en su tabla: ok=%v content=%q", ok, content)
		}
		if strings.Contains(dbScriptConfigText(t, db, exerciseID), "repo_t") {
			t.Error("Create no debe dejar el script en el JSONB")
		}

		saved, err := repo.GetByIDAndTenant(ctx, exerciseID, tenantID)
		if err != nil {
			t.Fatalf("GetByIDAndTenant: %v", err)
		}
		if saved == nil || saved.Config.IDE == nil || saved.Config.IDE.DBInitScript != firstScript {
			t.Fatalf("la lectura por id debe reconstruir el script: %+v", saved)
		}
		if !saved.Config.IDE.AllowResubmit {
			t.Error("la lectura por id debe conservar la política de entrega")
		}

		saved.Config.IDE.DBInitScript = "CREATE TABLE repo_t2 (id INT);"
		if err := repo.Update(ctx, saved); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if content, _, ok := dbScriptRow(t, db, exerciseID); !ok || content != saved.Config.IDE.DBInitScript {
			t.Fatalf("Update debe reemplazar el contenido: ok=%v content=%q", ok, content)
		}

		saved.Config.IDE.DBInitScript = "   "
		if err := repo.Update(ctx, saved); err != nil {
			t.Fatalf("Update vacío: %v", err)
		}
		if _, _, ok := dbScriptRow(t, db, exerciseID); ok {
			t.Error("un script vacío debe eliminar la fila")
		}
	})

	t.Run("la reversión restaura el config y se puede reaplicar", func(t *testing.T) {
		if err := goose.DownTo(db.DB, dbScriptMigrationsDir, dbScriptVersionBefore); err != nil {
			t.Fatalf("revertir la migración del script: %v", err)
		}

		var table *string
		if err := db.QueryRow(`SELECT to_regclass('exercise_db_init_scripts')::text`).Scan(&table); err != nil {
			t.Fatalf("consultar la tabla del script: %v", err)
		}
		if table != nil {
			t.Fatalf("la reversión debe eliminar la tabla, quedó %q", *table)
		}

		for _, tc := range []struct {
			id     string
			script string
		}{
			{legacyIDE, scriptIDE},
			{legacyMirror, scriptMirror},
		} {
			var restored string
			if err := db.QueryRow(`SELECT config->'ide'->>'db_init_script' FROM exercises WHERE id = $1`, tc.id).
				Scan(&restored); err != nil {
				t.Fatalf("leer el script restaurado de %s: %v", tc.id, err)
			}
			if restored != tc.script {
				t.Errorf("la reversión debe restaurar el script en la clave canónica: %q, esperado %q", restored, tc.script)
			}
		}

		if err := goose.Up(db.DB, dbScriptMigrationsDir); err != nil {
			t.Fatalf("reaplicar la migración del script: %v", err)
		}
		if content, _, ok := dbScriptRow(t, db, legacyIDE); !ok || content != scriptIDE {
			t.Fatalf("reaplicar la migración debe volver a hacer el backfill: ok=%v content=%q", ok, content)
		}
	})

}
