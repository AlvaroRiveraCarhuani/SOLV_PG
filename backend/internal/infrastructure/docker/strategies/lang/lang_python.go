package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type PythonStrategy struct {
	cli *client.Client
}

func NewPythonStrategy(cli *client.Client) domain.LanguageStrategy {
	return &PythonStrategy{cli: cli}
}

func (s *PythonStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return runContainerExecution(ctx, s.cli, "python:3.11-slim", "solution.py", []string{"python", "/runner/solution.py"}, config)
}
