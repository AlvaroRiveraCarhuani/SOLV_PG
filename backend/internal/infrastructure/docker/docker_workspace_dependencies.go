package docker

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

// preparedTreeMountPoint es donde el contenedor auxiliar ve el árbol preparado.
// Se monta solo lectura: el enlace no modifica el árbol, solo lo referencia.
const preparedTreeMountPoint = "/prepared"

// dependencyLinkMarkerName guarda la clave enlazada dentro del diario del
// volumen. Un volumen que ya recibió esa clave no se vuelve a enlazar.
const dependencyLinkMarkerName = "deps.json"

// LinkWorkspaceDependencies copia al volumen solo las rutas declaradas en el
// perfil. Usa enlaces duros, así que dos volúmenes de la misma clave no suman
// bytes; cuando el sistema de archivos no los permite, cae a una copia normal.
func (c *Client) LinkWorkspaceDependencies(
	ctx context.Context,
	volumeName string,
	tree domain.WorkspaceDependencyTree,
) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("environment volume name is required")
	}
	script, err := buildDependencyLinkScript(tree)
	if err != nil {
		return err
	}
	if err := c.ensurePracticeEnvironmentHelperImage(ctx); err != nil {
		return err
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s:rw", volumeName, environmentVolumeMountPoint),
			fmt.Sprintf("%s:%s:ro", tree.RootPath, preparedTreeMountPoint),
		},
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
		return fmt.Errorf("failed to create dependency link container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start dependency link container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var statusCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return fmt.Errorf("error waiting for dependency link container: %w", waitErr)
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}

	outStream, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return fmt.Errorf("failed to read dependency link output: %w", err)
	}
	defer func() { _ = outStream.Close() }()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, outStream); err != nil {
		return fmt.Errorf("failed to demultiplex dependency link output: %w", err)
	}
	if statusCode != 0 {
		return fmt.Errorf("dependency link failed with exit code %d: %s%s",
			statusCode, tail(stdout.String(), 800), tail(stderr.String(), 800))
	}
	return nil
}

// buildDependencyLinkScript arma el guion del contenedor auxiliar. Cada ruta se
// cita: viene del catálogo y no de una fuente confiable en tiempo de ejecución.
// El marcador se escribe al final, así una interrupción deja el volumen sin
// marcar y el próximo intento rehace el enlace completo.
func buildDependencyLinkScript(tree domain.WorkspaceDependencyTree) (string, error) {
	if err := tree.Validate(); err != nil {
		return "", err
	}
	var script strings.Builder
	script.WriteString("set -eu\n")
	script.WriteString("cd " + shellQuote(environmentVolumeMountPoint) + "\n")
	marker := shellQuote(path.Join(workspaceSeedMetadataDir, dependencyLinkMarkerName))
	script.WriteString("marker=" + marker + "\n")
	fmt.Fprintf(&script, "if [ -f \"$marker\" ] && [ \"$(cat \"$marker\")\" = %s ]; then\n", shellQuote(tree.Key))
	script.WriteString("  echo \"dependencias ya enlazadas\"\n  exit 0\nfi\n")
	fmt.Fprintf(&script, "if [ ! -d %s ]; then\n", shellQuote(workspaceSeedMetadataDir))
	script.WriteString("  echo \"el volumen no está inicializado\"\n  exit 3\nfi\n")

	for _, rawPath := range tree.Paths {
		source := shellQuote(path.Join(preparedTreeMountPoint, rawPath))
		target := shellQuote("./" + rawPath)
		parent := shellQuote("./" + path.Dir(rawPath))
		fmt.Fprintf(&script, "rm -rf %s\n", target)
		fmt.Fprintf(&script, "mkdir -p %s\n", parent)
		fmt.Fprintf(&script, "if ! cp -al %s %s 2>/dev/null; then\n", source, target)
		fmt.Fprintf(&script, "  echo \"enlace duro no disponible en %s: se copia\"\n", rawPath)
		fmt.Fprintf(&script, "  rm -rf %s\n", target)
		fmt.Fprintf(&script, "  cp -a %s %s\n", source, target)
		script.WriteString("fi\n")
		fmt.Fprintf(&script, "test -e %s\n", target)
	}
	fmt.Fprintf(&script, "printf '%%s' %s > \"$marker\"\n", shellQuote(tree.Key))
	return script.String(), nil
}
