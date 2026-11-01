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

// TestClassAdvanceRepository_EstadoDeCadaDestinatario comprueba la lectura que
// usa el docente para ver quién aplicó, quién descartó y quién sigue pendiente.
// La consulta trae nombre y correo de cada alumno, agrupa por emisión y no
// expone aulas de otro docente.
func TestClassAdvanceRepository_EstadoDeCadaDestinatario(t *testing.T) {
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
	}
	ctx := context.Background()
	conn := db.GetDB()
	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()
	otherTeacherID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	sessionID := uuid.NewString()
	advanceID := uuid.NewString()

	var templateID string
	if err := conn.GetContext(ctx, &templateID,
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status='approved' LIMIT 1"); err != nil {
		t.Skipf("se omite la prueba: no hay plantilla de juez aprobada: %v", err)
	}

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("fallo al preparar datos: %v", err)
		}
	}

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Avance', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_recipients_%s@uab.edu.bo", teacherID[:8]), tenantID)
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Otro', 'Docente', $2, 'teacher', $3)`,
		otherTeacherID, fmt.Sprintf("otro_docente_recipients_%s@uab.edu.bo", otherTeacherID[:8]), tenantID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Avance de Destinatarios', $3, $4)`,
		subjectID, tenantID, "AVR-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Práctica con estados', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		exerciseID, tenantID, subjectID, templateID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		sessionID, tenantID, subjectID, exerciseID)

	students := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	names := [][2]string{{"Ana", "Aplicada"}, {"Beto", "Descartado"}, {"Carla", "Pendiente"}}
	for i, studentID := range students {
		mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, $3, $4, 'student', $5)`,
			studentID, names[i][0], names[i][1],
			fmt.Sprintf("alumno_recipients_%d_%s@uab.edu.bo", i+1, studentID[:8]), tenantID)
	}

	mustExec(`INSERT INTO class_advances (id, tenant_id, session_id, exercise_id, scope,
		snapshot_key, snapshot_sha256, snapshot_size_bytes, file_count, created_at)
		VALUES ($1, $2, $3, $4, 'class', $5, $6, 512, 3, NOW())`,
		advanceID, tenantID, sessionID, exerciseID,
		"class-advances/"+tenantID+"/"+advanceID+".tar.gz", repeatHex('c'))

	mustExec(`INSERT INTO class_advance_recipients (advance_id, student_id, tenant_id, status, decided_at)
		VALUES ($1, $2, $3, 'applied', NOW())`, advanceID, students[0], tenantID)
	mustExec(`INSERT INTO class_advance_recipients (advance_id, student_id, tenant_id, status, decided_at)
		VALUES ($1, $2, $3, 'dismissed', NOW())`, advanceID, students[1], tenantID)
	mustExec(`INSERT INTO class_advance_recipients (advance_id, student_id, tenant_id, status)
		VALUES ($1, $2, $3, 'pending')`, advanceID, students[2], tenantID)

	sessions := postgres.NewPostgresClassSessionRepository(conn)
	repo := postgres.NewPostgresClassAdvanceRepository(conn, sessions)

	advances, err := repo.SessionAdvances(ctx, tenantID, teacherID, sessionID)
	if err != nil {
		t.Fatalf("SessionAdvances falló: %v", err)
	}
	if len(advances) != 1 {
		t.Fatalf("se esperaba una emisión, hubo %d", len(advances))
	}
	advance := advances[0]
	if advance.ID != advanceID || advance.ExerciseTitle != "Práctica con estados" {
		t.Errorf("la emisión no coincide con lo guardado: %+v", advance)
	}
	if advance.Counts.Total != 3 || advance.Counts.Applied != 1 ||
		advance.Counts.Dismissed != 1 || advance.Counts.Pending != 1 {
		t.Errorf("conteos inesperados: %+v", advance.Counts)
	}
	if len(advance.Recipients) != 3 {
		t.Fatalf("se esperaban tres destinatarios, hubo %d", len(advance.Recipients))
	}

	// Las filas llegan ordenadas por nombre: la vista no necesita ordenar.
	wantNames := []string{"Ana Aplicada", "Beto Descartado", "Carla Pendiente"}
	wantStatuses := []string{
		domain.AdvanceRecipientApplied,
		domain.AdvanceRecipientDismissed,
		domain.AdvanceRecipientPending,
	}
	for i, recipient := range advance.Recipients {
		if recipient.Name != wantNames[i] {
			t.Errorf("destinatario %d: nombre %q, se esperaba %q", i, recipient.Name, wantNames[i])
		}
		if recipient.Status != wantStatuses[i] {
			t.Errorf("destinatario %d: estado %q, se esperaba %q", i, recipient.Status, wantStatuses[i])
		}
		if recipient.Email == "" {
			t.Errorf("destinatario %d: el correo debía venir resuelto", i)
		}
		if wantStatuses[i] == domain.AdvanceRecipientPending && recipient.DecidedAt != nil {
			t.Errorf("un pendiente no tiene hora de decisión: %+v", recipient.DecidedAt)
		}
		if wantStatuses[i] != domain.AdvanceRecipientPending && recipient.DecidedAt == nil {
			t.Errorf("un destinatario decidido debía traer su hora: %+v", recipient)
		}
	}

	// El docente de otra materia no ve el aula ajena.
	foreign, err := repo.SessionAdvances(ctx, tenantID, otherTeacherID, sessionID)
	if err != nil {
		t.Fatalf("SessionAdvances de otro docente falló: %v", err)
	}
	if len(foreign) != 0 {
		t.Fatalf("otro docente no debía ver la emisión: %+v", foreign)
	}
}

// repeatHex devuelve un hash de 64 caracteres para el snapshot de prueba.
func repeatHex(c byte) string {
	out := make([]byte, 64)
	for i := range out {
		out[i] = c
	}
	return string(out)
}
