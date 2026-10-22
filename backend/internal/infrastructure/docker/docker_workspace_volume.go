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
	// workspaceVolumeHelperImage es la imagen efímera usada para operar sobre el
	// volumen del workspace sin depender del estado del contenedor principal.
	workspaceVolumeHelperImage = "alpine:3.20"
	workspaceVolumeMountPoint  = "/workspace"
	// maxWorkspaceTarBytes acota cuánto se lee del tar del volumen antes de
	// parsearlo (protege la RAM del backend frente a volúmenes con binarios).
	maxWorkspaceTarBytes = 64 * 1024 * 1024
)

var workspaceSnapshotExcludedDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"__pycache__":  {},
	".venv":        {},
}

// workspaceSnapshotTarCmd excluye en origen los directorios pesados para no
// transportarlos hasta el backend. Los patrones se comparan por nombre base.
var workspaceSnapshotTarCmd = "tar -cf - -C " + workspaceVolumeMountPoint + " " +
	"--exclude=.git --exclude=node_modules --exclude=__pycache__ --exclude=.venv ."

func (c *Client) ensureWorkspaceHelperImage(ctx context.Context) error {
	_, _, err := c.cli.ImageInspectWithRaw(ctx, workspaceVolumeHelperImage)
	if err == nil {
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect workspace helper image %q: %w", workspaceVolumeHelperImage, err)
	}
	log.Printf("[Workspace Volume] Image %q not found. Pulling...", workspaceVolumeHelperImage)
	out, pullErr := c.cli.ImagePull(ctx, workspaceVolumeHelperImage, image.PullOptions{})
	if pullErr != nil {
		return fmt.Errorf("failed to pull workspace helper image %q: %w", workspaceVolumeHelperImage, pullErr)
	}
	defer func() { _ = out.Close() }()
	_, _ = io.Copy(io.Discard, out)
	return nil
}

// SnapshotWorkspaceVolume lee el volumen nombrado con un contenedor efímero en
// solo lectura y devuelve los archivos de código encontrados. Funciona igual con
// el workspace corriendo, hibernado o eliminado.
func (c *Client) SnapshotWorkspaceVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.WorkspaceVolumeFile, error) {
	if strings.TrimSpace(volumeName) == "" {
		return nil, fmt.Errorf("workspace volume name is required")
	}
	if maxBytes <= 0 {
		maxBytes = 512 * 1024
	}
	if err := c.ensureWorkspaceHelperImage(ctx); err != nil {
		return nil, err
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s:ro", volumeName, workspaceVolumeMountPoint)},
		NetworkMode: "none",
		SecurityOpt: []string{"no-new-privileges:true"},
	}
	containerConfig := &container.Config{
		Image: workspaceVolumeHelperImage,
		User:  "0",
		Cmd:   []string{"sh", "-c", workspaceSnapshotTarCmd},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace snapshot container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start workspace snapshot container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for workspace snapshot container: %w", err)
		}
	case <-statusCh:
	}

	outStream, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: false})
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace snapshot stream: %w", err)
	}
	defer func() { _ = outStream.Close() }()

	var stdout bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, io.Discard, outStream); err != nil {
		return nil, fmt.Errorf("failed to demultiplex workspace snapshot stream: %w", err)
	}

	return parseWorkspaceSnapshotTar(stdout.Bytes(), maxBytes)
}

// WriteWorkspaceVolumeFiles escribe archivos en el volumen del workspace usando
// un contenedor efímero, preservando uid/gid 1000 para que el editor y el alumno
// puedan seguir escribiendo.
func (c *Client) WriteWorkspaceVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("workspace volume name is required")
	}
	if len(files) == 0 {
		return nil
	}
	archiveBytes, err := buildWorkspaceFilesTar(files)
	if err != nil {
		return err
	}
	if err := c.ensureWorkspaceHelperImage(ctx); err != nil {
		return err
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s:rw", volumeName, workspaceVolumeMountPoint)},
		NetworkMode: "none",
		SecurityOpt: []string{"no-new-privileges:true"},
	}
	containerConfig := &container.Config{
		Image: workspaceVolumeHelperImage,
		User:  "0",
		Cmd:   []string{"sh", "-c", "sleep 60"},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create workspace writer container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start workspace writer container: %w", err)
	}

	if err := c.cli.CopyToContainer(ctx, containerID, workspaceVolumeMountPoint, bytes.NewReader(archiveBytes), container.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("failed to copy files into workspace volume: %w", err)
	}

	return nil
}

// parseWorkspaceSnapshotTar convierte el tar del volumen en archivos de código
// deterministas: ignora directorios excluidos, binarios y archivos que exceden el
// tope indicado.
func parseWorkspaceSnapshotTar(raw []byte, maxBytes int64) ([]domain.WorkspaceVolumeFile, error) {
	reader := tar.NewReader(bytes.NewReader(raw))
	files := make([]domain.WorkspaceVolumeFile, 0, 32)
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

		relPath := normalizeWorkspacePath(header.Name)
		if relPath == "" || isExcludedWorkspacePath(relPath) {
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
		files = append(files, domain.WorkspaceVolumeFile{Path: relPath, Content: content})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// buildWorkspaceFilesTar arma el tar que se copia al volumen. Los paths se
// validan para que nunca escapen del workspace.
func buildWorkspaceFilesTar(files map[string]string) ([]byte, error) {
	paths := make([]string, 0, len(files))
	for rawPath := range files {
		cleanPath := normalizeWorkspacePath(rawPath)
		if cleanPath == "" {
			return nil, fmt.Errorf("invalid workspace file path %q", rawPath)
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
		return nil, fmt.Errorf("failed to finalize workspace files archive: %w", err)
	}
	return buf.Bytes(), nil
}

func normalizeWorkspacePath(rawPath string) string {
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

func isExcludedWorkspacePath(relPath string) bool {
	for _, segment := range strings.Split(relPath, "/") {
		if _, excluded := workspaceSnapshotExcludedDirs[segment]; excluded {
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
