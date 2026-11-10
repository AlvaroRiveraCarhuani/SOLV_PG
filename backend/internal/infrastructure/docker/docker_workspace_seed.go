package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"solv-backend/internal/core/domain"
)

const (
	workspaceSeedMetadataDir   = ".solv_workspace"
	workspaceSeedPendingName   = "pending.json"
	workspaceSeedMarkerName    = "init.json"
	workspaceSeedMaxFiles      = 300
	workspaceSeedMaxFileBytes  = 512 * 1024
	workspaceSeedMaxTotalBytes = 1024 * 1024
	workspaceSeedMaxMetadata   = 64 * 1024
	workspaceSeedMaxEntries    = 2000
)

// workspaceSeedMarker es el diario interno del volumen. No se interpreta como
// contenido del proyecto y nunca se escribe fuera de .solv_workspace.
type workspaceSeedMarker struct {
	VersionID     string `json:"version_id"`
	ContentSHA256 string `json:"content_sha256"`
	FileCount     int    `json:"file_count"`
	TotalBytes    int    `json:"total_bytes"`
	State         string `json:"state"`
	CreatedAt     string `json:"created_at"`
}

// PrepareWorkspaceVolume escribe una versión publicada en un volumen vacío o
// reanuda una inicialización propia que quedó pendiente. Si encuentra
// contenido no reconocido, otra versión o una modificación, devuelve un error
// y no borra ni reemplaza nada.
func (c *Client) PrepareWorkspaceVolume(ctx context.Context, volumeName string, seed domain.WorkspaceSeed) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("environment volume name is required")
	}
	files, totalBytes, err := validateWorkspaceSeed(seed)
	if err != nil {
		return err
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return err
	}

	rootEntries, err := c.listWorkspaceVolumeRoot(ctx, volumeName)
	if err != nil {
		return err
	}

	metadata, metadataExists, err := c.readWorkspaceVolumeMetadata(ctx, volumeName, rootEntries)
	if err != nil {
		return err
	}
	marker, markerExists, err := parseWorkspaceSeedMarker(metadata[workspaceSeedMarkerName])
	if err != nil {
		return fmt.Errorf("%w: marcador de inicialización ilegible", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	pending, pendingExists, err := parseWorkspaceSeedMarker(metadata[workspaceSeedPendingName])
	if err != nil {
		return fmt.Errorf("%w: diario de inicialización ilegible", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	if markerExists {
		if marker.VersionID == seed.VersionID && marker.ContentSHA256 == seed.ContentSHA256 && marker.State == "verified" {
			return nil
		}
		return fmt.Errorf("%w: el volumen ya fue inicializado con otra versión", domain.ErrWorkspaceSeedVolumeConflict)
	}

	contentEntries := make([]string, 0, len(rootEntries))
	for _, name := range rootEntries {
		if name == workspaceSeedMetadataDir {
			continue
		}
		contentEntries = append(contentEntries, name)
	}

	if len(contentEntries) == 0 && !metadataExists {
		return c.writeWorkspaceSeed(ctx, volumeName, seed, files, totalBytes, false, false)
	}
	if len(contentEntries) == 0 && metadataExists && !pendingExists {
		for name := range metadata {
			if name != workspaceSeedPendingName && name != workspaceSeedMarkerName {
				return fmt.Errorf("%w: metadatos no reconocidos", domain.ErrWorkspaceSeedVolumeAmbiguous)
			}
		}
		return c.writeWorkspaceSeed(ctx, volumeName, seed, files, totalBytes, true, pendingExists)
	}
	if !pendingExists {
		return fmt.Errorf("%w: el volumen contiene archivos ajenos a la inicialización", domain.ErrWorkspaceSeedVolumeNotEmpty)
	}
	if pending.State != "pending" {
		return fmt.Errorf("%w: el diario de inicialización no está en curso", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	if pending.VersionID != seed.VersionID || pending.ContentSHA256 != seed.ContentSHA256 {
		return fmt.Errorf("%w: la inicialización pendiente pertenece a otra versión", domain.ErrWorkspaceSeedVolumeConflict)
	}

	entries, err := c.listWorkspaceVolumeEntries(ctx, volumeName)
	if err != nil {
		return err
	}
	allowed := workspaceSeedAllowedEntries(files)
	for _, entry := range entries {
		if entry == workspaceSeedMetadataDir || strings.HasPrefix(entry, workspaceSeedMetadataDir+"/") {
			continue
		}
		if _, ok := allowed[entry]; !ok {
			return fmt.Errorf("%w: entrada inesperada %q", domain.ErrWorkspaceSeedVolumeAmbiguous, entry)
		}
	}

	existing, err := c.readWorkspaceVolumeTree(ctx, volumeName, "", workspaceSeedMaxTotalBytes+workspaceSeedMaxMetadata, workspaceSeedMaxFiles+16)
	if err != nil {
		return err
	}
	missing, err := workspaceSeedMissingFiles(existing, files)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		if err := c.WritePracticeEnvironmentVolumeFiles(ctx, volumeName, missing); err != nil {
			return fmt.Errorf("failed to resume workspace seed: %w", err)
		}
	}
	return c.finishWorkspaceSeed(ctx, volumeName, seed, files, totalBytes)
}

// VerifyWorkspaceVolume comprueba el marcador interno de una inicialización ya
// completada. No compara el contenido actual: el alumno puede haber modificado
// o borrado sus archivos legítimamente. Solo certifica que el volumen sigue
// siendo el que recibió la versión asignada y no fue reemplazado o perdido.
func (c *Client) VerifyWorkspaceVolume(ctx context.Context, volumeName string, seed domain.WorkspaceSeed) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("environment volume name is required")
	}
	if _, _, err := validateWorkspaceSeed(seed); err != nil {
		return err
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return err
	}

	rootEntries, err := c.listWorkspaceVolumeRoot(ctx, volumeName)
	if err != nil {
		return err
	}
	metadataExists := false
	for _, name := range rootEntries {
		if name == workspaceSeedMetadataDir {
			metadataExists = true
			break
		}
	}
	if !metadataExists {
		return fmt.Errorf("%w: falta el marcador de inicialización del volumen", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}

	metadata, err := c.readWorkspaceVolumeTree(ctx, volumeName, workspaceSeedMetadataDir, workspaceSeedMaxMetadata, 16)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceSeedVolumeAmbiguous) {
			return err
		}
		return fmt.Errorf("%w: no se pudieron leer los metadatos del volumen", domain.ErrWorkspaceSeedVolumeUnavailable)
	}
	markerRaw := ""
	for _, file := range metadata {
		if file.Path == workspaceSeedMarkerName {
			markerRaw = string(file.Content)
			break
		}
	}
	marker, exists, err := parseWorkspaceSeedMarker(markerRaw)
	if err != nil || !exists {
		return fmt.Errorf("%w: marcador de inicialización ausente o ilegible", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	if marker.VersionID != seed.VersionID || marker.ContentSHA256 != seed.ContentSHA256 {
		return fmt.Errorf("%w: el marcador pertenece a otra versión", domain.ErrWorkspaceSeedVolumeConflict)
	}
	if marker.State != "verified" {
		return fmt.Errorf("%w: la inicialización no está verificada", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	return nil
}

func (c *Client) writeWorkspaceSeed(ctx context.Context, volumeName string, seed domain.WorkspaceSeed, files map[string]string, totalBytes int, metadataExists, pendingExists bool) error {
	pendingJSON, err := json.Marshal(workspaceSeedMarker{
		VersionID:     seed.VersionID,
		ContentSHA256: seed.ContentSHA256,
		FileCount:     len(files),
		TotalBytes:    totalBytes,
		State:         "pending",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("failed to encode workspace seed journal: %w", err)
	}

	writeFiles := make(map[string]string, len(files)+1)
	if !metadataExists || !pendingExists {
		writeFiles[path.Join(workspaceSeedMetadataDir, workspaceSeedPendingName)] = string(pendingJSON)
	}
	for name, content := range files {
		writeFiles[name] = content
	}
	if err := c.WritePracticeEnvironmentVolumeFiles(ctx, volumeName, writeFiles); err != nil {
		return fmt.Errorf("failed to write workspace seed: %w", err)
	}
	return c.finishWorkspaceSeed(ctx, volumeName, seed, files, totalBytes)
}

func (c *Client) finishWorkspaceSeed(ctx context.Context, volumeName string, seed domain.WorkspaceSeed, files map[string]string, totalBytes int) error {
	existing, err := c.readWorkspaceVolumeTree(ctx, volumeName, "", workspaceSeedMaxTotalBytes+workspaceSeedMaxMetadata, workspaceSeedMaxFiles+16)
	if err != nil {
		return err
	}
	if err := verifyWorkspaceSeedTree(existing, files); err != nil {
		return err
	}

	markerJSON, err := json.Marshal(workspaceSeedMarker{
		VersionID:     seed.VersionID,
		ContentSHA256: seed.ContentSHA256,
		FileCount:     len(files),
		TotalBytes:    totalBytes,
		State:         "verified",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("failed to encode workspace seed marker: %w", err)
	}
	markerPath := path.Join(workspaceSeedMetadataDir, workspaceSeedMarkerName)
	if err := c.WritePracticeEnvironmentVolumeFiles(ctx, volumeName, map[string]string{markerPath: string(markerJSON)}); err != nil {
		return fmt.Errorf("failed to write workspace seed marker: %w", err)
	}
	if err := c.removeWorkspaceSeedPending(ctx, volumeName); err != nil {
		// El marcador ya es la prueba de una inicialización completa. Un
		// diario residual no autoriza a borrar ni reescribir contenido.
		log.Printf("[PracticeEnvironment Volume] no se pudo retirar el diario de inicialización de %s: %v", volumeName, err)
	}
	return nil
}

func (c *Client) removeWorkspaceSeedPending(ctx context.Context, volumeName string) error {
	file := path.Join(environmentVolumeMountPoint, workspaceSeedMetadataDir, workspaceSeedPendingName)
	_, err := c.runPracticeEnvironmentVolumeShell(ctx, volumeName, false, "rm -f "+shellQuote(file))
	return err
}

func (c *Client) listWorkspaceVolumeRoot(ctx context.Context, volumeName string) ([]string, error) {
	script := `if [ ! -d /environment ]; then echo __SOLV_VOLUME_MISSING__; exit 1; fi
	find /environment -mindepth 1 -maxdepth 1 -print`
	output, err := c.runPracticeEnvironmentVolumeShell(ctx, volumeName, true, script)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		names = append(names, path.Base(line))
	}
	sort.Strings(names)
	return names, nil
}

func (c *Client) listWorkspaceVolumeEntries(ctx context.Context, volumeName string) ([]string, error) {
	script := fmt.Sprintf("find %s -mindepth 1 -print | head -n %d", shellQuote(environmentVolumeMountPoint), workspaceSeedMaxEntries+1)
	output, err := c.runPracticeEnvironmentVolumeShell(ctx, volumeName, true, script)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	entries := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rel := strings.TrimPrefix(line, environmentVolumeMountPoint+"/")
		rel = strings.TrimPrefix(rel, "./")
		if rel == "" || rel == environmentVolumeMountPoint {
			continue
		}
		entries = append(entries, rel)
	}
	if len(entries) > workspaceSeedMaxEntries {
		return nil, fmt.Errorf("%w: el volumen tiene demasiadas entradas", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	sort.Strings(entries)
	return entries, nil
}

func (c *Client) readWorkspaceVolumeMetadata(ctx context.Context, volumeName string, rootEntries []string) (map[string]string, bool, error) {
	found := false
	for _, name := range rootEntries {
		if name == workspaceSeedMetadataDir {
			found = true
			break
		}
	}
	if !found {
		return map[string]string{}, false, nil
	}
	files, err := c.readWorkspaceVolumeTree(ctx, volumeName, workspaceSeedMetadataDir, workspaceSeedMaxMetadata, 16)
	if err != nil {
		return nil, true, fmt.Errorf("%w: no se pudieron leer los metadatos del volumen", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	metadata := make(map[string]string, len(files))
	for _, file := range files {
		metadata[file.Path] = string(file.Content)
	}
	return metadata, true, nil
}

func (c *Client) readWorkspaceVolumeTree(ctx context.Context, volumeName, root string, maxBytes, maxFiles int) ([]domain.PracticeEnvironmentVolumeFile, error) {
	if strings.TrimSpace(volumeName) == "" {
		return nil, fmt.Errorf("environment volume name is required")
	}
	cleanRoot := normalizePracticeEnvironmentPath(root)
	if strings.TrimSpace(root) != "" && cleanRoot == "" {
		return nil, fmt.Errorf("invalid volume tree root")
	}
	archiveCmd := "tar -cf - -C " + shellQuote(environmentVolumeMountPoint) + " ."
	if cleanRoot != "" {
		// Archivar la entrada relativa y no su ruta resuelta evita seguir un
		// symlink que apunte fuera del volumen; el parser lo rechaza como
		// entrada de enlace.
		archiveCmd = "tar -cf - -C " + shellQuote(environmentVolumeMountPoint) + " " + shellQuote(cleanRoot)
	}
	// El proceso tar no debe volcar un volumen grande en memoria del backend.
	// Si el archivo se trunca, el parser lo trata como estado ambiguo y no
	// autoriza ninguna escritura.
	archiveCmd += fmt.Sprintf(" | head -c %d", maxBytes+512*1024)
	archive, err := c.runPracticeEnvironmentVolumeShell(ctx, volumeName, true, archiveCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment volume tree: %w", err)
	}
	if len(archive) > maxBytes {
		return nil, fmt.Errorf("%w: el contenido del volumen excede el límite de lectura", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	return parseWorkspaceSeedTar(archive, cleanRoot, maxBytes, maxFiles)
}

func parseWorkspaceSeedTar(raw []byte, root string, maxBytes, maxFiles int) ([]domain.PracticeEnvironmentVolumeFile, error) {
	reader := tar.NewReader(bytes.NewReader(raw))
	files := make([]domain.PracticeEnvironmentVolumeFile, 0, 16)
	total := 0
	prefix := strings.TrimPrefix(root, "/")
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse environment volume tree: %w", err)
		}
		if header == nil {
			continue
		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return nil, fmt.Errorf("%w: el volumen contiene enlaces", domain.ErrWorkspaceSeedVolumeAmbiguous)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := strings.TrimPrefix(header.Name, "./")
		name = strings.TrimPrefix(name, "/")
		if prefix != "" {
			name = strings.TrimPrefix(name, prefix+"/")
		}
		rel := normalizePracticeEnvironmentPath(name)
		if rel == "" {
			return nil, fmt.Errorf("%w: ruta de volumen inválida", domain.ErrWorkspaceSeedVolumeAmbiguous)
		}
		if root == "" && (rel == workspaceSeedMetadataDir || strings.HasPrefix(rel, workspaceSeedMetadataDir+"/")) {
			continue
		}
		if header.Size < 0 || header.Size > int64(maxBytes) {
			return nil, fmt.Errorf("%w: archivo de volumen fuera de límite", domain.ErrWorkspaceSeedVolumeAmbiguous)
		}
		content := make([]byte, header.Size)
		if _, err := io.ReadFull(reader, content); err != nil {
			return nil, fmt.Errorf("%w: archivo de volumen truncado", domain.ErrWorkspaceSeedVolumeAmbiguous)
		}
		total += len(content)
		if total > maxBytes {
			return nil, fmt.Errorf("%w: el volumen excede el límite de lectura", domain.ErrWorkspaceSeedVolumeAmbiguous)
		}
		files = append(files, domain.PracticeEnvironmentVolumeFile{Path: rel, Content: content})
		if len(files) > maxFiles {
			return nil, fmt.Errorf("%w: el volumen tiene demasiados archivos", domain.ErrWorkspaceSeedVolumeAmbiguous)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func validateWorkspaceSeed(seed domain.WorkspaceSeed) (map[string]string, int, error) {
	if strings.TrimSpace(seed.VersionID) == "" || strings.TrimSpace(seed.ContentSHA256) == "" {
		return nil, 0, fmt.Errorf("%w: falta la identidad de la versión", domain.ErrWorkspaceSeedVersionInvalid)
	}
	if len(seed.Files) == 0 {
		return nil, 0, fmt.Errorf("%w: la versión no contiene archivos", domain.ErrWorkspaceSeedVersionInvalid)
	}
	if len(seed.Files) > workspaceSeedMaxFiles {
		return nil, 0, fmt.Errorf("%w: la versión excede el máximo de archivos", domain.ErrWorkspaceSeedVersionInvalid)
	}
	files := make(map[string]string, len(seed.Files))
	total := 0
	for _, file := range seed.Files {
		clean := normalizePracticeEnvironmentPath(file.Path)
		if clean == "" || clean != file.Path || strings.HasPrefix(clean, workspaceSeedMetadataDir+"/") || clean == workspaceSeedMetadataDir {
			return nil, 0, fmt.Errorf("%w: ruta de proyecto inválida %q", domain.ErrWorkspaceSeedVersionInvalid, file.Path)
		}
		for _, r := range clean {
			if r < 0x20 || r == 0x7f || unicode.IsControl(r) {
				return nil, 0, fmt.Errorf("%w: la ruta %q contiene caracteres de control", domain.ErrWorkspaceSeedVersionInvalid, file.Path)
			}
		}
		if _, exists := files[clean]; exists {
			return nil, 0, fmt.Errorf("%w: ruta duplicada %q", domain.ErrWorkspaceSeedVersionInvalid, clean)
		}
		if len(file.Content) > workspaceSeedMaxFileBytes {
			return nil, 0, fmt.Errorf("%w: el archivo %q excede el tamaño permitido", domain.ErrWorkspaceSeedVersionInvalid, clean)
		}
		total += len(file.Content)
		if total > workspaceSeedMaxTotalBytes {
			return nil, 0, fmt.Errorf("%w: el proyecto excede el tamaño permitido", domain.ErrWorkspaceSeedVersionInvalid)
		}
		files[clean] = file.Content
	}
	for name := range files {
		segments := strings.Split(name, "/")
		for i := 1; i < len(segments); i++ {
			parent := strings.Join(segments[:i], "/")
			if _, collision := files[parent]; collision {
				return nil, 0, fmt.Errorf("%w: la ruta %q colisiona con el archivo %q", domain.ErrWorkspaceSeedVersionInvalid, name, parent)
			}
		}
	}
	return files, total, nil
}

func workspaceSeedAllowedEntries(files map[string]string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(files)*2)
	for file := range files {
		segments := strings.Split(file, "/")
		for i := 1; i <= len(segments); i++ {
			allowed[strings.Join(segments[:i], "/")] = struct{}{}
		}
	}
	return allowed
}

func workspaceSeedMissingFiles(existing []domain.PracticeEnvironmentVolumeFile, expected map[string]string) (map[string]string, error) {
	byPath := make(map[string]string, len(existing))
	for _, file := range existing {
		if _, ok := expected[file.Path]; !ok {
			return nil, fmt.Errorf("%w: archivo inesperado %q", domain.ErrWorkspaceSeedVolumeAmbiguous, file.Path)
		}
		if prior, ok := byPath[file.Path]; ok && prior != string(file.Content) {
			return nil, fmt.Errorf("%w: archivo duplicado %q", domain.ErrWorkspaceSeedVolumeAmbiguous, file.Path)
		}
		byPath[file.Path] = string(file.Content)
	}
	missing := make(map[string]string)
	for name, content := range expected {
		if existingContent, ok := byPath[name]; ok {
			if existingContent != content {
				return nil, fmt.Errorf("%w: el archivo %q fue modificado", domain.ErrWorkspaceSeedVolumeAmbiguous, name)
			}
			continue
		}
		missing[name] = content
	}
	return missing, nil
}

func verifyWorkspaceSeedTree(existing []domain.PracticeEnvironmentVolumeFile, expected map[string]string) error {
	missing, err := workspaceSeedMissingFiles(existing, expected)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: faltan archivos por escribir", domain.ErrWorkspaceSeedVolumeAmbiguous)
	}
	return nil
}

func parseWorkspaceSeedMarker(raw string) (workspaceSeedMarker, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return workspaceSeedMarker{}, false, nil
	}
	var marker workspaceSeedMarker
	if err := json.Unmarshal([]byte(raw), &marker); err != nil {
		return workspaceSeedMarker{}, true, err
	}
	return marker, true, nil
}
