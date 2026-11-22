package integration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/postgres"
)

const (
	workspaceArtifactMigrationVersion int64 = 67
	workspaceArtifactMigrationsDir          = "../../migrations"
)

// TestWorkspaceProjectArtifactPublish publica un borrador con artefacto y
// comprueba la versión, el puntero del proyecto, el espejo del ejercicio y que
// no se encole una preparación que el worker todavía no sabe hacer.
func TestWorkspaceProjectArtifactPublish(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	handle := db.GetDB()
	seed := seedWorkspaceTestBase(t, handle)
	ctx := context.Background()
	repo := postgres.NewPostgresWorkspaceProjectRepository(handle)

	project, err := repo.CreateProject(ctx, &domain.WorkspaceProject{
		TenantID:   seed.tenantID,
		ExerciseID: seed.exerciseA,
		CreatedBy:  &seed.teacherID,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}

	archive := buildPublishArtifact(t, map[string]string{
		"main.py":            "print('hola')\n",
		"src/calculadora.py": "def sumar(a, b):\n    return a + b\n",
	})
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	key, err := domain.WorkspaceProjectArtifactKey(seed.tenantID, seed.exerciseA, hash)
	if err != nil {
		t.Fatalf("clave del artefacto: %v", err)
	}
	artifactBytes := int64(len(archive))
	artifactFiles := 2
	if _, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:       seed.tenantID,
		ProjectID:      project.ID,
		ContentSHA256:  hash,
		ArtifactKey:    &key,
		ArtifactSHA256: &hash,
		ArtifactBytes:  &artifactBytes,
		ArtifactFiles:  &artifactFiles,
	}, 0); err != nil {
		t.Fatalf("guardar borrador con artefacto: %v", err)
	}

	store := &testArtifactStore{objects: map[string][]byte{key: archive}}
	service := services.NewWorkspaceProjectService(repo, true)
	service.SetArtifactStore(store)
	version, err := service.Publish(ctx, seed.tenantID, seed.exerciseA, seed.teacherID)
	if err != nil {
		t.Fatalf("publicar el artefacto: %v", err)
	}
	if version.TextSnapshot != "" || version.ContentSHA256 != hash || version.FileCount != artifactFiles || version.TotalBytes != int(artifactBytes) {
		t.Fatalf("versión publicada inesperada: %+v", version)
	}
	var manifest []domain.WorkspaceProjectManifestEntry
	if err := json.Unmarshal(version.Manifest, &manifest); err != nil {
		t.Fatalf("manifiesto ilegible: %s (%v)", version.Manifest, err)
	}
	if len(manifest) != 2 || manifest[0].Path != "main.py" || manifest[1].Path != "src/calculadora.py" {
		t.Fatalf("manifiesto por archivo inesperado: %+v", manifest)
	}
	if manifest[1].Bytes != len("def sumar(a, b):\n    return a + b\n") || manifest[1].Lines != 2 {
		t.Fatalf("entrada de manifiesto inesperada: %+v", manifest[1])
	}
	published := version.Artifact()
	if published == nil || published.Key != key || published.SHA256 != hash || published.Files != artifactFiles || published.Bytes != artifactBytes {
		t.Fatalf("la versión no conserva el artefacto: %+v", version)
	}

	var pointer *string
	if err := handle.Get(&pointer, `SELECT current_published_version_id FROM workspace_projects WHERE id = $1`, project.ID); err != nil {
		t.Fatalf("leer el puntero del proyecto: %v", err)
	}
	if pointer == nil || *pointer != version.ID {
		t.Fatalf("el proyecto no apunta a la versión publicada: %v", pointer)
	}

	var base *string
	if err := handle.Get(&base, `SELECT base_version_id FROM workspace_project_drafts WHERE project_id = $1`, project.ID); err != nil {
		t.Fatalf("leer la base del borrador: %v", err)
	}
	if base == nil || *base != version.ID {
		t.Fatalf("el borrador no quedó sobre la versión publicada: %v", base)
	}

	// El espejo del ejercicio no puede guardar el árbol: queda vacío y atado a la
	// versión nueva, que es lo que el trigger de 00053 exige.
	var boilerplate string
	var mirrorSHA string
	var sourceVersion *string
	if err := handle.QueryRow(`
		SELECT COALESCE(boilerplate, ''), COALESCE(boilerplate_sha256, ''), boilerplate_source_version_id
		FROM exercises WHERE id = $1
	`, seed.exerciseA).Scan(&boilerplate, &mirrorSHA, &sourceVersion); err != nil {
		t.Fatalf("leer el espejo del ejercicio: %v", err)
	}
	if boilerplate != "" || mirrorSHA != hashTestProject("") || sourceVersion == nil || *sourceVersion != version.ID {
		t.Fatalf("espejo inesperado: boilerplate=%q sha=%s source=%v", boilerplate, mirrorSHA, sourceVersion)
	}

	var jobs int
	if err := handle.Get(&jobs, `SELECT COUNT(*) FROM dependency_preparation_jobs WHERE version_id = $1`, version.ID); err != nil {
		t.Fatalf("contar trabajos de preparación: %v", err)
	}
	if jobs != 0 {
		t.Fatalf("se encoló preparación para un artefacto: %d trabajos", jobs)
	}

	// La siembra recibe el artefacto desde el mismo camino que lee las versiones.
	loaded, err := repo.GetVersion(ctx, seed.tenantID, version.ID)
	if err != nil {
		t.Fatalf("releer la versión publicada: %v", err)
	}
	if loaded.Artifact() == nil || loaded.Artifact().Key != key {
		t.Fatalf("la versión releída perdió el artefacto: %+v", loaded)
	}

	// La revisión del docente arma el diff con el árbol del depósito, porque el
	// espejo de texto quedó vacío a propósito.
	baseFiles, baseSnapshot, ok, err := service.ProjectBaseFiles(ctx, seed.tenantID, seed.exerciseA)
	if err != nil {
		t.Fatalf("resolver el proyecto base: %v", err)
	}
	if !ok || len(baseFiles) != 2 {
		t.Fatalf("proyecto base: ok=%v archivos=%d", ok, len(baseFiles))
	}
	if !strings.Contains(baseSnapshot, "# === FILE: main.py ===") || !strings.Contains(baseSnapshot, "return a + b") {
		t.Fatalf("snapshot base inesperado: %q", baseSnapshot)
	}
}

