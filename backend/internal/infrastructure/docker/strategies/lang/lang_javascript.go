package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type JavaScriptStrategy struct {
	cli *client.Client
}

func NewJavaScriptStrategy(cli *client.Client) domain.LanguageStrategy {
	return &JavaScriptStrategy{cli: cli}
}

func (s *JavaScriptStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return runContainerExecution(ctx, s.cli, "node:20-alpine", "solution.js", []string{"node", "/runner/solution.js"}, config)
}
