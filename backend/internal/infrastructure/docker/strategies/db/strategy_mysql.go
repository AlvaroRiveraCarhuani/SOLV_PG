package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"solv-backend/internal/core/domain"
)

type MySQLStrategy struct {
	cli *client.Client
}

func NewMySQLStrategy(cli *client.Client) domain.DBEngineStrategy {
	return &MySQLStrategy{cli: cli}
}

func (s *MySQLStrategy) ExecuteDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	res, err := s.ExecuteEvaluation(ctx, config)
	if err != nil {
		return "", err
	}
	if res.Verdict != domain.VerdictAC {
		return "", fmt.Errorf("mysql dry run failed with verdict %s: %s", res.Verdict, res.ErrorDetails)
	}
	return res.ResultingJSON, nil
}

func (s *MySQLStrategy) ExecuteEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	imageName := "mysql:8.4"

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
		Cmd: []string{
			"mysqld",
			"--innodb-buffer-pool-size=16M",
			"--performance-schema=OFF",
		},
		Env: []string{
			"MYSQL_ROOT_PASSWORD=evalpass",
			"MYSQL_DATABASE=evaldb",
		},
	}

	resp, err := s.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return domain.DBEvaluationResult{}, fmt.Errorf("failed to create MySQL container: %w", err)
	}
	containerID := resp.ID

	defer func() {
		_ = s.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	startTime := time.Now()
	if err := s.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return domain.DBEvaluationResult{}, fmt.Errorf("failed to start MySQL container: %w", err)
	}

	// Ready check: espera a que la BD evaldb esté disponible en MySQL
	var lastErrStr string
	ready := false
	for i := 0; i < 100; i++ {
		execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
			Cmd:          []string{"mysql", "-u", "root", "-pevalpass", "evaldb", "-e", "SELECT 1;"},
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
				if err == nil && !inspect.Running && inspect.ExitCode == 0 {
					ready = true
					break
				} else {
					lastErrStr = fmt.Sprintf("exit=%d out=%s err=%s", inspect.ExitCode, strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()))
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	if !ready {
		return domain.DBEvaluationResult{
			Verdict:      domain.VerdictRE,
			ErrorDetails: fmt.Sprintf("MySQL engine failed to become ready inside ephemeral container (last check: %s)", lastErrStr),
		}, nil
	}

	// 1. Inyectar init_script
	if strings.TrimSpace(config.InitScript) != "" {
		if err := s.execMySQL(ctx, containerID, config.InitScript); err != nil {
			return domain.DBEvaluationResult{
				Verdict:      domain.VerdictRE,
				ErrorDetails: fmt.Sprintf("Error al ejecutar init_script en MySQL: %v", err),
			}, nil
		}
	}

	// 2. Inyectar solución SQL
	if strings.TrimSpace(config.SolutionSQL) != "" {
		if err := s.execMySQL(ctx, containerID, config.SolutionSQL); err != nil {
			return domain.DBEvaluationResult{
				Verdict:      domain.VerdictRE,
				ErrorDetails: fmt.Sprintf("Error de sintaxis o ejecución SQL en MySQL: %v", err),
			}, nil
		}
	}

	// 3. Extracción de estado en JSON usando mysql -B (batch TSV) y conversión a JSON
	valQuery := strings.TrimSpace(config.ValidationQuery)
	if valQuery == "" {
		valQuery = "SELECT 1;"
	}

	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"mysql", "-u", "root", "-pevalpass", "evaldb", "-B", "-e", valQuery},
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
			ErrorDetails: fmt.Sprintf("Error al ejecutar validation_query en MySQL: %s", errStr),
		}, nil
	}

	jsonRes, err := tsvToJSON(stdoutBuf.String())
	if err != nil {
		return domain.DBEvaluationResult{
			Verdict:      domain.VerdictRE,
			ErrorDetails: fmt.Sprintf("Error al formatear resultado de MySQL a JSON: %v", err),
		}, nil
	}

	execTime := time.Since(startTime)
	return domain.DBEvaluationResult{
		Verdict:       domain.VerdictAC,
		ExecutionTime: execTime,
		ResultingJSON: jsonRes,
	}, nil
}

func (s *MySQLStrategy) execMySQL(ctx context.Context, containerID string, sqlCommand string) error {
	execResp, err := s.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"mysql", "-u", "root", "-pevalpass", "evaldb", "-e", sqlCommand},
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
		return fmt.Errorf("MySQL Exit Code %d: %s", inspect.ExitCode, errStr)
	}

	return nil
}

func tsvToJSON(tsvStr string) (string, error) {
	lines := strings.Split(strings.TrimSpace(tsvStr), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "[]", nil
	}
	headers := strings.Split(lines[0], "\t")
	var records []map[string]interface{}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		row := make(map[string]interface{})
		for i, header := range headers {
			val := ""
			if i < len(cols) {
				val = cols[i]
			}
			if val == "NULL" {
				row[header] = nil
			} else if num, err := strconv.ParseInt(val, 10, 64); err == nil {
				row[header] = num
			} else if flt, err := strconv.ParseFloat(val, 64); err == nil {
				row[header] = flt
			} else {
				row[header] = val
			}
		}
		records = append(records, row)
	}
	if len(records) == 0 {
		return "[]", nil
	}
	bytes, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
