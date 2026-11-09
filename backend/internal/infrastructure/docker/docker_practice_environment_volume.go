package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

const (
	// environmentVolumeHelperImage es la imagen efímera usada para operar sobre el
	// volumen del environment sin depender del estado del contenedor principal.
	environmentVolumeHelperImage = "alpine:3.20"
	environmentVolumeMountPoint  = "/environment"
	// environmentBackupRoot es la única carpeta del volumen donde la poda puede
	// borrar. Los respaldos de avances viven ahí.
	environmentBackupRoot = ".solv_backups"
	// maxPracticeEnvironmentTarBytes acota cuánto se lee del tar del volumen antes de
	// parsearlo (protege la RAM del backend frente a volúmenes con binarios).
	maxPracticeEnvironmentTarBytes = 64 * 1024 * 1024
)

var environmentSnapshotExcludedDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"__pycache__":  {},
	".venv":        {},
}

// environmentSnapshotTarCmd excluye en origen los directorios pesados para no
// transportarlos hasta el backend. Los patrones se comparan por nombre base.
var environmentSnapshotTarCmd = "tar -cf - -C " + environmentVolumeMountPoint + " " +
	"--exclude=.git --exclude=node_modules --exclude=__pycache__ --exclude=.venv ."

func (c *Client) ensurePracticeEnvironmentHelperImage(ctx context.Context) error {
	_, _, err := c.cli.ImageInspectWithRaw(ctx, environmentVolumeHelperImage)
	if err == nil {
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect environment helper image %q: %w", environmentVolumeHelperImage, err)
	}
	log.Printf("[PracticeEnvironment Volume] Image %q not found. Pulling...", environmentVolumeHelperImage)
	out, pullErr := c.cli.ImagePull(ctx, environmentVolumeHelperImage, image.PullOptions{})
	if pullErr != nil {
		return fmt.Errorf("failed to pull environment helper image %q: %w", environmentVolumeHelperImage, pullErr)
	}
	defer func() { _ = out.Close() }()
	_, _ = io.Copy(io.Discard, out)
	return nil
}

// SnapshotPracticeEnvironmentVolume lee el volumen nombrado con un contenedor efímero en
// solo lectura y devuelve los archivos de código encontrados. Funciona igual con
// el environment corriendo, hibernado o eliminado.
func (c *Client) SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	if strings.TrimSpace(volumeName) == "" {
		return nil, fmt.Errorf("environment volume name is required")
	}
	if maxBytes <= 0 {
		maxBytes = 512 * 1024
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return nil, err
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s:ro", volumeName, environmentVolumeMountPoint)},
		NetworkMode: "none",
		SecurityOpt: []string{"no-new-privileges:true"},
	}
	containerConfig := &container.Config{
		Image:  environmentVolumeHelperImage,
		User:   "0",
		Labels: TestRunLabels(),
		Cmd:    []string{"sh", "-c", environmentSnapshotTarCmd},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create environment snapshot container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start environment snapshot container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for environment snapshot container: %w", err)
		}
	case <-statusCh:
	}

	outStream, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: false})
	if err != nil {
		return nil, fmt.Errorf("failed to read environment snapshot stream: %w", err)
	}
	defer func() { _ = outStream.Close() }()

	var stdout bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, io.Discard, outStream); err != nil {
		return nil, fmt.Errorf("failed to demultiplex environment snapshot stream: %w", err)
	}

	return parsePracticeEnvironmentSnapshotTar(stdout.Bytes(), maxBytes)
}

// WritePracticeEnvironmentVolumeFiles escribe archivos en el volumen del environment usando
// un contenedor efímero, preservando uid/gid 1000 para que el editor y el alumno
// puedan seguir escribiendo.
func (c *Client) WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("environment volume name is required")
	}
	if len(files) == 0 {
		return nil
	}
	archiveBytes, err := buildPracticeEnvironmentFilesTar(files)
	if err != nil {
		return err
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return err
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s:rw", volumeName, environmentVolumeMountPoint)},
		NetworkMode: "none",
		SecurityOpt: []string{"no-new-privileges:true"},
	}
	containerConfig := &container.Config{
		Image:  environmentVolumeHelperImage,
		User:   "0",
		Labels: TestRunLabels(),
		Cmd:    []string{"sh", "-c", "sleep 60"},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create environment writer container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start environment writer container: %w", err)
	}

	if err := c.cli.CopyToContainer(ctx, containerID, environmentVolumeMountPoint, bytes.NewReader(archiveBytes), container.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("failed to copy files into environment volume: %w", err)
	}

	return nil
}

