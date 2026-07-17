package docker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

var (
	ErrImagePullFailed = errors.New("failed to pull docker image")
	ErrContainerCreate = errors.New("failed to create container")
	ErrContainerStart  = errors.New("failed to start container")
)

type Manager struct {
	cli *client.Client
}

func NewManager(cli *client.Client) *Manager {
	return &Manager{
		cli: cli,
	}
}
func (m *Manager) StartTestContainer(ctx context.Context) error {
	imageName := "nginx:alpine"
	reader, err := m.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrImagePullFailed, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	labels := m.buildDockerLabels()

	containerConfig := &container.Config{
		Image:  imageName,
		Labels: labels,
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "solv_net",
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"solv_net": {},
		},
	}
	resp, err := m.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, "")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrContainerCreate, err)
	}
	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("%w: %v", ErrContainerStart, err)
	}

	return nil
}
func (m *Manager) buildDockerLabels() map[string]string {
	return map[string]string{
		"traefik.enable":                                        "true",
		"traefik.http.routers.prueba.rule":                      "Host(`prueba.solv.local`)",
		"traefik.http.services.prueba.loadbalancer.server.port": "80",
		"traefik.docker.network":                                "solv_net",
	}
}
