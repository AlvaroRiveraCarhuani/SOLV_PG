package integration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestRubricEvaluation_RequiresSubjectOwnership cubre el alcance docente de la
// rúbrica: solo califica quien tiene la materia asignada, dentro de su
// institución, y un intento rechazado no deja nota ni celda.
func TestRubricEvaluation_RequiresSubjectOwnership(t *testing.T) {
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
	ownerTeacher := uuid.NewString()
	otherTeacher := uuid.NewString()
	adminID := uuid.NewString()
	studentID := uuid.NewString()
	adminStudentID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	submissionID := uuid.NewString()
	adminSubmissionID := uuid.NewString()
	rubricID := uuid.NewString()

	var templateID string
	if err := db.GetDB().Get(&templateID, `SELECT id FROM lab_templates LIMIT 1`); err != nil {
		t.Skipf("Skipping integration test: no hay plantillas en la base de test: %v", err)
	}

	cleanup := func() {
		_, _ = db.GetDB().Exec(`DELETE FROM submission_comments WHERE submission_id IN ($1, $2)`, submissionID, adminSubmissionID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercise_grades WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM submissions WHERE id IN ($1, $2)`, submissionID, adminSubmissionID)
		_, _ = db.GetDB().Exec(`DELETE FROM rubrics WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2, $3, $4, $5)`, ownerTeacher, otherTeacher, studentID, adminID, adminStudentID)
	}
	defer cleanup()

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Titular', $2, 'teacher', $5),
		       ($3, 'Docente', 'Ajeno', $4, 'teacher', $5),
		       ($6, 'Alumno', 'Rúbrica', $7, 'student', $5),
		       ($8, 'Admin', 'Institución', $9, 'admin', $5),
		       ($10, 'Alumno', 'Admin', $11, 'student', $5)
	`, ownerTeacher, "rubric_owner_"+ownerTeacher[:8]+"@uab.edu.bo",
		otherTeacher, "rubric_other_"+otherTeacher[:8]+"@uab.edu.bo",
		tenantID, studentID, "rubric_student_"+studentID[:8]+"@uab.edu.bo",
		adminID, "rubric_admin_"+adminID[:8]+"@uab.edu.bo",
		adminStudentID, "rubric_admin_student_"+adminStudentID[:8]+"@uab.edu.bo"); err != nil {
		t.Fatalf("no se pudo sembrar usuarios: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Materia con rúbrica', $3, $4)
	`, subjectID, tenantID, "RUB-"+ownerTeacher[:8], ownerTeacher); err != nil {
		t.Fatalf("no se pudo sembrar la materia: %v", err)
	}
	for _, enrolled := range []string{studentID, adminStudentID} {
		if _, err := db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)
		`, tenantID, enrolled, subjectID); err != nil {
			t.Fatalf("no se pudo inscribir al alumno: %v", err)
		}
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, config,
		                       tenant_id, template_id, environment_type, language)
		VALUES ($1, $2, 'Proyecto con rúbrica', 'x', 'project', 'published', '{}'::jsonb,
		        $3, $4, 'IDE_PERSISTENTE', 'javascript')
	`, exerciseID, subjectID, tenantID, templateID); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO rubrics (id, exercise_id, criteria)
		VALUES ($1, $2, $3::jsonb)
	`, rubricID, exerciseID, `[{"id":"c1","name":"Criterio único","weight":100,
		"levels":[{"name":"Logrado","score":100},{"name":"Básico","score":50}]}]`); err != nil {
		t.Fatalf("no se pudo sembrar la rúbrica: %v", err)
	}

	for id, owner := range map[string]string{submissionID: studentID, adminSubmissionID: adminStudentID} {
		if _, err := db.GetDB().Exec(`
			INSERT INTO submissions (
				id, tenant_id, exercise_id, student_id, code, verdict, review_status,
				ast_result, generated_cases, complexity_analysis, submitted_at
			)
			VALUES ($1, $2, $3, $4, '# === FILE: app.js ===' || chr(10) || 'console.log(1);' || chr(10),
			        NULL, 'pending', '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, NOW())
		`, id, tenantID, exerciseID, owner); err != nil {
			t.Fatalf("no se pudo sembrar la entrega: %v", err)
		}
	}

	rubricService := services.NewDefaultRubricService(
		postgres.NewPostgresRubricRepository(db.GetDB()),
		postgres.NewPostgresExerciseRepository(db.GetDB()),
		db.GetDB(),
	)
	request := domain.RubricEvaluationRequest{
		Selections: map[string]string{"c1": "Logrado"},
		Comments:   "Cumple los criterios acordados en clase.",
	}

	ctxFor := func(tenant, user string) context.Context {
		ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenant)
		return context.WithValue(ctx, domain.UserIDKey, user)
	}

	// Un docente de otra materia no alcanza la entrega.
	if _, err := rubricService.EvaluateSubmission(ctxFor(tenantID, otherTeacher), uuid.MustParse(submissionID), request, domain.RubricActor{UserID: otherTeacher}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("un docente ajeno debe recibir ErrNotFound, obtenido %v", err)
	}
	assertRubricRejectionLeftNoTrace(t, db.GetDB(), submissionID, exerciseID)

	// Una institución distinta tampoco alcanza la entrega.
	otherTenant := "00000000-0000-0000-0000-000000000099"
	if _, err := rubricService.EvaluateSubmission(ctxFor(otherTenant, ownerTeacher), uuid.MustParse(submissionID), request, domain.RubricActor{UserID: ownerTeacher}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("otra institución debe recibir ErrNotFound, obtenido %v", err)
	}
	assertRubricRejectionLeftNoTrace(t, db.GetDB(), submissionID, exerciseID)

	// El docente titular sí califica.
	if _, err := rubricService.EvaluateSubmission(ctxFor(tenantID, ownerTeacher), uuid.MustParse(submissionID), request, domain.RubricActor{UserID: ownerTeacher}); err != nil {
		t.Fatalf("el docente titular debe poder calificar: %v", err)
	}
	assertRubricGrade(t, db.GetDB(), submissionID, exerciseID, 100, ownerTeacher)

	// La rúbrica tampoco reemplaza la nota manual de otro docente sin motivo.
	if _, err := db.GetDB().Exec(`UPDATE subjects SET teacher_id = $1 WHERE id = $2`, otherTeacher, subjectID); err != nil {
		t.Fatalf("no se pudo transferir la materia: %v", err)
	}
	lowerRequest := domain.RubricEvaluationRequest{Selections: map[string]string{"c1": "Básico"}}
	if _, err := rubricService.EvaluateSubmission(ctxFor(tenantID, otherTeacher), uuid.MustParse(submissionID), lowerRequest, domain.RubricActor{UserID: otherTeacher}); !errors.Is(err, domain.ErrGradeReasonRequired) {
		t.Fatalf("una rúbrica sin motivo no debe pisar la nota manual ajena, obtenido %v", err)
	}
	var foreignCell struct {
		Score        sql.NullInt64  `db:"score"`
		OverriddenBy sql.NullString `db:"overridden_by"`
	}
	if err := db.GetDB().Get(&foreignCell, `
		SELECT score, overridden_by::text FROM exercise_grades WHERE exercise_id = $1 AND student_id = $2
	`, exerciseID, studentID); err != nil {
		t.Fatalf("no se pudo releer la celda: %v", err)
	}
	if !foreignCell.Score.Valid || int(foreignCell.Score.Int64) != 100 ||
		!foreignCell.OverriddenBy.Valid || foreignCell.OverriddenBy.String != ownerTeacher {
		t.Fatalf("la celda debe conservar la nota y la autoría del titular: %+v", foreignCell)
	}

	lowerRequest.Comments = "Se ajusta la nota tras revisar el alcance entregado."
	if _, err := rubricService.EvaluateSubmission(ctxFor(tenantID, otherTeacher), uuid.MustParse(submissionID), lowerRequest, domain.RubricActor{UserID: otherTeacher}); err != nil {
		t.Fatalf("la recorrección justificada con rúbrica debe aceptarse: %v", err)
	}
	assertRubricGrade(t, db.GetDB(), submissionID, exerciseID, 50, otherTeacher)

	// Un administrador conserva el alcance de su institución.
	if _, err := rubricService.EvaluateSubmission(ctxFor(tenantID, adminID), uuid.MustParse(adminSubmissionID), request, domain.RubricActor{UserID: adminID, IsAdmin: true}); err != nil {
		t.Fatalf("un administrador debe poder calificar: %v", err)
	}
	assertRubricGrade(t, db.GetDB(), adminSubmissionID, exerciseID, 100, adminID)
}

