package services_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// testPracticeID es la práctica de los entornos de estas pruebas: el modelo por
// materia se retiró y todo entorno pertenece a un ejercicio.
const testPracticeID = "practice-1"

type mockPracticeEnvironmentRepo struct {
	environments map[string]*domain.PracticeEnvironment
	active       []*domain.PracticeEnvironment
}

func newMockPracticeEnvironmentRepo() *mockPracticeEnvironmentRepo {
	return &mockPracticeEnvironmentRepo{
		environments: make(map[string]*domain.PracticeEnvironment),
	}
}

// GetByStudentSubjectExercise busca el entorno de una práctica concreta: es la
// única identidad de un entorno desde que se retiró el modelo por materia.
func (m *mockPracticeEnvironmentRepo) GetByStudentSubjectExercise(ctx context.Context, studentID, subjectID, exerciseID string) (*domain.PracticeEnvironment, error) {
	ws, ok := m.environments[studentID+":"+subjectID+":"+exerciseID]
	if !ok {
		return nil, errors.New("environment not found")
	}
	return ws, nil
}

func (m *mockPracticeEnvironmentRepo) GetByID(ctx context.Context, id string) (*domain.PracticeEnvironment, error) {
	return m.environments[id], nil
}

func (m *mockPracticeEnvironmentRepo) Create(ctx context.Context, ws *domain.PracticeEnvironment) error {
	if ws.ExerciseID != nil {
		m.environments[ws.StudentID+":"+ws.SubjectID+":"+*ws.ExerciseID] = ws
	}
	m.environments[ws.ID] = ws
	return nil
}

func (m *mockPracticeEnvironmentRepo) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	if ws, ok := m.environments[id]; ok {
		ws.ContainerID = &containerID
	}
	return nil
}

func (m *mockPracticeEnvironmentRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if ws, ok := m.environments[id]; ok {
		ws.Status = status
	}
	return nil
}

func (m *mockPracticeEnvironmentRepo) UpdateMemoryLimit(ctx context.Context, id string, newLimitMB int64) error {
	if ws, ok := m.environments[id]; ok {
		ws.MemoryLimitMB = newLimitMB
	}
	return nil
}

func (m *mockPracticeEnvironmentRepo) RecordHeartbeat(ctx context.Context, id string) error {
	return nil
}

func (m *mockPracticeEnvironmentRepo) IncrementOOMStrike(ctx context.Context, id string) error {
	return nil
}

func (m *mockPracticeEnvironmentRepo) ResetOOMStrikes(ctx context.Context, id string) error {
	return nil
}

func (m *mockPracticeEnvironmentRepo) GetActivePracticeEnvironments(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
	return m.active, nil
}

func (m *mockPracticeEnvironmentRepo) GetAllRunningPracticeEnvironments(ctx context.Context) ([]*domain.PracticeEnvironment, error) {
	seen := make(map[*domain.PracticeEnvironment]bool)
	var running []*domain.PracticeEnvironment
	for _, ws := range m.environments {
		if ws == nil || seen[ws] || ws.Status != domain.PracticeEnvironmentStatusRunning {
			continue
		}
		seen[ws] = true
		running = append(running, ws)
	}
	return running, nil
}

func (m *mockPracticeEnvironmentRepo) GetByType(ctx context.Context, wsType string) ([]*domain.PracticeEnvironment, error) {
	return nil, nil
}

func (m *mockPracticeEnvironmentRepo) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	return nil
}

type satelliteExec struct {
	engine string
	cmd    []string
	// input es lo que el comando recibió por entrada estándar. Los scripts de
	// inicialización viajan por ahí desde ADR-072.
	input string
}

type mockOrchestrator struct {
	lastConfig  *domain.PracticeEnvironmentContainerConfig
	lastExecCmd []string

	// Satélites gestionados: el mock registra specs levantadas, motores detenidos y
	// comandos ejecutados, y simula el registro de migraciones del ejercicio para
	// verificar que el db_init_script se aplica una sola vez.
	satelliteSpecs       []domain.ManagedSatelliteSpec
	satelliteStops       []string
	satelliteExecs       []satelliteExec
	satelliteScriptRuns  int
	satelliteEnsureError error
}

