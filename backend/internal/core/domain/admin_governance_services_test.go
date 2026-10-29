package domain

import (
	"errors"
	"testing"
)

func servicesOf(items ...ServiceRequirement) *ServicesConfig {
	return &ServicesConfig{Services: items}
}

func TestNormalizeServiceBindings_LegacyCategoryCache(t *testing.T) {
	cfg := servicesOf(ServiceRequirement{Category: "cache", Engine: "redis"})

	if err := NormalizeServiceBindings(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Services) != 1 {
		t.Fatalf("expected one binding, got %d", len(cfg.Services))
	}
	binding := cfg.Services[0]
	if binding.Role != ServiceRoleCache {
		t.Errorf("role = %q, want %q", binding.Role, ServiceRoleCache)
	}
	if binding.Engine != "redis" {
		t.Errorf("engine = %q, want redis", binding.Engine)
	}
	if binding.EnvAlias != "cache" {
		t.Errorf("env_alias = %q, want cache", binding.EnvAlias)
	}
}

func TestNormalizeServiceBindings_AllowsPrimaryPlusCache(t *testing.T) {
	cfg := servicesOf(
		ServiceRequirement{Engine: "postgres"},
		ServiceRequirement{Engine: "redis"},
	)

	if err := NormalizeServiceBindings(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("expected two bindings, got %d", len(cfg.Services))
	}
	if cfg.Services[0].Role != ServiceRolePrimaryDatabase || cfg.Services[0].EnvAlias != "db" || cfg.Services[0].Category != "database" {
		t.Errorf("primary binding unexpected: %+v", cfg.Services[0])
	}
	if cfg.Services[1].Role != ServiceRoleCache || cfg.Services[1].EnvAlias != "cache" || cfg.Services[1].Category != "cache" {
		t.Errorf("cache binding unexpected: %+v", cfg.Services[1])
	}
}

func TestNormalizeServiceBindings_CategoryFollowsRole(t *testing.T) {
	// Un Redis etiquetado como "database" no puede quedar con esa categoría: category es
	// el alias de lectura de role, no una segunda fuente de verdad.
	cfg := servicesOf(ServiceRequirement{Category: "database", Engine: "redis"})

	if err := NormalizeServiceBindings(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Services[0].Role != ServiceRoleCache || cfg.Services[0].Category != "cache" {
		t.Fatalf("category debe derivarse del rol: %+v", cfg.Services[0])
	}
}

func TestNormalizeServiceBindings_RejectsTwoPrimaries(t *testing.T) {
	cfg := servicesOf(
		ServiceRequirement{Engine: "mysql"},
		ServiceRequirement{Engine: "postgres"},
	)

	err := NormalizeServiceBindings(cfg)
	if !errors.Is(err, ErrMultiplePrimaryDatabase) {
		t.Fatalf("expected ErrMultiplePrimaryDatabase, got %v", err)
	}
}

func TestNormalizeServiceBindings_RejectsDuplicateEngine(t *testing.T) {
	cfg := servicesOf(
		ServiceRequirement{Engine: "redis"},
		ServiceRequirement{Engine: "redis", EnvAlias: "cache2"},
	)

	err := NormalizeServiceBindings(cfg)
	if !errors.Is(err, ErrDuplicateServiceEngine) {
		t.Fatalf("expected ErrDuplicateServiceEngine, got %v", err)
	}
}

func TestNormalizeServiceBindings_RejectsRoleMismatch(t *testing.T) {
	cfg := servicesOf(ServiceRequirement{Role: ServiceRolePrimaryDatabase, Engine: "redis"})

	err := NormalizeServiceBindings(cfg)
	if !errors.Is(err, ErrServiceRoleMismatch) {
		t.Fatalf("expected ErrServiceRoleMismatch, got %v", err)
	}
}

func TestNormalizeServiceBindings_RejectsUnknownEngine(t *testing.T) {
	cfg := servicesOf(ServiceRequirement{Engine: "oracle"})

	err := NormalizeServiceBindings(cfg)
	if !errors.Is(err, ErrUnsupportedServiceEngine) {
		t.Fatalf("expected ErrUnsupportedServiceEngine, got %v", err)
	}
}

func TestNormalizeServiceBindings_DropsNone(t *testing.T) {
	cfg := servicesOf(
		ServiceRequirement{Engine: "none"},
		ServiceRequirement{Engine: "postgresql"},
	)

	if err := NormalizeServiceBindings(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Engine != "postgres" {
		t.Fatalf("expected canonical postgres only, got %+v", cfg.Services)
	}
}

func TestNormalizeServiceBindings_RejectsDuplicateAlias(t *testing.T) {
	cfg := servicesOf(
		ServiceRequirement{Engine: "postgres", EnvAlias: "data"},
		ServiceRequirement{Engine: "redis", EnvAlias: "data"},
	)

	err := NormalizeServiceBindings(cfg)
	if !errors.Is(err, ErrDuplicateServiceAlias) {
		t.Fatalf("expected ErrDuplicateServiceAlias, got %v", err)
	}
}

func TestServiceRequirement_InitKind(t *testing.T) {
	cases := []struct {
		engine string
		want   ServiceInitKind
	}{
		{engine: "postgres", want: ServiceInitSQL},
		{engine: "mysql", want: ServiceInitSQL},
		{engine: "mongodb", want: ServiceInitMongoJS},
		{engine: "redis", want: ServiceInitRedisCmds},
		{engine: "sqlite", want: ServiceInitNone},
	}

	for _, tc := range cases {
		t.Run(tc.engine, func(t *testing.T) {
			if got := (ServiceRequirement{Engine: tc.engine}).InitKind(); got != tc.want {
				t.Fatalf("InitKind() = %q, want %q", got, tc.want)
			}
		})
	}
}
