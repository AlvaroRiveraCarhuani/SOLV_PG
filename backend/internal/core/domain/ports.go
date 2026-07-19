package domain

import "context"

type LabContainerConfig struct {
	Image         string
	ContainerName string
	VolumeName    string
	MemoryLimitMB int64
	NetworkMode   string
	ReadOnly      bool
	Labels        map[string]string
}

type ContainerOrchestrator interface {
	EnsureVolumeExists(ctx context.Context, volumeName string) error
	ExecuteDryRun(ctx context.Context, image string) (int64, error)
	StartContainer(ctx context.Context, config LabContainerConfig) (string, error)
	HibernateContainer(ctx context.Context, containerID string) error
}

type Template struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	DockerImage string `db:"docker_image" json:"docker_image"`
	BaseRamMB   int    `db:"base_ram_mb" json:"base_ram_mb"`
}

type TemplateRepository interface {
	GetTemplateByID(ctx context.Context, id string) (*Template, error)
}