func (m *mockOrchestrator) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	return nil
}

func (m *mockOrchestrator) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	return nil
}

func (m *mockOrchestrator) StartPracticeEnvironmentContainer(ctx context.Context, config domain.PracticeEnvironmentContainerConfig) (string, error) {
	cfgCopy := config
	m.lastConfig = &cfgCopy
	return "container-12345", nil
}

func (m *mockOrchestrator) UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error {
	return nil
}

func (m *mockOrchestrator) GetContainerMetrics(ctx context.Context, containerID string) (*domain.ContainerMetrics, error) {
	return &domain.ContainerMetrics{IsRunning: true}, nil
}

func (m *mockOrchestrator) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockOrchestrator) PauseContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockOrchestrator) UnpauseContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockOrchestrator) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockOrchestrator) RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error) {
	return nil, nil
}

func (m *mockOrchestrator) GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error) {
	return "", nil
}

func (m *mockOrchestrator) ExecuteCommandInBackground(ctx context.Context, containerID string, workDir string, cmd []string) error {
	m.lastExecCmd = cmd
	return nil
}

func (m *mockOrchestrator) InspectLocalImage(ctx context.Context, imageRef string) (bool, int64, error) {
	return true, 1000000, nil
}

func (m *mockOrchestrator) PullImage(ctx context.Context, imageRef string) error {
	return nil
}

func (m *mockOrchestrator) EnsureManagedSatelliteRunning(ctx context.Context, spec domain.ManagedSatelliteSpec) (*domain.ManagedSatelliteEndpoint, error) {
	if m.satelliteEnsureError != nil {
		return nil, m.satelliteEnsureError
	}
	m.satelliteSpecs = append(m.satelliteSpecs, spec)
	return &domain.ManagedSatelliteEndpoint{
		Engine:      spec.Engine,
		Host:        spec.ContainerName,
		Port:        spec.Port,
		ContainerID: "sat-" + spec.Engine,
	}, nil
}

func (m *mockOrchestrator) StopManagedSatellite(ctx context.Context, engine string) error {
	m.satelliteStops = append(m.satelliteStops, engine)
	return nil
}

func (m *mockOrchestrator) ExecInManagedSatellite(ctx context.Context, engine string, cmd []string) (string, error) {
	return m.recordSatelliteExec(engine, cmd, "")
}

func (m *mockOrchestrator) ExecInManagedSatelliteWithInput(ctx context.Context, engine string, cmd []string, input io.Reader) (string, error) {
	body := ""
	if input != nil {
		raw, err := io.ReadAll(input)
		if err != nil {
			return "", err
		}
		body = string(raw)
	}
	return m.recordSatelliteExec(engine, cmd, body)
}

func (m *mockOrchestrator) recordSatelliteExec(engine string, cmd []string, input string) (string, error) {
	m.satelliteExecs = append(m.satelliteExecs, satelliteExec{engine: engine, cmd: cmd, input: input})
	if engine == "mongodb" {
		// El init de MongoDB se evalúa dentro de mongosh; el mock solo registra la
		// invocación para que el test verifique el guard y el sellado.
		return "", nil
	}
	// El script puede llegar como argumento (`-e`, `-c`) o por entrada estándar. La
	// clasificación mira la entrada primero porque es el camino vigente.
	sql := input
	if sql == "" && len(cmd) > 0 {
		sql = cmd[len(cmd)-1]
	}
	switch {
	case strings.Contains(sql, "SELECT COUNT(*) FROM _solv_exercise_migrations"):
		if m.satelliteScriptRuns > 0 {
			return "1\n", nil
		}
		return "0\n", nil
	case strings.Contains(sql, "SELECT 1 FROM _solv_exercise_migrations"):
		if m.satelliteScriptRuns > 0 {
			return "1\n", nil
		}
		return "", nil
	case strings.Contains(sql, "_solv_exercise_migrations") ||
		strings.Contains(sql, "CREATE DATABASE") ||
		strings.Contains(sql, "pg_roles") ||
		strings.Contains(sql, "pg_database") ||
		strings.Contains(sql, "REVOKE CONNECT"):
		return "", nil
	default:
		m.satelliteScriptRuns++
		return "", nil
	}
}

