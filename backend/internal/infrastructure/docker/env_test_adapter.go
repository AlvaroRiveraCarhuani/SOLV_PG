package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"

	"solv-backend/internal/core/domain"
)

var (
	ErrPullStalled = errors.New(domain.EnvTestErrPullStalled)
)

type dockerPullMsg struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// EnvTestDockerAdapter implementa domain.ImageRegistryPort y domain.ContainerRunnerPort usando Docker SDK
type EnvTestDockerAdapter struct {
	cli                *client.Client
	inactivityDeadline time.Duration
}

// NewEnvTestDockerAdapter inicializa el adaptador de docker para prueba de entorno
func NewEnvTestDockerAdapter(cli *client.Client, inactivityDeadline time.Duration) *EnvTestDockerAdapter {
	if inactivityDeadline <= 0 {
		inactivityDeadline = 60 * time.Second
	}
	return &EnvTestDockerAdapter{
		cli:                cli,
		inactivityDeadline: inactivityDeadline,
	}
}

// InspectLocal comprueba si la imagen está presente en el daemon local y retorna su tamaño y digest
func (a *EnvTestDockerAdapter) InspectLocal(ctx context.Context, imageRef string) (bool, int64, string, error) {
	inspect, _, err := a.cli.ImageInspectWithRaw(ctx, imageRef)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, 0, "", nil
		}
		return false, 0, "", err
	}

	localDigest := ""
	if len(inspect.RepoDigests) > 0 {
		// RepoDigests suele tener la forma "repo@sha256:..."
		parts := strings.Split(inspect.RepoDigests[0], "@")
		if len(parts) > 1 {
			localDigest = parts[1]
		} else {
			localDigest = inspect.RepoDigests[0]
		}
	} else {
		localDigest = inspect.ID
	}

	return true, inspect.Size, localDigest, nil
}

// InspectRemoteDigest consulta el registro OCI para obtener el digest del descriptor del manifiesto
func (a *EnvTestDockerAdapter) InspectRemoteDigest(ctx context.Context, imageRef string) (string, error) {
	dist, err := a.cli.DistributionInspect(ctx, imageRef, "")
	if err != nil {
		return "", err
	}
	return string(dist.Descriptor.Digest), nil
}

// PullImage realiza el pull de la imagen con deadline de inactividad (DA-02) y reporte continuo de progreso
func (a *EnvTestDockerAdapter) PullImage(ctx context.Context, imageRef string, onProgress func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string)) error {
	pullCtx, cancelPull := context.WithCancel(ctx)
	defer cancelPull()

	reader, err := a.cli.ImagePull(pullCtx, imageRef, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	type layerStats struct {
		current int64
		total   int64
	}
	layers := make(map[string]*layerStats)

	lineCh := make(chan []byte)
	readErrCh := make(chan error, 1)

	// Goroutine que lee líneas JSON del stream
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Bytes()
			cp := make([]byte, len(line))
			copy(cp, line)
			select {
			case lineCh <- cp:
			case <-pullCtx.Done():
				return
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			readErrCh <- scanErr
		} else {
			readErrCh <- io.EOF
		}
	}()

	timer := time.NewTimer(a.inactivityDeadline)
	defer timer.Stop()

	for {
		select {
		case <-pullCtx.Done():
			return pullCtx.Err()

		case <-timer.C:
			// Inactivity deadline expirado (DA-02)
			cancelPull()
			return ErrPullStalled

		case err := <-readErrCh:
			if err == io.EOF {
				return nil
			}
			return err

		case line := <-lineCh:
			var msg dockerPullMsg
			if jsonErr := json.Unmarshal(line, &msg); jsonErr != nil {
				continue
			}

			// Reseteamos el timer de inactividad con cada avance
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(a.inactivityDeadline)

			if msg.ID != "" {
				l, exists := layers[msg.ID]
				if !exists {
					l = &layerStats{}
					layers[msg.ID] = l
				}
				if msg.ProgressDetail.Current > 0 {
					l.current = msg.ProgressDetail.Current
				}
				if msg.ProgressDetail.Total > 0 {
					l.total = msg.ProgressDetail.Total
				}
			}

			var totalDone, totalBytes int64
			var completedLayers, totalLayerCount int
			totalLayerCount = len(layers)

			for _, l := range layers {
				totalDone += l.current
				totalBytes += l.total
				if l.total > 0 && l.current >= l.total {
					completedLayers++
				}
			}

			if onProgress != nil {
				action := msg.Status
				if msg.ID != "" {
					action = fmt.Sprintf("%s: %s", msg.ID, msg.Status)
				}
				onProgress(totalDone, totalBytes, completedLayers, totalLayerCount, action)
			}
		}
	}
}

