package services_test

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

const (
	advanceTenantID  = "11111111-1111-1111-1111-111111111111"
	advanceStudentID = "22222222-2222-2222-2222-222222222222"
	advanceID        = "33333333-3333-3333-3333-333333333333"
	advanceSubjectID = "44444444-4444-4444-4444-444444444444"
)

type advanceDecision struct {
	status     string
	backupPath *string
}

type fakeAdvanceStore struct {
	delivery        *domain.AdvanceDelivery
	decisions       []advanceDecision
	reopened        []string
	reopenErr       error
	decideErr       error
	participants    []string
	purpose         string
	created         *domain.ClassAdvance
	sessionAdvances []domain.AdvanceWithRecipients
}

func (f *fakeAdvanceStore) CreateAdvance(ctx context.Context, advance domain.ClassAdvance, studentIDs []string) error {
	f.created = &advance
	return nil
}

func (f *fakeAdvanceStore) SessionParticipants(ctx context.Context, tenantID, sessionID string) ([]string, error) {
	return f.participants, nil
}

func (f *fakeAdvanceStore) ExercisePurpose(ctx context.Context, tenantID, exerciseID string) (string, error) {
	return f.purpose, nil
}

func (f *fakeAdvanceStore) SetFrozen(ctx context.Context, tenantID, teacherID, sessionID string, frozen bool) (*domain.ClassSession, error) {
	return nil, nil
}

func (f *fakeAdvanceStore) SetGrace(ctx context.Context, tenantID, teacherID, sessionID string, graceUntil time.Time) (*domain.ClassSession, error) {
	return nil, nil
}

func (f *fakeAdvanceStore) PendingAdvances(ctx context.Context, tenantID, studentID string) ([]domain.AdvanceDelivery, error) {
	if f.delivery == nil {
		return nil, nil
	}
	return []domain.AdvanceDelivery{*f.delivery}, nil
}

func (f *fakeAdvanceStore) AdvanceForStudent(ctx context.Context, tenantID, studentID, advanceID string) (*domain.AdvanceDelivery, error) {
	if f.delivery == nil {
		return nil, domain.ErrAdvanceNotFound
	}
	return f.delivery, nil
}

func (f *fakeAdvanceStore) DecideRecipient(ctx context.Context, tenantID, studentID, advanceID, status string, backupPath *string) error {
	if f.decideErr != nil {
		return f.decideErr
	}
	f.decisions = append(f.decisions, advanceDecision{status: status, backupPath: backupPath})
	return nil
}

func (f *fakeAdvanceStore) SessionAdvances(ctx context.Context, tenantID, teacherID, sessionID string) ([]domain.AdvanceWithRecipients, error) {
	return f.sessionAdvances, nil
}

func (f *fakeAdvanceStore) ReopenRecipient(ctx context.Context, tenantID, studentID, advanceID string) error {
	if f.reopenErr != nil {
		return f.reopenErr
	}
	f.reopened = append(f.reopened, advanceID)
	if f.delivery != nil {
		f.delivery.Status = domain.AdvanceRecipientPending
	}
	return nil
}

type fakeAdvanceSnapshotsFiles struct {
	files    []domain.WorkspaceVolumeFile
	err      error
	putKey   string
	putFiles map[string]string
	putErr   error
	deleted  []string
}

func (f *fakeAdvanceSnapshotsFiles) Put(ctx context.Context, key string, body io.Reader) (string, int64, error) {
	return "", 0, nil
}

func (f *fakeAdvanceSnapshotsFiles) PutFiles(ctx context.Context, key string, files map[string]string) (string, int64, int, error) {
	if f.putErr != nil {
		return "", 0, 0, f.putErr
	}
	f.putKey = key
	f.putFiles = files
	return "hash", 128, len(files), nil
}

func (f *fakeAdvanceSnapshotsFiles) Files(ctx context.Context, key, expectedSHA256 string, maxFiles int) ([]domain.WorkspaceVolumeFile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.files, nil
}

