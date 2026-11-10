package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// newWorkspaceProjectTestDB crea una base descartable para esta prueba. No usa
// la base de desarrollo ni la del despliegue y no comparte filas con otras
// pruebas de integración.
func newWorkspaceProjectTestDB(t *testing.T) *database.Database {
	t.Helper()
	baseDSN := getTestDSN()
	if strings.TrimSpace(baseDSN) == "" {
		skipOrFailInfra(t, "TEST_DB_DSN o TEST_DATABASE_URL no configurado")
	}
	admin, err := sqlx.Connect("postgres", baseDSN)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL de tests no disponible: %v", err)
	}
	defer func() { _ = admin.Close() }()

	name := "solv_test_f1_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		t.Fatalf("no se pudo crear la base descartable %s: %v", name, err)
	}

	parsed, err := url.Parse(baseDSN)
	if err != nil {
		t.Fatalf("DSN de tests inválido: %v", err)
	}
	parsed.Path = "/" + name
	testDSN := parsed.String()

	db, err := database.NewPostgresDB(testDSN)
	if err != nil {
		t.Fatalf("no se pudo abrir la base descartable: %v", err)
	}
	cleanup := func() {
		_ = db.GetDB().Close()
		adminCleanup, err := sqlx.Connect("postgres", baseDSN)
		if err != nil {
			t.Logf("no se pudo reconectar para borrar %s: %v", name, err)
			return
		}
		defer func() { _ = adminCleanup.Close() }()
		if _, err := adminCleanup.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`); err != nil {
			t.Logf("no se pudo borrar la base descartable %s: %v", name, err)
		}
	}
	t.Cleanup(cleanup)

	if err := database.RunMigrations(db.GetDB().DB, "../../migrations"); err != nil {
		t.Fatalf("no se pudieron aplicar las migraciones: %v", err)
	}
	return db
}

func hashTestProject(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type workspaceTestSeed struct {
	tenantID  string
	teacherID string
	studentA  string
	studentB  string
	subjectID string
	exerciseA string
	exerciseB string
}

func seedWorkspaceTestBase(t *testing.T, db *sqlx.DB) workspaceTestSeed {
	t.Helper()
	ctx := context.Background()
	seed := workspaceTestSeed{
		tenantID:  uuid.NewString(),
		teacherID: uuid.NewString(),
		studentA:  uuid.NewString(),
		studentB:  uuid.NewString(),
		subjectID: uuid.NewString(),
		exerciseA: uuid.NewString(),
		exerciseB: uuid.NewString(),
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'F1 Tenant', $2)
	`, seed.tenantID, "f1-"+seed.tenantID[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	for _, user := range []struct {
		id, email, role string
	}{
		{seed.teacherID, "teacher-" + seed.teacherID[:8] + "@example.test", "teacher"},
		{seed.studentA, "student-a-" + seed.studentA[:8] + "@example.test", "student"},
		{seed.studentB, "student-b-" + seed.studentB[:8] + "@example.test", "student"},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'F1', 'User', $2, $3, $4)
		`, user.id, user.email, user.role, seed.tenantID); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'F1 Subject', $3, $4)
	`, seed.subjectID, seed.tenantID, "F1-"+seed.subjectID[:8], seed.teacherID); err != nil {
		t.Fatalf("seed subject: %v", err)
	}
	templateID := uuid.NewString()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, status)
		VALUES ($1, $2, 'F1 IDE Template', 'solv/ide:test', 512, 'IDE_PERSISTENTE', 'approved')
	`, templateID, seed.tenantID); err != nil {
		t.Fatalf("seed IDE template: %v", err)
	}
	for _, exercise := range []struct{ id, title string }{
		{seed.exerciseA, "F1 Exercise A"},
		{seed.exerciseB, "F1 Exercise B"},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO exercises (id, tenant_id, subject_id, title, type, status, environment_type, language, boilerplate, template_id)
			VALUES ($1, $2, $3, $4, 'project', 'draft', 'IDE_PERSISTENTE', 'python', '', $5)
		`, exercise.id, seed.tenantID, seed.subjectID, exercise.title, templateID); err != nil {
			t.Fatalf("seed exercise: %v", err)
		}
	}
	return seed
}

