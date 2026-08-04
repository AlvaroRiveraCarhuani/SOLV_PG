package domain

import "time"

const (
	WorkspaceStatusPending    = "pending"
	WorkspaceStatusRunning    = "running"
	WorkspaceStatusHibernated = "hibernated"
	WorkspaceStatusOOMKilled  = "oom_killed"
	WorkspaceStatusFailed     = "failed"

	WorkspaceTypeIDEPersistente = "IDE_PERSISTENTE"
	WorkspaceTypeJuezEfimero    = "JUEZ_EFIMERO"

	DefaultBaseMemoryMB int64   = 256
	HardQuotaMemoryMB   int64   = 2048
	MinHostFreeRAMPct   float64 = 15.0 // Porcentaje mínimo de RAM libre en el Host
	MaxOOMStrikes       int     = 3
	OOMCooldownDuration         = 5 * time.Minute

	// Factor de suavizado EWMA (alpha = 0.2) y factor de margen (1.25 -> 25% headroom)
	EWMAAlpha                 float64 = 0.2
	EWMASafetyMarginFactor    float64 = 1.25
)

type WorkspaceInstance struct {
	ID              string     `db:"id" json:"workspace_id"`
	StudentID       string     `db:"student_id" json:"student_id"`
	SubjectID       string     `db:"subject_id" json:"subject_id"`
	Type            string     `db:"type" json:"type"`
	ContainerID     *string    `db:"container_id" json:"-"`
	Status          string     `db:"status" json:"status"`
	AccessURL       string     `db:"access_url" json:"access_url"`
	MemoryLimitMB   int64      `db:"memory_limit_mb" json:"memory_limit_mb"`
	LastHeartbeatAt time.Time  `db:"last_heartbeat_at" json:"last_heartbeat_at"`
	LastOOMKilledAt *time.Time `db:"last_oom_killed_at" json:"last_oom_killed_at,omitempty"`
	OOMStrikeCount  int        `db:"oom_strike_count" json:"oom_strike_count"`
	SemgrepAudit    []byte     `db:"semgrep_audit" json:"semgrep_audit,omitempty"`
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

type EWMAState struct {
	CurrentEWMAMB float64   `json:"current_ewma_mb"`
	SampleCount   int       `json:"sample_count"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

type ResourceProfile struct {
	SignatureHash string    `json:"signature_hash"`
	BaseMemoryMB  int64     `json:"base_memory_mb"`
	MaxQuotaMB    int64     `json:"max_quota_mb"`
	EWMAState     EWMAState `json:"ewma_state"`
}

type LabTemplate struct {
	ID              string          `db:"id" json:"id"`
	Name            string          `db:"name" json:"name"`
	BaseImage       string          `db:"base_image" json:"base_image"`
	SetupScript     string          `db:"setup_script" json:"setup_script"`
	SignatureHash   string          `db:"signature_hash" json:"signature_hash"`
	ResourceProfile ResourceProfile `json:"resource_profile"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
}
