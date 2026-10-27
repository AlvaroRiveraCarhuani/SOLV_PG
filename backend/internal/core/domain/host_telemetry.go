package domain

import "context"

// DiskSnapshot describe el uso del sistema de archivos que realmente se midio.
// Path viaja en la respuesta para que la interfaz no afirme una ruta distinta
// de la que el backend consulto.
type DiskSnapshot struct {
	Path       string
	UsedBytes  uint64
	TotalBytes uint64
	Percent    float64
}

// DockerSnapshot describe el daemon de Docker del nodo.
type DockerSnapshot struct {
	ServerVersion string
	StorageDriver string
	Healthy       bool
}

// HostTelemetrySnapshot es una lectura instantanea del servidor fisico.
type HostTelemetrySnapshot struct {
	RAMUsedBytes      uint64
	RAMTotalBytes     uint64
	RAMAvailableBytes uint64
	RAMPercent        float64
	CPUCores          int
	CPUPercent        float64
	Disk              DiskSnapshot
	Docker            DockerSnapshot
	UptimeSeconds     uint64
}

// HostTelemetry entrega la telemetria del host fisico sin que la capa de
// entrega dependa de una libreria concreta. El muestreo de Docker es
// best-effort: un daemon inaccesible deja Healthy en false y no invalida el
// resto de la lectura.
type HostTelemetry interface {
	Snapshot(ctx context.Context) (HostTelemetrySnapshot, error)
}

// DockerRuntime expone el estado del daemon de Docker del nodo.
type DockerRuntime interface {
	Info(ctx context.Context) (DockerSnapshot, error)
}
