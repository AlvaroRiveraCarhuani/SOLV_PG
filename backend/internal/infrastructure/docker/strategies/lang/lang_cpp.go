package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type CppStrategy struct {
	cli *client.Client
}

// CppImage is the pinned runner image (immutable digest; mutable tags banned).
const CppImage = "gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91"

func NewCppStrategy(cli *client.Client) domain.LanguageStrategy {
	return &CppStrategy{cli: cli}
}

func (s *CppStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "g++ -O2 /runner/solution.cpp -o /tmp/sol && /tmp/sol"}
	return runContainerExecution(ctx, s.cli, CppImage, "solution.cpp", cmd, config)
}
