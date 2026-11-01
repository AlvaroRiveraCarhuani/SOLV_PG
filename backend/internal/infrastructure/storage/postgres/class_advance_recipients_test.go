package postgres

import (
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestGroupSessionAdvances_CountsRecipientsByStatus(t *testing.T) {
	created := time.Now()
	decided := created.Add(time.Minute)
	rows := []advanceRecipientRow{
		{AdvanceID: "avance-2", ExerciseID: "ej-2", ExerciseTitle: "Recorridos", Scope: "class", FileCount: 1, CreatedAt: created,
			StudentID: "s1", Name: "Ana", Email: "ana@solv.test", Status: domain.AdvanceRecipientApplied, DecidedAt: &decided},
		{AdvanceID: "avance-2", ExerciseID: "ej-2", ExerciseTitle: "Recorridos", Scope: "class", FileCount: 1, CreatedAt: created,
			StudentID: "s2", Name: "Beto", Email: "beto@solv.test", Status: domain.AdvanceRecipientPending},
		{AdvanceID: "avance-2", ExerciseID: "ej-2", ExerciseTitle: "Recorridos", Scope: "class", FileCount: 1, CreatedAt: created,
			StudentID: "s3", Name: "Caro", Email: "caro@solv.test", Status: domain.AdvanceRecipientDismissed, DecidedAt: &decided},
		{AdvanceID: "avance-2", ExerciseID: "ej-2", ExerciseTitle: "Recorridos", Scope: "class", FileCount: 1, CreatedAt: created,
			StudentID: "s4", Name: "Dani", Email: "dani@solv.test", Status: domain.AdvanceRecipientExpired},
		{AdvanceID: "avance-1", ExerciseID: "ej-1", ExerciseTitle: "Pilas", Scope: "selected", FileCount: 3, CreatedAt: created.Add(-time.Hour),
			StudentID: "s1", Name: "Ana", Email: "ana@solv.test", Status: domain.AdvanceRecipientFailed},
	}

	advances := groupSessionAdvances(rows)
	if len(advances) != 2 {
		t.Fatalf("esperadas 2 emisiones, hubo %d", len(advances))
	}

	first := advances[0]
	if first.ID != "avance-2" || first.ExerciseTitle != "Recorridos" {
		t.Fatalf("el orden debía conservar la emisión más nueva primero: %+v", first)
	}
	if first.Counts.Total != 4 || first.Counts.Applied != 1 || first.Counts.Dismissed != 1 ||
		first.Counts.Pending != 1 || first.Counts.Expired != 1 || first.Counts.Failed != 0 {
		t.Errorf("conteos inesperados: %+v", first.Counts)
	}
	if len(first.Recipients) != 4 {
		t.Fatalf("esperados 4 destinatarios, hubo %d", len(first.Recipients))
	}
	if first.Recipients[0].Name != "Ana" || first.Recipients[0].DecidedAt == nil {
		t.Errorf("el destinatario debía llegar con nombre y hora de decisión: %+v", first.Recipients[0])
	}
	if first.Recipients[1].DecidedAt != nil {
		t.Errorf("un destinatario pendiente no tiene hora de decisión: %+v", first.Recipients[1])
	}

	second := advances[1]
	if second.ID != "avance-1" || second.Scope != domain.AdvanceScopeSelected || second.Counts.Failed != 1 {
		t.Errorf("la segunda emisión quedó mal armada: %+v", second)
	}
}

func TestGroupSessionAdvances_EmptyComplementsWithoutAdvances(t *testing.T) {
	if got := groupSessionAdvances(nil); len(got) != 0 {
		t.Fatalf("sin filas no hay emisiones: %+v", got)
	}
}
