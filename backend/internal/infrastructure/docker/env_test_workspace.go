package docker

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

// workspaceSetupTestScript ejecuta el setup de la plantilla sobre un volumen vacío,
// comprueba que /home/workspace sea escribible, informa si el editor persistente
// viaja en la imagen y propaga el código del setup. No arranca OpenVSCode: la prueba
// valida preparación, dependencias y satélites.
const workspaceSetupTestScript = `mkdir -p /home/workspace
mkdir_rc=$?
chmod 0777 /home/workspace 2>/dev/null || true
if [ "$mkdir_rc" -eq 0 ] && touch /home/workspace/.solv_probe 2>/dev/null; then
  rm -f /home/workspace/.solv_probe
  echo "SOLV_REPO:ready"
else
  echo "SOLV_REPO:unwritable"
fi
rc=$mkdir_rc
if [ "$rc" -eq 0 ] && [ -n "$SOLV_SETUP_SCRIPT" ]; then
  sh -c "$SOLV_SETUP_SCRIPT" > /home/workspace/.solv_setup.log 2>&1 || rc=$?
fi
if [ -x /home/.openvscode-server/bin/openvscode-server ]; then
  echo "SOLV_VSCODE:present"
else
  echo "SOLV_VSCODE:missing"
fi
if [ "$rc" -ne 0 ] && [ -f /home/workspace/.solv_setup.log ]; then
  echo "SOLV_SETUP_LOG_BEGIN"
  tail -n 40 /home/workspace/.solv_setup.log 2>/dev/null || true
  echo "SOLV_SETUP_LOG_END"
fi
echo "SOLV_SETUP_EXIT:$rc"
exit "$rc"`

// RunWorkspaceSetupTest crea un volumen temporal, ejecuta el setup de la práctica en
// un contenedor efímero y limpia ambos al terminar. El volumen nunca se reutiliza.
func (a *EnvTestDockerAdapter) RunWorkspaceSetupTest(ctx context.Context, spec domain.WorkspaceSetupTestSpec) (domain.WorkspaceSetupTestResult, error) {
	result := domain.WorkspaceSetupTestResult{}

	volumeName := strings.TrimSpace(spec.VolumeName)
	if volumeName == "" {
		return result, fmt.Errorf("el volumen temporal de la prueba es requerido")
	}
	if strings.TrimSpace(spec.Image) == "" {
		return result, fmt.Errorf("la imagen de la prueba es requerida")
	}

	if err := a.ensureTestVolume(ctx, volumeName); err != nil {
		return result, err
	}
	defer func() {
		// El volumen solo existe para esta prueba: se elimina siempre.
		_ = a.cli.VolumeRemove(context.Background(), volumeName, true)
	}()

	memoryLimitMB := spec.MemoryLimitMB
	if memoryLimitMB <= 0 {
		memoryLimitMB = 512
	}
	timeoutMS := spec.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 5 * 60 * 1000
	}

	networkName := strings.TrimSpace(spec.NetworkName)
	if networkName == "" {
		networkName = "none"
	}

	env := append([]string{}, spec.Env...)
	env = append(env, "SOLV_SETUP_SCRIPT="+strings.TrimSpace(spec.SetupScript))

	hostConfig := &container.HostConfig{
		Binds: []string{volumeName + ":/home/workspace:rw"},
		Resources: container.Resources{
			Memory: memoryLimitMB * 1024 * 1024,
		},
		NetworkMode: container.NetworkMode(networkName),
		SecurityOpt: []string{"no-new-privileges:true"},
		AutoRemove:  false,
	}

	containerConfig := &container.Config{
		Image:      spec.Image,
		Env:        env,
		Entrypoint: []string{"/bin/sh", "-c"},
		Cmd:        []string{workspaceSetupTestScript},
	}

	resp, err := a.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return result, fmt.Errorf("error al crear el contenedor de prueba del workspace: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = a.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	startedAt := time.Now()
	if err := a.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return result, fmt.Errorf("error al iniciar el contenedor de prueba del workspace: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	statusCh, errCh := a.cli.ContainerWait(waitCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int
	select {
	case err := <-errCh:
		if err != nil {
			return result, fmt.Errorf("error esperando la prueba del workspace: %w", err)
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	case <-waitCtx.Done():
		_ = a.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
		return result, fmt.Errorf("la prueba del workspace excedió el tiempo límite de %d ms", timeoutMS)
	}

	result.DurationMs = time.Since(startedAt).Milliseconds()
	result.ExitCode = exitCode

	logs, err := a.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return result, fmt.Errorf("error al leer los logs de la prueba del workspace: %w", err)
	}
	defer func() { _ = logs.Close() }()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logs)
	result.Output = truncateTestOutput(stdoutBuf.String() + stderrBuf.String())

	for _, line := range strings.Split(result.Output, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "SOLV_REPO:ready":
			result.RepoReady = true
		case trimmed == "SOLV_VSCODE:present":
			result.OpenVSCodeSeen = true
		case strings.HasPrefix(trimmed, "SOLV_SETUP_EXIT:"):
			if parsed, parseErr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(trimmed, "SOLV_SETUP_EXIT:"))); parseErr == nil {
				result.ExitCode = parsed
			}
		}
	}

	return result, nil
}

func (a *EnvTestDockerAdapter) ensureTestVolume(ctx context.Context, name string) error {
	if _, err := a.cli.VolumeInspect(ctx, name); err == nil {
		return nil
	}
	_, err := a.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: "local",
		Labels: map[string]string{"solv.purpose": "practice-env-test"},
	})
	if err != nil {
		return fmt.Errorf("error al crear el volumen temporal de la prueba: %w", err)
	}
	return nil
}

func truncateTestOutput(output string) string {
	const maxOutputBytes = 16 * 1024
	trimmed := strings.TrimSpace(output)
	if len(trimmed) <= maxOutputBytes {
		return trimmed
	}
	return trimmed[:maxOutputBytes] + "\n…salida truncada"
}
