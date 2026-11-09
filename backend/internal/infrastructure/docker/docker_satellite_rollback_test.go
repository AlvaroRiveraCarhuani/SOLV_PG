package docker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
)

type fakeSatelliteRemover struct {
	containers   []string
	volumes      []string
	networks     []string
	containerErr error
	volumeErr    error
	networkErr   error
}

func (f *fakeSatelliteRemover) ContainerRemove(_ context.Context, id string, _ container.RemoveOptions) error {
	f.containers = append(f.containers, id)
	return f.containerErr
}

func (f *fakeSatelliteRemover) VolumeRemove(_ context.Context, id string, _ bool) error {
	f.volumes = append(f.volumes, id)
	return f.volumeErr
}

func (f *fakeSatelliteRemover) NetworkRemove(_ context.Context, id string) error {
	f.networks = append(f.networks, id)
	return f.networkErr
}

func TestRollbackSatelliteResourcesRemovesOnlyCreatedResources(t *testing.T) {
	cause := errors.New("start failed")
	fake := &fakeSatelliteRemover{}
	err := rollbackSatelliteResources(context.Background(), fake, "container-1", "volume-1", true, "network-1", true, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("rollback no conservó la causa: %v", err)
	}
	if len(fake.containers) != 1 || fake.containers[0] != "container-1" {
		t.Fatalf("contenedor no revertido: %v", fake.containers)
	}
	if len(fake.volumes) != 1 || fake.volumes[0] != "volume-1" {
		t.Fatalf("volumen no revertido: %v", fake.volumes)
	}
	if len(fake.networks) != 1 || fake.networks[0] != "network-1" {
		t.Fatalf("red no revertida: %v", fake.networks)
	}
}

func TestRollbackSatelliteResourcesLeavesPreExistingResources(t *testing.T) {
	fake := &fakeSatelliteRemover{}
	if err := rollbackSatelliteResources(context.Background(), fake, "existing-container", "existing-volume", false, "existing-network", false, errors.New("ready failed")); err == nil {
		t.Fatal("se esperaba el error original")
	}
	if len(fake.volumes) != 0 || len(fake.networks) != 0 {
		t.Fatalf("se tocaron recursos preexistentes: volumes=%v networks=%v", fake.volumes, fake.networks)
	}
}

func TestRollbackSatelliteResourcesReportsCleanupFailure(t *testing.T) {
	fake := &fakeSatelliteRemover{volumeErr: errors.New("volume busy")}
	err := rollbackSatelliteResources(context.Background(), fake, "", "volume-1", true, "network-1", true, errors.New("create failed"))
	if err == nil || !strings.Contains(err.Error(), "rollback de satélite falló") || !strings.Contains(err.Error(), "volume busy") {
		t.Fatalf("no se reportó el fallo de limpieza: %v", err)
	}
}
