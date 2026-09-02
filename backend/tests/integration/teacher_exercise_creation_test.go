package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupSlice13TestServer(t *testing.T) (*httptest.Server, *database.Database) {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "solv_user"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "solv_password"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "solv_db"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.RunInitialMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	validate := validator.New()
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	evalService := services.NewEvaluationService(exerciseRepo, nil, nil, nil)
	evalHandler := httpdelivery.NewEvaluationHandler(evalService, validate)

	handlers := &httpdelivery.Handlers{
		EvaluationHandler: evalHandler,
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tenantID := r.Header.Get("X-Tenant-Id")
				if tenantID == "" {
					tenantID = domain.DefaultTenantID
				}
				ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		},
		AuditMiddleware: func(next http.Handler) http.Handler {
			return next
		},
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, handlers)
	ts := httptest.NewServer(mux)
	return ts, db
}

// TestSlice13_Commit1_CreateAndVerifyDirectDB verifica la creación de ejercicio y su estado real en BD
func TestSlice13_Commit1_CreateAndVerifyDirectDB(t *testing.T) {
	ts, db := setupSlice13TestServer(t)
	defer ts.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	exerciseID := uuid.NewString()

	payload := domain.Exercise{
		ID:            exerciseID,
		Title:         "Algoritmo Dijkstra de Caminos Mínimos",
		Description:   "Implemente Dijkstra utilizando cola de prioridad",
		Type:          domain.ExerciseTypeAlgorithm,
		Boilerplate:   "def dijkstra(graph, start):\n    pass\n",
		Status:        "draft",
		Language:      "python",
		TimeLimitMS:   1500,
		MemoryLimitMB: 256,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TimeLimitMS:   1500,
				MemoryLimitMB: 256,
				TestCases: []domain.TestCase{
					{Input: "g1, start", ExpectedOutput: "[0, 2, 5]", IsHidden: false},
					{Input: "g2, start", ExpectedOutput: "[0, 1, 4]", IsHidden: false},
					{Input: "g_priv1, start", ExpectedOutput: "[0, 10, 25]", IsHidden: true},
					{Input: "g_priv2, start", ExpectedOutput: "[0, 3, 7]", IsHidden: true},
				},
				ASTRules: domain.ASTRules{
					ForbiddenImports:   []string{"networkx", "os"},
					ForbiddenFunctions: []string{"eval", "exec"},
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("X-Tenant-Id", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/exercises failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", resp.StatusCode, buf.String())
	}

	// AFIRMACIÓN DIRECTA EN POSTGRESQL CON SQLX
	var dbEx domain.Exercise
	query := `SELECT id, title, description, type, boilerplate, status, language, time_limit_ms, memory_limit_mb, config, tenant_id FROM exercises WHERE id = $1`
	err = db.GetDB().Get(&dbEx, query, exerciseID)
	if err != nil {
		t.Fatalf("Direct DB query failed: %v", err)
	}

	if dbEx.Title != "Algoritmo Dijkstra de Caminos Mínimos" {
		t.Errorf("DB title mismatch: expected '%s', got '%s'", payload.Title, dbEx.Title)
	}
	if dbEx.Boilerplate != payload.Boilerplate {
		t.Errorf("DB boilerplate mismatch: expected '%s', got '%s'", payload.Boilerplate, dbEx.Boilerplate)
	}
	if dbEx.Status != "draft" {
		t.Errorf("DB status mismatch: expected 'draft', got '%s'", dbEx.Status)
	}
	if dbEx.Language != "python" {
		t.Errorf("DB language mismatch: expected 'python', got '%s'", dbEx.Language)
	}
	if dbEx.TimeLimitMS != 1500 || dbEx.MemoryLimitMB != 256 {
		t.Errorf("DB limits mismatch: got time=%d, memory=%d", dbEx.TimeLimitMS, dbEx.MemoryLimitMB)
	}
	if dbEx.Config.Algorithm == nil || len(dbEx.Config.Algorithm.TestCases) != 4 {
		t.Fatalf("DB test cases mismatch: expected 4 cases, got %v", dbEx.Config.Algorithm)
	}
}

// TestSlice13_Commit1_PublishTransitionsAndZeroPublicValidation prueba el ciclo de publicación y validación 422
func TestSlice13_Commit1_PublishTransitionsAndZeroPublicValidation(t *testing.T) {
	ts, db := setupSlice13TestServer(t)
	defer ts.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	exZeroPublicID := uuid.NewString()

	// 1. Crear ejercicio con 0 casos públicos (solo privados)
	zeroPubExercise := domain.Exercise{
		ID:          exZeroPublicID,
		Title:       "Ejercicio Solo Casos Ocultos",
		Description: "Sin casos públicos para validar el 422",
		Type:        domain.ExerciseTypeAlgorithm,
		Status:      "draft",
		TenantID:    tenantID,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "secret1", ExpectedOutput: "res1", IsHidden: true},
					{Input: "secret2", ExpectedOutput: "res2", IsHidden: true},
				},
			},
		},
	}
	createJSON, _ := json.Marshal(zeroPubExercise)
	req1, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises", bytes.NewBuffer(createJSON))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User-Role", "teacher")
	req1.Header.Set("X-Tenant-Id", tenantID)
	resp1, _ := http.DefaultClient.Do(req1)
	resp1.Body.Close()

	// Intentar publicar ejercicio con 0 casos públicos -> DEBE RETORNAR 422
	pubReq1, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises/"+exZeroPublicID+"/publish", nil)
	pubReq1.Header.Set("X-User-Role", "teacher")
	pubReq1.Header.Set("X-Tenant-Id", tenantID)
	pubResp1, err := http.DefaultClient.Do(pubReq1)
	if err != nil {
		t.Fatalf("POST /publish failed: %v", err)
	}
	defer pubResp1.Body.Close()

	if pubResp1.StatusCode != 422 {
		t.Fatalf("Expected status 422 Unprocessable Entity for 0 public test cases, got %d", pubResp1.StatusCode)
	}

	// 2. Crear ejercicio válido con 1 caso público y 1 privado
	validExID := uuid.NewString()
	validExercise := domain.Exercise{
		ID:          validExID,
		Title:       "Ejercicio Con Caso Público",
		Description: "Listo para publicar",
		Type:        domain.ExerciseTypeAlgorithm,
		Status:      "draft",
		TenantID:    tenantID,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "pub1", ExpectedOutput: "res1", IsHidden: false},
					{Input: "secret1", ExpectedOutput: "res2", IsHidden: true},
				},
			},
		},
	}
	validJSON, _ := json.Marshal(validExercise)
	req2, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises", bytes.NewBuffer(validJSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Role", "teacher")
	req2.Header.Set("X-Tenant-Id", tenantID)
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()

	// Publicar ejercicio válido -> DEBE RETORNAR 200 OK
	pubReq2, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises/"+validExID+"/publish", nil)
	pubReq2.Header.Set("X-User-Role", "teacher")
	pubReq2.Header.Set("X-Tenant-Id", tenantID)
	pubResp2, err := http.DefaultClient.Do(pubReq2)
	if err != nil {
		t.Fatalf("POST /publish failed: %v", err)
	}
	defer pubResp2.Body.Close()

	if pubResp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for valid publish, got %d", pubResp2.StatusCode)
	}

	// AFIRMACIÓN EN BD
	var status string
	err = db.GetDB().Get(&status, "SELECT status FROM exercises WHERE id = $1", validExID)
	if err != nil || status != "published" {
		t.Fatalf("Expected status 'published' in database, got '%s' (err=%v)", status, err)
	}
}