func insertWorkspaceProject(t *testing.T, db *sqlx.DB, tenantID, exerciseID, createdBy string) string {
	t.Helper()
	projectID := uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO workspace_projects (id, tenant_id, exercise_id, created_by)
		VALUES ($1, $2, $3, $4)
	`, projectID, tenantID, exerciseID, createdBy); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	return projectID
}

func insertWorkspaceVersion(t *testing.T, db *sqlx.DB, tenantID, projectID, exerciseID string, number int, text string) string {
	t.Helper()
	versionID := uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO workspace_project_versions
			(id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, '[]'::jsonb, $7, 1, $8)
	`, versionID, tenantID, projectID, exerciseID, number, text, hashTestProject(text), len(text)); err != nil {
		t.Fatalf("insert version: %v", err)
	}
	return versionID
}

func expectDatabaseReject(t *testing.T, db *sqlx.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err == nil {
		t.Fatalf("PostgreSQL aceptó una relación inválida: %s", query)
	}
}

func TestWorkspaceProjectPublishRepositoryLiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)
	repo := postgres.NewPostgresWorkspaceProjectRepository(sqlDB)
	ctx := context.Background()

	project, err := repo.CreateProject(ctx, &domain.WorkspaceProject{
		TenantID:   seed.tenantID,
		ExerciseID: seed.exerciseA,
		CreatedBy:  &seed.teacherID,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	draft, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:     seed.tenantID,
		ProjectID:    project.ID,
		TextSnapshot: "# === FILE: main.py ===\nprint('v1')\n",
		UpdatedBy:    &seed.teacherID,
	}, 0)
	if err != nil {
		t.Fatalf("SaveDraft inicial: %v", err)
	}
	if draft.Revision != 1 {
		t.Fatalf("revisión inicial = %d", draft.Revision)
	}
	if _, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:     seed.tenantID,
		ProjectID:    project.ID,
		TextSnapshot: "# === FILE: main.py ===\nprint('reintento')\n",
	}, 0); !errors.Is(err, domain.ErrWorkspaceDraftRevisionConflict) {
		t.Fatalf("segundo SaveDraft con revisión 0 debía conflictuar: %v", err)
	}

	v1, err := repo.Publish(ctx, seed.tenantID, project.ID, seed.teacherID)
	if err != nil {
		t.Fatalf("Publish v1: %v", err)
	}
	if v1.VersionNumber != 1 {
		t.Fatalf("versión publicada = %d", v1.VersionNumber)
	}
	var currentVersion, mirrorSource, mirrorText string
	if err := sqlDB.QueryRow(`SELECT current_published_version_id FROM workspace_projects WHERE id = $1`, project.ID).Scan(&currentVersion); err != nil {
		t.Fatalf("leer puntero: %v", err)
	}
	if currentVersion != v1.ID {
		t.Fatalf("puntero publicado = %q, quería %q", currentVersion, v1.ID)
	}
	if err := sqlDB.QueryRow(`SELECT boilerplate_source_version_id, boilerplate FROM exercises WHERE id = $1`, seed.exerciseA).Scan(&mirrorSource, &mirrorText); err != nil {
		t.Fatalf("leer espejo: %v", err)
	}
	if mirrorSource != v1.ID || !strings.Contains(mirrorText, "v1") {
		t.Fatalf("espejo inesperado: source=%q text=%q", mirrorSource, mirrorText)
	}

	if _, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:     seed.tenantID,
		ProjectID:    project.ID,
		TextSnapshot: "# === FILE: main.py ===\nprint('v2')\n",
		UpdatedBy:    &seed.teacherID,
	}, 1); err != nil {
		t.Fatalf("SaveDraft v2: %v", err)
	}
	v2, err := repo.Publish(ctx, seed.tenantID, project.ID, seed.teacherID)
	if err != nil {
		t.Fatalf("Publish v2: %v", err)
	}
	if v2.VersionNumber != 2 {
		t.Fatalf("segunda versión = %d", v2.VersionNumber)
	}
	var v1Text string
	if err := sqlDB.QueryRow(`SELECT text_snapshot FROM workspace_project_versions WHERE id = $1`, v1.ID).Scan(&v1Text); err != nil {
		t.Fatalf("leer v1: %v", err)
	}
	if !strings.Contains(v1Text, "v1") || strings.Contains(v1Text, "v2") {
		t.Fatalf("la versión 1 cambió tras publicar v2: %q", v1Text)
	}
	if _, err := sqlDB.Exec(`UPDATE workspace_project_versions SET text_snapshot = 'alterado' WHERE id = $1`, v1.ID); err == nil {
		t.Fatal("PostgreSQL permitió mutar una versión publicada")
	}
}

