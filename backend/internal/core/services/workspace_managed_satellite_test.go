package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// envValue busca el valor de una variable inyectada al laboratorio.
func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix), true
		}
	}
	return "", false
}

// managedSatelliteFixture arma un laboratorio cuyo ejercicio declara un motor acoplado.
func managedSatelliteFixture(engine string, dbInitScript string) (*mockWorkspaceRepo, *mockOrchestrator, *services.WorkspaceService) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	templateID := "tpl-" + engine
	category := "database"
	if engine == "redis" {
		category = "cache"
	}

	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
		"subj001": {ID: "subj001", Name: "Bases de Datos"},
	}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		templateID: {
			ID:          templateID,
			Name:        "Laboratorio " + engine,
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
			ServicesConfig: domain.ServicesConfig{
				Services: []domain.ServiceRequirement{{Category: category, Engine: engine}},
			},
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-sat": {
			ID:         "ex-sat",
			TemplateID: templateID,
			Config:     domain.ExerciseConfig{IDE: &domain.IDEConfig{DBInitScript: dbInitScript}},
		},
	}}

	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})
	return wsRepo, orch, svc
}

func TestStartWorkspaceForExercise_ProvisionsManagedSatelliteEngines(t *testing.T) {
	sqlEngines := []struct {
		engine string
		image  string
	}{
		{engine: "mysql", image: "mysql:8.4"},
		{engine: "mariadb", image: "mariadb:11.4"},
	}

	for _, tc := range sqlEngines {
		t.Run(tc.engine, func(t *testing.T) {
			_, orch, svc := managedSatelliteFixture(tc.engine, "")

			if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-sat"); err != nil {
				t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
			}

			if len(orch.satelliteSpecs) != 1 {
				t.Fatalf("esperado un satélite levantado, obtenidos %d", len(orch.satelliteSpecs))
			}
			spec := orch.satelliteSpecs[0]
			if spec.Engine != tc.engine || spec.Image != tc.image {
				t.Errorf("satélite inesperado: engine=%q image=%q", spec.Engine, spec.Image)
			}
			if spec.ContainerName != "solv-sat-"+tc.engine || spec.VolumeName != "solv_sat_data_"+tc.engine {
				t.Errorf("contenedor/volumen inesperados: %q / %q", spec.ContainerName, spec.VolumeName)
			}
			if spec.MemoryLimitMB != 256 || spec.Port != 3306 {
				t.Errorf("tuning inesperado: %d MB puerto %d", spec.MemoryLimitMB, spec.Port)
			}
			if len(spec.ReadyCmd) == 0 {
				t.Error("el satélite debe declarar un comando de readiness")
			}
			if strings.Contains(spec.Image, ":latest") {
				t.Errorf("la imagen del satélite no puede usar :latest: %q", spec.Image)
			}

			// Aislamiento por alumno y materia: base y usuario exclusivos, idempotentes.
			if !orch.satelliteExecutedSQL("CREATE DATABASE IF NOT EXISTS `solv_std_std0001_subj001`") {
				t.Error("debe crearse la base aislada del alumno")
			}
			if !orch.satelliteExecutedSQL("CREATE USER IF NOT EXISTS 'u_") {
				t.Error("debe crearse el usuario exclusivo del alumno")
			}
			if !orch.satelliteExecutedSQL("GRANT ALL PRIVILEGES ON `solv_std_std0001_subj001`.*") {
				t.Error("el usuario debe recibir permisos solo sobre la base del alumno")
			}

			config := orch.lastConfig
			if config == nil {
				t.Fatal("no se envió configuración de contenedor al orquestador")
			}
			if !config.AttachSatellitesNet {
				t.Error("el laboratorio debe conectarse a la red interna de satélites")
			}
			if host, ok := envValue(config.Env, "MYSQL_HOST"); !ok || host != "solv-sat-"+tc.engine {
				t.Errorf("MYSQL_HOST inyectado inesperado: %q (%v)", host, ok)
			}
			if port, _ := envValue(config.Env, "MYSQL_PORT"); port != "3306" {
				t.Errorf("MYSQL_PORT inyectado inesperado: %q", port)
			}
			if db, _ := envValue(config.Env, "MYSQL_DATABASE"); db != "solv_std_std0001_subj001" {
				t.Errorf("MYSQL_DATABASE inyectado inesperado: %q", db)
			}
			user, ok := envValue(config.Env, "MYSQL_USER")
			if !ok || !strings.HasPrefix(user, "u_") {
				t.Errorf("MYSQL_USER debe ser un usuario aislado, obtenido %q", user)
			}
			password, _ := envValue(config.Env, "MYSQL_PASSWORD")
			dsn, _ := envValue(config.Env, "DATABASE_URL")
			if password == "" || !strings.HasPrefix(dsn, "mysql://"+user+":"+password+"@solv-sat-"+tc.engine+":3306/solv_std_std0001_subj001") {
				t.Errorf("DATABASE_URL inyectada inesperada: %q", dsn)
			}
		})
	}

	t.Run("mongodb", func(t *testing.T) {
		_, orch, svc := managedSatelliteFixture("mongodb", "")

		if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-sat"); err != nil {
			t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
		}

		if len(orch.satelliteSpecs) != 1 || orch.satelliteSpecs[0].Image != "mongo:7.0" {
			t.Fatalf("satélite mongodb inesperado: %+v", orch.satelliteSpecs)
		}
		if orch.satelliteSpecs[0].Port != 27017 || orch.satelliteSpecs[0].MemoryLimitMB != 256 {
			t.Errorf("tuning mongodb inesperado: %+v", orch.satelliteSpecs[0])
		}
		if len(orch.satelliteSpecs[0].Cmd) == 0 || !strings.Contains(strings.Join(orch.satelliteSpecs[0].Cmd, " "), "wiredTigerCacheSizeGB") {
			t.Errorf("mongodb debe arrancar con caché interna acotada: %v", orch.satelliteSpecs[0].Cmd)
		}

		config := orch.lastConfig
		if config == nil || !config.AttachSatellitesNet {
			t.Fatal("el laboratorio debe conectarse a la red interna de satélites")
		}
		if uri, _ := envValue(config.Env, "MONGODB_URI"); uri != "mongodb://solv-sat-mongodb:27017/solv_std_std0001_subj001" {
			t.Errorf("MONGODB_URI inyectada inesperada: %q", uri)
		}
		if db, _ := envValue(config.Env, "MONGO_DATABASE"); db != "solv_std_std0001_subj001" {
			t.Errorf("MONGO_DATABASE inyectada inesperada: %q", db)
		}
		if _, ok := envValue(config.Env, "DATABASE_URL"); ok {
			t.Error("un motor NoSQL no debe inyectar DATABASE_URL relacional")
		}
	})

	t.Run("redis", func(t *testing.T) {
		_, orch, svc := managedSatelliteFixture("redis", "")

		if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-sat"); err != nil {
			t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
		}

		if len(orch.satelliteSpecs) != 1 || orch.satelliteSpecs[0].Image != "redis:7.2-alpine" {
			t.Fatalf("satélite redis inesperado: %+v", orch.satelliteSpecs)
		}
		if orch.satelliteSpecs[0].Port != 6379 || orch.satelliteSpecs[0].MemoryLimitMB != 96 {
			t.Errorf("tuning redis inesperado: %+v", orch.satelliteSpecs[0])
		}

		config := orch.lastConfig
		if config == nil || !config.AttachSatellitesNet {
			t.Fatal("el laboratorio debe conectarse a la red interna de satélites")
		}
		if url, _ := envValue(config.Env, "REDIS_URL"); url != "redis://solv-sat-redis:6379/0" {
			t.Errorf("REDIS_URL inyectada inesperada: %q", url)
		}
		if host, _ := envValue(config.Env, "REDIS_HOST"); host != "solv-sat-redis" {
			t.Errorf("REDIS_HOST inyectado inesperado: %q", host)
		}
		if port, _ := envValue(config.Env, "REDIS_PORT"); port != "6379" {
			t.Errorf("REDIS_PORT inyectado inesperado: %q", port)
		}
		if prefix, _ := envValue(config.Env, "REDIS_KEY_PREFIX"); prefix != "solv_std_std0001_subj001:" {
			t.Errorf("REDIS_KEY_PREFIX inyectado inesperado: %q", prefix)
		}
	})

	// PostgreSQL de aula es un satélite dedicado más: nunca la base de la plataforma
	// (ADR-051). El alumno recibe rol y base propios dentro del motor de aula.
	t.Run("postgres", func(t *testing.T) {
		const script = "CREATE TABLE IF NOT EXISTS materias (id INT PRIMARY KEY);"
		_, orch, svc := managedSatelliteFixture("postgres", script)
		ctx := context.Background()

		if _, err := svc.StartWorkspaceForExercise(ctx, "std0001", "subj001", "ex-sat"); err != nil {
			t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
		}

		if len(orch.satelliteSpecs) != 1 {
			t.Fatalf("PostgreSQL de aula debe levantarse como satélite dedicado: %+v", orch.satelliteSpecs)
		}
		spec := orch.satelliteSpecs[0]
		if spec.Engine != "postgres" || spec.Image != "postgres:18-alpine" {
			t.Errorf("satélite inesperado: engine=%q image=%q", spec.Engine, spec.Image)
		}
		if spec.ContainerName != "solv-sat-postgres" || spec.VolumeName != "solv_sat_data_postgres" {
			t.Errorf("contenedor/volumen inesperados: %q / %q", spec.ContainerName, spec.VolumeName)
		}
		if spec.MemoryLimitMB != 384 || spec.Port != 5432 {
			t.Errorf("tuning inesperado: %d MB puerto %d", spec.MemoryLimitMB, spec.Port)
		}
		if len(spec.ReadyCmd) == 0 || spec.ReadyCmd[0] != "pg_isready" {
			t.Errorf("el satélite debe declarar readiness con pg_isready: %v", spec.ReadyCmd)
		}
		if strings.Contains(spec.Image, ":latest") {
			t.Errorf("la imagen del satélite no puede usar :latest: %q", spec.Image)
		}
		if cmd := strings.Join(spec.Cmd, " "); !strings.Contains(cmd, "shared_buffers") {
			t.Errorf("el motor debe arrancar con buffers acotados: %q", cmd)
		}
		// La imagen 18+ mueve el volumen al padre y PGDATA al subdirectorio de la versión.
		if env := strings.Join(spec.Env, " "); !strings.Contains(env, "PGDATA=/var/lib/postgresql/18/docker") {
			t.Errorf("PGDATA debe usar el layout oficial de la imagen 18+: %q", env)
		}
		if env := strings.Join(spec.Env, " "); strings.Contains(env, "PGDATA=/var/lib/postgresql/data") {
			t.Errorf("PGDATA no puede quedar en el layout heredado: %q", env)
		}

		// Rol y base exclusivos del alumno, nunca el superusuario del motor.
		if !orch.satelliteExecutedSQL("CREATE ROLE") {
			t.Error("debe crearse un rol exclusivo por alumno y materia")
		}
		if !orch.satelliteExecutedSQL("NOSUPERUSER NOCREATEDB NOCREATEROLE") {
			t.Error("el rol del alumno debe nacer sin privilegios de administración")
		}
		if !orch.satelliteExecutedSQL(`CREATE DATABASE "solv_std_std0001_subj001" OWNER`) {
			t.Error("debe crearse la base del alumno con su rol como dueño")
		}
		if !orch.satelliteExecutedSQL("REVOKE CONNECT ON DATABASE") {
			t.Error("la base del alumno no debe aceptar conexiones de PUBLIC")
		}
		if !orch.satelliteExecutedSQL("_solv_exercise_migrations") || !orch.satelliteExecutedSQL("CREATE TABLE IF NOT EXISTS materias") {
			t.Error("el db_init_script debe aplicarse dentro de la base del alumno con su sello")
		}
		if orch.satelliteScriptRuns != 1 {
			t.Fatalf("el db_init_script debe aplicarse una vez, aplicado %d veces", orch.satelliteScriptRuns)
		}

		config := orch.lastConfig
		if config == nil {
			t.Fatal("no se envió configuración de contenedor al orquestador")
		}
		if !config.AttachSatellitesNet {
			t.Error("el laboratorio debe conectarse a la red interna de satélites")
		}
		dsn, ok := envValue(config.Env, "DATABASE_URL")
		if !ok {
			t.Fatal("el laboratorio debe recibir DATABASE_URL")
		}
		if strings.Contains(dsn, "postgres:postgres@") {
			t.Errorf("DATABASE_URL no puede usar el superusuario del motor: %q", dsn)
		}
		user, _ := envValue(config.Env, "PGUSER")
		password, _ := envValue(config.Env, "PGPASSWORD")
		wantPrefix := "postgresql://" + user + ":" + password + "@solv-sat-postgres:5432/solv_std_std0001_subj001"
		if !strings.HasPrefix(dsn, wantPrefix) {
			t.Errorf("DATABASE_URL inyectada inesperada: %q", dsn)
		}
		if db, _ := envValue(config.Env, "PGDATABASE"); db != "solv_std_std0001_subj001" {
			t.Errorf("PGDATABASE inyectado inesperado: %q", db)
		}

		// Segundo arranque: el sello impide reaplicar el script del ejercicio.
		if _, err := svc.StartWorkspaceForExercise(ctx, "std0001", "subj001", "ex-sat"); err != nil {
			t.Fatalf("error inesperado al reabrir el laboratorio: %v", err)
		}
		if orch.satelliteScriptRuns != 1 {
			t.Fatalf("el db_init_script no debe reaplicarse: aplicado %d veces", orch.satelliteScriptRuns)
		}
	})
}

