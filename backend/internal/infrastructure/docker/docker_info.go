package docker

import (
	"context"
	"fmt"

	"solv-backend/internal/core/domain"
)

var _ domain.DockerRuntime = (*Client)(nil)

// Info implementa domain.DockerRuntime consultando el daemon real. La version
// y el driver se leen del servidor: hasta ahora la interfaz mostraba un
// literal fijo que nunca reflejaba el nodo en produccion.
func (c *Client) Info(ctx context.Context) (domain.DockerSnapshot, error) {
	info, err := c.cli.Info(ctx)
	if err != nil {
		return domain.DockerSnapshot{Healthy: false}, fmt.Errorf("failed to read docker server info: %w", err)
	}

	return domain.DockerSnapshot{
		ServerVersion: info.ServerVersion,
		StorageDriver: info.Driver,
		Healthy:       true,
	}, nil
}
