package domain

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type ExerciseType string

const (
	ExerciseTypeAlgorithm ExerciseType = "algorithm"
	ExerciseTypeDatabase  ExerciseType = "database"
	ExerciseTypeProject   ExerciseType = "project"
)

type EnvironmentType string

const (
	EnvironmentTypeJuezEfimero    EnvironmentType = "JUEZ_EFIMERO"
	EnvironmentTypeIDEPersistente EnvironmentType = "IDE_PERSISTENTE"
)

// Política de red del entorno de examen. `none` lo encierra en una red interna
// sin puerta de enlace: el editor sigue alcanzable por Traefik y la salida a
// internet no existe. `egress` es la excepción declarada por el docente.
const (
	ExamNetworkNone   = "none"
	ExamNetworkEgress = "egress"
)

var (
	ErrExamNetworkInvalid        = errors.New("invalid exam network policy")
	ErrExamNetworkReasonRequired = errors.New("exam network egress requires a reason")
	ErrExamNetworkRequiresExam   = errors.New("exam network policy applies only to exam exercises")
)

// NormalizeExamNetwork convierte vacío en `none` y recorta espacios.
func NormalizeExamNetwork(network string) string {
	trimmed := strings.TrimSpace(network)
	if trimmed == "" {
		return ExamNetworkNone
	}
	return trimmed
}

// ValidateExamNetwork exige que la excepción `egress` sea explícita, tenga
// motivo y pertenezca a un ejercicio de examen. La salida cerrada es el
// default; lo que se declara es la excepción.
func ValidateExamNetwork(purpose, network, reason string) error {
	switch NormalizeExamNetwork(network) {
	case ExamNetworkNone:
		return nil
	case ExamNetworkEgress:
		if purpose != string(ExercisePurposeExam) {
			return ErrExamNetworkRequiresExam
		}
		if len(strings.TrimSpace(reason)) < 10 {
			return ErrExamNetworkReasonRequired
		}
		return nil
	default:
		return ErrExamNetworkInvalid
	}
}

type ExercisePurpose string

const (
	ExercisePurposeClass ExercisePurpose = "class"
	ExercisePurposeExam  ExercisePurpose = "exam"
)

type ExerciseDifficulty string

const (
	ExerciseDifficultyEasy   ExerciseDifficulty = "easy"
	ExerciseDifficultyMedium ExerciseDifficulty = "medium"
	ExerciseDifficultyHard   ExerciseDifficulty = "hard"
)

type TestCaseVisibility string

const (
	TestCaseVisibilityExample TestCaseVisibility = "example"
	TestCaseVisibilityPublic  TestCaseVisibility = "public"
	TestCaseVisibilityHidden  TestCaseVisibility = "hidden"
)

var (
	ErrSeedRequiresExamPurpose     = errors.New("per_student_seed is only allowed when purpose is 'exam'")
	ErrInvalidVisibility           = errors.New("invalid test case visibility: must be 'example', 'public', or 'hidden'")
	ErrInvalidWeight               = errors.New("test case weight must be non-negative")
	ErrEmptyExpectedOutput         = errors.New("test case expected_output cannot be empty")
	ErrTestCasesNotArray           = errors.New("test_cases must be an array")
	ErrModuleLocked                = errors.New("MODULE_LOCKED: el módulo curricular se encuentra bloqueado")
	ErrMemoryGovernedByTemplate    = errors.New("memory_governed_by_template: la memoria RAM es gobernada por la plantilla homologada del administrador y no puede ser enviada por el docente")
	ErrTemplateEnvironmentMismatch = errors.New("template_environment_mismatch: el tipo de entorno de la plantilla no coincide con el environment_type del ejercicio")
	// ErrTimeLimitOutOfRange mantiene el mismo rango que el formulario docente
	// (100..30.000 ms): se valida en servidor porque el runner consume
	// config.algorithm.time_limit_ms y la columna time_limit_ms es la única
	// fuente persistente (no hay techos por plantilla todavía; ver ADR-047).
	ErrTimeLimitOutOfRange = errors.New("time_limit_out_of_range: el tiempo límite por caso debe estar entre 100 y 30.000 ms")
)