func TestWorkspaceProjectHTTPDraftAndPublishFlagLiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)
	repo := postgres.NewPostgresWorkspaceProjectRepository(sqlDB)
	handler := httpdelivery.NewWorkspaceProjectHandler(services.NewWorkspaceProjectService(repo, false))
	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &httpdelivery.Handlers{
		WorkspaceProjectHandler: handler,
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, withTestIdentityFromHeaders(r))
			})
		},
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	request := func(method, path, role, body string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("X-Test-Tenant-Id", seed.tenantID)
		req.Header.Set("X-Test-User-Id", seed.teacherID)
		req.Header.Set("X-Test-User-Role", role)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do %s: %v", path, err)
		}
		return resp
	}

	resp := request(http.MethodPut, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/draft", "student",
		`{"text_snapshot":"# === FILE: main.py ===\nprint(1)\n"}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("student debía recibir 403, recibió %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	body := `{"text_snapshot":"# === FILE: main.py ===\nprint(1)\n","expected_revision":0}`
	resp = request(http.MethodPut, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/draft", "teacher", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("guardar borrador devolvió %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = request(http.MethodPost, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/publish", "teacher", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("publish con flag apagado devolvió %d, quería 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = request(http.MethodGet, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project", "teacher", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET proyecto devolvió %d", resp.StatusCode)
	}
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(payload), "print(1)") {
		t.Fatalf("GET no devolvió el borrador: %s", payload)
	}
}

func TestWorkspaceProjectIntegrityLiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)

	projectA := insertWorkspaceProject(t, sqlDB, seed.tenantID, seed.exerciseA, seed.teacherID)
	projectB := insertWorkspaceProject(t, sqlDB, seed.tenantID, seed.exerciseB, seed.teacherID)
	versionA1 := insertWorkspaceVersion(t, sqlDB, seed.tenantID, projectA, seed.exerciseA, 1, "# === FILE: main.py ===\nprint('a1')\n")
	versionB1 := insertWorkspaceVersion(t, sqlDB, seed.tenantID, projectB, seed.exerciseB, 1, "# === FILE: main.py ===\nprint('b1')\n")

	// Una versión no puede declarar el ejercicio vecino del proyecto.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO workspace_project_versions
			(id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256)
		VALUES ($1, $2, $3, $4, 2, '# === FILE: main.py ===', '[]'::jsonb, $5)
	`, uuid.NewString(), seed.tenantID, projectA, seed.exerciseB, hashTestProject("x"))

	// El puntero publicado no puede cruzar proyectos.
	expectDatabaseReject(t, sqlDB, `
		UPDATE workspace_projects SET current_published_version_id = $1 WHERE id = $2
	`, versionB1, projectA)

	// El borrador no puede usar la versión de otro proyecto.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO workspace_project_drafts (tenant_id, project_id, base_version_id, text_snapshot)
		VALUES ($1, $2, $3, '# === FILE: main.py ===')
	`, seed.tenantID, projectA, versionB1)

	// Una entrega con versión no puede apuntar a una versión de otro ejercicio.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO submissions (tenant_id, exercise_id, student_id, code, verdict, base_status, workspace_project_version_id)
		VALUES ($1, $2, $3, 'x', NULL, 'version', $4)
	`, seed.tenantID, seed.exerciseB, seed.studentA, versionA1)

	// La clasificación y la versión deben corresponderse.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO submissions (tenant_id, exercise_id, student_id, code, verdict, base_status, workspace_project_version_id)
		VALUES ($1, $2, $3, 'x', NULL, 'version', NULL)
	`, seed.tenantID, seed.exerciseA, seed.studentA)
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO submissions (tenant_id, exercise_id, student_id, code, verdict, base_status, workspace_project_version_id)
		VALUES ($1, $2, $3, 'x', NULL, 'no_base', $4)
	`, seed.tenantID, seed.exerciseA, seed.studentA, versionA1)

	// El par legacy es bidireccional.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'legacy', 'pending')
	`, uuid.NewString(), seed.studentA, seed.subjectID, seed.tenantID, seed.exerciseA)
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'none', 'legacy')
	`, uuid.NewString(), seed.studentA, seed.subjectID, seed.tenantID, seed.exerciseA)

	// project exige versión y las modalidades sin proyecto no la admiten.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'project', 'pending')
	`, uuid.NewString(), seed.studentA, seed.subjectID, seed.tenantID, seed.exerciseA)
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status, workspace_project_version_id)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'none', 'not_required', $6)
	`, uuid.NewString(), seed.studentA, seed.subjectID, seed.tenantID, seed.exerciseA, versionA1)
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status, workspace_project_version_id)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'legacy', 'legacy', $6)
	`, uuid.NewString(), seed.studentA, seed.subjectID, seed.tenantID, seed.exerciseA, versionA1)

	// La versión del entorno no puede cruzar ejercicio.
	expectDatabaseReject(t, sqlDB, `
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status, workspace_project_version_id)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'project', 'pending', $6)
	`, uuid.NewString(), seed.studentA, seed.subjectID, seed.tenantID, seed.exerciseA, versionB1)

	// Los estados transitorios legítimos de project deben ser aceptados.
	for _, status := range []string{"pending", "volume_created", "setup_done", "failed", "quarantined", "missing"} {
		if _, err := sqlDB.Exec(`
			INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status, workspace_project_version_id)
			VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'project', $6, $7)
		`, uuid.NewString(), uuid.NewString(), seed.subjectID, seed.tenantID, seed.exerciseA, status, versionA1); err != nil {
			t.Fatalf("project/%s debía admitirse: %v", status, err)
		}
	}

	// Los estados transitorios de none también se admiten.
	for _, status := range []string{"pending", "volume_created", "setup_done", "failed", "quarantined", "missing", "not_required"} {
		if _, err := sqlDB.Exec(`
			INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status)
			VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'none', $6)
		`, uuid.NewString(), uuid.NewString(), seed.subjectID, seed.tenantID, seed.exerciseA, status); err != nil {
			t.Fatalf("none/%s debía admitirse: %v", status, err)
		}
	}

	// legacy con su estado exacto se admite.
	if _, err := sqlDB.Exec(`
		INSERT INTO environments (id, student_id, subject_id, tenant_id, exercise_id, type, status, workspace_seed_mode, workspace_init_status)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'pending', 'legacy', 'legacy')
	`, uuid.NewString(), uuid.NewString(), seed.subjectID, seed.tenantID, seed.exerciseA); err != nil {
		t.Fatalf("legacy/legacy debía admitirse: %v", err)
	}
}

func insertWorkspaceEnvironment(t *testing.T, db *sqlx.DB, tenantID, subjectID, exerciseID, studentID, versionID string) string {
	t.Helper()
	environmentID := uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO environments
			(id, student_id, subject_id, tenant_id, exercise_id, type, status, access_url,
			 workspace_seed_mode, workspace_init_status, workspace_project_version_id)
		VALUES ($1, $2, $3, $4, $5, 'IDE_PERSISTENTE', 'running', '', 'project', 'setup_done', $6)
	`, environmentID, studentID, subjectID, tenantID, exerciseID, versionID); err != nil {
		t.Fatalf("insert environment: %v", err)
	}
	return environmentID
}

