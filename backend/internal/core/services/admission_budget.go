package services

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"

	"solv-backend/internal/core/domain"
)

var (
	// ErrBuildBusy indica que ya hay una construcción de imagen en curso en el
	// nodo: el semáforo global de construcción es 1 y la segunda espera un turno.
	ErrBuildBusy = errors.New("ya hay una construcción de imagen en curso en el servidor")
	// ErrHostCPUSaturated indica que el load average del nodo no admite una
	// operación pesada (build o preview) en este momento.
	ErrHostCPUSaturated = errors.New("el servidor está procesando muchas tareas a la vez")
)

// Semáforo global de construcción: nunca corren dos `docker build` simultáneos
// en el mismo nodo. Lo comparten los servicios que construyen imágenes locales
// (pruebas de entorno y calentamiento de plantillas).
var globalBuildSemaphore = make(chan struct{}, 1)

// hostLoadProbe mide el load average de 1 minuto del nodo. Lo conecta el
// proceso principal con la lectura de infraestructura; sin sonda la compuerta
// de CPU queda abierta y solo manda el semáforo.
var (
	hostLoadProbeMu sync.RWMutex
	hostLoadProbe   func() (float64, error)
)

// SetHostLoadProbe conecta la medición de carga del nodo con la compuerta de
// operaciones pesadas.
func SetHostLoadProbe(probe func() (float64, error)) {
	hostLoadProbeMu.Lock()
	defer hostLoadProbeMu.Unlock()
	hostLoadProbe = probe
}

// currentHostLoad1 devuelve el load average de 1 minuto o 0 si aún no hay
// sonda o si la lectura falla: ante la duda, la compuerta deja pasar.
func currentHostLoad1() float64 {
	hostLoadProbeMu.RLock()
	probe := hostLoadProbe
	hostLoadProbeMu.RUnlock()
	if probe == nil {
		return 0
	}
	load1, err := probe()
	if err != nil {
		log.Printf("[Admission] no se pudo medir la carga del nodo: se omite la compuerta de CPU: %v", err)
		return 0
	}
	return load1
}

// acquireBuildSlot toma el único turno de construcción del nodo. Si ya hay una
// construcción en curso devuelve ErrBuildBusy sin esperar: el segundo build
// espera su turno con un 429, no compite con el primero.
func acquireBuildSlot(ctx context.Context) (func(), error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !domain.HeavyOperationAllowed(currentHostLoad1(), runtime.NumCPU()) {
		log.Printf("[Admission] CPU del nodo saturada: se rechaza la operación pesada")
		return nil, ErrHostCPUSaturated
	}
	select {
	case globalBuildSemaphore <- struct{}{}:
		var mu sync.Mutex
		released := false
		return func() {
			mu.Lock()
			defer mu.Unlock()
			if released {
				return
			}
			released = true
			<-globalBuildSemaphore
		}, nil
	default:
		return nil, ErrBuildBusy
	}
}

// tryAcquireBuildSlotForTest toma el turno directamente, sin compuerta de CPU,
// para los tests de concurrencia del semáforo.
func tryAcquireBuildSlotForTest() (func(), error) {
	select {
	case globalBuildSemaphore <- struct{}{}:
		var mu sync.Mutex
		released := false
		return func() {
			mu.Lock()
			defer mu.Unlock()
			if released {
				return
			}
			released = true
			<-globalBuildSemaphore
		}, nil
	default:
		return nil, ErrBuildBusy
	}
}