// satelliteExecsWith devuelve los comandos ejecutados con el binario indicado.
func (m *mockOrchestrator) satelliteExecsWith(binary string) []satelliteExec {
	var found []satelliteExec
	for _, exec := range m.satelliteExecs {
		if len(exec.cmd) > 0 && exec.cmd[0] == binary {
			found = append(found, exec)
		}
	}
	return found
}

// satelliteExecutedSQL indica si algún comando ejecutado contiene el fragmento dado.
func (m *mockOrchestrator) satelliteExecutedSQL(fragment string) bool {
	for _, exec := range m.satelliteExecs {
		if strings.Contains(exec.input, fragment) {
			return true
		}
		for _, arg := range exec.cmd {
			if strings.Contains(arg, fragment) {
				return true
			}
		}
	}
	return false
}

type mockHostMonitor struct{}

func (m *mockHostMonitor) CanAllocateMemory(mb int64) bool {
	return true
}

func (m *mockHostMonitor) GetHostMemoryStats() (float64, uint64, error) {
	return 50.0, 8192, nil
}

type mockSubjectRepo struct {
	subjects map[string]*domain.Subject
}

func (m *mockSubjectRepo) Create(ctx context.Context, subject *domain.Subject) error {
	return nil
}

func (m *mockSubjectRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Subject, error) {
	return m.subjects[id], nil
}

func (m *mockSubjectRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	return nil, nil
}

func (m *mockSubjectRepo) EnrollStudent(ctx context.Context, enrollment *domain.Enrollment) error {
	return nil
}

func (m *mockSubjectRepo) ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	return nil, nil
}

func (m *mockSubjectRepo) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Subject, error) {
	return nil, nil
}

func (m *mockSubjectRepo) ReassignTeacher(ctx context.Context, tenantID, subjectID, newTeacherID string) error {
	return nil
}

func (m *mockSubjectRepo) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	return nil
}

func (m *mockSubjectRepo) Update(ctx context.Context, tenantID, subjectID, name, code string) error {
	return nil
}

func (m *mockSubjectRepo) GetTemplateStatus(ctx context.Context, templateID string) (string, error) {
	return "approved", nil
}

type mockTemplateRepo struct {
	templates map[string]*domain.Template
}

func (m *mockTemplateRepo) GetTemplateByID(ctx context.Context, id string) (*domain.Template, error) {
	return m.templates[id], nil
}

func TestPracticeEnvironmentService_TemplateProvisioning(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	monitor := &mockHostMonitor{}

	tplID := "tpl-python-ds"
	subjRepo := &mockSubjectRepo{
		subjects: map[string]*domain.Subject{
			"subj-101": {
				ID:         "subj-101",
				Name:       "Ciencia de Datos",
				TemplateID: &tplID,
			},
		},
	}

	tplRepo := &mockTemplateRepo{
		templates: map[string]*domain.Template{
			tplID: {
				ID:          tplID,
				Name:        "Python Data Science",
				DockerImage: "solv/python:3.12-ds",
				BaseRamMB:   1024,
				SetupScript: "pip install pandas numpy",
				ServicesConfig: domain.ServicesConfig{
					Services: []domain.ServiceRequirement{
						{Category: "database", Engine: "postgres"},
					},
				},
			},
		},
	}

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, monitor).
		WithProvisioning(subjRepo, tplRepo)

	ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1", "subj-101", testPracticeID)
	if err != nil {
		t.Fatalf("error inesperado al iniciar environment con plantilla: %v", err)
	}

	// 1. Debe haber adoptado la memoria de la plantilla (1024 MB)
	if ws.MemoryLimitMB != 1024 {
		t.Errorf("esperado MemoryLimitMB = 1024, obtenido: %d", ws.MemoryLimitMB)
	}

	// 2. Debe haber registrado el TemplateID en la instancia
	if ws.TemplateID == nil || *ws.TemplateID != tplID {
		t.Errorf("esperado TemplateID = %s, obtenido: %v", tplID, ws.TemplateID)
	}

	// 3. El contenedor debe haber arrancado con la imagen especificada
	if orch.lastConfig == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if orch.lastConfig.Image != "solv/python:3.12-ds" {
		t.Errorf("esperado imagen solv/python:3.12-ds, obtenido: %s", orch.lastConfig.Image)
	}
	if orch.lastConfig.MemoryLimitMB != 1024 {
		t.Errorf("esperado container MemoryLimitMB = 1024, obtenido: %d", orch.lastConfig.MemoryLimitMB)
	}

	// 4. El script de la plantilla debe propagarse al contenedor para que el
	// shell de arranque lo ejecute una sola vez (sello .solv_initialized).
	if orch.lastConfig.SetupScript != "pip install pandas numpy" {
		t.Errorf("esperado SetupScript = %q, obtenido: %q", "pip install pandas numpy", orch.lastConfig.SetupScript)
	}
}

