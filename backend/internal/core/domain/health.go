package domain

// Estados de salud del panel (S6, ADR-049). Reemplazan al antiguo "healthy" /
// "warning" que no distinguia entre el daemon caido y un pico de RAM medido.
const (
	HealthStatusOperational = "operational"
	HealthStatusDegraded    = "degraded"
	HealthStatusCritical    = "critical"
)

// Codigos estables de motivo. La interfaz los traduce a texto; nunca cambian de
// nombre porque las alertas y el frontend dependen de la cadena.
const (
	HealthReasonDockerUnreachable     = "docker_unreachable"
	HealthReasonOrphanedWorkspaces    = "orphaned_workspaces"
	HealthReasonOOMKilledWorkspaces   = "oom_killed_workspaces"
	HealthReasonUnknownWorkspaces     = "unknown_workspaces"
	HealthReasonRAMAboveSafetyCeiling = "ram_above_safety_ceiling"
)

// HealthEvaluationInput son los hechos medidos que alimentan el veredicto.
// Todos vienen de fuentes reales (Docker, reconciliacion, conteo por estado y
// RAM del host), nunca de literales ni estimaciones.
type HealthEvaluationInput struct {
	DockerHealthy    bool
	OrphanedLabs     int
	OOMKilledLabs    int
	UnknownLabs      int
	RAMPercent       float64
	SafetyCeilingPct int
}

// EvaluateHealth deriva el estado y los motivos del panel. Reglas (S6):
//
//   - critical: el daemon de Docker no responde o hay contenedores huerfanos
//     (la integridad del entorno esta comprometida).
//   - degraded: hay laboratorios OOM, estados sin verificar (unknown) o la RAM
//     del host supera el techo de seguridad de la politica.
//   - operational: ninguno de los anteriores.
//
// Devuelve tambien los motivos que explican el veredicto; operacional va con
// lista vacia para que la interfaz no muestre un tooltip inventado.
func EvaluateHealth(in HealthEvaluationInput) (string, []string) {
	ceiling := in.SafetyCeilingPct
	if ceiling <= 0 {
		ceiling = DefaultServerPolicies().HostSafetyCeilingPct
	}

	reasons := make([]string, 0, 5)
	if !in.DockerHealthy {
		reasons = append(reasons, HealthReasonDockerUnreachable)
	}
	if in.OrphanedLabs > 0 {
		reasons = append(reasons, HealthReasonOrphanedWorkspaces)
	}
	if in.OOMKilledLabs > 0 {
		reasons = append(reasons, HealthReasonOOMKilledWorkspaces)
	}
	if in.UnknownLabs > 0 {
		reasons = append(reasons, HealthReasonUnknownWorkspaces)
	}
	if in.RAMPercent > float64(ceiling) {
		reasons = append(reasons, HealthReasonRAMAboveSafetyCeiling)
	}

	if len(reasons) == 0 {
		return HealthStatusOperational, []string{}
	}

	for _, r := range reasons {
		if r == HealthReasonDockerUnreachable || r == HealthReasonOrphanedWorkspaces {
			return HealthStatusCritical, reasons
		}
	}
	return HealthStatusDegraded, reasons
}
