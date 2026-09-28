package services_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockBackupRepo struct {
	cfg         *domain.BackupConfig
	upserted    *domain.BackupConfig
	lastUpdated *domain.BackupExecution
}

func newMockBackupRepo() *mockBackupRepo {
	return &mockBackupRepo{
		cfg: &domain.BackupConfig{
			ID:                  "cfg-1",
			TenantID:            "t-1",
			LocalFrequencyHours: 6,
			LocalRetentionDays:  7,
			IsActive:            true,
		},
	}
}

func (m *mockBackupRepo) GetConfig(ctx context.Context, tenantID string) (*domain.BackupConfig, error) {
	cpy := *m.cfg
	return &cpy, nil
}

func (m *mockBackupRepo) UpsertConfig(ctx context.Context, config *domain.BackupConfig) error {
	m.upserted = config
	m.cfg = config
	return nil
}

func (m *mockBackupRepo) CreateExecution(ctx context.Context, execution *domain.BackupExecution) error {
	return nil
}

func (m *mockBackupRepo) UpdateExecution(ctx context.Context, execution *domain.BackupExecution) error {
	m.lastUpdated = execution
	return nil
}

func (m *mockBackupRepo) GetExecutionByID(ctx context.Context, tenantID, id string) (*domain.BackupExecution, error) {
	return nil, nil
}

func (m *mockBackupRepo) ListExecutions(ctx context.Context, tenantID string, limit, offset int) ([]*domain.BackupExecution, int64, error) {
	return nil, 0, nil
}

func (m *mockBackupRepo) GetExpiredExecutions(ctx context.Context, tenantID string, retentionDays int) ([]*domain.BackupExecution, error) {
	return nil, nil
}

func (m *mockBackupRepo) DeleteExecution(ctx context.Context, id string) error { return nil }

func (m *mockBackupRepo) UpdateExecutionVerifyColumns(ctx context.Context, execution *domain.BackupExecution) error {
	return nil
}

func TestBackupService_UpdateConfig_FrequencyEdges(t *testing.T) {
	cases := []struct {
		name      string
		freq      int
		wantCode  string
		wantValid bool
	}{
		{"freq 0 rejected", 0, "backup_frequency_invalid", false},
		{"freq 1 accepted", 1, "", true},
		{"freq 168 accepted", 168, "", true},
		{"freq 169 rejected", 169, "backup_frequency_invalid", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockBackupRepo()
			svc := services.NewBackupService(repo, nil, t.TempDir())
			_, err := svc.UpdateConfig(context.Background(), "t-1", domain.UpdateBackupConfigDTO{
				LocalFrequencyHours: tc.freq,
				LocalRetentionDays:  7,
			})
			if tc.wantValid {
				if err != nil {
					t.Fatalf("expected valid freq %d, got err %v", tc.freq, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected BackupValidationError for freq %d, got nil", tc.freq)
			}
			vErr, ok := err.(*services.BackupValidationError)
			if !ok {
				t.Fatalf("expected *BackupValidationError, got %T (%v)", err, err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("expected code %q, got %q", tc.wantCode, vErr.Code)
			}
		})
	}
}

func TestBackupService_UpdateConfig_RetentionEdges(t *testing.T) {
	cases := []struct {
		name      string
		retention int
		wantCode  string
		wantValid bool
	}{
		{"retention 0 rejected", 0, "backup_retention_invalid", false},
		{"retention 1 accepted", 1, "", true},
		{"retention 365 accepted", 365, "", true},
		{"retention 366 rejected", 366, "backup_retention_invalid", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockBackupRepo()
			svc := services.NewBackupService(repo, nil, t.TempDir())
			_, err := svc.UpdateConfig(context.Background(), "t-1", domain.UpdateBackupConfigDTO{
				LocalFrequencyHours: 6,
				LocalRetentionDays:  tc.retention,
			})
			if tc.wantValid {
				if err != nil {
					t.Fatalf("expected valid retention %d, got err %v", tc.retention, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected BackupValidationError for retention %d, got nil", tc.retention)
			}
			vErr, ok := err.(*services.BackupValidationError)
			if !ok {
				t.Fatalf("expected *BackupValidationError, got %T (%v)", err, err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("expected code %q, got %q", tc.wantCode, vErr.Code)
			}
		})
	}
}

func TestBackupService_UpdateConfig_RemoteRetentionEdges(t *testing.T) {
	over := 400
	cases := []struct {
		name      string
		retention *int
		wantValid bool
	}{
		{"nil remote retention accepted", nil, true},
		{"remote retention 400 rejected", &over, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockBackupRepo()
			svc := services.NewBackupService(repo, nil, t.TempDir())
			_, err := svc.UpdateConfig(context.Background(), "t-1", domain.UpdateBackupConfigDTO{
				LocalFrequencyHours: 6,
				LocalRetentionDays:  7,
				RemoteRetentionDays: tc.retention,
			})
			if tc.wantValid && err != nil {
				t.Fatalf("expected valid, got err %v", err)
			}
			if !tc.wantValid && err == nil {
				t.Fatalf("expected BackupValidationError, got nil")
			}
		})
	}
}

type failingBackupContentSource struct{ err error }

func (f failingBackupContentSource) WriteBackup(context.Context, string, time.Time, io.Writer) error {
	return f.err
}

func TestBackupService_TriggerBackupFailsClosedOnContentWriteError(t *testing.T) {
	repo := newMockBackupRepo()
	backupDir := t.TempDir()
	svc := services.NewBackupServiceWithContentSource(repo, nil, backupDir, failingBackupContentSource{err: errors.New("fixture write failed")})

	_, err := svc.TriggerBackup(context.Background(), "tenant-1", "")
	if err == nil || !strings.Contains(err.Error(), "fixture write failed") {
		t.Fatalf("expected content write error, got %v", err)
	}
	if repo.lastUpdated == nil || repo.lastUpdated.Status != domain.BackupStatusFailed {
		t.Fatalf("expected failed execution persisted, got %#v", repo.lastUpdated)
	}
	if repo.lastUpdated.SHA256Checksum != "" {
		t.Fatalf("failed execution must not store a checksum, got %q", repo.lastUpdated.SHA256Checksum)
	}
	if _, statErr := os.Stat(filepath.Join(backupDir, repo.lastUpdated.FileName)); !os.IsNotExist(statErr) {
		t.Fatalf("partial backup should be removed, stat error = %v", statErr)
	}
}

func TestBackupService_TriggerBackupRejectsOutputBelowFloor(t *testing.T) {
	repo := newMockBackupRepo()
	backupDir := t.TempDir()
	svc := services.NewBackupService(repo, nil, backupDir)

	_, err := svc.TriggerBackup(context.Background(), "tenant-1", "")
	if err == nil || !strings.Contains(err.Error(), "below minimum size floor") {
		t.Fatalf("expected below-floor failure, got %v", err)
	}
	if repo.lastUpdated == nil || repo.lastUpdated.Status != domain.BackupStatusFailed {
		t.Fatalf("expected failed execution persisted, got %#v", repo.lastUpdated)
	}
	if repo.lastUpdated.SHA256Checksum != "" {
		t.Fatalf("below-floor execution must not store a checksum, got %q", repo.lastUpdated.SHA256Checksum)
	}
	if _, statErr := os.Stat(filepath.Join(backupDir, repo.lastUpdated.FileName)); !os.IsNotExist(statErr) {
		t.Fatalf("below-floor output should be removed, stat error = %v", statErr)
	}
}
