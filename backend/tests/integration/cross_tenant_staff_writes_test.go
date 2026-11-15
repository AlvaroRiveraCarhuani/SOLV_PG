package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupCrossTenantStaffServer(t *testing.T) (*httptest.Server, *database.Database) {
	t.Helper()

	db := openTestDB(t)

	moduleRepo := postgres.NewPostgresCourseModuleRepository(db.GetDB())
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	rubricRepo := postgres.NewPostgresRubricRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())

	moduleService := services.NewCourseModuleService(moduleRepo, exerciseRepo)
	moduleService.SetSubjectOwnership(subjectRepo)
	rubricService := services.NewDefaultRubricService(rubricRepo, exerciseRepo, db.GetDB())
	rubricService.SetSubjectOwnership(subjectRepo)

	handlers := httpdelivery.Handlers{
		CourseModuleHandler: httpdelivery.NewCourseModuleHandler(moduleService),
		RubricHandler:       httpdelivery.NewRubricHandler(rubricService),
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, withTestIdentityFromHeaders(r))
			})
		},
		AuditMiddleware: func(next http.Handler) http.Handler { return next },
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, db
}

// assignSubjectTeacher deja la materia a nombre del docente: la escritura de
// módulos y rúbricas se acota a quien la dicta.
func assignSubjectTeacher(t *testing.T, db *database.Database, subjectID, teacherID string) {
	t.Helper()

	if _, err := db.GetDB().Exec(`UPDATE subjects SET teacher_id = $1 WHERE id = $2`, teacherID, subjectID); err != nil {
		t.Fatalf("no se pudo asignar la materia al docente: %v", err)
	}
}

