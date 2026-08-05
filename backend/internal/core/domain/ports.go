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

type LabTemplateRepository interface {
	GetBySignatureHash(ctx context.Context, signatureHash string) (*LabTemplate, error)
	CreateOrUpdateProfile(ctx context.Context, template *LabTemplate) error
	UpdateProfileAtomic(ctx context.Context, signatureHash string, sampleMB float64) (*LabTemplate, error)
}

type EWMAProfilerService interface {
	CalculateSignatureHash(baseImage string, setupScript string) string
	RecordSessionPeakAndRecalculate(ctx context.Context, baseImage string, setupScript string, peakRAMMB float64) (*ResourceProfile, error)
}

type ExerciseRepository interface {
	GetByID(ctx context.Context, id string) (*Exercise, error)
	Create(ctx context.Context, exercise *Exercise) error
	UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error
}

type ASTAnalyzer interface {
	ValidateCode(language string, sourceCode string, rules ASTRules) (bool, string)
}

// CodeScanner runs Semgrep CLI against source code and returns structured violations
type CodeScanner interface {
	ScanCode(code string, language string) (*ScanResult, error)
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

type HostMonitor interface {
	GetHostMemoryStats() (freePct float64, availableMB uint64, err error)
	CanAllocateMemory(requiredMB int64) bool
}

type WorkspaceRepository interface {
	GetByStudentAndSubject(ctx context.Context, studentID string, subjectID string) (*WorkspaceInstance, error)
	GetByID(ctx context.Context, id string) (*WorkspaceInstance, error)
	Create(ctx context.Context, workspace *WorkspaceInstance) error
	UpdateContainerID(ctx context.Context, id string, containerID string) error
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateMemoryLimit(ctx context.Context, id string, memoryMB int64) error
	RecordHeartbeat(ctx context.Context, id string) error
	IncrementOOMStrike(ctx context.Context, id string) error
	ResetOOMStrikes(ctx context.Context, id string) error
	GetActiveWorkspaces(ctx context.Context) ([]*WorkspaceInstance, error)
	GetAllRunningWorkspaces(ctx context.Context) ([]*WorkspaceInstance, error)
	GetByType(ctx context.Context, workspaceType string) ([]*WorkspaceInstance, error)
	SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error
}

type WorkspaceOrchestrator interface {
	EnsureVolumeExists(ctx context.Context, volumeName string) error
	EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error
	StartWorkspaceContainer(ctx context.Context, config WorkspaceContainerConfig) (string, error)
	UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error
	GetContainerMetrics(ctx context.Context, containerID string) (*ContainerMetrics, error)
	StopAndRemoveContainer(ctx context.Context, containerID string) error
	ListAllManagedContainers(ctx context.Context) ([]string, error)
	RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error)
}

type TenantRepository interface {
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	GetAll(ctx context.Context) ([]*Tenant, error)
}

type SubjectRepository interface {
	Create(ctx context.Context, subject *Subject) error
	GetByID(ctx context.Context, tenantID, id string) (*Subject, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*Subject, error)
	EnrollStudent(ctx context.Context, enrollment *Enrollment) error
	ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error)
}

type SubmissionRepository interface {
	Create(ctx context.Context, submission *Submission) error
	GetByID(ctx context.Context, tenantID, id string) (*Submission, error)
	ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*Submission, error)
	ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*Submission, error)
}

type TeacherInvitationRepository interface {
	Create(ctx context.Context, invitation *TeacherInvitation) error
	GetByToken(ctx context.Context, tenantID, token string) (*TeacherInvitation, error)
	AcceptInvitationTx(ctx context.Context, tenantID, token, userID, userEmail string) error
}