type Verdict string

const (
	VerdictAC           Verdict = "AC"            // Accepted
	VerdictWA           Verdict = "WA"            // Wrong Answer
	VerdictTLE          Verdict = "TLE"           // Time Limit Exceeded
	VerdictRE           Verdict = "RE"            // Runtime Error
	VerdictCE           Verdict = "CE"            // Compilation Error
	VerdictMLE          Verdict = "MLE"           // Memory Limit Exceeded (OOM kill de cgroups)
	VerdictVE           Verdict = "VE"            // Verdict Error (fallo o timeout del checker docente)
	VerdictASTViolation Verdict = "AST_VIOLATION" // AST Security Violation (regex)
	VerdictASTBlocked   Verdict = "AST_BLOCKED"   // AST Security Violation (Semgrep)
)

// Origen del veredicto guardado en una entrega. 'legacy' agrupa las filas
// históricas que no se pueden atribuir a un camino concreto del código.
const (
	VerdictOriginJudge    = "judge"
	VerdictOriginRubric   = "rubric"
	VerdictOriginOverride = "override"
	VerdictOriginLegacy   = "legacy"
)

// ScanViolation represents a single Semgrep finding
type ScanViolation struct {
	RuleID  string `json:"rule_id"`
	Message string `json:"message"`
	Line    int    `json:"line"`
}

// ScanResult holds the aggregated output of a Semgrep pre-check scan
type ScanResult struct {
	Violations    []ScanViolation `json:"violations"`
	HasViolations bool            `json:"has_violations"`
}

type TestCase struct {
	ID             string             `json:"id,omitempty" db:"id"`
	ExerciseID     string             `json:"exercise_id,omitempty" db:"exercise_id"`
	OrderIndex     int                `json:"order_index" db:"order_index"`
	Input          string             `json:"input" db:"input"`
	ExpectedOutput string             `json:"expected_output" db:"expected_output"`
	Visibility     TestCaseVisibility `json:"visibility" db:"visibility"`
	Weight         float64            `json:"weight" db:"weight"`
	CreatedAt      *time.Time         `json:"created_at,omitempty" db:"created_at"`

	// Derived legacy booleans for transient API compatibility
	IsHidden bool `json:"is_hidden"`
	IsSample bool `json:"is_sample"`
}

func (tc *TestCase) Normalize() {
	if tc.Visibility == "" {
		if tc.IsHidden {
			tc.Visibility = TestCaseVisibilityHidden
		} else if tc.IsSample {
			tc.Visibility = TestCaseVisibilityExample
		} else {
			tc.Visibility = TestCaseVisibilityPublic
		}
	}
	tc.IsHidden = (tc.Visibility == TestCaseVisibilityHidden)
	tc.IsSample = (tc.Visibility == TestCaseVisibilityExample)
	if tc.Weight <= 0 {
		tc.Weight = 1.0
	}
}

func (tc *TestCase) Validate() error {
	if tc.ExpectedOutput == "" {
		return ErrEmptyExpectedOutput
	}
	if tc.Weight < 0 {
		return ErrInvalidWeight
	}
	switch tc.Visibility {
	case TestCaseVisibilityExample, TestCaseVisibilityPublic, TestCaseVisibilityHidden:
		return nil
	default:
		return fmt.Errorf("%w: got %q", ErrInvalidVisibility, tc.Visibility)
	}
}

type TestCases []TestCase

type ASTCustomRule struct {
	Language string `json:"language"`
	Pattern  string `json:"pattern"`
	Type     string `json:"type,omitempty"` // method, function, module
	Message  string `json:"message"`
}

