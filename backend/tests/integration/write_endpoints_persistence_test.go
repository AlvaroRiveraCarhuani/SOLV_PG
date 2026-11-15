package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// Estos tests verifican el EFECTO, no el código de respuesta. Un handler que
// devuelve 200 sin escribir en la base es indistinguible de uno que escribe si
// solo se mira el status; por eso cada caso lee la fila después.
func setupWritePersistenceServer(t *testing.T) (*httptest.Server, *database.Database) {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	if dsn == "" {
		t.Skip("sin TEST_DB_DSN: prueba de persistencia omitida")
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("base de tests no disponible: %v", err)
	}

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	periodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())

	handlers := httpdelivery.Handlers{
		AdminAcademicHandler: httpdelivery.NewAdminAcademicHandler(
			services.NewAcademicPeriodService(periodRepo),
			services.NewMaintenanceService(tenantRepo),
			services.NewAdminGovernanceService(subjectRepo, govRepo),
		),
		SubjectHandler: httpdelivery.NewSubjectHandler(services.NewSubjectService(subjectRepo)),
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

// seedTenant crea una institución descartable y la borra al terminar, en el
// orden que exigen las claves foráneas.
func seedTenant(t *testing.T, db *database.Database, name string) string {
	t.Helper()

	tenantID := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, name, "persist-"+tenantID[:8],
	); err != nil {
		t.Fatalf("no se pudo sembrar la institución: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})
	return tenantID
}

func seedUser(t *testing.T, db *database.Database, tenantID, role, email string) string {
	t.Helper()

	userID := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO users (id, tenant_id, email, first_name, last_name, role, status, origin)
		 VALUES ($1, $2, $3, 'Test', 'Persistencia', $4, 'active', 'manual')`,
		userID, tenantID, email, role,
	); err != nil {
		t.Fatalf("no se pudo sembrar el usuario: %v", err)
	}
	return userID
}

func seedSubject(t *testing.T, db *database.Database, tenantID, code string) string {
	t.Helper()

	subjectID := uuid.NewString()
	if _, err := db.GetDB().Exec(
		`INSERT INTO subjects (id, tenant_id, name, code) VALUES ($1, $2, $3, $4)`,
		subjectID, tenantID, "Materia "+code, code,
	); err != nil {
		t.Fatalf("no se pudo sembrar la materia: %v", err)
	}
	return subjectID
}

func adminHeaders(tenantID, adminID string) map[string]string {
	return map[string]string{
		"X-Test-Tenant-Id": tenantID,
		"X-Test-User-Id":   adminID,
		"X-Test-User-Role": "admin",
		"Content-Type":     "application/json",
	}
}

func doJSON(t *testing.T, client *http.Client, method, url string, headers map[string]string, body any) *http.Response {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("payload inválido: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		t.Fatalf("no se pudo crear la petición: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("la petición falló: %v", err)
	}
	return res
}

func TestWritePersistence_SubjectUpdateWritesRow(t *testing.T) {
	server, db := setupWritePersistenceServer(t)
	tenantID := seedTenant(t, db, "Persistencia Materias")
	adminID := seedUser(t, db, tenantID, "admin", "admin-"+uuid.NewString()[:8]+"@test.local")
	subjectID := seedSubject(t, db, tenantID, "PERS-101")

	res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/subjects/"+subjectID,
		adminHeaders(tenantID, adminID), map[string]string{"name": "Nombre Nuevo", "code": "PERS-202"})
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
	}

	var name, code string
	if err := db.GetDB().QueryRow(`SELECT name, code FROM subjects WHERE id = $1`, subjectID).Scan(&name, &code); err != nil {
		t.Fatalf("no se pudo leer la materia: %v", err)
	}
	if name != "Nombre Nuevo" || code != "PERS-202" {
		t.Errorf("la fila no cambió: name=%q code=%q", name, code)
	}
}

func TestWritePersistence_StudentStatusWritesRow(t *testing.T) {
	server, db := setupWritePersistenceServer(t)
	tenantID := seedTenant(t, db, "Persistencia Estudiantes")
	adminID := seedUser(t, db, tenantID, "admin", "admin-"+uuid.NewString()[:8]+"@test.local")
	studentID := seedUser(t, db, tenantID, "student", "alumno-"+uuid.NewString()[:8]+"@test.local")

	res := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/v1/admin/students/"+studentID+"/status",
		adminHeaders(tenantID, adminID), map[string]string{"status": "suspended", "reason": "Prueba de persistencia"})
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
	}

	var status string
	var reason *string
	if err := db.GetDB().QueryRow(`SELECT status, suspension_reason FROM users WHERE id = $1`, studentID).Scan(&status, &reason); err != nil {
		t.Fatalf("no se pudo leer el estudiante: %v", err)
	}
	if status != "suspended" {
		t.Errorf("el estado no se guardó: %q", status)
	}
	if reason == nil || *reason != "Prueba de persistencia" {
		t.Errorf("el motivo no se guardó: %v", reason)
	}
}

// La reconciliación de nómina (ADR-079) se prueba contra la base: suma, quita lo
// que ya no está y deja intacta la matrícula manual.
func TestWritePersistence_RosterReconcileAddsAndRemoves(t *testing.T) {
	server, db := setupWritePersistenceServer(t)
	tenantID := seedTenant(t, db, "Persistencia Nomina")
	adminID := seedUser(t, db, tenantID, "admin", "admin-"+uuid.NewString()[:8]+"@test.local")
	subjectID := seedSubject(t, db, tenantID, "NOM-101")

	suffix := uuid.NewString()[:8]
	emailA := fmt.Sprintf("ana-%s@test.local", suffix)
	emailB := fmt.Sprintf("bruno-%s@test.local", suffix)
	emailManual := fmt.Sprintf("manual-%s@test.local", suffix)

	rosterURL := server.URL + "/api/v1/admin/subjects/" + subjectID + "/roster"

	// 1. Primera nómina: dos alumnos.
	first := doJSON(t, server.Client(), http.MethodPost, rosterURL, adminHeaders(tenantID, adminID), map[string]any{
		"emails":        []string{emailA, emailB},
		"source_system": "planilla",
	})
	_ = first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("alta de nómina: se esperaba 201, se obtuvo %d", first.StatusCode)
	}

	// Matrícula manual de un tercero, que la reconciliación no debe tocar.
	manualID := seedUser(t, db, tenantID, "student", emailManual)
	if _, err := db.GetDB().Exec(
		`INSERT INTO enrollments (tenant_id, student_id, subject_id, source_system)
		 VALUES ($1, $2, $3, 'manual')`, tenantID, manualID, subjectID,
	); err != nil {
		t.Fatalf("no se pudo sembrar la matrícula manual: %v", err)
	}

	// 2. Segunda nómina: solo Ana, con reconciliación.
	second := doJSON(t, server.Client(), http.MethodPost, rosterURL, adminHeaders(tenantID, adminID), map[string]any{
		"emails":        []string{emailA},
		"source_system": "planilla",
		"reconcile":     true,
	})
	defer func() { _ = second.Body.Close() }()
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("segunda nómina: se esperaba 201, se obtuvo %d", second.StatusCode)
	}

	rows, err := db.GetDB().Query(`
		SELECT u.email, e.source_system
		FROM enrollments e JOIN users u ON u.id = e.student_id
		WHERE e.tenant_id = $1 AND e.subject_id = $2
		ORDER BY u.email`, tenantID, subjectID)
	if err != nil {
		t.Fatalf("no se pudieron leer las matrículas: %v", err)
	}
	defer func() { _ = rows.Close() }()

	enrolled := map[string]string{}
	for rows.Next() {
		var email, source string
		if err := rows.Scan(&email, &source); err != nil {
			t.Fatalf("lectura inválida: %v", err)
		}
		enrolled[email] = source
	}

	if _, ok := enrolled[emailA]; !ok {
		t.Errorf("Ana tenía que seguir matriculada: %v", enrolled)
	}
	if _, ok := enrolled[emailB]; ok {
		t.Errorf("Bruno tenía que salir con la reconciliación: %v", enrolled)
	}
	if source, ok := enrolled[emailManual]; !ok || source != "manual" {
		t.Errorf("la matrícula manual debía sobrevivir: %v", enrolled)
	}
}
