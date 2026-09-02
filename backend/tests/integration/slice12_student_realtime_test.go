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
	"time"

	"github.com/gorilla/websocket"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

type mockSlice12Docker struct{}

func (m *mockSlice12Docker) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	return nil
}
func (m *mockSlice12Docker) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	return nil
}
func (m *mockSlice12Docker) StartWorkspaceContainer(ctx context.Context, config domain.WorkspaceContainerConfig) (string, error) {
	return "mock-container-lote1", nil
}
func (m *mockSlice12Docker) UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error {
	return nil
}
func (m *mockSlice12Docker) GetContainerMetrics(ctx context.Context, containerID string) (*domain.ContainerMetrics, error) {
	return &domain.ContainerMetrics{MemoryUsageBytes: 50 * 1024 * 1024}, nil
}
func (m *mockSlice12Docker) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	return nil
}
func (m *mockSlice12Docker) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
func (m *mockSlice12Docker) RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error) {
	return []byte("{}"), nil
}

type mockSlice12HostMonitor struct{}

func (m *mockSlice12HostMonitor) GetHostMemoryStats() (freePct float64, availableMB uint64, err error) {
	return 0.50, 8192, nil
}
func (m *mockSlice12HostMonitor) CanAllocateMemory(requiredMB int64) bool {
	return true
}

func setupSlice12TestServer(t *testing.T) (*httptest.Server, *database.Database, *httpdelivery.WebSocketHub) {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "solv_user"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "solv_secure_password"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "solv_db"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil, nil
	}

	_ = db.RunInitialMigrations()

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	auditRepo := postgres.NewAuditLogRepository(db.GetDB())
	workspaceRepo := postgres.NewPostgresWorkspaceRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	teacherInvRepo := postgres.NewPostgresTeacherInvitationRepository(db.GetDB())

	dockerClient := &mockSlice12Docker{}
	hostMonitor := &mockSlice12HostMonitor{}

	evalService := services.NewEvaluationService(exerciseRepo, nil, nil, nil)
	subService := services.NewSubmissionService(submissionRepo)
	workspaceService := services.NewWorkspaceService(workspaceRepo, dockerClient, hostMonitor)
	subjectService := services.NewSubjectService(subjectRepo)
	teacherInvService := services.NewTeacherInvitationService(teacherInvRepo)

	wsHub := httpdelivery.NewWebSocketHub()
	go wsHub.Run()

	wsHandler := httpdelivery.NewWebSocketHandler(wsHub, nil)

	adminHandler := httpdelivery.NewAdminHandler(auditRepo, tenantRepo, workspaceRepo)
	studentHandler := httpdelivery.NewStudentHandler(subjectRepo, workspaceRepo, submissionRepo, exerciseRepo)

	evalHandler := httpdelivery.NewEvaluationHandler(evalService, nil)
	evalHandler.SetWebSocketHub(wsHub)

	subHandler := httpdelivery.NewSubmissionHandler(subService)
	subHandler.SetWebSocketHub(wsHub)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, "00000000-0000-0000-0000-000000000001")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handlers := httpdelivery.Handlers{
		UserHandler:              httpdelivery.NewUserHandler(db, nil),
		TemplateHandler:          httpdelivery.NewTemplateHandler(db, nil),
		AuthHandler:              nil,
		EvaluationHandler:        evalHandler,
		WorkspaceHandler:         httpdelivery.NewWorkspaceHandler(workspaceService, nil),
		MetricsHandler:           nil,
		ConfigHandler:            httpdelivery.NewConfigHandler(tenantRepo),
		SubjectHandler:           httpdelivery.NewSubjectHandler(subjectService),
		SubmissionHandler:        subHandler,
		TeacherInvitationHandler: httpdelivery.NewTeacherInvitationHandler(teacherInvService),
		ClassroomHandler:         httpdelivery.NewClassroomHandler(),
		AdminHandler:             adminHandler,
		StudentHandler:           studentHandler,
		WebSocketHandler:         wsHandler,
		TenantMiddleware:         tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db, wsHub
}