// CustomRuleViolation describe la infracción de una regla personalizada
// declarada por el docente en `ast_rules.custom_rules`.
type CustomRuleViolation struct {
	RuleID   string `json:"rule_id"`
	Type     string `json:"type"`
	Language string `json:"language"`
	Pattern  string `json:"pattern"`
	Target   string `json:"target"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
}

type ASTRules struct {
	BlockNativeSort    bool            `json:"block_native_sort,omitempty"`
	BlockSystemModules bool            `json:"block_system_modules,omitempty"`
	ForbiddenImports   []string        `json:"forbidden_imports"`
	ForbiddenFunctions []string        `json:"forbidden_functions"`
	CustomRules        []ASTCustomRule `json:"custom_rules,omitempty"`
}

type ChecklistReport struct {
	Blockers   []string `json:"blockers"`
	Warnings   []string `json:"warnings"`
	Info       []string `json:"info"`
	CanPublish bool     `json:"can_publish"`
}

type ReferenceSolutionInput struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type ComparatorConfig struct {
	ID     string         `json:"id"`
	Params map[string]any `json:"params,omitempty"`
}

type AlgorithmConfig struct {
	TestCases     TestCases         `json:"test_cases"`
	ASTRules      ASTRules          `json:"ast_rules"`
	Comparator    *ComparatorConfig `json:"comparator,omitempty"`
	TimeLimitMS   int               `json:"time_limit_ms"`
	MemoryLimitMB int               `json:"memory_limit_mb"`
	InputFormat   json.RawMessage   `json:"input_format,omitempty"`
}

type DatabaseConfig struct {
	Engine            string `json:"engine"`                // e.g. "postgres"
	InitScript        string `json:"init_script,omitempty"` // DDL / DML previo (vive en exercise_db_init_scripts, ADR-073)
	ReferenceSolution string `json:"reference_solution"`    // Solución de referencia del docente
	ValidationQuery   string `json:"validation_query"`      // Query para extraer el estado resultante
	ExpectedJSON      string `json:"expected_json"`         // JSON de referencia (autogenerado vía Dry Run)
	TimeLimitMS       int    `json:"time_limit_ms"`
	MemoryLimitMB     int    `json:"memory_limit_mb"`
}

type IDEConfig struct {
	AllowResubmit  bool   `json:"allow_resubmit"`
	AutoCheckpoint bool   `json:"auto_checkpoint"`
	DBInitScript   string `json:"db_init_script,omitempty"`
	// AppRunCommand arranca la aplicación del proyecto dentro del contenedor de
	// preview. Es docencia explícita: no se infiere del snapshot.
	AppRunCommand string `json:"app_run_command,omitempty"`
	// AppPort es el puerto TCP donde la app escucha dentro del contenedor.
	AppPort int `json:"app_port,omitempty"`
}

type ExerciseConfig struct {
	Algorithm   *AlgorithmConfig `json:"algorithm,omitempty"`
	Database    *DatabaseConfig  `json:"database,omitempty"`
	IDE         *IDEConfig       `json:"ide,omitempty"`
	InputFormat json.RawMessage  `json:"input_format,omitempty"`
	// AllowedLanguages son los lenguajes que el alumno puede elegir para
	// resolver la práctica. Vacía conserva el contrato anterior: `language` es
	// el único lenguaje admitido.
	AllowedLanguages []string `json:"allowed_languages,omitempty"`
	// Boilerplates guarda el esqueleto inicial por lenguaje, para que el alumno
	// reciba el código de partida correcto al cambiar de lenguaje. Las claves
	// están canonizadas y solo existen para lenguajes admitidos.
	Boilerplates map[string]string `json:"boilerplates,omitempty"`
}

// Value persiste el config sin los scripts de inicialización. Desde ADR-073 el texto
// del script vive en exercise_db_init_scripts: la columna config deja de cargar con el
// contenido completo del docente en cada lectura del ejercicio.
func (ec ExerciseConfig) Value() (driver.Value, error) {
	return json.Marshal(ec.withoutInitScripts())
}

// withoutInitScripts devuelve una copia del config con los scripts de inicialización en
// blanco. Copia los punteros que toca para no modificar el ejercicio del llamador, que
// sigue usando el texto en memoria durante el guardado y el arranque del laboratorio.
func (ec ExerciseConfig) withoutInitScripts() ExerciseConfig {
	sanitized := ec
	if ec.IDE != nil {
		ide := *ec.IDE
		ide.DBInitScript = ""
		sanitized.IDE = &ide
	}
	if ec.Database != nil {
		database := *ec.Database
		database.InitScript = ""
		sanitized.Database = &database
	}
	return sanitized
}

func (ec *ExerciseConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ExerciseConfig value: %v", value)
	}
	return json.Unmarshal(bytes, ec)
}

var (
	ErrMissingReferenceSolution          = errors.New("el ejercicio requiere una solución de referencia antes de ser publicado")
	ErrExerciseStale                     = errors.New("el ejercicio tiene cambios pendientes y requiere un dry-run exitoso antes de ser publicado")
	ErrPayloadMismatchForEnvironmentType = errors.New("payload_mismatch_for_environment_type: los campos del payload no coinciden con el tipo de entorno (environment_type) seleccionado")
	ErrTestCasesRequireJudge             = errors.New("test_cases_require_judge_environment: los casos de prueba I/O solo aplican a ejercicios de juez virtual (JUEZ_EFIMERO); los laboratorios IDE se corrigen con rúbrica en SpeedGrader")
	// ErrOverrideRequiresJudge: el veredicto es una salida del juez. Un proyecto
	// de IDE se califica con nota y rúbrica, nunca asignándole un veredicto.
	ErrOverrideRequiresJudge = errors.New("verdict_override_requires_judge: un proyecto de IDE se califica con nota y rúbrica, no con veredicto")
	// ErrEvaluationRequiresJudge: la evaluación automática solo existe donde hay
	// juez. Un proyecto de IDE no se ejecuta contra casos de prueba.
	ErrEvaluationRequiresJudge = errors.New("evaluation_requires_judge_environment: la evaluación automática solo aplica a ejercicios de juez virtual (JUEZ_EFIMERO)")
	// ErrInvalidEvaluationReceipt: el veredicto de una entrega de juez solo puede
	// venir de la evaluación que emitió el servidor, no de un valor del navegador.
	ErrInvalidEvaluationReceipt = errors.New("invalid_evaluation_receipt: la entrega requiere una evaluación vigente del servidor")
	// ErrDuplicateSubmission: un recibo registra una entrega. Reintentar el
	// envío devuelve la entrega ya creada, no una segunda.
	ErrDuplicateSubmission = errors.New("duplicate_submission: la evaluación ya tiene una entrega registrada")
	// ErrExerciseDeadlineExpired indica que la fecha límite (due_date) del ejercicio ya venció.
	ErrExerciseDeadlineExpired = errors.New("exercise_deadline_expired: el plazo de entrega para este ejercicio ha finalizado")
	// ErrResubmissionNotAllowed indica que el ejercicio no admite reentregas para el estudiante.
	ErrResubmissionNotAllowed = errors.New("resubmission_not_allowed: este ejercicio o examen no admite reentregas")
	// ErrNoExampleCasesForTesting indica que el ejercicio no tiene casos de ejemplo para probar.
	ErrNoExampleCasesForTesting = errors.New("no_example_cases_for_testing: el ejercicio no tiene casos de prueba de ejemplo disponibles para probar")
)

type DryRunJobStatus string

const (
	DryRunJobStatusQueued  DryRunJobStatus = "queued"
	DryRunJobStatusRunning DryRunJobStatus = "running"
	DryRunJobStatusDone    DryRunJobStatus = "done"
	DryRunJobStatusFailed  DryRunJobStatus = "failed"
)

type DryRunJob struct {
	ID              string            `json:"id" db:"id"`
	ExerciseID      string            `json:"exercise_id" db:"exercise_id"`
	Status          DryRunJobStatus   `json:"status" db:"status"`
	ProgressCurrent int               `json:"progress_current" db:"progress_current"`
	ProgressTotal   int               `json:"progress_total" db:"progress_total"`
	Result          *EvaluationResult `json:"result,omitempty" db:"result"`
	Error           string            `json:"error,omitempty" db:"error"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

type TemplateSummary struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	DockerImage       string         `json:"docker_image"`
	Entrypoint        string         `json:"entrypoint,omitempty"`
	TimeoutMS         int            `json:"timeout_ms,omitempty"`
	BaseRamMB         int            `json:"base_ram_mb"`
	TargetEnvironment string         `json:"target_environment"`
	ServicesConfig    ServicesConfig `json:"services_config"`
	// LanguageRunners son los lenguajes que el entorno sabe ejecutar. Cuando la
	// lista está vacía se usa la pareja DockerImage/Entrypoint del entorno.
	LanguageRunners []LanguageRunner `json:"language_runners,omitempty"`
}

