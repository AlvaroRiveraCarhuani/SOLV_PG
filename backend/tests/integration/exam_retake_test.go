package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestExamRetakeRepository comprueba contra PostgreSQL las piezas que habilitan
// un recuperatorio: la convocatoria de examen más nueva (solo de ejercicios de
// examen) y el reinicio de la práctica sobre el intento siguiente. La fila es la
// misma porque la base admite una sola por persona, materia y práctica.
func TestExamRetakeRepository(t *testing.T) {
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
	}
	ctx := context.Background()
	conn := db.GetDB()
	tenantID := "00000000-0000-0000-0000-000000000001"

	teacherID := uuid.NewString()
	studentID := uuid.NewString()
	subjectID := uuid.NewString()
	examExerciseID := uuid.NewString()
	classExerciseID := uuid.NewString()
	firstSessionID := uuid.NewString()
	retakeSessionID := uuid.NewString()
	classSessionID := uuid.NewString()
	environmentID := uuid.NewString()

	var templateID string
	if err := conn.GetContext(ctx, &templateID,
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status IN ('approved','APROBADA') LIMIT 1"); err != nil {
		t.Skipf("se omite la prueba: no hay plantilla de juez aprobada: %v", err)
	}

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("fallo al preparar datos: %v", err)
		}
	}

	t.Cleanup(func() {
		cleanup := context.Background()
		_, _ = conn.ExecContext(cleanup, `DELETE FROM class_sessions WHERE id IN ($1, $2, $3)`, firstSessionID, retakeSessionID, classSessionID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM environments WHERE id = $1`, environmentID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM exercises WHERE id IN ($1, $2)`, examExerciseID, classExerciseID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM users WHERE id IN ($1, $2)`, teacherID, studentID)
	})

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Recuperatorio', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_recup_%s@uab.edu.bo", teacherID[:8]), tenantID)
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alumno', 'Recuperatorio', $2, 'student', $3)`,
		studentID, fmt.Sprintf("alumno_recup_%s@uab.edu.bo", studentID[:8]), tenantID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Recuperatorio de Prueba', $3, $4)`,
		subjectID, tenantID, "REC-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Examen', 'published', 'JUEZ_EFIMERO', 'exam', $4)`,
		examExerciseID, tenantID, subjectID, templateID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Clase', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		classExerciseID, tenantID, subjectID, templateID)

	// El examen se rindió y cerró hace dos horas; el docente abrió la
	// convocatoria del recuperatorio hace diez minutos. El aula de clase se abrió
	// después: no puede confundirse con un recuperatorio.
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '90 minutes', NOW() - INTERVAL '90 minutes')`,
		firstSessionID, tenantID, subjectID, examExerciseID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW() - INTERVAL '10 minutes', NOW() + INTERVAL '80 minutes')`,
		retakeSessionID, tenantID, subjectID, examExerciseID)
	// Solo puede haber un aula abierta por materia: la de clase se modela cerrada.
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '1 minute', NOW() - INTERVAL '1 minute')`,
		classSessionID, tenantID, subjectID, classExerciseID)

	mustExec(`INSERT INTO environments (id, student_id, subject_id, exercise_id, tenant_id, status, type, access_url, environment_profile, exam_attempt, frozen_at)
		VALUES ($1, $2, $3, $4, $5, 'frozen', 'IDE_PERSISTENTE', 'http://127.0.0.1:3000', 'exam', 1, NOW() - INTERVAL '90 minutes')`,
		environmentID, studentID, subjectID, examExerciseID, tenantID)

	sessionRepo := postgres.NewPostgresClassSessionRepository(conn)
	openedAt, found, err := sessionRepo.LatestExamSessionOpen(ctx, tenantID, subjectID, examExerciseID)
	if err != nil {
		t.Fatalf("LatestExamSessionOpen falló: %v", err)
	}
	if !found {
		t.Fatal("la convocatoria del recuperatorio debía encontrarse")
	}
	if delta := time.Since(openedAt); delta < 9*time.Minute || delta > 11*time.Minute {
		t.Fatalf("la convocatoria más nueva debía ser la de hace diez minutos, se obtuvo %s", openedAt)
	}

	if _, found, err := sessionRepo.LatestExamSessionOpen(ctx, tenantID, subjectID, classExerciseID); err != nil {
		t.Fatalf("LatestExamSessionOpen de una práctica falló: %v", err)
	} else if found {
		t.Fatal("una práctica de clase no puede habilitar un recuperatorio")
	}

	envRepo := postgres.NewPostgresPracticeEnvironmentRepository(conn).(*postgres.PostgresPracticeEnvironmentRepository)
	if err := envRepo.RestartExamAttempt(ctx, environmentID, tenantID, 2, domain.EnvironmentProfileExam, nil, domain.WorkspaceSeedModeNone, domain.WorkspaceInitStatusNotRequired); err != nil {
		t.Fatalf("RestartExamAttempt falló: %v", err)
	}

	var restarted struct {
		Status       string     `db:"status"`
		FrozenAt     *time.Time `db:"frozen_at"`
		ExamAttempt  int        `db:"exam_attempt"`
		ContainerID  *string    `db:"container_id"`
		SeedMode     string     `db:"workspace_seed_mode"`
		ProfileValue string     `db:"environment_profile"`
	}
	if err := conn.GetContext(ctx, &restarted,
		`SELECT status, frozen_at, exam_attempt, container_id, workspace_seed_mode, environment_profile
		 FROM environments WHERE id = $1`, environmentID); err != nil {
		t.Fatalf("no se pudo leer la práctica reiniciada: %v", err)
	}
	if restarted.Status != "pending" || restarted.FrozenAt != nil || restarted.ContainerID != nil {
		t.Fatalf("el intento nuevo debía quedar pendiente, sin cierre ni contenedor: %+v", restarted)
	}
	if restarted.ExamAttempt != 2 || restarted.ProfileValue != "exam" {
		t.Fatalf("el intento nuevo debía ser el 2 de un examen: %+v", restarted)
	}
	if restarted.SeedMode != domain.WorkspaceSeedModeNone {
		t.Fatalf("sin proyecto publicado la siembra debía quedar en none: %+v", restarted)
	}

	current, err := envRepo.GetByStudentSubjectExercise(ctx, studentID, subjectID, examExerciseID)
	if err != nil {
		t.Fatalf("GetByStudentSubjectExercise falló: %v", err)
	}
	if current.ID != environmentID || current.ExamAttempt != 2 || current.FrozenAt != nil {
		t.Fatalf("la práctica vigente debía ser el intento 2 abierto: %+v", current)
	}
}
