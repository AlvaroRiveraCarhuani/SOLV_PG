package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestGetRuntimeCapabilities(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{serverMajorVersion: "18"}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	caps, err := svc.GetRuntimeCapabilities(context.Background())
	if err != nil {
		t.Fatalf("error inesperado al obtener capacidades de ejecución: %v", err)
	}

	if caps == nil {
		t.Fatal("las capacidades no deben ser nulas")
	}

	if caps.HostMemory.TotalRAMMB <= 0 {
		t.Errorf("total_ram_mb debe ser mayor a 0, obtenido: %d", caps.HostMemory.TotalRAMMB)
	}

	if len(caps.SatelliteServices) == 0 {
		t.Error("debe reportar catálogo de servicios satélite")
	}

	foundSQLite := false
	foundPostgres := false
	for _, s := range caps.SatelliteServices {
		if s.BaseRAMMB < 0 {
			t.Errorf("BaseRAMMB de %s no puede ser negativo, obtenido: %d", s.Engine, s.BaseRAMMB)
		}
		if s.Engine == "sqlite" {
			foundSQLite = true
			if !s.IsAvailable {
				t.Error("SQLite debe estar disponible en las capacidades")
			}
			if s.BaseRAMMB != 0 {
				t.Errorf("BaseRAMMB de sqlite debe ser 0 según ADR-045, obtenido: %d", s.BaseRAMMB)
			}
		}
		if s.Engine == "postgres" && s.IsAvailable {
			foundPostgres = true
			if s.Description == "" {
				t.Error("descripción de postgres no debe estar vacía")
			}
			if s.BaseRAMMB != 384 {
				t.Errorf("BaseRAMMB de postgres debe ser 384 (techo de cgroup del satélite, ADR-051), obtenido: %d", s.BaseRAMMB)
			}
			if s.Version != "18" {
				t.Errorf("versión de postgres debe ser 18 (resuelta en vivo), obtenido: %s", s.Version)
			}
		}
	}
	if !foundSQLite {
		t.Error("SQLite debe estar presente en las capacidades")
	}
	if !foundPostgres {
		t.Error("PostgreSQL debe estar disponible en las capacidades")
	}

	if len(caps.IDEPresets) == 0 || len(caps.JudgePresets) == 0 {
		t.Error("deben existir presets tanto para IDE como para Juez")
	}

	if caps.MaxAllowedRamMB <= 0 {
		t.Errorf("max_allowed_ram_mb debe ser positivo, obtenido: %d", caps.MaxAllowedRamMB)
	}
}

// Los motores de contenedor se levantan bajo demanda: no dependen de variables de host
// declaradas a mano y todos figuran disponibles con su modo de aprovisionamiento.
func TestGetRuntimeCapabilities_AllEnginesAvailableWithProvisioningMode(t *testing.T) {
	svc := services.NewAdminGovernanceService(nil, &mockAdminGovernanceRepo{serverMajorVersion: "18"})

	caps, err := svc.GetRuntimeCapabilities(context.Background())
	if err != nil {
		t.Fatalf("error inesperado al obtener capacidades: %v", err)
	}

	byEngine := make(map[string]domain.SatelliteServiceCapability, len(caps.SatelliteServices))
	for _, s := range caps.SatelliteServices {
		byEngine[s.Engine] = s
	}

	want := map[string]struct {
		mode          string
		baseRAMMB     int
		defaultImage  string
		requiresImage bool
	}{
		"sqlite":   {mode: domain.ProvisioningEmbedded, baseRAMMB: 0},
		"postgres": {mode: domain.ProvisioningManagedContainer, baseRAMMB: 384, defaultImage: "postgres:18-alpine", requiresImage: true},
		"mysql":    {mode: domain.ProvisioningManagedContainer, baseRAMMB: 128, defaultImage: "mysql:8.4", requiresImage: true},
		"mariadb":  {mode: domain.ProvisioningManagedContainer, baseRAMMB: 128, defaultImage: "mariadb:11.4", requiresImage: true},
		"mongodb":  {mode: domain.ProvisioningManagedContainer, baseRAMMB: 128, defaultImage: "mongo:7.0", requiresImage: true},
		"redis":    {mode: domain.ProvisioningManagedContainer, baseRAMMB: 64, defaultImage: "redis:7.2-alpine", requiresImage: true},
	}

	for engine, expected := range want {
		capability, ok := byEngine[engine]
		if !ok {
			t.Fatalf("el motor %s debe figurar en las capacidades del nodo", engine)
		}
		if !capability.IsAvailable {
			t.Errorf("el motor %s debe estar disponible: los motores gestionados se levantan bajo demanda", engine)
		}
		if capability.ProvisioningMode != expected.mode {
			t.Errorf("provisioning_mode de %s = %q, esperado %q", engine, capability.ProvisioningMode, expected.mode)
		}
		if capability.BaseRAMMB != expected.baseRAMMB {
			t.Errorf("base_ram_mb de %s = %d, esperado %d", engine, capability.BaseRAMMB, expected.baseRAMMB)
		}
		if expected.requiresImage {
			if capability.DefaultImage != expected.defaultImage {
				t.Errorf("default_image de %s = %q, esperado %q", engine, capability.DefaultImage, expected.defaultImage)
			}
			if strings.Contains(capability.DefaultImage, ":latest") || !strings.Contains(capability.DefaultImage, ":") {
				t.Errorf("default_image de %s debe llevar tag de versión fija, obtenido %q", engine, capability.DefaultImage)
			}
		}
		if capability.EnvVar == "" || capability.Label == "" || capability.Version == "" {
			t.Errorf("el motor %s debe declarar etiqueta, versión y variable de entorno", engine)
		}
	}

	if len(caps.SatelliteServices) != len(want) {
		t.Errorf("esperados %d motores soportados, obtenidos %d", len(want), len(caps.SatelliteServices))
	}
}

func TestAdminGovernanceService_GetImageUsageCounts(t *testing.T) {
	mockRepo := &mockAdminGovernanceRepo{}
	svc := services.NewAdminGovernanceService(nil, mockRepo)

	usageMap, err := svc.GetImageUsageCounts(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if usageMap["python:3.12-slim-bookworm"] != 4 {
		t.Errorf("esperado 4 usos para python, obtenido: %d", usageMap["python:3.12-slim-bookworm"])
	}
	if usageMap["node:20-bookworm-slim"] != 2 {
		t.Errorf("esperado 2 usos para node, obtenido: %d", usageMap["node:20-bookworm-slim"])
	}
}

func TestValidateRamAgainstHost(t *testing.T) {
	// Techo mínimo 512 MB incluso si host es muy chico
	ceilingLow := domain.CalculateHostMaxAllowedRAM(400)
	if ceilingLow != 512 {
		t.Errorf("esperado 512 MB de techo mínimo, obtenido: %d", ceilingLow)
	}

	// Host 16 GB (16384 MB) -> 75% = 12288 MB
	ceilingNormal := domain.CalculateHostMaxAllowedRAM(16384)
	if ceilingNormal != 12288 {
		t.Errorf("esperado 12288 MB para host de 16 GB, obtenido: %d", ceilingNormal)
	}

	// Dentro del límite
	if err := domain.ValidateRamAgainstHost(2048, ceilingNormal); err != nil {
		t.Errorf("error inesperado validando RAM válida: %v", err)
	}

	// Supera el límite
	if err := domain.ValidateRamAgainstHost(15000, ceilingNormal); !errors.Is(err, domain.ErrRamExceedsHostCapacity) {
		t.Errorf("esperado ErrRamExceedsHostCapacity, obtenido: %v", err)
	}
}