// SupportsLanguage indica si el entorno declara el lenguaje de una entrega.
func (t *TemplateSummary) SupportsLanguage(language string) bool {
	if t == nil {
		return false
	}
	_, ok := ResolveLanguageRunner(t.LanguageRunners, language)
	return ok
}

// ResolveRunner devuelve el ejecutor del lenguaje de la entrega.
func (t *TemplateSummary) ResolveRunner(language string) (LanguageRunner, bool) {
	if t == nil {
		return LanguageRunner{}, false
	}
	return ResolveLanguageRunner(t.LanguageRunners, language)
}

// EnvironmentSnapshotBinding es un binding de servicio resuelto al publicar.
// Guarda la decisión efectiva (rol, motor, alias e init), no la referencia viva
// a la plantilla: editar la plantilla no cambia una publicación anterior.
type EnvironmentSnapshotBinding struct {
	Role        ServiceRole     `json:"role,omitempty"`
	Engine      string          `json:"engine"`
	ComponentID string          `json:"component_id,omitempty"`
	Version     string          `json:"version,omitempty"`
	EnvAlias    string          `json:"env_alias,omitempty"`
	InitKind    ServiceInitKind `json:"init_kind"`
}

// EnvironmentInitContract deja explícito cómo se inicializa cada binding.
type EnvironmentInitContract struct {
	Alias  string          `json:"alias"`
	Engine string          `json:"engine"`
	Kind   ServiceInitKind `json:"kind"`
}