func (f *fakeAdvanceSnapshotsFiles) Delete(ctx context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

type fakeAdvanceVolumes struct {
	existing []domain.WorkspaceVolumeFile
	writes   []map[string]string
	// snapshots conserva los volúmenes leídos para comprobar cuál se usó.
	snapshots []string
	err       error
}

func (f *fakeAdvanceVolumes) SnapshotWorkspaceVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.WorkspaceVolumeFile, error) {
	f.snapshots = append(f.snapshots, volumeName)
	return f.existing, nil
}

func (f *fakeAdvanceVolumes) WriteWorkspaceVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	if f.err != nil {
		return f.err
	}
	f.writes = append(f.writes, files)
	return nil
}

func newAdvanceDelivery(status, purpose, sessionStatus string, graceUntil *time.Time) *domain.AdvanceDelivery {
	return &domain.AdvanceDelivery{
		ClassAdvance: domain.ClassAdvance{
			ID:             advanceID,
			TenantID:       advanceTenantID,
			SessionID:      "55555555-5555-5555-5555-555555555555",
			SnapshotKey:    "class-advances/" + advanceTenantID + "/" + advanceID + ".tar.gz",
			SnapshotSHA256: "hash",
		},
		Status:        status,
		SubjectID:     advanceSubjectID,
		SessionStatus: sessionStatus,
		GraceUntil:    graceUntil,
		Purpose:       purpose,
	}
}

func newRunningWorkspace(t *testing.T) *mockWorkspaceRepo {
	t.Helper()
	repo := newMockWorkspaceRepo()
	if err := repo.Create(context.Background(), &domain.WorkspaceInstance{
		ID:        "66666666-6666-6666-6666-666666666666",
		StudentID: advanceStudentID,
		SubjectID: advanceSubjectID,
		Status:    domain.WorkspaceStatusRunning,
	}); err != nil {
		t.Fatalf("no se pudo preparar el workspace: %v", err)
	}
	return repo
}

func TestApplyAdvance_BacksUpOverwrittenFilesAndWritesSnapshot(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{existing: []domain.WorkspaceVolumeFile{
		{Path: "main.py", Content: []byte("viejo\n")},
	}}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
			{Path: "utils.py", Content: []byte("helper\n")},
		}}, newRunningWorkspace(t), volumes)

	result, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if err != nil {
		t.Fatalf("error inesperado al aplicar: %v", err)
	}

	if len(store.decisions) != 1 || store.decisions[0].status != domain.AdvanceRecipientApplied {
		t.Fatalf("esperado destinatario aplicado, obtenido: %+v", store.decisions)
	}
	if len(volumes.writes) != 2 {
		t.Fatalf("esperado un respaldo y una copia, obtenidas %d escrituras", len(volumes.writes))
	}

	backup := volumes.writes[0]
	backupContent, ok := backup[".solv_backups/"+advanceID+"/main.py"]
	if !ok {
		t.Fatalf("esperado respaldo de main.py, obtenido: %v", backup)
	}
	if backupContent != "viejo\n" {
		t.Errorf("el respaldo debe guardar el contenido previo del alumno, obtenido: %q", backupContent)
	}
	if _, ok := backup[".solv_backups/"+advanceID+"/utils.py"]; ok {
		t.Error("un archivo nuevo no se respalda")
	}

	copied := volumes.writes[1]
	if copied["main.py"] != "nuevo\n" || copied["utils.py"] != "helper\n" {
		t.Errorf("esperada la copia del snapshot, obtenida: %v", copied)
	}

	if result.FilesWritten != 2 {
		t.Errorf("esperados 2 archivos escritos, obtenidos %d", result.FilesWritten)
	}
	if len(result.Overwritten) != 1 || result.Overwritten[0] != "main.py" {
		t.Errorf("esperado main.py sobrescrito, obtenido: %v", result.Overwritten)
	}
	if len(result.Added) != 1 || result.Added[0] != "utils.py" {
		t.Errorf("esperado utils.py agregado, obtenido: %v", result.Added)
	}
	if result.BackupPath == nil || *result.BackupPath != ".solv_backups/"+advanceID {
		t.Errorf("esperada la ruta del respaldo, obtenida: %v", result.BackupPath)
	}
	if store.decisions[0].backupPath == nil {
		t.Error("el destinatario debe registrar la ruta del respaldo")
	}
}

