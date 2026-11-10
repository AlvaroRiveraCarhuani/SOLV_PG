package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"solv-backend/internal/core/domain"
)

const (
	maxWorkspaceImportArchiveBytes = 10 * 1024 * 1024
	maxWorkspaceImportZipEntries   = 1000
	maxWorkspaceImportExpanded     = 20 * 1024 * 1024
	maxWorkspaceImportZipEntry     = 2 * 1024 * 1024
	maxWorkspaceImportPathBytes    = 1024
)

// WorkspaceProjectImportFile es una entrada ya descomprimida o leída por el
// navegador. La ruta es relativa al proyecto y el contenido se valida aquí, no
// en la capa HTTP.
type WorkspaceProjectImportFile struct {
	Path    string
	Content []byte
}

type WorkspaceProjectImportResult struct {
	Draft      *domain.WorkspaceProjectDraft `json:"draft"`
	FileCount  int                           `json:"file_count"`
	TotalBytes int                           `json:"total_bytes"`
}

// ImportFolder convierte la selección del navegador en el mismo snapshot que
// produce un ZIP. No publica: solo crea o actualiza el borrador.
func (s *WorkspaceProjectService) ImportFolder(ctx context.Context, tenantID, exerciseID, userID string, expectedRevision int64, files []WorkspaceProjectImportFile) (*WorkspaceProjectImportResult, error) {
	return s.importFiles(ctx, tenantID, exerciseID, userID, expectedRevision, files)
}

// ImportZip protege al extractor de Zip Slip, symlinks, cifrado, bombas y
// tamaños inconsistentes. No escribe en el sistema de archivos: todo se queda
// en memoria y luego pasa por las mismas reglas que una carpeta.
func (s *WorkspaceProjectService) ImportZip(ctx context.Context, tenantID, exerciseID, userID string, expectedRevision int64, archive []byte) (*WorkspaceProjectImportResult, error) {
	if len(archive) == 0 {
		return nil, fmt.Errorf("%w: el archivo ZIP está vacío", domain.ErrWorkspaceProjectImportInvalid)
	}
	if len(archive) > maxWorkspaceImportArchiveBytes {
		return nil, fmt.Errorf("%w: el ZIP supera %d bytes", domain.ErrWorkspaceProjectImportTooLarge, maxWorkspaceImportArchiveBytes)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("%w: ZIP corrupto o no reconocido", domain.ErrWorkspaceProjectImportInvalid)
	}
	if len(reader.File) > maxWorkspaceImportZipEntries {
		return nil, fmt.Errorf("%w: el ZIP supera %d entradas", domain.ErrWorkspaceProjectImportTooLarge, maxWorkspaceImportZipEntries)
	}

	files := make([]WorkspaceProjectImportFile, 0, len(reader.File))
	expanded := 0
	for _, entry := range reader.File {
		if entry.Flags&0x1 != 0 {
			return nil, fmt.Errorf("%w: el ZIP contiene entradas cifradas", domain.ErrWorkspaceProjectImportInvalid)
		}
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: el ZIP contiene symlinks", domain.ErrWorkspaceProjectImportInvalid)
		}
		if entry.FileInfo().IsDir() {
			if err := validateZipDirectory(entry.Name); err != nil {
				return nil, err
			}
			continue
		}
		if !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: el ZIP contiene una entrada especial", domain.ErrWorkspaceProjectImportInvalid)
		}
		if entry.UncompressedSize64 > maxWorkspaceImportZipEntry {
			return nil, fmt.Errorf("%w: la entrada %s supera %d bytes", domain.ErrWorkspaceProjectImportTooLarge, entry.Name, maxWorkspaceImportZipEntry)
		}

		rc, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: no se pudo leer %s", domain.ErrWorkspaceProjectImportInvalid, entry.Name)
		}
		data, readErr := io.ReadAll(io.LimitReader(rc, maxWorkspaceImportZipEntry+1))
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("%w: no se pudo leer %s", domain.ErrWorkspaceProjectImportInvalid, entry.Name)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("%w: no se pudo cerrar %s", domain.ErrWorkspaceProjectImportInvalid, entry.Name)
		}
		if len(data) > maxWorkspaceImportZipEntry {
			return nil, fmt.Errorf("%w: la entrada %s supera %d bytes", domain.ErrWorkspaceProjectImportTooLarge, entry.Name, maxWorkspaceImportZipEntry)
		}
		if uint64(len(data)) != entry.UncompressedSize64 {
			return nil, fmt.Errorf("%w: el tamaño declarado de %s no coincide con su contenido", domain.ErrWorkspaceProjectImportInvalid, entry.Name)
		}
		expanded += len(data)
		if expanded > maxWorkspaceImportExpanded {
			return nil, fmt.Errorf("%w: el ZIP expandido supera %d bytes", domain.ErrWorkspaceProjectImportTooLarge, maxWorkspaceImportExpanded)
		}
		files = append(files, WorkspaceProjectImportFile{Path: entry.Name, Content: data})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("%w: el ZIP no contiene archivos", domain.ErrWorkspaceProjectImportInvalid)
	}
	return s.importFiles(ctx, tenantID, exerciseID, userID, expectedRevision, files)
}

func (s *WorkspaceProjectService) importFiles(ctx context.Context, tenantID, exerciseID, userID string, expectedRevision int64, raw []WorkspaceProjectImportFile) (*WorkspaceProjectImportResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workspace project repository is not configured")
	}
	files, total, err := normalizeWorkspaceProjectFiles(raw)
	if err != nil {
		return nil, err
	}
	snapshot := domain.EncodeProjectSnapshot(files)

	draft, err := s.SaveDraft(ctx, tenantID, exerciseID, userID, SaveWorkspaceDraftRequest{
		TextSnapshot:     snapshot,
		ExpectedRevision: expectedRevision,
	})
	if err != nil {
		return nil, err
	}
	return &WorkspaceProjectImportResult{
		Draft:      draft,
		FileCount:  len(files),
		TotalBytes: total,
	}, nil
}

