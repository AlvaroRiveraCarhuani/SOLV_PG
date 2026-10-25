package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

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
}

type HealthMetricsResponse struct {
	TenantID               string                  `json:"tenant_id"`
	RunningLabs            int                     `json:"running_labs"`
	HibernatedLabs         int                     `json:"hibernated_labs"`
	OOMKilledLabs          int                     `json:"oom_killed_labs"`
	TotalRAMAllocMB        int64                   `json:"total_ram_alloc_mb"`
	HealthStatus           string                  `json:"health_status"`
	UptimeSeconds          uint64                  `json:"uptime_seconds"`
	HostHardware           *HostHardwareMetricsDTO `json:"host_hardware,omitempty"`
	LoadHistory            []LoadSnapshot          `json:"load_history"`
	HostSafetyCeilingPct   int                     `json:"host_safety_ceiling_pct"`
	EstimatedMaxContainers int                     `json:"estimated_max_containers"`
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

	runningCount := 0
	hibernatedCount := 0
	oomCount := 0
	var totalRAM int64 = 0

	for _, ws := range allRunning {
		if ws.TenantID == tenantID {
			switch ws.Status {
			case "running":
				runningCount++
				totalRAM += ws.MemoryLimitMB
			case "hibernated":
				hibernatedCount++
			}
			if ws.Status == "failed" || ws.Status == "oom_killed" || ws.OOMStrikeCount > 0 {
				oomCount++
			}
		}
	}

	ceilingPct := 85
	if h.tenantRepo != nil {
		if tenant, err := h.tenantRepo.GetByID(r.Context(), tenantID); err == nil && tenant != nil && len(tenant.Config) > 0 {
			var configMap map[string]json.RawMessage
			if err := json.Unmarshal(tenant.Config, &configMap); err == nil {
				if raw, ok := configMap["server_policies"]; ok {
					var pol domain.ServerPolicies
					if err := json.Unmarshal(raw, &pol); err == nil && pol.HostSafetyCeilingPct > 0 {
						ceilingPct = pol.HostSafetyCeilingPct
					}
				}
			}
		}
	}

	totalHostMB := uint64(8192)
	var hostHardware *HostHardwareMetricsDTO
	if v, memErr := mem.VirtualMemory(); memErr == nil {
		if v.Total > 0 {
			totalHostMB = v.Total / (1024 * 1024)
		}
		cores, _ := cpu.Counts(true)
		cpuPercents, _ := cpu.Percent(0, false)
		cpuVal := 0.0
		if len(cpuPercents) > 0 {
			cpuVal = cpuPercents[0]
		}
		diskUsage, _ := disk.Usage("/")
		diskUsed := uint64(0)
		diskTotal := uint64(0)
		diskPct := 0.0
		if diskUsage != nil {
			diskUsed = diskUsage.Used
			diskTotal = diskUsage.Total
			diskPct = diskUsage.UsedPercent
		}

		hostHardware = &HostHardwareMetricsDTO{
			RAMUsedBytes:   v.Used,
			RAMTotalBytes:  v.Total,
			RAMPercent:     v.UsedPercent,
			CPUCores:       cores,
			CPUPercent:     cpuVal,
			DiskUsedBytes:  diskUsed,
			DiskTotalBytes: diskTotal,
			DiskPercent:    diskPct,
		}
	}

	allocatableMB := float64(totalHostMB) * float64(ceilingPct) / 100.0
	avgMB := 512.0
	if runningCount > 0 && totalRAM > 0 {
		avgMB = float64(totalRAM) / float64(runningCount)
	}
	estimated := int(allocatableMB / avgMB)
	minContainers := max(runningCount, 1)
	if estimated < minContainers {
		estimated = minContainers
	}

	var uptimeSec uint64 = 0
	if u, err := host.Uptime(); err == nil {
		uptimeSec = u
	}

	resp := HealthMetricsResponse{
		TenantID:               tenantID,
		RunningLabs:            runningCount,
		HibernatedLabs:         hibernatedCount,
		OOMKilledLabs:          oomCount,
		TotalRAMAllocMB:        totalRAM,
		HealthStatus:           "healthy",
		UptimeSeconds:          uptimeSec,
		HostHardware:           hostHardware,
		LoadHistory:            h.getHistorySnapshots(),
		HostSafetyCeilingPct:   ceilingPct,
		EstimatedMaxContainers: estimated,
	}

	if oomCount > 5 {
		resp.HealthStatus = "warning"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