// RunSmokeTest ejecuta un contenedor efímero aislado para verificar las herramientas indicadas
func (a *EnvTestDockerAdapter) RunSmokeTest(ctx context.Context, imageRef string, tools []string, memoryLimitMB int64) ([]domain.ToolResult, int, error) {
	if memoryLimitMB <= 0 {
		memoryLimitMB = 256
	}

	// Construimos script de smoke test seguro
	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("for t in")
	for _, tool := range tools {
		clean := strings.TrimSpace(tool)
		if clean != "" {
			scriptBuilder.WriteString(" " + clean)
		}
	}
	scriptBuilder.WriteString("; do ")
	scriptBuilder.WriteString("if p=$(command -v \"$t\" 2>/dev/null); then ")
	scriptBuilder.WriteString("v=$(\"$t\" --version 2>&1 | head -n 1 || echo \"\"); ")
	scriptBuilder.WriteString("echo \"SOLV_TOOL:OK:$t:$p:$v\"; ")
	scriptBuilder.WriteString("else ")
	scriptBuilder.WriteString("echo \"SOLV_TOOL:MISSING:$t\"; ")
	scriptBuilder.WriteString("fi; ")
	scriptBuilder.WriteString("done")

	cmd := []string{"sh", "-c", scriptBuilder.String()}

	hostConfig := &container.HostConfig{
		NetworkMode: "none", // Aislamiento de red total
		Resources: container.Resources{
			Memory: memoryLimitMB * 1024 * 1024,
		},
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges:true"},
		AutoRemove:     false,
	}

	containerConfig := &container.Config{
		Image: imageRef,
		Cmd:   cmd,
	}

	resp, err := a.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, 1, fmt.Errorf("error al crear contenedor efímero: %w", err)
	}
	containerID := resp.ID

	// Limpieza garantizada del contenedor efímero al finalizar
	defer func() {
		_ = a.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := a.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, 1, fmt.Errorf("error al iniciar contenedor efímero: %w", err)
	}

	statusCh, errCh := a.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var exitCode int

	select {
	case err := <-errCh:
		if err != nil {
			return nil, 1, fmt.Errorf("error esperando contenedor: %w", err)
		}
	case waitStatus := <-statusCh:
		exitCode = int(waitStatus.StatusCode)
	case <-ctx.Done():
		return nil, 124, ctx.Err()
	}

	// Si fue matado por OOM (código 137)
	if exitCode == 137 {
		return nil, 137, errors.New(domain.EnvTestErrTestOOM)
	}

	logsReader, err := a.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: false})
	if err != nil {
		return nil, exitCode, fmt.Errorf("error al obtener logs del contenedor: %w", err)
	}
	defer logsReader.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, logsReader)
	rawOutput := buf.String()

	toolResultsMap := make(map[string]domain.ToolResult)
	for _, line := range strings.Split(rawOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SOLV_TOOL:OK:") {
			parts := strings.SplitN(trimmed[len("SOLV_TOOL:OK:"):], ":", 3)
			if len(parts) >= 2 {
				name := parts[0]
				path := parts[1]
				version := ""
				if len(parts) == 3 {
					version = parts[2]
				}
				toolResultsMap[name] = domain.ToolResult{
					Name:    name,
					Present: true,
					Path:    path,
					Version: version,
				}
			}
		} else if strings.HasPrefix(trimmed, "SOLV_TOOL:MISSING:") {
			name := strings.TrimSpace(trimmed[len("SOLV_TOOL:MISSING:"):])
			toolResultsMap[name] = domain.ToolResult{
				Name:    name,
				Present: false,
			}
		}
	}

	var results []domain.ToolResult
	missingFound := false
	for _, reqTool := range tools {
		clean := strings.TrimSpace(reqTool)
		if clean == "" {
			continue
		}
		if res, ok := toolResultsMap[clean]; ok {
			results = append(results, res)
			if !res.Present {
				missingFound = true
			}
		} else {
			results = append(results, domain.ToolResult{
				Name:    clean,
				Present: false,
			})
			missingFound = true
		}
	}

	if missingFound && exitCode == 0 {
		exitCode = 1
	}

	return results, exitCode, nil
}

// RunJudgeSmokeTest ejecuta un contenedor efímero aislado sin red para probar el entrypoint de un juez virtual con sample input y timeout
func (a *EnvTestDockerAdapter) RunJudgeSmokeTest(ctx context.Context, imageRef string, entrypoint string, sampleInput string, timeoutMS int, memoryLimitMB int64) (string, int64, int, error) {
	if memoryLimitMB <= 0 {
		memoryLimitMB = 256
	}
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}

	cleanCmd := strings.TrimSpace(entrypoint)
	if cleanCmd == "" {
		cleanCmd = "echo 'SOLV_JUDGE_OK'"
	}

	var runScript string
	if sampleInput != "" {
		runScript = fmt.Sprintf("printf '%%s' %q | (%s)", sampleInput, cleanCmd)
	} else {
		runScript = cleanCmd
	}

	cmd := []string{"sh", "-c", runScript}

	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory: memoryLimitMB * 1024 * 1024,
		},
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges:true"},
		AutoRemove:     false,
	}

	containerConfig := &container.Config{
		Image: imageRef,
		Cmd:   cmd,
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	resp, err := a.cli.ContainerCreate(execCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", 0, 1, fmt.Errorf("error al crear contenedor efímero de juez: %w", err)
	}
	containerID := resp.ID

	defer func() {
		_ = a.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	startTime := time.Now()
	if err := a.cli.ContainerStart(execCtx, containerID, container.StartOptions{}); err != nil {
		return "", 0, 1, fmt.Errorf("error al iniciar contenedor efímero de juez: %w", err)
	}

	statusCh, errCh := a.cli.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int
	select {
	case err := <-errCh:
		if err != nil {
			if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
				return "TLE: Tiempo límite de ejecución excedido", time.Since(startTime).Milliseconds(), 124, errors.New("timeout de ejecución excedido")
			}
			return "", time.Since(startTime).Milliseconds(), 1, err
		}
	case waitResp := <-statusCh:
		exitCode = int(waitResp.StatusCode)
	}
	durationMs := time.Since(startTime).Milliseconds()

	logsReader, err := a.cli.ContainerLogs(context.Background(), containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	output := ""
	if err == nil {
		defer logsReader.Close()
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, logsReader)
		output = strings.TrimSpace(buf.String())
	}

	return output, durationMs, exitCode, nil
}
