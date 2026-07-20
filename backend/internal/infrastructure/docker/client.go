package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"

	"solv-backend/internal/core/domain"
)

// Client implementa la interfaz domain.ContainerOrchestrator
type Client struct {
	cli *client.Client
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

func (c *Client) StartContainer(ctx context.Context, config domain.LabContainerConfig) (string, error) {
	mountMode := "rw"
	if config.ReadOnly {
		mountMode = "ro"
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/workspace:%s", config.VolumeName, mountMode),
		},
		Resources: container.Resources{
			Memory: config.MemoryLimitMB * 1024 * 1024, // Conversión a bytes
		},
		NetworkMode: container.NetworkMode(config.NetworkMode),
	}

	containerConfig := &container.Config{
		Image:  config.Image,
		Labels: config.Labels,
	}

	networkingConfig := &network.NetworkingConfig{}
	_, _, err := c.cli.ImageInspectWithRaw(ctx, config.Image)
	if err != nil {
		if errdefs.IsNotFound(err) {
			out, pullErr := c.cli.ImagePull(ctx, config.Image, image.PullOptions{})
			if pullErr != nil {
				return "", fmt.Errorf("failed to pull image %q: %w", config.Image, pullErr)
			}
			defer out.Close()
			_, _ = io.Copy(io.Discard, out)
		} else {
			return "", fmt.Errorf("failed to inspect image %q: %w", config.Image, err)
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container %q: %w", config.ContainerName, err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container %q (ID: %s): %w", config.ContainerName, resp.ID, err)
	}

	return resp.ID, nil
}

func (c *Client) HibernateContainer(ctx context.Context, containerID string) error {
	timeout := 5
	stopOpts := container.StopOptions{Timeout: &timeout}

	err := c.cli.ContainerStop(ctx, containerID, stopOpts)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to gracefully stop container %q during hibernation: %w", containerID, err)
	}

	removeOpts := container.RemoveOptions{
		Force: true, // Asegura la eliminación incluso si Stop falló parcialmente
	}
	err = c.cli.ContainerRemove(ctx, containerID, removeOpts)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove container %q during hibernation: %w", containerID, err)
	}

	return nil
}

// ExecuteDryRun arranca un contenedor efímero, obtiene sus stats y devuelve el pico de RAM usado.
func (c *Client) ExecuteDryRun(ctx context.Context, image string) (int64, error) {
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	containerConfig := &container.Config{
		Image: image,
		Cmd:   []string{"sh", "-c", "echo test && sleep 1"},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return 0, fmt.Errorf("dry-run create failed: %w", err)
	}
	containerID := resp.ID

	// Asegurar destrucción al finalizar
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return 0, fmt.Errorf("dry-run start failed: %w", err)
	}

	stats, err := c.cli.ContainerStats(ctx, containerID, true)
	if err != nil {
		return 0, fmt.Errorf("dry-run stats failed: %w", err)
	}
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)
	var maxRAM int64

	for {
		var s container.StatsResponse
		if err := decoder.Decode(&s); err != nil {
			break // EOF or container exit
		}
		usage := int64(s.MemoryStats.Usage)
		if usage > maxRAM {
			maxRAM = usage
		}
	}

	return maxRAM, nil
}
