package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"solv-backend/internal/core/domain"
)

type seedArtifactStore struct {
	objects map[string][]byte
}

func (s *seedArtifactStore) Put(context.Context, string, io.Reader) (string, int64, error) {
	return "", 0, nil
}

func (s *seedArtifactStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, domain.ErrWorkspaceProjectArchiveInvalid
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// La semilla valida el árbol expandido, no el tar.gz: los números tienen que
// salir del archivo real y no del tamaño comprimido de la versión.
func TestWorkspaceSeedForVersionArchiveUsesExpandedTotals(t *testing.T) {
	content := []byte("print('hola')\n")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "main.py", Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatalf("cabecera: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("contenido: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("cerrar tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("cerrar gzip: %v", err)
	}
	archive := buf.Bytes()

	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	key := "workspace-projects/tenant/exercise/" + hash + ".tar.gz"
	bytesLen := int64(len(archive))
	files := 1

	service := &PracticeEnvironmentService{workspaceArtifacts: &seedArtifactStore{objects: map[string][]byte{key: archive}}}
	version := &domain.WorkspaceProjectVersion{
		ID:             "version-1",
		FileCount:      files,
		TotalBytes:     int(bytesLen),
		ArtifactKey:    &key,
		ArtifactSHA256: &hash,
		ArtifactBytes:  &bytesLen,
		ArtifactFiles:  &files,
	}

	seed, err := service.workspaceSeedForVersion(context.Background(), version)
	if err != nil {
		t.Fatalf("semilla del artefacto: %v", err)
	}
	if seed.ArchiveFileCount != 1 {
		t.Fatalf("archivos de la semilla: %d", seed.ArchiveFileCount)
	}
	if seed.ArchiveTotalBytes != int64(len(content)) {
		t.Fatalf("la semilla debe medir el árbol expandido: %d (comprimido %d)", seed.ArchiveTotalBytes, len(archive))
	}
}
