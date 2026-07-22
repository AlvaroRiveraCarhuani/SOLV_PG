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

type MongoStrategy struct {
	cli *client.Client
}

func NewMongoStrategy(cli *client.Client) domain.DBEngineStrategy {
	return &MongoStrategy{cli: cli}
}

func (s *MongoStrategy) ExecuteDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	res, err := s.ExecuteEvaluation(ctx, config)
	if err != nil {
		return "", err
	}
	if res.Verdict != domain.VerdictAC {
		return "", fmt.Errorf("mongo dry run failed with verdict %s: %s", res.Verdict, res.ErrorDetails)
	}
	return res.ResultingJSON, nil
}

func (s *MongoStrategy) ExecuteEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	imageName := "mongo:7.0"

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
			"MONGO_INITDB_DATABASE=evaldb",
		},
	}

	resp, err := s.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return domain.DBEvaluationResult{}, fmt.Errorf("failed to create Mongo container: %w", err)
	}
	containerID := resp.ID

	defer func() {
		_ = s.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	startTime := time.Now()
	if err := s.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return domain.DBEvaluationResult{}, fmt.Errorf("failed to start Mongo container: %w", err)
	}

	// Ready check con mongosh eval db.adminCommand('ping') usando stdCopy demux
	ready := false
	for i := 0; i < 60; i++ {
		execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
			Cmd:          []string{"mongosh", "--quiet", "--eval", "db.adminCommand('ping')"},
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
			ErrorDetails: "MongoDB engine failed to become ready inside ephemeral container",
		}, nil
	}

	// 1. Inyectar init_script
	if strings.TrimSpace(config.InitScript) != "" {
		if err := s.execMongo(ctx, containerID, config.InitScript); err != nil {
			return domain.DBEvaluationResult{
				Verdict:      domain.VerdictRE,
				ErrorDetails: fmt.Sprintf("Error al ejecutar init_script NoSQL en MongoDB: %v", err),
			}, nil
		}
	}

	// 2. Inyectar solución de estudiante/docente
	if strings.TrimSpace(config.SolutionSQL) != "" {
		if err := s.execMongo(ctx, containerID, config.SolutionSQL); err != nil {
			return domain.DBEvaluationResult{
				Verdict:      domain.VerdictRE,
				ErrorDetails: fmt.Sprintf("Error de ejecución de script NoSQL en MongoDB: %v", err),
			}, nil
		}
	}

	// 3. Extracción de estado NoSQL serializado en JSON
	valQuery := strings.TrimSpace(config.ValidationQuery)
	if valQuery == "" {
		valQuery = "db.getCollectionNames()"
	}

	jsonWrappedScript := fmt.Sprintf("EJSON.stringify(%s)", strings.TrimRight(valQuery, ";"))
	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"mongosh", "--quiet", "evaldb", "--eval", jsonWrappedScript},
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
			ErrorDetails: fmt.Sprintf("Error al ejecutar validation_query en MongoDB: %s", errStr),
		}, nil
	}

	execTime := time.Since(startTime)
	return domain.DBEvaluationResult{
		Verdict:       domain.VerdictAC,
		ExecutionTime: execTime,
		ResultingJSON: strings.TrimSpace(stdoutBuf.String()),
	}, nil
}

func (s *MongoStrategy) execMongo(ctx context.Context, containerID string, jsScript string) error {
	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"mongosh", "--quiet", "evaldb", "--eval", jsScript},
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
		return fmt.Errorf("MongoDB Exit Code %d: %s", inspect.ExitCode, errStr)
	}

	return nil
}
