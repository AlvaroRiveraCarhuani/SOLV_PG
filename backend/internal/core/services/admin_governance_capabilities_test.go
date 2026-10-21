package services_test

import (
	"context"
	"errors"
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
			if s.BaseRAMMB != 50 {
				t.Errorf("BaseRAMMB de postgres debe ser 50 según ADR-045, obtenido: %d", s.BaseRAMMB)
			}
			if s.Version != "18" {
				t.Errorf("versión de postgres debe ser 18 (resuelta en vivo), obtenido: %s", s.Version)
			}
		}
		if s.Engine == "mysql" && s.IsAvailable {
			t.Error("MySQL no debe figurar disponible por defecto cuando MYSQL_SATELLITE_HOST no está configurado")
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
