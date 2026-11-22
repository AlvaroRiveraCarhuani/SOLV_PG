package services

import (
	"testing"
	"time"
)

// Dos pausas seguidas sobre el mismo laboratorio no pueden reconstruir el
// contenedor a la vez: comparten el nombre y la segunda creación falla. El
// bloqueo por entorno hace que la segunda espere y retome el contenedor vigente.
func TestLockEnvironmentMount_SerializesTheSameEnvironment(t *testing.T) {
	service := &PracticeEnvironmentService{}

	first := service.lockEnvironmentMount("env-1")
	acquired := make(chan struct{})
	go func() {
		unlock := service.lockEnvironmentMount("env-1")
		close(acquired)
		unlock()
	}()

	select {
	case <-acquired:
		t.Fatal("el segundo montaje entró antes de que terminara el primero")
	case <-time.After(50 * time.Millisecond):
	}

	first()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("el segundo montaje no entró después de liberar el primero")
	}
}

// Laboratorios distintos no se bloquean entre sí: la pausa de un aula no tiene
// que esperar a la de otro entorno.
func TestLockEnvironmentMount_DoesNotSerializeDifferentEnvironments(t *testing.T) {
	service := &PracticeEnvironmentService{}

	first := service.lockEnvironmentMount("env-1")
	defer first()

	acquired := make(chan struct{})
	go func() {
		unlock := service.lockEnvironmentMount("env-2")
		close(acquired)
		unlock()
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("un laboratorio distinto quedó bloqueado por otro")
	}
}
