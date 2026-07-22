package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type CSharpStrategy struct {
	cli *client.Client
}

func NewCSharpStrategy(cli *client.Client) domain.LanguageStrategy {
	return &CSharpStrategy{cli: cli}
}

func (s *CSharpStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "mcs /runner/solution.cs -out:/tmp/sol.exe && mono /tmp/sol.exe"}
	return runContainerExecution(ctx, s.cli, "mono:latest", "solution.cs", cmd, config)
}