// TestSlice13_Commit1_BulkImportCSV_MalformedAndQuoting prueba importación masiva de casos con comas, CRLF, acentos y fila malformada
func TestSlice13_Commit1_BulkImportCSV_MalformedAndQuoting(t *testing.T) {
	ts, db := setupSlice13TestServer(t)
	defer ts.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	exerciseID := uuid.NewString()

	// Crear ejercicio base
	ex := domain.Exercise{
		ID:          exerciseID,
		Title:       "Ejercicio para Bulk CSV",
		Description: "Testing bulk test cases",
		Type:        domain.ExerciseTypeAlgorithm,
		Status:      "draft",
		TenantID:    tenantID,
	}
	exBytes, _ := json.Marshal(ex)
	reqInit, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises", bytes.NewBuffer(exBytes))
	reqInit.Header.Set("Content-Type", "application/json")
	reqInit.Header.Set("X-User-Role", "teacher")
	reqInit.Header.Set("X-Tenant-Id", tenantID)
	respInit, _ := http.DefaultClient.Do(reqInit)
	respInit.Body.Close()

	// 1. Enviar CSV con fila malformada (línea 4 solo tiene 1 columna)
	csvMalformed := "input,expected_output,is_hidden\r\n" +
		"\"1, 2, 3\",\"6\",false\r\n" +
		"\"Árbol de Decisión, Raíz\",\"Nodo Principal\",true\r\n" +
		"FilaInválidaSinSalida\r\n"

	reqBad, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises/"+exerciseID+"/test-cases/bulk", strings.NewReader(csvMalformed))
	reqBad.Header.Set("Content-Type", "text/csv")
	reqBad.Header.Set("X-User-Role", "teacher")
	reqBad.Header.Set("X-Tenant-Id", tenantID)

	respBad, err := http.DefaultClient.Do(reqBad)
	if err != nil {
		t.Fatalf("POST bulk CSV failed: %v", err)
	}
	defer respBad.Body.Close()

	if respBad.StatusCode != 422 {
		t.Fatalf("Expected status 422 Unprocessable Entity for malformed CSV, got %d", respBad.StatusCode)
	}

	// 2. Enviar CSV corregido con comas embebidas entre comillas, acentos y CRLF
	csvValid := "input,expected_output,is_hidden\r\n" +
		"\"10, 20, 30\",\"60\",false\r\n" +
		"\"Árbol de Decisión, Raíz\",\"Nodo Principal con éxito\",true\r\n" +
		"\"Entrada Simple\",\"Salida Simple\",false\r\n"

	reqGood, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises/"+exerciseID+"/test-cases/bulk", strings.NewReader(csvValid))
	reqGood.Header.Set("Content-Type", "text/csv")
	reqGood.Header.Set("X-User-Role", "teacher")
	reqGood.Header.Set("X-Tenant-Id", tenantID)

	respGood, err := http.DefaultClient.Do(reqGood)
	if err != nil {
		t.Fatalf("POST bulk CSV valid failed: %v", err)
	}
	defer respGood.Body.Close()

	if respGood.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK for valid CSV, got %d", respGood.StatusCode)
	}

	// AFIRMACIÓN DIRECTA EN POSTGRESQL
	var dbEx domain.Exercise
	err = db.GetDB().Get(&dbEx, "SELECT config FROM exercises WHERE id = $1", exerciseID)
	if err != nil {
		t.Fatalf("Failed to fetch exercise config from DB: %v", err)
	}

	if dbEx.Config.Algorithm == nil || len(dbEx.Config.Algorithm.TestCases) != 3 {
		t.Fatalf("Expected 3 test cases saved in DB, got: %v", dbEx.Config.Algorithm)
	}

	tc1 := dbEx.Config.Algorithm.TestCases[0]
	if tc1.Input != "10, 20, 30" || tc1.ExpectedOutput != "60" || tc1.IsHidden != false {
		t.Errorf("TestCase 0 mismatch: got input='%s', output='%s', is_hidden=%v", tc1.Input, tc1.ExpectedOutput, tc1.IsHidden)
	}

	tc2 := dbEx.Config.Algorithm.TestCases[1]
	if tc2.Input != "Árbol de Decisión, Raíz" || tc2.ExpectedOutput != "Nodo Principal con éxito" || tc2.IsHidden != true {
		t.Errorf("TestCase 1 (Unicode) mismatch: got input='%s', output='%s', is_hidden=%v", tc2.Input, tc2.ExpectedOutput, tc2.IsHidden)
	}
}

