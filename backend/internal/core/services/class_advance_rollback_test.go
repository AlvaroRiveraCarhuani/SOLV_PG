package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// rollbackAudit guarda los movimientos que el servicio registra al deshacer.
type rollbackAudit struct {
	logs []*domain.AuditLog
}

func (a *rollbackAudit) Create(_ context.Context, log *domain.AuditLog) error {
	a.logs = append(a.logs, log)
	return nil
}

func (a *rollbackAudit) action(action string) *domain.AuditLog {
	for _, log := range a.logs {
		if log.Action == action {
			return log
		}
	}
	return nil
}

// pruningVolumes agrega la poda del respaldo al doble de volúmenes.
type pruningVolumes struct {
	fakeAdvanceVolumes
	deleted [][]string
}

func (v *pruningVolumes) ListWorkspaceBackupDirs(context.Context, string, string) ([]string, error) {
	return nil, nil
}

func (v *pruningVolumes) DeleteWorkspaceVolumePaths(_ context.Context, _ string, paths []string) error {
	v.deleted = append(v.deleted, paths)
	return nil
}

func previewService(t *testing.T, store *fakeAdvanceStore, snapshots map[string]string, current []domain.WorkspaceVolumeFile) (*services.ClassAdvanceService, *fakeAdvanceVolumes) {
	t.Helper()
	volumes := &fakeAdvanceVolumes{existing: current}
	files := make([]domain.WorkspaceVolumeFile, 0, len(snapshots))
	for path, content := range snapshots {
		files = append(files, domain.WorkspaceVolumeFile{Path: path, Content: []byte(content)})
	}
	return services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: files}, newRunningWorkspace(t), volumes), volumes
}

func TestPreviewAdvance_ReturnsFileContents(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "practice", domain.ClassSessionStatusOpen, nil)}
	svc, _ := previewService(t, store,
		map[string]string{"main.py": "nuevo\n", "utils.py": "helper\n"},
		[]domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("viejo\n")}},
	)

	preview, err := svc.Preview(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if err != nil {
		t.Fatalf("error inesperado al previsualizar: %v", err)
	}
	if preview.Files["main.py"] != "nuevo\n" || preview.Files["utils.py"] != "helper\n" {
		t.Fatalf("el preview debe traer el contenido de cada archivo, obtenido: %v", preview.Files)
	}
	if len(preview.Overwritten) != 1 || preview.Overwritten[0] != "main.py" {
		t.Errorf("esperado main.py sobrescrito, obtenido: %v", preview.Overwritten)
	}
	if len(preview.Added) != 1 || preview.Added[0] != "utils.py" {
		t.Errorf("esperado utils.py agregado, obtenido: %v", preview.Added)
	}
}

func TestApplyAdvance_SelectedFilesOnly(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "practice", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{existing: []domain.WorkspaceVolumeFile{
		{Path: "main.py", Content: []byte("viejo\n")},
		{Path: "utils.py", Content: []byte("viejo helper\n")},
	}}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
			{Path: "utils.py", Content: []byte("helper\n")},
			{Path: "docs/readme.md", Content: []byte("# guía\n")},
		}}, newRunningWorkspace(t), volumes)

	result, err := svc.ApplySelection(context.Background(), advanceTenantID, advanceStudentID, advanceID, []string{"main.py"})
	if err != nil {
		t.Fatalf("error inesperado al aplicar la selección: %v", err)
	}

	if len(volumes.writes) != 2 {
		t.Fatalf("esperado un respaldo y una copia, obtenidas %d escrituras", len(volumes.writes))
	}
	backup := volumes.writes[0]
	if len(backup) != 1 || backup[".solv_backups/"+advanceID+"/main.py"] != "viejo\n" {
		t.Fatalf("el respaldo debe incluir solo el archivo elegido, obtenido: %v", backup)
	}
	copied := volumes.writes[1]
	if len(copied) != 1 || copied["main.py"] != "nuevo\n" {
		t.Fatalf("la copia debe incluir solo el archivo elegido, obtenida: %v", copied)
	}
	if result.FilesWritten != 1 {
		t.Errorf("esperado 1 archivo escrito, obtenidos %d", result.FilesWritten)
	}
	if len(result.Overwritten) != 1 || result.Overwritten[0] != "main.py" || len(result.Added) != 0 {
		t.Errorf("resultado inesperado: %+v", result)
	}
}

