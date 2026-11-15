package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupExerciseScopeServer(t *testing.T) (*httptest.Server, *database.Database) {
	t.Helper()

	db := openTestDB(t)
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())

	evalService := services.NewEvaluationService(exerciseRepo, nil, nil, nil)
	evalService.SetSubjectOwnership(subjectRepo)
	evalService.SetAdminGovernanceRepository(postgres.NewPostgresAdminGovernanceRepository(db.GetDB()))

	handlers := &httpdelivery.Handlers{
		EvaluationHandler: httpdelivery.NewEvaluationHandler(evalService, validator.New()),
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, withTestIdentityFromHeaders(r))
			})
		},
		AuditMiddleware: func(next http.Handler) http.Handler { return next },
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, handlers)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, db
}

func exerciseBody(title, subjectID string) map[string]any {
	body := map[string]any{
		"title":         title,
		"description":   "Prueba de alcance docente",
		"type":          "algorithm",
		"status":        "draft",
		"language":      "python",
		"boilerplate":   "def solve(x):\n    pass\n",
		"time_limit_ms": 1000,
		"config": map[string]any{
			"algorithm": map[string]any{
				"time_limit_ms": 1000,
				"test_cases": []map[string]any{
					{"input": "1", "expected_output": "1", "is_hidden": false},
				},
			},
		},
	}
	if subjectID != "" {
		body["subject_id"] = subjectID
	}
	return body
}

func exerciseStatus(t *testing.T, db *database.Database, exerciseID string) string {
	t.Helper()

	var status string
	if err := db.GetDB().QueryRow(`SELECT status FROM exercises WHERE id = $1`, exerciseID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado del ejercicio: %v", err)
	}
	return status
}

func exerciseCaseCount(t *testing.T, db *database.Database, exerciseID string) int {
	t.Helper()

	var count int
	if err := db.GetDB().QueryRow(
		`SELECT COALESCE(jsonb_array_length(config->'algorithm'->'test_cases'), 0) FROM exercises WHERE id = $1`,
		exerciseID,
	).Scan(&count); err != nil {
		t.Fatalf("no se pudo contar los casos del ejercicio: %v", err)
	}
	return count
}

func setExerciseStatus(t *testing.T, db *database.Database, exerciseID, status string) {
	t.Helper()

	if _, err := db.GetDB().Exec(`UPDATE exercises SET status = $2 WHERE id = $1`, exerciseID, status); err != nil {
		t.Fatalf("no se pudo fijar el estado del ejercicio: %v", err)
	}
}

func exerciseSubject(t *testing.T, db *database.Database, exerciseID string) string {
	t.Helper()

	var subjectID string
	if err := db.GetDB().QueryRow(`SELECT COALESCE(subject_id::text, '') FROM exercises WHERE id = $1`, exerciseID).Scan(&subjectID); err != nil {
		t.Fatalf("no se pudo leer el ejercicio: %v", err)
	}
	return subjectID
}

func exerciseTitle(t *testing.T, db *database.Database, exerciseID string) string {
	t.Helper()

	var title string
	if err := db.GetDB().QueryRow(`SELECT title FROM exercises WHERE id = $1`, exerciseID).Scan(&title); err != nil {
		t.Fatalf("no se pudo leer el título del ejercicio: %v", err)
	}
	return title
}

func countExercisesInSubject(t *testing.T, db *database.Database, subjectID string) int {
	t.Helper()

	var count int
	if err := db.GetDB().QueryRow(`SELECT COUNT(*) FROM exercises WHERE subject_id = $1`, subjectID).Scan(&count); err != nil {
		t.Fatalf("no se pudo contar los ejercicios: %v", err)
	}
	return count
}

