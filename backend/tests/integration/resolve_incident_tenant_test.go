package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupResolveIncidentServer(t *testing.T) (*httptest.Server, *database.Database) {
	t.Helper()

	db := openTestDB(t)

	handlers := httpdelivery.Handlers{
		AdminHandler: httpdelivery.NewAdminHandler(
			nil,
			postgres.NewPostgresTenantRepository(db.GetDB()),
			postgres.NewPostgresPracticeEnvironmentRepository(db.GetDB()),
			nil,
			nil,
		),
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, withTestIdentityFromHeaders(r))
			})
		},
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, db
}

func seedEnvironment(t *testing.T, db *database.Database, tenantID, subjectID, studentID, status string, strikes int) string {
	t.Helper()

	id := uuid.NewString()
	if _, err := db.GetDB().Exec(`
		INSERT INTO environments (id, student_id, subject_id, tenant_id, type, status, oom_strike_count, last_oom_killed_at)
		VALUES ($1, $2, $3, $4, 'IDE_PERSISTENTE', $5, $6, NOW())`,
		id, studentID, subjectID, tenantID, status, strikes,
	); err != nil {
		t.Fatalf("no se pudo sembrar el entorno: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM environments WHERE id = $1`, id)
	})
	return id
}

func identityHeaders(tenantID, userID, role string) map[string]string {
	return map[string]string{
		"X-Test-Tenant-Id": tenantID,
		"X-Test-User-Id":   userID,
		"X-Test-User-Role": role,
		"Content-Type":     "application/json",
	}
}

type environmentSnapshot struct {
	Status  string
	Strikes int
	LastOOM *string
}

func readEnvironment(t *testing.T, db *database.Database, environmentID string) environmentSnapshot {
	t.Helper()

	var snapshot environmentSnapshot
	if err := db.GetDB().QueryRow(
		`SELECT status, oom_strike_count, last_oom_killed_at::text FROM environments WHERE id = $1`,
		environmentID,
	).Scan(&snapshot.Status, &snapshot.Strikes, &snapshot.LastOOM); err != nil {
		t.Fatalf("no se pudo leer el entorno: %v", err)
	}
	return snapshot
}

// Resolver una incidencia tiene que cambiar el entorno y no puede alcanzar el
// de otra institución. La lectura del entorno está acotada por el tenant del
// contexto, así que un id ajeno responde 404 (no confirma que exista) y la fila
// queda intacta.
func TestResolveIncident_ScopedToTenantAndPersisted(t *testing.T) {
	server, db := setupResolveIncidentServer(t)

	tenantA := seedTenant(t, db, "Incidencias Institucion A")
	tenantB := seedTenant(t, db, "Incidencias Institucion B")
	adminA := seedUser(t, db, tenantA, "admin", "admin-"+uuid.NewString()[:8]+"@test.local")
	studentA := seedUser(t, db, tenantA, "student", "alumno-"+uuid.NewString()[:8]+"@test.local")
	studentB := seedUser(t, db, tenantB, "student", "alumno-"+uuid.NewString()[:8]+"@test.local")
	subjectA := seedSubject(t, db, tenantA, "INC-A")
	subjectB := seedSubject(t, db, tenantB, "INC-B")

	environmentA := seedEnvironment(t, db, tenantA, subjectA, studentA, "failed", 2)
	environmentB := seedEnvironment(t, db, tenantB, subjectB, studentB, "oom_killed", 3)

	resolveURL := func(id string) string {
		return server.URL + "/api/v1/admin/environments/" + id + "/resolve"
	}

	t.Run("resuelve el entorno propio y lo deja hibernado", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, resolveURL(environmentA),
			adminHeaders(tenantA, adminA), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta inválida: %v", err)
		}
		if body["environment_status"] != "hibernated" {
			t.Errorf("la respuesta no informa el estado real: %v", body["environment_status"])
		}

		snapshot := readEnvironment(t, db, environmentA)
		if snapshot.Status != "hibernated" {
			t.Errorf("el entorno no quedó hibernado: %q", snapshot.Status)
		}
		if snapshot.Strikes != 0 {
			t.Errorf("los strikes no se normalizaron: %d", snapshot.Strikes)
		}
		if snapshot.LastOOM != nil {
			t.Errorf("la marca de OOM no se limpió: %v", *snapshot.LastOOM)
		}
	})

	t.Run("no alcanza el entorno de otra institución", func(t *testing.T) {
		antes := readEnvironment(t, db, environmentB)

		// El administrador de A se presenta como A y apunta al entorno de B.
		res := doJSON(t, server.Client(), http.MethodPost, resolveURL(environmentB),
			adminHeaders(tenantA, adminA), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode == http.StatusOK {
			t.Fatalf("la institución A resolvió un entorno de la institución B")
		}
		if res.StatusCode != http.StatusNotFound && res.StatusCode != http.StatusForbidden {
			t.Fatalf("se esperaba un rechazo (404/403), se obtuvo %d", res.StatusCode)
		}

		despues := readEnvironment(t, db, environmentB)
		if despues.Status != antes.Status || despues.Strikes != antes.Strikes {
			t.Errorf("la fila ajena cambió: antes=%+v despues=%+v", antes, despues)
		}
		if despues.LastOOM == nil {
			t.Error("se limpió la marca de OOM de un entorno ajeno")
		}
	})

	t.Run("un alumno no puede resolver incidencias", func(t *testing.T) {
		res := doJSON(t, server.Client(), http.MethodPost, resolveURL(environmentA),
			identityHeaders(tenantA, studentA, "student"), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("un alumno debe recibir 403, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("informa el estado real sin inventar hibernación", func(t *testing.T) {
		running := seedEnvironment(t, db, tenantA, subjectA, studentA, "running", 1)

		res := doJSON(t, server.Client(), http.MethodPost, resolveURL(running),
			adminHeaders(tenantA, adminA), nil)
		defer func() { _ = res.Body.Close() }()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta inválida: %v", err)
		}
		if body["environment_status"] != "running" {
			t.Errorf("un entorno en ejecución no debe reportarse como hibernado: %v", body["environment_status"])
		}

		snapshot := readEnvironment(t, db, running)
		if snapshot.Status != "running" {
			t.Errorf("el estado cambió sin motivo: %q", snapshot.Status)
		}
		if snapshot.Strikes != 0 {
			t.Errorf("los strikes no se normalizaron: %d", snapshot.Strikes)
		}
	})
}
