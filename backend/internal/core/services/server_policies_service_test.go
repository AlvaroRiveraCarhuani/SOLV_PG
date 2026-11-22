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
	if p.RAMLimitMB != def.RAMLimitMB || p.InactivityMinutes != def.InactivityMinutes || p.MaxContainers != def.MaxContainers || p.HostSafetyCeilingPct != def.HostSafetyCeilingPct {
		t.Fatalf("expected defaults %+v, got %+v", def, p)
	}
}

func TestServerPolicies_UpdateWithValidation(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo)
	ctx := context.Background()

	// Actualización válida
	updated, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{
		RAMLimitMB:           policiesIntPtr(1024),
		InactivityMinutes:    policiesIntPtr(60),
		MaxContainers:        policiesIntPtr(60),
		HostSafetyCeilingPct: policiesIntPtr(80),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.RAMLimitMB != 1024 || updated.InactivityMinutes != 60 || updated.MaxContainers != 60 || updated.HostSafetyCeilingPct != 80 {
		t.Fatalf("expected updated policies, got %+v", updated)
	}

	// Persistida y legible de vuelta
	reloaded, _ := svc.Get(ctx, "t-1")
	if reloaded.RAMLimitMB != 1024 || reloaded.HostSafetyCeilingPct != 80 {
		t.Fatalf("expected persisted ram 1024 and ceiling 80, got %d and %d", reloaded.RAMLimitMB, reloaded.HostSafetyCeilingPct)
	}

	// Fuera de rango: rechazado sin persistir
	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{RAMLimitMB: policiesIntPtr(9999)}); err == nil {
		t.Fatal("expected ram_limit_invalid error")
	} else if code := func() *services.PoliciesValidationError {
		target := &services.PoliciesValidationError{}
		_ = errors.As(err, &target)
		return target
	}().Code; code != "ram_limit_invalid" {
		t.Fatalf("expected ram_limit_invalid, got %s", code)
	}

	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{InactivityMinutes: policiesIntPtr(7)}); err == nil {
		t.Fatal("expected inactivity_invalid error")
	}

	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{MaxContainers: policiesIntPtr(0)}); err == nil {
		t.Fatal("expected max_containers_invalid error")
	}

	if _, err := svc.Update(ctx, "t-1", domain.UpdateServerPoliciesDTO{HostSafetyCeilingPct: policiesIntPtr(70)}); err == nil {
		t.Fatal("expected host_ceiling_invalid error")
	} else if code := func() *services.PoliciesValidationError {
		target := &services.PoliciesValidationError{}
		_ = errors.As(err, &target)
		return target
	}().Code; code != "host_ceiling_invalid" {
		t.Fatalf("expected host_ceiling_invalid, got %s", code)
	}

	// El valor rechazado no se persistió
	final, _ := svc.Get(ctx, "t-1")
	if final.RAMLimitMB != 1024 || final.InactivityMinutes != 60 || final.MaxContainers != 60 || final.HostSafetyCeilingPct != 80 {
		t.Fatalf("rejected update must not persist, got %+v", final)
	}
}

// Un tenant sin políticas guardadas usa un techo derivado del host medido. Con
// 64 GB y entornos de 512 MB entran 104, no los 40 pensados para una máquina
// chica.
func TestServerPolicies_DefaultCapFollowsHostRAM(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo).WithHostTotalRAM(func() (int64, error) {
		return 65536, nil
	})

	policies, err := svc.Get(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if policies.MaxContainers != 104 {
		t.Fatalf("expected a 104-lab ceiling for 64 GB, got %d", policies.MaxContainers)
	}
	if policies.RAMLimitMB != domain.DefaultServerPolicies().RAMLimitMB {
		t.Fatalf("el resto de las políticas debe seguir en el default, got %+v", policies)
	}
}