// Un ejercicio vive en una materia, y la materia tiene docente. Crear en la
// materia de un colega o mover un ejercicio hacia ella es la misma escritura
// cruzada: responde 404 y no toca la fila.
func TestExerciseWrites_NoCrossTeacher(t *testing.T) {
	server, db := setupExerciseScopeServer(t)

	tenant := seedTenant(t, db, "Docencia Ejercicios")
	teacherOne := seedUser(t, db, tenant, "teacher", "ejercicios-1-"+uuid.NewString()[:8]+"@test.local")
	teacherTwo := seedUser(t, db, tenant, "teacher", "ejercicios-2-"+uuid.NewString()[:8]+"@test.local")
	admin := seedUser(t, db, tenant, "admin", "ejercicios-admin-"+uuid.NewString()[:8]+"@test.local")

	subjectOne := seedSubject(t, db, tenant, "EJ-1")
	subjectTwo := seedSubject(t, db, tenant, "EJ-2")
	assignSubjectTeacher(t, db, subjectOne, teacherOne)
	assignSubjectTeacher(t, db, subjectTwo, teacherTwo)

	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE subject_id IN ($1, $2)`, subjectOne, subjectTwo)
	})

	var ownExerciseID string

	t.Run("el docente crea un ejercicio en su materia", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises",
			identityHeaders(tenant, teacherOne, "teacher"), exerciseBody("Practica de uno", subjectOne))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusCreated {
			t.Fatalf("se esperaba 201, se obtuvo %d", res.StatusCode)
		}

		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		ownExerciseID = created.Data.ID
		if ownExerciseID == "" {
			t.Fatal("la respuesta no trae el id del ejercicio")
		}
	})

	t.Run("el docente no crea un ejercicio en la materia del colega", func(t *testing.T) {
		before := countExercisesInSubject(t, db, subjectTwo)
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises",
			identityHeaders(tenant, teacherOne, "teacher"), exerciseBody("Intruso", subjectTwo))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if after := countExercisesInSubject(t, db, subjectTwo); after != before {
			t.Fatalf("se creó un ejercicio en la materia ajena: %d → %d", before, after)
		}
	})

	t.Run("el administrador sí crea en la materia del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises",
			identityHeaders(tenant, admin, "admin"), exerciseBody("Practica de dos", subjectTwo))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusCreated {
			t.Fatalf("se esperaba 201, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el docente edita un ejercicio de su materia", func(t *testing.T) {
		if ownExerciseID == "" {
			t.Skip("sin ejercicio propio del paso anterior")
		}
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+ownExerciseID,
			identityHeaders(tenant, teacherOne, "teacher"), exerciseBody("Practica de uno revisada", subjectOne))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}
		if title := exerciseTitle(t, db, ownExerciseID); title != "Practica de uno revisada" {
			t.Fatalf("la edición propia no se guardó: %q", title)
		}
	})

	t.Run("el docente no mueve su ejercicio a la materia del colega", func(t *testing.T) {
		if ownExerciseID == "" {
			t.Skip("sin ejercicio propio del paso anterior")
		}
		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+ownExerciseID,
			identityHeaders(tenant, teacherOne, "teacher"), exerciseBody("Practica de uno", subjectTwo))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404 al mover, se obtuvo %d", res.StatusCode)
		}
		if subject := exerciseSubject(t, db, ownExerciseID); subject != subjectOne {
			t.Fatalf("el ejercicio cambió de materia: %q", subject)
		}
	})

	t.Run("el docente no edita un ejercicio de la materia del colega", func(t *testing.T) {
		var foreignID string
		if err := db.GetDB().QueryRow(`SELECT id::text FROM exercises WHERE subject_id = $1 LIMIT 1`, subjectTwo).Scan(&foreignID); err != nil {
			t.Skipf("sin ejercicio ajeno sembrado: %v", err)
		}
		originalTitle := exerciseTitle(t, db, foreignID)

		res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/exercises/"+foreignID,
			identityHeaders(tenant, teacherOne, "teacher"), exerciseBody("Secuestrado", subjectTwo))
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if title := exerciseTitle(t, db, foreignID); title != originalTitle {
			t.Fatalf("el ejercicio ajeno cambió: %q → %q", originalTitle, title)
		}
	})
}

// Publicar, despublicar, cargar casos, generar casos desde un script y el
// dry-run escriben el ejercicio o su contenido: tampoco se ejercen sobre la
// materia de un colega, y el rechazo no deja cambios.
func TestExerciseStateWrites_NoCrossTeacher(t *testing.T) {
	server, db := setupExerciseScopeServer(t)

	tenant := seedTenant(t, db, "Estado de Ejercicios")
	teacherOne := seedUser(t, db, tenant, "teacher", "estado-1-"+uuid.NewString()[:8]+"@test.local")
	teacherTwo := seedUser(t, db, tenant, "teacher", "estado-2-"+uuid.NewString()[:8]+"@test.local")
	admin := seedUser(t, db, tenant, "admin", "estado-admin-"+uuid.NewString()[:8]+"@test.local")

	subjectOne := seedSubject(t, db, tenant, "EE-1")
	subjectTwo := seedSubject(t, db, tenant, "EE-2")
	assignSubjectTeacher(t, db, subjectOne, teacherOne)
	assignSubjectTeacher(t, db, subjectTwo, teacherTwo)

	foreignID := seedPersistentExercise(t, db, tenant, subjectTwo, "Ejercicio de dos")
	ownID := seedPersistentExercise(t, db, tenant, subjectOne, "Ejercicio de uno")
	setExerciseStatus(t, db, foreignID, "published")
	setExerciseStatus(t, db, ownID, "published")

	cases := []map[string]any{{"input": "1", "expected_output": "1", "is_hidden": false}}

	t.Run("no despublica el ejercicio del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises/"+foreignID+"/unpublish",
			identityHeaders(tenant, teacherOne, "teacher"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if status := exerciseStatus(t, db, foreignID); status != "published" {
			t.Fatalf("el ejercicio ajeno cambió de estado: %q", status)
		}
	})

	t.Run("no publica el ejercicio del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises/"+foreignID+"/publish",
			identityHeaders(tenant, teacherOne, "teacher"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if status := exerciseStatus(t, db, foreignID); status != "published" {
			t.Fatalf("el ejercicio ajeno cambió de estado: %q", status)
		}
	})

	t.Run("no carga casos en el ejercicio del colega", func(t *testing.T) {
		before := exerciseCaseCount(t, db, foreignID)
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises/"+foreignID+"/test-cases/bulk",
			identityHeaders(tenant, teacherOne, "teacher"), cases)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if after := exerciseCaseCount(t, db, foreignID); after != before {
			t.Fatalf("se agregaron casos al ejercicio ajeno: %d → %d", before, after)
		}
	})

	t.Run("no genera casos desde script en el ejercicio del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost,
			server.URL+"/api/v1/teacher/exercises/"+foreignID+"/generate-from-script",
			identityHeaders(tenant, teacherOne, "teacher"), map[string]any{"script": "print(1)"})
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("no inicia un dry-run sobre el ejercicio del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises/"+foreignID+"/dry-run",
			identityHeaders(tenant, teacherOne, "teacher"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("despublica el ejercicio de su materia", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises/"+ownID+"/unpublish",
			identityHeaders(tenant, teacherOne, "teacher"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}
		if status := exerciseStatus(t, db, ownID); status != "draft" {
			t.Fatalf("el ejercicio propio no quedó en borrador: %q", status)
		}
	})

	t.Run("carga casos en un ejercicio de su materia", func(t *testing.T) {
		created := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises",
			identityHeaders(tenant, teacherOne, "teacher"), exerciseBody("Practica con casos", subjectOne))
		defer func() { _ = created.Body.Close() }()
		if created.StatusCode != http.StatusCreated {
			t.Fatalf("no se pudo crear el ejercicio propio: %d", created.StatusCode)
		}

		var payload struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(created.Body).Decode(&payload); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}

		res := doJSON(t, server.Client(), http.MethodPost,
			server.URL+"/api/v1/exercises/"+payload.Data.ID+"/test-cases/bulk",
			identityHeaders(tenant, teacherOne, "teacher"), cases)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el administrador despublica el ejercicio del colega", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/v1/exercises/"+foreignID+"/unpublish",
			identityHeaders(tenant, admin, "admin"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}
		if status := exerciseStatus(t, db, foreignID); status != "draft" {
			t.Fatalf("el administrador no pudo despublicar: %q", status)
		}
	})
}

// La importación masiva escribe hasta 50 ejercicios en la materia del path y
// también responde sobre ella en dry-run: misma regla que el alta individual.
func TestExerciseImport_NoCrossTeacher(t *testing.T) {
	server, db := setupExerciseScopeServer(t)

	tenant := seedTenant(t, db, "Importacion de Ejercicios")
	teacherOne := seedUser(t, db, tenant, "teacher", "importa-1-"+uuid.NewString()[:8]+"@test.local")
	teacherTwo := seedUser(t, db, tenant, "teacher", "importa-2-"+uuid.NewString()[:8]+"@test.local")
	admin := seedUser(t, db, tenant, "admin", "importa-admin-"+uuid.NewString()[:8]+"@test.local")
	student := seedUser(t, db, tenant, "student", "importa-alumno-"+uuid.NewString()[:8]+"@test.local")

	subjectOne := seedSubject(t, db, tenant, "IM-1")
	subjectTwo := seedSubject(t, db, tenant, "IM-2")
	assignSubjectTeacher(t, db, subjectOne, teacherOne)
	assignSubjectTeacher(t, db, subjectTwo, teacherTwo)

	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE subject_id IN ($1, $2)`, subjectOne, subjectTwo)
	})

	payload := []map[string]any{{
		"title":     "Importado uno",
		"statement": "Dado N imprimir N",
		"language":  "python",
		"config": map[string]any{
			"algorithm": map[string]any{
				"test_cases": []map[string]any{{"input": "1", "expected_output": "1"}},
			},
		},
	}}

	importURL := func(subjectID string) string {
		return server.URL + "/api/v1/teacher/courses/" + subjectID + "/exercises/import"
	}

	t.Run("el docente importa en su materia", func(t *testing.T) {
		before := countExercisesInSubject(t, db, subjectOne)
		res := doJSON(t, server.Client(), http.MethodPost, importURL(subjectOne),
			identityHeaders(tenant, teacherOne, "teacher"), payload)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusCreated {
			t.Fatalf("se esperaba 201, se obtuvo %d", res.StatusCode)
		}
		if after := countExercisesInSubject(t, db, subjectOne); after != before+1 {
			t.Fatalf("la importación propia no se guardó: %d → %d", before, after)
		}
	})

	t.Run("el docente no importa en la materia del colega", func(t *testing.T) {
		before := countExercisesInSubject(t, db, subjectTwo)
		res := doJSON(t, server.Client(), http.MethodPost, importURL(subjectTwo),
			identityHeaders(tenant, teacherOne, "teacher"), payload)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
		if after := countExercisesInSubject(t, db, subjectTwo); after != before {
			t.Fatalf("se importaron ejercicios en la materia ajena: %d → %d", before, after)
		}
	})

	t.Run("el dry-run tampoco confirma la materia ajena", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, importURL(subjectTwo)+"?dry_run=true",
			identityHeaders(tenant, teacherOne, "teacher"), payload)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("se esperaba 404, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("un alumno no puede importar", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, importURL(subjectOne),
			identityHeaders(tenant, student, "student"), payload)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("se esperaba 403, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el administrador importa en la materia del colega", func(t *testing.T) {
		before := countExercisesInSubject(t, db, subjectTwo)
		res := doJSON(t, server.Client(), http.MethodPost, importURL(subjectTwo),
			identityHeaders(tenant, admin, "admin"), payload)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusCreated {
			t.Fatalf("se esperaba 201, se obtuvo %d", res.StatusCode)
		}
		if after := countExercisesInSubject(t, db, subjectTwo); after != before+1 {
			t.Fatalf("el administrador no importó: %d → %d", before, after)
		}
	})
}