func TestPracticeEnvironmentService_DefaultFallbackWithoutTemplate(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	monitor := &mockHostMonitor{}

	// Sin materia registrada o sin template asignado
	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{}}

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, monitor).
		WithProvisioning(subjRepo, tplRepo)

	ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-2", "subj-general", testPracticeID)
	if err != nil {
		t.Fatalf("error inesperado en fallback: %v", err)
	}

	// Sin políticas inyectadas rigen los defaults del dominio (512/15/40,
	// iguales a los de un tenant fresco y a los defaults de la vista).
	if ws.MemoryLimitMB != int64(domain.DefaultServerPolicies().RAMLimitMB) {
		t.Errorf("esperado default de políticas (%d), obtenido: %d", domain.DefaultServerPolicies().RAMLimitMB, ws.MemoryLimitMB)
	}
	if orch.lastConfig.Image != domain.OpenVSCodeImage {
		t.Errorf("esperado imagen default %s, obtenido: %s", domain.OpenVSCodeImage, orch.lastConfig.Image)
	}
}

func TestPracticeEnvironmentService_PoliciesAdmission(t *testing.T) {
	policiesJSON := []byte(`{"server_policies":{"ram_limit_mb":256,"inactivity_minutes":15,"max_containers":2,"updated_at":"2026-01-01T00:00:00Z"}}`)

	newSvcWithPolicies := func(wsRepo *mockPracticeEnvironmentRepo, subjRepo *mockSubjectRepo, tplRepo *mockTemplateRepo) *services.PracticeEnvironmentService {
		polSvc := services.NewServerPoliciesService(&mockTenantRepo{config: policiesJSON})
		return services.NewPracticeEnvironmentService(wsRepo, &mockOrchestrator{}, &mockHostMonitor{}).
			WithProvisioning(subjRepo, tplRepo).
			SetPoliciesService(polSvc)
	}

	t.Run("la RAM de la política es el default sin plantilla", func(t *testing.T) {
		wsRepo := newMockPracticeEnvironmentRepo()
		subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{}}
		svc := newSvcWithPolicies(wsRepo, subjRepo, &mockTemplateRepo{templates: map[string]*domain.Template{}})

		ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-pol", "subj-sin-plantilla", testPracticeID)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if ws.MemoryLimitMB != 256 {
			t.Errorf("esperado RAM de política = 256, obtenido: %d", ws.MemoryLimitMB)
		}
	})

	t.Run("la plantilla prevalece sobre la política y el inicio no se bloquea", func(t *testing.T) {
		wsRepo := newMockPracticeEnvironmentRepo()
		tplID := "tpl-1gb"
		subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
			"subj-1gb": {ID: "subj-1gb", TemplateID: &tplID},
		}}
		tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
			tplID: {ID: tplID, DockerImage: "solv/cs:1gb", BaseRamMB: 1024},
		}}
		svc := newSvcWithPolicies(wsRepo, subjRepo, tplRepo)

		ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1gb", "subj-1gb", testPracticeID)
		if err != nil {
			t.Fatalf("la plantilla debe prevalecer, pero falló: %v", err)
		}
		if ws.MemoryLimitMB != 1024 {
			t.Errorf("esperado RAM de plantilla = 1024, obtenido: %d", ws.MemoryLimitMB)
		}
	})

	t.Run("el techo de concurrencia rechaza inicios nuevos", func(t *testing.T) {
		wsRepo := newMockPracticeEnvironmentRepo()
		wsRepo.active = []*domain.PracticeEnvironment{{ID: "ws-1"}, {ID: "ws-2"}}
		subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{}}
		svc := newSvcWithPolicies(wsRepo, subjRepo, &mockTemplateRepo{templates: map[string]*domain.Template{}})

		_, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-full", "subj-x", testPracticeID)
		if err == nil {
			t.Fatal("esperado ErrMaxContainersReached con 2 activos y techo 2")
		}
		if !errors.Is(err, services.ErrMaxContainersReached) {
			t.Errorf("esperado ErrMaxContainersReached, obtenido: %v", err)
		}
	})
}

