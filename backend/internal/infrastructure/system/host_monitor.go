package system

import (
	"fmt"
	"log"

	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
)

type GopsutilHostMonitor struct {
	minFreePct float64
}

func NewGopsutilHostMonitor(minFreePct float64) domain.HostMonitor {
	if minFreePct <= 0 {
		minFreePct = domain.MinHostFreeRAMPct // 15% por defecto
	}
	return &GopsutilHostMonitor{
		minFreePct: minFreePct,
	}
}

func (h *GopsutilHostMonitor) GetHostMemoryStats() (freePct float64, availableMB uint64, err error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read virtual memory via gopsutil: %w", err)
	}

	total := v.Total
	if total == 0 {
		return 0, 0, fmt.Errorf("total host memory is 0")
	}

	available := v.Available
	freePct = (float64(available) / float64(total)) * 100.0
	availableMB = available / (1024 * 1024)

	return freePct, availableMB, nil
}

func (h *GopsutilHostMonitor) CanAllocateMemory(requiredMB int64) bool {
	freePct, availableMB, err := h.GetHostMemoryStats()
	if err != nil {
		log.Printf("Warning: Host resource monitor failed: %v", err)
		return false
	}

	if freePct < h.minFreePct {
		log.Printf("Admission Control Triggered: Host available RAM is %.2f%% (< %.2f%% threshold). Available: %d MB", freePct, h.minFreePct, availableMB)
		return false
	}

	if int64(availableMB) < requiredMB {
		log.Printf("Admission Control Triggered: Required %d MB exceeds available %d MB", requiredMB, availableMB)
		return false
	}

	return true
}