// ListPracticeEnvironmentBackupDirs devuelve las carpetas de respaldo de root, de la más
// nueva a la más vieja. Se ejecuta con la misma imagen efímera que el resto de
// las operaciones de volumen y no depende de que el environment esté corriendo.
func (c *Client) ListPracticeEnvironmentBackupDirs(ctx context.Context, volumeName, root string) ([]string, error) {
	cleanRoot := normalizePracticeEnvironmentPath(root)
	if cleanRoot == "" {
		return nil, fmt.Errorf("invalid environment backup root %q", root)
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return nil, err
	}

	mountRoot := path.Join(environmentVolumeMountPoint, cleanRoot)
	script := fmt.Sprintf("if [ -d %s ]; then ls -1dt %s/*/ 2>/dev/null || true; fi",
		shellQuote(mountRoot), shellQuote(mountRoot))
	output, err := c.runPracticeEnvironmentVolumeShell(ctx, volumeName, true, script)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	dirs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rel := strings.TrimPrefix(line, environmentVolumeMountPoint+"/")
		rel = strings.TrimSuffix(rel, "/")
		rel = normalizePracticeEnvironmentPath(rel)
		if !isDirectPracticeEnvironmentChild(rel, cleanRoot) {
			continue
		}
		dirs = append(dirs, rel)
	}
	return dirs, nil
}

// DeletePracticeEnvironmentVolumePaths borra carpetas de respaldo ya validadas. La lista
// viene del servicio de avances; esta capa vuelve a comprobar que cada ruta sea
// una hija directa de .solv_backups para que un error de llamada no pueda
// borrar código del alumno.
func (c *Client) DeletePracticeEnvironmentVolumePaths(ctx context.Context, volumeName string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return err
	}

	quoted := make([]string, 0, len(paths))
	for _, raw := range paths {
		rel := normalizePracticeEnvironmentPath(raw)
		if !isDirectPracticeEnvironmentChild(rel, environmentBackupRoot) {
			return fmt.Errorf("refusing to delete non-backup environment path %q", raw)
		}
		quoted = append(quoted, shellQuote(path.Join(environmentVolumeMountPoint, rel)))
	}

	script := "rm -rf " + strings.Join(quoted, " ")
	_, err := c.runPracticeEnvironmentVolumeShell(ctx, volumeName, false, script)
	return err
}

