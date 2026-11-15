package domain

const (
	// CapacityBasisMemAvailable indica que el limite efectivo lo impone la RAM
	// disponible medida en el host.
	CapacityBasisMemAvailable = "mem_available"
	// CapacityBasisPolicy indica que el limite efectivo lo impone la politica
	// del tenant (max_containers) o que no fue posible medir la RAM.
	CapacityBasisPolicy = "policy"
)

const (
	// PlatformReserveMB es lo que se descuenta del host antes de repartir RAM
	// entre laboratorios: sistema operativo, contenedores de la propia
	// plataforma (backend, frontend, traefik, base principal, métricas) y margen
	// del daemon. Es un piso de arranque, no una medición, y el panel lo muestra
	// junto al número recomendado para que el supuesto sea visible.
	PlatformReserveMB = 2048

	// MaxAllowedContainers es el mayor techo que acepta la validación de
	// políticas (server_policies_service.go).
	MaxAllowedContainers = 500

	// CapacityBasisHostRAM indica que el número recomendado sale de la RAM
	// total medida en el host.
	CapacityBasisHostRAM = "host_ram"
)

// CapacityRecommendation explica de dónde sale el techo recomendado de
// concurrencia, con los mismos supuestos que aplica la admisión.
type CapacityRecommendation struct {
	RecommendedMaxContainers int    `json:"recommended_max_containers"`
	HostTotalMB              int64  `json:"host_total_mb"`
	PlatformReserveMB        int    `json:"platform_reserve_mb"`
	RAMPerLabMB              int    `json:"ram_per_lab_mb"`
	HostSafetyCeilingPct     int    `json:"host_safety_ceiling_pct"`
	CapacityBasis            string `json:"capacity_basis"`
}

// RecommendMaxContainers estima cuántos entornos entran en este host con la
// RAM por laboratorio y el techo de seguridad vigentes.
//
//   - Se planifica sobre la RAM total, no sobre la disponible: lo disponible ya
//     incluye lo que ocupan los laboratorios abiertos y cambiaría según la hora.
//   - Del total se descuenta PlatformReserveMB y recién después se reparte entre
//     laboratorios.
//   - Devuelve 0 cuando no se pudo medir el host: quien llame no debe inventar
//     un techo a partir de ese cero.
func RecommendMaxContainers(hostTotalMB int64, ramPerLabMB, ceilingPct int) int {
	if hostTotalMB <= 0 || ramPerLabMB <= 0 {
		return 0
	}
	if ceilingPct <= 0 || ceilingPct > 100 {
		ceilingPct = DefaultServerPolicies().HostSafetyCeilingPct
	}

	budgetMB := hostTotalMB*int64(ceilingPct)/100 - PlatformReserveMB
	if budgetMB < int64(ramPerLabMB) {
		return 1
	}
	recommended := budgetMB / int64(ramPerLabMB)
	if recommended > MaxAllowedContainers {
		return MaxAllowedContainers
	}
	return int(recommended)
}

// BuildCapacityRecommendation arma la respuesta completa del panel. Con
// hostTotalMB <= 0 deja la base en "unavailable" y no recomienda nada.
func BuildCapacityRecommendation(hostTotalMB int64, ramPerLabMB, ceilingPct int) CapacityRecommendation {
	recommendation := CapacityRecommendation{
		HostTotalMB:          hostTotalMB,
		PlatformReserveMB:    PlatformReserveMB,
		RAMPerLabMB:          ramPerLabMB,
		HostSafetyCeilingPct: ceilingPct,
		CapacityBasis:        CapacityBasisHostRAM,
	}
	if hostTotalMB <= 0 {
		recommendation.CapacityBasis = "unavailable"
		return recommendation
	}
	recommendation.RecommendedMaxContainers = RecommendMaxContainers(hostTotalMB, ramPerLabMB, ceilingPct)
	return recommendation
}

// EstimateCapacity calcula cuantos laboratorios entran realmente en el nodo.
// Devuelve el limite efectivo y cual de los dos limites manda, para que la
// interfaz pueda explicar el numero en vez de mostrar una estimacion opaca.
//
// Reglas:
//   - El techo de politica (max_containers) y la RAM disponible compiten; gana
//     el menor de los dos.
//   - Nunca informa menos de lo que ya esta corriendo.
//   - Con availableMB == 0 se asume que no se pudo medir: manda la politica.
func EstimateCapacity(availableMB uint64, ramPerLabMB, maxContainers, running int) (int, string) {
	defaults := DefaultServerPolicies()
	if ramPerLabMB <= 0 {
		ramPerLabMB = defaults.RAMLimitMB
	}
	if maxContainers <= 0 {
		maxContainers = defaults.MaxContainers
	}
	if running < 0 {
		running = 0
	}

	if availableMB == 0 {
		return max(maxContainers, running), CapacityBasisPolicy
	}

	byMemory := int(availableMB / uint64(ramPerLabMB))
	if byMemory < maxContainers {
		return max(byMemory, running), CapacityBasisMemAvailable
	}
	return max(maxContainers, running), CapacityBasisPolicy
}
