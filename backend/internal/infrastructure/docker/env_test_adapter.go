package docker

import (
	"errors"
	"time"

	"github.com/docker/docker/client"

	"solv-backend/internal/core/domain"
)

var (
	ErrPullStalled = errors.New(domain.EnvTestErrPullStalled)
)

// EnvTestDockerAdapter implementa domain.ImageRegistryPort y domain.ContainerRunnerPort usando Docker SDK
type EnvTestDockerAdapter struct {
	cli                *client.Client
	inactivityDeadline time.Duration
}

// NewEnvTestDockerAdapter inicializa el adaptador de docker para prueba de entorno
func NewEnvTestDockerAdapter(cli *client.Client, inactivityDeadline time.Duration) *EnvTestDockerAdapter {
	if inactivityDeadline <= 0 {
		inactivityDeadline = 60 * time.Second
	}
	return &EnvTestDockerAdapter{
		cli:                cli,
		inactivityDeadline: inactivityDeadline,
	}
}