func assertRubricRejectionLeftNoTrace(t *testing.T, db *sqlx.DB, submissionID, exerciseID string) {
	t.Helper()

	var submission struct {
		Score        sql.NullInt64  `db:"score"`
		ReviewStatus string         `db:"review_status"`
		GradedBy     sql.NullString `db:"graded_by"`
	}
	if err := db.Get(&submission, `
		SELECT score, review_status, graded_by::text FROM submissions WHERE id = $1
	`, submissionID); err != nil {
		t.Fatalf("no se pudo releer la entrega: %v", err)
	}
	if submission.Score.Valid || submission.GradedBy.Valid || submission.ReviewStatus != "pending" {
		t.Fatalf("un intento rechazado no debe modificar la entrega: %+v", submission)
	}

	var cells int
	if err := db.Get(&cells, `SELECT COUNT(*) FROM exercise_grades WHERE selected_submission_id = $1`, submissionID); err != nil {
		t.Fatalf("no se pudo contar la libreta: %v", err)
	}
	if cells != 0 {
		t.Fatalf("un intento rechazado no debe escribir la libreta; celdas: %d", cells)
	}

	var comments int
	if err := db.Get(&comments, `SELECT COUNT(*) FROM submission_comments WHERE submission_id = $1`, submissionID); err != nil {
		t.Fatalf("no se pudo contar los comentarios: %v", err)
	}
	if comments != 0 {
		t.Fatalf("un intento rechazado no debe dejar comentarios; comentarios: %d", comments)
	}
}

