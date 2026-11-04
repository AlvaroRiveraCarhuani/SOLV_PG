package integration

import (
	"context"
	"os"
	"testing"

	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestSubjectSectionComposesCourseCode fija la identidad de un curso con sección.
//
// Dos paralelos de la misma materia en el mismo periodo son dos cursos, no dos
// materias: el código visible se compone ("SIS-500-1", "SIS-500-2") y el catálogo
// guarda la materia una sola vez ("SIS-500").
func TestSubjectSectionComposesCourseCode(t *testing.T) {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbDSN = getTestDSN()
	}

	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}
	sqlDB := db.GetDB()

	const tenantID = "90000000-0000-0000-0000-000000000001"
	_, _ = sqlDB.Exec(`DELETE FROM subjects WHERE tenant_id = $1`, tenantID)
	_, _ = sqlDB.Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	if _, err := sqlDB.Exec(
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, "Secciones", "secciones-test",
	); err != nil {
		t.Fatalf("failed to insert tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = sqlDB.Exec(`DELETE FROM subjects WHERE tenant_id = $1`, tenantID)
		_, _ = sqlDB.Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	svc := services.NewSubjectService(postgres.NewPostgresSubjectRepository(sqlDB))
	ctx := context.Background()

	primera, err := svc.Create(ctx, tenantID, services.NewSubjectInput{
		Name: "Bases de Datos", Code: "sis-500", Section: "1",
	})
	if err != nil {
		t.Fatalf("creating first section: %v", err)
	}
	if primera.Code != "SIS-500-1" {
		t.Fatalf("expected composed code SIS-500-1, got %q", primera.Code)
	}
	if primera.CatalogID == nil {
		t.Fatal("expected the course to be linked to a subject catalog")
	}

	segunda, err := svc.Create(ctx, tenantID, services.NewSubjectInput{
		Name: "Bases de Datos", Code: "SIS-500", Section: "2",
	})
	if err != nil {
		t.Fatalf("creating second section: %v", err)
	}
	if segunda.Code != "SIS-500-2" {
		t.Fatalf("expected composed code SIS-500-2, got %q", segunda.Code)
	}
	if segunda.CatalogID == nil || *segunda.CatalogID != *primera.CatalogID {
		t.Fatal("both sections must point to the same subject catalog entry")
	}

	var catalogCode string
	if err := sqlDB.Get(&catalogCode, `SELECT code FROM subject_catalog WHERE id = $1`, *primera.CatalogID); err != nil {
		t.Fatalf("reading subject catalog: %v", err)
	}
	if catalogCode != "SIS-500" {
		t.Fatalf("the catalog must keep the materia code without section, got %q", catalogCode)
	}

	var materias int
	if err := sqlDB.Get(&materias, `SELECT COUNT(*) FROM subject_catalog WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("counting subject catalog rows: %v", err)
	}
	if materias != 1 {
		t.Fatalf("expected one materia for two sections, got %d", materias)
	}

	// Un curso sin sección conserva el código limpio.
	unico, err := svc.Create(ctx, tenantID, services.NewSubjectInput{
		Name: "Redes", Code: "SIS-600",
	})
	if err != nil {
		t.Fatalf("creating a subject without section: %v", err)
	}
	if unico.Code != "SIS-600" {
		t.Fatalf("expected SIS-600 without section, got %q", unico.Code)
	}

	// La sección viaja dentro del código, así que no admite espacios ni barras.
	if _, err := svc.Create(ctx, tenantID, services.NewSubjectInput{
		Name: "Bases de Datos", Code: "SIS-500", Section: "1 A",
	}); err == nil {
		t.Fatal("expected an invalid section to be rejected")
	}
}
