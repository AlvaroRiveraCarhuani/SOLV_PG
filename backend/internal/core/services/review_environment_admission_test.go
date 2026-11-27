package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// El preview aparta su memoria del mismo presupuesto que los laboratorios: con
// la clase ocupando el nodo, no abre un contenedor más ni tumba un entorno.
func TestReviewEnvironmentStart_ComparteElPresupuestoConLosLaboratorios(t *testing.T) {
	svc, repo, source, orchestrator := newReviewEnvironmentFixture(t)

	admission := NewPracticeEnvironmentService(nil, nil, nil)
	admission.WithHostTotalMemory(func() (int, bool) { return 8192, true })
	svc.SetAdmissionService(admission)

	// La clase ocupa el nodo entero: 8192 MB en vuelo sobre un techo de
	// 8192 × 85% = 6963 MB deja el presupuesto sin hueco para una base.
	admission.admissionMu.Lock()
	admission.admissionInFlightMB = 8192
	admission.admissionMu.Unlock()

	_, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID)
	if !errors.Is(err, domain.ErrReviewEnvironmentHostMemory) {
		t.Fatalf("esperaba ErrReviewEnvironmentHostMemory, obtuve %v", err)
	}
	if repo.created != 0 || orchestrator.startCalls != 0 {
		t.Fatalf("un preview sin hueco no crea volumen ni contenedor: created=%d start=%d", repo.created, orchestrator.startCalls)
	}
}

// Con hueco, el preview reserva, arranca y solo devuelve la memoria cuando el
// contenedor y su volumen se retiran.
func TestReviewEnvironmentStart_LiberaLaReservaAlMorirElPreview(t *testing.T) {
	svc, repo, source, _ := newReviewEnvironmentFixture(t)

	admission := NewPracticeEnvironmentService(nil, nil, nil)
	admission.WithHostTotalMemory(func() (int, bool) { return 131072, true })
	svc.SetAdmissionService(admission)

	if _, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID); err != nil {
		t.Fatalf("Start: %v", err)
	}

	admission.admissionMu.Lock()
	inFlightWhileRunning := admission.admissionInFlightMB
	admission.admissionMu.Unlock()
	if inFlightWhileRunning <= 0 {
		t.Fatalf("el preview vivo debe contar en el presupuesto en vuelo, hay %d MB", inFlightWhileRunning)
	}

	// El QoS apaga el preview: contenedor y volumen se retiran juntos.
	svc.release(context.Background(), repo.env)

	admission.admissionMu.Lock()
	inFlightAfterRelease := admission.admissionInFlightMB
	admission.admissionMu.Unlock()
	if inFlightAfterRelease != 0 {
		t.Fatalf("al morir el preview la reserva debía volver al nodo, quedan %d MB", inFlightAfterRelease)
	}
}

// El preview toma su cuota base de la plantilla y nunca por debajo del piso de
// 512 MB: un entorno chico no abre un contenedor que el editor desborde.
func TestReviewEnvironmentStart_ReservaAlMenosElPisoDeMemoria(t *testing.T) {
	svc, _, source, _ := newReviewEnvironmentFixture(t)
	source.context.TemplateMemoryMB = 128

	admission := NewPracticeEnvironmentService(nil, nil, nil)
	admission.WithHostTotalMemory(func() (int, bool) { return 131072, true })
	svc.SetAdmissionService(admission)

	if _, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID); err != nil {
		t.Fatalf("Start: %v", err)
	}

	admission.admissionMu.Lock()
	defer admission.admissionMu.Unlock()
	if admission.admissionInFlightMB < domain.ReviewEnvironmentMinMemoryMB {
		t.Fatalf("la reserva no puede quedar por debajo del piso de %d MB, quedó en %d MB",
			domain.ReviewEnvironmentMinMemoryMB, admission.admissionInFlightMB)
	}
}

// Un fallo de arranque devuelve la memoria apartada: un preview roto no debe
// consumir el presupuesto hasta que venza el TTL.
func TestReviewEnvironmentStart_FalloDevuelveLaReserva(t *testing.T) {
	svc, _, source, orchestrator := newReviewEnvironmentFixture(t)
	orchestrator.startErr = errors.New("daemon no disponible")

	admission := NewPracticeEnvironmentService(nil, nil, nil)
	admission.WithHostTotalMemory(func() (int, bool) { return 131072, true })
	svc.SetAdmissionService(admission)

	if _, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID); err == nil {
		t.Fatal("el arranque debía fallar")
	}

	admission.admissionMu.Lock()
	defer admission.admissionMu.Unlock()
	if admission.admissionInFlightMB != 0 {
		t.Fatalf("el fallo debe devolver la reserva, quedan %d MB", admission.admissionInFlightMB)
	}
}

// El preview reutilizado no vuelve a reservar: su memoria ya estaba apartada.
func TestReviewEnvironmentStart_ReutilizaSinReservarDosVeces(t *testing.T) {
	svc, repo, source, _ := newReviewEnvironmentFixture(t)

	admission := NewPracticeEnvironmentService(nil, nil, nil)
	admission.WithHostTotalMemory(func() (int, bool) { return 131072, true })
	svc.SetAdmissionService(admission)

	if _, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID); err != nil {
		t.Fatalf("primer Start: %v", err)
	}
	if repo.env.Status != domain.ReviewEnvironmentStatusRunning {
		t.Fatalf("el preview debía quedar running, está %s", repo.env.Status)
	}

	admission.admissionMu.Lock()
	reservedOnce := admission.admissionInFlightMB
	admission.admissionMu.Unlock()

	// El registro queda vigente: el segundo Start reabre el mismo preview.
	repo.env.CreatedAt = time.Now().UTC()
	if _, err := svc.Start(context.Background(), "tenant-1", "teacher-1", source.context.SubmissionID); err != nil {
		t.Fatalf("segundo Start: %v", err)
	}

	admission.admissionMu.Lock()
	defer admission.admissionMu.Unlock()
	if admission.admissionInFlightMB != reservedOnce {
		t.Fatalf("reabrir no debe volver a reservar: antes %d MB, ahora %d MB", reservedOnce, admission.admissionInFlightMB)
	}
}
