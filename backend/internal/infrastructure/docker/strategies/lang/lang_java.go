package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type JavaStrategy struct {
	cli *client.Client
}

// JavaImage is the pinned runner image (immutable digest; mutable tags banned).
const JavaImage = "eclipse-temurin:21-alpine@sha256:1ff763083f2993d57d0bf374ab10bb3e2cb873af6c13a04458ebbd3e0337dc76"

func NewJavaStrategy(cli *client.Client) domain.LanguageStrategy {
	return &JavaStrategy{cli: cli}
}

func (s *JavaStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "javac -d /tmp /runner/Solution.java && java -cp /tmp Solution"}
	return runContainerExecution(ctx, s.cli, JavaImage, "Solution.java", cmd, config)
}
