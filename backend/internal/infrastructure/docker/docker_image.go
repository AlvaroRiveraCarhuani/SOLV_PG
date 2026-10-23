package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"

	"solv-backend/internal/core/domain"
)

// InspectLocalImage comprueba si una imagen reside físicamente en el almacenamiento local del host Docker
func (c *Client) InspectLocalImage(ctx context.Context, imageRef string) (bool, int64, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageRef)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, inspect.Size, nil
}

// PullImage descarga una imagen con deduplicación single-flight para evitar peticiones concurrentes duplicadas
func (c *Client) PullImage(ctx context.Context, imageRef string) error {
	_, err, _ := c.pullGroup.Do(imageRef, func() (interface{}, error) {
		out, pullErr := c.cli.ImagePull(ctx, imageRef, image.PullOptions{})
		if pullErr != nil {
			return nil, fmt.Errorf("failed to pull image %q: %w", imageRef, pullErr)
		}
		defer func() { _ = out.Close() }()
		_, _ = io.Copy(io.Discard, out)
		return nil, nil
	})
	return err
}

// PruneOrphans ejecuta la poda selectiva de emergencia (ADR-032,
// docker-prune): elimina contenedores gestionados por SOLV que ya no
// corren (exitados, fallidos, muertos) e imágenes huérfanas no etiquetadas,
// preservando los volúmenes nombrados con el trabajo de los estudiantes.
func (c *Client) PruneOrphans(ctx context.Context) (containersPruned int64, spaceReclaimedMB int64, err error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list docker containers for prune: %w", err)
	}

	for _, cnt := range containers {
		if cnt.State == "running" || cnt.State == "paused" {
			continue
		}
		isSolvManaged := cnt.Labels["solv.managed"] == "true"
		if !isSolvManaged {
			for _, name := range cnt.Names {
				if strings.Contains(name, "solv-workspace-") {
					isSolvManaged = true
					break
				}
			}
		}
		if !isSolvManaged {
			continue
		}
		if err := c.cli.ContainerRemove(ctx, cnt.ID, container.RemoveOptions{Force: true}); err != nil {
			log.Printf("[EMERGENCY-PRUNE] No se pudo eliminar el contenedor %s: %v", cnt.ID[:12], err)
			continue
		}
		containersPruned++
	}

	// Imágenes huérfanas: solo dangling (sin etiqueta), jamás las de plantilla en uso.
	pruneFilters := filters.NewArgs(filters.Arg("dangling", "true"))
	pruneReport, err := c.cli.ImagesPrune(ctx, pruneFilters)
	if err != nil {
		return containersPruned, spaceReclaimedMB, fmt.Errorf("failed to prune dangling images: %w", err)
	}
	spaceReclaimedMB = domain.Uint64ToInt64(pruneReport.SpaceReclaimed / (1024 * 1024))

	return containersPruned, spaceReclaimedMB, nil
}
