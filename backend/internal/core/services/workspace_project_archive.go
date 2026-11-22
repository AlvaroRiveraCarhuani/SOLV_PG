package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"solv-backend/internal/core/domain"
)

// Límites del corte mínimo del proyecto grande (ADR-090). Son más bajos que los
// propuestos en el ADR porque la extracción todavía ocurre en memoria: subirlos
// exige escribir en streaming dentro del volumen antes de habilitar la carga
// para todo un semestre.
const (
	maxWorkspaceProjectArchiveBytes    = 32 * 1024 * 1024
	maxWorkspaceProjectArchiveExpanded = 128 * 1024 * 1024
	maxWorkspaceProjectArchiveFiles    = 20000
	maxWorkspaceProjectArchiveFile     = 4 * 1024 * 1024
	maxWorkspaceProjectArchivePath     = 1024
)

// ImportArchive recibe el tar.gz del proyecto, lo valida completo, lo guarda en
// el depósito de objetos y lo deja como borrador. No publica: la versión y la
// siembra siguen el camino de siempre.
func (s *WorkspaceProjectService) ImportArchive(
	ctx context.Context,
	tenantID, exerciseID, userID string,
	expectedRevision int64,
	archive []byte,
) (*WorkspaceProjectImportResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workspace project repository is not configured")
	}
	if !s.archiveEnabled {
		return nil, domain.ErrWorkspaceProjectArchiveDisabled
	}
	if s.artifacts == nil {
		return nil, domain.ErrWorkspaceProjectArchiveStoreUnavailable
	}
	files, total, err := inspectWorkspaceProjectArchive(archive)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	key, err := domain.WorkspaceProjectArtifactKey(tenantID, exerciseID, hash)
	if err != nil {
		return nil, err
	}
	storedHash, storedBytes, err := s.artifacts.Put(ctx, key, bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("no se pudo guardar el archivo del proyecto: %w", err)
	}
	if !strings.EqualFold(storedHash, hash) || storedBytes != int64(len(archive)) {
		return nil, fmt.Errorf("%w: el depósito devolvió otra huella o tamaño", domain.ErrWorkspaceProjectArchiveInvalid)
	}

	draft, err := s.SaveDraft(ctx, tenantID, exerciseID, userID, SaveWorkspaceDraftRequest{
		ExpectedRevision: expectedRevision,
		Artifact: &domain.WorkspaceProjectArtifact{
			Key:    key,
			SHA256: hash,
			Bytes:  int64(len(archive)),
			Files:  files,
		},
	})
	if err != nil {
		return nil, err
	}
	return &WorkspaceProjectImportResult{
		Draft:      draft,
		FileCount:  files,
		TotalBytes: int(total),
	}, nil
}

