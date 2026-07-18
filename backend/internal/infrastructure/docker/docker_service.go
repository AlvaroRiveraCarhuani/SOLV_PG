package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{cli: cli}, nil
}

func (d *DockerClient) StartContainer(ctx context.Context, image, containerName, traefikHost string) error {
	labels := map[string]string{
		"traefik.enable": "true",
		"traefik.http.routers." + containerName + ".rule": "Host(\"" + traefikHost + "\")",
		"traefik.docker.network":                          "solv_net",
	}

	config := &container.Config{
		Image:  image,
		Labels: labels,
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "solv_net",
	}

	networkingConfig := &network.NetworkingConfig{}

	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}
