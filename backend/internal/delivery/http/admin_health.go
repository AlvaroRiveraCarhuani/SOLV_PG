package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
)

type HostHardwareMetricsDTO struct {
	RAMUsedBytes   uint64  `json:"ram_used_bytes"`
	RAMTotalBytes  uint64  `json:"ram_total_bytes"`
	RAMPercent     float64 `json:"ram_percent"`
	CPUCores       int     `json:"cpu_cores"`
	CPUPercent     float64 `json:"cpu_percent"`
	DiskUsedBytes  uint64  `json:"disk_used_bytes"`
	DiskTotalBytes uint64  `json:"disk_total_bytes"`
	DiskPercent    float64 `json:"disk_percent"`
	DiskPath       string  `json:"disk_path"`
}

// DockerStatusDTO expone el daemon real del nodo. Hasta ahora la interfaz
// mostraba un literal fijo que nunca reflejaba el servidor en produccion.
type DockerStatusDTO struct {
	ServerVersion string `json:"server_version"`
	StorageDriver string `json:"storage_driver"`
	Healthy       bool   `json:"healthy"`
}

type HealthMetricsResponse struct {
	TenantID               string                  `json:"tenant_id"`
	RunningLabs            int                     `json:"running_labs"`
	HibernatedLabs         int                     `json:"hibernated_labs"`
	OOMKilledLabs          int                     `json:"oom_killed_labs"`
	UnknownLabs            int                     `json:"unknown_labs"`
	OrphanedLabs           int                     `json:"orphaned_labs"`
	TotalRAMAllocMB        int64                   `json:"total_ram_alloc_mb"`
	HealthStatus           string                  `json:"health_status"`
	HealthReasons          []string                `json:"health_reasons,omitempty"`
	UptimeSeconds          uint64                  `json:"uptime_seconds"`
	HostHardware           *HostHardwareMetricsDTO `json:"host_hardware,omitempty"`
	Docker                 *DockerStatusDTO        `json:"docker,omitempty"`
	LoadHistory            []LoadSnapshot          `json:"load_history"`
	HostSafetyCeilingPct   int                     `json:"host_safety_ceiling_pct"`
	EstimatedMaxContainers int                     `json:"estimated_max_containers"`
	CapacityBasis          string                  `json:"capacity_basis"`
	MaxContainersPolicy    int                     `json:"max_containers_policy"`
	RAMPerLabMB            int                     `json:"ram_per_lab_mb"`
}

