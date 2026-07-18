package domain

import "context"

type UserRepository interface {
	CreateUser(ctx context.Context, dto CreateUserDTO) (string, error)
	GetUserByID(ctx context.Context, id string) (UserResponseDTO, error)
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
