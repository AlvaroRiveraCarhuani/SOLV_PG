package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// managedSatelliteEngines son los motores que SOLV levanta por demanda como contenedor
// satélite. PostgreSQL de aula tiene motor propio desde ADR-051: no comparte instancia
// con la base de la plataforma. SQLite es un archivo dentro del laboratorio.
var managedSatelliteEngines = []string{"postgres", "mysql", "mariadb", "mongodb", "redis"}

// managedSatelliteIdleGrace es el margen que el scale-to-zero respeta desde el último
// uso de un motor. Un laboratorio que se está aprovisionando todavía no figura como
// activo en la base de datos, y sin este margen el motor podía apagarse a mitad del
// arranque y arruinar la sesión del alumno.
const managedSatelliteIdleGrace = 2 * time.Minute

// satelliteProvision es el resultado de asegurar los motores declarados por la
// plantilla para un laboratorio: las variables de entorno con credenciales aisladas que
// recibe el contenedor del alumno.
type satelliteProvision struct {
	// Env son las variables de entorno inyectadas al laboratorio (DATABASE_URL y
	// afines) que apuntan al motor con usuario, base y contraseña exclusivos.
	Env []string
	// UsesManagedContainer indica que el laboratorio debe conectarse a la red interna
	// de satélites para alcanzar el motor gestionado.
	UsesManagedContainer bool
}

// noteManagedSatelliteUse registra que un laboratorio pidió el motor en este momento.
func (s *WorkspaceService) noteManagedSatelliteUse(engine string) {
	if strings.TrimSpace(engine) == "" {
		return
	}
	s.satelliteUseMu.Lock()
	if s.satelliteLastUse == nil {
		s.satelliteLastUse = make(map[string]time.Time)
	}
	s.satelliteLastUse[engine] = time.Now()
	s.satelliteUseMu.Unlock()
}

// managedSatelliteRecentlyUsed indica si el motor se pidió dentro del margen de gracia.
func (s *WorkspaceService) managedSatelliteRecentlyUsed(engine string) bool {
	s.satelliteUseMu.Lock()
	last, ok := s.satelliteLastUse[engine]
	s.satelliteUseMu.Unlock()
	if !ok {
		return false
	}
	return time.Since(last) < managedSatelliteIdleGrace
}

func isManagedSatelliteEngine(engine string) bool {
	for _, candidate := range managedSatelliteEngines {
		if candidate == engine {
			return true
		}
	}
	return false
}

// templateServiceBindings resuelve todos los servicios declarados por la plantilla, con
// el motor canonizado y la base principal primero. Los motores desconocidos y 'none' se
// descartan: la validación de alta ya los rechaza, pero una plantilla antigua puede
// tenerlos y el arranque no debe romperse por eso.
func templateServiceBindings(template *domain.Template) []domain.ServiceRequirement {
	if template == nil {
		return nil
	}
	var bindings []domain.ServiceRequirement
	for _, service := range template.ServicesConfig.Services {
		engine := domain.CanonicalServiceEngine(service.Engine)
		if engine == "" || engine == "none" {
			continue
		}
		service.Engine = engine
		bindings = append(bindings, service)
	}
	sort.SliceStable(bindings, func(i, j int) bool {
		return serviceRoleRank(bindings[i]) < serviceRoleRank(bindings[j])
	})
	return bindings
}

func serviceRoleRank(binding domain.ServiceRequirement) int {
	if binding.EffectiveRole() == domain.ServiceRolePrimaryDatabase {
		return 0
	}
	return 1
}