func TestStartWorkspaceForExercise_AppliesManagedDBInitScriptOnce(t *testing.T) {
	script := "CREATE TABLE IF NOT EXISTS materias (id INT PRIMARY KEY, nombre VARCHAR(80));"
	_, orch, svc := managedSatelliteFixture("mysql", script)
	ctx := context.Background()

	if _, err := svc.StartWorkspaceForExercise(ctx, "std0001", "subj001", "ex-sat"); err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
	}

	if !orch.satelliteExecutedSQL("_solv_exercise_migrations") {
		t.Error("el db_init_script necesita su registro de migraciones para ser idempotente")
	}
	if !orch.satelliteExecutedSQL("CREATE TABLE IF NOT EXISTS materias") {
		t.Error("el db_init_script del ejercicio debe aplicarse en la base del alumno")
	}
	if orch.satelliteScriptRuns != 1 {
		t.Fatalf("el db_init_script debe aplicarse una vez, aplicado %d veces", orch.satelliteScriptRuns)
	}

	// Segundo arranque del mismo laboratorio: el registro de migraciones evita
	// reaplicar el script y las credenciales aisladas se reescriben igual.
	if _, err := svc.StartWorkspaceForExercise(ctx, "std0001", "subj001", "ex-sat"); err != nil {
		t.Fatalf("error inesperado al reabrir el laboratorio: %v", err)
	}
	if orch.satelliteScriptRuns != 1 {
		t.Fatalf("el db_init_script no debe reaplicarse: aplicado %d veces", orch.satelliteScriptRuns)
	}
}

