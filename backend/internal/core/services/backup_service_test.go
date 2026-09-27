package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockBackupRepo struct {
	cfg      *domain.BackupConfig
	upserted *domain.BackupConfig
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
