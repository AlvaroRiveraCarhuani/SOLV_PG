package domain

import (
	"context"
	"encoding/json"
	"time"
)

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
	ID             string         `db:"id" json:"id"`
	Name           string         `db:"name" json:"name"`
	DockerImage    string         `db:"docker_image" json:"docker_image"`
	BaseRamMB      int            `db:"base_ram_mb" json:"base_ram_mb"`
	ServicesConfig ServicesConfig `db:"services_config" json:"services_config"`
	SetupScript    string         `db:"setup_script" json:"setup_script"`
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
	GetByIDAndTenant(ctx context.Context, id, tenantID string) (*Exercise, error)
	Create(ctx context.Context, exercise *Exercise) error
	Update(ctx context.Context, exercise *Exercise) error
	UpdateStatus(ctx context.Context, id, tenantID, status string) error
	UpdateConfig(ctx context.Context, id, tenantID string, config ExerciseConfig) error
	UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error
	ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*DueAssignment, error)
	ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*Exercise, error)
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
	PauseContainer(ctx context.Context, containerID string) error
	UnpauseContainer(ctx context.Context, containerID string) error
	ListAllManagedContainers(ctx context.Context) ([]string, error)
	RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error)
	GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error)
	ExecuteCommandInBackground(ctx context.Context, containerID string, workDir string, cmd []string) error
}

type TenantRepository interface {
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	GetAll(ctx context.Context) ([]*Tenant, error)
	UpdateConfig(ctx context.Context, id string, config []byte) error
	SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error
	GetMaintenance(ctx context.Context, tenantID string) (*MaintenanceStatus, error)
}

type AcademicPeriodRepository interface {
	Create(ctx context.Context, period *AcademicPeriod) error
	GetByID(ctx context.Context, tenantID, id string) (*AcademicPeriod, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*AcademicPeriod, error)
	Update(ctx context.Context, period *AcademicPeriod) error
	Delete(ctx context.Context, tenantID, id string) error
	ArchiveExpiredPeriods(ctx context.Context) (int64, error)
	// Archive congela formalmente el periodo (is_archived=true, archived_at/by)
	// y sella sus materias con is_archived=true en la misma transacción (ADR-029).
	Archive(ctx context.Context, tenantID, id, archivedBy string) error
}

type SubjectRepository interface {
	Create(ctx context.Context, subject *Subject) error
	GetByID(ctx context.Context, tenantID, id string) (*Subject, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*Subject, error)
	EnrollStudent(ctx context.Context, enrollment *Enrollment) error
	ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error)
	ListByStudent(ctx context.Context, tenantID, studentID string) ([]*Subject, error)
	ReassignTeacher(ctx context.Context, tenantID, subjectID, newTeacherID string) error
	ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error
	Update(ctx context.Context, tenantID, subjectID, name, code string) error
	GetTemplateStatus(ctx context.Context, templateID string) (string, error)
}

type AdminGovernanceRepository interface {
	ListStudentsDirectory(ctx context.Context, tenantID, search, subjectID, status, periodID string) ([]*AdminStudentDirectoryItem, error)
	GetStudentCourses(ctx context.Context, tenantID, studentID string) ([]*AdminStudentCourseItem, error)
	CreateStudent(ctx context.Context, tenantID, email, firstName, lastName string) (*AdminStudentDirectoryItem, error)
	UpdateStudentStatus(ctx context.Context, tenantID, studentID, status, reason string) error
	ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error)
	ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error)
	ListTemplates(ctx context.Context, tenantID, status, search string) ([]*AdminTemplateReviewItem, error)
	ReviewTemplate(ctx context.Context, tenantID, templateID, adminID, status, rejectionReason string, baseRamMB *int) (*AdminTemplateReviewItem, error)
	CreateOfficialTemplate(ctx context.Context, tenantID, adminID string, dto CreateOfficialTemplateDTO) (*AdminTemplateReviewItem, error)
	ListPendingAuditTemplates(ctx context.Context) ([]*AdminTemplateReviewItem, error)
	UpdateAuditResults(ctx context.Context, templateID string, smokeStatus, smokeOutput, secStatus string, cveCritical, cveHigh int, secReportJSON []byte, finalStatus string) error
	DuplicateTemplate(ctx context.Context, tenantID, templateID, adminID string) (*AdminTemplateReviewItem, error)
	UpdateEOLStatus(ctx context.Context, templateID string, status, eolDate, message string) error
	TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error)
	HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error)
	ListTemplateCategories(ctx context.Context, tenantID string) ([]*TemplateCategory, error)
	CreateTemplateCategory(ctx context.Context, tenantID string, dto CreateCategoryDTO) (*TemplateCategory, error)
	UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto UpdateCategoryDTO) (*TemplateCategory, error)
	DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error
	ReorderTemplateCategories(ctx context.Context, tenantID string, items []ReorderCategoryItemDTO) error
	ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*TemplateModelItemDTO, error)
	UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto UpdateTemplateModelDTO) (*TemplateModelItemDTO, error)
	SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error
	PromoteTemplateToModel(ctx context.Context, tenantID, templateID, adminID string, dto PromoteTemplateToModelDTO) (*TemplateModelItemDTO, error)
	SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*TemplateDraft, error)
	GetDraftByUser(ctx context.Context, tenantID, userID string) (*TemplateDraft, error)
	DeleteDraft(ctx context.Context, tenantID, userID string) error
	GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error)
}