func insertWorkspaceSubmission(t *testing.T, db *sqlx.DB, tenantID, subjectID, exerciseID, studentID, environmentID, versionID, code string) string {
	t.Helper()
	submissionID := uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO submissions
			(id, tenant_id, exercise_id, student_id, environment_id, code, verdict, review_status,
			 ast_result, generated_cases, complexity_analysis, base_status, workspace_project_version_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, 'pending', '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, 'version', $7)
	`, submissionID, tenantID, exerciseID, studentID, environmentID, code, versionID); err != nil {
		t.Fatalf("insert submission: %v", err)
	}
	return submissionID
}

// T47: publicar una versión nueva mueve el espejo del ejercicio, pero no la
// base de comparación de una entrega histórica. SpeedGrader, antiplagio y
// timeline deben seguir leyendo la versión exacta de la entrega.
func TestWorkspaceProjectT47HistoricalBaseLiveDB(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()
	seed := seedWorkspaceTestBase(t, sqlDB)

	projectID := insertWorkspaceProject(t, sqlDB, seed.tenantID, seed.exerciseA, seed.teacherID)
	v1Text := "# === FILE: main.py ===\nprint('base-v1')\n"
	v2Text := "# === FILE: main.py ===\nprint('base-v2')\n"
	v1 := insertWorkspaceVersion(t, sqlDB, seed.tenantID, projectID, seed.exerciseA, 1, v1Text)
	v2 := insertWorkspaceVersion(t, sqlDB, seed.tenantID, projectID, seed.exerciseA, 2, v2Text)

	if _, err := sqlDB.Exec(`UPDATE workspace_projects SET current_published_version_id = $1 WHERE id = $2`, v1, projectID); err != nil {
		t.Fatalf("set current v1: %v", err)
	}
	if _, err := sqlDB.Exec(`
		UPDATE exercises
		SET boilerplate = $1, boilerplate_sha256 = $2, boilerplate_source_version_id = $3
		WHERE id = $4
	`, v1Text, hashTestProject(v1Text), v1, seed.exerciseA); err != nil {
		t.Fatalf("mirror v1: %v", err)
	}

	environmentA := insertWorkspaceEnvironment(t, sqlDB, seed.tenantID, seed.subjectID, seed.exerciseA, seed.studentA, v1)
	environmentB := insertWorkspaceEnvironment(t, sqlDB, seed.tenantID, seed.subjectID, seed.exerciseA, seed.studentB, v1)
	submissionA := insertWorkspaceSubmission(t, sqlDB, seed.tenantID, seed.subjectID, seed.exerciseA, seed.studentA, environmentA, v1,
		"# === FILE: main.py ===\nprint('base-v1')\nprint('student-a')\n")
	submissionB := insertWorkspaceSubmission(t, sqlDB, seed.tenantID, seed.subjectID, seed.exerciseA, seed.studentB, environmentB, v1,
		"# === FILE: main.py ===\nprint('base-v1')\nprint('student-b')\n")

	// Publicar v2: el espejo cambia, la entrega histórica no.
	if _, err := sqlDB.Exec(`
		UPDATE exercises
		SET boilerplate = $1, boilerplate_sha256 = $2, boilerplate_source_version_id = $3
		WHERE id = $4
	`, v2Text, hashTestProject(v2Text), v2, seed.exerciseA); err != nil {
		t.Fatalf("mirror v2: %v", err)
	}
	if _, err := sqlDB.Exec(`UPDATE workspace_projects SET current_published_version_id = $1 WHERE id = $2`, v2, projectID); err != nil {
		t.Fatalf("set current v2: %v", err)
	}

	teacherRepo := postgres.NewPostgresTeacherRepository(sqlDB)
	ctx := context.Background()

	review, err := teacherRepo.GetTeacherSubmissionReview(ctx, seed.tenantID, "", submissionA)
	if err != nil {
		t.Fatalf("SpeedGrader: %v", err)
	}
	if review.BaseStatus != domain.BaseStatusVersion {
		t.Fatalf("SpeedGrader base_status = %q", review.BaseStatus)
	}
	if review.WorkspaceProjectVersionID == nil || *review.WorkspaceProjectVersionID != v1 {
		t.Fatalf("SpeedGrader versión = %v, quería %s", review.WorkspaceProjectVersionID, v1)
	}
	if !review.BaseFilesAvailable || len(review.BaseFiles) != 1 || !strings.Contains(review.BaseFiles[0].Content, "base-v1") {
		t.Fatalf("SpeedGrader base_files no conserva v1: %+v", review.BaseFiles)
	}
	if strings.Contains(review.BaseFiles[0].Content, "base-v2") {
		t.Fatalf("SpeedGrader mezcló la versión publicada nueva")
	}

	plagiarism, err := teacherRepo.GetExerciseSubmissionsForPlagiarism(ctx, seed.tenantID, "", seed.subjectID, seed.exerciseA)
	if err != nil {
		t.Fatalf("antiplagio: %v", err)
	}
	if len(plagiarism) != 2 {
		t.Fatalf("antiplagio devolvió %d entregas", len(plagiarism))
	}
	for _, row := range plagiarism {
		if row.BaseStatus != domain.BaseStatusVersion || !strings.Contains(row.Boilerplate, "base-v1") {
			t.Fatalf("antiplagio no conserva v1: status=%q base=%q", row.BaseStatus, row.Boilerplate)
		}
	}

	pair, err := teacherRepo.GetPlagiarismPairSubmissions(ctx, seed.tenantID, "", submissionA, submissionB)
	if err != nil {
		t.Fatalf("par antiplagio: %v", err)
	}
	if len(pair) != 2 {
		t.Fatalf("par antiplagio devolvió %d entregas", len(pair))
	}
	for _, row := range pair {
		if row.BaseStatus != domain.BaseStatusVersion || !strings.Contains(row.Boilerplate, "base-v1") {
			t.Fatalf("par antiplagio no conserva v1: status=%q base=%q", row.BaseStatus, row.Boilerplate)
		}
	}

	teacherService := services.NewTeacherService(teacherRepo, postgres.NewPostgresSubmissionRepository(sqlDB))
	timeline, err := teacherService.GetSubmissionTimeline(ctx, seed.tenantID, "", submissionA)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(timeline.Keyframes) == 0 || !strings.Contains(timeline.Keyframes[0].Content, "base-v1") {
		t.Fatalf("timeline no arranca en v1: %+v", timeline.Keyframes)
	}
	if strings.Contains(timeline.Keyframes[0].Content, "base-v2") {
		t.Fatalf("timeline mezcló la versión publicada nueva")
	}
}
