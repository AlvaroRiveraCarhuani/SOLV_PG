package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func TestAcademicSchemaAndMultiTenancy(t *testing.T) {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:postgres@127.0.0.1:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}

	if err := db.RunInitialMigrations(); err != nil {
		t.Fatalf("Failed to run initial migrations: %v", err)
	}

	sqlDB := db.GetDB()

	// 1. Limpieza de datos de prueba previos
	sqlDB.Exec("DELETE FROM submissions")
	sqlDB.Exec("DELETE FROM enrollments")
	sqlDB.Exec("DELETE FROM workspaces")
	sqlDB.Exec("DELETE FROM subjects")
	sqlDB.Exec("DELETE FROM teacher_invitations")
	sqlDB.Exec("DELETE FROM users WHERE email LIKE '%@test-academic.edu.bo'")
	sqlDB.Exec("DELETE FROM tenants WHERE slug LIKE 'academic-tenant-%'")

	// 2. Crear 2 Tenants de prueba para verificar aislamiento
	tenantA := &domain.Tenant{
		ID:   "10000000-0000-0000-0000-000000000001",
		Name: "Tenant A Academic",
		Slug: "academic-tenant-a",
	}
	tenantB := &domain.Tenant{
		ID:   "20000000-0000-0000-0000-000000000002",
		Name: "Tenant B Academic",
		Slug: "academic-tenant-b",
	}

	if _, err := sqlDB.Exec("INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)", tenantA.ID, tenantA.Name, tenantA.Slug); err != nil {
		t.Fatalf("Failed to insert tenantA: %v", err)
	}
	if _, err := sqlDB.Exec("INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)", tenantB.ID, tenantB.Name, tenantB.Slug); err != nil {
		t.Fatalf("Failed to insert tenantB: %v", err)
	}

	// Crear usuarios en Tenant A (Student 1, Student 2, Teacher Candidate)
	student1ID := "a0000000-0000-0000-0000-000000000001"
	student2ID := "a0000000-0000-0000-0000-000000000002"
	teacherID := "a0000000-0000-0000-0000-000000000003"

	userQuery := `INSERT INTO users (id, tenant_id, email, first_name, last_name, role) VALUES ($1, $2, $3, $4, $5, $6)`
	sqlDB.Exec(userQuery, student1ID, tenantA.ID, "student1@test-academic.edu.bo", "Estudiante", "Uno", "student")
	sqlDB.Exec(userQuery, student2ID, tenantA.ID, "student2@test-academic.edu.bo", "Estudiante", "Dos", "student")
	sqlDB.Exec(userQuery, teacherID, tenantA.ID, "teacher@test-academic.edu.bo", "Docente", "Candidato", "student")

	subjectRepo := postgres.NewPostgresSubjectRepository(sqlDB)
	submissionRepo := postgres.NewPostgresSubmissionRepository(sqlDB)
	teacherInvRepo := postgres.NewPostgresTeacherInvitationRepository(sqlDB)

	subjectService := services.NewSubjectService(subjectRepo)
	submissionService := services.NewSubmissionService(submissionRepo)
	teacherInvService := services.NewTeacherInvitationService(teacherInvRepo)

	ctxA := context.Background()

	// 3. Crear Materia e Inscribir 2 Alumnos
	subjA, err := subjectService.CreateSubject(ctxA, tenantA.ID, "Estructura de Datos", "INF-210", nil)
	if err != nil {
		t.Fatalf("Failed to create subject: %v", err)
	}

	if _, err := subjectService.EnrollStudent(ctxA, tenantA.ID, student1ID, subjA.ID); err != nil {
		t.Fatalf("Failed to enroll student 1: %v", err)
	}
	if _, err := subjectService.EnrollStudent(ctxA, tenantA.ID, student2ID, subjA.ID); err != nil {
		t.Fatalf("Failed to enroll student 2: %v", err)
	}

	studentsEnrolled, err := subjectService.ListStudents(ctxA, tenantA.ID, subjA.ID)
	if err != nil || len(studentsEnrolled) != 2 {
		t.Fatalf("Expected 2 enrolled students, got %d (err: %v)", len(studentsEnrolled), err)
	}

	// 4. Registrar 3 Submissions con distintos veredictos (AC, WA, TLE)
	exerciseID := "e1e1e1e1-e1e1-4e1e-a1e1-e1e1e1e1e1e1"

	sub1, err := submissionService.CreateSubmission(ctxA, tenantA.ID, services.CreateSubmissionDTO{
		ExerciseID:      exerciseID,
		StudentID:       student1ID,
		Code:            "print(a + b)",
		Verdict:         "AC",
		ExecutionTimeMS: 12,
		MemoryUsedMB:    16,
	})
	if err != nil {
		t.Fatalf("Failed to create submission 1: %v", err)
	}

	sub2, err := submissionService.CreateSubmission(ctxA, tenantA.ID, services.CreateSubmissionDTO{
		ExerciseID:      exerciseID,
		StudentID:       student1ID,
		Code:            "print(a - b)",
		Verdict:         "WA",
		ExecutionTimeMS: 15,
		MemoryUsedMB:    16,
	})
	if err != nil {
		t.Fatalf("Failed to create submission 2: %v", err)
	}

	sub3, err := submissionService.CreateSubmission(ctxA, tenantA.ID, services.CreateSubmissionDTO{
		ExerciseID:      exerciseID,
		StudentID:       student2ID,
		Code:            "while True: pass",
		Verdict:         "TLE",
		ExecutionTimeMS: 2000,
		MemoryUsedMB:    32,
	})
	if err != nil {
		t.Fatalf("Failed to create submission 3: %v", err)
	}

	if sub1.ID == "" || sub2.ID == "" || sub3.ID == "" {
		t.Fatalf("Expected non-empty submission IDs")
	}

	// 5. Verificar Regla (b): Student ve solo sus 2 entregas; Teacher ve las 3
	studentSubs, err := submissionService.GetSubmissionsForExercise(ctxA, tenantA.ID, exerciseID, student1ID, "student")
	if err != nil || len(studentSubs) != 2 {
		t.Fatalf("Expected student1 to see 2 submissions, got %d (err: %v)", len(studentSubs), err)
	}

	teacherSubs, err := submissionService.GetSubmissionsForExercise(ctxA, tenantA.ID, exerciseID, teacherID, "teacher")
	if err != nil || len(teacherSubs) != 3 {
		t.Fatalf("Expected teacher to see 3 submissions, got %d (err: %v)", len(teacherSubs), err)
	}

	// 6. Verificar Regla (a): Aceptación de Invitación Docente en Transacción Atómica
	inv, err := teacherInvService.CreateInvitation(ctxA, tenantA.ID, "teacher@test-academic.edu.bo", 24)
	if err != nil {
		t.Fatalf("Failed to create teacher invitation: %v", err)
	}

	// Intento fallido con email incorrecto
	errWrongEmail := teacherInvService.AcceptInvitation(ctxA, tenantA.ID, inv.Token, teacherID, "wrong@test.edu.bo")
	if errWrongEmail == nil {
		t.Fatalf("Expected error when accepting invitation with wrong email, got nil")
	}

	// Intento exitoso con email correcto
	errAccept := teacherInvService.AcceptInvitation(ctxA, tenantA.ID, inv.Token, teacherID, "teacher@test-academic.edu.bo")
	if errAccept != nil {
		t.Fatalf("Failed to accept teacher invitation: %v", errAccept)
	}

	// Verificar rol actualizado a 'teacher'
	var updatedRole string
	err = sqlDB.GetContext(ctxA, &updatedRole, "SELECT role FROM users WHERE id = $1", teacherID)
	if err != nil || updatedRole != "teacher" {
		t.Fatalf("Expected user role to be 'teacher', got '%s' (err: %v)", updatedRole, err)
	}

	// Intento de reuso del token debe fallar
	errReuse := teacherInvService.AcceptInvitation(ctxA, tenantA.ID, inv.Token, teacherID, "teacher@test-academic.edu.bo")
	if errReuse == nil {
		t.Fatalf("Expected error on invitation reuse, got nil")
	}

	// 7. Verificar Aislamiento Multi-Tenant (Tenant B no debe ver materias ni entregas de Tenant A)
	subjBList, err := subjectService.ListSubjects(ctxA, tenantB.ID)
	if err != nil || len(subjBList) != 0 {
		t.Fatalf("Tenant B should have 0 subjects, got %d", len(subjBList))
	}

	tenantBSubs, err := submissionService.GetSubmissionsForExercise(ctxA, tenantB.ID, exerciseID, "any-user", "teacher")
	if err != nil || len(tenantBSubs) != 0 {
		t.Fatalf("Tenant B should have 0 submissions for exercise, got %d", len(tenantBSubs))
	}

	t.Log("PASS: Academic Schema and Multi-Tenancy isolation verified successfully!")
}

