package services_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type fakeAdvanceRetentionStore struct {
	candidates []domain.ClassAdvance
	keys       []string
	cutoff     time.Time
	limit      int
	marked     []string
	markErr    error
}

func (f *fakeAdvanceRetentionStore) ExpiredAdvanceSnapshots(ctx context.Context, cutoff time.Time, limit int) ([]domain.ClassAdvance, error) {
	f.cutoff = cutoff
	f.limit = limit
	return f.candidates, nil
}

func (f *fakeAdvanceRetentionStore) MarkSnapshotPurged(ctx context.Context, tenantID, advanceID string, purgedAt time.Time) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.marked = append(f.marked, advanceID)
	return nil
}

func (f *fakeAdvanceRetentionStore) AdvanceSnapshotKeys(ctx context.Context) ([]string, error) {
	return f.keys, nil
}

type fakeRetentionSnapshots struct {
	deleted   []string
	objects   []domain.AdvanceSnapshotObject
	deleteErr map[string]error
	listErr   error
}

func (f *fakeRetentionSnapshots) Put(ctx context.Context, key string, body io.Reader) (string, int64, error) {
	return "", 0, nil
}

func (f *fakeRetentionSnapshots) PutFiles(ctx context.Context, key string, files map[string]string) (string, int64, int, error) {
	return "", 0, 0, nil
}

func (f *fakeRetentionSnapshots) Files(ctx context.Context, key, expectedSHA256 string, maxFiles int) ([]domain.PracticeEnvironmentVolumeFile, error) {
	return nil, nil
}

func (f *fakeRetentionSnapshots) Delete(ctx context.Context, key string) error {
	if err := f.deleteErr[key]; err != nil {
		return err
	}
	f.deleted = append(f.deleted, key)
	return nil
}

func (f *fakeRetentionSnapshots) List(ctx context.Context, prefix string) ([]domain.AdvanceSnapshotObject, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.objects, nil
}

func TestPurgeExpiredSnapshots_DeletesExpiredAndOrphans(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	tenantID := "11111111-1111-1111-1111-111111111111"
	expired := domain.ClassAdvance{
		ID:          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		TenantID:    tenantID,
		SnapshotKey: "class-advances/" + tenantID + "/expired.tar.gz",
	}
	referencedKey := "class-advances/" + tenantID + "/referenced.tar.gz"
	orphanKey := "class-advances/" + tenantID + "/orphan.tar.gz"
	store := &fakeAdvanceRetentionStore{
		candidates: []domain.ClassAdvance{expired},
		keys:       []string{expired.SnapshotKey, referencedKey},
	}
	snapshots := &fakeRetentionSnapshots{
		objects: []domain.AdvanceSnapshotObject{
			{Key: expired.SnapshotKey, LastModified: now.Add(-10 * 24 * time.Hour)},
			{Key: referencedKey, LastModified: now.Add(-10 * 24 * time.Hour)},
			{Key: orphanKey, LastModified: now.Add(-25 * time.Hour)},
			{Key: "class-advances/" + tenantID + "/fresh.tar.gz", LastModified: now.Add(-time.Hour)},
		},
	}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, nil).
		WithApplier(snapshots, nil, nil)
	svc.SetRetentionStore(store)

	result, err := svc.PurgeExpiredSnapshots(context.Background(), now)
	if err != nil {
		t.Fatalf("PurgeExpiredSnapshots falló: %v", err)
	}
	if result.SnapshotsDeleted != 1 || result.OrphansDeleted != 1 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if result.SnapshotFailures != 0 || result.OrphanFailures != 0 {
		t.Fatalf("no debía haber fallos: %+v", result)
	}
	if len(store.marked) != 1 || store.marked[0] != expired.ID {
		t.Fatalf("el avance vencido debía quedar marcado: %+v", store.marked)
	}
	if len(snapshots.deleted) != 2 || snapshots.deleted[0] != expired.SnapshotKey ||
		snapshots.deleted[1] != orphanKey {
		t.Fatalf("objetos borrados inesperados: %+v", snapshots.deleted)
	}
	wantCutoff := now.Add(-8 * 24 * time.Hour)
	if !store.cutoff.Equal(wantCutoff) {
		t.Fatalf("el corte debía ser %v, fue %v", wantCutoff, store.cutoff)
	}
}

func TestPurgeExpiredSnapshots_DeleteFailureLeavesTheRowUnmarked(t *testing.T) {
	now := time.Now()
	tenantID := "11111111-1111-1111-1111-111111111111"
	expired := domain.ClassAdvance{
		ID:          "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TenantID:    tenantID,
		SnapshotKey: "class-advances/" + tenantID + "/failing.tar.gz",
	}
	store := &fakeAdvanceRetentionStore{
		candidates: []domain.ClassAdvance{expired},
		keys:       []string{expired.SnapshotKey},
	}
	snapshots := &fakeRetentionSnapshots{
		deleteErr: map[string]error{expired.SnapshotKey: errors.New("storage down")},
	}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, nil).
		WithApplier(snapshots, nil, nil)
	svc.SetRetentionStore(store)

	result, err := svc.PurgeExpiredSnapshots(context.Background(), now)
	if err != nil {
		t.Fatalf("un fallo de borrado no debía abortar el barrido: %v", err)
	}
	if result.SnapshotsDeleted != 0 || result.SnapshotFailures != 1 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if len(store.marked) != 0 {
		t.Fatalf("un objeto que no se borró no debía marcarse: %+v", store.marked)
	}
}