// EnvironmentSnapshot es el pin del entorno que se persiste al publicar un
// ejercicio (ADR-050, D4). El arranque del workspace y el juez lo leen antes
// que la plantilla viva; sin snapshot se usa la plantilla viva y se registra el
// fallback.
type EnvironmentSnapshot struct {
	TemplateID              string                       `json:"template_id"`
	TemplateName            string                       `json:"template_name,omitempty"`
	DockerImage             string                       `json:"docker_image"`
	Entrypoint              string                       `json:"entrypoint"`
	TimeoutMS               int                          `json:"timeout_ms"`
	BaseRAMMB               int                          `json:"base_ram_mb"`
	TargetEnvironment       string                       `json:"target_environment,omitempty"`
	Bindings                []EnvironmentSnapshotBinding `json:"bindings"`
	InitContract            []EnvironmentInitContract    `json:"init_contract"`
	SetupScript             string                       `json:"setup_script,omitempty"`
	LanguageRunners         []LanguageRunner             `json:"language_runners,omitempty"`
	VerificationFingerprint string                       `json:"verification_fingerprint"`
}

// AsTemplateSummary reconstruye la vista de plantilla que consumen el juez y
// los editores. No incluye el script de setup: quien arranca el contenedor usa
// AsTemplate.
func (s *EnvironmentSnapshot) AsTemplateSummary() *TemplateSummary {
	if s == nil {
		return nil
	}
	services := make([]ServiceRequirement, 0, len(s.Bindings))
	for _, binding := range s.Bindings {
		services = append(services, ServiceRequirement{
			Role:        binding.Role,
			Engine:      binding.Engine,
			ComponentID: binding.ComponentID,
			Version:     binding.Version,
			EnvAlias:    binding.EnvAlias,
		})
	}
	return &TemplateSummary{
		ID:                s.TemplateID,
		Name:              s.TemplateName,
		DockerImage:       s.DockerImage,
		Entrypoint:        s.Entrypoint,
		TimeoutMS:         s.TimeoutMS,
		BaseRamMB:         s.BaseRAMMB,
		TargetEnvironment: s.TargetEnvironment,
		ServicesConfig:    ServicesConfig{Services: services},
		LanguageRunners:   s.LanguageRunners,
	}
}

