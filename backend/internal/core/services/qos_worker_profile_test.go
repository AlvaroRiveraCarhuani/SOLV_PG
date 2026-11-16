package services

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type qosRepoStub struct {
	domain.PracticeEnvironmentRepository
	active  []*domain.PracticeEnvironment
	updated []string
}

func (r *qosRepoStub) GetActivePracticeEnvironments(context.Context) ([]*domain.PracticeEnvironment, error) {
	return r.active, nil
}

func (r *qosRepoStub) UpdateStatus(_ context.Context, id, status string) error {
	r.updated = append(r.updated, id+":"+status)
	return nil
}

type qosDockerStub struct {
	domain.PracticeEnvironmentOrchestrator
	stopped []string
	metrics *domain.ContainerMetrics
}

func (d *qosDockerStub) GetContainerMetrics(context.Context, string) (*domain.ContainerMetrics, error) {
	return d.metrics, nil
}

func (d *qosDockerStub) StopAndRemoveContainer(_ context.Context, containerID string) error {
	d.stopped = append(d.stopped, containerID)
	return nil
}

type qosHostStub struct{ domain.HostMonitor }

// El ciclo corre con el contenedor por debajo del 80% de memoria, así que el
// auto-bursting no interviene y lo único que puede apagar un entorno es la
// inactividad.
func TestQoSWorker_ElExamenNoSeDuermePorInactividad(t *testing.T) {
	containerExam := "c-exam"
	containerPractice := "c-practice"
	repo := &qosRepoStub{active: []*domain.PracticeEnvironment{
		{ID: "ws-exam", ContainerID: &containerExam, EnvironmentProfile: domain.EnvironmentProfileExam},
		{ID: "ws-practice", ContainerID: &containerPractice, EnvironmentProfile: domain.EnvironmentProfilePractice},
	}}
	docker := &qosDockerStub{metrics: &domain.ContainerMetrics{
		IsRunning:        true,
		MemoryLimitBytes: 512 * 1024 * 1024,
		MemoryUsageBytes: 10 * 1024 * 1024,
	}}

	worker := NewQoSOrchestratorWorker(repo, docker, qosHostStub{}, time.Minute, time.Second)
	worker.runCycle(context.Background())

	if len(docker.stopped) != 1 || docker.stopped[0] != containerPractice {
		t.Fatalf("solo la práctica debe apagarse, apagados: %v", docker.stopped)
	}
	if len(repo.updated) != 1 || repo.updated[0] != "ws-practice:"+domain.PracticeEnvironmentStatusHibernated {
		t.Fatalf("estado inesperado en el repositorio: %v", repo.updated)
	}
}
