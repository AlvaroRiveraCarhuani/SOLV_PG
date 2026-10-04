package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type CSharpStrategy struct {
	cli *client.Client
}

// CSharpImage is the pinned runner image (immutable digest; mutable tags banned).
const CSharpImage = "mono@sha256:34d816779b1248b5cfd095770b64ecbaf1798e2aca693a91c11a018dce9c7ad5"

func NewCSharpStrategy(cli *client.Client) domain.LanguageStrategy {
	return &CSharpStrategy{cli: cli}
}

func (s *CSharpStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "mcs /runner/solution.cs -out:/tmp/sol.exe && mono /tmp/sol.exe"}
	return runContainerExecution(ctx, s.cli, CSharpImage, "solution.cs", cmd, config)
}
