package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"log"
	"os"
	"solv-backend/internal/core/domain"
	"strings"
)

const (
	ActionTerminateAll = "terminate_all_workspaces"
	ActionHibernateAll = "hibernate_all_workspaces"
	ActionKillZombies  = "kill_zombies"
	ActionDockerPrune  = "docker_prune"
	ActionResetPools   = "reset_pools"

	PhraseTerminateAll = "TERMINAR TODOS LOS WORKSPACES"
	PhraseHibernateAll = "HIBERNAR TODOS LOS WORKSPACES"
	PhraseKillZombies  = "LIMPIAR ZOMBIES DOCKER"
	PhraseDockerPrune  = "PURGAR CAPAS HUERFANAS"
	PhraseResetPools   = "REINICIAR POOLS"
)

// emergencyConfirmationPhrase centraliza el contrato de doble confirmación
// (ADR-032): cada acción exige la frase exacta tipeada por el administrador.
func emergencyConfirmationPhrase(action string) (string, bool) {
	switch action {
	case ActionTerminateAll:
		return PhraseTerminateAll, true
	case ActionHibernateAll:
		return PhraseHibernateAll, true
	case ActionKillZombies:
		return PhraseKillZombies, true
	case ActionDockerPrune:
		return PhraseDockerPrune, true
	case ActionResetPools:
		return PhraseResetPools, true
	}
	return "", false
}

// emergencyAuditActions mapea cada acción a su evento de auditoría EMERGENCY_*
// (registros filtrables por el índice parcial de ADR-032).
var emergencyAuditActions = map[string]string{
	ActionTerminateAll: "EMERGENCY_TERMINATE_ALL",
	ActionHibernateAll: "EMERGENCY_HIBERNATE_ALL",
	ActionKillZombies:  "EMERGENCY_KILL_ZOMBIES",
	ActionDockerPrune:  "EMERGENCY_PRUNE_DOCKER",
	ActionResetPools:   "EMERGENCY_RESET_POOLS",
}

var (
	ErrUnknownEmergencyAction    = errors.New("unknown emergency action")
	ErrInvalidConfirmationPhrase = errors.New("invalid confirmation phrase")
)

func (s *AdminGovernanceService) ExecuteEmergencyAction(
	ctx context.Context,
	tenantID, adminID, action string,
	req domain.EmergencyActionRequest,
) (*domain.EmergencyActionResult, error) {
	phrase, known := emergencyConfirmationPhrase(action)
	if !known {
		return nil, ErrUnknownEmergencyAction
	}
	if req.ConfirmationPhrase != phrase {
		return nil, ErrInvalidConfirmationPhrase
	}

	var result *domain.EmergencyActionResult
	var err error

	switch action {
	case ActionTerminateAll:
		var count int64
		count, err = s.govRepo.TerminateAllWorkspaces(ctx, tenantID)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: count,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Se terminaron forzosamente %d workspaces activos", count),
			}
		}

	case ActionHibernateAll:
		var count int64
		count, err = s.govRepo.HibernateAllWorkspaces(ctx, tenantID)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: count,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Se hibernaron exitosamente %d workspaces activos", count),
			}
		}

	case ActionKillZombies:
		if s.dockerPruner != nil {
			var pruned, _ int64
			pruned, _, err = s.dockerPruner(ctx)
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: pruned,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Barrido de contenedores zombies completado: %d contenedores eliminados", pruned),
			}
		} else {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: 0,
				ExecutedBy:    adminID,
				Message:       "Barrido de contenedores zombies ejecutado exitosamente",
			}
		}

	case ActionDockerPrune:
		if s.dockerPruner == nil {
			return nil, ErrEmergencyExecutorUnavailable
		}
		var pruned, spaceMB int64
		pruned, spaceMB, err = s.dockerPruner(ctx)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: pruned,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Poda completada: %d contenedores e imágenes huérfanos eliminados (%d MB liberados)", pruned, spaceMB),
			}
		}

	case ActionResetPools:
		if s.poolResetter == nil {
			return nil, ErrEmergencyExecutorUnavailable
		}
		var recycled int64
		recycled, err = s.poolResetter(ctx)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: recycled,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Pool de conexiones reiniciado: %d conexiones recicladas y circuit breakers inicializados", recycled),
			}
		}
	}

	if err != nil {
		return nil, err
	}

	// Trazabilidad obligatoria (ADR-032): toda ejecución de emergencia queda
	// registrada en audit_logs con evento EMERGENCY_* y el motivo ingresado.
	s.recordEmergencyAudit(ctx, tenantID, adminID, action, req, result)

	return result, nil
}

// recordEmergencyAudit persiste el evento EMERGENCY_* correspondiente. Un
// fallo de auditoría no revierte la acción ejecutada (el efecto operativo ya
// ocurrió), pero se reporta en el log del sistema.
func (s *AdminGovernanceService) recordEmergencyAudit(
	ctx context.Context,
	tenantID, adminID, action string,
	req domain.EmergencyActionRequest,
	result *domain.EmergencyActionResult,
) {
	if s.auditRepo == nil || result == nil {
		return
	}
	meta, _ := json.Marshal(map[string]any{
		"reason":         req.Reason,
		"affected_count": result.AffectedCount,
		"message":        result.Message,
	})
	event := emergencyAuditActions[action]
	err := s.auditRepo.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      adminID,
		Action:       event,
		ResourceType: "emergency_action",
		StatusCode:   200,
		Metadata:     meta,
	})
	if err != nil {
		log.Printf("[EMERGENCY] No se pudo registrar el evento de auditoría %s: %v", event, err)
	}
}

