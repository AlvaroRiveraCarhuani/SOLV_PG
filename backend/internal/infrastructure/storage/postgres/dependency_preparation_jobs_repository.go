package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// dependencyPreparationJobColumns es la lista única que leen el encolado, el
// reclamo y las consultas por versión. component_id se devuelve vacío en lugar
// de NULL para que el trabajo tenga una forma estable en JSON.
const dependencyPreparationJobColumns = `id, tenant_id, version_id, project_id,
	COALESCE(component_id::text, '') AS component_id, preparation_key, status, attempts,
	log_path, message, tree_path, bytes, created_at, updated_at`

// PostgresDependencyPreparationJobRepository persiste la cola de preparación.
// La cola está en la base para que un reinicio del nodo no pierda trabajo.
type PostgresDependencyPreparationJobRepository struct {
	db *sqlx.DB
}

func NewPostgresDependencyPreparationJobRepository(db *sqlx.DB) *PostgresDependencyPreparationJobRepository {
	return &PostgresDependencyPreparationJobRepository{db: db}
}

// Enqueue inserta el trabajo y, si la versión ya tenía uno, devuelve el
// existente. El estado del trabajo no se reescribe: resolver dos veces la misma
// publicación no puede devolver a `pending` algo que ya está preparado.
func (r *PostgresDependencyPreparationJobRepository) Enqueue(
	ctx context.Context,
	job *domain.DependencyPreparationJob,
) (*domain.DependencyPreparationJob, error) {
	if job == nil {
		return nil, fmt.Errorf("%w: trabajo ausente", domain.ErrInvalidDependencyPreparationJob)
	}
	candidate := *job
	if candidate.ID == "" {
		candidate.ID = uuid.NewString()
	}
	if err := candidate.Validate(); err != nil {
		return nil, err
	}

	var component any
	if candidate.ComponentID != "" {
		component = candidate.ComponentID
	}

	query := `
		INSERT INTO dependency_preparation_jobs
			(id, tenant_id, version_id, project_id, component_id, preparation_key, status, attempts, log_path, message, tree_path, bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (version_id) DO NOTHING
		RETURNING ` + dependencyPreparationJobColumns

	var stored domain.DependencyPreparationJob
	err := r.db.QueryRowxContext(ctx, query,
		candidate.ID, candidate.TenantID, candidate.VersionID, candidate.ProjectID, component,
		candidate.Key, string(candidate.Status), candidate.Attempts, candidate.LogPath,
		candidate.Message, candidate.TreePath, candidate.Bytes,
	).StructScan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return r.GetByVersion(ctx, candidate.TenantID, candidate.VersionID)
	}
	if err != nil {
		return nil, fmt.Errorf("no se pudo encolar la preparación: %w", err)
	}
	return &stored, nil
}

// ClaimNext toma el pendiente más antiguo y lo marca en curso dentro de la misma
// sentencia. SKIP LOCKED permite que un segundo nodo tome el siguiente en vez de
// esperar por el mismo.
func (r *PostgresDependencyPreparationJobRepository) ClaimNext(
	ctx context.Context,
) (*domain.DependencyPreparationJob, error) {
	query := `
		WITH next AS (
			SELECT id AS job_id FROM dependency_preparation_jobs
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE dependency_preparation_jobs
		SET status = 'running', updated_at = NOW()
		FROM next
		WHERE dependency_preparation_jobs.id = next.job_id
		RETURNING ` + dependencyPreparationJobColumns

	var claimed domain.DependencyPreparationJob
	if err := r.db.GetContext(ctx, &claimed, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("no se pudo reclamar un trabajo de preparación: %w", err)
	}
	return &claimed, nil
}

