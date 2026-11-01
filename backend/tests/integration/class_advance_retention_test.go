package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestClassAdvanceRepository_Retencion comprueba el adaptador Postgres del
// barrido: solo los avances de aulas cerradas hace más de la retención son
// candidatos, la marca de purgado los saca de la lista y las claves siguen
// disponibles para no borrar objetos todavía referenciados.
func TestClassAdvanceRepository_Retencion(t *testing.T) {
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
	oldSessionID := uuid.NewString()
	recentSessionID := uuid.NewString()
	sevenDaySessionID := uuid.NewString()
	nineDaySessionID := uuid.NewString()
	openSessionID := uuid.NewString()
	futureGraceSessionID := uuid.NewString()

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
	// El aula se borra al final y arrastra avances y destinatarios.
	t.Cleanup(func() {
		_, _ = conn.ExecContext(context.Background(),
			"DELETE FROM class_sessions WHERE id IN ($1, $2, $3, $4, $5, $6)",
			oldSessionID, recentSessionID, sevenDaySessionID, nineDaySessionID, openSessionID, futureGraceSessionID)
	})

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Retención', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_retencion_%s@uab.edu.bo", teacherID[:8]), tenantID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Retención de Prueba', $3, $4)`,
		subjectID, tenantID, "RET-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Práctica con retención', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		exerciseID, tenantID, subjectID, templateID)

	// Un aula cerrada hace 30 días ya no puede reabrirse; otra de ayer sigue
	// dentro de la retención y conserva su objeto.
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days')`,
		oldSessionID, tenantID, subjectID, exerciseID)
	// El borde exacto de la política: a los 7 días todavía se conserva y a los
	// 9 ya está fuera de la retención de 8.
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days')`,
		sevenDaySessionID, tenantID, subjectID, exerciseID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '9 days', NOW() - INTERVAL '9 days', NOW() - INTERVAL '9 days')`,
		nineDaySessionID, tenantID, subjectID, exerciseID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day')`,
		recentSessionID, tenantID, subjectID, exerciseID)
	// Un aula abierta no se toca aunque haya quedado vieja, y una ventana
	// todavía vigente tampoco: el objeto tiene que sobrevivir a la reapertura.
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days')`,
		openSessionID, tenantID, subjectID, exerciseID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at, closes_at, grace_until)
		VALUES ($1, $2, $3, $4, 'closed', NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days', NOW() + INTERVAL '1 day')`,
		futureGraceSessionID, tenantID, subjectID, exerciseID)

	oldAdvanceID := uuid.NewString()
	recentAdvanceID := uuid.NewString()
	sevenDayAdvanceID := uuid.NewString()
	nineDayAdvanceID := uuid.NewString()
	openAdvanceID := uuid.NewString()
	futureGraceAdvanceID := uuid.NewString()
	oldKey := "class-advances/" + tenantID + "/" + oldAdvanceID + ".tar.gz"
	recentKey := "class-advances/" + tenantID + "/" + recentAdvanceID + ".tar.gz"
	sevenDayKey := "class-advances/" + tenantID + "/" + sevenDayAdvanceID + ".tar.gz"
	nineDayKey := "class-advances/" + tenantID + "/" + nineDayAdvanceID + ".tar.gz"
	openKey := "class-advances/" + tenantID + "/" + openAdvanceID + ".tar.gz"
	futureGraceKey := "class-advances/" + tenantID + "/" + futureGraceAdvanceID + ".tar.gz"
	insertAdvance := func(id, sessionID, key string) {
		mustExec(`INSERT INTO class_advances (
				id, tenant_id, session_id, exercise_id, scope,
				snapshot_key, snapshot_sha256, snapshot_size_bytes, file_count, created_by
			) VALUES ($1, $2, $3, $4, 'class', $5, $6, 512, 2, $7)`,
			id, tenantID, sessionID, exerciseID, key, strings.Repeat("b", 64), teacherID)
	}
	insertAdvance(oldAdvanceID, oldSessionID, oldKey)
	insertAdvance(recentAdvanceID, recentSessionID, recentKey)
	insertAdvance(sevenDayAdvanceID, sevenDaySessionID, sevenDayKey)
	insertAdvance(nineDayAdvanceID, nineDaySessionID, nineDayKey)
	insertAdvance(openAdvanceID, openSessionID, openKey)
	insertAdvance(futureGraceAdvanceID, futureGraceSessionID, futureGraceKey)

	sessions := postgres.NewPostgresClassSessionRepository(conn)
	repo := postgres.NewPostgresClassAdvanceRepository(conn, sessions)
	cutoff := time.Now().Add(-time.Duration(domain.AdvanceSnapshotRetentionDays) * 24 * time.Hour)

	candidates, err := repo.ExpiredAdvanceSnapshots(ctx, cutoff, 500)
	if err != nil {
		t.Fatalf("ExpiredAdvanceSnapshots falló: %v", err)
	}
	byID := make(map[string]domain.ClassAdvance, len(candidates))
	for _, advance := range candidates {
		byID[advance.ID] = advance
	}
	if _, ok := byID[oldAdvanceID]; !ok {
		t.Fatalf("el avance de un aula cerrada hace 30 días debía ser candidato: %+v", candidates)
	}
	if _, ok := byID[recentAdvanceID]; ok {
		t.Fatalf("el avance de ayer no debía borrarse todavía: %+v", byID[recentAdvanceID])
	}
	if _, ok := byID[sevenDayAdvanceID]; ok {
		t.Fatalf("el avance de hace 7 días debía conservarse: %+v", byID[sevenDayAdvanceID])
	}
	if _, ok := byID[nineDayAdvanceID]; !ok {
		t.Fatalf("el avance de hace 9 días debía ser candidato: %+v", candidates)
	}
	if _, ok := byID[openAdvanceID]; ok {
		t.Fatalf("un avance de un aula abierta nunca debía ser candidato: %+v", byID[openAdvanceID])
	}
	if _, ok := byID[futureGraceAdvanceID]; ok {
		t.Fatalf("un avance con la ventana todavía vigente no debía ser candidato: %+v", byID[futureGraceAdvanceID])
	}

	if err := repo.MarkSnapshotPurged(ctx, tenantID, oldAdvanceID, time.Now()); err != nil {
		t.Fatalf("MarkSnapshotPurged falló: %v", err)
	}
	var purged *time.Time
	if err := conn.GetContext(ctx, &purged,
		`SELECT snapshot_purged_at FROM class_advances WHERE id = $1`, oldAdvanceID); err != nil {
		t.Fatalf("no se pudo leer la marca de purgado: %v", err)
	}
	if purged == nil {
		t.Fatal("snapshot_purged_at debía quedar fijado")
	}

	again, err := repo.ExpiredAdvanceSnapshots(ctx, cutoff, 500)
	if err != nil {
		t.Fatalf("segundo ExpiredAdvanceSnapshots falló: %v", err)
	}
	for _, advance := range again {
		if advance.ID == oldAdvanceID {
			t.Fatalf("un avance ya purgado no debía volver a la lista")
		}
	}

	keys, err := repo.AdvanceSnapshotKeys(ctx)
	if err != nil {
		t.Fatalf("AdvanceSnapshotKeys falló: %v", err)
	}
	referenced := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		referenced[key] = struct{}{}
	}
	if _, ok := referenced[oldKey]; !ok {
		t.Fatalf("la clave del avance purgado seguía registrada en la fila: %v", keys)
	}
	if _, ok := referenced[recentKey]; !ok {
		t.Fatalf("la clave del avance reciente debía estar referenciada: %v", keys)
	}
}