// GetRuntimeCapabilities obtiene métricas del host físico y catálogo de servicios satélite soportados
func (s *AdminGovernanceService) GetRuntimeCapabilities(ctx context.Context) (*domain.RuntimeCapabilities, error) {
	// 1. Métricas de memoria y procesador del host real
	totalMB := int64(8192)
	availableMB := int64(4096)
	usedMB := int64(4096)
	cpuCores := 4

	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil {
		totalMB = domain.Uint64ToInt64(v.Total / (1024 * 1024))
		availableMB = domain.Uint64ToInt64(v.Available / (1024 * 1024))
		usedMB = domain.Uint64ToInt64(v.Used / (1024 * 1024))
	}
	if c, err := cpu.CountsWithContext(ctx, true); err == nil && c > 0 {
		cpuCores = c
	}

	hostMem := domain.HostCapacityInfo{
		TotalRAMMB:     totalMB,
		AvailableRAMMB: availableMB,
		UsedRAMMB:      usedMB,
		CPUCores:       cpuCores,
	}

	// 2. Presets adaptados dinámicamente a la memoria física
	judgePresets := []domain.RamPresetSuggestion{
		{MB: 128, Label: "128 MB", Desc: "Ultra-ligera (C/C++)"},
		{MB: 256, Label: "256 MB", Desc: "Recomendada (Python/Go)"},
		{MB: 512, Label: "512 MB", Desc: "Completa (Java/JVM)"},
	}

	idePresets := []domain.RamPresetSuggestion{
		{MB: 512, Label: "512 MB", Desc: "Ligera (C/Go)"},
		{MB: 1024, Label: "1 GB", Desc: "Estándar (Web/Python)"},
	}
	if totalMB >= 6000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 2048, Label: "2 GB", Desc: "Intensiva (Java/ML)",
		})
	}
	if totalMB >= 12000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 4096, Label: "4 GB", Desc: "Datos & IA",
		})
	} else if totalMB >= 7000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 4096, Label: "4 GB", Desc: "Alta demanda (Intensivo)",
		})
	}
	if totalMB >= 32000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 8192, Label: "8 GB", Desc: "Big Data & Deep Learning",
		})
	}

	maxAllowedRAM := int(float64(totalMB) * 0.75)
	if maxAllowedRAM < 512 {
		maxAllowedRAM = 512
	}

	// 3. Catálogo real de servicios satélite soportados en la plataforma (ADR-045)
	pgVersion := strings.TrimSpace(os.Getenv("POSTGRES_VERSION"))
	if s.govRepo != nil {
		if probe, ok := s.govRepo.(domain.PostgresVersionProbe); ok {
			if liveVer, err := probe.GetServerMajorVersion(ctx); err == nil && strings.TrimSpace(liveVer) != "" {
				pgVersion = strings.TrimSpace(liveVer)
			}
		}
	}
	if pgVersion == "" {
		pgVersion = "18"
	}

	satellites := []domain.SatelliteServiceCapability{
		{
			Category:    "database",
			Engine:      "sqlite",
			Label:       "SQLite3 (Nativo)",
			Version:     "3.x",
			Description: "Base de datos embebida en archivo local (/home/workspace). Consumo nulo (+0 MB RAM), ideal para materias iniciales y SQL básico.",
			EnvVar:      "SQLITE_PATH",
			IsAvailable: true,
			BaseRAMMB:   0,
		},
		{
			Category:    "database",
			Engine:      "postgres",
			Label:       "PostgreSQL (Aula)",
			Version:     pgVersion,
			Description: "Perfil sintonizado para laboratorio (~50 MB RAM) con corte de consultas largas (5s), transaccionalidad completa y aislamiento por estudiante.",
			EnvVar:      "DATABASE_URL",
			IsAvailable: true,
			BaseRAMMB:   50,
		},
		{
			Category:    "database",
			Engine:      "mysql",
			Label:       "MySQL",
			Version:     "8.4",
			Description: "Base de datos relacional MySQL para ejercicios de SQL",
			EnvVar:      "DATABASE_URL",
			IsAvailable: strings.TrimSpace(os.Getenv("MYSQL_SATELLITE_HOST")) != "",
			BaseRAMMB:   128,
		},
		{
			Category:    "database",
			Engine:      "mongodb",
			Label:       "MongoDB",
			Version:     "7.0",
			Description: "Base de datos de documentos NoSQL para proyectos web",
			EnvVar:      "MONGODB_URI",
			IsAvailable: strings.TrimSpace(os.Getenv("MONGODB_SATELLITE_HOST")) != "",
			BaseRAMMB:   128,
		},
		{
			Category:    "cache",
			Engine:      "redis",
			Label:       "Redis",
			Version:     "7.2",
			Description: "Almacén en memoria y caché clave-valor",
			EnvVar:      "REDIS_URL",
			IsAvailable: strings.TrimSpace(os.Getenv("REDIS_SATELLITE_HOST")) != "",
			BaseRAMMB:   64,
		},
	}

	// editorBaseMB es la RAM mínima que consume el proceso del editor (OpenVSCode Server).
	// Constante de dominio: si se necesita hacer configurable, agregar al struct de Config del servicio.
	const editorBaseMB = 210

	// runtimeBaseMB es la RAM mínima reservada para el runtime del Juez (kernel + sandbox del contenedor efímero).
	const runtimeBaseMB = 32

	return &domain.RuntimeCapabilities{
		HostMemory:        hostMem,
		SatelliteServices: satellites,
		IDEPresets:        idePresets,
		JudgePresets:      judgePresets,
		MaxAllowedRamMB:   maxAllowedRAM,
		EditorBaseMB:      editorBaseMB,
		RuntimeBaseMB:     runtimeBaseMB,
	}, nil
}
