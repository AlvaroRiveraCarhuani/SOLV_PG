package domain

import "time"

// StorageStatusType representa el nivel de riesgo de espacio en disco del host
type StorageStatusType string

const (
	StorageStatusOK              StorageStatusType = "OK"
	StorageStatusWarning         StorageStatusType = "WARNING"
	StorageStatusCriticalBlocked StorageStatusType = "CRITICAL_BLOCKED"
)

// CapabilitiesProbeResult resultado del smoke test seguro en contenedor efímero aislado
type CapabilitiesProbeResult struct {
	Success          bool     `json:"success"`
	DetectedRuntimes []string `json:"detected_runtimes"`
	Output           string   `json:"output,omitempty"`
}

// ImageVerificationResult resultado detallado de la verificación de imagen Docker
type ImageVerificationResult struct {
	ImageRef                string                   `json:"image_ref"`
	IsLocal                 bool                     `json:"is_local"`
	Exists                  bool                     `json:"exists"`
	ArchitectureCompatible  bool                     `json:"architecture_compatible"`
	HostArch                string                   `json:"host_arch"`
	SupportedPlatforms      []string                 `json:"supported_platforms"`
	SizeBytes               int64                    `json:"size_bytes"`
	SizeFormatted           string                   `json:"size_formatted"`
	EstimatedUncompressedMB int64                    `json:"estimated_uncompressed_mb"`
	HostDiskFreeGB          float64                  `json:"host_disk_free_gb"`
	HostDiskTotalGB         float64                  `json:"host_disk_total_gb"`
	StorageStatus           StorageStatusType        `json:"storage_status"`
	StorageMessage          string                   `json:"storage_message"`
	CapabilitiesProbe       *CapabilitiesProbeResult `json:"capabilities_probe,omitempty"`
	BuildxSuggestion        string                   `json:"buildx_suggestion,omitempty"`
	ErrorMessage            string                   `json:"error_message,omitempty"`
	Cached                  bool                     `json:"cached"`
	VerifiedAt              time.Time                `json:"verified_at"`
}

// LocalImageItem imagen residente en el daemon Docker local para autocompletado y selección inmediata
type LocalImageItem struct {
	RepoTag      string    `json:"repo_tag"`
	SizeMB       int64     `json:"size_mb"`
	CreatedAt    time.Time `json:"created_at"`
	IsOfficial   bool      `json:"is_official"`
	HasLatestTag bool      `json:"has_latest_tag"`
}

// VerifyImageRequest DTO para solicitar la verificación de una imagen Docker
type VerifyImageRequest struct {
	Image string `json:"image" validate:"required"`
	Force bool   `json:"force"`
}
