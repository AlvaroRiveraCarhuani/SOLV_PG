package services

import (
	"context"
	"solv-backend/internal/core/domain"
	"sort"
	"strings"
)

type EvaluationService struct {
	exerciseRepo    domain.ExerciseRepository
	astAnalyzer     domain.ASTAnalyzer
	codeScanner     domain.CodeScanner
	runner          domain.EvaluationRunner
	metrics         RunMetricsRecorder
	formatValidator FormatValidator
	moduleRepo      domain.CourseModuleRepository
	scriptRunner    domain.ScriptSandboxRunner
	adminGovRepo    domain.AdminGovernanceRepository
	snapshotRepo    domain.GeneratedCaseSnapshotRepository
}

func (s *EvaluationService) SetAdminGovernanceRepository(repo domain.AdminGovernanceRepository) {
	s.adminGovRepo = repo
}

// SetGeneratedCaseSnapshotRepository activa el guardado del snapshot
// autoritativo de los casos generados por semilla.
func (s *EvaluationService) SetGeneratedCaseSnapshotRepository(repo domain.GeneratedCaseSnapshotRepository) {
	s.snapshotRepo = repo
}

// RunMetricsRecorder persiste la telemetria por caso (tabla run_metrics,
// migracion 00012). Es opcional: sin registrador la evaluacion omite el
// registro pero evalua todos los casos igual.
type RunMetricsRecorder interface {
	RecordRunMetric(ctx context.Context, metric domain.RunMetric) error
}

// P95Query calcula el p95 de duration_ms sobre veredictos AC por lenguaje en
// la ventana de language_profiles.p95_window_days (design.md, D-EJ-04).
const P95Query = `SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) FROM run_metrics WHERE language = $1 AND verdict = 'AC' AND created_at > now() - ($2 || ' days')::interval`

// SetMetricsRecorder activa el registro de run_metrics por caso.
func (s *EvaluationService) SetMetricsRecorder(r RunMetricsRecorder) {
	s.metrics = r
}

// CanonicalLanguage normaliza el alias a clave canonica (c++ -> cpp,
// c#/cs -> csharp). Los perfiles y run_metrics rechazan alias (spec 3.1).
func CanonicalLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "c++":
		return "cpp"
	case "c#", "cs":
		return "csharp"
	default:
		return strings.ToLower(strings.TrimSpace(language))
	}
}

// P95OverAC calcula el p95 de duration_ms sobre veredictos AC con
// nearest-rank. Sin muestras AC devuelve 0.
func P95OverAC(metrics []domain.RunMetric) int {
	samples := make([]int, 0, len(metrics))
	for _, m := range metrics {
		if m.Verdict == domain.VerdictAC {
			samples = append(samples, m.DurationMS)
		}
	}
	if len(samples) == 0 {
		return 0
	}
	sort.Ints(samples)
	rank := (95*len(samples) + 99) / 100
	return samples[rank-1]
}

func NewEvaluationService(
	exerciseRepo domain.ExerciseRepository,
	astAnalyzer domain.ASTAnalyzer,
	codeScanner domain.CodeScanner,
	runner domain.EvaluationRunner,
) *EvaluationService {
	return &EvaluationService{
		exerciseRepo:    exerciseRepo,
		astAnalyzer:     astAnalyzer,
		codeScanner:     codeScanner,
		runner:          runner,
		formatValidator: NewFormatValidator(),
	}
}

func (s *EvaluationService) SetFormatValidator(v FormatValidator) {
	s.formatValidator = v
}

func (s *EvaluationService) SetModuleRepository(repo domain.CourseModuleRepository) {
	s.moduleRepo = repo
}
