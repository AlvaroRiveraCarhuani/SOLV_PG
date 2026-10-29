package services

import (
	"context"
	"log"

	"solv-backend/internal/core/domain"
)

// satelliteReserveMB devuelve el techo de cgroup que un motor satélite ocupa en el nodo.
// Es la reserva que la admisión descuenta una sola vez por motor, nunca por laboratorio:
// el contenedor es compartido y su memoria no se multiplica por alumno.
func satelliteReserveMB(engine string) int64 {
	spec, ok := managedSatelliteSpec(engine)
	if !ok {
		return 0
	}
	return spec.MemoryLimitMB
}

// nodeSatelliteReserveMB suma la reserva de los motores gestionados que el nodo necesita
// mantener vivos: los declarados por la plantilla solicitada y los de los laboratorios
// activos, sin duplicar el mismo motor. Un motor con destino externo declarado por el
// operador no consume memoria del nodo.
//
// Si no se puede leer la plantilla de un laboratorio activo se reserva el conjunto
// completo de motores gestionados: ante la duda, la admisión no acepta de más.
func (s *WorkspaceService) nodeSatelliteReserveMB(ctx context.Context, active []*domain.WorkspaceInstance, requested *domain.Template) int64 {
	engines := make(map[string]bool, len(managedSatelliteEngines))
	for _, engine := range templateManagedEngines(requested) {
		engines[engine] = true
	}

	if s.templateRepo != nil {
		resolved := make(map[string]bool, len(active))
		for _, ws := range active {
			if ws == nil || ws.TemplateID == nil || *ws.TemplateID == "" || resolved[*ws.TemplateID] {
				continue
			}
			resolved[*ws.TemplateID] = true
			tpl, err := s.templateRepo.GetTemplateByID(ctx, *ws.TemplateID)
			if err != nil || tpl == nil {
				log.Printf("[Admission] WARNING: no se pudo resolver la plantilla %s de un laboratorio activo: se reserva el conjunto completo de satélites", *ws.TemplateID)
				for _, engine := range managedSatelliteEngines {
					engines[engine] = true
				}
				continue
			}
			for _, engine := range templateManagedEngines(tpl) {
				engines[engine] = true
			}
		}
	}

	var reserveMB int64
	for engine := range engines {
		if managedSatelliteHostOverride(engine) != "" {
			continue
		}
		reserveMB += satelliteReserveMB(engine)
	}
	return reserveMB
}

// admissionBudgetDenied decide si el presupuesto comprometido del nodo admite un
// laboratorio nuevo. committedMB es la suma de los techos de los laboratorios activos;
// labLimitMB es el techo del contenedor del alumno que se quiere admitir y reserveMB la
// memoria que los contenedores satélite compartidos ocupan una sola vez en el nodo.
func admissionBudgetDenied(committedMB, labLimitMB, reserveMB, hostMB, ceilingPct int64) bool {
	if hostMB <= 0 {
		return false
	}
	if ceilingPct <= 0 {
		ceilingPct = 85
	}
	allocatableMB := hostMB * ceilingPct / 100
	return committedMB+labLimitMB+reserveMB > allocatableMB
}
