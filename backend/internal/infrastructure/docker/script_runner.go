package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"solv-backend/internal/core/domain"
)

type DockerScriptSandboxRunner struct {
	cli *client.Client
}

func NewDockerScriptSandboxRunner(cli *client.Client) domain.ScriptSandboxRunner {
	return &DockerScriptSandboxRunner{cli: cli}
}

func (r *DockerScriptSandboxRunner) RunPythonScript(ctx context.Context, scriptCode string, timeoutSec int) (string, string, error) {
	if r.cli == nil {
		return "", "", fmt.Errorf("docker client is not initialized")
	}

	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "solv-script-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir for script: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptFile := filepath.Join(tmpDir, "generate.py")
	if err := os.WriteFile(scriptFile, []byte(scriptCode), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write script file: %w", err)
	}

	pidsLimit := int64(64)
	memBytes := int64(256 * 1024 * 1024)

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/runner:ro", tmpDir),
		},
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:     memBytes,
			MemorySwap: memBytes,
			NanoCPUs:   1000000000,
			PidsLimit:  &pidsLimit,
		},
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges:true"},
	}

	imageName := "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93"

	containerConfig := &container.Config{
		Image:        imageName,
		Cmd:          []string{"python3", "/runner/generate.py"},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	resp, err := r.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		containerConfig.Image = "python:3.11-slim"
		resp, err = r.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
		if err != nil {
			return "", "", fmt.Errorf("failed to create script sandbox container: %w", err)
		}
	}
	containerID := resp.ID
	defer func() {
		_ = r.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := r.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", "", fmt.Errorf("failed to start script sandbox container: %w", err)
	}

	statusCh, errCh := r.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return "", "", fmt.Errorf("error waiting for script container: %w", err)
		}
	case waitStatus := <-statusCh:
		if waitStatus.StatusCode != 0 {
			logsReader, _ := r.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
			var outBuf, errBuf bytes.Buffer
			if logsReader != nil {
				_, _ = stdcopy.StdCopy(&outBuf, &errBuf, logsReader)
				logsReader.Close()
			}
			return outBuf.String(), errBuf.String(), fmt.Errorf("script exited with non-zero code %d: %s", waitStatus.StatusCode, errBuf.String())
		}
	case <-ctx.Done():
		return "", "timeout: script execution exceeded 30s limit", fmt.Errorf("script execution timed out")
	}

	logsReader, err := r.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", "", fmt.Errorf("failed to read script logs: %w", err)
	}
	defer logsReader.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)

	return stdoutBuf.String(), stderrBuf.String(), nil
}