func TestHibernateWorkspace_StopsManagedSatelliteWhenNoLabUsesIt(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	templateID := "tpl-mysql-algebra"
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
				Services: []domain.ServiceRequirement{{Category: "database", Engine: "mysql"}},
			},
		},
	}}

	// Ambos laboratorios declaran la misma plantilla por ejercicio.
	newExercise := func(id, subjectID string) *domain.Exercise {
		return &domain.Exercise{ID: id, TemplateID: templateID, SubjectID: &subjectID}
	}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-1": newExercise("ex-1", "subj001"),
		"ex-2": newExercise("ex-2", "subj002"),
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

	// El primer alumno cierra sesión: el satélite sigue en uso por el segundo.
	if err := svc.HibernateWorkspace(ctx, first.ID); err != nil {
		t.Fatalf("error inesperado al hibernar: %v", err)
	}
	if containsEngine(orch.satelliteStops, "mysql") {
		t.Fatalf("el satélite mysql no debe detenerse mientras un laboratorio lo usa: %v", orch.satelliteStops)
	}

	// El último alumno cierra sesión: el motor se apaga y libera su memoria.
	if err := svc.HibernateWorkspace(ctx, second.ID); err != nil {
		t.Fatalf("error inesperado al hibernar: %v", err)
	}
	if !containsEngine(orch.satelliteStops, "mysql") {
		t.Fatalf("el satélite mysql debe detenerse cuando ningún laboratorio lo usa: %v", orch.satelliteStops)
	}
}

