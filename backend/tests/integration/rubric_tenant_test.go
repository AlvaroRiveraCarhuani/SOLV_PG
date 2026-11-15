package integration

import (
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// Las rúbricas cuelgan de un ejercicio y el ejercicio de una institución. Sin el
// join, un id de ejercicio ajeno devolvía la rúbrica de otra institución y
// permitía sobrescribirla.
func TestRubrics_NoCrossTenant(t *testing.T) {
	db := openTestDB(t)
	repo := postgres.NewPostgresRubricRepository(db.GetDB())

	tenantA := seedTenant(t, db, "Rubricas Institucion A")
	tenantB := seedTenant(t, db, "Rubricas Institucion B")
	subjectB := seedSubject(t, db, tenantB, "RUB-"+uuid.NewString()[:5])

	templateB := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, status)
		 VALUES ($1, $2, 'Plantilla rubricas', 'python:3.12-slim-bookworm', 512, 'JUEZ_EFIMERO', 'approved')`,
		templateB, tenantB,
	); err != nil {
		t.Fatalf("no se pudo sembrar la plantilla: %v", err)
	}

	exerciseB := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO exercises (id, tenant_id, subject_id, template_id, title) VALUES ($1, $2, $3, $4, $5)`,
		exerciseB, tenantB, subjectB, templateB, "Ejercicio de B",
	); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM rubrics WHERE exercise_id = $1`, exerciseB)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseB)
		_, _ = db.GetDB().Exec(`DELETE FROM lab_templates WHERE id = $1`, templateB)
	})

	rubric := &domain.Rubric{
		ExerciseID: uuid.MustParse(exerciseB),
		Criteria: []domain.RubricCriterion{
			{ID: "c1", Name: "Correctitud", Description: "Resuelve el caso base", Weight: 1},
		},
	}
	if err := repo.CreateOrUpdate(courseModuleTenantContext(tenantB), rubric); err != nil {
		t.Fatalf("la institución dueña no pudo guardar su rúbrica: %v", err)
	}

	if _, err := repo.GetByExerciseID(courseModuleTenantContext(tenantA), uuid.MustParse(exerciseB)); err == nil {
		t.Error("la institución A leyó la rúbrica de un ejercicio de la institución B")
	}

	robada := &domain.Rubric{
		ExerciseID: uuid.MustParse(exerciseB),
		Criteria: []domain.RubricCriterion{
			{ID: "c1", Name: "Secuestrada", Description: "Otra institución", Weight: 1},
		},
	}
	if err := repo.CreateOrUpdate(courseModuleTenantContext(tenantA), robada); err == nil {
		t.Error("la institución A sobrescribió la rúbrica de la institución B")
	}

	verificada, err := repo.GetByExerciseID(courseModuleTenantContext(tenantB), uuid.MustParse(exerciseB))
	if err != nil {
		t.Fatalf("la institución dueña perdió su rúbrica: %v", err)
	}
	if len(verificada.Criteria) != 1 || verificada.Criteria[0].Name != "Correctitud" {
		t.Errorf("la rúbrica cambió desde otra institución: %+v", verificada.Criteria)
	}
}