// UpdateResult guarda el desenlace del worker. La clave solo se escribe si viene
// informada, y los intentos nunca bajan: un reintento manual no borra el
// historial de lo que ya se intentó.
func (r *PostgresDependencyPreparationJobRepository) UpdateResult(
	ctx context.Context,
	id string,
	status domain.DependencyPreparationJobStatus,
	result domain.DependencyPreparationResult,
) error {
	query := `
		UPDATE dependency_preparation_jobs
		SET status = $2,
		    preparation_key = CASE WHEN $3 = '' THEN preparation_key ELSE $3 END,
		    log_path = $4,
		    message = $5,
		    tree_path = $6,
		    bytes = $7,
		    attempts = CASE WHEN $8 > 0 THEN $8 ELSE attempts END,
		    updated_at = NOW()
		WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id, string(status), result.Key,
		result.LogPath, result.Message, result.Dir, result.Bytes, result.Attempts)
	if err != nil {
		return fmt.Errorf("no se pudo guardar el resultado de la preparación: %w", err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return domain.ErrDependencyPreparationJobNotFound
	}
	return nil
}

// ResetForRetry devuelve el trabajo a `pending` con la clave recalculada. El
// reintento es una ejecución nueva: el motivo, el log y el árbol anteriores se
// descartan, y los intentos vuelven a empezar. La condición de estado evita
// pisar un trabajo en curso o uno que ya está listo.
func (r *PostgresDependencyPreparationJobRepository) ResetForRetry(
	ctx context.Context,
	id, key string,
) error {
	query := `
		UPDATE dependency_preparation_jobs
		SET status = 'pending',
		    preparation_key = $2,
		    message = '',
		    log_path = '',
		    tree_path = '',
		    bytes = 0,
		    attempts = 0,
		    updated_at = NOW()
		WHERE id = $1 AND status IN ('failed', 'skipped', 'prepared')`

	res, err := r.db.ExecContext(ctx, query, id, key)
	if err != nil {
		return fmt.Errorf("no se pudo reprogramar la preparación: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrDependencyPreparationNotRetryable
	}
	return nil
}

func (r *PostgresDependencyPreparationJobRepository) GetByVersion(
	ctx context.Context,
	tenantID, versionID string,
) (*domain.DependencyPreparationJob, error) {
	query := `SELECT ` + dependencyPreparationJobColumns + `
		FROM dependency_preparation_jobs
		WHERE tenant_id = $1 AND version_id = $2`
	var job domain.DependencyPreparationJob
	if err := r.db.GetContext(ctx, &job, query, tenantID, versionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDependencyPreparationJobNotFound
		}
		return nil, fmt.Errorf("no se pudo leer el trabajo de preparación: %w", err)
	}
	return &job, nil
}

// ListActiveKeys devuelve las claves que alguna versión preparada sigue
// usando. Es la lista que exige la poda: sin ella no se borra nada.
func (r *PostgresDependencyPreparationJobRepository) ListActiveKeys(
	ctx context.Context,
) (map[string]bool, error) {
	query := `
		SELECT DISTINCT preparation_key
		FROM dependency_preparation_jobs
		WHERE status = 'prepared' AND preparation_key <> ''`
	keys := make([]string, 0)
	if err := r.db.SelectContext(ctx, &keys, query); err != nil {
		return nil, fmt.Errorf("no se pudieron listar las claves preparadas: %w", err)
	}
	active := make(map[string]bool, len(keys))
	for _, key := range keys {
		active[key] = true
	}
	return active, nil
}

// ListVersionsWithoutJob encuentra versiones publicadas de las que nadie encoló
// nada. Cubre la publicación que se guardó pero cuyo encolado no llegó a
// persistirse.
func (r *PostgresDependencyPreparationJobRepository) ListVersionsWithoutJob(
	ctx context.Context,
	limit int,
) ([]domain.DependencyPreparationJobCandidate, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT p.tenant_id, v.id AS version_id, v.project_id, v.exercise_id
		FROM workspace_projects p
		JOIN workspace_project_versions v ON v.id = p.current_published_version_id
		LEFT JOIN dependency_preparation_jobs j ON j.version_id = v.id
		WHERE j.id IS NULL
		ORDER BY v.published_at
		LIMIT $1`
	candidates := make([]domain.DependencyPreparationJobCandidate, 0)
	if err := r.db.SelectContext(ctx, &candidates, query, limit); err != nil {
		return nil, fmt.Errorf("no se pudieron listar las versiones sin preparación: %w", err)
	}
	return candidates, nil
}

