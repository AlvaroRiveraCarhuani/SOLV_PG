package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"solv-backend/internal/core/domain"
)

// managedSatelliteEngines son los motores que SOLV levanta por demanda como contenedor
// satélite. PostgreSQL de aula vive en el host compartido y Redis comparte el mismo
// mecanismo; SQLite es un archivo dentro del laboratorio.
var managedSatelliteEngines = []string{"mysql", "mariadb", "mongodb", "redis"}

// satelliteProvision es el resultado de asegurar el motor declarado por la plantilla
// para un laboratorio: la base compartida de PostgreSQL cuando aplica y las variables de
// entorno con credenciales aisladas que recibe el contenedor del alumno.
type satelliteProvision struct {
	// PostgresDBName es la base del aula creada para el par alumno/materia.
	PostgresDBName string
	// Env son las variables de entorno inyectadas al laboratorio (DATABASE_URL y
	// afines) que apuntan al motor con usuario, base y contraseña exclusivos.
	Env []string
	// UsesManagedContainer indica que el laboratorio debe conectarse a la red interna
	// de satélites para alcanzar el motor gestionado.
	UsesManagedContainer bool
}

func isManagedSatelliteEngine(engine string) bool {
	for _, candidate := range managedSatelliteEngines {
		if candidate == engine {
			return true
		}
	}
	return false
}

// templateSatelliteEngine resuelve el motor de base de datos declarado por la plantilla
// en services_config.services (postgres, mysql, mariadb, mongodb o redis).
func templateSatelliteEngine(template *domain.Template) string {
	if template == nil {
		return ""
	}
	for _, service := range template.ServicesConfig.Services {
		engine := strings.ToLower(strings.TrimSpace(service.Engine))
		if engine == "postgres" || isManagedSatelliteEngine(engine) {
			return engine
		}
	}
	return ""
}

// templateManagedEngine resuelve el motor de contenedor declarado por la plantilla,
// vacío cuando el laboratorio no depende de un satélite gestionado.
func templateManagedEngine(template *domain.Template) string {
	engine := templateSatelliteEngine(template)
	if isManagedSatelliteEngine(engine) {
		return engine
	}
	return ""
}

// managedSatelliteHostOverride resuelve el destino externo declarado por un
// administrador de sistema. Es un override de destino: cuando está presente, el motor ya
// existe fuera del nodo y SOLV no levanta ni apaga contenedores.
func managedSatelliteHostOverride(engine string) string {
	switch engine {
	case "mysql":
		return strings.TrimSpace(os.Getenv("MYSQL_SATELLITE_HOST"))
	case "mariadb":
		if host := strings.TrimSpace(os.Getenv("MARIADB_SATELLITE_HOST")); host != "" {
			return host
		}
		return strings.TrimSpace(os.Getenv("MYSQL_SATELLITE_HOST"))
	case "mongodb":
		return strings.TrimSpace(os.Getenv("MONGODB_SATELLITE_HOST"))
	case "redis":
		return strings.TrimSpace(os.Getenv("REDIS_SATELLITE_HOST"))
	default:
		return ""
	}
}

