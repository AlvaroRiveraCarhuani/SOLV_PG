package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestSlice13_TeacherReviewModalityAndRubricClosure cubre el contrato de la
// revisión por modalidad: environment_type y language siempre presentes,
// files[] en IDE y el cierre de la rúbrica en review_status + exercise_grades.
func TestSlice13_TeacherReviewModalityAndRubricClosure(t *testing.T) {
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
	rubricID := uuid.NewString()

	var templateID string
	err = db.GetDB().Get(&templateID, `
		SELECT id FROM lab_templates
		WHERE target_environment = 'IDE_PERSISTENTE' AND status = 'approved'
		LIMIT 1
	`)
	if err != nil {
		t.Skipf("Skipping integration test: no hay plantilla IDE en la base de test: %v", err)
	}

	cleanup := func() {
		_, _ = db.GetDB().Exec(`DELETE FROM exercise_grades WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM submission_comments WHERE submission_id = $1`, submissionID)
		_, _ = db.GetDB().Exec(`DELETE FROM rubrics WHERE exercise_id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM submissions WHERE id = $1`, submissionID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2)`, teacherID, studentID)
	}
	defer cleanup()

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Modalidad', $2, 'teacher', $3),
		       ($4, 'Alumno', 'Modalidad', $5, 'student', $3)
	`, teacherID, "docente_modalidad_"+teacherID[:8]+"@uab.edu.bo", tenantID,
		studentID, "alumno_modalidad_"+studentID[:8]+"@uab.edu.bo"); err != nil {
		t.Fatalf("no se pudo sembrar usuarios: %v", err)
	}

	subjectCode := "SIS-" + teacherID[:8]
	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Proyecto de Software - Modalidad', $3, $4)
	`, subjectID, tenantID, subjectCode, teacherID); err != nil {
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
			tenant_id, template_id, environment_type, language, boilerplate
		)
		VALUES ($1, $2, 'Práctica IDE con snapshot', 'Proyecto de prueba',
		        'project', 'published', '{}'::jsonb, $3, $4, 'IDE_PERSISTENTE', 'javascript', $5)
	`, exerciseID, subjectID, tenantID, templateID, "# === FILE: app.js ===\nimport express from \"express\";\n"); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio IDE: %v", err)
	}

	code := "# === FILE: app.js ===\n" +
		"import express from \"express\";\n" +
		"\n" +
		"# === FILE: package.json ===\n" +
		"{\"name\":\"practica-3\"}\n"

	if _, err := db.GetDB().Exec(`
		INSERT INTO submissions (
			id, tenant_id, exercise_id, student_id, code, verdict, review_status,
			ast_result, generated_cases, complexity_analysis, submitted_at
		)
		VALUES ($1, $2, $3, $4, $5, 'PENDING', 'pending',
		        '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, NOW())
	`, submissionID, tenantID, exerciseID, studentID, code); err != nil {
		t.Fatalf("no se pudo sembrar la entrega IDE: %v", err)
	}

	criteria := `[
		{"id": "c1", "name": "Arquitectura", "description": "", "weight": 50,
		 "levels": [
		   {"name": "Cumple", "score": 100, "description": ""},
		   {"name": "No cumple", "score": 0, "description": ""}
		 ]},
		{"id": "c2", "name": "Calidad", "description": "", "weight": 50,
		 "levels": [
		   {"name": "Cumple", "score": 100, "description": ""},
		   {"name": "No cumple", "score": 0, "description": ""}
		 ]}
	]`
	if _, err := db.GetDB().Exec(`
		INSERT INTO rubrics (id, exercise_id, criteria)
		VALUES ($1, $2, $3::jsonb)
	`, rubricID, exerciseID, criteria); err != nil {
		t.Fatalf("no se pudo sembrar la rúbrica: %v", err)
	}

	repo := postgres.NewPostgresTeacherRepository(db.GetDB())
	review, err := repo.GetTeacherSubmissionReview(context.Background(), tenantID, teacherID, submissionID)
	if err != nil {
		t.Fatalf("GetTeacherSubmissionReview devolvió error: %v", err)
	}

	if review.EnvironmentType != domain.WorkspaceTypeIDEPersistente {
		t.Errorf("environment_type: esperado %s, obtenido %q", domain.WorkspaceTypeIDEPersistente, review.EnvironmentType)
	}
	if review.Language != "javascript" {
		t.Errorf("language: esperado javascript, obtenido %q", review.Language)
	}
	if review.Boilerplate == "" {
		t.Error("boilerplate en IDE debe conservarse como proyecto base")
	}
	if !review.BaseFilesAvailable || len(review.BaseFiles) != 1 {
		t.Fatalf("base_files debe traer el proyecto base del ejercicio: %+v", review.BaseFiles)
	}
	if review.BaseFiles[0].Path != "app.js" || review.BaseFiles[0].Content != "import express from \"express\";\n" {
		t.Errorf("archivo base inesperado: %+v", review.BaseFiles[0])
	}
	if !review.FilesAvailable {
		t.Error("files_available debe ser true para un snapshot IDE válido")
	}
	if review.MetricsAvailable {
		t.Error("metrics_available debe ser false en IDE")
	}
	if len(review.TestCases) != 0 {
		t.Errorf("test_cases en IDE debe ser [], se obtuvieron %d", len(review.TestCases))
	}
	if len(review.Files) != 2 {
		t.Fatalf("se esperaban 2 archivos, se obtuvieron %d", len(review.Files))
	}
	if review.Files[0].Path != "app.js" || review.Files[1].Path != "package.json" {
		t.Errorf("orden de archivos inesperado: %+v", review.Files)
	}
	if review.Files[0].StartLine != 2 || review.Files[0].LineCount != 1 {
		t.Errorf("offset de app.js inesperado: %+v", review.Files[0])
	}
	if review.Files[1].StartLine != 5 || review.Files[1].LineCount != 1 {
		t.Errorf("offset de package.json inesperado: %+v", review.Files[1])
	}
	if review.Files[0].Content != "import express from \"express\";\n" {
		t.Errorf("contenido de app.js inesperado: %q", review.Files[0].Content)
	}
	if review.Files[1].Content != "{\"name\":\"practica-3\"}\n" {
		t.Errorf("contenido de package.json inesperado: %q", review.Files[1].Content)
	}

	ctx := context.WithValue(context.Background(), domain.UserIDKey, teacherID)
	rubricService := services.NewDefaultRubricService(
		postgres.NewPostgresRubricRepository(db.GetDB()),
		postgres.NewPostgresExerciseRepository(db.GetDB()),
		db.GetDB(),
	)
	result, err := rubricService.EvaluateSubmission(ctx, uuid.MustParse(submissionID), domain.RubricEvaluationRequest{
		Selections: map[string]string{"c1": "Cumple", "c2": "No cumple"},
		Comments:   "Arquitectura correcta; falta endurecer la validación del body.",
	})
	if err != nil {
		t.Fatalf("EvaluateSubmission devolvió error: %v", err)
	}
	if result.Score != 50 {
		t.Errorf("score de la rúbrica: esperado 50, obtenido %.2f", result.Score)
	}

	var submission struct {
		Score          sql.NullInt64  `db:"score"`
		Verdict        string         `db:"verdict"`
		ReviewStatus   string         `db:"review_status"`
		ReviewedAt     sql.NullTime   `db:"reviewed_at"`
		ReviewedBy     sql.NullString `db:"reviewed_by"`
		Feedback       string         `db:"feedback_summary"`
		ManualOverride sql.NullBool   `db:"manual_override"`
	}
	if err := db.GetDB().Get(&submission, `
		SELECT score, verdict, review_status, reviewed_at, reviewed_by, feedback_summary, manual_override
		FROM submissions WHERE id = $1
	`, submissionID); err != nil {
		t.Fatalf("no se pudo leer la entrega calificada: %v", err)
	}
	if !submission.Score.Valid || submission.Score.Int64 != 50 {
		t.Errorf("submissions.score: esperado 50, obtenido %+v", submission.Score)
	}
	if submission.ReviewStatus != string(domain.ReviewGraded) {
		t.Errorf("review_status: esperado graded, obtenido %q", submission.ReviewStatus)
	}
	if !submission.ReviewedAt.Valid || submission.ReviewedAt.Time.Before(time.Now().Add(-time.Minute)) {
		t.Errorf("reviewed_at no quedó escrito: %+v", submission.ReviewedAt)
	}
	if !submission.ReviewedBy.Valid || submission.ReviewedBy.String != teacherID {
		t.Errorf("reviewed_by: esperado %s, obtenido %+v", teacherID, submission.ReviewedBy)
	}
	if submission.Feedback == "" {
		t.Error("feedback_summary no quedó escrito")
	}
	if !submission.ManualOverride.Valid || !submission.ManualOverride.Bool {
		t.Error("manual_override debe quedar en true tras la rúbrica")
	}

	var cell struct {
		SelectedSubmissionID string `db:"selected_submission_id"`
		Score                int    `db:"score"`
		Source               string `db:"source"`
		SubjectID            string `db:"subject_id"`
	}
	if err := db.GetDB().Get(&cell, `
		SELECT selected_submission_id, score, source, subject_id
		FROM exercise_grades WHERE student_id = $1 AND exercise_id = $2
	`, studentID, exerciseID); err != nil {
		t.Fatalf("no se escribió la celda de la libreta: %v", err)
	}
	if cell.SelectedSubmissionID != submissionID || cell.Score != 50 || cell.Source != "auto" || cell.SubjectID != subjectID {
		t.Errorf("celda de libreta inesperada: %+v", cell)
	}
}
