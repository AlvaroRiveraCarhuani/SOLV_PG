package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestStudentAulasBringsPracticeAndPresence fija el contrato que consume el
// aula del alumno: la sesión abierta llega con el nombre de la materia, el
// título de la práctica de la sesión y la presencia contada por el latido de
// los últimos 120 segundos, no con ceros.
func TestStudentAulasBringsPracticeAndPresence(t *testing.T) {
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
		return
	}
	ctx := context.WithValue(t.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	conn := db.GetDB()

	var templateID string
	if err := conn.GetContext(ctx, &templateID,
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status IN ('approved','APROBADA') LIMIT 1"); err != nil {
		t.Skipf("se omite la prueba: no hay plantilla de juez aprobada: %v", err)
		return
	}

	tenantID := domain.DefaultTenantID
	teacherID := uuid.NewString()
	studentID := uuid.NewString()
	otherStudentID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	sessionID := uuid.NewString()

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := conn.ExecContext(ctx, query, args...); execErr != nil {
			t.Fatalf("fallo al preparar datos: %v", execErr)
		}
	}

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Aula', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_aula_%s@uab.edu.bo", teacherID[:8]), tenantID)
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alumno', 'Aula', $2, 'student', $3)`,
		studentID, fmt.Sprintf("alumno_aula_%s@uab.edu.bo", studentID[:8]), tenantID)
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Otro', 'Alumno', $2, 'student', $3)`,
		otherStudentID, fmt.Sprintf("otro_aula_%s@uab.edu.bo", otherStudentID[:8]), tenantID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Estructuras de Datos', $3, $4)`,
		subjectID, tenantID, "EDA-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Recursión básica', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		exerciseID, tenantID, subjectID, templateID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		sessionID, tenantID, subjectID, exerciseID)
	mustExec(`INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)`,
		tenantID, studentID, subjectID)
	mustExec(`INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)`,
		tenantID, otherStudentID, subjectID)
	mustExec(`INSERT INTO class_session_participants (session_id, student_id, tenant_id, joined_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())`, sessionID, studentID, tenantID)
	mustExec(`INSERT INTO class_session_participants (session_id, student_id, tenant_id, joined_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW() - INTERVAL '10 minutes')`, sessionID, otherStudentID, tenantID)

	repo := postgres.NewPostgresClassSessionRepository(conn)
	aulas, err := repo.OpenSessionsForStudent(ctx, tenantID, studentID)
	if err != nil {
		t.Fatalf("error inesperado al leer las aulas: %v", err)
	}
	if len(aulas) != 1 {
		t.Fatalf("se esperaba un aula abierta, hubo %d", len(aulas))
	}

	aula := aulas[0]
	if aula.SubjectName != "Estructuras de Datos" {
		t.Fatalf("materia inesperada: %q", aula.SubjectName)
	}
	if aula.ExerciseTitle == nil || *aula.ExerciseTitle != "Recursión básica" {
		t.Fatalf("práctica inesperada: %v", aula.ExerciseTitle)
	}
	if aula.ExerciseID == nil || *aula.ExerciseID != exerciseID {
		t.Fatalf("la práctica del aula debía ser la de la sesión: %v", aula.ExerciseID)
	}
	if aula.Enrolled != 2 {
		t.Fatalf("inscritos inesperados: %d", aula.Enrolled)
	}
	if aula.Present != 1 {
		t.Fatalf("presentes inesperados: %d (solo el latido reciente cuenta)", aula.Present)
	}
}