// reserveSharedQuota aparta memoria del presupuesto en vuelo que comparten los
// laboratorios, los previews de entrega y las pruebas de entorno. Cuando el
// servicio no tiene políticas configuradas se comporta como hacía antes: solo
// chequea que la memoria viva alcance.
func (s *PracticeEnvironmentService) reserveSharedQuota(ctx context.Context, baseMB int64, template *domain.Template) (int64, func(), error) {
	active, _ := s.listActiveForCapacity(ctx)
	policies := s.effectivePolicies(ctx, domain.GetTenantID(ctx))
	ceilingPct := policies.HostSafetyCeilingPct
	if ceilingPct <= 0 {
		ceilingPct = 85
	}

	// Sin políticas guardadas rigen los defaults del dominio: el presupuesto
	// compartido se calcula igual, porque el nodo es el mismo para todos.
	hostMB, measured := s.hostTotalMB()
	if !measured || hostMB <= 0 {
		if s.hostMonitor != nil && !s.hostMonitor.CanAllocateMemory(baseMB) {
			return 0, func() {}, ErrHostMemoryExhausted
		}
		// Sin medición del host no se puede presupuestar: se deja pasar solo lo
		// que la guarda viva de memoria acepta.
		log.Printf("[Admission] no se pudo medir la RAM total del host: el presupuesto compartido queda en manos de la guarda de memoria")
		return baseMB, func() {}, nil
	}

	reserveMB := s.nodeSatelliteReserveMB(ctx, active, template)
	grantedMB, release, grantErr := s.reserveAdmissionQuota(active, baseMB, reserveMB, int64(hostMB), ceilingPct)
	if grantErr != nil {
		return 0, func() {}, grantErr
	}

	// La guarda viva de memoria sigue mandando además del presupuesto: el
	// hueco libre puede haber caído por otros procesos desde la medición.
	if s.hostMonitor != nil && !s.hostMonitor.CanAllocateMemory(grantedMB) {
		release()
		return 0, func() {}, ErrHostMemoryExhausted
	}
	return grantedMB, release, nil
}

// ReservePreviewQuota aparta la memoria de un preview de entrega del mismo
// presupuesto en vuelo que los laboratorios: un preview compite con la clase.
func (s *PracticeEnvironmentService) ReservePreviewQuota(ctx context.Context, baseMB int64) (int64, func(), error) {
	if s == nil {
		return baseMB, func() {}, nil
	}
	return s.reserveSharedQuota(ctx, baseMB, nil)
}

// ReservePracticeTestQuota aparta la memoria de una prueba de entorno del
// mismo presupuesto en vuelo que los laboratorios.
func (s *PracticeEnvironmentService) ReservePracticeTestQuota(ctx context.Context, baseMB int64) (int64, func(), error) {
	if s == nil {
		return baseMB, func() {}, nil
	}
	return s.reserveSharedQuota(ctx, baseMB, nil)
}

// listActiveForCapacity lista los entornos vivos del nodo para la admisión
// compartida. Un error de conteo no bloquea: se admite con lo que se pudo leer.
func (s *PracticeEnvironmentService) listActiveForCapacity(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	active, err := s.repo.GetActivePracticeEnvironments(ctx)
	if err != nil {
		log.Printf("[Admission] WARNING: no se pudieron listar los entornos activos para la admisión compartida: %v", err)
		return nil, err
	}
	return active, nil
}

// capacitySnapshot devuelve lo que el dashboard de capacidad necesita: RAM
// total medida, memoria comprometida (entornos y satélites) y techo de
// seguridad vigente. Sin medición no hay números: ok=false sin inventar.
func (s *PracticeEnvironmentService) capacitySnapshot(ctx context.Context, template *domain.Template) (hostMB, committedMB int64, ceilingPct int, ok bool) {
	policies := s.effectivePolicies(ctx, domain.GetTenantID(ctx))
	ceilingPct = policies.HostSafetyCeilingPct
	if ceilingPct <= 0 {
		ceilingPct = 85
	}

	hostRAMMB, measured := s.hostTotalMB()
	if !measured || hostRAMMB <= 0 {
		return 0, 0, ceilingPct, false
	}

	active, _ := s.listActiveForCapacity(ctx)
	reserveMB := s.nodeSatelliteReserveMB(ctx, active, template)

	committedMB = reserveMB
	for _, ws := range active {
		if ws != nil {
			committedMB += ws.MemoryLimitMB
		}
	}

	s.admissionMu.Lock()
	committedMB += s.admissionInFlightMB
	s.admissionMu.Unlock()

	return int64(hostRAMMB), committedMB, ceilingPct, true
}

// capacityImpact arma el bloque en lenguaje humano para un entorno que pide
// ramPerEnvMB por estudiante en el servidor actual.
func (s *PracticeEnvironmentService) capacityImpact(ctx context.Context, template *domain.Template, ramPerEnvMB int64) *domain.CapacityImpact {
	hostMB, committedMB, ceilingPct, ok := s.capacitySnapshot(ctx, template)
	if !ok {
		return nil
	}
	impact := domain.BuildCapacityImpact(hostMB, ceilingPct, ramPerEnvMB, committedMB)
	return &impact
}
