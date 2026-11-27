package services_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type exportVolumeOrchestrator struct {
	*mockOrchestrator
	files  []domain.PracticeEnvironmentVolumeFile
	err    error
	volume string
	calls  int
}

func (f *exportVolumeOrchestrator) SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	f.volume = volumeName
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.files, nil
}

// WritePracticeEnvironmentVolumeFiles completa PracticeEnvironmentVolumeStore:
// la exportación solo lee, pero la capacidad del orquestador se resuelve como
// un todo.
func (f *exportVolumeOrchestrator) WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	return nil
}

// exportTemplateRepo suma al doble de plantillas la capacidad opcional del
// perfil de dependencias, sin tocar el doble que usan las pruebas anteriores.
type exportTemplateRepo struct {
	*mockTemplateRepo
	profile domain.DependencyProfile
	found   bool
	slugs   []string
	err     error
}

func (m *exportTemplateRepo) GetDependencyProfileForStack(ctx context.Context, slugs []string) (domain.DependencyProfile, bool, error) {
	m.slugs = slugs
	if m.err != nil {
		return domain.DependencyProfile{}, false, m.err
	}
	return m.profile, m.found, nil
}

func readExportZip(t *testing.T, content []byte) map[string]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("el paquete no es un zip válido: %v", err)
	}
	files := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("no se pudo abrir %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", file.Name, err)
		}
		files[file.Name] = string(body)
	}
	return files
}

func exportFixture(t *testing.T, profile domain.EnvironmentProfile, status string) (*services.PracticeEnvironmentService, *mockPracticeEnvironmentRepo, *exportVolumeOrchestrator, *exportTemplateRepo, *domain.PracticeEnvironment) {
	t.Helper()
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &exportVolumeOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		files: []domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("print('hola')\n")},
			{Path: "requirements.txt", Content: []byte("fastapi\n")},
			{Path: "../fuera.txt", Content: []byte("no debe viajar")},
		},
	}
	templateID := "tpl-node"
	tplRepo := &exportTemplateRepo{
		mockTemplateRepo: &mockTemplateRepo{
			templates: map[string]*domain.Template{
				templateID: {
					ID:          templateID,
					Name:        "Laboratorio: Python 3.12 + PostgreSQL",
					DockerImage: "solv-local/stack:abc123",
					StackConfig: domain.StackConfig{
						RuntimeID:         "python-312",
						DockerfileContent: "FROM python:3.12-slim-bookworm\nCOPY --from=gitpod/openvscode-server:latest /home/.openvscode-server /home/.openvscode-server\n",
					},
					ServicesConfig: domain.ServicesConfig{Services: []domain.ServiceRequirement{
						{Role: domain.ServiceRolePrimaryDatabase, Engine: "postgres"},
					}},
				},
			},
		},
		profile: domain.DependencyProfile{
			Version:         1,
			Detect:          []string{"requirements.txt"},
			Install:         "python3 -m venv .venv && .venv/bin/pip install -r requirements.txt",
			Materialize:     domain.DependencyMaterializeTree,
			DependencyPaths: []string{".venv"},
		},
		found: true,
	}

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(nil, tplRepo)

	practice := testPracticeID
	environment := &domain.PracticeEnvironment{
		ID:                 "eeeeeeee-1111-2222-3333-444444444444",
		TenantID:           domain.DefaultTenantID,
		StudentID:          "student-1",
		SubjectID:          "subject-1",
		ExerciseID:         &practice,
		TemplateID:         &templateID,
		Status:             status,
		EnvironmentProfile: profile,
	}
	if err := wsRepo.Create(context.Background(), environment); err != nil {
		t.Fatalf("no se pudo preparar el entorno: %v", err)
	}
	return svc, wsRepo, orch, tplRepo, environment
}

