package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types"

	"solv-backend/internal/core/domain"
)

const bytesPerMB = int64(1024 * 1024)

// ListDanglingEnvironmentVolumes devuelve los volúmenes sin contenedor con su
// tamaño, en una sola lectura del daemon. Equivale a la columna `LINKS = 0` de
// `docker system df -v`, pero sin abrir un contenedor por volumen.
//
// UsageData llega con RefCount -1 cuando el daemon no puede calcular la
// referencia: ese volumen no se ofrece para liberar, porque no se pudo
// comprobar que esté libre.
func (c *Client) ListDanglingEnvironmentVolumes(ctx context.Context) ([]domain.DanglingVolume, error) {
	usage, err := c.cli.DiskUsage(ctx, types.DiskUsageOptions{
		Types: []types.DiskUsageObject{types.VolumeObject},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read docker disk usage: %w", err)
	}

	volumes := make([]domain.DanglingVolume, 0, len(usage.Volumes))
	for _, volume := range usage.Volumes {
		if volume == nil || volume.UsageData == nil {
			continue
		}
		if volume.UsageData.RefCount != 0 {
			continue
		}
		sizeMB := int64(0)
		if volume.UsageData.Size > 0 {
			sizeMB = volume.UsageData.Size / bytesPerMB
			if volume.UsageData.Size%bytesPerMB != 0 {
				sizeMB++
			}
		}
		volumes = append(volumes, domain.DanglingVolume{Name: volume.Name, SizeMB: sizeMB})
	}

	sort.Slice(volumes, func(i, j int) bool { return volumes[i].Name < volumes[j].Name })
	return volumes, nil
}

// RemoveEnvironmentVolume borra un volumen del workspace. force=false: si un
// contenedor lo montó entre el inventario y el borrado, el daemon rechaza la
// operación en lugar de destruir trabajo en curso.
func (c *Client) RemoveEnvironmentVolume(ctx context.Context, name string) error {
	if err := domain.ValidateOrphanVolumeName(name); err != nil {
		return err
	}
	if err := c.cli.VolumeRemove(ctx, name, false); err != nil {
		return fmt.Errorf("failed to remove environment volume %s: %w", name, err)
	}
	return nil
}