func TestTerminateWorkspace_StopsManagedSatelliteWhenNoLabUsesIt(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &mockOrchestrator{}
	templateID := "tpl-mongodb-web"
	subjectID := "subj001"
	subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{subjectID: {ID: subjectID}}}
	tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
		templateID: {
			ID:          templateID,
			DockerImage: "solv/stack:abc123",
			BaseRamMB:   1024,
			ServicesConfig: domain.ServicesConfig{
				Services: []domain.ServiceRequirement{{Category: "database", Engine: "mongodb"}},
			},
		},
	}}
	exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-1": {ID: "ex-1", TemplateID: templateID, SubjectID: &subjectID},
	}}

	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(subjRepo, tplRepo).
		WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})

	ctx := context.Background()
	ws, err := svc.StartWorkspaceForExercise(ctx, "std0001", subjectID, "ex-1")
	if err != nil {
		t.Fatalf("error inesperado al iniciar el laboratorio: %v", err)
	}

	// El laboratorio termina en fallo: deja de contar como uso activo.
	if err := svc.TerminateWorkspace(ctx, ws.ID); err != nil {
		t.Fatalf("error inesperado al terminar el laboratorio: %v", err)
	}
	if !containsEngine(orch.satelliteStops, "mongodb") {
		t.Fatalf("el satélite mongodb debe detenerse al no quedar laboratorios activos: %v", orch.satelliteStops)
	}
	if !containsEngine(orch.satelliteStops, "mysql") {
		t.Errorf("todos los motores gestionados sin uso deben apagarse, obtenidos: %v", orch.satelliteStops)
	}
}

func TestStartWorkspaceForExercise_ManagedSatelliteFailureAbortsStart(t *testing.T) {
	wsRepo, orch, svc := managedSatelliteFixture("mysql", "")
	orch.satelliteEnsureError = errors.New("docker daemon unavailable")

	if _, err := svc.StartWorkspaceForExercise(context.Background(), "std0001", "subj001", "ex-sat"); err == nil {
		t.Fatal("esperado error cuando el motor gestionado no puede levantarse")
	}

	// El laboratorio no debe quedar registrado como activo sin su base de datos.
	if len(wsRepo.workspaces) != 0 {
		t.Errorf("no debe crearse el workspace cuando el motor no está disponible: %+v", wsRepo.workspaces)
	}
}

func containsEngine(engines []string, want string) bool {
	for _, engine := range engines {
		if engine == want {
			return true
		}
	}
	return false
}
