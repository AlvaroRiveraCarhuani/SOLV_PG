package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

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
	scriptBuilder.WriteString("elif dpkg -s \"$t\" >/dev/null 2>&1; then ")
	scriptBuilder.WriteString("v=$(dpkg-query -W -f='${Version}' \"$t\" 2>/dev/null || echo \"installed\"); ")
	scriptBuilder.WriteString("echo \"SOLV_TOOL:OK:$t:/var/lib/dpkg:$v\"; ")
	scriptBuilder.WriteString("elif apk info -e \"$t\" >/dev/null 2>&1; then ")
	scriptBuilder.WriteString("echo \"SOLV_TOOL:OK:$t:/lib/apk:installed\"; ")
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
	defer func() { _ = logsReader.Close() }()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
	rawOutput := stdoutBuf.String()

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

	seedStubs := strings.Join([]string{
		`cat << 'EOF' > /tmp/solution.py`,
		`def test_smoke(): assert True`,
		`if __name__ == "__main__": print("SOLV_JUDGE_OK")`,
		`EOF`,
		`cat << 'EOF' > /tmp/solution.c`,
		`#include <stdio.h>`,
		`int main(void) { printf("SOLV_JUDGE_OK\n"); return 0; }`,
		`EOF`,
		`cat << 'EOF' > /tmp/solution.cpp`,
		`#include <iostream>`,
		`int main() { std::cout << "SOLV_JUDGE_OK\n"; return 0; }`,
		`EOF`,
		`cat << 'EOF' > /tmp/solution.rs`,
		`fn main() { println!("SOLV_JUDGE_OK"); }`,
		`EOF`,
		`cat << 'EOF' > /tmp/main.go`,
		`package main`,
		`import "fmt"`,
		`func main() { fmt.Println("SOLV_JUDGE_OK") }`,
		`EOF`,
		`cp /tmp/main.go /tmp/solution.go 2>/dev/null || true`,
		`cat << 'EOF' > /tmp/solution.js`,
		`console.log("SOLV_JUDGE_OK");`,
		`EOF`,
		`cat << 'EOF' > /tmp/Solution.java`,
		`public class Solution { public static void main(String[] a) { System.out.println("SOLV_JUDGE_OK"); } }`,
		`EOF`,
		`cat << 'EOF' > /tmp/solution.php`,
		`<?php echo "SOLV_JUDGE_OK\n";`,
		`EOF`,
		`cat << 'EOF' > /tmp/solution.sh`,
		`echo "SOLV_JUDGE_OK"`,
		`EOF`,
		`cp /tmp/solution.sh /tmp/solution 2>/dev/null || true`,
	}, "\n")

	var runScript string
	if sampleInput != "" {
		runScript = fmt.Sprintf("%s\ncd /tmp && (printf '%%s' %q | (%s))", seedStubs, sampleInput, cleanCmd)
	} else {
		runScript = fmt.Sprintf("%s\ncd /tmp && (%s)", seedStubs, cleanCmd)
	}

	cmd := []string{"sh", "-c", runScript}

	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory: memoryLimitMB * 1024 * 1024,
		},
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp": "rw,exec,nosuid,size=64m",
		},
		SecurityOpt: []string{"no-new-privileges:true"},
		AutoRemove:  false,
	}

	containerConfig := &container.Config{
		Image:      imageRef,
		WorkingDir: "/tmp",
		Cmd:        cmd,
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
		defer func() { _ = logsReader.Close() }()
		var stdoutBuf, stderrBuf bytes.Buffer
		_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
		output = strings.TrimSpace(stdoutBuf.String())
		if stderrBuf.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += strings.TrimSpace(stderrBuf.String())
		}
	}

	return output, durationMs, exitCode, nil
}