// AsTemplate reconstruye la plantilla que arranca el workspace con los valores
// congelados en la publicación.
func (s *EnvironmentSnapshot) AsTemplate() *Template {
	summary := s.AsTemplateSummary()
	if summary == nil {
		return nil
	}
	return &Template{
		ID:             summary.ID,
		Name:           summary.Name,
		DockerImage:    summary.DockerImage,
		BaseRamMB:      summary.BaseRamMB,
		ServicesConfig: summary.ServicesConfig,
		SetupScript:    s.SetupScript,
	}
}

// ComputeVerificationFingerprint devuelve una huella determinista del pin. El
// propio campo de huella no participa del cálculo.
func (s EnvironmentSnapshot) ComputeVerificationFingerprint() string {
	s.VerificationFingerprint = ""
	raw, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])[:16]
}

type Exercise struct {
	ID              string         `json:"id" db:"id"`
	SubjectID       *string        `json:"subject_id,omitempty" db:"subject_id"`
	Title           string         `json:"title" db:"title"`
	Description     string         `json:"description" db:"description"`
	Type            ExerciseType   `json:"type" db:"type"`
	EnvironmentType string         `json:"environment_type" db:"environment_type"`
	TemplateID      string         `json:"template_id" db:"template_id"`
	Difficulty      *string        `json:"difficulty,omitempty" db:"difficulty"`
	Tags            pq.StringArray `json:"tags" db:"tags"`
	Purpose         string         `json:"purpose" db:"purpose"` // class, exam
	// ExamNetwork declara si el entorno puede salir a internet. `none` por
	// defecto; `egress` exige motivo y solo aplica a ejercicios de examen.
	ExamNetwork       string `json:"exam_network" db:"exam_network"`
	ExamNetworkReason string `json:"exam_network_reason" db:"exam_network_reason"`
	// ExamOffline* guarda el resultado de la verificación previa a publicar un
	// examen sin salida. Vacío significa que todavía no se corrió.
	ExamOfflineStatus    string           `json:"exam_offline_status" db:"exam_offline_status"`
	ExamOfflineOutput    string           `json:"exam_offline_output" db:"exam_offline_output"`
	ExamOfflineCheckedAt *time.Time       `json:"exam_offline_checked_at,omitempty" db:"exam_offline_checked_at"`
	PerStudentSeed       bool             `json:"per_student_seed" db:"per_student_seed"`
	DueDate              *time.Time       `json:"due_date,omitempty" db:"due_date"`
	Boilerplate          string           `json:"boilerplate" db:"boilerplate"`
	Status               string           `json:"status" db:"status"` // draft, published, closed
	Language             string           `json:"language" db:"language"`
	TimeLimitMS          int              `json:"time_limit_ms" db:"time_limit_ms"`
	MemoryLimitMB        int              `json:"memory_limit_mb" db:"memory_limit_mb"`
	ReferenceSolution    string           `json:"reference_solution" db:"reference_solution"`
	Stale                bool             `json:"stale" db:"stale"`
	LastValidDryRunAt    *time.Time       `json:"last_valid_dry_run_at,omitempty" db:"last_valid_dry_run_at"`
	ModuleID             *string          `json:"module_id,omitempty" db:"module_id"`
	ExpectedComplexity   *string          `json:"expected_complexity,omitempty" db:"expected_complexity"`
	Config               ExerciseConfig   `json:"config" db:"config"`
	TenantID             string           `json:"tenant_id" db:"tenant_id"`
	Template             *TemplateSummary `json:"template,omitempty" db:"-"`
	// EnvironmentSnapshot es el pin de ADR-050. Se lee junto con el ejercicio y
	// tiene prioridad sobre la plantilla viva al arrancar un entorno.
	EnvironmentSnapshot     *EnvironmentSnapshot `json:"environment_snapshot,omitempty" db:"-"`
	EnvironmentSnapshotJSON []byte               `json:"-" db:"environment_snapshot"`
	CreatedAt               time.Time            `json:"created_at" db:"created_at"`
}

