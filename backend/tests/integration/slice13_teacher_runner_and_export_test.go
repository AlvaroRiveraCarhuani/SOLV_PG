package integration

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupTeacherRunnerAndExportTestServer(t *testing.T) (*httptest.Server, *database.Database) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://solv_user:solv_secure_password@localhost:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil
	}

	_ = db.RunInitialMigrations()

	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())
	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	teacherService := services.NewTeacherService(teacherRepo, submissionRepo)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = "00000000-0000-0000-0000-000000000001"
			}
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handlers := httpdelivery.Handlers{
		TeacherHandler:   teacherHandler,
		TenantMiddleware: tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

func TestSlice13_TeacherRunnerAndExportSuite(t *testing.T) {
	server, db := setupTeacherRunnerAndExportTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()

	// 1. Crear Docente
	_, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Profesor', 'RunnerExport', $2, 'teacher', $3)
		ON CONFLICT (id) DO NOTHING;
	`, teacherID, fmt.Sprintf("prof_runexp_%s@uab.edu.bo", teacherID[:8]), tenantID)
	if err != nil {
		t.Fatalf("Failed to seed teacher: %v", err)
	}

	// 2. Crear Materia
	subjectID := uuid.NewString()
	_, err = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Sistemas Distribuidos', 'SIS-402', $3)
		ON CONFLICT (id) DO NOTHING;
	`, subjectID, tenantID, teacherID)
	if err != nil {
		t.Fatalf("Failed to seed subject: %v", err)
	}

	// 3. Crear 3 Estudiantes
	student1ID := uuid.NewString()
	student2ID := uuid.NewString()
	student3ID := uuid.NewString()

	stuData := []struct {
		id    string
		first string
		last  string
	}{
		{student1ID, "Carlos", "Ruiz"},
		{student2ID, "Ana", "Torres"},
		{student3ID, "María", "López"},
	}

	for _, s := range stuData {
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, $3, $4, 'student', $5)
			ON CONFLICT (id) DO NOTHING;
		`, s.id, s.first, s.last, fmt.Sprintf("%s_%s@uab.edu.bo", strings.ToLower(s.first), s.id[:6]), tenantID)

		_, _ = db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3)
		`, tenantID, s.id, subjectID)
	}

	// 4. Crear 2 Laboratorios
	lab1ID := uuid.NewString()
	lab2ID := uuid.NewString()

	_, err = db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, created_at, config, tenant_id)
		VALUES 
			($1, $3, 'Lab 1: Sockets TCP', 'Comunicación cliente servidor', 'algorithm', 'published', NOW() - INTERVAL '2 days', '{}'::jsonb, $4),
			($2, $3, 'Lab 2: RPC Concurrente', 'Llamadas a procedimiento remoto', 'algorithm', 'published', NOW() - INTERVAL '1 day', '{}'::jsonb, $4)
	`, lab1ID, lab2ID, subjectID, tenantID)
	if err != nil {
		t.Fatalf("Failed to create exercises: %v", err)
	}

	// 5. Crear Entregas:
	// - Estudiante 1: Lab 1 (AC -> 100), Lab 2 (WA -> 0) => Promedio: 50.00
	// - Estudiante 2: Lab 1 (AC -> 100), Lab 2 (WA con Override a 90) => Promedio: 95.00
	// - Estudiante 3: Sin entregas => Promedio: 0.00
	sub1ID := uuid.NewString()
	sub2ID := uuid.NewString()
	sub3ID := uuid.NewString()
	sub4ID := uuid.NewString()

	_, err = db.GetDB().Exec(`
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, code, verdict, score, manual_override, submitted_at)
		VALUES 
			($1, $5, $6, $7, 'def s1(): pass', 'AC', 100, FALSE, NOW() - INTERVAL '10 hours'),
			($2, $5, $8, $7, 'def s2(): pass', 'WA', 0, FALSE, NOW() - INTERVAL '8 hours'),
			($3, $5, $6, $9, 'def s3(): pass', 'AC', 100, FALSE, NOW() - INTERVAL '6 hours'),
			($4, $5, $8, $9, 'def s4(): pass', 'WA', 90, TRUE, NOW() - INTERVAL '4 hours')
	`, sub1ID, sub2ID, sub3ID, sub4ID, tenantID, lab1ID, student1ID, lab2ID, student2ID)
	if err != nil {
		t.Fatalf("Failed to seed submissions: %v", err)
	}

	client := &http.Client{}

	// =========================================================================
	// 1. TEST Runner Efímero (POST /api/v1/teacher/submissions/{id}/run-ephemeral)
	// =========================================================================
	t.Run("1. Runner Efímero - Ejecución en Memoria sin Persistencia en DB", func(t *testing.T) {
		// Contar entregas antes de la ejecución efímera
		var countBefore int
		_ = db.GetDB().Get(&countBefore, "SELECT COUNT(*) FROM submissions")

		payload := []byte(`{"code": "def solution(): return 42", "language": "python"}`)
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/run-ephemeral", server.URL, sub1ID), bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for ephemeral run, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		data := body["data"].(map[string]interface{})

		if data["submission_id"] != sub1ID {
			t.Errorf("Expected submission_id %s, got %v", sub1ID, data["submission_id"])
		}
		if data["verdict"] == "" {
			t.Errorf("Expected non-empty verdict in ephemeral run result")
		}

		// REGLA CRÍTICA: Contar entregas después de la ejecución efímera -> DEBE SER EXACTAMENTE IGUAL
		var countAfter int
		_ = db.GetDB().Get(&countAfter, "SELECT COUNT(*) FROM submissions")
		if countBefore != countAfter {
			t.Fatalf("CRITICAL: Ephemeral run persisted a new submission to database! Before: %d, After: %d", countBefore, countAfter)
		}

		// Seguridad: Estudiante no puede ejecutar runner efímero
		reqStudent, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/run-ephemeral", server.URL, sub1ID), bytes.NewBuffer(payload))
		reqStudent.Header.Set("X-User-Role", "student")
		respStudent, _ := client.Do(reqStudent)
		if respStudent.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student ephemeral run, got %d", respStudent.StatusCode)
		}
	})

	// =========================================================================
	// 2. TEST Exportación CSV de Calificaciones (GET /api/v1/teacher/courses/{id}/grades/export?format=csv)
	// =========================================================================
	t.Run("2. Exportación de Calificaciones CSV - Matriz, Headers y UTF-8 BOM", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/courses/%s/grades/export?format=csv", server.URL, subjectID), nil)
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for CSV export, got %d", resp.StatusCode)
		}

		// 2.1 Verificar Headers HTTP
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/csv") {
			t.Errorf("Expected Content-Type text/csv, got %s", contentType)
		}

		contentDisposition := resp.Header.Get("Content-Disposition")
		if !strings.Contains(contentDisposition, "attachment") || !strings.Contains(contentDisposition, "SIS-402") {
			t.Errorf("Expected Content-Disposition attachment with subject code, got %s", contentDisposition)
		}

		// 2.2 Leer contenido del CSV
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		csvBytes := buf.Bytes()

		// Verificar UTF-8 BOM (\xef\xbb\xbf)
		if !bytes.HasPrefix(csvBytes, []byte("\xef\xbb\xbf")) {
			t.Errorf("Expected CSV to start with UTF-8 BOM for Excel compatibility")
		}

		// Parsear CSV
		r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(csvBytes, []byte("\xef\xbb\xbf"))))
		records, err := r.ReadAll()
		if err != nil {
			t.Fatalf("Failed to parse CSV output: %v", err)
		}

		// Encabezado + 3 Estudiantes = 4 filas
		if len(records) != 4 {
			t.Fatalf("Expected 4 rows in CSV (header + 3 students), got %d", len(records))
		}

		// Validar Encabezado
		header := records[0]
		expectedHeader := []string{"ID Estudiante", "Nombre Completo", "Email", "Lab 1: Sockets TCP", "Lab 2: RPC Concurrente", "Promedio Final"}
		for i, h := range expectedHeader {
			if header[i] != h {
				t.Errorf("Header col %d: expected %s, got %s", i, h, header[i])
			}
		}

		// Mapear filas por nombre de estudiante
		studentRows := make(map[string][]string)
		for _, row := range records[1:] {
			studentRows[row[1]] = row
		}

		// Validar notas de Carlos Ruiz (Lab1: 100, Lab2: 0, Promedio: 50.00)
		carlosRow := studentRows["Carlos Ruiz"]
		if carlosRow == nil {
			t.Fatalf("Carlos Ruiz not found in CSV rows")
		}
		if carlosRow[3] != "100" || carlosRow[4] != "0" || carlosRow[5] != "50.00" {
			t.Errorf("Carlos Ruiz expected [100, 0, 50.00], got [%s, %s, %s]", carlosRow[3], carlosRow[4], carlosRow[5])
		}

		// Validar notas de Ana Torres (Lab1: 100, Lab2: 90, Promedio: 95.00)
		anaRow := studentRows["Ana Torres"]
		if anaRow == nil {
			t.Fatalf("Ana Torres not found in CSV rows")
		}
		if anaRow[3] != "100" || anaRow[4] != "90" || anaRow[5] != "95.00" {
			t.Errorf("Ana Torres expected [100, 90, 95.00], got [%s, %s, %s]", anaRow[3], anaRow[4], anaRow[5])
		}

		// Validar notas de María López (Lab1: 0, Lab2: 0, Promedio: 0.00)
		mariaRow := studentRows["María López"]
		if mariaRow == nil {
			t.Fatalf("María López not found in CSV rows")
		}
		if mariaRow[3] != "0" || mariaRow[4] != "0" || mariaRow[5] != "0.00" {
			t.Errorf("María López expected [0, 0, 0.00], got [%s, %s, %s]", mariaRow[3], mariaRow[4], mariaRow[5])
		}

		// 2.3 Seguridad: Estudiante intentando exportar notas -> 403 Forbidden
		reqStudent, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/courses/%s/grades/export?format=csv", server.URL, subjectID), nil)
		reqStudent.Header.Set("X-User-Role", "student")
		respStudent, _ := client.Do(reqStudent)
		if respStudent.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student CSV export, got %d", respStudent.StatusCode)
		}
	})
}
