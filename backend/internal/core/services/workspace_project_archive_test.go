package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"solv-backend/internal/core/domain"
)

type workspaceArtifactStoreFake struct {
	objects map[string][]byte
	puts    int
}

func (s *workspaceArtifactStoreFake) Put(_ context.Context, key string, body io.Reader) (string, int64, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", 0, err
	}
	if s.objects == nil {
		s.objects = make(map[string][]byte)
	}
	s.objects[key] = data
	s.puts++
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), int64(len(data)), nil
}

func (s *workspaceArtifactStoreFake) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("objeto ausente")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

type workspaceArchiveEntry struct {
	name     string
	body     string
	typeflag byte
}

func buildWorkspaceArchive(t *testing.T, entries []workspaceArchiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o644,
			Size:     int64(len(entry.body)),
			Typeflag: typeflag,
		}
		if typeflag == tar.TypeLink || typeflag == tar.TypeSymlink {
			header.Linkname = "otro/archivo"
			header.Size = 0
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("write header %s: %v", entry.name, err)
		}
		if header.Size > 0 {
			if _, err := writer.Write([]byte(entry.body)); err != nil {
				t.Fatalf("write body %s: %v", entry.name, err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buffer.Bytes()
}

func TestWorkspaceImportArchiveStoresArtifactDraft(t *testing.T) {
	repo := &workspaceProjectRepoFake{}
	store := &workspaceArtifactStoreFake{}
	service := NewWorkspaceProjectService(repo, false)
	service.SetArtifactStore(store)
	service.SetArchiveEnabled(true)

	archive := buildWorkspaceArchive(t, []workspaceArchiveEntry{
		{name: "proyecto/main.go", body: "package main\n\nfunc main() {}\n"},
		{name: "proyecto/internal/app.go", body: "package internal\n"},
	})
	result, err := service.ImportArchive(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, archive)
	if err != nil {
		t.Fatalf("ImportArchive: %v", err)
	}
	if result.FileCount != 2 || result.TotalBytes == 0 {
		t.Fatalf("resumen inesperado: %+v", result)
	}
	if result.Draft == nil || result.Draft.TextSnapshot != "" {
		t.Fatalf("el borrador debía ser un artefacto: %+v", result.Draft)
	}
	artifact := result.Draft.Artifact()
	if artifact == nil {
		t.Fatalf("el borrador no conserva el artefacto: %+v", result.Draft)
	}
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	if artifact.SHA256 != hash || artifact.Bytes != int64(len(archive)) || artifact.Files != 2 {
		t.Fatalf("artefacto inesperado: %+v", artifact)
	}
	expectedKey, err := domain.WorkspaceProjectArtifactKey("tenant-1", workspaceImportExerciseID, hash)
	if err != nil {
		t.Fatalf("clave: %v", err)
	}
	if artifact.Key != expectedKey {
		t.Fatalf("clave %q, esperada %q", artifact.Key, expectedKey)
	}
	if store.puts != 1 || len(store.objects[artifact.Key]) != len(archive) {
		t.Fatalf("el depósito no recibió el archivo una vez: %+v", store.puts)
	}

	// La misma carga no duplica el objeto: la clave es la huella.
	if _, err := service.ImportArchive(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 1, archive); err != nil {
		t.Fatalf("segunda carga: %v", err)
	}
	if len(store.objects) != 1 {
		t.Fatalf("el depósito guardó objetos duplicados: %d", len(store.objects))
	}
}

func TestWorkspaceImportArchiveRejectsUnsafeEntries(t *testing.T) {
	cases := []struct {
		name    string
		entries []workspaceArchiveEntry
	}{
		{"traversal", []workspaceArchiveEntry{{name: "../secreto.txt", body: "x\n"}}},
		{"absoluta", []workspaceArchiveEntry{{name: "/etc/passwd", body: "x\n"}}},
		{"sin normalizar", []workspaceArchiveEntry{{name: "src//main.go", body: "x\n"}}},
		{"symlink", []workspaceArchiveEntry{{name: "enlace", body: "", typeflag: tar.TypeSymlink}}},
		{"hardlink", []workspaceArchiveEntry{{name: "duro", body: "", typeflag: tar.TypeLink}}},
		{"directorio de metadatos", []workspaceArchiveEntry{{name: ".solv_workspace/init.json", body: "{}"}}},
		{"duplicada", []workspaceArchiveEntry{{name: "main.go", body: "a\n"}, {name: "main.go", body: "b\n"}}},
		{"colisión archivo directorio", []workspaceArchiveEntry{{name: "src", body: "x\n"}, {name: "src/main.go", body: "y\n"}}},
		{"vacío", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &workspaceProjectRepoFake{}
			store := &workspaceArtifactStoreFake{}
			service := NewWorkspaceProjectService(repo, false)
			service.SetArtifactStore(store)
			service.SetArchiveEnabled(true)

			archive := buildWorkspaceArchive(t, tc.entries)
			if _, err := service.ImportArchive(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, archive); err == nil {
				t.Fatal("ImportArchive aceptó un archivo inválido")
			}
			if repo.saveCalls != 0 || store.puts != 0 {
				t.Fatalf("un archivo inválido tocó el borrador o el depósito: save=%d puts=%d", repo.saveCalls, store.puts)
			}
		})
	}

	unbacked := NewWorkspaceProjectService(&workspaceProjectRepoFake{}, false)
	unbacked.SetArchiveEnabled(true)
	if _, err := unbacked.ImportArchive(
		context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0,
		buildWorkspaceArchive(t, []workspaceArchiveEntry{{name: "main.go", body: "package main\n"}}),
	); !errors.Is(err, domain.ErrWorkspaceProjectArchiveStoreUnavailable) {
		t.Fatalf("sin depósito se esperaba store_unavailable, llegó %v", err)
	}
	withStore := NewWorkspaceProjectService(&workspaceProjectRepoFake{}, false)
	withStore.SetArchiveEnabled(true)
	withStore.SetArtifactStore(&workspaceArtifactStoreFake{})
	if _, err := withStore.ImportArchive(
		context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, []byte("no es gzip"),
	); !errors.Is(err, domain.ErrWorkspaceProjectArchiveInvalid) {
		t.Fatalf("un cuerpo que no es tar.gz debía rechazarse, llegó %v", err)
	}
}

func TestWorkspaceImportArchiveRequiresFlag(t *testing.T) {
	repo := &workspaceProjectRepoFake{}
	store := &workspaceArtifactStoreFake{}
	service := NewWorkspaceProjectService(repo, false)
	service.SetArtifactStore(store)
	archive := buildWorkspaceArchive(t, []workspaceArchiveEntry{{name: "main.go", body: "package main\n"}})

	if _, err := service.ImportArchive(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, archive); !errors.Is(err, domain.ErrWorkspaceProjectArchiveDisabled) {
		t.Fatalf("apagada se esperaba archive_disabled, llegó %v", err)
	}
	if repo.saveCalls != 0 || store.puts != 0 {
		t.Fatalf("la carga apagada tocó el borrador o el depósito: save=%d puts=%d", repo.saveCalls, store.puts)
	}

	service.SetArchiveEnabled(true)
	if _, err := service.ImportArchive(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, archive); err != nil {
		t.Fatalf("encendida la carga debe aceptar el archivo: %v", err)
	}
}

func TestWorkspacePublishAcceptsArtifactDraft(t *testing.T) {
	archive := buildWorkspaceArchive(t, []workspaceArchiveEntry{
		{name: "main.go", body: "package main\n"},
		{name: "internal/app.go", body: "package internal\n"},
	})
	sum := sha256.Sum256(archive)
	sha := hex.EncodeToString(sum[:])
	key := "workspace-projects/tenant-1/exercise/" + sha + ".tar.gz"
	artifactBytes := int64(len(archive))
	artifactFiles := 2
	repo := &workspaceProjectRepoFake{
		project: &domain.WorkspaceProject{ID: "project-1", TenantID: "tenant-1", ExerciseID: workspaceImportExerciseID},
		draft: &domain.WorkspaceProjectDraft{
			TenantID:       "tenant-1",
			ProjectID:      "project-1",
			ContentSHA256:  sha,
			ArtifactKey:    &key,
			ArtifactSHA256: &sha,
			ArtifactBytes:  &artifactBytes,
			ArtifactFiles:  &artifactFiles,
		},
	}
	service := NewWorkspaceProjectService(repo, true)
	service.SetArtifactStore(&workspaceArtifactStoreFake{objects: map[string][]byte{key: archive}})

	version, err := service.Publish(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1")
	if err != nil {
		t.Fatalf("publicar un borrador con artefacto: %v", err)
	}
	if repo.publishCalls != 1 || version == nil {
		t.Fatalf("publicación inesperada: calls=%d version=%v", repo.publishCalls, version)
	}

	// El manifiesto sale del archivo, no del texto: dos entradas ordenadas.
	manifest, err := service.baseManifest(context.Background(), &domain.WorkspaceProjectArtifact{Key: key, SHA256: sha, Bytes: artifactBytes, Files: artifactFiles})
	if err != nil {
		t.Fatalf("manifiesto del artefacto: %v", err)
	}
	if len(manifest) != 2 || manifest[0].Path != "internal/app.go" || manifest[1].Path != "main.go" {
		t.Fatalf("manifiesto inesperado: %+v", manifest)
	}
	if manifest[1].Bytes != len("package main\n") || manifest[1].Lines != 1 {
		t.Fatalf("entrada de manifiesto inesperada: %+v", manifest[1])
	}
}

func TestWorkspacePublishRejectsIncompleteArtifactDraft(t *testing.T) {
	key := "clave"
	repo := &workspaceProjectRepoFake{
		project: &domain.WorkspaceProject{ID: "project-1", TenantID: "tenant-1", ExerciseID: workspaceImportExerciseID},
		draft: &domain.WorkspaceProjectDraft{
			TenantID:    "tenant-1",
			ProjectID:   "project-1",
			ArtifactKey: &key,
		},
	}
	service := NewWorkspaceProjectService(repo, true)
	if _, err := service.Publish(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1"); err == nil {
		t.Fatal("un borrador con artefacto incompleto no debe publicarse")
	}
	if repo.publishCalls != 0 {
		t.Fatalf("la publicación llegó al repositorio: %d", repo.publishCalls)
	}
}

func TestWorkspaceImportArchiveRejectsOversizedPayload(t *testing.T) {
	service := NewWorkspaceProjectService(&workspaceProjectRepoFake{}, false)
	service.SetArtifactStore(&workspaceArtifactStoreFake{})
	service.SetArchiveEnabled(true)
	oversized := make([]byte, maxWorkspaceProjectArchiveBytes+1)
	if _, err := service.ImportArchive(context.Background(), "tenant-1", workspaceImportExerciseID, "user-1", 0, oversized); !errors.Is(err, domain.ErrWorkspaceProjectArchiveTooLarge) {
		t.Fatalf("se esperaba archive_too_large, llegó %v", err)
	}
}
