package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// findPending busca un avance en la lista de pendientes. Devuelve un valor vacío
// si no está: así la prueba distingue "no aparece" de una lista con otros avisos.
func findPending(t *testing.T, pending []domain.AdvanceDelivery, advanceID string) domain.AdvanceDelivery {
	t.Helper()
	for _, delivery := range pending {
		if delivery.ID == advanceID {
			return delivery
		}
	}
	return domain.AdvanceDelivery{}
}

// mustPending lista los avisos pendientes del alumno y falla si la consulta no
// responde, para no confundir un error con una lista vacía.
func mustPending(t *testing.T, repo *postgres.PostgresClassAdvanceRepository, tenantID, studentID string) []domain.AdvanceDelivery {
	t.Helper()
	pending, err := repo.PendingAdvances(context.Background(), tenantID, studentID)
	if err != nil {
		t.Fatalf("PendingAdvances falló: %v", err)
	}
	return pending
}

// TestClassAdvanceRepository_EmitirCongelarYVentana comprueba el adaptador
// Postgres de avances: la emisión con sus destinatarios en una transacción, la
// congelación idempotente y la ventana de rezago, que solo se fija con el aula cerrada.
func TestClassAdvanceRepository_EmitirCongelarYVentana(t *testing.T) {
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
	}
	ctx := context.Background()
	conn := db.GetDB()
	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	sessionID := uuid.NewString()

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

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Avance', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_avance_%s@uab.edu.bo", teacherID[:8]), tenantID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Avance de Prueba', $3, $4)`,
		subjectID, tenantID, "AVA-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Práctica con avance', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		exerciseID, tenantID, subjectID, templateID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		sessionID, tenantID, subjectID, exerciseID)

	students := []string{uuid.NewString(), uuid.NewString()}
	for i, studentID := range students {
		mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Alumno', $3, 'student', $4)`,
			studentID, fmt.Sprintf("AlumnoAvance%d", i+1), fmt.Sprintf("alumno_avance%d_%s@uab.edu.bo", i+1, studentID[:8]), tenantID)
		mustExec(`INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)`,
			tenantID, studentID, subjectID)
		mustExec(`INSERT INTO class_session_participants (session_id, student_id, tenant_id)
			VALUES ($1, $2, $3)`, sessionID, studentID, tenantID)
	}

	sessions := postgres.NewPostgresClassSessionRepository(conn)
	repo := postgres.NewPostgresClassAdvanceRepository(conn, sessions)

	t.Run("la emisión guarda el avance y un destinatario pendiente por alumno", func(t *testing.T) {
		advanceID := uuid.NewString()
		advance := domain.ClassAdvance{
			ID:             advanceID,
			TenantID:       tenantID,
			SessionID:      sessionID,
			ExerciseID:     exerciseID,
			Scope:          domain.AdvanceScopeClass,
			SnapshotKey:    "advances/" + advanceID + ".tar.gz",
			SnapshotSHA256: strings.Repeat("a", 64),
			SnapshotSize:   1024,
			FileCount:      3,
			CreatedAt:      time.Now(),
		}
		if err := repo.CreateAdvance(ctx, advance, students); err != nil {
			t.Fatalf("CreateAdvance falló: %v", err)
		}

		var pending int
		if err := conn.GetContext(ctx, &pending,
			`SELECT COUNT(*) FROM class_advance_recipients WHERE advance_id = $1 AND status = 'pending'`,
			advanceID); err != nil {
			t.Fatalf("no se pudo contar destinatarios: %v", err)
		}
		if pending != len(students) {
			t.Fatalf("se esperaban %d destinatarios pendientes, hubo %d", len(students), pending)
		}
	})

	t.Run("congelar dos veces conserva la primera hora y descongelar la limpia", func(t *testing.T) {
		frozen, err := repo.SetFrozen(ctx, tenantID, teacherID, sessionID, true)
		if err != nil {
			t.Fatalf("SetFrozen(true) falló: %v", err)
		}
		if frozen.FrozenAt == nil {
			t.Fatal("frozen_at debía quedar fijado")
		}
		first := *frozen.FrozenAt

		again, err := repo.SetFrozen(ctx, tenantID, teacherID, sessionID, true)
		if err != nil {
			t.Fatalf("segundo SetFrozen(true) falló: %v", err)
		}
		if again.FrozenAt == nil || !again.FrozenAt.Equal(first) {
			t.Fatal("congelar dos veces debía conservar la primera hora")
		}

		unfrozen, err := repo.SetFrozen(ctx, tenantID, teacherID, sessionID, false)
		if err != nil {
			t.Fatalf("SetFrozen(false) falló: %v", err)
		}
		if unfrozen.FrozenAt != nil {
			t.Fatal("frozen_at debía quedar limpio tras descongelar")
		}
	})

	t.Run("cada alumno solo ve y decide su propio avance", func(t *testing.T) {
		advanceID := uuid.NewString()
		mustExec(`INSERT INTO class_advances (id, tenant_id, session_id, exercise_id, scope,
			snapshot_key, snapshot_sha256, snapshot_size_bytes, file_count, created_at)
			VALUES ($1, $2, $3, $4, 'selected', $5, $6, 512, 2, NOW())`,
			advanceID, tenantID, sessionID, exerciseID,
			"class-advances/"+advanceID+".tar.gz", strings.Repeat("b", 64))
		mustExec(`INSERT INTO class_advance_recipients (advance_id, student_id, tenant_id, status)
			VALUES ($1, $2, $3, 'pending')`, advanceID, students[0], tenantID)

		delivery := findPending(t, mustPending(t, repo, tenantID, students[0]), advanceID)
		if delivery.Purpose != "class" || delivery.SubjectID != subjectID {
			t.Errorf("el avance debía traer propósito y materia resueltos: %+v", delivery)
		}
		if delivery.SessionStatus != domain.ClassSessionStatusOpen {
			t.Errorf("el avance debía traer el estado de su sesión: %+v", delivery)
		}

		if other := findPending(t, mustPending(t, repo, tenantID, students[1]), advanceID); other.ID != "" {
			t.Fatal("un avance dirigido a otro alumno no debía aparecer en los pendientes")
		}

		if _, err := repo.AdvanceForStudent(ctx, tenantID, students[1], advanceID); !errors.Is(err, domain.ErrAdvanceNotFound) {
			t.Fatalf("un avance ajeno debía responder no encontrado, fue: %v", err)
		}
		if _, err := repo.AdvanceForStudent(ctx, tenantID, students[0], advanceID); err != nil {
			t.Fatalf("el destinatario debía leer su avance: %v", err)
		}

		backupPath := ".solv_backups/" + advanceID
		if err := repo.DecideRecipient(ctx, tenantID, students[0], advanceID, domain.AdvanceRecipientApplied, &backupPath); err != nil {
			t.Fatalf("DecideRecipient falló: %v", err)
		}
		if err := repo.DecideRecipient(ctx, tenantID, students[0], advanceID, domain.AdvanceRecipientDismissed, nil); !errors.Is(err, domain.ErrAdvanceNotPending) {
			t.Fatalf("un avance ya decidido no puede cambiar de estado: %v", err)
		}
		if applied := findPending(t, mustPending(t, repo, tenantID, students[0]), advanceID); applied.ID != "" {
			t.Fatalf("un avance aplicado ya no está pendiente: %+v", applied)
		}
	})

	t.Run("el cierre fija la ventana automática y el ajuste exige el aula cerrada", func(t *testing.T) {
		graceUntil := time.Now().Add(15 * time.Minute)
		if _, err := repo.SetGrace(ctx, tenantID, teacherID, sessionID, graceUntil); err == nil {
			t.Fatal("SetGrace con el aula abierta debía fallar")
		}

		closed, err := sessions.CloseSession(ctx, tenantID, teacherID, sessionID)
		if err != nil {
			t.Fatalf("CloseSession falló: %v", err)
		}
		if closed.GraceUntil == nil {
			t.Fatal("cerrar el aula debía fijar grace_until")
		}
		window := time.Until(*closed.GraceUntil)
		if window < 29*time.Minute || window > 31*time.Minute {
			t.Fatalf("la ventana automática debía quedar en 30 minutos, quedó a %v", window)
		}

		adjusted, err := repo.SetGrace(ctx, tenantID, teacherID, sessionID, graceUntil)
		if err != nil {
			t.Fatalf("SetGrace con el aula cerrada falló: %v", err)
		}
		if adjusted.GraceUntil == nil {
			t.Fatal("grace_until debía quedar fijado tras el ajuste")
		}
	})

	t.Run("reabrir revive los vencidos y cerrar vence los pendientes", func(t *testing.T) {
		if _, err := repo.SetGrace(ctx, tenantID, teacherID, sessionID, time.Now().Add(30*time.Minute)); err != nil {
			t.Fatalf("no se pudo reabrir la ventana: %v", err)
		}

		mustExec(`UPDATE class_advance_recipients r
			SET status = 'expired', decided_at = NOW()
			FROM class_advances ca
			WHERE r.advance_id = ca.id AND ca.session_id = $1 AND r.status = 'pending'`, sessionID)

		if _, err := repo.SetGrace(ctx, tenantID, teacherID, sessionID, time.Now().Add(45*time.Minute)); err != nil {
			t.Fatalf("no se pudo reabrir la ventana con vencidos: %v", err)
		}
		var revived int
		if err := conn.GetContext(ctx, &revived, `
			SELECT COUNT(*) FROM class_advance_recipients r
			JOIN class_advances ca ON ca.id = r.advance_id
			WHERE ca.session_id = $1 AND r.status = 'pending'`, sessionID); err != nil {
			t.Fatalf("no se pudo contar los revividos: %v", err)
		}
		if revived == 0 {
			t.Fatal("reabrir la ventana debía devolver los vencidos a pendientes")
		}

		if _, err := repo.SetGrace(ctx, tenantID, teacherID, sessionID, time.Now()); err != nil {
			t.Fatalf("no se pudo cerrar la ventana: %v", err)
		}
		var expired int
		if err := conn.GetContext(ctx, &expired, `
			SELECT COUNT(*) FROM class_advance_recipients r
			JOIN class_advances ca ON ca.id = r.advance_id
			WHERE ca.session_id = $1 AND r.status = 'expired'`, sessionID); err != nil {
			t.Fatalf("no se pudo contar los vencidos: %v", err)
		}
		if expired == 0 {
			t.Fatal("cerrar la ventana debía vencer los pendientes")
		}
	})

	t.Run("el historial lee solo los movimientos CLASS_ de esa sesión", func(t *testing.T) {
		appliedMeta, _ := json.Marshal(map[string]any{
			"advance_id":    uuid.NewString(),
			"files_written": 2,
		})
		mustExec(`INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, resource_id, status_code, metadata, created_at)
			VALUES ($1, $2, $3, 'class_session', $4, 200, $5::jsonb, NOW() - INTERVAL '1 minute')`,
			tenantID, students[0], domain.AuditClassAdvanceApplied, sessionID, string(appliedMeta))

		extendedMeta, _ := json.Marshal(map[string]any{
			"minutes":        15,
			"previous_until": nil,
			"until":          time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		})
		mustExec(`INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, resource_id, status_code, metadata, created_at)
			VALUES ($1, $2, $3, 'class_session', $4, 200, $5::jsonb, NOW() - INTERVAL '2 minutes')`,
			tenantID, teacherID, domain.AuditClassAdvanceGraceExtended, sessionID, string(extendedMeta))

		// Petición HTTP genérica y movimiento de otra aula: ninguno entra al historial.
		mustExec(`INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, resource_id, status_code, metadata)
			VALUES ($1, $2, 'PATCH /api/v1/teacher/sessions/x/grace', 'class_session', $3, 200, '{}'::jsonb)`,
			tenantID, teacherID, sessionID)
		mustExec(`INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, resource_id, status_code, metadata)
			VALUES ($1, $2, $3, 'class_session', $4, 200, '{}'::jsonb)`,
			tenantID, teacherID, domain.AuditClassFrozen, uuid.NewString())

		events, err := repo.SessionHistory(ctx, tenantID, sessionID, 10)
		if err != nil {
			t.Fatalf("SessionHistory falló: %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("se esperaban dos movimientos del aula, hubo %d: %+v", len(events), events)
		}
		if events[0].Action != domain.AuditClassAdvanceApplied || events[1].Action != domain.AuditClassAdvanceGraceExtended {
			t.Fatalf("el historial debía venir del más nuevo al más viejo: %+v", events)
		}
		if events[0].ActorName == "" || events[0].FilesWritten != 2 || events[0].AdvanceID == nil {
			t.Errorf("el movimiento del alumno debía traer nombre y detalle: %+v", events[0])
		}
		if events[1].Minutes != 15 || events[1].Until == nil {
			t.Errorf("la suma debía traer minutos y nueva ventana: %+v", events[1])
		}
	})

	t.Run("el cierre por vencimiento fija la ventana desde la hora de fin", func(t *testing.T) {
		expiredID := uuid.NewString()
		mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
			VALUES ($1, $2, $3, $4, 'open', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '30 minutes')`,
			expiredID, tenantID, subjectID, exerciseID)

		expired, err := sessions.GetSession(ctx, tenantID, teacherID, expiredID)
		if err != nil {
			t.Fatalf("GetSession falló: %v", err)
		}
		if expired.Status != domain.ClassSessionStatusClosed {
			t.Fatalf("el aula vencida debía devolverse cerrada, quedó en %q", expired.Status)
		}
		if expired.GraceUntil == nil {
			t.Fatal("el cierre por vencimiento debía fijar grace_until")
		}
		window := time.Until(*expired.GraceUntil)
		if window < -time.Minute || window > time.Minute {
			t.Fatalf("la ventana debía vencer con el fin de la clase, quedó a %v", window)
		}
	})
}
