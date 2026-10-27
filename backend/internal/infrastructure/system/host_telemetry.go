package system

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

// SystemTelemetry implementa domain.HostTelemetry leyendo el host fisico via
// gopsutil. La ruta de disco se inyecta para que el valor medido y la ruta
// informada sean siempre el mismo dato.
type SystemTelemetry struct {
	diskPath string
	docker   domain.DockerRuntime
}

func NewSystemTelemetry(diskPath string, docker domain.DockerRuntime) domain.HostTelemetry {
	if diskPath == "" {
		diskPath = "/"
	}
	return &SystemTelemetry{
		diskPath: diskPath,
		docker:   docker,
	}
}

var _ domain.HostTelemetry = (*SystemTelemetry)(nil)

// Snapshot devuelve la lectura del host. Solo falla si no puede leerse la
// memoria del host: sin ese dato el resto de la telemetria no es interpretable.
// El disco y el daemon de Docker degradan a cero / no saludable sin abortar.
func (s *SystemTelemetry) Snapshot(ctx context.Context) (domain.HostTelemetrySnapshot, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return domain.HostTelemetrySnapshot{}, fmt.Errorf("failed to read host memory: %w", err)
	}
	if v.Total == 0 {
		return domain.HostTelemetrySnapshot{}, fmt.Errorf("host total memory is 0")
	}

	snapshot := domain.HostTelemetrySnapshot{
		RAMUsedBytes:      v.Used,
		RAMTotalBytes:     v.Total,
		RAMAvailableBytes: v.Available,
		RAMPercent:        v.UsedPercent,
		Disk: domain.DiskSnapshot{
			Path: s.diskPath,
		},
	}

	if cores, coreErr := cpu.Counts(true); coreErr == nil {
		snapshot.CPUCores = cores
	}
	// El primer cpu.Percent(0, false) devuelve 0 porque no hay delta previo;
	// es una lectura instantanea aceptable para un panel de estado actual.
	if percents, cpuErr := cpu.Percent(0, false); cpuErr == nil && len(percents) > 0 {
		snapshot.CPUPercent = percents[0]
	}

	if usage, diskErr := disk.Usage(s.diskPath); diskErr == nil && usage != nil {
		snapshot.Disk.UsedBytes = usage.Used
		snapshot.Disk.TotalBytes = usage.Total
		snapshot.Disk.Percent = usage.UsedPercent
	}

	if u, uptimeErr := host.Uptime(); uptimeErr == nil {
		snapshot.UptimeSeconds = u
	}

	if s.docker != nil {
		if info, dockerErr := s.docker.Info(ctx); dockerErr == nil {
			snapshot.Docker = info
		}
	}

	return snapshot, nil
}
