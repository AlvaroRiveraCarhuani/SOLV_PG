package httpdelivery

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

type fakeStudentDirectory struct {
	items []*domain.AdminStudentDirectoryItem
	err   error
}

func (f *fakeStudentDirectory) ListStudentsDirectory(_ context.Context, _, _, _, _, _ string) ([]*domain.AdminStudentDirectoryItem, error) {
	return f.items, f.err
}

// fakeMetricsOrchestrator implementa WorkspaceOrchestrator embebiendo la
// interfaz (nil) y sobrescribiendo solo GetContainerMetrics.
type fakeMetricsOrchestrator struct {
	domain.WorkspaceOrchestrator
	metrics *domain.ContainerMetrics
	err     error
}

func (f *fakeMetricsOrchestrator) GetContainerMetrics(_ context.Context, _ string) (*domain.ContainerMetrics, error) {
	return f.metrics, f.err
}

const testStudentUUID = "1c444b04-4270-496f-95a1-86d76a9b3521"

func TestStudentIdentitiesUsaNombreRealYNuncaExponeUUID(t *testing.T) {
	h := &AdminHandler{
		studentDir: &fakeStudentDirectory{
			items: []*domain.AdminStudentDirectoryItem{
				{ID: testStudentUUID, FirstName: "AlumnoA1", LastName: "Estudiante", Email: "alumno_a1_1c444b04@uab.edu.bo"},
			},
		},
	}

	identities := h.studentIdentities(context.Background(), "tenant-1", []string{testStudentUUID})

	got := identities[testStudentUUID]
	if got.Name != "AlumnoA1 Estudiante" {
		t.Errorf("nombre real esperado, obtenido %q", got.Name)
	}
	if got.Email != "alumno_a1_1c444b04@uab.edu.bo" {
		t.Errorf("correo real esperado, obtenido %q", got.Email)
	}
	if strings.Contains(got.Name, testStudentUUID) || strings.Contains(got.Email, testStudentUUID) {
		t.Errorf("la identidad expone el UUID: %q / %q", got.Name, got.Email)
	}
}

func TestStudentIdentitiesFallbackSinDirectorioNoExponeUUIDs(t *testing.T) {
	h := &AdminHandler{}

	identities := h.studentIdentities(context.Background(), "tenant-1", []string{testStudentUUID})
	got := identities[testStudentUUID]

	if got.Name != "Estudiante" {
		t.Errorf("rótulo neutro esperado, obtenido %q", got.Name)
	}
	if got.Email != "" {
		t.Errorf("no debe inventarse un correo, obtenido %q", got.Email)
	}
	if strings.Contains(got.Name, testStudentUUID) || strings.Contains(got.Email, testStudentUUID) {
		t.Errorf("el fallback expone el UUID: %q / %q", got.Name, got.Email)
	}
	if strings.Contains(got.Email, "@uab.edu.bo") {
		t.Errorf("el fallback no debe concatenar dominios sobre el UUID: %q", got.Email)
	}
}

func TestStudentIdentitiesDirectorioConErrorNoRompeYNoExponeUUID(t *testing.T) {
	h := &AdminHandler{
		studentDir: &fakeStudentDirectory{err: context.DeadlineExceeded},
	}

	identities := h.studentIdentities(context.Background(), "tenant-1", []string{testStudentUUID})
	got := identities[testStudentUUID]

	if got.Name != "Estudiante" {
		t.Errorf("ante error del directorio se espera rótulo neutro, obtenido %q", got.Name)
	}
	if strings.Contains(got.Name+got.Email, testStudentUUID) {
		t.Errorf("el fallback por error expone el UUID: %q / %q", got.Name, got.Email)
	}
}

func TestContainerUsageMBSinOrchestratorNoInventaMemoria(t *testing.T) {
	h := &AdminHandler{}
	ws := &domain.WorkspaceInstance{ContainerID: &[]string{"abc123"}[0]}

	mb, available := h.containerUsageMB(context.Background(), ws)
	if mb != 0 || available {
		t.Errorf("sin métricas se espera (0, false), obtenido (%d, %v)", mb, available)
	}
}

func TestContainerUsageMBLeeUsoRealDeDocker(t *testing.T) {
	containerID := "abc123"
	h := &AdminHandler{
		orchestrator: &fakeMetricsOrchestrator{
			metrics: &domain.ContainerMetrics{
				IsRunning:          true,
				MemoryUsageBytes:   256 * 1024 * 1024,
				MemoryLimitBytes:   512 * 1024 * 1024,
			},
		},
	}
	ws := &domain.WorkspaceInstance{ContainerID: &containerID}

	mb, available := h.containerUsageMB(context.Background(), ws)
	if !available {
		t.Fatal("debe haber métricas disponibles con contenedor en ejecución")
	}
	if mb != 256 {
		t.Errorf("256 MB esperados, obtenidos %d", mb)
	}
}

func TestContainerUsageMBContenedorDetenidoNoDisponible(t *testing.T) {
	containerID := "abc123"
	h := &AdminHandler{
		orchestrator: &fakeMetricsOrchestrator{
			metrics: &domain.ContainerMetrics{IsRunning: false},
		},
	}
	ws := &domain.WorkspaceInstance{ContainerID: &containerID}

	if _, available := h.containerUsageMB(context.Background(), ws); available {
		t.Error("contenedor detenido no debe reportar métricas disponibles")
	}
}

func TestCourseNamesFallbackSinRepoNoExponeUUID(t *testing.T) {
	h := &AdminHandler{}
	names := h.courseNames(context.Background(), "tenant-1", []string{"aaaa-bbbb-cccc"})

	if names["aaaa-bbbb-cccc"] != "Materia" {
		t.Errorf("rótulo neutro esperado, obtenido %q", names["aaaa-bbbb-cccc"])
	}
}
