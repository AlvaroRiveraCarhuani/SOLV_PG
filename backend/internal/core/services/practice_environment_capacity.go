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
func (s *PracticeEnvironmentService) nodeSatelliteReserveMB(ctx context.Context, active []*domain.PracticeEnvironment, requested *domain.Template) int64 {
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

// admissionAllocatableMB es la RAM que la admisión reparte entre laboratorios:
// el techo de seguridad aplicado sobre la RAM total medida del nodo.
//
// La reserva de plataforma (PlatformReserveMB) es un supuesto de planificación
// del panel de capacidad, no una compuerta de admisión: descontarla acá negaría
// entornos que hoy entran. El techo de seguridad ya deja ese margen libre.
func admissionAllocatableMB(hostMB, ceilingPct int64) int64 {
	if hostMB <= 0 {
		return 0
	}
	if ceilingPct <= 0 || ceilingPct > 100 {
		ceilingPct = 85
	}
	return hostMB * ceilingPct / 100
}

// loanQuotaMB decide cuánta memoria recibe un entorno nuevo con el presupuesto
// que sobra en el nodo.
//
// La cuota base la fijan la política del tenant y la plantilla, y es el piso:
// ningún alumno recibe menos. Mientras el nodo tenga hueco, la cuota se dobla
// hasta el techo duro, pero solo si al concederla sigue entrando otro entorno
// del mismo tamaño. Así el préstamo no deja sin lugar al siguiente alumno que
// abra su laboratorio, y cuando la clase se llena la base vuelve a ser la cuota.
//
// committedMB es la suma de los techos de los laboratorios activos y reserveMB
// la memoria que los satélites compartidos ocupan una sola vez en el nodo.
// Devuelve 0 cuando ni la base entra: la admisión rechaza el entorno.
//
// Solo concede perfiles reconocidos: una cuota de 3072 MB no la entiende nadie
// que mire el panel ni una plantilla.
func loanQuotaMB(baseMB, committedMB, reserveMB, hostMB, ceilingPct, hardQuotaMB int64) int64 {
	if baseMB <= 0 {
		return 0
	}
	budgetMB := admissionAllocatableMB(hostMB, ceilingPct) - committedMB - reserveMB
	if budgetMB < baseMB {
		return 0
	}
	if hardQuotaMB < baseMB {
		hardQuotaMB = baseMB
	}

	grantedMB := baseMB
	for _, tierMB := range loanQuotaTiersMB {
		if tierMB <= baseMB {
			continue
		}
		if tierMB > hardQuotaMB {
			break
		}
		// El préstamo deja sitio para otro entorno del tamaño que concede.
		if tierMB*2 > budgetMB {
			break
		}
		grantedMB = tierMB
	}
	return grantedMB
}

// loanQuotaTiersMB son los perfiles que el préstamo puede conceder, de menor a
// mayor. Son los mismos niveles con los que el administrador define plantillas:
// el préstamo no inventa tamaños que la interfaz no pueda explicar.
var loanQuotaTiersMB = []int64{512, 768, 1024, 1536, 2048, 4096}

// reserveAdmissionQuota decide la cuota de un entorno nuevo y aparta ese monto
// del presupuesto mientras se crea su registro. El apartado cubre el hueco
// entre la decisión y la fila en la base: dos aperturas simultáneas no pueden
// quedarse con el mismo hueco libre. Devuelve la cuota concedida y la función
// que libera la reserva, que debe llamarse cuando el entorno ya tiene su fila o
// cuando el alta falló.
func (s *PracticeEnvironmentService) reserveAdmissionQuota(active []*domain.PracticeEnvironment, baseMB, reserveMB, hostMB int64, ceilingPct int) (int64, func(), error) {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()

	var committedMB int64
	for _, ws := range active {
		if ws != nil {
			committedMB += ws.MemoryLimitMB
		}
	}
	// Lo que está en vuelo todavía no tiene fila y también ocupa lugar.
	committedMB += s.admissionInFlightMB

	grantedMB := loanQuotaMB(baseMB, committedMB, reserveMB, hostMB, int64(ceilingPct), domain.HardQuotaMemoryMB)
	if grantedMB <= 0 {
		return 0, func() {}, ErrHostMemoryExhausted
	}

	s.admissionInFlightMB += grantedMB
	released := false
	release := func() {
		s.admissionMu.Lock()
		defer s.admissionMu.Unlock()
		if released {
			return
		}
		released = true
		s.admissionInFlightMB -= grantedMB
	}
	return grantedMB, release, nil
}

// loanQuotaForReactivation recalcula la cuota de un entorno que se reconstruye al
// despertar. El préstamo no sobrevive a la hibernación: al volver, la cuota se
// decide otra vez con el presupuesto del momento y nunca queda por debajo de la
// base. Un entorno que no cabe no se rechaza —ya existe—: despierta con la base.
//
// El propio entorno se excluye del comprometido: al hibernar dejó de contar y
// volver a sumarlo sería contar dos veces el mismo cupo.
func (s *PracticeEnvironmentService) loanQuotaForReactivation(ctx context.Context, instance *domain.PracticeEnvironment, baseMB int64, active []*domain.PracticeEnvironment, ceilingPct int) int64 {
	if instance == nil || baseMB <= 0 {
		return baseMB
	}
	hostMB, measured := s.hostTotalMB()
	if !measured || hostMB <= 0 {
		return baseMB
	}
	template := s.resolveTemplateForInstance(ctx, instance)
	reserveMB := s.nodeSatelliteReserveMB(ctx, active, template)

	var committedMB int64
	for _, ws := range active {
		if ws == nil || ws.ID == instance.ID {
			continue
		}
		committedMB += ws.MemoryLimitMB
	}

	grantedMB := loanQuotaMB(baseMB, committedMB, reserveMB, int64(hostMB), int64(ceilingPct), domain.HardQuotaMemoryMB)
	if grantedMB <= 0 {
		return baseMB
	}
	if grantedMB > baseMB {
		log.Printf("[Admission Loan] el entorno %s despierta con %d MB sobre una base de %d MB", instance.ID, grantedMB, baseMB)
	}
	return grantedMB
}
