package docker

import (
	"fmt"
	"strings"

	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
	dbstrategy "solv-backend/internal/infrastructure/docker/strategies/db"
	langstrategy "solv-backend/internal/infrastructure/docker/strategies/lang"
)

type DBStrategyFactory struct {
	strategies map[string]domain.DBEngineStrategy
}

func NewDBStrategyFactory(cli *client.Client) *DBStrategyFactory {
	postgres := dbstrategy.NewPostgresStrategy(cli)
	mysql := dbstrategy.NewMySQLStrategy(cli)
	mongo := dbstrategy.NewMongoStrategy(cli)

	return &DBStrategyFactory{
		strategies: map[string]domain.DBEngineStrategy{
			"postgres":   postgres,
			"postgresql": postgres,
			"mysql":      mysql,
			"mariadb":    mysql,
			"mongo":      mongo,
			"mongodb":    mongo,
		},
	}
}

func (f *DBStrategyFactory) Get(engine string) (domain.DBEngineStrategy, error) {
	key := strings.ToLower(strings.TrimSpace(engine))
	strategy, ok := f.strategies[key]
	if !ok {
		return nil, fmt.Errorf("unsupported database engine strategy: %q", engine)
	}
	return strategy, nil
}

type LanguageStrategyFactory struct {
	strategies map[string]domain.LanguageStrategy
}

func NewLanguageStrategyFactory(cli *client.Client) *LanguageStrategyFactory {
	py := langstrategy.NewPythonStrategy(cli)
	c := langstrategy.NewCStrategy(cli)
	cpp := langstrategy.NewCppStrategy(cli)
	cs := langstrategy.NewCSharpStrategy(cli)
	java := langstrategy.NewJavaStrategy(cli)
	js := langstrategy.NewJavaScriptStrategy(cli)

	return &LanguageStrategyFactory{
		strategies: map[string]domain.LanguageStrategy{
			"python":     py,
			"py":         py,
			"c":          c,
			"cpp":        cpp,
			"c++":        cpp,
			"csharp":     cs,
			"c#":         cs,
			"cs":         cs,
			"java":       java,
			"javascript": js,
			"js":         js,
			"node":       js,
		},
	}
}

func (f *LanguageStrategyFactory) Get(language string) (domain.LanguageStrategy, error) {
	key := strings.ToLower(strings.TrimSpace(language))
	strategy, ok := f.strategies[key]
	if !ok {
		return nil, fmt.Errorf("unsupported programming language strategy: %q", language)
	}
	return strategy, nil
}
