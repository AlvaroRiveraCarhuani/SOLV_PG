package lang

import (
	"context"
	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type JavaStrategy struct {
	cli *client.Client
}

func NewJavaStrategy(cli *client.Client) domain.LanguageStrategy {
	return &JavaStrategy{cli: cli}
}

func (s *JavaStrategy) ExecuteTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	cmd := []string{"sh", "-c", "javac -d /tmp /runner/Solution.java && java -cp /tmp Solution"}
	return runContainerExecution(ctx, s.cli, "eclipse-temurin:21-alpine", "Solution.java", cmd, config)
}