// inspectWorkspaceProjectArchive recorre el tar.gz sin escribir nada. Rechaza
// rutas que salen del proyecto, entradas que no son archivos regulares y
// cualquier tamaño que no se pueda comprobar con el contenido real.
func inspectWorkspaceProjectArchive(archive []byte) (int, int64, error) {
	if len(archive) == 0 {
		return 0, 0, fmt.Errorf("%w: el archivo está vacío", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	if len(archive) > maxWorkspaceProjectArchiveBytes {
		return 0, 0, fmt.Errorf("%w: el archivo supera %d bytes", domain.ErrWorkspaceProjectArchiveTooLarge, maxWorkspaceProjectArchiveBytes)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return 0, 0, fmt.Errorf("%w: no es un gzip válido", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	seen := make(map[string]struct{}, 64)
	count := 0
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, fmt.Errorf("%w: tar corrupto o truncado", domain.ErrWorkspaceProjectArchiveInvalid)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := normalizeArchiveEntryPath(header.Name); err != nil {
				return 0, 0, err
			}
			continue
		case tar.TypeReg:
		default:
			return 0, 0, fmt.Errorf("%w: la entrada %q no es un archivo regular", domain.ErrWorkspaceProjectArchiveInvalid, header.Name)
		}

		name, err := normalizeArchiveEntryPath(header.Name)
		if err != nil {
			return 0, 0, err
		}
		if _, duplicate := seen[name]; duplicate {
			return 0, 0, fmt.Errorf("%w: ruta duplicada %q", domain.ErrWorkspaceProjectArchiveInvalid, name)
		}
		if header.Size < 0 || header.Size > maxWorkspaceProjectArchiveFile {
			return 0, 0, fmt.Errorf("%w: el archivo %q excede %d bytes", domain.ErrWorkspaceProjectArchiveTooLarge, name, maxWorkspaceProjectArchiveFile)
		}
		written, err := io.Copy(io.Discard, io.LimitReader(reader, header.Size+1))
		if err != nil {
			return 0, 0, fmt.Errorf("%w: no se pudo leer %q", domain.ErrWorkspaceProjectArchiveInvalid, name)
		}
		if written != header.Size {
			return 0, 0, fmt.Errorf("%w: el tamaño declarado de %q no coincide con su contenido", domain.ErrWorkspaceProjectArchiveInvalid, name)
		}
		seen[name] = struct{}{}
		count++
		total += written
		if count > maxWorkspaceProjectArchiveFiles {
			return 0, 0, fmt.Errorf("%w: el proyecto supera %d archivos", domain.ErrWorkspaceProjectArchiveTooLarge, maxWorkspaceProjectArchiveFiles)
		}
		if total > maxWorkspaceProjectArchiveExpanded {
			return 0, 0, fmt.Errorf("%w: el proyecto expandido supera %d bytes", domain.ErrWorkspaceProjectArchiveTooLarge, maxWorkspaceProjectArchiveExpanded)
		}
	}

	if count == 0 {
		return 0, 0, fmt.Errorf("%w: el archivo no contiene archivos", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	for name := range seen {
		segments := strings.Split(name, "/")
		for i := 1; i < len(segments); i++ {
			if _, collision := seen[strings.Join(segments[:i], "/")]; collision {
				return 0, 0, fmt.Errorf("%w: la ruta %q colisiona con un archivo del mismo nombre", domain.ErrWorkspaceProjectArchiveInvalid, name)
			}
		}
	}
	return count, total, nil
}

// extractWorkspaceProjectArchive devuelve el contenido del artefacto ya
// validado. Se usa al publicar para dejar el manifiesto por archivo y al revisar
// una entrega para reconstruir la base sin depender del texto del ejercicio.
// packWorkspaceProjectArchive empaqueta un árbol ya validado en tar.gz. Las
// entradas salen ordenadas por ruta para que el mismo árbol produzca el mismo
// hash y el depósito no duplique objetos.
func packWorkspaceProjectArchive(files []WorkspaceProjectImportFile) ([]byte, error) {
	sorted := make([]WorkspaceProjectImportFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	writer := tar.NewWriter(gzipWriter)
	for _, file := range sorted {
		header := &tar.Header{
			Name:     file.Path,
			Mode:     0o644,
			Size:     int64(len(file.Content)),
			Typeflag: tar.TypeReg,
			ModTime:  time.Unix(0, 0),
		}
		if err := writer.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("no se pudo empaquetar el proyecto: %w", err)
		}
		if _, err := writer.Write(file.Content); err != nil {
			return nil, fmt.Errorf("no se pudo empaquetar el proyecto: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("no se pudo empaquetar el proyecto: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("no se pudo empaquetar el proyecto: %w", err)
	}
	return buf.Bytes(), nil
}

// submissionArtifactKey arma la clave del objeto de una entrega. El id es único
// por entrega, así que dos entregas del mismo alumno no se pisan aunque el árbol
// sea idéntico.
func submissionArtifactKey(tenantID, exerciseID, submissionID string) string {
	return path.Join("submissions", tenantID, exerciseID, submissionID+".tar.gz")
}

func extractWorkspaceProjectArchive(archive []byte) ([]WorkspaceProjectImportFile, error) {
	if _, _, err := inspectWorkspaceProjectArchive(archive); err != nil {
		return nil, err
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("%w: no es un gzip válido", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	files := make([]WorkspaceProjectImportFile, 0, 64)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: tar corrupto o truncado", domain.ErrWorkspaceProjectArchiveInvalid)
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		name, err := normalizeArchiveEntryPath(header.Name)
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(io.LimitReader(reader, header.Size+1))
		if err != nil {
			return nil, fmt.Errorf("%w: no se pudo leer %q", domain.ErrWorkspaceProjectArchiveInvalid, name)
		}
		if int64(len(content)) != header.Size {
			return nil, fmt.Errorf("%w: el tamaño declarado de %q no coincide con su contenido", domain.ErrWorkspaceProjectArchiveInvalid, name)
		}
		files = append(files, WorkspaceProjectImportFile{Path: name, Content: content})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// normalizeArchiveEntryPath exige una ruta relativa, limpia y sin el directorio
// de metadatos de la siembra. Una entrada como "a/../b" se rechaza en vez de
// corregirse: la intención del archivo no es verificable.
func normalizeArchiveEntryPath(name string) (string, error) {
	raw := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" || len(raw) > maxWorkspaceProjectArchivePath {
		return "", fmt.Errorf("%w: ruta vacía o demasiado larga", domain.ErrWorkspaceProjectArchiveInvalid)
	}
	if strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("%w: la ruta %q es absoluta", domain.ErrWorkspaceProjectArchiveInvalid, name)
	}
	clean := path.Clean(raw)
	if clean != raw {
		return "", fmt.Errorf("%w: la ruta %q no está normalizada", domain.ErrWorkspaceProjectArchiveInvalid, name)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: la ruta %q sale del proyecto", domain.ErrWorkspaceProjectArchiveInvalid, name)
	}
	if clean == ".solv_workspace" || strings.HasPrefix(clean, ".solv_workspace/") {
		return "", fmt.Errorf("%w: la ruta %q usa el directorio de metadatos", domain.ErrWorkspaceProjectArchiveInvalid, name)
	}
	for _, r := range clean {
		if r < 0x20 || r == 0x7f || unicode.IsControl(r) {
			return "", fmt.Errorf("%w: la ruta %q contiene caracteres de control", domain.ErrWorkspaceProjectArchiveInvalid, name)
		}
	}
	return clean, nil
}
