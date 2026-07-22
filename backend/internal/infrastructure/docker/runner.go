package docker

import (
	"context"

	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

type DockerEvaluationRunner struct {
	cli         *client.Client
	langFactory *LanguageStrategyFactory
	dbFactory   *DBStrategyFactory
}

func NewDockerEvaluationRunner(cli *client.Client) domain.EvaluationRunner {
	return &DockerEvaluationRunner{
		cli:         cli,
		langFactory: NewLanguageStrategyFactory(cli),
		dbFactory:   NewDBStrategyFactory(cli),
	}
}

func (r *DockerEvaluationRunner) RunTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	strategy, err := r.langFactory.Get(config.Language)
	if err != nil {
		return domain.TestCaseRunResult{}, err
	}
	return strategy.ExecuteTestCase(ctx, config)
}

func (r *DockerEvaluationRunner) RunDBDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	strategy, err := r.dbFactory.Get(config.Engine)
	if err != nil {
		return "", err
	}
	return strategy.ExecuteDryRun(ctx, config)
}

func (r *DockerEvaluationRunner) RunDBEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	strategy, err := r.dbFactory.Get(config.Engine)
	if err != nil {
		return domain.DBEvaluationResult{}, err
	}
	return strategy.ExecuteEvaluation(ctx, config)
}