// templateManagedEngines devuelve los motores de contenedor que la plantilla declara,
// sin duplicados. La lista vacía significa que el laboratorio no depende de un satélite
// gestionado.
func templateManagedEngines(template *domain.Template) []string {
	bindings := templateServiceBindings(template)
	var engines []string
	seen := make(map[string]bool, len(bindings))
	for _, binding := range bindings {
		engine := domain.CanonicalServiceEngine(binding.Engine)
		if !isManagedSatelliteEngine(engine) || seen[engine] {
			continue
		}
		seen[engine] = true
		engines = append(engines, engine)
	}
	return engines
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
	case "postgres":
		return strings.TrimSpace(os.Getenv("POSTGRES_SATELLITE_HOST"))
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
	case "postgres":
		return domain.ManagedSatelliteSpec{
			Engine:        "postgres",
			Image:         "postgres:18-alpine",
			ContainerName: "solv-sat-postgres",
			VolumeName:    "solv_sat_data_postgres",
			NetworkName:   "solv-satellites-net",
			MemoryLimitMB: 384,
			Port:          5432,
			Env: []string{
				"POSTGRES_PASSWORD=" + rootPassword,
				// Layout oficial de la imagen 18+: el volumen se monta en el padre y
				// PGDATA vive en el subdirectorio de la versión mayor. Fijarlo evita
				// depender del default de la imagen y del layout heredado que la 18
				// rechaza.
				"PGDATA=/var/lib/postgresql/18/docker",
			},
			Cmd: []string{
				"-c", "shared_buffers=96MB",
				"-c", "max_connections=100",
				"-c", "work_mem=2MB",
				"-c", "maintenance_work_mem=32MB",
			},
			ReadyCmd: []string{"pg_isready", "-h", "127.0.0.1", "-U", "postgres"},
		}, true
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
	s.noteManagedSatelliteUse(engine)
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
	case "postgres":
		user, password := satelliteIsolationIdentity(studentID, subjectID)
		if usesManagedContainer && hasSpec && orchestrator != nil {
			if err := s.ensurePostgresIsolatedDatabase(ctx, orchestrator, spec, dbName, user, password, exercise); err != nil {
				return satelliteProvision{}, err
			}
		} else {
			log.Printf("[Satellites] Motor postgres sin aprovisionamiento local (host %q): se inyectan credenciales sin base aislada", host)
		}
		port := spec.Port
		if port == 0 {
			port = 5432
		}
		return satelliteProvision{
			UsesManagedContainer: usesManagedContainer,
			Env: []string{
				fmt.Sprintf("DATABASE_URL=postgresql://%s:%s@%s:%d/%s?sslmode=disable", user, password, host, port, dbName),
				fmt.Sprintf("PGHOST=%s", host),
				fmt.Sprintf("PGPORT=%d", port),
				fmt.Sprintf("PGUSER=%s", user),
				fmt.Sprintf("PGPASSWORD=%s", password),
				fmt.Sprintf("PGDATABASE=%s", dbName),
			},
		}, nil
	case "mysql", "mariadb":
		user, password := satelliteIsolationIdentity(studentID, subjectID)
		if usesManagedContainer && hasSpec && orchestrator != nil {
			if err := s.ensureMySQLIsolatedDatabase(ctx, orchestrator, spec, dbName, user, password, exercise); err != nil {
				return satelliteProvision{}, err
			}
		} else {
			log.Printf("[Satellites] Motor %s sin aprovisionamiento local (host %q): se inyectan credenciales sin base aislada", engine, host)
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
		if usesManagedContainer && hasSpec && orchestrator != nil {
			if err := s.applyManagedMongoExerciseDBInitScript(ctx, orchestrator, spec, dbName, exercise); err != nil {
				return satelliteProvision{}, err
			}
		}
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

// managedSatelliteMigrationCollection sella, dentro de cada base del alumno, qué
// ejercicio ya aplicó su db_init_script. Es el equivalente NoSQL de
// _solv_exercise_migrations en SQL.
const managedSatelliteMigrationCollection = "_solv_exercise_migrations"

// applyManagedMongoExerciseDBInitScript aplica el db_init_script del ejercicio
// (JavaScript de mongosh) una sola vez sobre la base aislada del alumno. El guard y el
// sellado viajan en la misma invocación: si el script falla a mitad, el ejercicio no
// queda registrado y se reintenta en el próximo arranque.
func (s *WorkspaceService) applyManagedMongoExerciseDBInitScript(
	ctx context.Context,
	orchestrator domain.ManagedSatelliteOrchestrator,
	spec domain.ManagedSatelliteSpec,
	dbName string,
	exercise *domain.Exercise,
) error {
	script := exerciseDBInitScript(exercise)
	if script == "" {
		return nil
	}
	if kind := (domain.ServiceRequirement{Engine: spec.Engine}).InitKind(); kind != domain.ServiceInitMongoJS {
		return fmt.Errorf("el motor %s no admite db_init_script (init %s)", spec.Engine, kind)
	}

	exerciseID := escapeJSString(exercise.ID)
	eval := fmt.Sprintf(
		"if (db.getCollection(\"%s\").findOne({ exercise_id: \"%s\" }) === null) {\n%s\n  db.getCollection(\"%s\").insertOne({ exercise_id: \"%s\", applied_at: new Date() });\n}",
		managedSatelliteMigrationCollection, exerciseID, script, managedSatelliteMigrationCollection, exerciseID,
	)

	port := spec.Port
	if port == 0 {
		port = 27017
	}
	args := []string{
		"mongosh", "--quiet",
		fmt.Sprintf("mongodb://127.0.0.1:%d/%s", port, dbName),
		"--eval", eval,
	}
	if _, err := orchestrator.ExecInManagedSatellite(ctx, spec.Engine, args); err != nil {
		return fmt.Errorf("exercise %s db_init_script failed on %s: %w", exercise.ID, dbName, err)
	}
	log.Printf("[Satellites] db_init_script del ejercicio %s aplicado/verificado en %s", exercise.ID, dbName)
	return nil
}

// ensurePostgresIsolatedDatabase crea el rol y la base exclusivos del alumno y aplica
// una sola vez el db_init_script del ejercicio. Todo el SQL viaja a psql dentro del
// contenedor: el backend no abre conexiones al motor de aula ni usa la base de la
// plataforma para las prácticas (ADR-051).
func (s *WorkspaceService) ensurePostgresIsolatedDatabase(
	ctx context.Context,
	orchestrator domain.ManagedSatelliteOrchestrator,
	spec domain.ManagedSatelliteSpec,
	dbName, user, password string,
	exercise *domain.Exercise,
) error {
	port := spec.Port
	if port == 0 {
		port = 5432
	}
	rootURL := fmt.Sprintf("postgresql://postgres:%s@127.0.0.1:%d/postgres?sslmode=disable", satelliteRootPassword(), port)

	roleSQL := fmt.Sprintf(
		"DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%s') THEN "+
			"CREATE ROLE \"%s\" LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE; "+
			"ELSE ALTER ROLE \"%s\" WITH LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE; END IF; END $$;",
		escapeSQLString(user), user, password, user, password,
	)
	if _, err := execPostgresSQL(ctx, orchestrator, spec.Engine, rootURL, roleSQL); err != nil {
		return fmt.Errorf("no se pudo preparar el rol aislado %s en el satélite postgres: %w", user, err)
	}

	existsSQL := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s';", escapeSQLString(dbName))
	exists, err := execPostgresSQL(ctx, orchestrator, spec.Engine, rootURL, existsSQL)
	if err != nil {
		return fmt.Errorf("no se pudo verificar la base aislada %s: %w", dbName, err)
	}
	if strings.TrimSpace(exists) != "1" {
		createSQL := fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\";", dbName, user)
		if _, err := execPostgresSQL(ctx, orchestrator, spec.Engine, rootURL, createSQL); err != nil {
			return fmt.Errorf("no se pudo crear la base aislada %s: %w", dbName, err)
		}
	}

	grantSQL := fmt.Sprintf(
		"REVOKE CONNECT ON DATABASE \"%s\" FROM PUBLIC; GRANT CONNECT ON DATABASE \"%s\" TO \"%s\";",
		dbName, dbName, user,
	)
	if _, err := execPostgresSQL(ctx, orchestrator, spec.Engine, rootURL, grantSQL); err != nil {
		return fmt.Errorf("no se pudieron ajustar los permisos de %s: %w", dbName, err)
	}

	script := exerciseDBInitScript(exercise)
	if script == "" {
		return nil
	}
	studentURL := fmt.Sprintf("postgresql://%s:%s@127.0.0.1:%d/%s?sslmode=disable", user, password, port, dbName)
	if err := s.applyManagedPostgresExerciseDBInitScript(ctx, orchestrator, spec, studentURL, exercise.ID, script); err != nil {
		return fmt.Errorf("exercise %s db_init_script failed on %s: %w", exercise.ID, dbName, err)
	}
	log.Printf("[Satellites] db_init_script del ejercicio %s aplicado/verificado en %s", exercise.ID, dbName)
	return nil
}

// execPostgresSQL ejecuta un statement con psql dentro del satélite y devuelve stdout.
// ON_ERROR_STOP hace que un error del script corte la ejecución y no se selle.
func execPostgresSQL(ctx context.Context, orchestrator domain.ManagedSatelliteOrchestrator, engine, url, sql string) (string, error) {
	return orchestrator.ExecInManagedSatellite(ctx, engine, []string{"psql", url, "-v", "ON_ERROR_STOP=1", "-tAc", sql})
}

// applyManagedPostgresExerciseDBInitScript aplica el script del ejercicio una sola vez
// por exercise_id, dentro de la base aislada y con el rol del alumno.
func (s *WorkspaceService) applyManagedPostgresExerciseDBInitScript(
	ctx context.Context,
	orchestrator domain.ManagedSatelliteOrchestrator,
	spec domain.ManagedSatelliteSpec,
	studentURL, exerciseID, script string,
) error {
	const migrationsDDL = "CREATE TABLE IF NOT EXISTS _solv_exercise_migrations " +
		"(exercise_id TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW());"

	if _, err := execPostgresSQL(ctx, orchestrator, spec.Engine, studentURL, migrationsDDL); err != nil {
		return fmt.Errorf("failed to ensure exercise migrations table: %w", err)
	}

	checkSQL := fmt.Sprintf("SELECT 1 FROM _solv_exercise_migrations WHERE exercise_id = '%s';", escapeSQLString(exerciseID))
	applied, err := execPostgresSQL(ctx, orchestrator, spec.Engine, studentURL, checkSQL)
	if err != nil {
		return fmt.Errorf("failed to check applied db_init_script: %w", err)
	}
	if strings.TrimSpace(applied) == "1" {
		return nil
	}

	if _, err := execPostgresSQL(ctx, orchestrator, spec.Engine, studentURL, script); err != nil {
		return fmt.Errorf("db_init_script failed: %w", err)
	}

	registerSQL := fmt.Sprintf(
		"INSERT INTO _solv_exercise_migrations (exercise_id) VALUES ('%s') ON CONFLICT (exercise_id) DO NOTHING;",
		escapeSQLString(exerciseID),
	)
	if _, err := execPostgresSQL(ctx, orchestrator, spec.Engine, studentURL, registerSQL); err != nil {
		return fmt.Errorf("failed to register applied db_init_script: %w", err)
	}
	return nil
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

// escapeJSString neutraliza los caracteres que rompen un literal JavaScript construido
// en Go.
func escapeJSString(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`).Replace(value)
}

// ReconcileIdleManagedSatellites apaga los motores de contenedor que ya no usa ningún
// laboratorio en ejecución, liberando toda su memoria del host. Se llama desde una acción
// explícita (hibernar, pausar o terminar un laboratorio), así que apaga en el acto. El
// volumen nombrado y los datos sobreviven; la próxima sesión reactiva el mismo contenedor.
func (s *WorkspaceService) ReconcileIdleManagedSatellites(ctx context.Context) {
	s.reconcileIdleManagedSatellites(ctx, false)
}

// ReconcileIdleManagedSatellitesScheduled es la variante que corre el worker de QoS. Como
// pasa cada pocos segundos y un laboratorio en aprovisionamiento todavía no figura como
// activo en la base de datos, respeta el margen de gracia desde el último uso del motor.
func (s *WorkspaceService) ReconcileIdleManagedSatellitesScheduled(ctx context.Context) {
	s.reconcileIdleManagedSatellites(ctx, true)
}

func (s *WorkspaceService) reconcileIdleManagedSatellites(ctx context.Context, respectGrace bool) {
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
		for _, engine := range s.managedEnginesForWorkspace(ctx, workspace) {
			inUse[engine]++
		}
	}

	var stopped []string
	for _, engine := range managedSatelliteEngines {
		if inUse[engine] > 0 {
			continue
		}
		// Un motor recién usado por un laboratorio que aún no figura como activo sigue
		// siendo necesario: se apaga en el siguiente ciclo, ya fuera del arranque.
		if respectGrace && s.managedSatelliteRecentlyUsed(engine) {
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

// managedEnginesForWorkspace resuelve todos los motores gestionados que un laboratorio
// concreto mantiene en uso, resolviendo primero la plantilla del propio laboratorio.
func (s *WorkspaceService) managedEnginesForWorkspace(ctx context.Context, workspace *domain.WorkspaceInstance) []string {
	if workspace == nil {
		return nil
	}
	if workspace.TemplateID != nil && strings.TrimSpace(*workspace.TemplateID) != "" && s.templateRepo != nil {
		if template, err := s.templateRepo.GetTemplateByID(ctx, strings.TrimSpace(*workspace.TemplateID)); err == nil && template != nil {
			return templateManagedEngines(template)
		}
	}
	return templateManagedEngines(s.resolveTemplate(ctx, workspace.SubjectID))
}