func TestAcademicHTTPAPIEndToEnd(t *testing.T) {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:postgres@127.0.0.1:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		t.Skipf("Skipping HTTP test: database not available: %v", err)
	}

	sqlDB := db.GetDB()
	tenantRepo := postgres.NewPostgresTenantRepository(sqlDB)
	subjectRepo := postgres.NewPostgresSubjectRepository(sqlDB)
	submissionRepo := postgres.NewPostgresSubmissionRepository(sqlDB)
	teacherInvRepo := postgres.NewPostgresTeacherInvitationRepository(sqlDB)

	subjectService := services.NewSubjectService(subjectRepo)
	submissionService := services.NewSubmissionService(submissionRepo)
	teacherInvService := services.NewTeacherInvitationService(teacherInvRepo)

	jwtSecret := []byte("secret")
	tenantMiddleware := middleware.WithTenant(tenantRepo, jwtSecret)

	handlers := &httpdelivery.Handlers{
		SubjectHandler:           httpdelivery.NewSubjectHandler(subjectService),
		SubmissionHandler:        httpdelivery.NewSubmissionHandler(submissionService),
		TeacherInvitationHandler: httpdelivery.NewTeacherInvitationHandler(teacherInvService),
		ClassroomHandler:         httpdelivery.NewClassroomHandler(),
		TenantMiddleware:         tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, handlers)

	// Generar token JWT para la prueba HTTP
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": "00000000-0000-0000-0000-000000000001",
		"user_id":   "a0000000-0000-0000-0000-000000000001",
		"email":     "test@uab.edu.bo",
		"role":      "teacher",
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("Failed to sign test JWT token: %v", err)
	}

	// Test GET /api/v1/classroom/import
	req, _ := http.NewRequest("GET", "/api/v1/classroom/import?course_id=course123", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for classroom import, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "unidirectional_manual_import") {
		t.Fatalf("Expected D6 unidirectional import response, got %s", rr.Body.String())
	}
}