type mockTenantRepo struct {
	config []byte
}

func (m *mockTenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	return &domain.Tenant{ID: id, Config: m.config}, nil
}

func (m *mockTenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) GetAll(ctx context.Context) ([]*domain.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) UpdateConfig(ctx context.Context, id string, config []byte) error {
	return nil
}

func (m *mockTenantRepo) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	return nil
}

func (m *mockTenantRepo) GetMaintenance(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	return nil, nil
}

func TestPracticeEnvironmentService_StartPracticeEnvironment_RamExceedsHost(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	monitor := &mockHostMonitor{}

	targetSubjectID := "subj-too-much-ram"
	targetTemplateID := "tpl-too-much-ram"

	subjRepo := &mockSubjectRepo{
		subjects: map[string]*domain.Subject{
			targetSubjectID: {
				ID:         targetSubjectID,
				TemplateID: &targetTemplateID,
			},
		},
	}

	tplRepo := &mockTemplateRepo{
		templates: map[string]*domain.Template{
			targetTemplateID: {
				ID:          targetTemplateID,
				DockerImage: "solv/big-data:latest",
				BaseRamMB:   1000000, // 1 TB de RAM, excede cualquier host
			},
		},
	}

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, monitor).
		WithProvisioning(subjRepo, tplRepo)

	_, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-excess-ram", targetSubjectID, testPracticeID)
	if err == nil {
		t.Fatalf("esperado error por exceder capacidad estructural del host, pero la creación tuvo éxito")
	}
	if !strings.Contains(err.Error(), "excede la capacidad estructural del host") {
		t.Errorf("mensaje de error esperado con explicación clara, obtenido: %v", err)
	}
}

