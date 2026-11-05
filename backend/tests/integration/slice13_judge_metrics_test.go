package integration

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestCourseAnalytics_SeparatesJudgeFromIDEGrades comprueba que las tasas de
// éxito y los casos fallados son métricas del juez: un proyecto de IDE con
// nota docente no entra en esas series aunque comparta dificultad y etiqueta.
func TestCourseAnalytics_SeparatesJudgeFromIDEGrades(t *testing.T) {
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
	judgeExerciseID := uuid.NewString()
	ideExerciseID := uuid.NewString()

	var templateID string
	if err := db.GetDB().Get(&templateID, `SELECT id FROM lab_templates LIMIT 1`); err != nil {
		t.Skipf("Skipping integration test: no hay plantillas en la base de test: %v", err)
	}

	defer func() {
		_, _ = db.GetDB().Exec(`DELETE FROM submissions WHERE exercise_id IN ($1, $2)`, judgeExerciseID, ideExerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id IN ($1, $2)`, judgeExerciseID, ideExerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2)`, teacherID, studentID)
	}()

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Métricas', $2, 'teacher', $4),
		       ($3, 'Alumno', 'Métricas', $5, 'student', $4)
	`, teacherID, "metrics_teacher_"+teacherID[:8]+"@uab.edu.bo",
		studentID, tenantID, "metrics_student_"+studentID[:8]+"@uab.edu.bo"); err != nil {
		t.Fatalf("no se pudo sembrar usuarios: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Materia de métricas', $3, $4)
	`, subjectID, tenantID, "MET-"+teacherID[:8], teacherID); err != nil {
		t.Fatalf("no se pudo sembrar la materia: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)
	`, tenantID, studentID, subjectID); err != nil {
		t.Fatalf("no se pudo inscribir al alumno: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, config,
		                       tenant_id, template_id, environment_type, language, difficulty, tags)
		VALUES ($1, $2, 'Práctica con juez', 'x', 'algorithm', 'published', '{}'::jsonb,
		        $3, $4, 'JUEZ_EFIMERO', 'python', 'medium', ARRAY['grafos']),
		       ($5, $2, 'Proyecto de IDE', 'x', 'project', 'published', '{}'::jsonb,
		        $3, $4, 'IDE_PERSISTENTE', 'javascript', 'medium', ARRAY['grafos'])
	`, judgeExerciseID, subjectID, tenantID, templateID, ideExerciseID); err != nil {
		t.Fatalf("no se pudo sembrar los ejercicios: %v", err)
	}

	if _, err := db.GetDB().Exec(`
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, code, verdict,
		                         review_status, score, ast_result, generated_cases, complexity_analysis)
		VALUES ($1, $2, $3, $4, 'print(1)', 'WA', 'not_required', NULL,
		        '{}'::jsonb, '{}'::jsonb, '{}'::jsonb),
		       ($5, $2, $6, $4, 'console.log(1)', NULL, 'graded', 85,
		        '{}'::jsonb, '{}'::jsonb, '{}'::jsonb)
	`, uuid.NewString(), tenantID, judgeExerciseID, studentID, uuid.NewString(), ideExerciseID); err != nil {
		t.Fatalf("no se pudieron sembrar las entregas: %v", err)
	}

	repo := postgres.NewPostgresTeacherRepository(db.GetDB())
	analytics, err := repo.GetCourseAnalytics(context.Background(), tenantID, teacherID, subjectID)
	if err != nil {
		t.Fatalf("no se pudieron obtener las métricas: %v", err)
	}

	medium, ok := analytics.DifficultyDistribution["medium"]
	if !ok {
		t.Fatalf("falta la dificultad medium: %+v", analytics.DifficultyDistribution)
	}
	if medium.Count != 1 {
		t.Fatalf("la serie del juez solo debe contar la práctica evaluada; ejercicios: %d", medium.Count)
	}
	if medium.SuccessRate != 0 {
		t.Fatalf("una práctica con WA no es éxito; tasa: %.2f", medium.SuccessRate)
	}

	if len(analytics.TopTags) != 1 || analytics.TopTags[0].Tag != "grafos" || analytics.TopTags[0].Count != 1 {
		t.Fatalf("las etiquetas deben salir solo del juez: %+v", analytics.TopTags)
	}

	for _, failed := range analytics.MostFailedCases {
		if failed.ExerciseTitle == "Proyecto de IDE" {
			t.Fatalf("un proyecto de IDE no es un caso fallado del juez: %+v", failed)
		}
	}
	if len(analytics.MostFailedCases) != 1 || analytics.MostFailedCases[0].ExerciseTitle != "Práctica con juez" {
		t.Fatalf("los casos fallados deben salir solo del juez: %+v", analytics.MostFailedCases)
	}
}
