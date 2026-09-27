package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockMaintenanceTenantRepo struct {
	status        *domain.MaintenanceStatus
	setCalls      int
	lastEnabled   bool
	lastUntil     *time.Time
	lastReason    string
	allowNotFound bool
}

func (m *mockMaintenanceTenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	return &domain.Tenant{ID: id}, nil
}

func (m *mockMaintenanceTenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return &domain.Tenant{ID: "t-1", Slug: slug}, nil
}

func (m *mockMaintenanceTenantRepo) GetAll(ctx context.Context) ([]*domain.Tenant, error) {
	return nil, nil
}

func (m *mockMaintenanceTenantRepo) UpdateConfig(ctx context.Context, id string, config []byte) error {
	return nil
}

func (m *mockMaintenanceTenantRepo) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	m.setCalls++
	m.lastEnabled = enabled
	m.lastUntil = until
	m.lastReason = reason
	if m.status == nil {
		m.status = &domain.MaintenanceStatus{}
	}
	m.status.MaintenanceMode = enabled
	m.status.MaintenanceUntil = until
	m.status.MaintenanceReason = reason
	return nil
}

func (m *mockMaintenanceTenantRepo) GetMaintenance(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	if m.status == nil {
		return &domain.MaintenanceStatus{MaintenanceMode: false}, nil
	}
	cpy := *m.status
	return &cpy, nil
}

func validMaintenanceDTO() domain.EnableMaintenanceDTO {
	return domain.EnableMaintenanceDTO{
		Until:         time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		Reason:        "Ventana programada de actualizacion",
		ConfirmPhrase: "MANTENIMIENTO",
	}
}

func TestMaintenanceService_EnableValidation(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*domain.EnableMaintenanceDTO)
		wantCode string
	}{
		{"phrase mismatch rejected", func(d *domain.EnableMaintenanceDTO) { d.ConfirmPhrase = "MANTENIMIENTO!" }, "maintenance_confirm_invalid"},
		{"phrase casing rejected", func(d *domain.EnableMaintenanceDTO) { d.ConfirmPhrase = "mantenimiento" }, "maintenance_confirm_invalid"},
		{"phrase empty rejected", func(d *domain.EnableMaintenanceDTO) { d.ConfirmPhrase = "" }, "maintenance_confirm_invalid"},
		{"motive 9 chars rejected", func(d *domain.EnableMaintenanceDTO) { d.Reason = "123456789" }, "maintenance_reason_invalid"},
		{"past until rejected", func(d *domain.EnableMaintenanceDTO) {
			d.Until = time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		}, "maintenance_until_invalid"},
		{"until garbage rejected", func(d *domain.EnableMaintenanceDTO) { d.Until = "not-a-date" }, "maintenance_until_invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockMaintenanceTenantRepo{}
			svc := services.NewMaintenanceService(repo)
			dto := validMaintenanceDTO()
			tc.mutate(&dto)
			err := svc.EnableMaintenance(context.Background(), "t-1", dto)
			if err == nil {
				t.Fatalf("expected MaintenanceValidationError, got nil")
			}
			vErr, ok := err.(*services.MaintenanceValidationError)
			if !ok {
				t.Fatalf("expected *MaintenanceValidationError, got %T (%v)", err, err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("expected code %q, got %q", tc.wantCode, vErr.Code)
			}
			if repo.setCalls != 0 {
				t.Fatalf("invalid request must not persist maintenance state")
			}
		})
	}
}

func TestMaintenanceService_EnableHappyPath(t *testing.T) {
	repo := &mockMaintenanceTenantRepo{}
	svc := services.NewMaintenanceService(repo)

	// Motive exactly 10 chars is accepted; empty until means indefinite.
	dto := domain.EnableMaintenanceDTO{
		Until:         "",
		Reason:        "1234567890",
		ConfirmPhrase: "MANTENIMIENTO",
	}
	if err := svc.EnableMaintenance(context.Background(), "t-1", dto); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.setCalls != 1 || !repo.lastEnabled {
		t.Fatalf("expected one enabling SetMaintenance call, got %+v", repo)
	}
	if repo.lastUntil != nil {
		t.Fatalf("empty until must persist as nil (indefinite), got %v", repo.lastUntil)
	}
	if !strings.Contains(repo.lastReason, "1234567890") {
		t.Fatalf("motive must persist verbatim, got %q", repo.lastReason)
	}
}

func TestMaintenanceService_GetStatus_LazyClearExpired(t *testing.T) {
	past := time.Now().Add(-30 * time.Minute)
	repo := &mockMaintenanceTenantRepo{
		status: &domain.MaintenanceStatus{
			MaintenanceMode:   true,
			MaintenanceUntil:  &past,
			MaintenanceReason: "Ventana programada de actualizacion",
		},
	}
	svc := services.NewMaintenanceService(repo)

	status, err := svc.GetStatus(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.MaintenanceMode {
		t.Fatalf("expired vigencia must read as off")
	}
	if repo.setCalls != 1 || repo.lastEnabled {
		t.Fatalf("expired vigencia must trigger one disabling SetMaintenance call")
	}
}
