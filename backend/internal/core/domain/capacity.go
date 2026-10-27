package domain

const (
	// CapacityBasisMemAvailable indica que el limite efectivo lo impone la RAM
	// disponible medida en el host.
	CapacityBasisMemAvailable = "mem_available"
	// CapacityBasisPolicy indica que el limite efectivo lo impone la politica
	// del tenant (max_containers) o que no fue posible medir la RAM.
	CapacityBasisPolicy = "policy"
)

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
