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
	if memLimit < 512 {
		memLimit = 512
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

	// Ready check via container logs: espera el mensaje "waiting for connections" que mongod
	// imprime cuando está listo. No ejecuta procesos adicionales dentro del container.
	ready := false
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		logsOpts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "50",
		}
		logReader, err := s.cli.ContainerLogs(ctx, containerID, logsOpts)
		if err == nil {
			var logBuf bytes.Buffer
			_, _ = io.Copy(&logBuf, logReader)
			logReader.Close()
			if strings.Contains(logBuf.String(), "Waiting for connections") ||
				strings.Contains(logBuf.String(), "waiting for connections") {
				ready = true
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		return domain.DBEvaluationResult{
			Verdict:      domain.VerdictRE,
			ErrorDetails: "MongoDB engine failed to become ready inside ephemeral container",
		}, nil
	}

	time.Sleep(500 * time.Millisecond)

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
	// print(EJSON.stringify(...)) fuerza la iteración del cursor y cierra el stream correctamente.
	// --json=relaxed produce un cursor lazy que nunca hace EOF al stream de Docker exec.
	time.Sleep(500 * time.Millisecond)

	valQuery := strings.TrimSpace(config.ValidationQuery)
	if valQuery == "" {
		valQuery = "db.getCollectionNames()"
	}

	jsonScript := fmt.Sprintf("print(EJSON.stringify(%s))", strings.TrimRight(valQuery, ";"))
	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"mongosh", "--quiet", "mongodb://127.0.0.1:27017/evaldb?serverSelectionTimeoutMS=10000", "--eval", jsonScript},
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

	var stdoutBuf, stderrBuf bytes.Buffer
	_ = copyWithTimeout(&stdoutBuf, &stderrBuf, attach.Reader, &attach, 120*time.Second)

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
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
			Cmd:          []string{"mongosh", "--quiet", "mongodb://127.0.0.1:27017/evaldb?serverSelectionTimeoutMS=10000", "--eval", jsScript},
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

		var stdoutBuf, stderrBuf bytes.Buffer
		_ = copyWithTimeout(&stdoutBuf, &stderrBuf, attach.Reader, &attach, 120*time.Second)

		inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return err
		}

		if inspect.ExitCode == 0 {
			return nil
		}

		errStr := strings.TrimSpace(stderrBuf.String())
		if errStr == "" {
			errStr = strings.TrimSpace(stdoutBuf.String())
		}
		lastErr = fmt.Errorf("MongoDB Exit Code %d: %s", inspect.ExitCode, errStr)
		if strings.Contains(errStr, "Server selection timed out") || strings.Contains(errStr, "MongoServerSelectionError") || strings.Contains(errStr, "ECONNREFUSED") || strings.Contains(errStr, "MongoNetworkError") || strings.Contains(errStr, "connect") {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	return lastErr
}
