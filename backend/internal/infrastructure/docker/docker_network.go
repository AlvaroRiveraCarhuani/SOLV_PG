package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
)

// EnsureICCDisabledNetworkExists asegura que exista la red de Docker con la directiva ICC (Inter-Container Communication) desactivada
func (c *Client) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	_, err := c.cli.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err == nil {
		return nil // La red ya existe
	}

	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect docker network %q: %w", networkName, err)
	}

	_, err = c.cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "false",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create network with enable_icc=false %q: %w", networkName, err)
	}

	return nil
}