func seedModule(t *testing.T, db *database.Database, subjectID, title string, orderIndex int) string {
	t.Helper()

	id := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO course_modules (id, subject_id, title, description, order_index, pass_score)
		 VALUES ($1, $2, $3, '', $4, 60)`,
		id, subjectID, title, orderIndex,
	); err != nil {
		t.Fatalf("no se pudo sembrar el módulo: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.GetDB().Exec(`DELETE FROM course_modules WHERE id = $1`, id); err != nil {
			t.Logf("limpieza de módulo: %v", err)
		}
	})
	return id
}

func seedPersistentExercise(t *testing.T, db *database.Database, tenantID, subjectID, title string) string {
	t.Helper()

	templateID := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, status)
		 VALUES ($1, $2, $3, 'python:3.12-slim-bookworm', 512, 'IDE_PERSISTENTE', 'approved')`,
		templateID, tenantID, "Plantilla "+templateID[:8],
	); err != nil {
		t.Fatalf("no se pudo sembrar la plantilla: %v", err)
	}

	exerciseID := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO exercises (id, tenant_id, subject_id, template_id, title, environment_type)
		 VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE')`,
		exerciseID, tenantID, subjectID, templateID, title,
	); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM rubrics WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM lab_templates WHERE id = $1`, templateID)
	})
	return exerciseID
}

func moduleTitle(t *testing.T, db *database.Database, moduleID string) string {
	t.Helper()

	var title string
	if err := db.GetDB().QueryRow(`SELECT title FROM course_modules WHERE id = $1`, moduleID).Scan(&title); err != nil {
		t.Fatalf("no se pudo leer el módulo: %v", err)
	}
	return title
}

func rubricCriteriaName(t *testing.T, db *database.Database, exerciseID string) string {
	t.Helper()

	var criteriaJSON string
	if err := db.GetDB().QueryRow(`SELECT criteria::text FROM rubrics WHERE exercise_id = $1`, exerciseID).Scan(&criteriaJSON); err != nil {
		t.Fatalf("no se pudo leer la rúbrica: %v", err)
	}

	var criteria []map[string]any
	if err := json.Unmarshal([]byte(criteriaJSON), &criteria); err != nil {
		t.Fatalf("rúbrica ilegible: %v", err)
	}
	if len(criteria) == 0 {
		return ""
	}
	name, _ := criteria[0]["name"].(string)
	return name
}

func rubricPayload(name string) map[string]any {
	return map[string]any{
		"criteria": []map[string]any{
			{
				"id":          "c1",
				"name":        name,
				"description": "Resuelve los casos base",
				"weight":      100,
				"levels": []map[string]any{
					{"name": "Logrado", "score": 100, "description": "Completo"},
					{"name": "No logrado", "score": 0, "description": "Incompleto"},
				},
			},
		},
	}
}

// Un docente de otra institución tiene que rebotar en las dos escrituras que
// cuelgan de un id ajeno: módulos de curso y rúbricas. El caso positivo con el
// docente dueño confirma que el rechazo viene del alcance y no de una ruta rota.
func TestStaffWrites_NoCrossTenant(t *testing.T) {
	server, db := setupCrossTenantStaffServer(t)

	tenantA := seedTenant(t, db, "Docencia Institucion A")
	tenantB := seedTenant(t, db, "Docencia Institucion B")
	teacherA := seedUser(t, db, tenantA, "teacher", "docente-a-"+uuid.NewString()[:8]+"@test.local")
	teacherB := seedUser(t, db, tenantB, "teacher", "docente-b-"+uuid.NewString()[:8]+"@test.local")
	subjectA := seedSubject(t, db, tenantA, "DOC-A")
	subjectB := seedSubject(t, db, tenantB, "DOC-B")
	assignSubjectTeacher(t, db, subjectA, teacherA)
	assignSubjectTeacher(t, db, subjectB, teacherB)

	moduleA := seedModule(t, db, subjectA, "Modulo original de A", 1)
	moduleB := seedModule(t, db, subjectB, "Modulo original de B", 1)
	moduleBParaBorrar := seedModule(t, db, subjectB, "Modulo borrable de B", 2)
	exerciseB := seedPersistentExercise(t, db, tenantB, subjectB, "Practica de B")

	t.Run("el docente de A escribe su propio módulo", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/teacher/modules/"+moduleA,
			identityHeaders(tenantA, teacherA, "teacher"),
			map[string]any{"title": "Modulo actualizado por A", "description": "ok", "order_index": 2, "pass_score": 70})
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("el docente debe poder editar su propio módulo: %d", res.StatusCode)
		}
		if title := moduleTitle(t, db, moduleA); title != "Modulo actualizado por A" {
			t.Errorf("la edición propia no se guardó: %q", title)
		}
	})

	t.Run("el docente dueño escribe su módulo", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/teacher/modules/"+moduleB,
			identityHeaders(tenantB, teacherB, "teacher"),
			map[string]any{"title": "Modulo actualizado por B", "description": "ok", "order_index": 3, "pass_score": 70})
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("el docente dueño debe poder editar: %d", res.StatusCode)
		}
		if title := moduleTitle(t, db, moduleB); title != "Modulo actualizado por B" {
			t.Errorf("la edición propia no se guardó: %q", title)
		}
	})

	t.Run("el docente de A no edita el módulo de B", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/teacher/modules/"+moduleB,
			identityHeaders(tenantA, teacherA, "teacher"),
			map[string]any{"title": "Secuestrado desde A", "description": "no", "order_index": 9, "pass_score": 10})
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if title := moduleTitle(t, db, moduleB); title != "Modulo actualizado por B" {
			t.Errorf("el módulo cambió desde otra institución: %q", title)
		}
	})

	t.Run("el docente de A no borra el módulo de B", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodDelete, server.URL+"/api/v1/teacher/modules/"+moduleBParaBorrar,
			identityHeaders(tenantA, teacherA, "teacher"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if title := moduleTitle(t, db, moduleBParaBorrar); title != "Modulo borrable de B" {
			t.Errorf("el módulo desapareció desde otra institución: %q", title)
		}
	})

	t.Run("el docente de A no escribe la rúbrica de B", func(t *testing.T) {
		// El dueño deja una rúbrica legítima; después A intenta pisarla.
		ok := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+exerciseB+"/rubric",
			identityHeaders(tenantB, teacherB, "teacher"), rubricPayload("Correctitud"))
		_ = ok.Body.Close()
		if ok.StatusCode != http.StatusOK {
			t.Fatalf("el docente dueño debe poder guardar su rúbrica: %d", ok.StatusCode)
		}

		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+exerciseB+"/rubric",
			identityHeaders(tenantA, teacherA, "teacher"), rubricPayload("Secuestrada"))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if name := rubricCriteriaName(t, db, exerciseB); name != "Correctitud" {
			t.Errorf("la rúbrica cambió desde otra institución: %q", name)
		}
	})
}

// Dentro de una misma institución un docente tampoco escribe en la materia de un
// colega: el aislamiento por institución no cubría este caso, y es el que aparece
// en cuanto una facultad tiene dos docentes.
func TestStaffWrites_NoCrossTeacherSameTenant(t *testing.T) {
	server, db := setupCrossTenantStaffServer(t)

	tenant := seedTenant(t, db, "Docencia Misma Institucion")
	teacherOne := seedUser(t, db, tenant, "teacher", "docente-1-"+uuid.NewString()[:8]+"@test.local")
	teacherTwo := seedUser(t, db, tenant, "teacher", "docente-2-"+uuid.NewString()[:8]+"@test.local")
	admin := seedUser(t, db, tenant, "admin", "admin-"+uuid.NewString()[:8]+"@test.local")

	subjectOne := seedSubject(t, db, tenant, "SOLV-1")
	subjectTwo := seedSubject(t, db, tenant, "SOLV-2")
	assignSubjectTeacher(t, db, subjectOne, teacherOne)
	assignSubjectTeacher(t, db, subjectTwo, teacherTwo)

	moduleOne := seedModule(t, db, subjectOne, "Modulo de uno", 1)
	moduleTwo := seedModule(t, db, subjectTwo, "Modulo de dos", 1)
	exerciseTwo := seedPersistentExercise(t, db, tenant, subjectTwo, "Practica de dos")

	t.Run("cada docente edita su propia materia", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/teacher/modules/"+moduleOne,
			identityHeaders(tenant, teacherOne, "teacher"),
			map[string]any{"title": "Modulo propio de uno", "description": "ok", "order_index": 2, "pass_score": 70})
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("el docente debe poder editar su materia: %d", res.StatusCode)
		}
	})

	t.Run("el docente no edita el módulo del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/teacher/modules/"+moduleTwo,
			identityHeaders(tenant, teacherOne, "teacher"),
			map[string]any{"title": "Secuestrado", "description": "no", "order_index": 9, "pass_score": 10})
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if title := moduleTitle(t, db, moduleTwo); title != "Modulo de dos" {
			t.Errorf("el módulo del colega cambió: %q", title)
		}
	})

	t.Run("el docente no borra el módulo del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodDelete, server.URL+"/api/v1/teacher/modules/"+moduleOne,
			identityHeaders(tenant, teacherTwo, "teacher"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if title := moduleTitle(t, db, moduleOne); title == "" {
			t.Error("el módulo del colega desapareció")
		}
	})

	t.Run("el docente no escribe la rúbrica del colega", func(t *testing.T) {
		ok := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+exerciseTwo+"/rubric",
			identityHeaders(tenant, teacherTwo, "teacher"), rubricPayload("Correctitud"))
		_ = ok.Body.Close()
		if ok.StatusCode != http.StatusOK {
			t.Fatalf("el docente dueño debe poder guardar su rúbrica: %d", ok.StatusCode)
		}

		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+exerciseTwo+"/rubric",
			identityHeaders(tenant, teacherOne, "teacher"), rubricPayload("Secuestrada"))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if name := rubricCriteriaName(t, db, exerciseTwo); name != "Correctitud" {
			t.Errorf("la rúbrica del colega cambió: %q", name)
		}
	})

	t.Run("el administrador sí escribe cualquier materia de la institución", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/teacher/modules/"+moduleTwo,
			identityHeaders(tenant, admin, "admin"),
			map[string]any{"title": "Revisado por administración", "description": "ok", "order_index": 2, "pass_score": 70})
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("el administrador debe poder editar: %d", res.StatusCode)
		}
	})
}