type ComplexityMeasurement struct {
	InputSize int     `json:"input_size"`
	TimeMs    float64 `json:"time_ms"`
	MemoryKb  float64 `json:"memory_kb"`
}

type ComplexityAnalysis struct {
	TimeComplexity  string                  `json:"time_complexity"`
	SpaceComplexity string                  `json:"space_complexity"`
	Measurements    []ComplexityMeasurement `json:"measurements"`
	Confidence      float64                 `json:"confidence"`
}

func (ex *Exercise) Validate() error {
	if ex.PerStudentSeed && ex.Purpose != string(ExercisePurposeExam) {
		return ErrSeedRequiresExamPurpose
	}
	if ex.Difficulty != nil && *ex.Difficulty != "" {
		diff := *ex.Difficulty
		if diff != string(ExerciseDifficultyEasy) && diff != string(ExerciseDifficultyMedium) && diff != string(ExerciseDifficultyHard) {
			return fmt.Errorf("invalid difficulty %q: must be 'easy', 'medium', or 'hard'", diff)
		}
	}
	if ex.Purpose == "" {
		ex.Purpose = string(ExercisePurposeClass)
	}
	if ex.Purpose != string(ExercisePurposeClass) && ex.Purpose != string(ExercisePurposeExam) {
		return fmt.Errorf("invalid purpose %q: must be 'class' or 'exam'", ex.Purpose)
	}
	if ex.Tags == nil {
		ex.Tags = pq.StringArray{}
	}
	// El valor 0 se permite: el repositorio lo normaliza a 1000 ms como en
	// flujos históricos (import y DTOs que omiten el campo).
	if ex.TimeLimitMS != 0 && (ex.TimeLimitMS < 100 || ex.TimeLimitMS > 30000) {
		return ErrTimeLimitOutOfRange
	}
	if err := ValidateExamNetwork(ex.Purpose, ex.ExamNetwork, ex.ExamNetworkReason); err != nil {
		return err
	}
	ex.ExamNetwork = NormalizeExamNetwork(ex.ExamNetwork)
	if ex.ExamNetwork == ExamNetworkNone {
		ex.ExamNetworkReason = ""
	}
	if ide := ex.Config.IDE; ide != nil {
		if ide.AppPort != 0 && (ide.AppPort < 1 || ide.AppPort > 65535) {
			return ErrInvalidAppPort
		}
		if len(strings.TrimSpace(ide.AppRunCommand)) > ReviewEnvironmentMaxCommandLength {
			return fmt.Errorf("app_run_command exceeds %d characters", ReviewEnvironmentMaxCommandLength)
		}
	}
	return nil
}

