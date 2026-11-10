package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestExamFreezeRepository comprueba contra PostgreSQL qué aulas son exámenes,
// qué laboratorios participan y cuáles quedan pendientes de congelar. El aula
// de clase no entra nunca: congelar un volumen fuera de un examen dejaría al
// alumno sin poder guardar su trabajo (ADR-005 §5).
func TestExamFreezeRepository(t *testing.T) {
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
	}
	ctx := context.Background()
	conn := db.GetDB()
	tenantID := "00000000-0000-0000-0000-000000000001"

	teacherID := uuid.NewString()
	subjectID := uuid.NewString()
	examExerciseID := uuid.NewString()
	classExerciseID := uuid.NewString()
	closedExamID := uuid.NewString()
	openExamID := uuid.NewString()
	closedClassID := uuid.NewString()
	studentClosedID := uuid.NewString()
	studentOpenID := uuid.NewString()
	studentClassID := uuid.NewString()
	environmentClosedID := uuid.NewString()
	environmentOpenID := uuid.NewString()
	environmentClassID := uuid.NewString()

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
		_, _ = conn.ExecContext(cleanup, `DELETE FROM class_sessions WHERE id IN ($1, $2, $3)`, closedExamID, openExamID, closedClassID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM environments WHERE id IN ($1, $2, $3)`, environmentClosedID, environmentOpenID, environmentClassID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM exercises WHERE id IN ($1, $2)`, examExerciseID, classExerciseID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = conn.ExecContext(cleanup, `DELETE FROM users WHERE id IN ($1, $2, $3, $4)`, teacherID, studentClosedID, studentOpenID, studentClassID)
	})

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Examen', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_examen_%s@uab.edu.bo", teacherID[:8]), tenantID)
	for _, student := range []struct{ id, role string }{
		{studentClosedID, "student"}, {studentOpenID, "student"}, {studentClassID, "student"},
	} {
		mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Alumno', 'Examen', $2, 'student', $3)`,
			student.id, fmt.Sprintf("alumno_examen_%s@uab.edu.bo", student.id[:8]), tenantID)
	}
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Examen de Prueba', $3, $4)`,
		subjectID, tenantID, "EXF-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Examen', 'published', 'JUEZ_EFIMERO', 'exam', $4)`,
		examExerciseID, tenantID, subjectID, templateID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Clase', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		classExerciseID, tenantID, subjectID, templateID)

	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes')`,
		closedExamID, tenantID, subjectID, examExerciseID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		openExamID, tenantID, subjectID, examExerciseID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes')`,
		closedClassID, tenantID, subjectID, classExerciseID)

	for _, participant := range []struct{ sessionID, studentID string }{
		{closedExamID, studentClosedID},
		{openExamID, studentOpenID},
		{closedClassID, studentClassID},
	} {
		mustExec(`INSERT INTO class_session_participants (session_id, student_id, tenant_id)
			VALUES ($1, $2, $3)`, participant.sessionID, participant.studentID, tenantID)
	}
	for _, environment := range []struct{ id, studentID string }{
		{environmentClosedID, studentClosedID},
		{environmentOpenID, studentOpenID},
		{environmentClassID, studentClassID},
	} {
		mustExec(`INSERT INTO environments (id, student_id, subject_id, tenant_id, status, type, access_url)
			VALUES ($1, $2, $3, $4, 'running', 'IDE_PERSISTENTE', 'http://127.0.0.1:3000')`,
			environment.id, environment.studentID, subjectID, tenantID)
	}

	repo := postgres.NewPostgresExamFreezeRepository(conn)

	isExam, err := repo.SessionIsExam(ctx, tenantID, closedExamID)
	if err != nil {
		t.Fatalf("SessionIsExam falló: %v", err)
	}
	if !isExam {
		t.Fatal("la sesión del examen debía reconocerse como examen")
	}
	isExam, err = repo.SessionIsExam(ctx, tenantID, closedClassID)
	if err != nil {
		t.Fatalf("SessionIsExam falló: %v", err)
	}
	if isExam {
		t.Fatal("una práctica de clase no debía tratarse como examen")
	}

	participants, err := repo.ParticipantPracticeEnvironments(ctx, tenantID, closedExamID)
	if err != nil {
		t.Fatalf("ParticipantPracticeEnvironments falló: %v", err)
	}
	if len(participants) != 1 || participants[0].EnvironmentID != environmentClosedID || participants[0].StudentID != studentClosedID {
		t.Fatalf("los participantes del examen no eran los esperados: %+v", participants)
	}

	candidates, err := repo.ClosedExamSessions(ctx, 50)
	if err != nil {
		t.Fatalf("ClosedExamSessions falló: %v", err)
	}
	in := func(sessionID string) bool {
		for _, ref := range candidates {
			if ref.SessionID == sessionID {
				return true
			}
		}
		return false
	}
	if !in(closedExamID) {
		t.Fatalf("el examen cerrado con laboratorio editable debía ser candidato: %+v", candidates)
	}
	if in(openExamID) {
		t.Fatalf("un examen abierto no debía ser candidato: %+v", candidates)
	}
	if in(closedClassID) {
		t.Fatalf("un aula de clase no debía ser candidato: %+v", candidates)
	}

	if err := repo.SetPracticeEnvironmentFrozen(ctx, tenantID, environmentClosedID, true, time.Now()); err != nil {
		t.Fatalf("SetPracticeEnvironmentFrozen falló: %v", err)
	}
	var frozen struct {
		Status   string     `db:"status"`
		FrozenAt *time.Time `db:"frozen_at"`
	}
	if err := conn.GetContext(ctx, &frozen,
		`SELECT status, frozen_at FROM environments WHERE id = $1`, environmentClosedID); err != nil {
		t.Fatalf("no se pudo leer el laboratorio congelado: %v", err)
	}
	if frozen.Status != "frozen" || frozen.FrozenAt == nil {
		t.Fatalf("el laboratorio debía quedar frozen con hora: %+v", frozen)
	}

	candidates, err = repo.ClosedExamSessions(ctx, 50)
	if err != nil {
		t.Fatalf("second ClosedExamSessions falló: %v", err)
	}
	if in(closedExamID) {
		t.Fatalf("un examen ya congelado no debía volver al barrido: %+v", candidates)
	}

	// La pantalla del docente decide con `purpose` si muestra los controles del
	// examen: la sesión tiene que traerlo, no deducirlo del título.
	sessionRepo := postgres.NewPostgresClassSessionRepository(conn)
	examSession, err := sessionRepo.GetSession(ctx, tenantID, teacherID, closedExamID)
	if err != nil {
		t.Fatalf("GetSession del examen falló: %v", err)
	}
	if examSession.Purpose != "exam" {
		t.Fatalf("la sesión del examen debía traer purpose=exam: %+v", examSession)
	}
	classSession, err := sessionRepo.GetSession(ctx, tenantID, teacherID, closedClassID)
	if err != nil {
		t.Fatalf("GetSession de la clase falló: %v", err)
	}
	if classSession.Purpose != "class" {
		t.Fatalf("un aula de clase debía traer purpose=class: %+v", classSession)
	}

	if err := repo.SetPracticeEnvironmentFrozen(ctx, tenantID, environmentClosedID, false, time.Now()); err != nil {
		t.Fatalf("SetPracticeEnvironmentFrozen(false) falló: %v", err)
	}
	if err := conn.GetContext(ctx, &frozen,
		`SELECT status, frozen_at FROM environments WHERE id = $1`, environmentClosedID); err != nil {
		t.Fatalf("no se pudo releer el laboratorio: %v", err)
	}
	if frozen.Status != "running" || frozen.FrozenAt != nil {
		t.Fatalf("la reapertura debía dejar running sin hora: %+v", frozen)
	}
}