type testArtifactStore struct {
	objects map[string][]byte
}

func (s *testArtifactStore) Put(_ context.Context, key string, body io.Reader) (string, int64, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", 0, err
	}
	if s.objects == nil {
		s.objects = make(map[string][]byte)
	}
	s.objects[key] = data
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), int64(len(data)), nil
}

func (s *testArtifactStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, domain.ErrWorkspaceProjectArchiveInvalid
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func buildPublishArtifact(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for name, content := range files {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatalf("cabecera %s: %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("contenido %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("cerrar gzip: %v", err)
	}
	return buffer.Bytes()
}

// TestWorkspaceProjectArtifactDraftRoundTrip ejerce la migración 00067 y el SQL
// nuevo del repositorio sobre una base descartable: guardar y leer un borrador
// con artefacto, rechazar una fila a medio camino, limpiar el artefacto al
// volver a texto y revertir la migración sin dejar restos.
func TestWorkspaceProjectArtifactDraftRoundTrip(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	handle := db.GetDB()
	seed := seedWorkspaceTestBase(t, handle)
	ctx := context.Background()
	repo := postgres.NewPostgresWorkspaceProjectRepository(handle)

	var applied int64
	if err := handle.Get(&applied, `SELECT MAX(version_id) FROM goose_db_version WHERE is_applied`); err != nil {
		t.Fatalf("leer la versión de goose: %v", err)
	}
	if applied < workspaceArtifactMigrationVersion {
		t.Fatalf("goose aplicó hasta %d, se esperaba al menos %d", applied, workspaceArtifactMigrationVersion)
	}

	project, err := repo.CreateProject(ctx, &domain.WorkspaceProject{
		TenantID:   seed.tenantID,
		ExerciseID: seed.exerciseA,
		CreatedBy:  &seed.teacherID,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}

	archive := []byte("tar.gz de prueba con contenido real")
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	key, err := domain.WorkspaceProjectArtifactKey(seed.tenantID, seed.exerciseA, hash)
	if err != nil {
		t.Fatalf("clave del artefacto: %v", err)
	}
	bytes := int64(len(archive))
	files := 3

	saved, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:       seed.tenantID,
		ProjectID:      project.ID,
		TextSnapshot:   "",
		ContentSHA256:  hash,
		UpdatedBy:      &seed.teacherID,
		ArtifactKey:    &key,
		ArtifactSHA256: &hash,
		ArtifactBytes:  &bytes,
		ArtifactFiles:  &files,
	}, 0)
	if err != nil {
		t.Fatalf("guardar borrador con artefacto: %v", err)
	}
	if saved.Revision != 1 || saved.TextSnapshot != "" || saved.ContentSHA256 != hash {
		t.Fatalf("borrador guardado inesperado: %+v", saved)
	}
	artifact := saved.Artifact()
	if artifact == nil || artifact.Key != key || artifact.SHA256 != hash || artifact.Bytes != bytes || artifact.Files != files {
		t.Fatalf("el borrador no conserva el artefacto: %+v", saved)
	}

	read, err := repo.GetDraftByProject(ctx, seed.tenantID, project.ID)
	if err != nil {
		t.Fatalf("leer borrador: %v", err)
	}
	readArtifact := read.Artifact()
	if readArtifact == nil || *readArtifact != *artifact {
		t.Fatalf("lectura distinta de lo guardado: guardado=%+v leído=%+v", artifact, readArtifact)
	}

	// El CHECK por tabla exige los cuatro campos juntos.
	if _, err := handle.Exec(`
		INSERT INTO workspace_project_drafts (id, tenant_id, project_id, text_snapshot, content_sha256, revision, artifact_key)
		VALUES ($1, $2, $3, '', '', 1, 'incompleto')
	`, uuid.NewString(), seed.tenantID, project.ID); err == nil {
		t.Fatal("el CHECK debía rechazar un borrador con artefacto parcial")
	}
	if _, err := handle.Exec(`
		INSERT INTO workspace_project_versions
			(id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes, artifact_key)
		VALUES ($1, $2, $3, $4, 1, '', '[]'::jsonb, $5, 0, 0, 'incompleto')
	`, uuid.NewString(), seed.tenantID, project.ID, seed.exerciseA, hash); err == nil {
		t.Fatal("el CHECK debía rechazar una versión con artefacto parcial")
	}

	// Volver a texto limpia el artefacto del borrador.
	text := "# === FILE: main.go ===\npackage main\n"
	updated, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:     seed.tenantID,
		ProjectID:    project.ID,
		TextSnapshot: text,
		UpdatedBy:    &seed.teacherID,
	}, saved.Revision)
	if err != nil {
		t.Fatalf("volver el borrador a texto: %v", err)
	}
	if updated.Revision != 2 || updated.Artifact() != nil || updated.ContentSHA256 != hashTestProject(text) {
		t.Fatalf("el borrador de texto conserva rastros del artefacto: %+v", updated)
	}

	// La versión con artefacto se lee completa por el mismo camino de la siembra.
	versionID := uuid.NewString()
	if _, err := handle.Exec(`
		INSERT INTO workspace_project_versions
			(id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes,
			 artifact_key, artifact_sha256, artifact_bytes, artifact_files)
		VALUES ($1, $2, $3, $4, 1, '', '[]'::jsonb, $5, $6, $7, $8, $9, $10, $11)
	`, versionID, seed.tenantID, project.ID, seed.exerciseA, hash, files, bytes, key, hash, bytes, files); err != nil {
		t.Fatalf("insertar versión con artefacto: %v", err)
	}
	version, err := repo.GetVersion(ctx, seed.tenantID, versionID)
	if err != nil {
		t.Fatalf("leer versión: %v", err)
	}
	versionArtifact := version.Artifact()
	if versionArtifact == nil || versionArtifact.Key != key || versionArtifact.Files != files || versionArtifact.Bytes != bytes {
		t.Fatalf("la versión no conserva el artefacto: %+v", version)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialecto de goose: %v", err)
	}
	if err := goose.DownTo(handle.DB, workspaceArtifactMigrationsDir, workspaceArtifactMigrationVersion-1); err != nil {
		t.Fatalf("revertir 00067: %v", err)
	}
	for _, table := range []string{"workspace_project_drafts", "workspace_project_versions"} {
		var columns int
		if err := handle.Get(&columns, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1 AND column_name LIKE 'artifact_%'
		`, table); err != nil {
			t.Fatalf("contar columnas de %s: %v", table, err)
		}
		if columns != 0 {
			t.Fatalf("la reversión dejó %d columnas de artefacto en %s", columns, table)
		}
		var constraints int
		if err := handle.Get(&constraints, `
			SELECT COUNT(*) FROM pg_constraint WHERE conname = $1
		`, table+"_artifact_all_or_none"); err != nil {
			t.Fatalf("contar constraints de %s: %v", table, err)
		}
		if constraints != 0 {
			t.Fatalf("la reversión dejó el CHECK de %s", table)
		}
	}

	// La reversión conserva el borrador de texto y elimina las versiones con
	// artefacto, que no se pueden representar con el esquema anterior.
	var keptText string
	if err := handle.Get(&keptText, `SELECT text_snapshot FROM workspace_project_drafts WHERE project_id = $1`, project.ID); err != nil {
		t.Fatalf("leer el borrador tras la reversión: %v", err)
	}
	if keptText != text {
		t.Fatalf("la reversión perdió el borrador de texto: %q", keptText)
	}
	var remaining int
	if err := handle.Get(&remaining, `SELECT COUNT(*) FROM workspace_project_versions WHERE id = $1`, versionID); err != nil {
		t.Fatalf("contar la versión tras la reversión: %v", err)
	}
	if remaining != 0 {
		t.Fatal("la reversión dejó la versión con artefacto, que el esquema anterior rechaza")
	}

	if err := goose.Up(handle.DB, workspaceArtifactMigrationsDir); err != nil {
		t.Fatalf("reaplicar 00067: %v", err)
	}
	if err := handle.Get(&applied, `SELECT MAX(version_id) FROM goose_db_version WHERE is_applied`); err != nil || applied < workspaceArtifactMigrationVersion {
		t.Fatalf("tras reaplicar, goose en %d (%v)", applied, err)
	}
	var currentRevision int64
	if err := handle.Get(&currentRevision, `SELECT revision FROM workspace_project_drafts WHERE project_id = $1`, project.ID); err != nil {
		t.Fatalf("leer la revisión del borrador: %v", err)
	}
	if _, err := repo.SaveDraft(ctx, &domain.WorkspaceProjectDraft{
		TenantID:       seed.tenantID,
		ProjectID:      project.ID,
		ContentSHA256:  hash,
		ArtifactKey:    &key,
		ArtifactSHA256: &hash,
		ArtifactBytes:  &bytes,
		ArtifactFiles:  &files,
	}, currentRevision); err != nil {
		t.Fatalf("guardar artefacto tras reaplicar: %v", err)
	}
}