// runPracticeEnvironmentVolumeShell ejecuta un comando corto en un contenedor efímero
// montado sobre el volumen. readOnly evita que una consulta pueda escribir.
func (c *Client) runPracticeEnvironmentVolumeShell(ctx context.Context, volumeName string, readOnly bool, script string) ([]byte, error) {
	if strings.TrimSpace(volumeName) == "" {
		return nil, fmt.Errorf("environment volume name is required")
	}

	mode := "rw"
	if readOnly {
		mode = "ro"
	}
	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s:%s", volumeName, environmentVolumeMountPoint, mode)},
		NetworkMode: "none",
		SecurityOpt: []string{"no-new-privileges:true"},
	}
	containerConfig := &container.Config{
		Image:  environmentVolumeHelperImage,
		User:   "0",
		Labels: TestRunLabels(),
		Cmd:    []string{"sh", "-c", script},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create environment volume container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start environment volume container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for environment volume container: %w", err)
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}

	outStream, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, fmt.Errorf("failed to read environment volume output: %w", err)
	}
	defer func() { _ = outStream.Close() }()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, outStream); err != nil {
		return nil, fmt.Errorf("failed to demultiplex environment volume output: %w", err)
	}
	if statusCode != 0 {
		return nil, fmt.Errorf("environment volume command failed (%d): %s", statusCode, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// isDirectPracticeEnvironmentChild dice si rel es una carpeta inmediata de root. Evita
// que una ruta con ../ o con más segmentos llegue al borrado.
func isDirectPracticeEnvironmentChild(rel, root string) bool {
	prefix := root + "/"
	if !strings.HasPrefix(rel, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(rel, prefix)
	return remainder != "" && !strings.Contains(remainder, "/")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// parsePracticeEnvironmentSnapshotTar convierte el tar del volumen en archivos de código
// deterministas: ignora directorios excluidos, binarios y archivos que exceden el
// tope indicado.
func parsePracticeEnvironmentSnapshotTar(raw []byte, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	reader := tar.NewReader(bytes.NewReader(raw))
	files := make([]domain.PracticeEnvironmentVolumeFile, 0, 32)
	var accumulated int64

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Un tar truncado por el tope de lectura conserva los archivos ya
			// parseados: se devuelve lo recolectado.
			break
		}
		if header == nil || header.Typeflag != tar.TypeReg {
			continue
		}

		relPath := normalizePracticeEnvironmentPath(header.Name)
		if relPath == "" || isExcludedPracticeEnvironmentPath(relPath) {
			continue
		}
		if header.Size <= 0 || header.Size > maxBytes {
			continue
		}

		content := make([]byte, header.Size)
		if _, err := io.ReadFull(reader, content); err != nil {
			continue
		}
		if !isTextContent(content) {
			continue
		}
		if accumulated+int64(len(content)) > maxBytes {
			break
		}
		accumulated += int64(len(content))
		files = append(files, domain.PracticeEnvironmentVolumeFile{Path: relPath, Content: content})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// buildPracticeEnvironmentFilesTar arma el tar que se copia al volumen. Los paths se
// validan para que nunca escapen del environment.
func buildPracticeEnvironmentFilesTar(files map[string]string) ([]byte, error) {
	paths := make([]string, 0, len(files))
	for rawPath := range files {
		cleanPath := normalizePracticeEnvironmentPath(rawPath)
		if cleanPath == "" {
			return nil, fmt.Errorf("invalid environment file path %q", rawPath)
		}
		paths = append(paths, cleanPath)
	}
	sort.Strings(paths)

	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	declaredDirs := make(map[string]struct{})

	for _, relPath := range paths {
		for _, dir := range parentDirs(relPath) {
			if _, ok := declaredDirs[dir]; ok {
				continue
			}
			declaredDirs[dir] = struct{}{}
			if err := writer.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     dir + "/",
				Mode:     0o777,
				Uid:      1000,
				Gid:      1000,
			}); err != nil {
				return nil, fmt.Errorf("failed to write directory header %q: %w", dir, err)
			}
		}

		content := []byte(files[relPath])
		if err := writer.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     relPath,
			Mode:     0o644,
			Size:     int64(len(content)),
			Uid:      1000,
			Gid:      1000,
		}); err != nil {
			return nil, fmt.Errorf("failed to write file header %q: %w", relPath, err)
		}
		if _, err := writer.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write file content %q: %w", relPath, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize environment files archive: %w", err)
	}
	return buf.Bytes(), nil
}

func normalizePracticeEnvironmentPath(rawPath string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	trimmed = strings.TrimPrefix(trimmed, "./")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return ""
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func isExcludedPracticeEnvironmentPath(relPath string) bool {
	for _, segment := range strings.Split(relPath, "/") {
		if _, excluded := environmentSnapshotExcludedDirs[segment]; excluded {
			return true
		}
	}
	return false
}

func parentDirs(relPath string) []string {
	segments := strings.Split(relPath, "/")
	dirs := make([]string, 0, len(segments)-1)
	for i := 1; i < len(segments); i++ {
		dirs = append(dirs, strings.Join(segments[:i], "/"))
	}
	return dirs
}

func isTextContent(content []byte) bool {
	if bytes.IndexByte(content, 0) >= 0 {
		return false
	}
	return utf8.Valid(content)
}