// satelliteIsolationSecret es la clave con la que se derivan las credenciales por alumno.
// Un despliegue sin SOLV_SATELLITE_SECRET reutiliza el secreto de sesión del nodo y, como
// último recurso, una clave fija para que el desarrollo local y los tests sean
// deterministas: la contraseña derivada es idempotente entre arranques.
func satelliteIsolationSecret() string {
	for _, key := range []string{"SOLV_SATELLITE_SECRET", "JWT_SECRET"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return "solv-satellite-isolation"
}

func satelliteHMAC(seed string) string {
	mac := hmac.New(sha256.New, []byte(satelliteIsolationSecret()))
	mac.Write([]byte(seed))
	return hex.EncodeToString(mac.Sum(nil))
}

// satelliteIsolationIdentity deriva el usuario y la contraseña exclusivos del par
// alumno/materia. Al ser deterministas, una reactivación del mismo laboratorio reutiliza
// las mismas credenciales en lugar de acumular usuarios huérfanos.
func satelliteIsolationIdentity(studentID, subjectID string) (string, string) {
	digest := satelliteHMAC(studentID + ":" + subjectID)
	return "u_" + digest[:12], digest[12:28]
}

// satelliteRootPassword deriva la contraseña de administración del contenedor satélite.
func satelliteRootPassword() string {
	return satelliteHMAC("solv-satellite-root")[:24]
}

// satelliteDatabaseName es el espacio de datos exclusivo del par alumno/materia.
func satelliteDatabaseName(studentID, subjectID string) string {
	return fmt.Sprintf("solv_std_%s_%s", sanitizeIDForDB(studentID), sanitizeIDForDB(subjectID))
}

// exerciseDBInitScript devuelve el script de inicialización declarado por el ejercicio.
func exerciseDBInitScript(exercise *domain.Exercise) string {
	if exercise == nil || exercise.Config.IDE == nil {
		return ""
	}
	return strings.TrimSpace(exercise.Config.IDE.DBInitScript)
}

// managedSatelliteSpec describe cómo levantar cada motor gestionado con la sintonización
// más liviana posible: la RAM se acota por cgroups y cada motor se configura para no
// reservar buffers que un laboratorio docente no necesita.
func managedSatelliteSpec(engine string) (domain.ManagedSatelliteSpec, bool) {
	rootPassword := satelliteRootPassword()

	switch engine {
	case "mysql":
		return domain.ManagedSatelliteSpec{
			Engine:        "mysql",
			Image:         "mysql:8.4",
			ContainerName: "solv-sat-mysql",
			VolumeName:    "solv_sat_data_mysql",
			NetworkName:   "solv-satellites-net",
			MemoryLimitMB: 256,
			Port:          3306,
			Env:           []string{"MYSQL_ROOT_PASSWORD=" + rootPassword},
			Cmd: []string{
				"--innodb-buffer-pool-size=64M",
				"--performance-schema=OFF",
				"--max-connections=100",
			},
			ReadyCmd: []string{"mysql", "-uroot", "-p" + rootPassword, "-e", "SELECT 1"},
		}, true
	case "mariadb":
		return domain.ManagedSatelliteSpec{
			Engine:        "mariadb",
			Image:         "mariadb:11.4",
			ContainerName: "solv-sat-mariadb",
			VolumeName:    "solv_sat_data_mariadb",
			NetworkName:   "solv-satellites-net",
			MemoryLimitMB: 256,
			Port:          3306,
			Env:           []string{"MARIADB_ROOT_PASSWORD=" + rootPassword},
			Cmd: []string{
				"--innodb-buffer-pool-size=64M",
				"--performance-schema=OFF",
				"--max-connections=100",
			},
			ReadyCmd: []string{"mysql", "-uroot", "-p" + rootPassword, "-e", "SELECT 1"},
		}, true
	case "mongodb":
		return domain.ManagedSatelliteSpec{
			Engine:        "mongodb",
			Image:         "mongo:7.0",
			ContainerName: "solv-sat-mongodb",
			VolumeName:    "solv_sat_data_mongodb",
			NetworkName:   "solv-satellites-net",
			MemoryLimitMB: 256,
			Port:          27017,
			Cmd:           []string{"--wiredTigerCacheSizeGB", "0.25", "--quiet"},
			ReadyCmd:      []string{"mongosh", "--quiet", "--eval", "db.adminCommand({ping:1}).ok"},
		}, true
	case "redis":
		return domain.ManagedSatelliteSpec{
			Engine:        "redis",
			Image:         "redis:7.2-alpine",
			ContainerName: "solv-sat-redis",
			VolumeName:    "solv_sat_data_redis",
			NetworkName:   "solv-satellites-net",
			MemoryLimitMB: 96,
			Port:          6379,
			Cmd: []string{
				"redis-server",
				"--maxmemory", "64mb",
				"--maxmemory-policy", "allkeys-lru",
				"--appendonly", "yes",
			},
			ReadyCmd: []string{"redis-cli", "ping"},
		}, true
	default:
		return domain.ManagedSatelliteSpec{}, false
	}
}

// provisionManagedSatellite asegura el motor gestionado declarado por la plantilla y
// devuelve las variables de entorno con credenciales aisladas por alumno y materia. La
// base y el usuario son idempotentes: reconectar el mismo laboratorio no duplica nada.
func (s *WorkspaceService) provisionManagedSatellite(
	ctx context.Context,
	engine, studentID, subjectID string,
	exercise *domain.Exercise,
) (satelliteProvision, error) {
	dbName := satelliteDatabaseName(studentID, subjectID)
	spec, hasSpec := managedSatelliteSpec(engine)
	orchestrator, hasOrchestrator := s.docker.(domain.ManagedSatelliteOrchestrator)

	host := managedSatelliteHostOverride(engine)
	usesManagedContainer := false
	if host == "" {
		if !hasSpec || !hasOrchestrator {
			// Sin orquestador de satélites (tests, mocks y despliegues parciales) el
			// arranque del laboratorio no se rompe: se omite el motor y se registra.
			log.Printf("[Satellites] El orquestador inyectado no soporta satélites gestionados: motor %s omitido", engine)
			return satelliteProvision{}, nil
		}
		endpoint, err := orchestrator.EnsureManagedSatelliteRunning(ctx, spec)
		if err != nil {
			return satelliteProvision{}, fmt.Errorf("no se pudo levantar el satélite %s: %w", engine, err)
		}
		host = strings.TrimSpace(endpoint.Host)
		usesManagedContainer = true
	}

	switch engine {
	case "mysql", "mariadb":
		user, password := satelliteIsolationIdentity(studentID, subjectID)
		if host != "" && hasSpec && orchestrator != nil {
			if err := s.ensureMySQLIsolatedDatabase(ctx, orchestrator, spec, dbName, user, password, exercise); err != nil {
				return satelliteProvision{}, err
			}
		} else {
			log.Printf("[Satellites] Motor %s apunta a un host externo %q: se inyectan credenciales sin aprovisionar", engine, host)
		}
		port := spec.Port
		if port == 0 {
			port = 3306
		}
		return satelliteProvision{
			UsesManagedContainer: usesManagedContainer,
			Env: []string{
				fmt.Sprintf("DATABASE_URL=mysql://%s:%s@%s:%d/%s", user, password, host, port, dbName),
				fmt.Sprintf("MYSQL_HOST=%s", host),
				fmt.Sprintf("MYSQL_PORT=%d", port),
				fmt.Sprintf("MYSQL_USER=%s", user),
				fmt.Sprintf("MYSQL_PASSWORD=%s", password),
				fmt.Sprintf("MYSQL_DATABASE=%s", dbName),
			},
		}, nil

	case "mongodb":
		return satelliteProvision{
			UsesManagedContainer: usesManagedContainer,
			Env: []string{
				fmt.Sprintf("MONGODB_URI=mongodb://%s:27017/%s", host, dbName),
				fmt.Sprintf("MONGO_DATABASE=%s", dbName),
			},
		}, nil

	case "redis":
		return satelliteProvision{
			UsesManagedContainer: usesManagedContainer,
			Env: []string{
				fmt.Sprintf("REDIS_URL=redis://%s:6379/0", host),
				fmt.Sprintf("REDIS_HOST=%s", host),
				"REDIS_PORT=6379",
				fmt.Sprintf("REDIS_KEY_PREFIX=%s:", dbName),
			},
		}, nil

	default:
		return satelliteProvision{}, nil
	}
}

// ensureMySQLIsolatedDatabase crea la base y el usuario exclusivos del alumno y aplica
// una sola vez el db_init_script del ejercicio, registrándolo en
// _solv_exercise_migrations dentro de la propia base del alumno.
func (s *WorkspaceService) ensureMySQLIsolatedDatabase(
	ctx context.Context,
	orchestrator domain.ManagedSatelliteOrchestrator,
	spec domain.ManagedSatelliteSpec,
	dbName, user, password string,
	exercise *domain.Exercise,
) error {
	rootPassword := satelliteRootPassword()

	setupSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s`; "+
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'; "+
			"ALTER USER '%s'@'%%' IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'; FLUSH PRIVILEGES;",
		dbName, user, password, user, password, dbName, user,
	)
	if _, err := orchestrator.ExecInManagedSatellite(ctx, spec.Engine, mysqlAdminArgs(rootPassword, "-e", setupSQL)); err != nil {
		return fmt.Errorf("no se pudo preparar la base aislada %s en el satélite %s: %w", dbName, spec.Engine, err)
	}

	script := exerciseDBInitScript(exercise)
	if script == "" {
		return nil
	}
	if err := s.applyManagedExerciseDBInitScript(ctx, orchestrator, spec, rootPassword, dbName, exercise.ID, script); err != nil {
		return fmt.Errorf("exercise %s db_init_script failed on %s: %w", exercise.ID, dbName, err)
	}
	log.Printf("[Satellites] db_init_script del ejercicio %s aplicado/verificado en %s", exercise.ID, dbName)
	return nil
}

// applyManagedExerciseDBInitScript aplica el script del ejercicio una sola vez por
// exercise_id, con el mismo contrato idempotente que la ruta PostgreSQL.
func (s *WorkspaceService) applyManagedExerciseDBInitScript(
	ctx context.Context,
	orchestrator domain.ManagedSatelliteOrchestrator,
	spec domain.ManagedSatelliteSpec,
	rootPassword, dbName, exerciseID, script string,
) error {
	const migrationsDDL = "CREATE TABLE IF NOT EXISTS _solv_exercise_migrations " +
		"(exercise_id VARCHAR(191) PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);"

	if _, err := orchestrator.ExecInManagedSatellite(ctx, spec.Engine, mysqlDatabaseArgs(rootPassword, dbName, "-e", migrationsDDL)); err != nil {
		return fmt.Errorf("failed to ensure exercise migrations table: %w", err)
	}

	checkSQL := fmt.Sprintf("SELECT COUNT(*) FROM _solv_exercise_migrations WHERE exercise_id = '%s';", escapeSQLString(exerciseID))
	applied, err := orchestrator.ExecInManagedSatellite(ctx, spec.Engine, mysqlDatabaseArgs(rootPassword, dbName, "-N", "-B", "-e", checkSQL))
	if err != nil {
		return fmt.Errorf("failed to check applied db_init_script: %w", err)
	}
	if strings.TrimSpace(applied) != "0" {
		return nil
	}

	if _, err := orchestrator.ExecInManagedSatellite(ctx, spec.Engine, mysqlDatabaseArgs(rootPassword, dbName, "-e", script)); err != nil {
		return fmt.Errorf("db_init_script failed: %w", err)
	}

	registerSQL := fmt.Sprintf(
		"INSERT INTO _solv_exercise_migrations (exercise_id) VALUES ('%s') ON DUPLICATE KEY UPDATE exercise_id = exercise_id;",
		escapeSQLString(exerciseID),
	)
	if _, err := orchestrator.ExecInManagedSatellite(ctx, spec.Engine, mysqlDatabaseArgs(rootPassword, dbName, "-e", registerSQL)); err != nil {
		return fmt.Errorf("failed to register applied db_init_script: %w", err)
	}
	return nil
}

// mysqlAdminArgs construye la invocación del cliente MySQL como administrador del motor.
func mysqlAdminArgs(rootPassword string, extra ...string) []string {
	args := []string{"mysql", "-uroot", "-p" + rootPassword}
	return append(args, extra...)
}

// mysqlDatabaseArgs construye la invocación del cliente MySQL sobre la base del alumno.
func mysqlDatabaseArgs(rootPassword, dbName string, extra ...string) []string {
	args := []string{"mysql", "-uroot", "-p" + rootPassword, dbName}
	return append(args, extra...)
}

// escapeSQLString neutraliza comillas simples en literales SQL construidos en Go.
func escapeSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

// ReconcileIdleManagedSatellites apaga los motores de contenedor que ya no usa ningún
// laboratorio en ejecución, liberando toda su memoria del host. El volumen nombrado y los
// datos sobreviven; la próxima sesión reactiva el mismo contenedor.
func (s *WorkspaceService) ReconcileIdleManagedSatellites(ctx context.Context) {
	orchestrator, ok := s.docker.(domain.ManagedSatelliteOrchestrator)
	if !ok || s.repo == nil {
		return
	}

	running, err := s.repo.GetAllRunningWorkspaces(ctx)
	if err != nil {
		log.Printf("[Scale-to-Zero] no se pudo listar los laboratorios activos: %v", err)
		return
	}

	inUse := make(map[string]int)
	for _, workspace := range running {
		if workspace == nil {
			continue
		}
		if engine := s.managedEngineForWorkspace(ctx, workspace); engine != "" {
			inUse[engine]++
		}
	}

	var stopped []string
	for _, engine := range managedSatelliteEngines {
		if inUse[engine] > 0 {
			continue
		}
		// Un destino externo declarado por el sysadmin no es un contenedor del nodo.
		if managedSatelliteHostOverride(engine) != "" {
			continue
		}
		if err := orchestrator.StopManagedSatellite(ctx, engine); err != nil {
			log.Printf("[Scale-to-Zero] no se pudo detener el satélite %s: %v", engine, err)
			continue
		}
		stopped = append(stopped, engine)
	}

	if len(stopped) > 0 {
		log.Printf("[Scale-to-Zero] motores gestionados sin laboratorios activos, detenidos: %s", strings.Join(stopped, ", "))
	}
}

// managedEngineForWorkspace resuelve el motor gestionado que un laboratorio concreto
// mantiene en uso, resolviendo primero la plantilla del propio laboratorio.
func (s *WorkspaceService) managedEngineForWorkspace(ctx context.Context, workspace *domain.WorkspaceInstance) string {
	if workspace == nil {
		return ""
	}
	if workspace.TemplateID != nil && strings.TrimSpace(*workspace.TemplateID) != "" && s.templateRepo != nil {
		if template, err := s.templateRepo.GetTemplateByID(ctx, strings.TrimSpace(*workspace.TemplateID)); err == nil && template != nil {
			return templateManagedEngine(template)
		}
	}
	return templateManagedEngine(s.resolveTemplate(ctx, workspace.SubjectID))
}
