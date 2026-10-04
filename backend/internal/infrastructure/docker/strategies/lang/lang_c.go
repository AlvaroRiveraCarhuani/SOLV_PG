package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type CStrategy struct {
	cli *client.Client
}

// CImage is the pinned runner image (immutable digest; mutable tags banned).
const CImage = "gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91"

func NewCStrategy(cli *client.Client) domain.LanguageStrategy {
	return &CStrategy{cli: cli}
}

func (s *CStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "gcc -O2 /runner/solution.c -o /tmp/sol && /tmp/sol"}
	return runContainerExecution(ctx, s.cli, CImage, "solution.c", cmd, config)
}
