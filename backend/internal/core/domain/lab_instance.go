package domain

import (
	"context"
	"time"
)

type LabInstance struct {
	ID          string    `db:"id" json:"id"`
	TemplateID  string    `db:"template_id" json:"template_id"`
	UserID      string    `db:"user_id" json:"user_id"`
	ContainerID *string   `db:"container_id" json:"container_id"`
	Status      string    `db:"status" json:"status"`
	RAMLimitMB  int       `db:"ram_limit_mb" json:"ram_limit_mb"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type LabInstanceRepository interface {
	Create(ctx context.Context, instance *LabInstance) error
	GetByID(ctx context.Context, id string) (*LabInstance, error)
	GetByUserAndTemplate(ctx context.Context, userID string, templateID string) (*LabInstance, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateContainerID(ctx context.Context, id string, containerID string) error
}
