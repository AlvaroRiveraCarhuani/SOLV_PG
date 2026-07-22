package domain

import "context"

type LabContainerConfig struct {
	Image         string
	ContainerName string
	VolumeName    string
	MemoryLimitMB int64
	NetworkMode   string
	ReadOnly      bool
	Labels        map[string]string
}

type ContainerOrchestrator interface {
	EnsureVolumeExists(ctx context.Context, volumeName string) error
	ExecuteDryRun(ctx context.Context, image string) (int64, error)
	StartContainer(ctx context.Context, config LabContainerConfig) (string, error)
	HibernateContainer(ctx context.Context, containerID string) error
	StopAndRemoveContainer(ctx context.Context, containerID string) error
}

type Template struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	DockerImage string `db:"docker_image" json:"docker_image"`
	BaseRamMB   int    `db:"base_ram_mb" json:"base_ram_mb"`
}

type TemplateRepository interface {
	GetTemplateByID(ctx context.Context, id string) (*Template, error)
}

type ExerciseRepository interface {
	GetByID(ctx context.Context, id string) (*Exercise, error)
	Create(ctx context.Context, exercise *Exercise) error
	UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error
}

type ASTAnalyzer interface {
	ValidateCode(language string, sourceCode string, rules ASTRules) (bool, string)
}

type LanguageStrategy interface {
	ExecuteTestCase(ctx context.Context, config EvaluationRunConfig) (TestCaseRunResult, error)
}

type DBEngineStrategy interface {
	ExecuteDryRun(ctx context.Context, config DBEvaluationRunConfig) (string, error)
	ExecuteEvaluation(ctx context.Context, config DBEvaluationRunConfig) (DBEvaluationResult, error)
}

type EvaluationRunner interface {
	RunTestCase(ctx context.Context, config EvaluationRunConfig) (TestCaseRunResult, error)
	RunDBDryRun(ctx context.Context, config DBEvaluationRunConfig) (string, error)
	RunDBEvaluation(ctx context.Context, config DBEvaluationRunConfig) (DBEvaluationResult, error)
}

type WorkspaceRepository interface {
	GetByStudentAndSubject(ctx context.Context, studentID string, subjectID string) (*WorkspaceInstance, error)
	Create(ctx context.Context, workspace *WorkspaceInstance) error
	UpdateContainerID(ctx context.Context, id string, containerID string) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

type WorkspaceOrchestrator interface {
	EnsureVolumeExists(ctx context.Context, volumeName string) error
	EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error
	StartWorkspaceContainer(ctx context.Context, config WorkspaceContainerConfig) (string, error)
	StopAndRemoveContainer(ctx context.Context, containerID string) error
}
