package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockPoliciesTenantRepo struct {
	configs map[string][]byte
}

func (m *mockPoliciesTenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return &domain.Tenant{ID: id, Config: cfg}, nil
}

func (m *mockPoliciesTenantRepo) UpdateConfig(ctx context.Context, id string, config []byte) error {
	m.configs[id] = config
	return nil
}

// Métodos restantes de la interfaz: no usados en estos tests.
func (m *mockPoliciesTenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return nil, nil
}
func (m *mockPoliciesTenantRepo) GetAll(ctx context.Context) ([]*domain.Tenant, error) {
	return nil, nil
}
func (m *mockPoliciesTenantRepo) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	return nil
}
func (m *mockPoliciesTenantRepo) GetMaintenance(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	return &domain.MaintenanceStatus{}, nil
}

func policiesIntPtr(v int) *int {
	return &v
}

func TestServerPolicies_DefaultsWhenNeverConfigured(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{
		"t-1": []byte(`{"tenant_primary_color":"#2563EB"}`),
	}}
	svc := services.NewServerPoliciesService(repo)

	p, err := svc.Get(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	def := domain.DefaultServerPolicies()
	if p.RAMLimitMB != def.RAMLimitMB || p.InactivityMinutes != def.InactivityMinutes || p.MaxContainers != def.MaxContainers {
		t.Fatalf("expected defaults %+v, got %+v", def, p)
	}
}

func TestServerPolicies_UpdateWithValidation(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo)
	ctx := context.Background()

	// Actualización válida
	updated, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{
		RAMLimitMB:        policiesIntPtr(1024),
		InactivityMinutes: policiesIntPtr(10),
		MaxContainers:     policiesIntPtr(60),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.RAMLimitMB != 1024 || updated.InactivityMinutes != 10 || updated.MaxContainers != 60 {
		t.Fatalf("expected updated policies, got %+v", updated)
	}

	// Persistida y legible de vuelta
	reloaded, _ := svc.Get(ctx, "t-1")
	if reloaded.RAMLimitMB != 1024 {
		t.Fatalf("expected persisted ram 1024, got %d", reloaded.RAMLimitMB)
	}

	// Fuera de rango: rechazado sin persistir
	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{RAMLimitMB: policiesIntPtr(9999)}); err == nil {
		t.Fatal("expected ram_limit_invalid error")
	} else if code := err.(*services.PoliciesValidationError).Code; code != "ram_limit_invalid" {
		t.Fatalf("expected ram_limit_invalid, got %s", code)
	}

	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{InactivityMinutes: policiesIntPtr(7)}); err == nil {
		t.Fatal("expected inactivity_invalid error")
	}

	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{MaxContainers: policiesIntPtr(0)}); err == nil {
		t.Fatal("expected max_containers_invalid error")
	}

	// El valor rechazado no se persistió
	final, _ := svc.Get(ctx, "t-1")
	if final.RAMLimitMB != 1024 || final.InactivityMinutes != 10 || final.MaxContainers != 60 {
		t.Fatalf("rejected update must not persist, got %+v", final)
	}
}
