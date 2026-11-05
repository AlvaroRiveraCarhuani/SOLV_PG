package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// seedReceiptExercise crea un alumno, una materia y un ejercicio de juez en la
// institución base y devuelve sus identificadores. Limpia al terminar.
func openIntegrationDB(t *testing.T) *database.Database {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}
	t.Cleanup(func() { _ = db.GetDB().Close() })
	return db
}

func seedReceiptExercise(t *testing.T, db *database.Database) (tenantID, studentID, exerciseID string) {
	t.Helper()

	tenantID = "00000000-0000-0000-0000-000000000001"
	studentID = uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID = uuid.NewString()

	var templateID string
	if err := db.GetDB().Get(&templateID, `SELECT id FROM lab_templates LIMIT 1`); err != nil {
		t.Skipf("Skipping integration test: no hay plantillas en la base de test: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM submissions WHERE student_id = $1`, studentID)
		_, _ = db.GetDB().Exec(`DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id = $1`, studentID)
	})

	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alumno', 'Recibo', $2, 'student', $3)
	`, studentID, "receipt_replay_"+studentID[:8]+"@uab.edu.bo", tenantID); err != nil {
		t.Fatalf("no se pudo sembrar el alumno: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code)
		VALUES ($1, $2, 'Materia de recibo', $3)
	`, subjectID, tenantID, "REC-"+subjectID[:8]); err != nil {
		t.Fatalf("no se pudo sembrar la materia: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, config,
		                       tenant_id, template_id, environment_type, language)
		VALUES ($1, $2, 'Practica de recibo', 'x', 'algorithm', 'published', '{}'::jsonb,
		        $3, $4, 'JUEZ_EFIMERO', 'python')
	`, exerciseID, subjectID, tenantID, templateID); err != nil {
		t.Fatalf("no se pudo sembrar el ejercicio: %v", err)
	}
	return tenantID, studentID, exerciseID
}

func signReceipt(t *testing.T, tenantID, exerciseID, studentID, code, verdict string, issuedAt time.Time) string {
	t.Helper()
	token, err := services.SignEvaluationReceipt(services.EvaluationReceipt{
		TenantID:        tenantID,
		ExerciseID:      exerciseID,
		StudentID:       studentID,
		Verdict:         verdict,
		ExecutionTimeMS: 12,
		MemoryUsedMB:    16,
		CodeHash:        services.HashEvaluatedCode(code),
		IssuedAt:        issuedAt.Unix(),
	})
	if err != nil {
		t.Fatalf("no se pudo firmar el recibo: %v", err)
	}
	return token
}

func countSubmissions(t *testing.T, db *sqlx.DB, studentID, exerciseID string) int {
	t.Helper()
	var stored int
	if err := db.Get(&stored, `SELECT COUNT(*) FROM submissions WHERE student_id = $1 AND exercise_id = $2`, studentID, exerciseID); err != nil {
		t.Fatalf("no se pudo contar las entregas: %v", err)
	}
	return stored
}

// TestSubmissionReceipt_RegistersOneSubmissionPerEvaluation comprueba contra
// PostgreSQL que el recibo identifica la entrega: reenviarlo devuelve la misma
// fila y no crea una segunda.
func TestSubmissionReceipt_RegistersOneSubmissionPerEvaluation(t *testing.T) {
	db := openIntegrationDB(t)
	tenantID, studentID, exerciseID := seedReceiptExercise(t, db)

	svc := services.NewSubmissionService(postgres.NewPostgresSubmissionRepository(db.GetDB()))
	svc.SetExerciseRepository(postgres.NewPostgresExerciseRepository(db.GetDB()))

	code := "print(1)"
	receipt := signReceipt(t, tenantID, exerciseID, studentID, code, "AC", time.Now())
	dto := services.CreateSubmissionDTO{
		ExerciseID:        exerciseID,
		StudentID:         studentID,
		Code:              code,
		Verdict:           "AC",
		EvaluationReceipt: receipt,
	}

	first, err := svc.CreateSubmission(context.Background(), tenantID, dto)
	if err != nil {
		t.Fatalf("la primera entrega debe registrarse: %v", err)
	}
	second, err := svc.CreateSubmission(context.Background(), tenantID, dto)
	if err != nil {
		t.Fatalf("el reenvío del mismo recibo debe devolver la entrega existente: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("el reenvío debe devolver la misma entrega: %s != %s", second.ID, first.ID)
	}
	if stored := countSubmissions(t, db.GetDB(), studentID, exerciseID); stored != 1 {
		t.Fatalf("un recibo debe dejar una sola entrega, hay %d", stored)
	}

	// Una evaluación nueva (otro recibo) sí registra otra entrega.
	otherCode := "print(2)"
	otherReceipt := signReceipt(t, tenantID, exerciseID, studentID, otherCode, "WA", time.Now().Add(time.Minute))
	third, err := svc.CreateSubmission(context.Background(), tenantID, services.CreateSubmissionDTO{
		ExerciseID:        exerciseID,
		StudentID:         studentID,
		Code:              otherCode,
		EvaluationReceipt: otherReceipt,
	})
	if err != nil {
		t.Fatalf("una evaluación nueva debe registrar otra entrega: %v", err)
	}
	if third.ID == first.ID {
		t.Fatal("dos evaluaciones distintas no pueden compartir entrega")
	}
	if stored := countSubmissions(t, db.GetDB(), studentID, exerciseID); stored != 2 {
		t.Fatalf("esperadas 2 entregas para 2 evaluaciones, hay %d", stored)
	}
}

// TestSubmissionReceipt_ConcurrentReplayPersistsOneSubmission envía el mismo
// recibo desde dos solicitudes a la vez: la clave de la entrega deja una sola
// fila y ambas respuestas coinciden.
func TestSubmissionReceipt_ConcurrentReplayPersistsOneSubmission(t *testing.T) {
	db := openIntegrationDB(t)
	tenantID, studentID, exerciseID := seedReceiptExercise(t, db)

	svc := services.NewSubmissionService(postgres.NewPostgresSubmissionRepository(db.GetDB()))
	svc.SetExerciseRepository(postgres.NewPostgresExerciseRepository(db.GetDB()))

	code := "print(1)"
	receipt := signReceipt(t, tenantID, exerciseID, studentID, code, "AC", time.Now())
	dto := services.CreateSubmissionDTO{
		ExerciseID:        exerciseID,
		StudentID:         studentID,
		Code:              code,
		Verdict:           "AC",
		EvaluationReceipt: receipt,
	}

	type outcome struct {
		submission *domain.Submission
		err        error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			sub, err := svc.CreateSubmission(context.Background(), tenantID, dto)
			results <- outcome{submission: sub, err: err}
		}()
	}
	close(start)

	first := <-results
	second := <-results
	if first.err != nil {
		t.Fatalf("la primera solicitud falló: %v", first.err)
	}
	if second.err != nil {
		t.Fatalf("la segunda solicitud falló: %v", second.err)
	}
	if first.submission.ID != second.submission.ID {
		t.Fatalf("ambas solicitudes deben recuperar la misma entrega: %s != %s",
			first.submission.ID, second.submission.ID)
	}
	if stored := countSubmissions(t, db.GetDB(), studentID, exerciseID); stored != 1 {
		t.Fatalf("dos envíos simultáneos deben dejar una sola entrega, hay %d", stored)
	}
}

// Un error distinto del choque de la clave de la entrega no se reporta como
// reintento: el mapeo no debe ocultar otras violaciones.
func TestSubmissionReceipt_MappingDoesNotMaskOtherErrors(t *testing.T) {
	db := openIntegrationDB(t)
	tenantID, _, exerciseID := seedReceiptExercise(t, db)

	repo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	err := repo.Create(context.Background(), &domain.Submission{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		ExerciseID:    exerciseID,
		StudentID:     "no-es-un-uuid",
		Code:          "print(1)",
		VerdictOrigin: "judge",
	})
	if err == nil {
		t.Fatal("una entrega inválida debe fallar")
	}
	if errors.Is(err, domain.ErrDuplicateSubmission) {
		t.Fatalf("otro fallo de escritura no puede reportarse como reintento: %v", err)
	}
}
