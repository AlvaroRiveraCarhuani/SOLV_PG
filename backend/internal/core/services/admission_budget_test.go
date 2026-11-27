package services

import (
	"context"
	"errors"
	"sync"
	"testing"

	"solv-backend/internal/core/domain"
)

// El preview de una entrega compite con el laboratorio en el mismo presupuesto
// en vuelo: mientras uno aparta memoria, el otro ve ese monto tomado.
func TestReservePreviewQuota_CompiteConLaboratorioEnElMismoPresupuesto(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)
	svc.WithHostTotalMemory(func() (int, bool) { return 12000, true })
	svc.policiesService = nil // sin catálogo de políticas: el presupuesto sigue vivo

	grantedLab, releaseLab, err := svc.reserveAdmissionQuota(nil, 4096, 0, 12000, 85)
	if err != nil {
		t.Fatalf("el laboratorio debía entrar: %v", err)
	}
	if grantedLab != 4096 {
		t.Fatalf("el laboratorio esperaba 4096 MB, obtuvo %d MB", grantedLab)
	}

	// El preview mira el mismo presupuesto y ya no ve el hueco completo.
	grantedPreview, releasePreview, err := svc.reserveAdmissionQuota(nil, 512, 0, 12000, 85)
	if err != nil {
		t.Fatalf("el preview debía entrar en el hueco restante: %v", err)
	}
	if grantedPreview >= grantedLab {
		t.Fatalf("el preview no puede recibir %d MB con %d MB ya apartados por el laboratorio", grantedPreview, grantedLab)
	}

	releaseLab()
	releasePreview()

	// Sin reservas, el presupuesto vuelve entero al nodo.
	after, releaseAfter, err := svc.reserveAdmissionQuota(nil, 512, 0, 12000, 85)
	if err != nil {
		t.Fatalf("con el presupuesto liberado debía entrar: %v", err)
	}
	if after != 4096 {
		t.Fatalf("esperaba el presupuesto completo (4096 MB), obtuvo %d MB", after)
	}
	releaseAfter()
}

// Sin hueco para la base, el preview se rechaza con el mismo error que un
// laboratorio: ni se crea el contenedor ni se tumba una clase.
func TestReservePreviewQuota_SinHuecoRespondeMemoria(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)
	svc.WithHostTotalMemory(func() (int, bool) { return 12000, true })

	// Un primer preview toma el nodo entero con su préstamo.
	if _, _, err := svc.reserveAdmissionQuota(nil, 10200, 0, 12000, 100); err != nil {
		t.Fatalf("la primera reserva debía entrar: %v", err)
	}

	if _, _, err := svc.ReservePreviewQuota(context.Background(), 2048); !errors.Is(err, ErrHostMemoryExhausted) {
		t.Fatalf("esperaba ErrHostMemoryExhausted para el preview sin hueco, obtuvo %v", err)
	}
}

// Una prueba de entorno reserva del mismo presupuesto que los laboratorios.
func TestReservePracticeTestQuota_ComparteElPresupuesto(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)
	svc.WithHostTotalMemory(func() (int, bool) { return 8192, true })

	granted, release, err := svc.ReservePracticeTestQuota(context.Background(), 512)
	if err != nil {
		t.Fatalf("la prueba de entorno debía reservar: %v", err)
	}
	if granted < 512 {
		t.Fatalf("la prueba nunca recibe menos que su base: %d MB", granted)
	}

	svc.admissionMu.Lock()
	inFlight := svc.admissionInFlightMB
	svc.admissionMu.Unlock()
	if inFlight != granted {
		t.Fatalf("el presupuesto en vuelo debía ser %d MB, es %d MB", granted, inFlight)
	}

	release()
	release()

	svc.admissionMu.Lock()
	defer svc.admissionMu.Unlock()
	if svc.admissionInFlightMB != 0 {
		t.Fatalf("liberar dos veces no debe devolver el monto dos veces: quedan %d MB", svc.admissionInFlightMB)
	}
}

