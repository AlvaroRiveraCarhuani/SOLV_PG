package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type JavaScriptStrategy struct {
	cli *client.Client
}

// JavaScriptImage is the pinned runner image (immutable digest; mutable tags banned).
const JavaScriptImage = "node:20-alpine@sha256:fb4cd12c85ee03686f6af5362a0b0d56d50c58a04632e6c0fb8363f609372293"

func NewJavaScriptStrategy(cli *client.Client) domain.LanguageStrategy {
	return &JavaScriptStrategy{cli: cli}
}

func (s *JavaScriptStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return runContainerExecution(ctx, s.cli, JavaScriptImage, "solution.js", []string{"node", "/runner/solution.js"}, config)
}
