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
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func TestSlice11_5BackendUIBridges(t *testing.T) {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		t.Skip("DATABASE_URL not set, skipping integration tests")
	}

	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.RunInitialMigrations(); err != nil {
		t.Fatalf("Failed to run initial migrations: %v", err)
	}

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	auditRepo := postgres.NewAuditLogRepository(db.GetDB())
	workspaceRepo := postgres.NewPostgresWorkspaceRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())

	evalService := services.NewEvaluationService(exerciseRepo, nil, nil, nil)
	subService := services.NewSubmissionService(submissionRepo)

	adminHandler := httpdelivery.NewAdminHandler(auditRepo, tenantRepo, workspaceRepo)
	studentHandler := httpdelivery.NewStudentHandler(subjectRepo, workspaceRepo, submissionRepo, exerciseRepo)
	evalHandler := httpdelivery.NewEvaluationHandler(evalService, nil)
	subHandler := httpdelivery.NewSubmissionHandler(subService)

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := "11111111-1111-4111-a111-111111111111"
	studentID := "22222222-2222-4222-a222-222222222222"

	// Insert test teacher and student
	_, err = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES 
			($1, 'Profesor', 'Test', 'profesor_slice11_5@uab.edu.bo', 'teacher', $3),
			($2, 'Alumno', 'Test', 'alumno_slice11_5@uab.edu.bo', 'student', $3)
		ON CONFLICT (id) DO NOTHING
	`, teacherID, studentID, tenantID)
	if err != nil {
		t.Fatalf("Failed to seed test users: %v", err)
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &httpdelivery.Handlers{
		EvaluationHandler: evalHandler,
		SubmissionHandler: subHandler,
		AdminHandler:      adminHandler,
		StudentHandler:    studentHandler,
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		},
	})

	// 1. Test Admin Branding Update
	t.Run("1. Admin Update Branding (COST-03)", func(t *testing.T) {
		brandingPayload := []byte(`{
			"logo_url": "https://solv.uab.edu.bo/assets/uab_custom_logo.png",
			"institution_name": "Universidad Adventista de Bolivia - Central",
			"tenant_primary_color": "#1E40AF",
			"support_email": "soporte@uab.edu.bo"
		}`)

		req := httptest.NewRequest("PUT", "/api/v1/admin/branding", bytes.NewReader(brandingPayload))
		req.Header.Set("X-User-Role", "admin")

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		tenant, err := tenantRepo.GetByID(context.Background(), tenantID)
		if err != nil {
			t.Fatalf("Failed to fetch tenant: %v", err)
		}

		var cfg map[string]interface{}
		json.Unmarshal(tenant.Config, &cfg)
		if cfg["tenant_primary_color"] != "#1E40AF" {
			t.Errorf("Expected primary color #1E40AF, got %v", cfg["tenant_primary_color"])
		}
	})

	// 2. Test Admin Audit Logs Filtering
	t.Run("2. Admin Audit Logs Filtering", func(t *testing.T) {
		// Insert test audit log
		testLog := &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      teacherID,
			Action:       "CREATE_SUBJECT",
			ResourceType: "subject",
			StatusCode:   201,
			Metadata:     []byte(`{"name":"Redes II"}`),
			IPAddress:    "127.0.0.1",
			UserAgent:    "IntegrationTest",
		}
		if err := auditRepo.Create(context.Background(), testLog); err != nil {
			t.Fatalf("Failed to insert test audit log: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/admin/audit-logs?action=CREATE_SUBJECT", nil)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		dataList := resp["data"].([]interface{})
		if len(dataList) == 0 {
			t.Fatalf("Expected at least 1 audit log record, got 0")
		}
	})

	// 3. Test Admin Health Metrics
	t.Run("3. Admin Health Metrics Calculation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/metrics/health", nil)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var health httpdelivery.HealthMetricsResponse
		json.Unmarshal(rr.Body.Bytes(), &health)
		if health.TenantID != tenantID {
			t.Errorf("Expected tenant ID %s, got %s", tenantID, health.TenantID)
		}
	})

	// 4. Test Teacher Exercise Creation
	var createdExerciseID string
	t.Run("4. Teacher Exercise Creation (Wizard Endpoint)", func(t *testing.T) {
		freshExID := uuid.New().String()
		exercisePayload := []byte(fmt.Sprintf(`{
			"id": "%s",
			"title": "Búsqueda Binaria Recursiva",
			"description": "Implemente búsqueda binaria recursiva.",
			"type": "algorithm",
			"config": {
				"algorithm": {
					"time_limit_ms": 1500,
					"memory_limit_mb": 128,
					"test_cases": [
						{"input": "[1,2,3,4], 3", "expected_output": "2", "is_hidden": false},
						{"input": "[10,20,30], 99", "expected_output": "-1", "is_hidden": true}
					],
					"ast_rules": {
						"forbidden_imports": ["os", "sys"],
						"forbidden_functions": ["sort"]
					}
				}
			}
		}`, freshExID))

		req := httptest.NewRequest("POST", "/api/v1/exercises", bytes.NewReader(exercisePayload))
		req.Header.Set("X-User-Role", "teacher")

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("Expected HTTP 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		created := resp["data"].(map[string]interface{})
		createdExerciseID = created["id"].(string)
	})

	// 5. Test Public DTO with Public Test Cases for Student
	t.Run("5. Student Public Exercise DTO with Public Test Cases", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/exercises/"+createdExerciseID, nil)
		req.Header.Set("X-User-Role", "student")

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var publicResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &publicResp)

		data := publicResp["data"].(map[string]interface{})
		publicCases, ok := data["public_test_cases"].([]interface{})
		if !ok || len(publicCases) != 1 {
			t.Fatalf("Expected exactly 1 public test case, got %v", data["public_test_cases"])
		}
	})

	// 6. Test Teacher Submission Override and Score
	t.Run("6. Teacher Submission Override and Detail", func(t *testing.T) {
		sampleSubID := uuid.New().String()
		sampleSub := &domain.Submission{
			ID:              sampleSubID,
			TenantID:        tenantID,
			ExerciseID:      createdExerciseID,
			StudentID:       studentID,
			Code:            "def busqueda(): pass",
			Verdict:         "WA",
			ASTResult:       []byte(`{}`),
			ExecutionTimeMS: 50,
			MemoryUsedMB:    10,
			SubmittedAt:     time.Now(),
		}
		if err := submissionRepo.Create(context.Background(), sampleSub); err != nil {
			t.Fatalf("Failed to create sample submission: %v", err)
		}

		// Perform override
		overridePayload := []byte(`{
			"verdict": "AC",
			"override_reason": "Lógica correcta, caso extremo validado por docente.",
			"score": 90
		}`)

		req := httptest.NewRequest("POST", "/api/v1/submissions/"+sampleSubID+"/override", bytes.NewReader(overridePayload))
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		// Fetch detail
		reqGet := httptest.NewRequest("GET", "/api/v1/submissions/"+sampleSubID, nil)

		rrGet := httptest.NewRecorder()
		mux.ServeHTTP(rrGet, reqGet)

		if rrGet.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rrGet.Code, rrGet.Body.String())
		}

		var updatedSub domain.Submission
		json.Unmarshal(rrGet.Body.Bytes(), &updatedSub)
		if updatedSub.Verdict != "AC" || !updatedSub.ManualOverride || *updatedSub.Score != 90 {
			t.Errorf("Override failed: %+v", updatedSub)
		}
	})

	// 7. Test Student Consolidated Dashboard
	t.Run("7. Student Consolidated Dashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/student/dashboard", nil)
		req.Header.Set("X-User-Id", studentID)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected HTTP 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var dash httpdelivery.StudentDashboardResponse
		json.Unmarshal(rr.Body.Bytes(), &dash)
		if dash.StudentID != studentID {
			t.Errorf("Expected student ID %s, got %s", studentID, dash.StudentID)
		}
	})
}