// El semáforo de construcción es global y de concurrencia 1: nunca dos builds
// a la vez, aunque los pidan servicios distintos.
func TestBuildSemaphore_EsUnicoPorNodo(t *testing.T) {
	release, err := tryAcquireBuildSlotForTest()
	if err != nil {
		t.Fatalf("el primer turno de construcción debía obtenerse: %v", err)
	}

	if _, err := tryAcquireBuildSlotForTest(); !errors.Is(err, ErrBuildBusy) {
		t.Fatalf("el segundo build simultáneo esperaba ErrBuildBusy, obtuvo %v", err)
	}

	release()

	second, err := tryAcquireBuildSlotForTest()
	if err != nil {
		t.Fatalf("con el turno liberado debía obtenerse: %v", err)
	}
	second()
	second() // liberar dos veces no rompe el semáforo
}

// Varias goroutines que piden el turno solo consiguen una a la vez.
func TestBuildSemaphore_SerializaGoroutines(t *testing.T) {
	const workers = 8
	var mu sync.Mutex
	inside := 0
	maxInside := 0

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := tryAcquireBuildSlotForTest()
			if err != nil {
				return
			}
			defer release()
			mu.Lock()
			inside++
			if inside > maxInside {
				maxInside = inside
			}
			mu.Unlock()
			mu.Lock()
			inside--
			mu.Unlock()
		}()
	}
	wg.Wait()

	if maxInside > 1 {
		t.Fatalf("nunca puede haber dos builds a la vez, se observaron %d", maxInside)
	}
}

// La compuerta de CPU frena operaciones pesadas con el nodo saturado.
func TestAcquireBuildSlot_RespetaLaCompuertaDeCPU(t *testing.T) {
	SetHostLoadProbe(func() (float64, error) { return 1000, nil })
	t.Cleanup(func() { SetHostLoadProbe(nil) })

	if _, err := acquireBuildSlot(context.Background()); !errors.Is(err, ErrHostCPUSaturated) {
		t.Fatalf("esperaba ErrHostCPUSaturated con el nodo saturado, obtuvo %v", err)
	}

	SetHostLoadProbe(func() (float64, error) { return 0.1, nil })
	release, err := acquireBuildSlot(context.Background())
	if err != nil {
		t.Fatalf("con el nodo en reposo la operación debía entrar: %v", err)
	}
	release()
}

