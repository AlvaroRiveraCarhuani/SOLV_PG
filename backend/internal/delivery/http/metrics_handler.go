package httpdelivery

import (
	"fmt"
	"net/http"
	"strconv"
	"os"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type MetricsHandler struct {
	workspaceRepo   domain.WorkspaceRepository
	hostMonitor     domain.HostMonitor
	zombieCollector *services.ZombieCollectorWorker
}

func NewMetricsHandler(workspaceRepo domain.WorkspaceRepository, hostMonitor domain.HostMonitor, zombieCollector *services.ZombieCollectorWorker) *MetricsHandler {
	return &MetricsHandler{
		workspaceRepo:   workspaceRepo,
		hostMonitor:     hostMonitor,
		zombieCollector: zombieCollector,
	}
}

func (h *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Obtener workspaces activos
	activeWorkspaces, err := h.workspaceRepo.GetActiveWorkspaces(ctx)
	activeCount := 0
	if err == nil {
		activeCount = len(activeWorkspaces)
	}

	// 2. Obtener métricas del Host
	_, availMB, err := h.hostMonitor.GetHostMemoryStats()
	availBytes := uint64(0)
	if err == nil {
		availBytes = availMB * 1024 * 1024
	}

	// 3. Obtener OOM_GUARD_MB
	oomGuardMB := uint64(1408)
	if envGuard := os.Getenv("OOM_GUARD_MB"); envGuard != "" {
		if val, parseErr := strconv.ParseUint(envGuard, 10, 64); parseErr == nil {
			oomGuardMB = val
		}
	}
	oomGuardBytes := oomGuardMB * 1024 * 1024

	// 4. Obtener conteo de contenedores huérfanos reclamados
	reclaimedCount := uint64(0)
	if h.zombieCollector != nil {
		reclaimedCount = h.zombieCollector.GetReclaimedCount()
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "# HELP solv_active_workspaces_total Total number of active running/pending student workspaces\n")
	fmt.Fprintf(w, "# TYPE solv_active_workspaces_total gauge\n")
	fmt.Fprintf(w, "solv_active_workspaces_total %d\n\n", activeCount)

	fmt.Fprintf(w, "# HELP solv_host_available_memory_bytes Physical host available RAM in bytes\n")
	fmt.Fprintf(w, "# TYPE solv_host_available_memory_bytes gauge\n")
	fmt.Fprintf(w, "solv_host_available_memory_bytes %d\n\n", availBytes)

	fmt.Fprintf(w, "# HELP solv_host_oom_guard_bytes Calibrated Node Allocatable OOM Guard threshold in bytes\n")
	fmt.Fprintf(w, "# TYPE solv_host_oom_guard_bytes gauge\n")
	fmt.Fprintf(w, "solv_host_oom_guard_bytes %d\n\n", oomGuardBytes)

	fmt.Fprintf(w, "# HELP solv_orphan_containers_reclaimed_total Total number of orphan/zombie containers reclaimed by collector\n")
	fmt.Fprintf(w, "# TYPE solv_orphan_containers_reclaimed_total counter\n")
	fmt.Fprintf(w, "solv_orphan_containers_reclaimed_total %d\n", reclaimedCount)
}