func (h *AdminHandler) GetHealthMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	allRunning, err := h.workspaceRepo.GetAllRunningWorkspaces(r.Context())
	if err != nil {
		allRunning = []*domain.WorkspaceInstance{}
	}

	// El listado de activos solo puede aportar lo que ya filtra: estado
	// 'running'. Los hibernados y los fallidos se cuentan por su propio
	// agregado, porque preguntarle "hibernated" a una lista de running siempre
	// devolvia cero.
	runningCount := 0
	oomStrikeCount := 0
	var totalRAM int64 = 0
	for _, ws := range allRunning {
		if ws.TenantID != tenantID {
			continue
		}
		runningCount++
		totalRAM += ws.MemoryLimitMB
		if ws.OOMStrikeCount > 0 {
			oomStrikeCount++
		}
	}

	statusCounts := h.workspaceStatusCounts(r.Context(), tenantID)
	hibernatedCount := statusCounts["hibernated"]
	unknownCount := statusCounts[domain.WorkspaceStatusUnknown]
	orphanedCount := statusCounts[domain.WorkspaceStatusOrphaned]
	oomCount := oomStrikeCount + statusCounts["failed"] + statusCounts["oom_killed"]

	// Las politicas se leen por el servicio que ya usa la admision, no por un
	// parseo paralelo: el panel debe mostrar exactamente el techo que se aplica.
	policies := domain.DefaultServerPolicies()
	if h.policiesService != nil {
		if resolved, polErr := h.policiesService.Get(r.Context(), tenantID); polErr == nil && resolved != nil {
			policies = *resolved
		}
	}
	ceilingPct := policies.HostSafetyCeilingPct

	// Telemetria del host detras del puerto de dominio: la capa de entrega ya no
	// lee /proc ni consulta Docker por su cuenta.
	var hostHardware *HostHardwareMetricsDTO
	var dockerStatus *DockerStatusDTO
	var availableMB uint64
	var uptimeSec uint64
	var ramPercent float64

	if h.telemetry != nil {
		if snapshot, snapErr := h.telemetry.Snapshot(r.Context()); snapErr == nil {
			hostHardware = &HostHardwareMetricsDTO{
				RAMUsedBytes:   snapshot.RAMUsedBytes,
				RAMTotalBytes:  snapshot.RAMTotalBytes,
				RAMPercent:     snapshot.RAMPercent,
				CPUCores:       snapshot.CPUCores,
				CPUPercent:     snapshot.CPUPercent,
				DiskUsedBytes:  snapshot.Disk.UsedBytes,
				DiskTotalBytes: snapshot.Disk.TotalBytes,
				DiskPercent:    snapshot.Disk.Percent,
				DiskPath:       snapshot.Disk.Path,
			}
			dockerStatus = &DockerStatusDTO{
				ServerVersion: snapshot.Docker.ServerVersion,
				StorageDriver: snapshot.Docker.StorageDriver,
				Healthy:       snapshot.Docker.Healthy,
			}
			availableMB = snapshot.RAMAvailableBytes
			uptimeSec = snapshot.UptimeSeconds
			ramPercent = snapshot.RAMPercent
		}
	}

	// Fallback de capacidad: sin fuente de telemetria inyectada se conserva el
	// monitor de admision, que mide la misma RAM disponible.
	if availableMB == 0 && h.hostMonitor != nil {
		if _, avail, monErr := h.hostMonitor.GetHostMemoryStats(); monErr == nil {
			availableMB = avail
		}
	}

	// Capacidad real: compiten la RAM disponible medida y el techo de politica
	// del tenant. El calculo anterior usaba el techo absoluto de RAM sin
	// descontar el uso ya presente e ignoraba max_containers, que es el limite
	// que la admision aplica de verdad al crear un laboratorio.
	estimated, capacityBasis := domain.EstimateCapacity(
		availableMB, policies.RAMLimitMB, policies.MaxContainers, runningCount,
	)

	// Salud real con motivo (S6): el veredicto sale de hechos medidos, no de un
	// contador heuristico. Docker caido u huerfanos deriva en critical; OOM,
	// estados sin verificar o RAM sobre el techo de seguridad, en degraded. Sin
	// telemetria la RAM medida queda en 0: no falsea un techo superado.
	dockerHealthy := true
	if dockerStatus != nil {
		dockerHealthy = dockerStatus.Healthy
	}
	healthStatus, healthReasons := domain.EvaluateHealth(domain.HealthEvaluationInput{
		DockerHealthy:    dockerHealthy,
		OrphanedLabs:     orphanedCount,
		OOMKilledLabs:    oomCount,
		UnknownLabs:      unknownCount,
		RAMPercent:       ramPercent,
		SafetyCeilingPct: ceilingPct,
	})

	resp := HealthMetricsResponse{
		TenantID:               tenantID,
		RunningLabs:            runningCount,
		HibernatedLabs:         hibernatedCount,
		OOMKilledLabs:          oomCount,
		UnknownLabs:            unknownCount,
		OrphanedLabs:           orphanedCount,
		TotalRAMAllocMB:        totalRAM,
		HealthStatus:           healthStatus,
		HealthReasons:          healthReasons,
		UptimeSeconds:          uptimeSec,
		HostHardware:           hostHardware,
		Docker:                 dockerStatus,
		LoadHistory:            h.getHistorySnapshots(r.Context(), r.URL.Query().Get("window"), tenantID),
		HostSafetyCeilingPct:   ceilingPct,
		EstimatedMaxContainers: estimated,
		CapacityBasis:          capacityBasis,
		MaxContainersPolicy:    policies.MaxContainers,
		RAMPerLabMB:            policies.RAMLimitMB,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// workspaceStatusCounts devuelve el desglose por estado del tenant. Si el
// repositorio no implementa el agregado, el panel degrada a cero en vez de
// fallar: el desglose es informativo y no debe bloquear la lectura de hardware.
func (h *AdminHandler) workspaceStatusCounts(ctx context.Context, tenantID string) map[string]int {
	counter, ok := h.workspaceRepo.(workspaceStatusCounter)
	if !ok || counter == nil {
		return nil
	}
	counts, err := counter.CountWorkspacesByStatus(ctx, tenantID)
	if err != nil {
		return nil
	}
	return counts
}