func TestPurgeExpiredSnapshots_DoesNotDeleteAnotherTenantKey(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	expired := domain.ClassAdvance{
		ID:          "cccccccc-cccc-cccc-cccc-cccccccccccc",
		TenantID:    tenantID,
		SnapshotKey: "class-advances/22222222-2222-2222-2222-222222222222/foreign.tar.gz",
	}
	store := &fakeAdvanceRetentionStore{
		candidates: []domain.ClassAdvance{expired},
		keys:       []string{expired.SnapshotKey},
	}
	snapshots := &fakeRetentionSnapshots{}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, nil).
		WithApplier(snapshots, nil, nil)
	svc.SetRetentionStore(store)

	result, err := svc.PurgeExpiredSnapshots(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("un prefijo ajeno no debía abortar el barrido: %v", err)
	}
	if len(snapshots.deleted) != 0 {
		t.Fatalf("no se podía borrar un objeto de otro tenant: %+v", snapshots.deleted)
	}
	if result.SnapshotFailures != 1 || len(store.marked) != 0 {
		t.Fatalf("la fila debía quedar sin marcar: %+v, marcadas: %+v", result, store.marked)
	}
}

func TestPurgeExpiredSnapshots_RequiresStorage(t *testing.T) {
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, nil)
	_, err := svc.PurgeExpiredSnapshots(context.Background(), time.Now())
	if !errors.Is(err, services.ErrAdvanceStorageUnavailable) {
		t.Fatalf("se esperaba ErrAdvanceStorageUnavailable, fue: %v", err)
	}
}

type fakePruningVolumes struct {
	fakeAdvanceVolumes
	dirs      []string
	deleted   []string
	listErr   error
	deleteErr error
}

func (f *fakePruningVolumes) ListPracticeEnvironmentBackupDirs(ctx context.Context, volumeName, root string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.dirs, nil
}

func (f *fakePruningVolumes) DeletePracticeEnvironmentVolumePaths(ctx context.Context, volumeName string, paths []string) error {
	f.deleted = append(f.deleted, paths...)
	return f.deleteErr
}

func TestApplyAdvance_PrunesBackupsBeyondTheLimit(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)}
	snapshots := &fakeAdvanceSnapshotsFiles{files: []domain.PracticeEnvironmentVolumeFile{
		{Path: "main.py", Content: []byte("nuevo\n")},
	}}
	dirs := []string{
		".solv_backups/" + advanceID,
		".solv_backups/b", ".solv_backups/c", ".solv_backups/d",
		".solv_backups/e", ".solv_backups/f", ".solv_backups/g",
	}
	volumes := &fakePruningVolumes{
		fakeAdvanceVolumes: fakeAdvanceVolumes{existing: []domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("viejo\n")},
		}},
		dirs: dirs,
	}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(snapshots, newRunningPracticeEnvironment(t), volumes)

	if _, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID); err != nil {
		t.Fatalf("Apply falló: %v", err)
	}
	if len(volumes.deleted) != 2 || volumes.deleted[0] != dirs[5] || volumes.deleted[1] != dirs[6] {
		t.Fatalf("debía borrar los dos respaldos más viejos, borró: %+v", volumes.deleted)
	}
}

func TestApplyAdvance_PruneFailureDoesNotBreakApply(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)}
	snapshots := &fakeAdvanceSnapshotsFiles{files: []domain.PracticeEnvironmentVolumeFile{
		{Path: "main.py", Content: []byte("nuevo\n")},
	}}
	volumes := &fakePruningVolumes{
		fakeAdvanceVolumes: fakeAdvanceVolumes{existing: []domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("viejo\n")},
		}},
		dirs:      []string{".solv_backups/" + advanceID, ".solv_backups/b", ".solv_backups/c", ".solv_backups/d", ".solv_backups/e", ".solv_backups/f"},
		deleteErr: errors.New("volume busy"),
	}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(snapshots, newRunningPracticeEnvironment(t), volumes)

	result, err := svc.Apply(context.Background(), advanceTenantID, advanceStudentID, advanceID)
	if err != nil {
		t.Fatalf("un fallo de poda no debía romper la aplicación: %v", err)
	}
	if result == nil || result.FilesWritten != 1 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if len(store.decisions) != 1 || store.decisions[0].status != domain.AdvanceRecipientApplied {
		t.Fatalf("el destinatario debía quedar aplicado: %+v", store.decisions)
	}
}
