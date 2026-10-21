package domain

import "encoding/json"

type CreateTemplateDTO struct {
	Name        string `json:"name" validate:"required"`
	DockerImage string `json:"docker_image" validate:"required"`
	BaseRamMB   int    `json:"base_ram_mb" validate:"required,gt=0"`
}

type TemplateResponseDTO struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	DockerImage       string          `json:"docker_image"`
	BaseRamMB         int             `json:"base_ram_mb"`
	TargetEnvironment string          `json:"target_environment"`
	Status            string          `json:"status"`
	Entrypoint        string          `json:"entrypoint,omitempty"`
	ServicesConfig    json.RawMessage `json:"services_config,omitempty"`
}
