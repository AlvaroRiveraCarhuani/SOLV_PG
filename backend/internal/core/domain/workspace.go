package domain

import "time"

type WorkspaceInstance struct {
	ID          string    `db:"id" json:"workspace_id"`
	StudentID   string    `db:"student_id" json:"student_id"`
	SubjectID   string    `db:"subject_id" json:"subject_id"`
	ContainerID *string   `db:"container_id" json:"-"` // Identificador interno opaco, no expuesto en la API
	Status      string    `db:"status" json:"status"`
	AccessURL   string    `db:"access_url" json:"access_url"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type WorkspaceContainerConfig struct {
	Image         string
	ContainerName string
	VolumeName    string
	MemoryLimitMB int64
	NetworkName   string
	Labels        map[string]string
	Env           []string
}
