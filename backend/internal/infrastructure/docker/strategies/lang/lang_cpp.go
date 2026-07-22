package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type CppStrategy struct {
	cli *client.Client
}

func NewCppStrategy(cli *client.Client) domain.LanguageStrategy {
	return &CppStrategy{cli: cli}
}

func (s *CppStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "g++ -O2 /runner/solution.cpp -o /tmp/sol && /tmp/sol"}
	return runContainerExecution(ctx, s.cli, "gcc:latest", "solution.cpp", cmd, config)
}
