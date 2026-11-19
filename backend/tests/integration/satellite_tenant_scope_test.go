package integration

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// recordingSatellite responde a las llamadas que hace el apagado de motores
// ociosos y anota qué motor se detuvo. El resto de la interfaz del orquestador
// no participa de este camino.
type recordingSatellite struct {
	domain.PracticeEnvironmentOrchestrator
	stopped []string
}

func (r *recordingSatellite) EnsureManagedSatelliteRunning(context.Context, domain.ManagedSatelliteSpec) (*domain.ManagedSatelliteEndpoint, error) {
	return nil, nil
}

func (r *recordingSatellite) StopManagedSatellite(_ context.Context, engine string) error {
	r.stopped = append(r.stopped, engine)
	return nil
}

func (r *recordingSatellite) ExecInManagedSatellite(context.Context, string, []string) (string, error) {
	return "", nil
}

func (r *recordingSatellite) ExecInManagedSatelliteWithInput(context.Context, string, []string, io.Reader) (string, error) {
	return "", nil
}

func (r *recordingSatellite) wasStopped(engine string) bool {
	for _, stopped := range r.stopped {
		if stopped == engine {
			return true
		}
	}
	return false
}

// seedDefaultTenantLab devuelve una plantilla con MySQL y un laboratorio en
// ejecución dentro del tenant del nodo. El apagado de motores corre en
// background, sin tenant en el contexto, así que el laboratorio tiene que vivir
// donde ese camino lo encuentra.
func seedDefaultTenantLab(t *testing.T, repo domain.PracticeEnvironmentRepository, templateID, studentID, subjectID, exerciseID string) {
	t.Helper()
	ctx := context.Background()
	if err := repo.Create(ctx, &domain.PracticeEnvironment{
		ID:         uuid.NewString(),
		TenantID:   domain.DefaultTenantID,
		StudentID:  studentID,
		SubjectID:  subjectID,
		ExerciseID: &exerciseID,
		TemplateID: &templateID,
		Type:       domain.PracticeEnvironmentTypeIDEPersistente,
		Status:     domain.PracticeEnvironmentStatusRunning,
	}); err != nil {
		t.Fatalf("crear laboratorio en ejecución: %v", err)
	}
}

func TestIdleSatelliteKeepsEngineOfRunningLabWithoutTenantInContext(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	sqlDB := db.GetDB()

	studentID := uuid.NewString()
	teacherID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	templateID := uuid.NewString()

	// El tenant del nodo puede venir sembrado por las migraciones: se reutiliza.
	if _, err := sqlDB.Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'Nodo', 'nodo')
		ON CONFLICT (id) DO NOTHING
	`, domain.DefaultTenantID); err != nil {
		t.Fatalf("seed tenant del nodo: %v", err)
	}
	for _, user := range []struct{ id, email, role string }{
		{teacherID, "scope-teacher-" + teacherID[:8] + "@example.test", "teacher"},
		{studentID, "scope-student-" + studentID[:8] + "@example.test", "student"},
	} {
		if _, err := sqlDB.Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Scope', 'User', $2, $3, $4)
		`, user.id, user.email, user.role, domain.DefaultTenantID); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Scope Subject', $3, $4)
	`, subjectID, domain.DefaultTenantID, "SCOPE-"+subjectID[:8], teacherID); err != nil {
		t.Fatalf("seed subject: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, status, services_config)
		VALUES ($1, $2, 'Scope MySQL', 'solv/ide:test', 512, 'IDE_PERSISTENTE', 'APROBADA',
			'{"services": [{"role": "primary_database", "engine": "mysql", "category": "database", "env_alias": "db"}]}'::jsonb)
	`, templateID, domain.DefaultTenantID); err != nil {
		t.Fatalf("seed plantilla con MySQL: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO exercises (id, tenant_id, subject_id, title, type, status, environment_type, language, boilerplate, template_id)
		VALUES ($1, $2, $3, 'Scope Exercise', 'project', 'draft', 'IDE_PERSISTENTE', 'python', '', $4)
	`, exerciseID, domain.DefaultTenantID, subjectID, templateID); err != nil {
		t.Fatalf("seed ejercicio: %v", err)
	}

	templateRepo := postgres.NewPostgresTemplateRepository(sqlDB)

	// El apagado de motores corre en background: sin tenant en el contexto.
	// Antes del arreglo esta lectura fallaba y la plantilla quedaba ilegible.
	template, err := templateRepo.GetTemplateByID(context.Background(), templateID)
	if err != nil {
		t.Fatalf("la plantilla debía leerse sin tenant en el contexto: %v", err)
	}
	if template == nil || template.ID != templateID {
		t.Fatalf("se esperaba la plantilla %s, llegó %+v", templateID, template)
	}
	if len(template.ServicesConfig.Services) != 1 || domain.CanonicalServiceEngine(template.ServicesConfig.Services[0].Engine) != "mysql" {
		t.Fatalf("la plantilla debía declarar MySQL: %+v", template.ServicesConfig.Services)
	}

	envRepo := postgres.NewPostgresPracticeEnvironmentRepository(sqlDB)
	seedDefaultTenantLab(t, envRepo, templateID, studentID, subjectID, exerciseID)

	satellite := &recordingSatellite{}
	service := services.NewPracticeEnvironmentService(envRepo, satellite, &mockSlice12HostMonitor{}).
		WithProvisioning(postgres.NewPostgresSubjectRepository(sqlDB), templateRepo)

	service.ReconcileIdleManagedSatellitesScheduled(context.Background())

	if satellite.wasStopped("mysql") {
		t.Fatalf("el motor se detuvo con un laboratorio abierto: %v", satellite.stopped)
	}
	if !satellite.wasStopped("redis") {
		t.Fatalf("el motor sin laboratorios debía detenerse; detenidos: %v", satellite.stopped)
	}
}