func assertRubricGrade(t *testing.T, db *sqlx.DB, submissionID, exerciseID string, wantScore int, wantAuthor string) {
	t.Helper()

	var submission struct {
		Score    sql.NullInt64  `db:"score"`
		Verdict  sql.NullString `db:"verdict"`
		GradedBy sql.NullString `db:"graded_by"`
	}
	if err := db.Get(&submission, `
		SELECT score, verdict, graded_by::text FROM submissions WHERE id = $1
	`, submissionID); err != nil {
		t.Fatalf("no se pudo releer la entrega: %v", err)
	}
	if !submission.Score.Valid || int(submission.Score.Int64) != wantScore {
		t.Fatalf("nota esperada %d, obtenida %+v", wantScore, submission.Score)
	}
	if submission.Verdict.Valid {
		t.Fatalf("la rúbrica no debe escribir veredicto, obtenido %q", submission.Verdict.String)
	}
	if !submission.GradedBy.Valid || submission.GradedBy.String != wantAuthor {
		t.Fatalf("graded_by esperado %s, obtenido %+v", wantAuthor, submission.GradedBy)
	}

	var cell struct {
		Score        sql.NullInt64  `db:"score"`
		Source       string         `db:"source"`
		OverriddenBy sql.NullString `db:"overridden_by"`
	}
	if err := db.Get(&cell, `
		SELECT score, source, overridden_by::text
		FROM exercise_grades
		WHERE selected_submission_id = $1 AND exercise_id = $2
	`, submissionID, exerciseID); err != nil {
		t.Fatalf("no se escribió la celda de la libreta: %v", err)
	}
	if !cell.Score.Valid || int(cell.Score.Int64) != wantScore || cell.Source != "manual" {
		t.Fatalf("celda inesperada: %+v", cell)
	}
	if !cell.OverriddenBy.Valid || cell.OverriddenBy.String != wantAuthor {
		t.Fatalf("overridden_by esperado %s, obtenido %+v", wantAuthor, cell.OverriddenBy)
	}
}