func TestSlice12_GetMeEndpoint(t *testing.T) {
	server, db, _ := setupSlice12TestServer(t)
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "33333333-3333-4333-a333-333333333333"

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Ada', 'Lovelace', 'ada_lote1@uab.edu.bo', 'student', $2)
		ON CONFLICT (id) DO NOTHING;
	`, userID, tenantID)

	// 1. Petición sin X-User-Id -> 401 Unauthorized
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/users/me", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized without header, got %d", resp.StatusCode)
	}

	// 2. Petición con X-User-Id válido -> 200 OK
	req, _ = http.NewRequest("GET", server.URL+"/api/v1/users/me", nil)
	req.Header.Set("X-User-Id", userID)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected JSON with data payload, got %v", body)
	}

	if data["email"] != "ada_lote1@uab.edu.bo" {
		t.Errorf("Expected email ada_lote1@uab.edu.bo, got %v", data["email"])
	}
	if data["role"] != "student" {
		t.Errorf("Expected role student, got %v", data["role"])
	}
	if data["full_name"] != "Ada Lovelace" {
		t.Errorf("Expected full_name Ada Lovelace, got %v", data["full_name"])
	}
}

func TestSlice12_GetDueAssignmentsEndpoint(t *testing.T) {
	server, db, _ := setupSlice12TestServer(t)
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	studentID := "44444444-4444-4444-a444-444444444444"
	subjectID := "55555555-5555-4555-a555-555555555555"
	exerciseID := "66666666-6666-4666-a666-666666666666"

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alan', 'Turing', 'alan_lote1@uab.edu.bo', 'student', $2)
		ON CONFLICT (id) DO NOTHING;
	`, studentID, tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code)
		VALUES ($1, $2, 'Estructuras de Datos Lote1', 'ED-101')
		ON CONFLICT (id) DO NOTHING;
	`, subjectID, tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO enrollments (tenant_id, student_id, subject_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`, tenantID, studentID, subjectID)

	futureDue := time.Now().Add(48 * time.Hour)
	_, _ = db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, due_date, config, tenant_id)
		VALUES ($1, $2, 'Laboratorio #01: Árboles AVL', 'Implemente balanceo AVL', 'algorithm', $3, '{}'::jsonb, $4)
		ON CONFLICT (id) DO UPDATE SET due_date = $3, subject_id = $2;
	`, exerciseID, subjectID, futureDue, tenantID)

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/student/assignments/due", nil)
	req.Header.Set("X-User-Id", studentID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	dataList, ok := body["data"].([]interface{})
	if !ok || len(dataList) == 0 {
		t.Fatalf("Expected non-empty data list of assignments, got %v", body)
	}

	first := dataList[0].(map[string]interface{})
	if first["title"] != "Laboratorio #01: Árboles AVL" {
		t.Errorf("Expected title 'Laboratorio #01: Árboles AVL', got %v", first["title"])
	}
	if first["subject_name"] != "Estructuras de Datos Lote1" {
		t.Errorf("Expected subject_name 'Estructuras de Datos Lote1', got %v", first["subject_name"])
	}
}

func TestSlice12_PauseWorkspaceEndpoint(t *testing.T) {
	server, db, _ := setupSlice12TestServer(t)
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	studentID := "77777777-7777-4777-a777-777777777777"
	subjectID := "88888888-8888-4888-a888-888888888888"
	workspaceID := "99999999-9999-4999-a999-999999999999"

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Grace', 'Hopper', 'grace_lote1@uab.edu.bo', 'student', $2)
		ON CONFLICT (id) DO NOTHING;
	`, studentID, tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code)
		VALUES ($1, $2, 'Compiladores Lote1', 'CMP-202')
		ON CONFLICT (id) DO NOTHING;
	`, subjectID, tenantID)

	if _, err := db.GetDB().Exec(`
		INSERT INTO workspaces (id, student_id, subject_id, status, type, access_url, memory_limit_mb, tenant_id)
		VALUES ($1, $2, $3, 'running', 'IDE_PERSISTENTE', 'http://workspace.local', 256, $4)
		ON CONFLICT (id) DO UPDATE SET status = 'running';
	`, workspaceID, studentID, subjectID, tenantID); err != nil {
		t.Fatalf("Failed to insert test workspace: %v", err)
	}

	// 1. Intento por usuario no autorizado -> 403 Forbidden
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/workspaces/"+workspaceID+"/pause", nil)
	req.Header.Set("X-User-Id", "intruder-id-123")
	req.Header.Set("X-User-Role", "student")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for unauthorized user, got %d", resp.StatusCode)
	}

	// 2. Intento por el estudiante dueño -> 200 OK
	req, _ = http.NewRequest("POST", server.URL+"/api/v1/workspaces/"+workspaceID+"/pause", nil)
	req.Header.Set("X-User-Id", studentID)
	req.Header.Set("X-User-Role", "student")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	if data["status"] != "hibernated" {
		t.Errorf("Expected workspace status to be 'hibernated', got %v", data["status"])
	}
}

func TestSlice12_WebSocketHubAndEvaluation(t *testing.T) {
	server, db, _ := setupSlice12TestServer(t)
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	studentID := "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	exerciseID := "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb"

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v1/evaluations?token=mock-token"
	header := http.Header{}
	header.Set("X-User-Id", studentID)
	header.Set("X-Tenant-Id", tenantID)

	wsConn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v, resp: %v", err, resp)
	}
	defer wsConn.Close()

	// 1. Leer mensaje inicial de bienvenida
	var initMsg httpdelivery.WebSocketMessage
	err = wsConn.ReadJSON(&initMsg)
	if err != nil {
		t.Fatalf("Failed to read initial WS message: %v", err)
	}
	if initMsg.Event != "CONNECTION_ESTABLISHED" {
		t.Errorf("Expected CONNECTION_ESTABLISHED event, got %v", initMsg.Event)
	}

	// 2. Disparar creación de submission HTTP para este usuario y verificar evento emitido por WS
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'WS', 'Student', 'ws_student@uab.edu.bo', 'student', $2)
		ON CONFLICT (id) DO NOTHING;
	`, studentID, tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO exercises (id, title, description, type, config, tenant_id)
		VALUES ($1, 'Ejercicio WS Test', 'Suma simple', 'algorithm', '{}'::jsonb, $2)
		ON CONFLICT (id) DO NOTHING;
	`, exerciseID, tenantID)

	subReqBody := map[string]interface{}{
		"exercise_id": exerciseID,
		"student_id":  studentID,
		"code":        "print('hola')",
		"verdict":     "AC",
	}
	subJSON, _ := json.Marshal(subReqBody)

	req, _ := http.NewRequest("POST", server.URL+"/api/v1/submissions", bytes.NewReader(subJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", tenantID)

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to create submission: %v", err)
	}
	if httpResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created from submission API, got %d", httpResp.StatusCode)
	}

	// 3. Verificar que el WebSocket recibió el evento EVALUATION_COMPLETED
	_ = wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var wsEvent httpdelivery.WebSocketMessage
	err = wsConn.ReadJSON(&wsEvent)
	if err != nil {
		t.Fatalf("Failed to receive WS notification for submission: %v", err)
	}
	if wsEvent.Event != "EVALUATION_COMPLETED" {
		t.Errorf("Expected event EVALUATION_COMPLETED, got %v", wsEvent.Event)
	}
}
