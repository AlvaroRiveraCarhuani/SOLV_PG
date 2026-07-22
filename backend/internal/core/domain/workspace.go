package domain

import "time"

const (
	WorkspaceStatusPending    = "pending"
	WorkspaceStatusRunning    = "running"
	WorkspaceStatusHibernated = "hibernated"
	WorkspaceStatusOOMKilled  = "oom_killed"
	WorkspaceStatusFailed     = "failed"

	DefaultBaseMemoryMB int64   = 256
	HardQuotaMemoryMB   int64   = 2048
	MinHostFreeRAMPct   float64 = 15.0 // Porcentaje mínimo de RAM libre en el Host
	MaxOOMStrikes       int     = 3
	OOMCooldownDuration         = 5 * time.Minute
)

type WorkspaceInstance struct {
	ID              string     `db:"id" json:"workspace_id"`
	StudentID       string     `db:"student_id" json:"student_id"`
	SubjectID       string     `db:"subject_id" json:"subject_id"`
	ContainerID     *string    `db:"container_id" json:"-"`
	Status          string     `db:"status" json:"status"`
	AccessURL       string     `db:"access_url" json:"access_url"`
	MemoryLimitMB   int64      `db:"memory_limit_mb" json:"memory_limit_mb"`
	LastHeartbeatAt time.Time  `db:"last_heartbeat_at" json:"last_heartbeat_at"`
	LastOOMKilledAt *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
	OOMStrikeCount  int        `db:"oom_strike_count" json:"oom_strike_count"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
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

type ContainerMetrics struct {
	MemoryUsageBytes int64
	MemoryLimitBytes int64
	CPUPercent       float64
	RxBytes          uint64
	TxBytes          uint64
	IsRunning        bool
	OOMKilled        bool
	ExitCode         int
}
