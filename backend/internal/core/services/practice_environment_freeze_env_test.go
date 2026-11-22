package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

type fakeCompanionEnvDocker struct {
	domain.PracticeEnvironmentOrchestrator
	env []string
	err error
	got string
}

func (f *fakeCompanionEnvDocker) ReadCompanionEnv(_ context.Context, containerID string) ([]string, error) {
	f.got = containerID
	return f.env, f.err
}

// La pausa corre con el contexto del docente y reconstruye el contenedor del
// alumno: la configuración del Companion tiene que copiarse del contenedor que
// se reemplaza, no regenerarse con el rol y el token de quien pausa.
func TestPreserveCompanionEnv_CopiesTheStudentConfiguration(t *testing.T) {
	docker := &fakeCompanionEnvDocker{
		env: []string{"SOLV_API_URL=https://solv.dedyn.io", "SOLV_STUDENT_TOKEN=alumno", "SOLV_USER_ROLE=student"},
	}

	env := preserveCompanionEnv(context.Background(), docker, "container-1")

	if docker.got != "container-1" {
		t.Fatalf("se esperaba leer el contenedor vigente, fue %q", docker.got)
	}
	if len(env) != 3 || env[1] != "SOLV_STUDENT_TOKEN=alumno" || env[2] != "SOLV_USER_ROLE=student" {
		t.Fatalf("no se conservaron las variables del alumno: %+v", env)
	}
}

func TestPreserveCompanionEnv_WithoutContainerDoesNotRead(t *testing.T) {
	docker := &fakeCompanionEnvDocker{env: []string{"SOLV_API_URL=https://solv.dedyn.io"}}

	if env := preserveCompanionEnv(context.Background(), docker, "  "); env != nil {
		t.Fatalf("no se esperaba configuración sin contenedor: %+v", env)
	}
	if docker.got != "" {
		t.Fatalf("no se debía inspeccionar ningún contenedor, fue %q", docker.got)
	}
}

func TestPreserveCompanionEnv_ReadFailureDoesNotStopTheRebuild(t *testing.T) {
	docker := &fakeCompanionEnvDocker{err: errors.New("docker no responde")}

	if env := preserveCompanionEnv(context.Background(), docker, "container-1"); env != nil {
		t.Fatalf("un fallo de lectura no debía devolver variables: %+v", env)
	}
}
