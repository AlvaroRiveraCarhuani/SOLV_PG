package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type CStrategy struct {
	cli *client.Client
}

func NewCStrategy(cli *client.Client) domain.LanguageStrategy {
	return &CStrategy{cli: cli}
}

func (s *CStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "gcc -O2 /runner/solution.c -o /tmp/sol && /tmp/sol"}
	return runContainerExecution(ctx, s.cli, "gcc:latest", "solution.c", cmd, config)
}
