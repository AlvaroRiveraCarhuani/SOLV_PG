package db

import (
	"bytes"
	"context"
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

type PostgresStrategy struct {
	cli *client.Client
}

func NewPostgresStrategy(cli *client.Client) domain.DBEngineStrategy {
	return &PostgresStrategy{cli: cli}
}

func (s *PostgresStrategy) ExecuteDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	res, err := s.ExecuteEvaluation(ctx, config)
	if err != nil {
		return "", err
	}
	if res.Verdict != domain.VerdictAC {
		return "", fmt.Errorf("postgres dry run failed with verdict %s: %s", res.Verdict, res.ErrorDetails)
	}
	return res.ResultingJSON, nil
}

func (s *PostgresStrategy) ExecuteEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	imageName := "postgres:18-alpine"

	_, _, err := s.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil && errdefs.IsNotFound(err) {
		out, pullErr := s.cli.ImagePull(ctx, imageName, image.PullOptions{})
		if pullErr != nil {
			return domain.DBEvaluationResult{}, fmt.Errorf("failed to pull image %s: %w", imageName, pullErr)
		}
		_, _ = io.Copy(io.Discard, out)
		out.Close()
	}

	memLimit := int64(config.MemoryLimitMB)
	if memLimit <= 0 {
		memLimit = 256
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:     memLimit * 1024 * 1024,
			MemorySwap: memLimit * 1024 * 1024,
		},
		NetworkMode: "none",
	}

	containerConfig := &container.Config{
		Image: imageName,
		Env: []string{
			"POSTGRES_PASSWORD=evalpass",
			"POSTGRES_USER=evaluser",
			"POSTGRES_DB=evaldb",
		},
	}

	resp, err := s.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return domain.DBEvaluationResult{}, fmt.Errorf("failed to create Postgres container: %w", err)
	}
	containerID := resp.ID

	defer func() {
		_ = s.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	startTime := time.Now()
	if err := s.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return domain.DBEvaluationResult{}, fmt.Errorf("failed to start Postgres container: %w", err)
	}

	// Ready check con psql -U evaluser -d evaldb -c "SELECT 1;" usando stdCopy demux
	ready := false
	for i := 0; i < 60; i++ {
		execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
			Cmd:          []string{"psql", "-U", "evaluser", "-d", "evaldb", "-c", "SELECT 1;"},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err == nil {
			attach, err := s.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
			if err == nil {
				var outBuf, errBuf bytes.Buffer
				_, _ = stdCopy(&outBuf, &errBuf, attach.Reader)
				attach.Close()

				inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
				if err == nil && inspect.ExitCode == 0 {
					ready = true
					break
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	if !ready {
		return domain.DBEvaluationResult{
			Verdict:      domain.VerdictRE,
			ErrorDetails: "PostgreSQL engine failed to become ready inside ephemeral container",
		}, nil
	}

	// 1. Inyectar init_script
	if strings.TrimSpace(config.InitScript) != "" {
		if err := s.execPSQL(ctx, containerID, config.InitScript); err != nil {
			return domain.DBEvaluationResult{
				Verdict:      domain.VerdictRE,
				ErrorDetails: fmt.Sprintf("Error al ejecutar init_script en Postgres: %v", err),
			}, nil
		}
	}

	// 2. Inyectar solución SQL
	if strings.TrimSpace(config.SolutionSQL) != "" {
		if err := s.execPSQL(ctx, containerID, config.SolutionSQL); err != nil {
			return domain.DBEvaluationResult{
				Verdict:      domain.VerdictRE,
				ErrorDetails: fmt.Sprintf("Error de sintaxis o ejecución SQL en Postgres: %v", err),
			}, nil
		}
	}

	// 3. Extracción del resultado en JSON con json_agg
	valQuery := strings.TrimSpace(config.ValidationQuery)
	if valQuery == "" {
		valQuery = "SELECT 1;"
	}

	jsonWrappedQuery := fmt.Sprintf("SELECT json_agg(t) FROM (%s) t;", strings.TrimRight(valQuery, ";"))
	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"psql", "-t", "-A", "-U", "evaluser", "-d", "evaldb", "-c", jsonWrappedQuery},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return domain.DBEvaluationResult{}, err
	}

	attach, err := s.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return domain.DBEvaluationResult{}, err
	}
	defer attach.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdCopy(&stdoutBuf, &stderrBuf, attach.Reader)

	inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil || inspect.ExitCode != 0 {
		errStr := strings.TrimSpace(stderrBuf.String())
		if errStr == "" {
			errStr = strings.TrimSpace(stdoutBuf.String())
		}
		return domain.DBEvaluationResult{
			Verdict:      domain.VerdictRE,
			ErrorDetails: fmt.Sprintf("Error al ejecutar validation_query en Postgres: %s", errStr),
		}, nil
	}

	execTime := time.Since(startTime)
	return domain.DBEvaluationResult{
		Verdict:       domain.VerdictAC,
		ExecutionTime: execTime,
		ResultingJSON: strings.TrimSpace(stdoutBuf.String()),
	}, nil
}

func (s *PostgresStrategy) execPSQL(ctx context.Context, containerID string, sqlCommand string) error {
	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"psql", "-U", "evaluser", "-d", "evaldb", "-c", sqlCommand},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return err
	}

	attach, err := s.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return err
	}
	defer attach.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdCopy(&stdoutBuf, &stderrBuf, attach.Reader)

	inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return err
	}

	if inspect.ExitCode != 0 {
		errStr := strings.TrimSpace(stderrBuf.String())
		if errStr == "" {
			errStr = strings.TrimSpace(stdoutBuf.String())
		}
		return fmt.Errorf("Postgres Exit Code %d: %s", inspect.ExitCode, errStr)
	}

	return nil
}