func TestApplyAdvance_EmptySelectionAppliesEverything(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "practice", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
			{Path: "utils.py", Content: []byte("helper\n")},
		}}, newRunningWorkspace(t), volumes)

	result, err := svc.ApplySelection(context.Background(), advanceTenantID, advanceStudentID, advanceID, nil)
	if err != nil {
		t.Fatalf("error inesperado al aplicar sin selección: %v", err)
	}
	if result.FilesWritten != 2 {
		t.Fatalf("sin selección se aplica el avance completo, se escribieron %d archivos", result.FilesWritten)
	}
	if len(volumes.writes) != 1 {
		t.Fatalf("sin archivos previos no hay respaldo, hubo %d escrituras", len(volumes.writes))
	}
}

func TestApplyAdvance_RejectsFilesOutsideSnapshot(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "practice", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
		}}, newRunningWorkspace(t), volumes)

	_, err := svc.ApplySelection(context.Background(), advanceTenantID, advanceStudentID, advanceID,
		[]string{"main.py", "../../etc/passwd"})
	if !errors.Is(err, domain.ErrAdvanceSelectionInvalid) {
		t.Fatalf("esperado error de selección inválida, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 {
		t.Error("una selección inválida no debe tocar el volumen")
	}
	if len(store.decisions) != 0 {
		t.Error("una selección inválida no debe decidir el destinatario")
	}
}

func TestRollbackAdvance_RestoresBackupAndReopensRecipient(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientApplied, "practice", domain.ClassSessionStatusOpen, nil)}
	audit := &rollbackAudit{}
	volumes := &pruningVolumes{fakeAdvanceVolumes: fakeAdvanceVolumes{existing: []domain.WorkspaceVolumeFile{
		{Path: "main.py", Content: []byte("avance\n")},
		{Path: ".solv_backups/" + advanceID + "/main.py", Content: []byte("version previa\n")},
	}}}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, newRunningWorkspace(t), volumes)
	svc.SetAuditRepository(audit)

	result, err := svc.Rollback(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if err != nil {
		t.Fatalf("error inesperado al deshacer: %v", err)
	}

	if len(volumes.writes) != 1 || volumes.writes[0]["main.py"] != "version previa\n" {
		t.Fatalf("el rollback debe restaurar el contenido respaldado, obtenido: %v", volumes.writes)
	}
	if result.FilesRestored != 1 || result.Restored[0] != "main.py" {
		t.Fatalf("resultado inesperado del rollback: %+v", result)
	}
	if len(store.reopened) != 1 || store.reopened[0] != advanceID {
		t.Fatalf("el destinatario debe volver a pendiente, obtenido: %v", store.reopened)
	}
	if len(volumes.deleted) != 1 || volumes.deleted[0][0] != ".solv_backups/"+advanceID {
		t.Fatalf("el respaldo restaurado debe borrarse, obtenido: %v", volumes.deleted)
	}
	if audit.action(domain.AuditClassAdvanceRollback) == nil {
		t.Fatalf("el rollback debe quedar en auditoría, obtenido: %v", audit.logs)
	}

	// Ya aplicado de nuevo el aviso queda pendiente: deshacer otra vez no
	// encuentra nada que restaurar y no toca el volumen.
	before := len(volumes.writes)
	if _, err := svc.Rollback(context.Background(), advanceTenantID, advanceStudentID, advanceID); !errors.Is(err, domain.ErrAdvanceRollbackNotApplied) {
		t.Fatalf("esperado error de avance no aplicado, obtenido: %v", err)
	}
	if len(volumes.writes) != before {
		t.Error("un segundo rollback no debe escribir en el volumen")
	}
}

func TestRollbackAdvance_RequiresAppliedRecipient(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "practice", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, newRunningWorkspace(t), volumes)

	_, err := svc.Rollback(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceRollbackNotApplied) {
		t.Fatalf("esperado error de avance no aplicado, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 || len(store.reopened) != 0 {
		t.Error("un aviso pendiente no se deshace")
	}
}

func TestRollbackAdvance_WithoutBackupReportsMissing(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientApplied, "practice", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{existing: []domain.WorkspaceVolumeFile{
		{Path: "main.py", Content: []byte("avance\n")},
	}}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, newRunningWorkspace(t), volumes)

	_, err := svc.Rollback(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceBackupMissing) {
		t.Fatalf("esperado error de respaldo faltante, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 || len(store.reopened) != 0 {
		t.Error("sin respaldo no se escribe ni se reabre el aviso")
	}
}
