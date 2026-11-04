package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"golang.org/x/sync/singleflight"
)

// Client implementa la interfaz domain.ContainerOrchestrator y domain.PracticeEnvironmentOrchestrator
type Client struct {
	cli       *client.Client
	pullGroup singleflight.Group
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// EnsureVolumeExists verifica y crea el volumen local si no existe (ADR-001)
func (c *Client) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	_, err := c.cli.VolumeInspect(ctx, volumeName)
	if err == nil {
		return nil // El volumen ya existe
	}

	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect volume %q: %w", volumeName, err)
	}

	// Crear el volumen ya que no fue encontrado
	_, err = c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   volumeName,
		Driver: "local",
	})
	if err != nil {
		return fmt.Errorf("failed to create local volume %q: %w", volumeName, err)
	}

	return nil
}

// RemoveVolume elimina un volumen nombrado y su contenido. Solo lo usan los
// previews efímeros: el volumen del alumno nunca pasa por acá.
func (c *Client) RemoveVolume(ctx context.Context, volumeName string) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("volume name is required")
	}
	if err := c.cli.VolumeRemove(ctx, volumeName, true); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove volume %q: %w", volumeName, err)
	}
	return nil
}

// GetRawClient expone el cliente Docker subyacente para uso directo del SDK
func (c *Client) GetRawClient() *client.Client {
	return c.cli
}
