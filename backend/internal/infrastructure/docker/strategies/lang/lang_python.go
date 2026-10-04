package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type PythonStrategy struct {
	cli *client.Client
}

// PythonImage is the pinned runner image (immutable digest; mutable tags banned).
const PythonImage = "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93"

func NewPythonStrategy(cli *client.Client) domain.LanguageStrategy {
	return &PythonStrategy{cli: cli}
}

func (s *PythonStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return runContainerExecution(ctx, s.cli, PythonImage, "solution.py", []string{"python", "/runner/solution.py"}, config)
}
