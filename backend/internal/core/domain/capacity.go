package domain

import "fmt"

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

// Rectángulos humanos de capacidad: cuántos estudiantes simultáneos caben con
// un entorno en el nodo actual, sin que el administrador calcule MB a mano.
const (
	// CapacityLevelComfortable indica que el hueco sobra para más de un entorno
	// del mismo tamaño.
	CapacityLevelComfortable = "comfortable"
	// CapacityLevelModerate indica que queda hueco pero ya no sobra.
	CapacityLevelModerate = "moderate"
	// CapacityLevelTight indica que el hueco está al límite del nodo.
	CapacityLevelTight = "tight"
)

// CapacityImpact es la capacidad en lenguaje humano para un entorno en el
// servidor actual. Se calcula en vivo con la RAM total medida del nodo y el
// techo de seguridad que aplica la admisión, nunca con un número fijo.
type CapacityImpact struct {
	EstimatedConcurrentStudents int    `json:"estimated_concurrent_students"`
	CurrentlyAvailableStudents  int    `json:"currently_available_students"`
	Level                       string `json:"level"`
	Summary                     string `json:"summary"`
}

// BuildCapacityImpact traduce el presupuesto de admisión a estudiantes
// simultáneos para un entorno que pide ramPerEnvMB por alumno.
//
//   - committedMB es la memoria ya comprometida (entornos activos y reserva
//     de motores del nodo): lo que queda libre tras descontarlo.
//   - EstimatedConcurrentStudents se calcula sin descontar lo comprometido:
//     es el tamaño total de una clase con ese entorno en este servidor.
//   - CurrentlyAvailableStudents descuenta lo comprometido: cuántos más caben
//     ahora mismo.
//   - Level se decide por la fracción de hueco libre que queda: al menos el
//     80% es cómoda, desde el 40% es moderada y por debajo es ajustada.
func BuildCapacityImpact(hostTotalMB int64, ceilingPct int, ramPerEnvMB, committedMB int64) CapacityImpact {
	if ramPerEnvMB <= 0 {
		ramPerEnvMB = 512
	}
	if ceilingPct <= 0 || ceilingPct > 100 {
		ceilingPct = DefaultServerPolicies().HostSafetyCeilingPct
	}
	if committedMB < 0 {
		committedMB = 0
	}

	allocatableMB := int64(0)
	if hostTotalMB > 0 {
		allocatableMB = hostTotalMB * int64(ceilingPct) / 100
	}

	estimated := int64(0)
	if allocatableMB > 0 {
		estimated = allocatableMB / ramPerEnvMB
	}
	if estimated < 0 {
		estimated = 0
	}

	freeMB := allocatableMB - committedMB
	if freeMB < 0 {
		freeMB = 0
	}
	available := freeMB / ramPerEnvMB

	level := CapacityLevelTight
	switch {
	case estimated <= 0:
		level = CapacityLevelTight
	case available*5 >= estimated*4: // al menos el 80% libre
		level = CapacityLevelComfortable
	case available*5 >= estimated*2: // al menos el 40% libre
		level = CapacityLevelModerate
	default:
		level = CapacityLevelTight
	}

	return CapacityImpact{
		EstimatedConcurrentStudents: int(estimated),
		CurrentlyAvailableStudents:  int(available),
		Level:                       level,
		Summary:                     capacitySummary(int(estimated)),
	}
}

// capacitySummary redacta la frase corta para un administrador no técnico.
func capacitySummary(estimated int) string {
	if estimated <= 0 || estimated == 1 {
		return "Con este entorno solo cabe 1 estudiante a la vez en tu servidor actual"
	}
	return fmt.Sprintf("Con este entorno caben aprox. %d estudiantes a la vez en tu servidor actual", estimated)
}

// HeavyOperationGate es el múltiplo del load average por núcleo por encima del
// cual se frenan operaciones pesadas (builds y previews). Conservador: con más
// carga que 1,5 núcleos por núcleo el arranque de clase ya se degrada.
const HeavyOperationGate = 1.5

// HeavyOperationAllowed decide si un build o un preview puede arrancar con el
// load average de 1 minuto medido. numCPU <= 0 se trata como 1 núcleo.
func HeavyOperationAllowed(load1 float64, numCPU int) bool {
	if numCPU <= 0 {
		numCPU = 1
	}
	if load1 < 0 {
		return true
	}
	return load1 < float64(numCPU)*HeavyOperationGate
}

// DiskAlertFreePct es el porcentaje libre por debajo del cual se avisa de disco
// con margen: el piso duro que bloquea altas sigue siendo DefaultHostDiskFloorPct
// (5%).
const DiskAlertFreePct = 15.0

// DiskAlertRecommended avisa cuando el disco se acerca al piso duro pero aún no
// lo cruzó: entre el aviso y el bloqueo el administrador puede liberar espacio.
// Por debajo del piso ya no es un aviso: el alta de entornos está bloqueada.
func DiskAlertRecommended(freePct float64) bool {
	if freePct < 0 {
		return false
	}
	return freePct > DefaultHostDiskFloorPct && freePct < DiskAlertFreePct
}
