package httpdelivery

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// verdictRunner simula los casos de prueba ejecutados por el juez: el
// veredicto sale de cada caso, no de la petición.
type verdictRunner struct {
	verdicts []domain.Verdict
	calls    int
}

func (r *verdictRunner) RunTestCase(_ context.Context, _ domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	v := domain.VerdictAC
	if r.calls < len(r.verdicts) {
		v = r.verdicts[r.calls]
	}
	r.calls++
	return domain.TestCaseRunResult{Verdict: v, ExecutionTime: 5 * time.Millisecond}, nil
}

func (r *verdictRunner) RunDBDryRun(_ context.Context, _ domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func (r *verdictRunner) RunDBEvaluation(_ context.Context, _ domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

type httpExerciseRepo struct {
	exercise *domain.Exercise
}

func (r *httpExerciseRepo) GetByID(_ context.Context, _ string) (*domain.Exercise, error) {
	return r.exercise, nil
}

func (r *httpExerciseRepo) GetByIDAndTenant(_ context.Context, _, _ string) (*domain.Exercise, error) {
	return r.exercise, nil
}

func (r *httpExerciseRepo) Create(_ context.Context, _ *domain.Exercise) error { return nil }

func (r *httpExerciseRepo) Update(_ context.Context, _ *domain.Exercise) error { return nil }

func (r *httpExerciseRepo) UpdateStatus(_ context.Context, _, _, _ string) error { return nil }

func (r *httpExerciseRepo) UpdateConfig(_ context.Context, _, _ string, _ domain.ExerciseConfig) error {
	return nil
}

func (r *httpExerciseRepo) UpdateExpectedJSON(_ context.Context, _, _ string) error { return nil }

func (r *httpExerciseRepo) MarkExerciseStale(_ context.Context, _, _ string, _ bool) error {
	return nil
}

func (r *httpExerciseRepo) UpdateExerciseLastValidDryRun(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

func (r *httpExerciseRepo) CreateDryRunJob(_ context.Context, _ *domain.DryRunJob) error { return nil }

func (r *httpExerciseRepo) GetDryRunJob(_ context.Context, _ string) (*domain.DryRunJob, error) {
	return nil, nil
}

func (r *httpExerciseRepo) UpdateDryRunJobProgress(_ context.Context, _ string, _ domain.DryRunJobStatus, _, _ int, _ *domain.EvaluationResult, _ string) error {
	return nil
}

func (r *httpExerciseRepo) ListDueByStudent(_ context.Context, _, _ string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (r *httpExerciseRepo) ListBySubject(_ context.Context, _, _ string) ([]*domain.Exercise, error) {
	return nil, nil
}

func (r *httpExerciseRepo) GetStudentRecommendations(_ context.Context, _, _, _ string) (*domain.StudentRecommendations, error) {
	return nil, nil
}

type httpAllowAnalyzer struct{}

func (httpAllowAnalyzer) ValidateCustomRules(_ string, _ string, _ domain.ASTRules) *domain.CustomRuleViolation {
	return nil
}

func (httpAllowAnalyzer) ValidateCode(_ string, _ string, _ domain.ASTRules) (bool, string) {
	return true, ""
}

func verdictExercise() *domain.Exercise {
	return &domain.Exercise{
		ID:   "ex-verdicto",
		Type: domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: []domain.TestCase{
					{Input: "1", ExpectedOutput: "1"},
					{Input: "2", ExpectedOutput: "4"},
				},
				TimeLimitMS:   2000,
				MemoryLimitMB: 128,
			},
		},
	}
}

// El navegador puede escribir "verdict" en el cuerpo: el handler no lo lee. El
// juez ejecuta los casos y el recibo firma el resultado que calculó.
func TestEvaluate_ComputesVerdictOnServerAndIgnoresClientVerdict(t *testing.T) {
	runner := &verdictRunner{verdicts: []domain.Verdict{domain.VerdictAC, domain.VerdictWA}}
	svc := services.NewEvaluationService(&httpExerciseRepo{exercise: verdictExercise()}, httpAllowAnalyzer{}, nil, runner)
	handler := NewEvaluationHandler(svc, validator.New())

	payload, err := json.Marshal(map[string]string{
		"exercise_id":     "ex-verdicto",
		"language":        "python",
		"source_code_b64": base64.StdEncoding.EncodeToString([]byte("print(1)")),
		"verdict":         "TLE",
	})
	if err != nil {
		t.Fatalf("no se pudo armar la petición: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluations", bytes.NewReader(payload))
	req = withTestIdentity(req, "student-1", "tenant-test", "student")

	rr := httptest.NewRecorder()
	handler.Evaluate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Verdict           string `json:"verdict"`
			EvaluationReceipt string `json:"evaluation_receipt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta ilegible: %v", err)
	}

	if runner.calls != 2 {
		t.Fatalf("el juez debe ejecutar los 2 casos, ejecutó %d", runner.calls)
	}
	if resp.Data.Verdict != string(domain.VerdictWA) {
		t.Fatalf("el veredicto debe salir de los casos (WA), no del cuerpo (TLE); obtenido %q", resp.Data.Verdict)
	}

	receipt, err := services.ParseEvaluationReceipt(resp.Data.EvaluationReceipt, time.Now())
	if err != nil {
		t.Fatalf("el recibo devuelto debe ser válido: %v", err)
	}
	if receipt.Verdict != string(domain.VerdictWA) {
		t.Fatalf("el recibo debe firmar el veredicto del servidor, obtenido %q", receipt.Verdict)
	}
	if receipt.StudentID != "student-1" {
		t.Fatalf("el recibo debe identificar al alumno autenticado, obtenido %q", receipt.StudentID)
	}
	if receipt.CodeHash != services.HashEvaluatedCode("print(1)") {
		t.Fatalf("el recibo debe atar el código ejecutado, obtenido %q", receipt.CodeHash)
	}
}