// capacity_impact traduce el presupuesto a estudiantes simultáneos para
// distintos tamaños de host, sin números fijos.
func TestCapacityImpact_EscalaConElTamanoDelHost(t *testing.T) {
	tests := []struct {
		name        string
		hostMB      int64
		ceilingPct  int
		ramPerEnvMB int64
		committedMB int64
		wantLevel   string
		wantTotal   int
		wantFree    int
	}{
		{
			// Nodo de 12 GB vacío con entornos de 512 MB y techo del 85%.
			name: "host chico vacío", hostMB: 12000, ceilingPct: 85, ramPerEnvMB: 512,
			wantLevel: domain.CapacityLevelComfortable, wantTotal: 19, wantFree: 19,
		},
		{
			// El mismo entorno en un host de 128 GB.
			name: "host grande vacío", hostMB: 131072, ceilingPct: 85, ramPerEnvMB: 512,
			wantLevel: domain.CapacityLevelComfortable, wantTotal: 217, wantFree: 217,
		},
		{
			// Con la clase a la mitad, el nivel baja aunque el techo sea el mismo.
			name: "host grande con la mitad ocupada", hostMB: 131072, ceilingPct: 85, ramPerEnvMB: 512,
			committedMB: 55705, wantLevel: domain.CapacityLevelModerate, wantTotal: 217, wantFree: 108,
		},
		{
			// Casi lleno: entra muy poco más.
			name: "host grande casi lleno", hostMB: 131072, ceilingPct: 85, ramPerEnvMB: 512,
			committedMB: 107000, wantLevel: domain.CapacityLevelTight, wantTotal: 217, wantFree: 8,
		},
		{
			// Un entorno Intensivo de 4096 MB reduce la capacidad de la clase.
			name: "entorno intensivo", hostMB: 12000, ceilingPct: 85, ramPerEnvMB: 4096,
			wantLevel: domain.CapacityLevelComfortable, wantTotal: 2, wantFree: 2,
		},
		{
			// Sin medición del host no hay capacidad imaginaria.
			name: "sin medición", hostMB: 0, ceilingPct: 85, ramPerEnvMB: 512,
			wantLevel: domain.CapacityLevelTight, wantTotal: 0, wantFree: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impact := domain.BuildCapacityImpact(tt.hostMB, tt.ceilingPct, tt.ramPerEnvMB, tt.committedMB)
			if impact.EstimatedConcurrentStudents != tt.wantTotal {
				t.Fatalf("estudiantes estimados: esperaba %d, obtuve %d", tt.wantTotal, impact.EstimatedConcurrentStudents)
			}
			if impact.CurrentlyAvailableStudents != tt.wantFree {
				t.Fatalf("estudiantes disponibles: esperaba %d, obtuve %d", tt.wantFree, impact.CurrentlyAvailableStudents)
			}
			if impact.Level != tt.wantLevel {
				t.Fatalf("nivel: esperaba %q, obtuve %q", tt.wantLevel, impact.Level)
			}
			if impact.Summary == "" {
				t.Fatalf("el resumen humano no puede estar vacío")
			}
		})
	}
}

// La compuerta de carga usa los núcleos del nodo como referencia.
func TestHeavyOperationAllowed(t *testing.T) {
	tests := []struct {
		name   string
		load1  float64
		numCPU int
		want   bool
	}{
		{name: "nodo en reposo", load1: 0.4, numCPU: 4, want: true},
		{name: "justo debajo del umbral", load1: 5.9, numCPU: 4, want: true},
		{name: "por encima del umbral", load1: 6.1, numCPU: 4, want: false},
		{name: "sin núcleos medidos", load1: 1.6, numCPU: 0, want: false},
		{name: "lectura negativa no frena", load1: -1, numCPU: 4, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.HeavyOperationAllowed(tt.load1, tt.numCPU); got != tt.want {
				t.Fatalf("esperaba %v, obtuve %v", tt.want, got)
			}
		})
	}
}

// El aviso de disco se dispara antes del piso duro del 5%.
func TestDiskAlertRecommended(t *testing.T) {
	if !domain.DiskAlertRecommended(9) {
		t.Fatalf("9%% libre debe avisar")
	}
	if domain.DiskAlertRecommended(domain.DefaultHostDiskFloorPct) {
		t.Fatalf("el piso duro no es un aviso: ya bloquea altas")
	}
	if domain.DiskAlertRecommended(40) {
		t.Fatalf("40%% libre no necesita aviso")
	}
}

// El impacto de la ficha del administrador usa la RAM de la plantilla resuelta.
func TestAdminRequestCapacityImpact_UsaLaPlantillaResuelta(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)
	svc.WithHostTotalMemory(func() (int, bool) { return 131072, true })

	impact := svc.capacityImpact(context.Background(), nil, 512)
	if impact == nil {
		t.Fatalf("con el nodo medido debe haber impacto de capacidad")
	}
	if impact.EstimatedConcurrentStudents <= 0 {
		t.Fatalf("esperaba estudiantes estimados positivos, obtuve %d", impact.EstimatedConcurrentStudents)
	}

	unmeasured := NewPracticeEnvironmentService(nil, nil, nil)
	unmeasured.WithHostTotalMemory(func() (int, bool) { return 0, false })
	if got := unmeasured.capacityImpact(context.Background(), nil, 512); got != nil {
		t.Fatalf("sin medición del host no debe inventarse capacidad, obtuve %+v", got)
	}
}