func TestApplyAdvance_ExpiresWhenGraceWindowIsOver(t *testing.T) {
	grace := time.Now().Add(-time.Minute)
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusClosed, &grace)}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, newRunningWorkspace(t), volumes)

	_, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceGraceExpired) {
		t.Fatalf("esperado error de ventana vencida, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 {
		t.Error("una ventana vencida no debe tocar el volumen")
	}
	if len(store.decisions) != 1 || store.decisions[0].status != domain.AdvanceRecipientExpired {
		t.Errorf("esperado destinatario expirado, obtenido: %+v", store.decisions)
	}
}

func TestApplyAdvance_RequiresRunningWorkspace(t *testing.T) {
	repo := newMockWorkspaceRepo()
	if err := repo.Create(context.Background(), &domain.WorkspaceInstance{
		ID:        "66666666-6666-6666-6666-666666666666",
		StudentID: advanceStudentID,
		SubjectID: advanceSubjectID,
		Status:    domain.WorkspaceStatusHibernated,
	}); err != nil {
		t.Fatalf("no se pudo preparar el workspace: %v", err)
	}
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, repo, volumes)

	_, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceWorkspaceStopped) {
		t.Fatalf("esperado error de laboratorio detenido, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 {
		t.Error("sin laboratorio en ejecución no se escribe nada")
	}
	if len(store.decisions) != 0 {
		t.Error("el aviso debe seguir pendiente hasta abrir el laboratorio")
	}
}

// gatedAdvanceSnapshots bloquea cada lectura del snapshot hasta liberarla, para
// observar cuántas aplicaciones están dentro del semáforo a la vez.
type gatedAdvanceSnapshots struct {
	fakeAdvanceSnapshotsFiles
	calls   int32
	entered chan struct{}
	release chan struct{}
}

func (g *gatedAdvanceSnapshots) Files(ctx context.Context, key, expectedSHA256 string, maxFiles int) ([]domain.WorkspaceVolumeFile, error) {
	atomic.AddInt32(&g.calls, 1)
	g.entered <- struct{}{}
	<-g.release
	return g.files, nil
}

func TestApplyAdvance_LimitsConcurrentApplications(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)}
	gated := &gatedAdvanceSnapshots{
		fakeAdvanceSnapshotsFiles: fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}}},
		entered:                   make(chan struct{}, 2),
		release:                   make(chan struct{}),
	}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(gated, newRunningWorkspace(t), volumes)
	svc.SetApplyConcurrency(1)

	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() {
		_, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
		first <- err
	}()
	<-gated.entered // la primera aplicación ya ocupa el único turno

	go func() {
		_, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
		second <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&gated.calls); got != 1 {
		t.Fatalf("la segunda aplicación no debe leer el snapshot mientras la primera ocupa el turno, lecturas: %d", got)
	}

	close(gated.release)
	if err := <-first; err != nil {
		t.Fatalf("primera aplicación falló: %v", err)
	}
	if err := <-second; err != nil {
		t.Fatalf("segunda aplicación falló: %v", err)
	}
	if got := atomic.LoadInt32(&gated.calls); got != 2 {
		t.Errorf("esperadas 2 lecturas del snapshot al terminar la cola, obtenidas %d", got)
	}
}

func TestApplyAdvance_HidesAdvancesFromOtherStudents(t *testing.T) {
	store := &fakeAdvanceStore{}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, newRunningWorkspace(t), volumes)

	_, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceNotFound) {
		t.Fatalf("esperado no encontrado, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 {
		t.Error("un avance ajeno no debe escribir en el volumen")
	}
}

func TestApplyAdvance_BlocksExam(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "exam", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{}, newRunningWorkspace(t), volumes)

	_, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if !errors.Is(err, domain.ErrAdvanceExamBlocked) {
		t.Fatalf("esperado error de examen, obtenido: %v", err)
	}
	if len(volumes.writes) != 0 {
		t.Error("en examen no se copia nada al alumno")
	}
}