// Si el host no se puede medir, el valor guardado por el administrador manda y
// no se inventa un techo nuevo.
func TestServerPolicies_DefaultCapWithoutMeasuredHost(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo).WithHostTotalRAM(func() (int64, error) {
		return 0, errors.New("sin telemetría")
	})

	policies, err := svc.Get(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policies.MaxContainers != domain.DefaultServerPolicies().MaxContainers {
		t.Fatalf("expected the fallback ceiling, got %d", policies.MaxContainers)
	}
}

// Un techo guardado explícitamente no se pisa, aunque el host sostenga más: es
// una decisión del administrador. La recomendación se informa aparte.
func TestServerPolicies_PersistedCapWinsOverRecommendation(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{
		"t-1": []byte(`{"server_policies":{"ram_limit_mb":512,"inactivity_minutes":15,"max_containers":40,"host_safety_ceiling_pct":85}}`),
	}}
	svc := services.NewServerPoliciesService(repo).WithHostTotalRAM(func() (int64, error) {
		return 65536, nil
	})

	policies, err := svc.Get(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policies.MaxContainers != 40 {
		t.Fatalf("expected the persisted ceiling of 40, got %d", policies.MaxContainers)
	}

	recommendation := svc.Recommendation(*policies)
	if recommendation.RecommendedMaxContainers != 104 {
		t.Fatalf("expected a 104-lab recommendation next to the persisted cap, got %d", recommendation.RecommendedMaxContainers)
	}
}

// Guardar solo el techo de seguridad persiste el máximo derivado, para que el
// host no quede limitado por un número que nadie eligió.
func TestServerPolicies_UpdatePersistsTheDerivedCap(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo).WithHostTotalRAM(func() (int64, error) {
		return 32768, nil
	})

	ceiling := 80
	updated, err := svc.Update(context.Background(), "t-1", domain.UpdateServerPoliciesDTO{HostSafetyCeilingPct: &ceiling})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 GB al 80% menos la reserva de plataforma, entre 512 MB: 47.
	if updated.MaxContainers != 47 {
		t.Fatalf("expected the derived ceiling of 47, got %d", updated.MaxContainers)
	}

	reloaded, _ := svc.Get(context.Background(), "t-1")
	if reloaded.MaxContainers != 47 {
		t.Fatalf("expected the derived ceiling to persist, got %d", reloaded.MaxContainers)
	}
}

func TestServerPolicies_UpdatePracticeCPUMillis_Valid(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo)

	validCases := []int{0, 500, 1000, 1500, 2000}
	for _, val := range validCases {
		cpuVal := val
		updated, err := svc.Update(context.Background(), "t-1", domain.UpdateServerPoliciesDTO{
			PracticeCPUMillis: &cpuVal,
		})
		if err != nil {
			t.Fatalf("esperaba exito para cpu %d, obtuve error: %v", val, err)
		}
		if updated.PracticeCPUMillis != val {
			t.Fatalf("esperaba PracticeCPUMillis=%d, obtuve %d", val, updated.PracticeCPUMillis)
		}

		reloaded, _ := svc.Get(context.Background(), "t-1")
		if reloaded.PracticeCPUMillis != val {
			t.Fatalf("esperaba valor persistido %d, obtuve %d", val, reloaded.PracticeCPUMillis)
		}
	}
}

func TestServerPolicies_UpdatePracticeCPUMillis_Invalid(t *testing.T) {
	repo := &mockPoliciesTenantRepo{configs: map[string][]byte{"t-1": []byte(`{}`)}}
	svc := services.NewServerPoliciesService(repo)

	invalidCases := []int{-1, 333, 750, 3000}
	for _, val := range invalidCases {
		cpuVal := val
		_, err := svc.Update(context.Background(), "t-1", domain.UpdateServerPoliciesDTO{
			PracticeCPUMillis: &cpuVal,
		})
		if err == nil {
			t.Fatalf("esperaba error para valor invalido %d pero paso", val)
		}
		var valErr *services.PoliciesValidationError
		if !errors.As(err, &valErr) || valErr.Code != "practice_cpu_invalid" {
			t.Fatalf("esperaba error con codigo practice_cpu_invalid para %d, obtuve: %v", val, err)
		}
	}
}