// TestStartPracticeEnvironmentForExercise_SeparaLosEntornosDeCadaPractica fija el cambio:
// cada práctica crea su propio entorno y su propio volumen, y abrir otra no
// reutiliza el de la anterior.
func TestStartPracticeEnvironmentForExercise_SeparaLosEntornosDeCadaPractica(t *testing.T) {
	ctx := context.Background()
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	subjectID := "subj-310"
	exerciseID := "ex-0022"
	secondExerciseID := "ex-0023"

	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		subjectID: {ID: subjectID, Name: "Bases de Datos"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		exerciseID:       {ID: exerciseID, Title: "Práctica 2: Índices y planes", SubjectID: stringPtr(subjectID)},
		secondExerciseID: {ID: secondExerciseID, Title: "Práctica 3: Transacciones", SubjectID: stringPtr(subjectID)},
	}}

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})

	practicePracticeEnvironment, err := svc.StartPracticeEnvironmentForExercise(ctx, "std-1", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("error inesperado al abrir la práctica: %v", err)
	}
	if practicePracticeEnvironment.ExerciseID == nil || *practicePracticeEnvironment.ExerciseID != exerciseID {
		t.Fatalf("esperado entorno acotado a la práctica %s, got %v", exerciseID, practicePracticeEnvironment.ExerciseID)
	}
	practiceVolume := orch.lastConfig.VolumeName
	if practiceVolume == "" {
		t.Fatal("el entorno de la práctica debe tener volumen propio")
	}

	// Volver a abrir la misma práctica devuelve el mismo entorno, no uno nuevo.
	again, err := svc.StartPracticeEnvironmentForExercise(ctx, "std-1", subjectID, exerciseID)
	if err != nil {
		t.Fatalf("error inesperado al reabrir la práctica: %v", err)
	}
	if again.ID != practicePracticeEnvironment.ID {
		t.Fatalf("esperado el mismo entorno de práctica (%s), got %s", practicePracticeEnvironment.ID, again.ID)
	}

	// Otra práctica de la misma materia tampoco comparte entorno ni volumen.
	secondPractice, err := svc.StartPracticeEnvironmentForExercise(ctx, "std-1", subjectID, secondExerciseID)
	if err != nil {
		t.Fatalf("error inesperado al abrir la segunda práctica: %v", err)
	}
	if secondPractice.ID == practicePracticeEnvironment.ID {
		t.Fatalf("la segunda práctica reutilizó el entorno de la primera (%s)", practicePracticeEnvironment.ID)
	}
	if secondPractice.ExerciseID == nil || *secondPractice.ExerciseID != secondExerciseID {
		t.Fatalf("esperado entorno acotado a la práctica %s, got %v", secondExerciseID, secondPractice.ExerciseID)
	}
	if orch.lastConfig.VolumeName == practicePracticeEnvironment.VolumeName() {
		t.Fatalf("la segunda práctica reutilizó el volumen de la primera (%s)", practicePracticeEnvironment.VolumeName())
	}
}

// recordingSessionLinker anota qué entornos se enlazan al aula abierta.
type recordingSessionLinker struct {
	attached []string
}

func (r *recordingSessionLinker) AttachPracticeEnvironmentToOpenSession(_ context.Context, _, _, _, environmentID string) error {
	r.attached = append(r.attached, environmentID)
	return nil
}

// TestStartPracticeEnvironmentForExercise_EntraAlAulaYElIdeDelDocenteNo fija el lado de
// presencia: abrir la práctica del alumno lo registra en el aula abierta, pero
// abrir el IDE de la práctica del docente no lo mete en el roster de alumnos.
func TestStartPracticeEnvironmentForExercise_EntraAlAulaYElIdeDelDocenteNo(t *testing.T) {
	ctx := context.Background()
	wsRepo := newMockPracticeEnvironmentRepo()
	subjectID := "subj-410"
	studentExerciseID := "ex-0101"
	teacherExerciseID := "ex-0102"
	teacherID := "tea-1"

	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		subjectID: {ID: subjectID, Name: "Redes", TeacherID: stringPtr(teacherID)},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		studentExerciseID: {ID: studentExerciseID, SubjectID: stringPtr(subjectID)},
		teacherExerciseID: {
			ID:              teacherExerciseID,
			SubjectID:       stringPtr(subjectID),
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
	}}
	linker := &recordingSessionLinker{}
	svc := services.NewPracticeEnvironmentService(wsRepo, &mockOrchestrator{}, &mockHostMonitor{}).
		WithProvisioning(subjRepo, &mockTemplateRepo{templates: map[string]*domain.Template{}}).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{}).
		WithClassSessions(linker)

	studentPracticeEnvironment, err := svc.StartPracticeEnvironmentForExercise(ctx, "std-9", subjectID, studentExerciseID)
	if err != nil {
		t.Fatalf("error inesperado al abrir la práctica del alumno: %v", err)
	}
	if len(linker.attached) != 1 || linker.attached[0] != studentPracticeEnvironment.ID {
		t.Fatalf("la práctica del alumno debió enlazarse al aula, enlaces: %v", linker.attached)
	}

	if _, err := svc.StartTeacherPracticePracticeEnvironment(ctx, teacherID, subjectID, teacherExerciseID, "teacher"); err != nil {
		t.Fatalf("error inesperado al abrir el IDE de la práctica del docente: %v", err)
	}
	if len(linker.attached) != 1 {
		t.Fatalf("el IDE del docente no debe entrar al roster del aula, enlaces: %v", linker.attached)
	}
}