// AllowsResubmit indica si el ejercicio admite múltiples entregas para un mismo estudiante.
// En entornos IDE consulta la propiedad explícita de su configuración.
// En jueces efímeros, las prácticas permiten reentregas y los exámenes son de entrega única.
func (ex *Exercise) AllowsResubmit() bool {
	if ex == nil {
		return false
	}
	if ex.Config.IDE != nil {
		return ex.Config.IDE.AllowResubmit
	}
	if ex.Purpose == string(ExercisePurposeExam) {
		return false
	}
	return true
}

type DueAssignment struct {
	ExerciseID  string     `json:"exercise_id" db:"exercise_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	SubjectID   string     `json:"subject_id" db:"subject_id"`
	SubjectName string     `json:"subject_name" db:"subject_name"`
	SubjectCode string     `json:"subject_code" db:"subject_code"`
	DueDate     *time.Time `json:"due_date" db:"due_date"`
	Type        string     `json:"type" db:"type"`
}

type EvaluationResult struct {
	Verdict         Verdict   `json:"verdict"`
	ExecutionTimeMS int       `json:"execution_time_ms"`
	MemoryUsedMB    float64   `json:"memory_used_mb"`
	Message         string    `json:"message"`
	FailedTestCase  *TestCase `json:"failed_test_case,omitempty"`
	// CaseResults agrega el veredicto de cada caso evaluado. La evaluacion
	// recorre todos los casos sin detencion temprana (D-EJ-03).
	CaseResults        []CaseResult        `json:"case_results,omitempty"`
	ActualJSON         string              `json:"actual_json,omitempty"`
	ExpectedJSON       string              `json:"expected_json,omitempty"`
	GeneratedCases     []TestCase          `json:"generated_cases,omitempty"`
	ComplexityAnalysis *ComplexityAnalysis `json:"complexity_analysis,omitempty"`
	// Receipt firma el resultado para que la entrega del juez no acepte un
	// veredicto declarado por el navegador. Ver EvaluationReceipt.
	Receipt string `json:"evaluation_receipt,omitempty"`
}

// CaseResult es el resultado de un unico caso de prueba.
type CaseResult struct {
	Index        int     `json:"index"`
	Verdict      Verdict `json:"verdict"`
	DurationMS   int     `json:"duration_ms"`
	Message      string  `json:"message,omitempty"`
	ActualOutput string  `json:"actual_output,omitempty"`
}

// RunMetric registra la telemetria de un caso ejecutado. Language usa la
// clave canonica (c++ -> cpp, c#/cs -> csharp). Ver design.md y 00012.
type RunMetric struct {
	ExerciseID  string  `json:"exercise_id" db:"exercise_id"`
	Language    string  `json:"language" db:"language"`
	ImageDigest string  `json:"image_digest" db:"image_digest"`
	DurationMS  int     `json:"duration_ms" db:"duration_ms"`
	Verdict     Verdict `json:"verdict" db:"verdict"`
	CaseIndex   int     `json:"case_index" db:"case_index"`
}

type EvaluationRunConfig struct {
	Language      string
	SourceCode    string
	MemoryLimitMB int
	TimeLimitMS   int
	TestCase      TestCase
	Comparator    *ComparatorConfig
	DockerImage   string
	Entrypoint    string
}

type TestCaseRunResult struct {
	Verdict       Verdict
	ExecutionTime time.Duration
	ActualOutput  string
	StdErr        string
	ErrorDetails  string
	// ImageDigest identifica la imagen del runner (forma repo@digest) y
	// alimenta run_metrics. Lo informa la estrategia de lenguaje.
	ImageDigest string
}

type DBEvaluationRunConfig struct {
	Engine          string
	InitScript      string
	SolutionSQL     string
	ValidationQuery string
	TimeLimitMS     int
	MemoryLimitMB   int
}

type DBEvaluationResult struct {
	Verdict       Verdict
	ExecutionTime time.Duration
	ResultingJSON string
	ErrorDetails  string
}
