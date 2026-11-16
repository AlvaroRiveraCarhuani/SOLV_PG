package domain

// ServerPolicies define los límites QoS configurables por el administrador
// institucional (ADR-014 / submódulo 14.6). Se persisten en tenants.config
// bajo la clave "server_policies" y el worker QoS las recarga en cada ciclo.
type ServerPolicies struct {
	RAMLimitMB           int `json:"ram_limit_mb"`            // 256 | 512 | 1024
	InactivityMinutes    int `json:"inactivity_minutes"`      // 10 | 15 | 30 | 60
	MaxContainers        int `json:"max_containers"`          // >= 1
	HostSafetyCeilingPct int `json:"host_safety_ceiling_pct"` // 75 | 80 | 85 | 90
	// PracticeCPUMillis es la cuota de CPU de cada entorno, en milésimas de
	// núcleo. Cero significa sin límite, como antes de la política.
	PracticeCPUMillis int `json:"practice_cpu_millis"`
	// PidsLimit acota los procesos del contenedor. Cero delega el valor al
	// runtime, que usa el mismo tope que el entorno de revisión.
	PidsLimit int `json:"pids_limit"`
	// DiskQuotaMB es la cuota blanda por entorno medida al entregar. Cero
	// desactiva la comprobación.
	DiskQuotaMB int `json:"disk_quota_mb"`
	// DiskWarningPct y DiskBlockPct son los umbrales del aviso y del bloqueo.
	DiskWarningPct int    `json:"disk_warning_pct"`
	DiskBlockPct   int    `json:"disk_block_pct"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// DefaultServerPolicies valores por defecto vigentes antes de este cambio
// (equivalentes al hardcode histórico del worker QoS en cmd/api/main.go).
func DefaultServerPolicies() ServerPolicies {
	return ServerPolicies{
		RAMLimitMB:           512,
		InactivityMinutes:    15,
		MaxContainers:        40,
		HostSafetyCeilingPct: 85,
		// Un núcleo por entorno: el valor que el contenedor ya tenía fijo en el
		// arranque, ahora declarado en la política del nodo.
		PracticeCPUMillis: 1000,
		PidsLimit:         512,
		// La cuota sale de la medición de capacidad: (34 GB libres + 19 GB
		// recuperables) × 0,85 ÷ 40 alumnos ≈ 1,1 GB por entorno.
		DiskQuotaMB:    1024,
		DiskWarningPct: 80,
		DiskBlockPct:   100,
	}
}

// UpdateServerPoliciesDTO actualización parcial de políticas.
type UpdateServerPoliciesDTO struct {
	RAMLimitMB           *int `json:"ram_limit_mb,omitempty"`
	InactivityMinutes    *int `json:"inactivity_minutes,omitempty"`
	MaxContainers        *int `json:"max_containers,omitempty"`
	HostSafetyCeilingPct *int `json:"host_safety_ceiling_pct,omitempty"`
	PracticeCPUMillis    *int `json:"practice_cpu_millis,omitempty"`
	PidsLimit            *int `json:"pids_limit,omitempty"`
	DiskQuotaMB          *int `json:"disk_quota_mb,omitempty"`
	DiskWarningPct       *int `json:"disk_warning_pct,omitempty"`
	DiskBlockPct         *int `json:"disk_block_pct,omitempty"`
}

// Valores permitidos por el wireframe oficial (docs/UI/ADMIN/CONFIGURACION.md 2.4)
var AllowedRAMLimitsMB = []int{256, 512, 1024}
var AllowedInactivityMinutes = []int{10, 15, 30, 60}
var AllowedHostSafetyCeilingsPct = []int{75, 80, 85, 90}

// Valores admitidos de la política de CPU y procesos del entorno.
var AllowedPracticeCPUMillis = []int{0, 500, 1000, 1500, 2000}
var AllowedPidsLimits = []int{256, 512, 1024, 2048}

// Valores admitidos de la cuota blanda de disco por entorno.
var AllowedDiskQuotasMB = []int{0, 512, 1024, 2048, 4096}

// EffectivePracticeCPUMillis devuelve la cuota vigente; un cero persistido
// significa sin límite y se respeta tal cual.
func (p ServerPolicies) EffectivePracticeCPUMillis() int64 {
	if p.PracticeCPUMillis <= 0 {
		return 0
	}
	return int64(p.PracticeCPUMillis)
}

// EffectivePidsLimit cae al tope del runtime cuando la política no lo declara.
func (p ServerPolicies) EffectivePidsLimit() int64 {
	if p.PidsLimit <= 0 {
		return 512
	}
	return int64(p.PidsLimit)
}

// EffectiveDiskQuotaMB devuelve 0 cuando la cuota está desactivada.
func (p ServerPolicies) EffectiveDiskQuotaMB() int64 { return int64(p.DiskQuotaMB) }

func (p ServerPolicies) EffectiveDiskWarningPct() int {
	if p.DiskWarningPct <= 0 {
		return 80
	}
	return p.DiskWarningPct
}

func (p ServerPolicies) EffectiveDiskBlockPct() int {
	if p.DiskBlockPct <= 0 {
		return 100
	}
	return p.DiskBlockPct
}

// CPUQuotaNanos traduce milésimas de CPU a NanoCPUs. Cero o negativo deja el
// contenedor sin cuota, que es el comportamiento previo a la política.
func CPUQuotaNanos(millis int64) int64 {
	if millis <= 0 {
		return 0
	}
	return millis * 1_000_000
}
