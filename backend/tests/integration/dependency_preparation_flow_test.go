package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/docker"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// TestDependencyPreparationFlow ejercita la cadena completa contra Postgres y
// Docker reales: la cola persiste el trabajo, el worker prepara la versión con el
// runner de verdad y el árbol resultante se enlaza a un volumen. Los tests
// unitarios prueban cada pieza por separado; este prueba que encajan.
func TestDependencyPreparationFlow(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	seed := seedWorkspaceTestBase(t, db.GetDB())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	user := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	slug := "python-integracion-" + uuid.NewString()[:8]
	profile := fmt.Sprintf(`{"version":1,"detect":["requirements.txt"],"lockfiles":["requirements.txt"],`+
		`"install":"python -m venv .venv && .venv/bin/pip install --no-cache-dir -r requirements.txt",`+
		`"materialize":"tree","dependency_paths":[".venv"],`+
		`"verify":".venv/bin/python -c 'import six'","network":"build","user":%q}`, user)

	componentID := uuid.NewString()
	if _, err := db.GetDB().Exec(`
		INSERT INTO template_components (id, tenant_id, slug, layer, sub_category, name, base_image, dependency_profile)
		VALUES ($1, $2, $3, 'runtime', 'backend', 'Python integración', 'python:3.12-slim-bookworm', $4::jsonb)
	`, componentID, seed.tenantID, slug, profile); err != nil {
		t.Fatalf("no se pudo crear la tecnología de ejecución: %v", err)
	}

	var templateID string
	if err := db.GetDB().Get(&templateID, `SELECT id FROM lab_templates WHERE tenant_id = $1 LIMIT 1`, seed.tenantID); err != nil {
		t.Fatalf("no se pudo leer la plantilla del seed: %v", err)
	}
	if _, err := db.GetDB().Exec(`UPDATE exercises SET template_id = $2 WHERE id = $1`, seed.exerciseA, templateID); err != nil {
		t.Fatalf("no se pudo asociar la plantilla al ejercicio: %v", err)
	}
	updated, err := db.GetDB().Exec(`
		UPDATE lab_templates
		SET stack_config = jsonb_build_object('runtime_id', $2::text)
		WHERE id = $1
	`, templateID, slug)
	if err != nil {
		t.Fatalf("no se pudo apuntar la plantilla a la tecnología: %v", err)
	}
	if affected, _ := updated.RowsAffected(); affected != 1 {
		t.Fatalf("la plantilla debía quedar apuntando a la tecnología, filas afectadas: %d", affected)
	}

	snapshot := domain.EncodeProjectSnapshot([]domain.ProjectFile{
		{Path: "requirements.txt", Content: "six==1.16.0\n"},
	})
	sum := sha256.Sum256([]byte(snapshot))
	projectID := uuid.NewString()
	versionID := uuid.NewString()
	if _, err := db.GetDB().Exec(`
		INSERT INTO workspace_projects (id, tenant_id, exercise_id)
		VALUES ($1, $2, $3)
	`, projectID, seed.tenantID, seed.exerciseA); err != nil {
		t.Fatalf("no se pudo crear el proyecto: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO workspace_project_versions
			(id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes)
		VALUES ($1, $2, $3, $4, 1, $5, '[]'::jsonb, $6, 1, $7)
	`, versionID, seed.tenantID, projectID, seed.exerciseA, snapshot, hex.EncodeToString(sum[:]), len(snapshot)); err != nil {
		t.Fatalf("no se pudo publicar la versión: %v", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	worker, err := services.NewDependencyPreparationWorker(
		docker.NewDependencyPreparationDockerRunner(cli),
		t.TempDir(),
		// El techo real es de 10 GB libres; la prueba solo verifica el flujo.
		domain.DependencyPreparationLimits{RetryBackoff: time.Second, MinFreeBytes: 1},
	)
	if err != nil {
		t.Fatalf("no se pudo crear el worker: %v", err)
	}
	resolver := postgres.NewPostgresDependencyPreparationTargetResolver(db.GetDB())
	target, err := resolver.ResolveDependencyPreparationTarget(ctx, seed.tenantID, seed.exerciseA)
	if err != nil {
		t.Fatalf("el resolvedor no debía fallar: %v", err)
	}
	if target == nil || target.Profile == nil {
		var diagExerciseTemplate, diagStack, diagSlug, diagTemplateTenant string
		var diagComponents, diagByTenant int
		_ = db.GetDB().Get(&diagExerciseTemplate, `SELECT COALESCE(template_id::text, '') FROM exercises WHERE id = $1`, seed.exerciseA)
		_ = db.GetDB().Get(&diagStack, `SELECT COALESCE(stack_config->>'runtime_id', '') FROM lab_templates WHERE id = $1`, templateID)
		_ = db.GetDB().Get(&diagTemplateTenant, `SELECT COALESCE(tenant_id::text, '') FROM lab_templates WHERE id = $1`, templateID)
		_ = db.GetDB().Get(&diagSlug, `SELECT COALESCE(slug, '') FROM template_components WHERE id = $1`, componentID)
		_ = db.GetDB().Get(&diagComponents, `SELECT COUNT(*) FROM template_components WHERE id = $1`, componentID)
		_ = db.GetDB().Get(&diagByTenant, `SELECT COUNT(*) FROM template_components WHERE tenant_id = $1`, seed.tenantID)
		var join1, join2 int
		var exerciseTenant string
		_ = db.GetDB().Get(&exerciseTenant, `SELECT tenant_id::text FROM exercises WHERE id = $1`, seed.exerciseA)
		_ = db.GetDB().Get(&join1, `
			SELECT COUNT(*) FROM exercises e
			JOIN lab_templates lt ON lt.id = e.template_id AND lt.tenant_id = e.tenant_id
			WHERE e.id = $1 AND e.tenant_id = $2`, seed.exerciseA, seed.tenantID)
		_ = db.GetDB().Get(&join2, `
			SELECT COUNT(*) FROM exercises e
			JOIN lab_templates lt ON lt.id = e.template_id AND lt.tenant_id = e.tenant_id
			JOIN template_components tc
			  ON (tc.tenant_id = lt.tenant_id OR tc.tenant_id IS NULL)
			 AND tc.slug = lt.stack_config->>'runtime_id'
			WHERE e.id = $1 AND e.tenant_id = $2`, seed.exerciseA, seed.tenantID)
		t.Fatalf("sin perfil: ejercicio.template_id=%q plantilla=%q runtime_id=%q plantilla.tenant=%q ejercicio.tenant=%q seed.tenant=%q componente.slug=%q componente=%d por_tenant=%d join1=%d join2=%d",
			diagExerciseTemplate, templateID, diagStack, diagTemplateTenant, exerciseTenant, seed.tenantID, diagSlug, diagComponents, diagByTenant, join1, join2)
	}

	// El seed deja la plantilla apuntando a 'solv/ide:test' y la pieza declara
	// la base del runtime. La preparación tiene que correr en la imagen de la
	// plantilla: es la que trae los complementos que el alumno usa.
	if target.BaseImage != "solv/ide:test" {
		t.Fatalf("la preparación debía usar la imagen de la plantilla, obtuvo %q", target.BaseImage)
	}
	// Para el resto del flujo hace falta una imagen que exista en la máquina.
	if _, err := db.GetDB().Exec(`UPDATE lab_templates SET docker_image = $2 WHERE id = $1`,
		templateID, "python:3.12-slim-bookworm"); err != nil {
		t.Fatalf("no se pudo fijar la imagen real de la prueba: %v", err)
	}

	queue, err := services.NewDependencyPreparationQueue(
		postgres.NewPostgresDependencyPreparationJobRepository(db.GetDB()),
		resolver,
		postgres.NewPostgresWorkspaceProjectRepository(db.GetDB()),
		worker,
	)
	if err != nil {
		t.Fatalf("no se pudo crear la cola: %v", err)
	}

	job, err := queue.Enqueue(ctx, seed.tenantID, seed.exerciseA, versionID)
	if err != nil {
		t.Fatalf("el encolado no debía fallar: %v", err)
	}
	if job.Status != domain.DependencyPreparationJobPending || len(job.Key) != 64 {
		t.Fatalf("el trabajo debía quedar pendiente con clave: %+v", job)
	}

	job, processed, err := queue.ProcessNext(ctx)
	if err != nil {
		t.Fatalf("la preparación no debía fallar: %v", err)
	}
	if !processed || job == nil || job.Status != domain.DependencyPreparationJobPrepared {
		t.Fatalf("la versión debía quedar preparada: %+v", job)
	}
	if _, err := os.Stat(job.TreePath); err != nil {
		t.Fatalf("el árbol preparado debía existir: %v", err)
	}

	tree, found, err := queue.PreparedDependencies(ctx, seed.tenantID, versionID)
	if err != nil || !found || tree == nil {
		t.Fatalf("la versión preparada debía informar su árbol: found=%v err=%v", found, err)
	}
	if len(tree.Paths) != 1 || tree.Paths[0] != ".venv" {
		t.Fatalf("las rutas debían salir del perfil, obtuvo %v", tree.Paths)
	}

	volumeClient, err := docker.NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de volúmenes: %v", err)
	}
	volume := "solv-flow-" + strings.ToLower(uuid.NewString()[:8])
	if err := volumeClient.EnsureVolumeExists(ctx, volume); err != nil {
		t.Fatalf("no se pudo preparar el volumen: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "volume", "rm", "-f", volume).Run()
	})
	// El enlace exige un volumen ya inicializado: es la misma condición que deja
	// la siembra antes de escribir el proyecto.
	if err := volumeClient.WritePracticeEnvironmentVolumeFiles(ctx, volume, map[string]string{
		".solv_workspace/init.json": `{"version_id":"` + versionID + `","content_sha256":"x","state":"verified"}`,
	}); err != nil {
		t.Fatalf("no se pudo sembrar el diario del volumen: %v", err)
	}
	if err := volumeClient.LinkWorkspaceDependencies(ctx, volume, *tree); err != nil {
		t.Fatalf("el enlace al volumen debía completarse: %v", err)
	}

	// El árbol desapareció del disco —poda o mudanza del nodo—: el estado sigue
	// diciendo preparado y el enlace ya no tendría de dónde copiar. El reintento
	// tiene que aceptarse y volver a la cola conservando la clave, porque el
	// perfil no cambió.
	if err := os.RemoveAll(tree.RootPath); err != nil {
		t.Fatalf("no se pudo borrar el árbol: %v", err)
	}
	requeued, err := queue.Retry(ctx, seed.tenantID, seed.exerciseA, versionID)
	if err != nil {
		t.Fatalf("sin árbol el reintento debía aceptarse: %v", err)
	}
	if requeued.Status != domain.DependencyPreparationJobPending {
		t.Fatalf("el reintento debía dejar el trabajo pendiente, obtuvo %s", requeued.Status)
	}
	if requeued.Key != job.Key {
		t.Fatalf("el perfil no cambió: la clave debía conservarse, %q y %q", job.Key, requeued.Key)
	}
}
