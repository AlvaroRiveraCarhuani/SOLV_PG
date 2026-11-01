package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func newPreviewService(t *testing.T, delivery *domain.AdvanceDelivery, snapshots []domain.WorkspaceVolumeFile, existing []domain.WorkspaceVolumeFile) (*services.ClassAdvanceService, *fakeAdvanceStore, *fakeAdvanceVolumes) {
	t.Helper()
	store := &fakeAdvanceStore{delivery: delivery}
	volumes := &fakeAdvanceVolumes{existing: existing}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: snapshots}, newRunningWorkspace(t), volumes)
	return svc, store, volumes
}

func TestPreviewAdvance_SeparatesAddedFromOverwrittenWithoutWriting(t *testing.T) {
	svc, _, volumes := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil),
		[]domain.WorkspaceVolumeFile{
			{Path: "utils.py", Content: []byte("nuevo\n")},
			{Path: "main.py", Content: []byte("nuevo\n")},
		},
		[]domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("del alumno\n")},
		},
	)

	preview, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if err != nil {
		t.Fatalf("error inesperado al previsualizar: %v", err)
	}

	if got, want := preview.Added, []string{"utils.py"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("archivos agregados = %v, se esperaba %v", got, want)
	}
	if got, want := preview.Overwritten, []string{"main.py"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("archivos reemplazados = %v, se esperaba %v", got, want)
	}
	if preview.FileCount != 2 {
		t.Errorf("la confirmación debía anunciar 2 archivos, anunció %d", preview.FileCount)
	}
	if len(volumes.writes) != 0 {
		t.Fatalf("previsualizar no escribe en el volumen: %v", volumes.writes)
	}
}

func TestPreviewAdvance_ReadsTheSnapshotOfTheAdvance(t *testing.T) {
	// El snapshot es la única fuente: un archivo que no está en él no aparece
	// aunque el alumno lo tenga.
	svc, _, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil),
		[]domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		[]domain.WorkspaceVolumeFile{{Path: "apuntes.txt", Content: []byte("mío\n")}},
	)

	preview, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if err != nil {
		t.Fatalf("error inesperado al previsualizar: %v", err)
	}
	if len(preview.Added) != 1 || preview.Added[0] != "main.py" {
		t.Errorf("main.py debía contarse como agregado: %v", preview.Added)
	}
	if len(preview.Overwritten) != 0 {
		t.Errorf("un archivo que el avance no trae no se reemplaza: %v", preview.Overwritten)
	}
}

func TestPreviewAdvance_RejectsExam(t *testing.T) {
	svc, _, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "exam", domain.ClassSessionStatusOpen, nil),
		[]domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		nil,
	)

	_, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceExamBlocked) {
		t.Fatalf("esperado avance bloqueado por examen, obtenido: %v", err)
	}
}

func TestPreviewAdvance_RejectsDecidedAdvance(t *testing.T) {
	svc, _, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientApplied, "class", domain.ClassSessionStatusOpen, nil),
		[]domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		nil,
	)

	_, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceNotPending) {
		t.Fatalf("esperado aviso ya decidido, obtenido: %v", err)
	}
}

func TestPreviewAdvance_ClosesAnExpiredNotice(t *testing.T) {
	expired := time.Now().Add(-time.Minute)
	svc, store, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusClosed, &expired),
		[]domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		nil,
	)

	_, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceGraceExpired) {
		t.Fatalf("esperado ventana vencida, obtenido: %v", err)
	}
	if len(store.decisions) != 1 || store.decisions[0].status != domain.AdvanceRecipientExpired {
		t.Fatalf("el aviso vencido debía cerrarse como expired: %+v", store.decisions)
	}
}

func TestPreviewAdvance_AllowsTheGraceWindowOfAClosedAula(t *testing.T) {
	grace := time.Now().Add(10 * time.Minute)
	svc, _, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusClosed, &grace),
		[]domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		nil,
	)

	if _, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID); err != nil {
		t.Fatalf("con la ventana activa el aviso se puede ver: %v", err)
	}
}

func TestPendingAdvances_HidesAndClosesExpiredNotices(t *testing.T) {
	expired := time.Now().Add(-time.Minute)
	svc, store, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusClosed, &expired),
		nil,
		nil,
	)

	deliveries, err := svc.PendingAdvances(context.Background(), advanceTenantID, advanceStudentID)
	if err != nil {
		t.Fatalf("error inesperado al consultar avisos: %v", err)
	}
	if len(deliveries) != 0 {
		t.Fatalf("un aviso vencido no se muestra en el laboratorio: %v", deliveries)
	}
	if len(store.decisions) != 1 || store.decisions[0].status != domain.AdvanceRecipientExpired {
		t.Fatalf("el aviso vencido debía cerrarse: %+v", store.decisions)
	}
}

func TestPendingAdvances_KeepsNoticesInsideTheWindow(t *testing.T) {
	grace := time.Now().Add(5 * time.Minute)
	svc, store, _ := newPreviewService(t,
		newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusClosed, &grace),
		nil,
		nil,
	)

	deliveries, err := svc.PendingAdvances(context.Background(), advanceTenantID, advanceStudentID)
	if err != nil {
		t.Fatalf("error inesperado al consultar avisos: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("un aviso dentro de la ventana sigue visible, hubo %d", len(deliveries))
	}
	if len(store.decisions) != 0 {
		t.Fatalf("un aviso vigente no se decide: %+v", store.decisions)
	}
}

// El avance con práctica se lee contra el volumen de esa práctica: desde
// ADR-059 el alumno abre su laboratorio por ejercicio y el entorno por materia
// ya no es el que tiene su código.
func TestPreviewAdvance_PrefersTheStudentPracticeWorkspace(t *testing.T) {
	delivery := newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)
	delivery.ExerciseID = "11111111-1111-1111-1111-111111111111"

	repo := newPracticeWorkspaceRepo()
	practice := &domain.WorkspaceInstance{
		ID:         "66666666-6666-6666-6666-666666666667",
		StudentID:  advanceStudentID,
		SubjectID:  advanceSubjectID,
		ExerciseID: &delivery.ExerciseID,
		Status:     domain.WorkspaceStatusRunning,
	}
	if err := repo.Create(context.Background(), practice); err != nil {
		t.Fatalf("no se pudo preparar el entorno de la práctica: %v", err)
	}

	volumes := &fakeAdvanceVolumes{existing: []domain.WorkspaceVolumeFile{
		{Path: "main.py", Content: []byte("del alumno\n")},
	}}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{delivery: delivery}, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
		}}, repo, volumes)

	if _, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID); err != nil {
		t.Fatalf("error inesperado al previsualizar: %v", err)
	}
	if len(volumes.snapshots) == 0 || volumes.snapshots[0] != practice.VolumeName() {
		t.Fatalf("se esperaba leer el volumen de la práctica %q, se leyó %v", practice.VolumeName(), volumes.snapshots)
	}
}
