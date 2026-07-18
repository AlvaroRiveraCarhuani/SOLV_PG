package domain

type CreateInstanceDTO struct {
	UserID        string `json:"user_id" validate:"required"`
	TemplateID    string `json:"template_id" validate:"required"`
	ContainerName string `json:"container_name" validate:"required"`
	TraefikURL    string `json:"traefik_url" validate:"required"`
	Status        string `json:"status" validate:"required"`
}
