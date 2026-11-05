package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestManualGradeCell_RequiresJustificationFromAnotherTeacher comprueba que la
// nota manual de un docente no la reemplaza otro sin motivo, y que la
// recorrección justificada conserva la autoría y el motivo nuevos.
func TestManualGradeCell_RequiresJustificationFromAnotherTeacher(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}
	defer func() { _ = db.GetDB().Close() }()

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherA := uuid.NewString()
	teacherB := uuid.NewString()
	studentID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	submissionID := uuid.NewString()

	var templateID string
	if err := db.GetDB().Get(&templateID, `SELECT id FROM lab_templates LIMIT 1`); err != nil {
		t.Skipf("Skipping integration test: no hay plantillas en la base de test: %v", err)
	}

	defer func() {
		_, _ = db.GetDB().Exec(`DELETE FROM exercise_grades WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM submissions WHERE id = $1`, submissionID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2, $3)`, teacherA, teacherB, studentID)
	}()

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Uno', $2, 'teacher', $6),
		       ($3, 'Docente', 'Dos', $4, 'teacher', $6),
		       ($5, 'Alumno', 'Celda', $7, 'student', $6)
	`, teacherA, "cell_uno_"+teacherA[:8]+"@uab.edu.bo",
		teacherB, "cell_dos_"+teacherB[:8]+"@uab.edu.bo",
		studentID, tenantID, "cell_student_"+studentID[:8]+"@uab.edu.bo"); err != nil {
		t.Fatalf("no se pudo sembrar usuarios: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Materia de celda', $3, $4)
	`, subjectID, tenantID, "CEL-"+teacherA[:8], teacherA); err != nil {
		t.Fatalf("no se pudo sembrar la materia: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)
	`, tenantID, studentID, subjectID); err != nil {
		t.Fatalf("no se pudo inscribir al alumno: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, config,
		                       tenant_id, template_id, environment_type, language)
		VALUES ($1, $2, 'Proyecto de celda', 'x', 'project', 'published', '{}'::jsonb,
		        $3, $4, 'IDE_PERSISTENTE', 'javascript')
	`, exerciseID, subjectID, tenantID, templateID); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO submissions (
			id, tenant_id, exercise_id, student_id, code, verdict, review_status,
			ast_result, generated_cases, complexity_analysis, submitted_at
		)
		VALUES ($1, $2, $3, $4, '# === FILE: app.js ===' || chr(10) || 'console.log(1);' || chr(10),
		        NULL, 'pending', '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, NOW())
	`, submissionID, tenantID, exerciseID, studentID); err != nil {
		t.Fatalf("no se pudo sembrar la entrega: %v", err)
	}

	repo := postgres.NewPostgresTeacherRepository(db.GetDB())
	service := services.NewTeacherService(repo, nil)
	ctx := context.Background()

	// El docente titular deja una nota manual con su motivo.
	firstScore := 70
	if _, err := service.SetExerciseGrade(ctx, tenantID, teacherA, studentID, exerciseID, domain.SetExerciseGradeRequest{
		SelectedSubmissionID: submissionID,
		Score:                &firstScore,
		Reason:               "Revisión oral acordada con el alumno.",
	}); err != nil {
		t.Fatalf("no se pudo fijar la nota manual: %v", err)
	}

	// La materia pasa al segundo docente, que ya puede corregir la entrega.
	if _, err := db.GetDB().Exec(`UPDATE subjects SET teacher_id = $1 WHERE id = $2`, teacherB, subjectID); err != nil {
		t.Fatalf("no se pudo transferir la materia: %v", err)
	}

	secondScore := 85
	if _, err := service.UpdateSubmissionReview(ctx, tenantID, teacherB, submissionID, domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &secondScore,
	}); err == nil {
		t.Fatal("cambiar la nota de otro docente sin motivo debe rechazarse")
	}

	// El rechazo no deja rastro: ni la entrega ni la celda cambian.
	var untouched struct {
		Score        sql.NullInt64 `db:"score"`
		ReviewStatus string        `db:"review_status"`
	}
	if err := db.GetDB().Get(&untouched, `SELECT score, review_status FROM submissions WHERE id = $1`, submissionID); err != nil {
		t.Fatalf("no se pudo releer la entrega: %v", err)
	}
	if untouched.Score.Valid || untouched.ReviewStatus != "pending" {
		t.Fatalf("un intento rechazado no debe modificar la entrega: %+v", untouched)
	}

	var cellBefore struct {
		Score        sql.NullInt64  `db:"score"`
		OverriddenBy sql.NullString `db:"overridden_by"`
		Reason       string         `db:"reason"`
	}
	if err := db.GetDB().Get(&cellBefore, `
		SELECT score, overridden_by::text, reason FROM exercise_grades WHERE exercise_id = $1
	`, exerciseID); err != nil {
		t.Fatalf("no se pudo releer la celda: %v", err)
	}
	if !cellBefore.Score.Valid || int(cellBefore.Score.Int64) != 70 ||
		!cellBefore.OverriddenBy.Valid || cellBefore.OverriddenBy.String != teacherA {
		t.Fatalf("la celda debe conservar la nota y la autoría anteriores: %+v", cellBefore)
	}

	// La recorrección con motivo sí se registra, con la autoría vigente.
	justification := "Se ajusta la nota tras revisar la entrega final acordada."
	if _, err := service.UpdateSubmissionReview(ctx, tenantID, teacherB, submissionID, domain.UpdateReviewRequest{
		Status:          "graded",
		Score:           &secondScore,
		FeedbackSummary: justification,
	}); err != nil {
		t.Fatalf("la recorrección justificada debe aceptarse: %v", err)
	}

	var cellAfter struct {
		Score        sql.NullInt64  `db:"score"`
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
		Reason       string         `db:"reason"`
	}
	if err := db.GetDB().Get(&cellAfter, `
		SELECT score, source, overridden_by::text, reason FROM exercise_grades WHERE exercise_id = $1
	`, exerciseID); err != nil {
		t.Fatalf("no se pudo releer la celda tras recalificar: %v", err)
	}
	if !cellAfter.Score.Valid || int(cellAfter.Score.Int64) != 85 || cellAfter.Source != "manual" {
		t.Fatalf("la celda debe quedar con la nota nueva y origen manual: %+v", cellAfter)
	}
	if !cellAfter.OverriddenBy.Valid || cellAfter.OverriddenBy.String != teacherB {
		t.Fatalf("la autoría vigente debe ser el segundo docente: %+v", cellAfter)
	}
	if cellAfter.Reason != justification {
		t.Fatalf("el motivo de la recorrección debe quedar escrito: %q", cellAfter.Reason)
	}

	// El mismo docente puede ajustar su nota sin un motivo nuevo, y el motivo
	// que ya estaba escrito no se pierde.
	adjusted := 90
	if _, err := service.UpdateSubmissionReview(ctx, tenantID, teacherB, submissionID, domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &adjusted,
	}); err != nil {
		t.Fatalf("el mismo docente debe poder ajustar su nota: %v", err)
	}
	var cellAdjusted struct {
		Score  sql.NullInt64 `db:"score"`
		Reason string        `db:"reason"`
	}
	if err := db.GetDB().Get(&cellAdjusted, `
		SELECT score, reason FROM exercise_grades WHERE exercise_id = $1
	`, exerciseID); err != nil {
		t.Fatalf("no se pudo releer la celda tras el ajuste: %v", err)
	}
	if !cellAdjusted.Score.Valid || int(cellAdjusted.Score.Int64) != 90 {
		t.Fatalf("el ajuste del mismo docente debe reflejarse en la celda: %+v", cellAdjusted)
	}
	if cellAdjusted.Reason != justification {
		t.Fatalf("el ajuste sin motivo nuevo debe conservar el motivo anterior: %q", cellAdjusted.Reason)
	}
}
