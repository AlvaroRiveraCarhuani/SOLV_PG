package domain

type CreateTemplateDTO struct {
	Name        string `json:"name" validate:"required"`
	DockerImage string `json:"docker_image" validate:"required"`
	BaseRamMB   int    `json:"base_ram_mb" validate:"required,gt=0"`
}

type TemplateResponseDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DockerImage string `json:"docker_image"`
	BaseRamMB   int    `json:"base_ram_mb"`
}
