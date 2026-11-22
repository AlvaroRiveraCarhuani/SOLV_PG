package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"solv-backend/internal/core/domain"
)

// workspaceProjectBaseProvider entrega el árbol base de un ejercicio publicado.
// Lo implementa WorkspaceProjectService; el docente lo usa al revisar una
// entrega para armar el diff sin depender del espejo de texto.
type workspaceProjectBaseProvider interface {
	ProjectBaseFiles(ctx context.Context, tenantID, exerciseID string) ([]domain.ProjectFile, string, bool, error)
}

// ProjectBaseFiles devuelve el proyecto base de la versión publicada vigente.
// Con texto se devuelve tal cual; con artefacto se descarga del depósito,
// verifica su huella y se expande. Un ejercicio sin proyecto o sin versión
// publicada devuelve ok=false: no es un error.
func (s *WorkspaceProjectService) ProjectBaseFiles(
	ctx context.Context,
	tenantID, exerciseID string,
) ([]domain.ProjectFile, string, bool, error) {
	if s == nil || s.repo == nil {
		return nil, "", false, nil
	}
	project, err := s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
			return nil, "", false, nil
		}
		return nil, "", false, fmt.Errorf("no se pudo leer el proyecto: %w", err)
	}
	if project == nil || project.CurrentPublishedVersionID == nil || strings.TrimSpace(*project.CurrentPublishedVersionID) == "" {
		return nil, "", false, nil
	}
	version, err := s.repo.GetVersion(ctx, tenantID, *project.CurrentPublishedVersionID)
	if err != nil {
		return nil, "", false, fmt.Errorf("no se pudo leer la versión publicada: %w", err)
	}
	if version == nil {
		return nil, "", false, nil
	}

	artifact := version.Artifact()
	if artifact == nil {
		files, ok := domain.ParseProjectSnapshot(version.TextSnapshot, "")
		if !ok || len(files) == 0 {
			return nil, "", false, nil
		}
		return files, version.TextSnapshot, true, nil
	}

	entries, err := s.readArtifactFiles(ctx, artifact, maxWorkspaceProjectArchiveBytes)
	if err != nil {
		return nil, "", false, err
	}
	files := make([]domain.ProjectFile, 0, len(entries))
	for _, entry := range entries {
		files = append(files, domain.ProjectFile{Path: entry.Path, Content: string(entry.Content)})
	}
	if len(files) == 0 {
		return nil, "", false, nil
	}
	return files, domain.EncodeProjectSnapshot(files), true, nil
}

// baseManifest arma el manifiesto por archivo de un artefacto. Se calcula al
// publicar, no al revisar: la versión queda inmutable con su lista.
func (s *WorkspaceProjectService) baseManifest(
	ctx context.Context,
	artifact *domain.WorkspaceProjectArtifact,
) ([]domain.WorkspaceProjectManifestEntry, error) {
	entries, err := s.readArtifactFiles(ctx, artifact, maxWorkspaceProjectArchiveBytes)
	if err != nil {
		return nil, err
	}
	if artifact == nil {
		return nil, nil
	}
	manifest := make([]domain.WorkspaceProjectManifestEntry, 0, len(entries))
	for _, entry := range entries {
		manifest = append(manifest, domain.WorkspaceProjectManifestEntry{
			Path:  entry.Path,
			Bytes: len(entry.Content),
			Lines: countProjectLines(entry.Content),
		})
	}
	return manifest, nil
}

// ProjectSubmissionFiles trae el árbol entregado por el alumno desde el
// depósito. La entrega grande no guarda texto en la base: la revisión lee el
// tar.gz, verifica su huella y lo expande igual que publicar o sembrar.
func (s *WorkspaceProjectService) ProjectSubmissionFiles(
	ctx context.Context,
	artifact *domain.WorkspaceProjectArtifact,
) ([]domain.ProjectFile, error) {
	entries, err := s.readArtifactFiles(ctx, artifact, int64(maxWorkspaceProjectArchiveExpanded))
	if err != nil {
		return nil, err
	}
	files := make([]domain.ProjectFile, 0, len(entries))
	for _, entry := range entries {
		files = append(files, domain.ProjectFile{Path: entry.Path, Content: string(entry.Content)})
	}
	return files, nil
}

// readArtifactFiles descarga, verifica y expande el artefacto. Es la única
// puerta al depósito para quienes necesitan el árbol: publicar y revisar.
func (s *WorkspaceProjectService) readArtifactFiles(
	ctx context.Context,
	artifact *domain.WorkspaceProjectArtifact,
	maxBytes int64,
) ([]WorkspaceProjectImportFile, error) {
	data, err := downloadVerifiedArtifact(ctx, s.artifactStore(), artifact, maxBytes)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	return extractWorkspaceProjectArchive(data)
}

func (s *WorkspaceProjectService) artifactStore() domain.WorkspaceProjectArtifactStore {
	if s == nil {
		return nil
	}
	return s.artifacts
}

// downloadVerifiedArtifact descarga el objeto y comprueba que su SHA-256 sea el
// emitido. Es la única descarga del depósito: publicar, revisar y sembrar pasan
// por acá. Sin artefacto devuelve nil sin error.
func downloadVerifiedArtifact(
	ctx context.Context,
	store domain.WorkspaceProjectArtifactStore,
	artifact *domain.WorkspaceProjectArtifact,
	maxBytes int64,
) ([]byte, error) {
	if artifact == nil {
		return nil, nil
	}
	if err := artifact.Valid(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, domain.ErrWorkspaceProjectArchiveStoreUnavailable
	}
	reader, err := store.Get(ctx, artifact.Key)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el artefacto: %w", err)
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el artefacto: %w", err)
	}
	if len(data) == 0 || int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: el artefacto tiene un tamaño inesperado", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), artifact.SHA256) {
		return nil, fmt.Errorf("%w: el artefacto no coincide con su huella", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	return data, nil
}

func countProjectLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lines := 0
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines
}
