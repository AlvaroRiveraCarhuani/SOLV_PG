package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

const freezeEnvironmentID = "aaaaaaaa-1111-4111-8111-111111111111"

func newFreezePracticeEnvironmentFixture(instance *domain.PracticeEnvironment) (*ExamFreezeService, *fakeExamFreezeStore, *fakeMounts, *fakeExamAudit) {
	store := &fakeExamFreezeStore{}
	environments := &fakeExamPracticeEnvironments{byID: map[string]*domain.PracticeEnvironment{
		freezeEnvironmentID: instance,
	}}
	mounts := &fakeMounts{}
	audit := &fakeExamAudit{}
	service := NewExamFreezeService(store, &fakeExamSessions{}, environments, mounts).
		SetAuditRepository(audit).
		WithClock(func() time.Time { return time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC) })
	return service, store, mounts, audit
}

// La entrega anticipada congela solo el volumen del alumno: el montaje pasa a
// solo lectura, la fila queda marcada y el aula sigue abierta para los demás.
func TestFreezePracticeEnvironment_MountsReadOnlyAndMarksTheRow(t *testing.T) {
	service, store, mounts, audit := newFreezePracticeEnvironmentFixture(&domain.PracticeEnvironment{
		ID:        freezeEnvironmentID,
		TenantID:  domain.DefaultTenantID,
		StudentID: "student-a",
		SubjectID: "subject-1",
		Status:    domain.PracticeEnvironmentStatusRunning,
	})

	if err := service.FreezePracticeEnvironment(context.Background(), domain.DefaultTenantID, "student-a", freezeEnvironmentID); err != nil {
		t.Fatalf("error inesperado al congelar el volumen: %v", err)
	}

	if len(mounts.calls) != 1 || mounts.calls[0].environmentID != freezeEnvironmentID || !mounts.calls[0].readOnly {
		t.Fatalf("el volumen debía reconstruirse en solo lectura: %+v", mounts.calls)
	}
	if !store.frozenMarks[freezeEnvironmentID] {
		t.Fatalf("la fila debía quedar marcada como congelada: %+v", store.frozenMarks)
	}
	if len(audit.logs) != 1 {
		t.Fatalf("la entrega anticipada debía dejar una entrada de auditoría: %+v", audit.logs)
	}
	if audit.logs[0].ResourceType != "environment" || audit.logs[0].Action != domain.AuditExamPracticeEnvironmentFrozen {
		t.Fatalf("auditoría inesperada: %+v", audit.logs[0])
	}
}

// Reabrir y volver a entregar deja el volumen en el mismo estado: no se
// reconstruye dos veces.
func TestFreezePracticeEnvironment_SkipsAlreadyFrozen(t *testing.T) {
	frozenAt := time.Date(2026, 10, 31, 15, 30, 0, 0, time.UTC)
	service, store, mounts, _ := newFreezePracticeEnvironmentFixture(&domain.PracticeEnvironment{
		ID:        freezeEnvironmentID,
		TenantID:  domain.DefaultTenantID,
		StudentID: "student-a",
		SubjectID: "subject-1",
		Status:    domain.PracticeEnvironmentStatusFrozen,
		FrozenAt:  &frozenAt,
	})

	if err := service.FreezePracticeEnvironment(context.Background(), domain.DefaultTenantID, "student-a", freezeEnvironmentID); err != nil {
		t.Fatalf("repetir el cierre no debía fallar: %v", err)
	}
	if len(mounts.calls) != 0 {
		t.Fatalf("un volumen ya congelado no debía reconstruirse: %+v", mounts.calls)
	}
	if len(store.frozenMarks) != 0 {
		t.Fatalf("un volumen ya congelado no debía volver a marcarse: %+v", store.frozenMarks)
	}
}

// Un laboratorio que ya no existe no es un error interno: el cierre responde
// not-found para que el llamador no lo trate como una caída del servidor.
func TestFreezePracticeEnvironment_ReturnsNotFoundForVanishedPracticeEnvironment(t *testing.T) {
	service, store, mounts, _ := newFreezePracticeEnvironmentFixture(nil)

	err := service.FreezePracticeEnvironment(context.Background(), domain.DefaultTenantID, "student-a", "bbbbbbbb-2222-4222-8222-222222222222")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperado ErrNotFound, obtenido: %v", err)
	}
	if len(mounts.calls) != 0 || len(store.frozenMarks) != 0 {
		t.Fatal("un laboratorio inexistente no debía montarse ni marcarse")
	}
}

// Si el contenedor no se puede reconstruir, la fila no se marca: el estado no
// puede mentir sobre un volumen que sigue escribible.
func TestFreezePracticeEnvironment_ReportsMountFailureWithoutMarking(t *testing.T) {
	service, store, mounts, audit := newFreezePracticeEnvironmentFixture(&domain.PracticeEnvironment{
		ID:        freezeEnvironmentID,
		TenantID:  domain.DefaultTenantID,
		StudentID: "student-a",
		SubjectID: "subject-1",
		Status:    domain.PracticeEnvironmentStatusRunning,
	})
	mounts.err = errors.New("docker no responde")

	err := service.FreezePracticeEnvironment(context.Background(), domain.DefaultTenantID, "student-a", freezeEnvironmentID)
	if err == nil {
		t.Fatal("el fallo del montaje debía devolverse")
	}
	if len(store.frozenMarks) != 0 {
		t.Fatalf("sin montaje no se podía marcar el volumen: %+v", store.frozenMarks)
	}
	if len(audit.logs) != 0 {
		t.Fatalf("un cierre fallido no debía auditarse como hecho: %+v", audit.logs)
	}
}
