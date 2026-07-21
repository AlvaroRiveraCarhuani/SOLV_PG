package domain

import "context"

type UserRepository interface {
	CreateUser(ctx context.Context, dto CreateUserDTO) (string, error)
	GetUserByID(ctx context.Context, id string) (UserResponseDTO, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUserFromSSO(ctx context.Context, user *User) (string, error)
}

type TemplateRepository interface {
	CreateTemplate(ctx context.Context, dto CreateTemplateDTO) (string, error)
	GetAllTemplates(ctx context.Context) ([]TemplateResponseDTO, error)
	GetTemplateByID(ctx context.Context, id string) (TemplateResponseDTO, error)
}

type InstanceRepository interface {
	CreateInstance(ctx context.Context, dto CreateInstanceDTO) (string, error)
	UpdateInstanceStatus(ctx context.Context, id string, status string) error
}

type DockerService interface {
	StartContainer(ctx context.Context, image, containerName, traefikHost string) error
}

type LabContainerConfig struct {
	Image         string
	ContainerName string
	VolumeName    string
	MemoryLimitMB int64
	NetworkMode   string
	ReadOnly      bool
}

type ContainerOrchestrator interface {
	EnsureVolumeExists(ctx context.Context, volumeName string) error
	StartContainer(ctx context.Context, config LabContainerConfig) (string, error)
	HibernateContainer(ctx context.Context, containerID string) error
}
