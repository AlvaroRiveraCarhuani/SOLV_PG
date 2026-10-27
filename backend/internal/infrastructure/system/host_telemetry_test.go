package system

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

type fakeDockerRuntime struct {
	snapshot domain.DockerSnapshot
	err      error
}

func (f *fakeDockerRuntime) Info(context.Context) (domain.DockerSnapshot, error) {
	if f.err != nil {
		return domain.DockerSnapshot{}, f.err
	}
	return f.snapshot, nil
}

func TestSystemTelemetrySnapshotReadsHostMemory(t *testing.T) {
	telemetry := NewSystemTelemetry("/", nil)

	snapshot, err := telemetry.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot.RAMTotalBytes == 0 {
		t.Fatal("expected a non-zero host memory total")
	}
	if snapshot.RAMAvailableBytes > snapshot.RAMTotalBytes {
		t.Fatalf("available memory %d exceeds total %d", snapshot.RAMAvailableBytes, snapshot.RAMTotalBytes)
	}
	if snapshot.RAMPercent <= 0 {
		t.Fatalf("expected a positive memory percentage, got %f", snapshot.RAMPercent)
	}
}

func TestSystemTelemetryReportsMeasuredDiskPath(t *testing.T) {
	telemetry := NewSystemTelemetry("/var/lib/docker", nil)

	snapshot, err := telemetry.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot.Disk.Path != "/var/lib/docker" {
		t.Fatalf("expected measured disk path %q, got %q", "/var/lib/docker", snapshot.Disk.Path)
	}
	if snapshot.Disk.TotalBytes == 0 {
		t.Fatal("expected a non-zero disk total for an existing path")
	}
}

func TestSystemTelemetryPropagatesDockerInfo(t *testing.T) {
	telemetry := NewSystemTelemetry("/", &fakeDockerRuntime{
		snapshot: domain.DockerSnapshot{
			ServerVersion: "27.1.1",
			StorageDriver: "overlay2",
			Healthy:       true,
		},
	})

	snapshot, err := telemetry.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !snapshot.Docker.Healthy {
		t.Fatal("expected docker to be reported healthy")
	}
	if snapshot.Docker.ServerVersion != "27.1.1" || snapshot.Docker.StorageDriver != "overlay2" {
		t.Fatalf("unexpected docker snapshot: %+v", snapshot.Docker)
	}
}

// Un daemon inaccesible no puede invalidar la lectura de hardware: el panel
// debe seguir mostrando RAM/CPU/disco y marcar Docker como no saludable.
func TestSystemTelemetryDegradesWhenDockerIsUnreachable(t *testing.T) {
	telemetry := NewSystemTelemetry("/", &fakeDockerRuntime{
		err: errors.New("cannot connect to the docker daemon"),
	})

	snapshot, err := telemetry.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("docker failure must not abort the snapshot: %v", err)
	}
	if snapshot.Docker.Healthy {
		t.Fatal("expected docker to be reported unhealthy")
	}
	if snapshot.RAMTotalBytes == 0 {
		t.Fatal("expected host memory to still be reported")
	}
}