// ReclaimStale devuelve a `pending` los trabajos que llevan demasiado tiempo en
// curso. El margen lo decide quien llama: un nodo caído no avisa, así que el
// único testigo es el tiempo.
func (r *PostgresDependencyPreparationJobRepository) ReclaimStale(
	ctx context.Context,
	olderThan time.Duration,
) (int, error) {
	if olderThan <= 0 {
		return 0, fmt.Errorf("%w: el vencimiento debe ser positivo", domain.ErrInvalidDependencyPreparationJob)
	}
	query := `
		UPDATE dependency_preparation_jobs
		SET status = 'pending',
		    message = 'reintento: el nodo no terminó la preparación',
		    updated_at = NOW()
		WHERE status = 'running'
		  AND updated_at < NOW() - make_interval(secs => $1)`
	res, err := r.db.ExecContext(ctx, query, olderThan.Seconds())
	if err != nil {
		return 0, fmt.Errorf("no se pudieron recuperar los trabajos vencidos: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return int(affected), nil
}

// PostgresDependencyPreparationTargetResolver lee del catálogo el entorno del
// ejercicio: la plantilla de laboratorio apunta a la tecnología de ejecución y
// esa tecnología declara imagen y perfil. El perfil es dato, no código.
type PostgresDependencyPreparationTargetResolver struct {
	db *sqlx.DB
}

func NewPostgresDependencyPreparationTargetResolver(db *sqlx.DB) *PostgresDependencyPreparationTargetResolver {
	return &PostgresDependencyPreparationTargetResolver{db: db}
}

func (r *PostgresDependencyPreparationTargetResolver) ResolveDependencyPreparationTarget(
	ctx context.Context,
	tenantID, exerciseID string,
) (*domain.DependencyPreparationTarget, error) {
	query := `
		SELECT tc.id::text AS component_id,
		       tc.slug,
		       -- La preparación corre en la imagen que arma el laboratorio, no en
		       -- la base del runtime: una plantilla compuesta trae los complementos
		       -- (Maven, Gradle, toolchains) y preparar sin ellos falla por una
		       -- herramienta que el alumno sí tiene. La base queda como respaldo.
		       COALESCE(NULLIF(lt.docker_image, ''), NULLIF(tc.base_image, ''), '') AS base_image,
		       COALESCE(tc.dependency_profile, '{}'::jsonb) AS dependency_profile
		FROM exercises e
		JOIN lab_templates lt ON lt.id = e.template_id AND lt.tenant_id = e.tenant_id
		JOIN template_components tc
		  ON (tc.tenant_id = lt.tenant_id OR tc.tenant_id IS NULL)
		 AND (tc.slug = lt.stack_config->>'runtime_id' OR tc.id::text = lt.stack_config->>'runtime_id')
		WHERE e.tenant_id = $1 AND e.id = $2
		ORDER BY (tc.tenant_id IS NOT DISTINCT FROM lt.tenant_id) DESC
		LIMIT 1`

	var row struct {
		ComponentID string `db:"component_id"`
		Slug        string `db:"slug"`
		BaseImage   string `db:"base_image"`
		Profile     []byte `db:"dependency_profile"`
	}
	if err := r.db.GetContext(ctx, &row, query, tenantID, exerciseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("no se pudo resolver el entorno del proyecto: %w", err)
	}
	profile := domain.DependencyProfile{}
	if len(row.Profile) > 0 {
		if err := json.Unmarshal(row.Profile, &profile); err != nil {
			return nil, fmt.Errorf("el perfil de dependencias del catálogo no es legible: %w", err)
		}
	}

	target := &domain.DependencyPreparationTarget{
		ComponentID:   row.ComponentID,
		ComponentSlug: row.Slug,
		BaseImage:     row.BaseImage,
	}
	if !profile.IsEmpty() {
		target.Profile = &profile
		target.User = profile.User
		target.Home = profile.Home
	}
	if strings.TrimSpace(target.User) == "" {
		// Decisión 1: el efectivo del laboratorio es 1000:1000. El perfil puede
		// declarar otro; el árbol se crea con ese dueño.
		target.User = "1000:1000"
	}
	return target, nil
}
