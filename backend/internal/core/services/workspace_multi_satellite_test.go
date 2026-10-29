package services_test

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func multiServiceFixture(servicesCfg []domain.ServiceRequirement) (*mockWorkspaceRepo, *mockOrchestrator, *services.WorkspaceService) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	subjectID := "subj001"
	templateID := "tpl-multi"

	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		subjectID: {ID: subjectID, Name: "Bases de Datos"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		templateID: {
			ID:          templateID,
			Name:        "Laboratorio multi servicio",
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
			ServicesConfig: domain.ServicesConfig{
				Services: servicesCfg,
			},
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-multi": {ID: "ex-multi", TemplateID: templateID, SubjectID: &subjectID},
	}}

	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})
	return wsRepo, orch, svc
}

func TestStartWorkspaceForExercise_ProvisionsAllBindings(t *testing.T) {
	_, orch, svc := multiServiceFixture([]domain.ServiceRequirement{
		{Category: "cache", Engine: "redis"},
		{Category: "database", Engine: "mongodb"},
	})

	if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-multi"); err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
	}

	if len(orch.satelliteSpecs) != 2 {
		t.Fatalf("esperados dos satélites, obtenidos %d: %+v", len(orch.satelliteSpecs), orch.satelliteSpecs)
	}
	if orch.satelliteSpecs[0].Engine != "mongodb" || orch.satelliteSpecs[1].Engine != "redis" {
		t.Fatalf("orden de aprovisionamiento inesperado: %q, %q", orch.satelliteSpecs[0].Engine, orch.satelliteSpecs[1].Engine)
	}

	config := orch.lastConfig
	if config == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if !config.AttachSatellitesNet {
		t.Error("el laboratorio debe conectarse a la red interna de satélites")
	}
	if uri, ok := envValue(config.Env, "MONGODB_URI"); !ok || uri != "mongodb://solv-sat-mongodb:27017/solv_std_std0001_subj001" {
		t.Errorf("MONGODB_URI inyectada inesperada: %q (%v)", uri, ok)
	}
	if url, ok := envValue(config.Env, "REDIS_URL"); !ok || url != "redis://solv-sat-redis:6379/0" {
		t.Errorf("REDIS_URL inyectada inesperada: %q (%v)", url, ok)
	}
	if prefix, _ := envValue(config.Env, "REDIS_KEY_PREFIX"); prefix != "solv_std_std0001_subj001:" {
		t.Errorf("REDIS_KEY_PREFIX inyectado inesperado: %q", prefix)
	}
	if _, ok := envValue(config.Env, "DATABASE_URL"); ok {
		t.Error("un stack NoSQL no debe inyectar DATABASE_URL relacional")
	}
}

func TestHibernateWorkspace_KeepsBothSatellitesWhileAnotherLabUsesThem(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	templateID := "tpl-multi"
	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		"subj001": {ID: "subj001"},
		"subj002": {ID: "subj002"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		templateID: {
			ID:          templateID,
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
			ServicesConfig: domain.ServicesConfig{
				Services: []domain.ServiceRequirement{
					{Category: "database", Engine: "mongodb"},
					{Category: "cache", Engine: "redis"},
				},
			},
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-1": {ID: "ex-1", TemplateID: templateID, SubjectID: stringPtr("subj001")},
		"ex-2": {ID: "ex-2", TemplateID: templateID, SubjectID: stringPtr("subj002")},
	}}

	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})

	ctx := context.Background()
	first, err := svc.StartWorkspaceForExercise(ctx, "std0001", "subj001", "ex-1")
	if err != nil {
		t.Fatalf("error inesperado al iniciar el primer laboratorio: %v", err)
	}
	second, err := svc.StartWorkspaceForExercise(ctx, "std0002", "subj002", "ex-2")
	if err != nil {
		t.Fatalf("error inesperado al iniciar el segundo laboratorio: %v", err)
	}

	if err := svc.HibernateWorkspace(ctx, first.ID); err != nil {
		t.Fatalf("error inesperado al hibernar: %v", err)
	}
	if containsEngine(orch.satelliteStops, "mongodb") || containsEngine(orch.satelliteStops, "redis") {
		t.Fatalf("ningún motor debe detenerse mientras otro laboratorio los usa: %v", orch.satelliteStops)
	}

	if err := svc.HibernateWorkspace(ctx, second.ID); err != nil {
		t.Fatalf("error inesperado al hibernar el segundo laboratorio: %v", err)
	}
	if !containsEngine(orch.satelliteStops, "mongodb") || !containsEngine(orch.satelliteStops, "redis") {
		t.Fatalf("ambos motores deben detenerse al quedar sin uso: %v", orch.satelliteStops)
	}
}