// normalizeWorkspaceProjectFiles aplica las reglas de la importación (rutas
// seguras, UTF-8, contenido textual y límites) sin persistir nada. La usa la
// importación antes de guardar el borrador y la generación con IA antes de
// ofrecer una propuesta al docente.
func normalizeWorkspaceProjectFiles(raw []WorkspaceProjectImportFile) ([]domain.ProjectFile, int, error) {
	if len(raw) == 0 {
		return nil, 0, fmt.Errorf("%w: no hay archivos para importar", domain.ErrWorkspaceProjectImportInvalid)
	}
	if len(raw) > maxWorkspaceProjectFiles {
		return nil, 0, fmt.Errorf("%w: el proyecto supera %d archivos", domain.ErrWorkspaceProjectImportTooLarge, maxWorkspaceProjectFiles)
	}

	files := make([]domain.ProjectFile, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	total := 0
	for _, entry := range raw {
		normalizedPath, err := normalizeWorkspaceImportPath(entry.Path)
		if err != nil {
			return nil, 0, err
		}
		key := norm.NFC.String(normalizedPath)
		if _, exists := seen[key]; exists {
			return nil, 0, fmt.Errorf("%w: ruta duplicada %s", domain.ErrWorkspaceProjectImportInvalid, normalizedPath)
		}
		seen[key] = struct{}{}

		if len(entry.Content) > maxWorkspaceProjectFileBytes {
			return nil, 0, fmt.Errorf("%w: el archivo %s supera %d bytes", domain.ErrWorkspaceProjectImportTooLarge, normalizedPath, maxWorkspaceProjectFileBytes)
		}
		total += len(entry.Content)
		if total > maxWorkspaceProjectBytes {
			return nil, 0, fmt.Errorf("%w: el proyecto supera %d bytes", domain.ErrWorkspaceProjectImportTooLarge, maxWorkspaceProjectBytes)
		}
		content, err := normalizeWorkspaceImportContent(normalizedPath, entry.Content)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, domain.ProjectFile{Path: normalizedPath, Content: content})
	}

	snapshot := domain.EncodeProjectSnapshot(files)
	parsed, ok := domain.ParseProjectSnapshot(snapshot, "")
	if !ok || len(parsed) != len(files) {
		return nil, 0, fmt.Errorf("%w: el proyecto no se puede representar como snapshot", domain.ErrWorkspaceProjectImportInvalid)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for i := range files {
		if parsed[i].Path != files[i].Path || parsed[i].Content != files[i].Content {
			return nil, 0, fmt.Errorf("%w: el archivo %s no se conserva completo en el snapshot", domain.ErrWorkspaceProjectImportInvalid, files[i].Path)
		}
	}
	return files, total, nil
}

func normalizeWorkspaceImportPath(raw string) (string, error) {
	if raw == "" || len(raw) > maxWorkspaceImportPathBytes {
		return "", fmt.Errorf("%w: ruta vacía o demasiado larga", domain.ErrWorkspaceProjectImportInvalid)
	}
	if strings.Contains(raw, `\`) {
		return "", fmt.Errorf("%w: la ruta %q usa separadores ambiguos", domain.ErrWorkspaceProjectImportInvalid, raw)
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "", fmt.Errorf("%w: la ruta %q es absoluta", domain.ErrWorkspaceProjectImportInvalid, raw)
	}
	for _, r := range raw {
		if unicode.IsControl(r) || r == 0x7f {
			return "", fmt.Errorf("%w: la ruta %q contiene caracteres de control", domain.ErrWorkspaceProjectImportInvalid, raw)
		}
	}
	normalized := norm.NFC.String(raw)
	if path.Clean(normalized) != normalized || strings.HasSuffix(normalized, "/") {
		return "", fmt.Errorf("%w: la ruta %q no está normalizada", domain.ErrWorkspaceProjectImportInvalid, raw)
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.TrimSpace(segment) != segment {
			return "", fmt.Errorf("%w: la ruta %q contiene segmentos ambiguos", domain.ErrWorkspaceProjectImportInvalid, raw)
		}
	}
	return normalized, nil
}

func normalizeWorkspaceImportContent(filePath string, data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%w: el archivo %s no es texto UTF-8", domain.ErrWorkspaceProjectImportInvalid, filePath)
	}
	for _, b := range data {
		if b == 0 || (b < 0x20 && b != '\n' && b != '\r' && b != '\t') || b == 0x7f {
			return "", fmt.Errorf("%w: el archivo %s contiene datos binarios o de control", domain.ErrWorkspaceProjectImportInvalid, filePath)
		}
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.TrimPrefix(content, "\ufeff")
	if strings.TrimSpace(content) == "" {
		content = ""
	} else {
		// El formato de snapshot no distingue el final de archivo de una línea
		// en blanco. Se canonicaliza a un único salto final para que el archivo
		// se lea y se guarde igual en todas las capas.
		content = strings.TrimRight(content, "\n") + "\n"
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# === FILE: ") && strings.HasSuffix(line, " ===") {
			return "", fmt.Errorf("%w: el archivo %s contiene una cabecera de snapshot ambigua", domain.ErrWorkspaceProjectImportInvalid, filePath)
		}
	}
	return content, nil
}

func validateZipDirectory(name string) error {
	trimmed := strings.TrimSuffix(name, "/")
	if trimmed == "" {
		return nil
	}
	if _, err := normalizeWorkspaceImportPath(trimmed); err != nil {
		return err
	}
	return nil
}