type SubmissionRepository interface {
	Create(ctx context.Context, submission *Submission) error
	GetByID(ctx context.Context, tenantID, id string) (*Submission, error)
	ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*Submission, error)
	ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*Submission, error)
	ListByStudent(ctx context.Context, tenantID, studentID string) ([]*Submission, error)
	UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error
}

type TeacherInvitationRepository interface {
	Create(ctx context.Context, invitation *TeacherInvitation) error
	GetByToken(ctx context.Context, tenantID, token string) (*TeacherInvitation, error)
	GetByID(ctx context.Context, tenantID, id string) (*TeacherInvitation, error)
	Update(ctx context.Context, invitation *TeacherInvitation) error
	AcceptInvitationTx(ctx context.Context, tenantID, token, userID, userEmail string) error
	ListTeachers(ctx context.Context, tenantID, search, status, origin string) ([]*TeacherListItem, error)
	GetTeacherCourses(ctx context.Context, tenantID, teacherID string) ([]*TeacherCourseItem, error)
	DeleteInvitation(ctx context.Context, tenantID, id string) error
	DeleteTeacher(ctx context.Context, tenantID, teacherID string) error
}

type TeacherRepository interface {
	GetCoursesSummary(ctx context.Context, tenantID, teacherID string) ([]*TeacherCourseSummary, error)
	GetAttentionWidget(ctx context.Context, tenantID, teacherID string) (*TeacherAttentionWidget, error)
	GetCourseLabsStats(ctx context.Context, tenantID, teacherID, subjectID string) ([]*TeacherLabStats, error)
	ListCourseSubmissions(ctx context.Context, tenantID, teacherID, subjectID, exerciseID, verdict string) ([]*SubmissionQueueItem, error)
	GetTeacherSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string) (*TeacherSubmissionReviewDTO, error)
	AddComment(ctx context.Context, comment *SubmissionComment) error
	GetCommentsBySubmission(ctx context.Context, tenantID, submissionID string) ([]*SubmissionComment, error)
	GetCourseGradesMatrix(ctx context.Context, tenantID, teacherID, subjectID string) (*CourseGradesMatrix, error)
	GetExerciseSubmissionsForPlagiarism(ctx context.Context, tenantID, teacherID, subjectID, exerciseID string) ([]*SubmissionForPlagiarism, error)
	ListLiveWorkspaceSessions(ctx context.Context, tenantID, teacherID string) ([]*LiveWorkspaceSession, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	CreateBatch(ctx context.Context, list []*Notification) error
	ListByRecipient(ctx context.Context, tenantID, recipientUserID string, unreadOnly bool, limit, offset int) ([]*Notification, int64, error)
	CountUnread(ctx context.Context, tenantID, recipientUserID string) (int64, error)
	MarkAsRead(ctx context.Context, tenantID, recipientUserID, notificationID string) error
	MarkAllAsRead(ctx context.Context, tenantID, recipientUserID string) (int64, error)
}

type BackupRepository interface {
	GetConfig(ctx context.Context, tenantID string) (*BackupConfig, error)
	UpsertConfig(ctx context.Context, config *BackupConfig) error
	CreateExecution(ctx context.Context, execution *BackupExecution) error
	UpdateExecution(ctx context.Context, execution *BackupExecution) error
	UpdateExecutionVerifyColumns(ctx context.Context, execution *BackupExecution) error
	GetExecutionByID(ctx context.Context, tenantID, id string) (*BackupExecution, error)
	ListExecutions(ctx context.Context, tenantID string, limit, offset int) ([]*BackupExecution, int64, error)
	GetExpiredExecutions(ctx context.Context, tenantID string, retentionDays int) ([]*BackupExecution, error)
	DeleteExecution(ctx context.Context, id string) error
}

// EnvTestJobRepository gestiona la persistencia y ciclo de vida de los jobs de prueba de entorno
type EnvTestJobRepository interface {
	Save(ctx context.Context, job *EnvTestJob) error
	GetByID(ctx context.Context, id string) (*EnvTestJob, error)
	UpdateProgress(ctx context.Context, id string, progress EnvTestProgress) error
	Complete(ctx context.Context, id string, result *EnvTestResult) error
	Fail(ctx context.Context, id string, errorCode, errorMessage string) error
	Cancel(ctx context.Context, id string) error
	ListActive(ctx context.Context) ([]*EnvTestJob, error)
}

// ImageRegistryPort abstrae las operaciones sobre registros OCI y almacenamiento local de imágenes
type ImageRegistryPort interface {
	InspectLocal(ctx context.Context, imageRef string) (isLocal bool, sizeBytes int64, localDigest string, err error)
	InspectRemoteDigest(ctx context.Context, imageRef string) (remoteDigest string, err error)
	PullImage(ctx context.Context, imageRef string, onProgress func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string)) error
}

// ContainerRunnerPort abstrae la ejecución efímera de contenedores para pruebas de entorno
type ContainerRunnerPort interface {
	RunSmokeTest(ctx context.Context, imageRef string, tools []string, memoryLimitMB int64) (results []ToolResult, exitCode int, err error)
	RunJudgeSmokeTest(ctx context.Context, imageRef string, entrypoint string, sampleInput string, timeoutMS int, memoryLimitMB int64) (output string, durationMs int64, exitCode int, err error)
}
