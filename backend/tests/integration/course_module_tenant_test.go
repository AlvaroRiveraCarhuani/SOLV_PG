package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func openTestDB(t *testing.T) *database.Database {
	t.Helper()
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("base de tests no disponible: %v", err)
	}
	return db
}

func courseModuleTenantContext(tenantID string) context.Context {
	return context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
}

// Los módulos no tienen tenant_id: cuelgan de una materia. Antes, cualquier
// consulta por id los alcanzaba desde otra institución; este test fija que el
// filtro por la materia se mantenga.
func TestCourseModules_NoCrossTenant(t *testing.T) {
	db := openTestDB(t)
	repo := postgres.NewPostgresCourseModuleRepository(db.GetDB())

	tenantA := seedTenant(t, db, "Modulos Institucion A")
	tenantB := seedTenant(t, db, "Modulos Institucion B")
	subjectB := seedSubject(t, db, tenantB, "MOD-"+uuid.NewString()[:5])

	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM module_prerequisites WHERE module_id IN (SELECT id FROM course_modules WHERE subject_id = $1)`, subjectB)
		_, _ = db.GetDB().Exec(`DELETE FROM course_modules WHERE subject_id = $1`, subjectB)
	})

	module := &domain.CourseModule{
		ID:          uuid.NewString(),
		SubjectID:   subjectB,
		Title:       "Modulo de B",
		Description: "Solo de la institucion B",
		OrderIndex:  1,
		PassScore:   60,
	}
	if err := repo.CreateModule(courseModuleTenantContext(tenantB), module); err != nil {
		t.Fatalf("la institución dueña no pudo crear el módulo: %v", err)
	}

	// La institución A conoce el id pero no debe alcanzar el módulo.
	if _, err := repo.GetModuleByID(courseModuleTenantContext(tenantA), module.ID); err == nil {
		t.Error("la institución A leyó un módulo de la institución B")
	}

	modules, err := repo.ListModulesBySubject(courseModuleTenantContext(tenantA), subjectB)
	if err != nil {
		t.Fatalf("listar módulos falló: %v", err)
	}
	if len(modules) != 0 {
		t.Errorf("la institución A listó %d módulos de la institución B", len(modules))
	}

	robado := *module
	robado.Title = "Titulo secuestrado"
	if err := repo.UpdateModule(courseModuleTenantContext(tenantA), &robado); err == nil {
		t.Error("la institución A actualizó un módulo de la institución B")
	}

	if err := repo.DeleteModule(courseModuleTenantContext(tenantA), module.ID); err == nil {
		t.Error("la institución A borró un módulo de la institución B")
	}

	// La institución dueña sigue viendo el módulo intacto.
	verificado, err := repo.GetModuleByID(courseModuleTenantContext(tenantB), module.ID)
	if err != nil {
		t.Fatalf("la institución dueña perdió su módulo: %v", err)
	}
	if verificado.Title != "Modulo de B" {
		t.Errorf("el título cambió desde otra institución: %q", verificado.Title)
	}

	// Y la materia de B no se puede usar para crear un módulo desde A.
	ajeno := &domain.CourseModule{
		ID:         uuid.NewString(),
		SubjectID:  subjectB,
		Title:      "Modulo colado",
		OrderIndex: 2,
		PassScore:  60,
	}
	if err := repo.CreateModule(courseModuleTenantContext(tenantA), ajeno); err == nil {
		t.Error("la institución A creó un módulo dentro de una materia de la institución B")
	}
}