func TestHibernateWorkspace_StopsOnlyTheEngineWithoutOtherUsers(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	multiTemplateID := "tpl-multi"
	mongoTemplateID := "tpl-mongo"
	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		"subj001": {ID: "subj001"},
		"subj002": {ID: "subj002"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		multiTemplateID: {
			ID:          multiTemplateID,
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
			ServicesConfig: domain.ServicesConfig{
				Services: []domain.ServiceRequirement{
					{Category: "database", Engine: "mongodb"},
					{Category: "cache", Engine: "redis"},
				},
			},
		},
		mongoTemplateID: {
			ID:          mongoTemplateID,
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
			ServicesConfig: domain.ServicesConfig{
				Services: []domain.ServiceRequirement{{Category: "database", Engine: "mongodb"}},
			},
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-multi": {ID: "ex-multi", TemplateID: multiTemplateID, SubjectID: stringPtr("subj001")},
		"ex-mongo": {ID: "ex-mongo", TemplateID: mongoTemplateID, SubjectID: stringPtr("subj002")},
	}}

	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})

	ctx := context.Background()
	multi, err := svc.StartWorkspaceForExercise(ctx, "std0001", "subj001", "ex-multi")
	if err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio multi servicio: %v", err)
	}
	if _, err := svc.StartWorkspaceForExercise(ctx, "std0002", "subj002", "ex-mongo"); err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio de mongodb: %v", err)
	}

	if err := svc.HibernateWorkspace(ctx, multi.ID); err != nil {
		t.Fatalf("error inesperado al hibernar: %v", err)
	}
	if !containsEngine(orch.satelliteStops, "redis") {
		t.Fatalf("redis debe detenerse al quedar sin laboratorios: %v", orch.satelliteStops)
	}
	if containsEngine(orch.satelliteStops, "mongodb") {
		t.Fatalf("mongodb no debe detenerse mientras otro laboratorio lo usa: %v", orch.satelliteStops)
	}
}

// El db_init_script de un ejercicio con MongoDB se aplica dentro de la base aislada y
// sellado en la colección de migraciones, con el mismo contrato idempotente que SQL.
func TestStartWorkspaceForExercise_AppliesMongoDBInitScriptWithSeal(t *testing.T) {
	const script = `db.createCollection("materias");`
	_, orch, svc := managedSatelliteFixture("mongodb", script)

	if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-sat"); err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
	}

	execs := orch.satelliteExecsWith("mongosh")
	if len(execs) != 1 {
		t.Fatalf("esperada una invocación de init en mongodb, obtenidas %d", len(execs))
	}
	joined := strings.Join(execs[0].cmd, " ")
	for _, want := range []string{
		"mongodb://127.0.0.1:27017/solv_std_std0001_subj001",
		`db.getCollection("_solv_exercise_migrations").findOne({ exercise_id: "ex-sat" })`,
		script,
		`insertOne({ exercise_id: "ex-sat", applied_at: new Date() })`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("el init de mongodb debe incluir %q: %s", want, joined)
		}
	}
}

// Un motor con host externo declarado por el sysadmin no se levanta ni se aprovisiona
// dentro de un contenedor local: SOLV solo inyecta credenciales hacia ese destino.
func TestStartWorkspaceForExercise_ExternalSatelliteHostSkipsLocalProvisioning(t *testing.T) {
	t.Setenv("MYSQL_SATELLITE_HOST", "db.satellites.internal")
	_, orch, svc := managedSatelliteFixture("mysql", "")

	if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-sat"); err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
	}

	if len(orch.satelliteSpecs) != 0 {
		t.Fatalf("con host externo no debe levantarse un contenedor local: %+v", orch.satelliteSpecs)
	}
	if orch.satelliteExecutedSQL("CREATE DATABASE IF NOT EXISTS") {
		t.Error("con host externo no debe intentarse aprovisionar dentro de un contenedor local")
	}
	config := orch.lastConfig
	if config == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if dsn, _ := envValue(config.Env, "DATABASE_URL"); !strings.Contains(dsn, "@db.satellites.internal:3306/solv_std_std0001_subj001") {
		t.Errorf("DATABASE_URL debe apuntar al host externo declarado: %q", dsn)
	}
}

func stringPtr(value string) *string {
	return &value
}
