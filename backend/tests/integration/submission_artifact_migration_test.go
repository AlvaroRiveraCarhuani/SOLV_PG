package integration

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

// TestSubmissionArtifactMigrationRoundTrip ejerce la migración 00068 sobre una
// base descartable: una entrega con artefacto se rechaza si mezcla texto, la
// reversión elimina esas filas y quita las columnas, y reaplicar las restaura.
func TestSubmissionArtifactMigrationRoundTrip(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	handle := db.GetDB()
	seed := seedWorkspaceTestBase(t, handle)

	key := "submissions/" + seed.tenantID + "/" + seed.exerciseA + "/test.tar.gz"
	hash := strings.Repeat("a", 64)

	// Con artefacto el texto tiene que quedar vacío: mezclar ambos no dice cuál
	// es el árbol entregado.
	if _, err := handle.Exec(`
		INSERT INTO submissions (tenant_id, exercise_id, student_id, code, review_status, artifact_key, artifact_sha256, artifact_bytes, artifact_files)
		VALUES ($1, $2, $3, 'print(1)', 'pending', $4, $5, 10, 1)
	`, seed.tenantID, seed.exerciseA, seed.studentA, key, hash); err == nil {
		t.Fatal("una entrega con artefacto y texto debía rechazarse")
	}

	if _, err := handle.Exec(`
		INSERT INTO submissions (tenant_id, exercise_id, student_id, code, review_status, artifact_key, artifact_sha256, artifact_bytes, artifact_files)
		VALUES ($1, $2, $3, '', 'pending', $4, $5, 10, 1)
	`, seed.tenantID, seed.exerciseA, seed.studentA, key, hash); err != nil {
		t.Fatalf("insertar la entrega con artefacto: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialecto de goose: %v", err)
	}
	if err := goose.DownTo(handle.DB, workspaceArtifactMigrationsDir, 66); err != nil {
		t.Fatalf("revertir 00068: %v", err)
	}

	var columns int
	if err := handle.Get(&columns, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'submissions' AND column_name LIKE 'artifact_%'
	`); err != nil {
		t.Fatalf("contar columnas de la entrega: %v", err)
	}
	if columns != 0 {
		t.Fatalf("la reversión dejó %d columnas de artefacto en submissions", columns)
	}
	var remaining int
	if err := handle.Get(&remaining, `SELECT COUNT(*) FROM submissions WHERE exercise_id = $1`, seed.exerciseA); err != nil {
		t.Fatalf("contar entregas tras la reversión: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("la reversión dejó %d entregas con artefacto", remaining)
	}

	if err := goose.Up(handle.DB, workspaceArtifactMigrationsDir); err != nil {
		t.Fatalf("reaplicar 00068: %v", err)
	}
	if err := handle.Get(&columns, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'submissions' AND column_name LIKE 'artifact_%'
	`); err != nil {
		t.Fatalf("contar columnas tras reaplicar: %v", err)
	}
	if columns != 4 {
		t.Fatalf("tras reaplicar se esperaban 4 columnas de artefacto, hay %d", columns)
	}
}
