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

// TestProjectManualGrade_RecordsScoreProvenanceAndAuthor cubre el recorrido de
// la corrección manual de un proyecto de IDE: la nota, su procedencia manual y
// la identidad del docente quedan escritas en la entrega y en la libreta, y una
// recorrección por otro docente reemplaza la autoría de la nota vigente.
func TestProjectManualGrade_RecordsScoreProvenanceAndAuthor(t *testing.T) {
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
	teacherOne := uuid.NewString()
	teacherTwo := uuid.NewString()
	studentID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	submissionID := uuid.NewString()

	var templateID string
	if err := db.GetDB().Get(&templateID, `
		SELECT id FROM lab_templates
		WHERE target_environment = 'IDE_PERSISTENTE' AND status = 'approved'
		LIMIT 1
	`); err != nil {
		t.Skipf("Skipping integration test: no hay plantilla IDE en la base de test: %v", err)
	}

	cleanup := func() {
		_, _ = db.GetDB().Exec(`DELETE FROM exercise_grades WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM submissions WHERE id = $1`, submissionID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2, $3)`, teacherOne, teacherTwo, studentID)
	}
	defer cleanup()

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Uno', $2, 'teacher', $6),
		       ($3, 'Docente', 'Dos', $4, 'teacher', $6),
		       ($5, 'Alumno', 'Proyecto', $7, 'student', $6)
	`, teacherOne, "docente_uno_"+teacherOne[:8]+"@uab.edu.bo",
		teacherTwo, "docente_dos_"+teacherTwo[:8]+"@uab.edu.bo",
		studentID, tenantID, "alumno_proyecto_"+studentID[:8]+"@uab.edu.bo"); err != nil {
		t.Fatalf("no se pudo sembrar usuarios: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Proyecto Manual', $3, $4)
	`, subjectID, tenantID, "MAN-"+teacherOne[:8], teacherOne); err != nil {
		t.Fatalf("no se pudo sembrar la materia: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO enrollments (tenant_id, student_id, subject_id)
		VALUES ($1, $2, $3)
	`, tenantID, studentID, subjectID); err != nil {
		t.Fatalf("no se pudo inscribir al alumno: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (
			id, subject_id, title, description, type, status, config,
			tenant_id, template_id, environment_type, language
		)
		VALUES ($1, $2, 'Proyecto sin rúbrica', 'Proyecto de prueba',
		        'project', 'published', '{}'::jsonb, $3, $4, 'IDE_PERSISTENTE', 'javascript')
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

	score := 73
	if _, err := service.UpdateSubmissionReview(context.Background(), tenantID, teacherOne, submissionID, domain.UpdateReviewRequest{
		Status:          "graded",
		Score:           &score,
		FeedbackSummary: "Cumple los criterios acordados.",
	}); err != nil {
		t.Fatalf("la corrección manual del proyecto falló: %v", err)
	}

	var stored struct {
		Score        sql.NullInt64  `db:"score"`
		Verdict      sql.NullString `db:"verdict"`
		ReviewStatus string         `db:"review_status"`
		ReviewedBy   sql.NullString `db:"reviewed_by"`
		GradedBy     sql.NullString `db:"graded_by"`
	}
	if err := db.GetDB().Get(&stored, `
		SELECT score, verdict, review_status, reviewed_by::text, graded_by::text
		FROM submissions WHERE id = $1
	`, submissionID); err != nil {
		t.Fatalf("no se pudo leer la entrega: %v", err)
	}
	if !stored.Score.Valid || stored.Score.Int64 != 73 {
		t.Errorf("submissions.score: esperado 73, obtenido %+v", stored.Score)
	}
	if stored.Verdict.Valid {
		t.Errorf("un proyecto no debe recibir veredicto, obtenido %q", stored.Verdict.String)
	}
	if stored.ReviewStatus != string(domain.ReviewGraded) {
		t.Errorf("review_status: esperado graded, obtenido %q", stored.ReviewStatus)
	}
	if !stored.GradedBy.Valid || stored.GradedBy.String != teacherOne {
		t.Errorf("graded_by debe ser el docente que asignó la nota: esperado %s, obtenido %+v", teacherOne, stored.GradedBy)
	}
	if !stored.ReviewedBy.Valid || stored.ReviewedBy.String != teacherOne {
		t.Errorf("reviewed_by: esperado %s, obtenido %+v", teacherOne, stored.ReviewedBy)
	}

	var cell struct {
		Score        int            `db:"score"`
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
		Reason       string         `db:"reason"`
	}
	if err := db.GetDB().Get(&cell, `
		SELECT score, source, overridden_by::text, reason
		FROM exercise_grades WHERE student_id = $1 AND exercise_id = $2
	`, studentID, exerciseID); err != nil {
		t.Fatalf("no se escribió la celda de la libreta: %v", err)
	}
	if cell.Score != 73 || cell.Source != "manual" {
		t.Errorf("celda de libreta: esperado 73/manual, obtenido %d/%s", cell.Score, cell.Source)
	}
	if !cell.OverriddenBy.Valid || cell.OverriddenBy.String != teacherOne {
		t.Errorf("overridden_by debe ser el docente: esperado %s, obtenido %+v", teacherOne, cell.OverriddenBy)
	}
	if cell.Reason == "" {
		t.Error("la celda debe declarar el motivo de la corrección")
	}

	// Recalificación por un segundo docente: la autoría de la nota vigente pasa
	// al docente actual, no se conserva la anterior.
	if _, err := db.GetDB().Exec(`UPDATE subjects SET teacher_id = $1 WHERE id = $2`, teacherTwo, subjectID); err != nil {
		t.Fatalf("no se pudo transferir la materia al segundo docente: %v", err)
	}

	secondScore := 88
	if _, err := service.UpdateSubmissionReview(context.Background(), tenantID, teacherTwo, submissionID, domain.UpdateReviewRequest{
		Status:          "graded",
		Score:           &secondScore,
		FeedbackSummary: "Se ajusta la nota tras revisar la entrega final acordada.",
	}); err != nil {
		t.Fatalf("la recalificación por el segundo docente falló: %v", err)
	}

	var regraded struct {
		Score    sql.NullInt64  `db:"score"`
		GradedBy sql.NullString `db:"graded_by"`
	}
	if err := db.GetDB().Get(&regraded, `SELECT score, graded_by::text FROM submissions WHERE id = $1`, submissionID); err != nil {
		t.Fatalf("no se pudo releer la entrega: %v", err)
	}
	if !regraded.Score.Valid || regraded.Score.Int64 != 88 {
		t.Errorf("nota tras recalificar: esperado 88, obtenido %+v", regraded.Score)
	}
	if !regraded.GradedBy.Valid || regraded.GradedBy.String != teacherTwo {
		t.Errorf("graded_by tras recalificar: esperado %s (autor vigente), obtenido %+v", teacherTwo, regraded.GradedBy)
	}

	var regradedCell struct {
		Score        int            `db:"score"`
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
	}
	if err := db.GetDB().Get(&regradedCell, `
		SELECT score, source, overridden_by::text
		FROM exercise_grades WHERE student_id = $1 AND exercise_id = $2
	`, studentID, exerciseID); err != nil {
		t.Fatalf("no se pudo releer la celda: %v", err)
	}
	if regradedCell.Score != 88 || regradedCell.Source != "manual" {
		t.Errorf("celda tras recalificar: esperado 88/manual, obtenido %d/%s", regradedCell.Score, regradedCell.Source)
	}
	if !regradedCell.OverriddenBy.Valid || regradedCell.OverriddenBy.String != teacherTwo {
		t.Errorf("overridden_by tras recalificar: esperado %s, obtenido %+v", teacherTwo, regradedCell.OverriddenBy)
	}

	// Entrega IDE antigua en 'not_required': el docente debe poder corregirla.
	if _, err := db.GetDB().Exec(`
		UPDATE submissions SET review_status = 'not_required', score = NULL, graded_by = NULL WHERE id = $1
	`, submissionID); err != nil {
		t.Fatalf("no se pudo preparar la entrega heredada: %v", err)
	}
	legacyScore := 64
	if _, err := service.UpdateSubmissionReview(context.Background(), tenantID, teacherTwo, submissionID, domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &legacyScore,
	}); err != nil {
		t.Fatalf("una entrega IDE en not_required debe poder calificarse: %v", err)
	}

	var legacy struct {
		Score        sql.NullInt64 `db:"score"`
		ReviewStatus string        `db:"review_status"`
	}
	if err := db.GetDB().Get(&legacy, `SELECT score, review_status FROM submissions WHERE id = $1`, submissionID); err != nil {
		t.Fatalf("no se pudo releer la entrega heredada: %v", err)
	}
	if !legacy.Score.Valid || legacy.Score.Int64 != 64 {
		t.Errorf("nota de la entrega heredada: esperado 64, obtenido %+v", legacy.Score)
	}
	if legacy.ReviewStatus != string(domain.ReviewGraded) {
		t.Errorf("review_status de la entrega heredada: esperado graded, obtenido %q", legacy.ReviewStatus)
	}

	// Volver a registrar la misma entrega no convierte la celda manual en
	// automática ni pierde la autoría del docente vigente.
	if err := repo.RecordGradeFromSubmission(context.Background(), tenantID, submissionID, teacherTwo, "Recalificación acordada con el alumno."); err != nil {
		t.Fatalf("no se pudo volver a registrar la nota: %v", err)
	}
	var reRecorded struct {
		Score        int            `db:"score"`
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
	}
	if err := db.GetDB().Get(&reRecorded, `
		SELECT score, source, overridden_by::text
		FROM exercise_grades WHERE student_id = $1 AND exercise_id = $2
	`, studentID, exerciseID); err != nil {
		t.Fatalf("no se pudo releer la celda tras el nuevo registro: %v", err)
	}
	if reRecorded.Source != "manual" {
		t.Errorf("la celda de una corrección docente debe seguir siendo manual, obtenido %q", reRecorded.Source)
	}
	if !reRecorded.OverriddenBy.Valid || reRecorded.OverriddenBy.String != teacherTwo {
		t.Errorf("la autoría vigente debe conservarse: esperado %s, obtenido %+v", teacherTwo, reRecorded.OverriddenBy)
	}
}

// TestJudgeGrade_KeepsAutomaticProvenanceAndExposesOrigin comprueba que una
// calificación del juez sigue registrándose como automática y que el origen del
// veredicto llega al modelo que leen las vistas del alumno, sin inventar uno
// cuando la fila es histórica.
func TestJudgeGrade_KeepsAutomaticProvenanceAndExposesOrigin(t *testing.T) {
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
	teacherID := uuid.NewString()
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
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2)`, teacherID, studentID)
	}()

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Juez', $2, 'teacher', $4),
		       ($3, 'Alumno', 'Juez', $5, 'student', $4)
	`, teacherID, "docente_juez_"+teacherID[:8]+"@uab.edu.bo",
		studentID, tenantID, "alumno_juez_"+studentID[:8]+"@uab.edu.bo"); err != nil {
		t.Fatalf("no se pudo sembrar usuarios: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Practica Automatica', $3, $4)
	`, subjectID, tenantID, "AUT-"+teacherID[:8], teacherID); err != nil {
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
		VALUES ($1, $2, 'Practica con juez', 'x', 'algorithm', 'published', '{}'::jsonb,
		        $3, $4, 'JUEZ_EFIMERO', 'python')
	`, exerciseID, subjectID, tenantID, templateID); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, code, verdict, verdict_origin,
		                         review_status, graded_by, ast_result, generated_cases, complexity_analysis, submitted_at)
		VALUES ($1, $2, $3, $4, 'print(1)', 'AC', 'judge', 'graded', NULL, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, NOW())
	`, submissionID, tenantID, exerciseID, studentID); err != nil {
		t.Fatalf("no se pudo sembrar la entrega: %v", err)
	}

	repo := postgres.NewPostgresTeacherRepository(db.GetDB())
	if err := repo.RecordGradeFromSubmission(context.Background(), tenantID, submissionID, "", ""); err != nil {
		t.Fatalf("no se pudo registrar la nota: %v", err)
	}

	var cell struct {
		Score        int            `db:"score"`
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
	}
	if err := db.GetDB().Get(&cell, `
		SELECT score, source, overridden_by::text
		FROM exercise_grades WHERE student_id = $1 AND exercise_id = $2
	`, studentID, exerciseID); err != nil {
		t.Fatalf("no se escribió la celda de la libreta: %v", err)
	}
	if cell.Source != "auto" {
		t.Errorf("una calificación del juez debe registrarse como automatica, obtenido %q", cell.Source)
	}
	if cell.Score != 100 {
		t.Errorf("nota automatica: esperado 100, obtenido %d", cell.Score)
	}
	if cell.OverriddenBy.Valid {
		t.Errorf("una celda automatica no tiene docente autor, obtenido %q", cell.OverriddenBy.String)
	}

	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	submissions, err := submissionRepo.ListByStudent(context.Background(), tenantID, studentID)
	if err != nil {
		t.Fatalf("no se pudo listar las entregas del alumno: %v", err)
	}
	if len(submissions) != 1 {
		t.Fatalf("entregas listadas: esperado 1, obtenido %d", len(submissions))
	}
	if submissions[0].VerdictOrigin != domain.VerdictOriginJudge {
		t.Errorf("origen del veredicto: esperado judge, obtenido %q", submissions[0].VerdictOrigin)
	}

	// Una fila histórica conserva su clasificación y nunca se presenta como judge.
	if _, err := db.GetDB().Exec(`UPDATE submissions SET verdict_origin = 'legacy' WHERE id = $1`, submissionID); err != nil {
		t.Fatalf("no se pudo marcar la fila como historica: %v", err)
	}
	submissions, err = submissionRepo.ListByStudent(context.Background(), tenantID, studentID)
	if err != nil {
		t.Fatalf("no se pudo releer el historial: %v", err)
	}
	if submissions[0].VerdictOrigin != domain.VerdictOriginLegacy {
		t.Errorf("origen historico: esperado legacy, obtenido %q", submissions[0].VerdictOrigin)
	}
}
