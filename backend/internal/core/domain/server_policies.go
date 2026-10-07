package domain

// ServerPolicies define los límites QoS configurables por el administrador
// institucional (ADR-014 / submódulo 14.6). Se persisten en tenants.config
// bajo la clave "server_policies" y el worker QoS las recarga en cada ciclo.
type ServerPolicies struct {
	RAMLimitMB        int    `json:"ram_limit_mb"`       // 256 | 512 | 1024
	InactivityMinutes int    `json:"inactivity_minutes"` // 10 | 15 | 30
	MaxContainers     int    `json:"max_containers"`     // >= 1
	UpdatedAt         string `json:"updated_at,omitempty"`
}

// DefaultServerPolicies valores por defecto vigentes antes de este cambio
// (equivalentes al hardcode histórico del worker QoS en cmd/api/main.go).
func DefaultServerPolicies() ServerPolicies {
	return ServerPolicies{
		RAMLimitMB:        512,
		InactivityMinutes: 15,
		MaxContainers:     40,
	}
}

// UpdateServerPoliciesDTO actualización parcial de políticas.
type UpdateServerPoliciesDTO struct {
	RAMLimitMB        *int `json:"ram_limit_mb,omitempty"`
	InactivityMinutes *int `json:"inactivity_minutes,omitempty"`
	MaxContainers     *int `json:"max_containers,omitempty"`
}

// Valores permitidos por el wireframe oficial (docs/UI/ADMIN/CONFIGURACION.md 2.4)
var AllowedRAMLimitsMB = []int{256, 512, 1024}
var AllowedInactivityMinutes = []int{10, 15, 30}