// TestSlice13_Commit1_AuthorizationAndCrossTenantIsolation valida rol student -> 403 y cross-tenant -> 404
func TestSlice13_Commit1_AuthorizationAndCrossTenantIsolation(t *testing.T) {
	ts, _ := setupSlice13TestServer(t)
	defer ts.Close()

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"
	exerciseID := uuid.NewString()

	// 1. Rol student intenta crear ejercicio -> 403 Forbidden
	studentPayload := domain.Exercise{
		ID:          exerciseID,
		Title:       "Ejercicio No Autorizado",
		Type:        domain.ExerciseTypeAlgorithm,
		Status:      "draft",
		TenantID:    tenantA,
	}
	stBytes, _ := json.Marshal(studentPayload)
	reqStudent, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises", bytes.NewBuffer(stBytes))
	reqStudent.Header.Set("Content-Type", "application/json")
	reqStudent.Header.Set("X-User-Role", "student") // ROL STUDENT
	reqStudent.Header.Set("X-Tenant-Id", tenantA)

	respStudent, err := http.DefaultClient.Do(reqStudent)
	if err != nil {
		t.Fatalf("Student request failed: %v", err)
	}
	defer respStudent.Body.Close()

	if respStudent.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected status 403 Forbidden for student role, got %d", respStudent.StatusCode)
	}

	// 2. Docente de Tenant A crea el ejercicio exitosamente
	reqTeacher, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises", bytes.NewBuffer(stBytes))
	reqTeacher.Header.Set("Content-Type", "application/json")
	reqTeacher.Header.Set("X-User-Role", "teacher")
	reqTeacher.Header.Set("X-Tenant-Id", tenantA)
	respTeacher, _ := http.DefaultClient.Do(reqTeacher)
	respTeacher.Body.Close()

	// 3. Docente de Tenant B intenta modificar el ejercicio de Tenant A -> 404 Not Found
	updatePayload := domain.Exercise{
		Title: "Intento de modificación Cross-Tenant",
	}
	upBytes, _ := json.Marshal(updatePayload)
	reqCross, _ := http.NewRequest("PUT", ts.URL+"/api/v1/exercises/"+exerciseID, bytes.NewBuffer(upBytes))
	reqCross.Header.Set("Content-Type", "application/json")
	reqCross.Header.Set("X-User-Role", "teacher")
	reqCross.Header.Set("X-Tenant-Id", tenantB) // TENANT B

	respCross, err := http.DefaultClient.Do(reqCross)
	if err != nil {
		t.Fatalf("Cross-tenant request failed: %v", err)
	}
	defer respCross.Body.Close()

	if respCross.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404 Not Found for cross-tenant update, got %d", respCross.StatusCode)
	}

	// 4. Docente de Tenant B intenta publicar ejercicio de Tenant A -> 404 Not Found
	reqPubCross, _ := http.NewRequest("POST", ts.URL+"/api/v1/exercises/"+exerciseID+"/publish", nil)
	reqPubCross.Header.Set("X-User-Role", "teacher")
	reqPubCross.Header.Set("X-Tenant-Id", tenantB) // TENANT B

	respPubCross, err := http.DefaultClient.Do(reqPubCross)
	if err != nil {
		t.Fatalf("Cross-tenant publish request failed: %v", err)
	}
	defer respPubCross.Body.Close()

	if respPubCross.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404 Not Found for cross-tenant publish, got %d", respPubCross.StatusCode)
	}
}