func TestExportPracticeEnvironment_BuildsReproducibleBundle(t *testing.T) {
	svc, _, orch, tplRepo, environment := exportFixture(t, domain.EnvironmentProfilePractice, domain.PracticeEnvironmentStatusRunning)

	bundle, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-1")
	if err != nil {
		t.Fatalf("error inesperado al exportar: %v", err)
	}
	if orch.calls != 1 {
		t.Fatalf("esperaba una lectura del volumen, obtuve %d", orch.calls)
	}
	if orch.volume != environment.VolumeName() {
		t.Errorf("volumen inesperado: %s", orch.volume)
	}
	if bundle.Filename != "solv-proyecto-de-practica-eeeeeeee.zip" {
		t.Errorf("nombre inesperado: %s", bundle.Filename)
	}
	if len(tplRepo.slugs) == 0 || tplRepo.slugs[0] != "python-312" {
		t.Errorf("esperaba resolver el perfil con el runtime del stack, obtuve %v", tplRepo.slugs)
	}

	files := readExportZip(t, bundle.Content)
	if _, ok := files["main.py"]; !ok {
		t.Fatalf("falta el código del alumno: %v", files)
	}
	if _, ok := files["../fuera.txt"]; ok {
		t.Fatal("el paquete incluye una ruta fuera del proyecto")
	}
	for _, name := range []string{"Dockerfile", "compose.yml", ".env.example", "README.md"} {
		if _, ok := files[name]; !ok {
			t.Errorf("falta %s en el paquete", name)
		}
	}
	if strings.Contains(files["Dockerfile"], "openvscode-server") {
		t.Errorf("el Dockerfile local no debe traer el editor del aula:\n%s", files["Dockerfile"])
	}
	if !strings.Contains(files["Dockerfile"], "FROM python:3.12-slim-bookworm") {
		t.Errorf("el Dockerfile debe conservar la imagen base:\n%s", files["Dockerfile"])
	}
	if !strings.Contains(files["README.md"], ".venv/bin/pip install -r requirements.txt") {
		t.Errorf("el README debe traer el comando de instalación del perfil de dependencias:\n%s", files["README.md"])
	}
	if !strings.Contains(files["README.md"], "DATABASE_URL") {
		t.Errorf("el README debe explicar la variable que SOLV configuraba:\n%s", files["README.md"])
	}
	if !strings.Contains(files["compose.yml"], "postgres:16-alpine") {
		t.Errorf("compose debe levantar el motor declarado:\n%s", files["compose.yml"])
	}
	if bundle.WorkspaceFiles != 2 {
		t.Errorf("esperaba 2 archivos válidos del volumen, obtuve %d", bundle.WorkspaceFiles)
	}
}

func TestExportPracticeEnvironment_WorksWhileHibernated(t *testing.T) {
	svc, _, orch, _, environment := exportFixture(t, domain.EnvironmentProfilePractice, domain.PracticeEnvironmentStatusHibernated)
	environment.ContainerID = nil

	if _, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-1"); err != nil {
		t.Fatalf("un entorno hibernado también se exporta: %v", err)
	}
	if orch.calls != 1 {
		t.Fatalf("esperaba una lectura del volumen, obtuve %d", orch.calls)
	}
}

func TestExportPracticeEnvironment_RejectsAnotherStudent(t *testing.T) {
	svc, _, _, _, environment := exportFixture(t, domain.EnvironmentProfilePractice, domain.PracticeEnvironmentStatusRunning)

	_, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-2")
	if !errors.Is(err, services.ErrPracticeEnvironmentForbidden) {
		t.Fatalf("esperaba forbidden, obtuve %v", err)
	}
}

func TestExportPracticeEnvironment_BlocksActiveExamAndAllowsClosedOne(t *testing.T) {
	svc, _, _, _, environment := exportFixture(t, domain.EnvironmentProfileExam, domain.PracticeEnvironmentStatusRunning)

	if _, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-1"); !errors.Is(err, services.ErrPracticeEnvironmentExportExamActive) {
		t.Fatalf("esperaba el bloqueo de examen activo, obtuve %v", err)
	}

	frozenAt := time.Now().Add(-time.Hour)
	environment.FrozenAt = &frozenAt
	environment.Status = domain.PracticeEnvironmentStatusFrozen
	if _, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-1"); err != nil {
		t.Fatalf("un examen cerrado sí se exporta: %v", err)
	}
}

func TestExportPracticeEnvironment_EmptyVolumeIsReported(t *testing.T) {
	svc, _, orch, _, environment := exportFixture(t, domain.EnvironmentProfilePractice, domain.PracticeEnvironmentStatusRunning)
	orch.files = nil

	_, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-1")
	if !errors.Is(err, domain.ErrExportBundleEmpty) {
		t.Fatalf("esperaba volumen vacío, obtuve %v", err)
	}
}

func TestExportPracticeEnvironment_ReportsVolumeFailure(t *testing.T) {
	svc, _, orch, _, environment := exportFixture(t, domain.EnvironmentProfilePractice, domain.PracticeEnvironmentStatusRunning)
	orch.err = errors.New("daemon no responde")

	_, err := svc.ExportPracticeEnvironment(context.Background(), environment.ID, "student-1")
	if err == nil || !strings.Contains(err.Error(), "no se pudo leer el proyecto") {
		t.Fatalf("esperaba un error de lectura del volumen, obtuve %v", err)
	}
}

func TestExportPracticeEnvironment_MissingEnvironmentIsNotFound(t *testing.T) {
	svc, _, _, _, _ := exportFixture(t, domain.EnvironmentProfilePractice, domain.PracticeEnvironmentStatusRunning)

	_, err := svc.ExportPracticeEnvironment(context.Background(), "99999999-9999-9999-9999-999999999999", "student-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperaba not found, obtuve %v", err)
	}
}
