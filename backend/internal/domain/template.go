package domain

type CreateTemplateDTO struct {
	Name        string `json:"name"`
	DockerImage string `json:"docker_image"`
	BaseRamMB   int    `json:"base_ram_mb"`
}

type TemplateResponseDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DockerImage string `json:"docker_image"`
	BaseRamMB   int    `json:"base_ram_mb"`
}
