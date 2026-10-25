package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

// mockAICopilotRunner implementa domain.EvaluationRunner con salida/veredicto
// configurables para el sello determinista.
type mockAICopilotRunner struct {
	verdict      domain.Verdict
	actualOutput string
	err          error
	calls        []domain.EvaluationRunConfig
}

func (m *mockAICopilotRunner) RunTestCase(_ context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	m.calls = append(m.calls, config)
	if m.err != nil {
		return domain.TestCaseRunResult{}, m.err
	}
	return domain.TestCaseRunResult{Verdict: m.verdict, ActualOutput: m.actualOutput}, nil
}

func (m *mockAICopilotRunner) RunDBDryRun(_ context.Context, _ domain.DBEvaluationRunConfig) (string, error) {
	return "", nil
}

func (m *mockAICopilotRunner) RunDBEvaluation(_ context.Context, _ domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{}, nil
}

// newCopilotLLMServer devuelve un servidor httptest que responde siempre con
// el contenido dado dentro de la envolvente OpenAI-compatible.
func newCopilotLLMServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
	})
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta del LLM: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

// newEnabledCopilotSettings devuelve un AISettingsService habilitado (openai)
// apuntando por transporte inyectable al servidor httptest del LLM.
func newEnabledCopilotSettings(t *testing.T, server *httptest.Server) *AISettingsService {
	t.Helper()
	svc := NewAISettingsService(newTestAISettingsRepo(`{}`))
	if _, err := svc.Update(context.Background(), "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 120,
		TimeoutSeconds:     10,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})
	return svc
}

func TestAICopilot_SuggestComponent_SynthesizesAndSanitizes(t *testing.T) {
	componentJSON := `{"component":{"name":"PyTorch CPU + scikit-learn","slug":"python-ml-stack","layer":"addon","sub_category":"toolkit","description":"Framework de machine learning para Python","icon_key":"python","base_image":"python:3.12-slim-bookworm","compatible_runtimes":["python"],"install_mode":"pip","package_names":["torch","scikit-learn"],"build_layer_snippet":"","ram_delta_mb":0},"rationale":"torch cubre el entrenamiento y scikit-learn los modelos clásicos."}`

	// Muchos modelos envuelven el JSON en fences Markdown aun con json_object.
	server := newCopilotLLMServer(t, "```json\n"+componentJSON+"\n```")
	svc := newEnabledCopilotSettings(t, server)
	copilot := NewAICopilotService(svc, nil)

	resp, err := copilot.SuggestComponent(context.Background(), "t-1", "u-1", domain.AISuggestComponentRequestDTO{
		Prompt: "PyTorch CPU y scikit-learn para Inteligencia Artificial",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Component.BuildLayerSnippet != "RUN pip install --no-cache-dir torch scikit-learn" {
		t.Errorf("capa de build sintetizada inesperada: %q", resp.Component.BuildLayerSnippet)
	}
	if resp.Component.InstallMode != domain.InstallModePip {
		t.Errorf("esperaba install_mode pip, got %q", resp.Component.InstallMode)
	}
	if strings.Contains(resp.Component.BaseImage, ":latest") {
		t.Errorf("la imagen debe llevar tag fijo, got %q", resp.Component.BaseImage)
	}
	if strings.TrimSpace(resp.Rationale) == "" {
		t.Errorf("esperaba una rationale técnica en español")
	}
	if resp.LatencyMS < 0 {
		t.Errorf("latency_ms inválida: %d", resp.LatencyMS)
	}

	// Prompt fuera de rango -> error de validación (HTTP 400 en el handler).
	if _, err := copilot.SuggestComponent(context.Background(), "t-1", "u-1", domain.AISuggestComponentRequestDTO{
		Prompt: "ab",
	}); err == nil {
		t.Errorf("esperaba rechazo de prompt menor a 3 caracteres")
	}
}

func TestAICopilot_SuggestComponent_RejectsLatestTag(t *testing.T) {
	latestJSON := `{"component":{"name":"Runtime Python","layer":"runtime","sub_category":"backend","base_image":"python:latest","install_mode":"none"},"rationale":"imagen genérica"}`
	server := newCopilotLLMServer(t, latestJSON)
	svc := newEnabledCopilotSettings(t, server)
	copilot := NewAICopilotService(svc, nil)

	_, err := copilot.SuggestComponent(context.Background(), "t-1", "u-1", domain.AISuggestComponentRequestDTO{
		Prompt: "Runtime de Python para laboratorios",
	})
	if err == nil {
		t.Fatalf("esperaba rechazo de imagen con tag :latest")
	}
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "base_image_latest" {
		t.Errorf("esperaba AISettingsValidationError base_image_latest, got %v", err)
	}

	// Imagen sin tag también se rechaza.
	unpinnedJSON := `{"component":{"name":"Runtime Python","layer":"runtime","sub_category":"backend","base_image":"python-slim","install_mode":"none"},"rationale":"imagen sin tag"}`
	serverUnpinned := newCopilotLLMServer(t, unpinnedJSON)
	svcUnpinned := NewAISettingsService(newTestAISettingsRepo(`{}`))
	if _, err := svcUnpinned.Update(context.Background(), "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 120,
		TimeoutSeconds:     10,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}
	svcUnpinned.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, serverUnpinned.URL)})
	copilotUnpinned := NewAICopilotService(svcUnpinned, nil)

	_, err = copilotUnpinned.SuggestComponent(context.Background(), "t-1", "u-1", domain.AISuggestComponentRequestDTO{
		Prompt: "Runtime de Python para laboratorios",
	})
	if err == nil {
		t.Fatalf("esperaba rechazo de imagen sin tag de versión")
	}
	if !errors.As(err, &vErr) || vErr.Code != "base_image_unpinned" {
		t.Errorf("esperaba AISettingsValidationError base_image_unpinned, got %v", err)
	}
}

func TestAICopilot_GenerateExerciseDraft_SealsExpectedOutputWithRunner(t *testing.T) {
	exerciseJSON := `{"title":"Suma de dos enteros","description":"# Suma\nLea dos enteros e imprima su suma.","difficulty":"easy","boilerplate":"a, b = map(int, input().split())\n# imprima a + b","reference_solution":"a, b = map(int, input().split())\nprint(a + b)","test_cases":[{"input":"1 2\n","expected_output":"999","visibility":"public","weight":1.0},{"input":"10 5\n","expected_output":"15","visibility":"hidden","weight":2.0}]}`

	server := newCopilotLLMServer(t, exerciseJSON)
	svc := newEnabledCopilotSettings(t, server)

	// El LLM alucina "999"; el runner devuelve la salida real "42".
	runner := &mockAICopilotRunner{verdict: domain.VerdictAC, actualOutput: "42\n"}
	copilot := NewAICopilotService(svc, runner)

	draft, err := copilot.GenerateExerciseDraft(context.Background(), "t-1", "u-1", domain.AIGenerateExerciseRequestDTO{
		Prompt:           "Suma de dos enteros leídos por stdin",
		Language:         "python",
		Difficulty:       "easy",
		PublicCasesCount: 2,
		HiddenCasesCount: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !draft.RunnerVerifiedAll {
		t.Fatalf("esperaba runner_verified_all=true, warning=%q", draft.VerificationWarning)
	}
	if len(draft.TestCases) != 2 {
		t.Fatalf("esperaba 2 casos, got %d", len(draft.TestCases))
	}
	for i, tc := range draft.TestCases {
		if tc.ExpectedOutput != "42" {
			t.Errorf("caso %d: esperaba expected_output sellado \"42\", got %q", i, tc.ExpectedOutput)
		}
		if !tc.VerifiedByRunner {
			t.Errorf("caso %d: esperaba verified_by_runner=true", i)
		}
		if tc.OrderIndex != i+1 {
			t.Errorf("caso %d: esperaba order_index %d, got %d", i, i+1, tc.OrderIndex)
		}
		if tc.Weight < 1.0 {
			t.Errorf("caso %d: weight debe ser >= 1.0, got %v", i, tc.Weight)
		}
	}
	if draft.TestCases[0].Visibility != domain.TestCaseVisibilityExample {
		t.Errorf("el primer caso debía quedar como example, got %q", draft.TestCases[0].Visibility)
	}
	if draft.Language != "python" || draft.Difficulty != "easy" {
		t.Errorf("metadatos inesperados: language=%q difficulty=%q", draft.Language, draft.Difficulty)
	}

	// El runner recibió la solución de referencia con los límites del sello.
	if len(runner.calls) != 2 {
		t.Fatalf("esperaba 2 invocaciones del runner, got %d", len(runner.calls))
	}
	first := runner.calls[0]
	if first.SourceCode != draft.ReferenceSolution || first.Language != "python" {
		t.Errorf("configuración del runner inesperada: language=%q source=%q", first.Language, first.SourceCode)
	}
	if first.MemoryLimitMB != 256 || first.TimeLimitMS != 5000 {
		t.Errorf("límites del sello inesperados: %d MB / %d ms", first.MemoryLimitMB, first.TimeLimitMS)
	}
}

func TestAICopilot_GenerateExerciseDraft_ReportsWarningWhenReferenceFailsInRunner(t *testing.T) {
	exerciseJSON := `{"title":"Suma de dos enteros","description":"# Suma\nLea dos enteros e imprima su suma.","difficulty":"easy","boilerplate":"a, b = map(int, input().split())","reference_solution":"a, b = map(int, input().split())\nprint(a + b)","test_cases":[{"input":"1 2\n","expected_output":"999","visibility":"public","weight":1.0}]}`

	server := newCopilotLLMServer(t, exerciseJSON)
	svc := newEnabledCopilotSettings(t, server)

	// La solución de referencia falla con error de compilación.
	runner := &mockAICopilotRunner{verdict: domain.VerdictCE, actualOutput: ""}
	copilot := NewAICopilotService(svc, runner)

	draft, err := copilot.GenerateExerciseDraft(context.Background(), "t-1", "u-1", domain.AIGenerateExerciseRequestDTO{
		Prompt:   "Suma de dos enteros leídos por stdin",
		Language: "python",
	})
	if err != nil {
		t.Fatalf("la falla del runner no debe romper con error: %v", err)
	}
	if draft.RunnerVerifiedAll {
		t.Errorf("esperaba runner_verified_all=false")
	}
	if strings.TrimSpace(draft.VerificationWarning) == "" {
		t.Errorf("esperaba verification_warning poblado")
	}
	if !strings.Contains(draft.VerificationWarning, string(domain.VerdictCE)) {
		t.Errorf("el aviso debía mencionar el veredicto CE, got %q", draft.VerificationWarning)
	}
	if draft.TestCases[0].ExpectedOutput != "999" {
		t.Errorf("el borrador debía conservar el expected_output alucinado, got %q", draft.TestCases[0].ExpectedOutput)
	}
	if draft.TestCases[0].VerifiedByRunner {
		t.Errorf("el caso no fue verificado: verified_by_runner debe ser false")
	}

	// VerifyWithRunner=false omite el sello por decisión del docente.
	noVerify := false
	draftNoVerify, err := copilot.GenerateExerciseDraft(context.Background(), "t-1", "u-1", domain.AIGenerateExerciseRequestDTO{
		Prompt:           "Suma de dos enteros leídos por stdin",
		Language:         "python",
		VerifyWithRunner: &noVerify,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if draftNoVerify.RunnerVerifiedAll || draftNoVerify.VerificationWarning != "" {
		t.Errorf("con verify_with_runner=false no debe haber sello ni advertencia")
	}
}

func TestAICopilot_RespectsDisabledAndQuota(t *testing.T) {
	ctx := context.Background()

	// IA deshabilitada para el instituto -> 409 en el handler.
	disabled := NewAICopilotService(NewAISettingsService(newTestAISettingsRepo(`{}`)), nil)
	_, err := disabled.SuggestComponent(ctx, "t-1", "u-1", domain.AISuggestComponentRequestDTO{
		Prompt: "PyTorch para ejercicios de inteligencia artificial",
	})
	if !errors.Is(err, domain.ErrAIDisabled) {
		t.Fatalf("esperaba domain.ErrAIDisabled, got %v", err)
	}

	// Cuota horaria agotada -> 429 en el handler.
	server := newCopilotLLMServer(t, `{"component":{"name":"X","layer":"addon"},"rationale":"n/a"}`)
	settings := newEnabledCopilotSettings(t, server)
	if _, err := settings.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "",
		HourlyQuotaPerUser: 1,
		TimeoutSeconds:     10,
	}); err != nil {
		t.Fatalf("unexpected error ajustando la cuota: %v", err)
	}
	// Consumimos el único punto disponible de la ventana en curso.
	if err := settings.ConsumeQuota("t-1", "u-1", 1); err != nil {
		t.Fatalf("unexpected error consumiendo cuota: %v", err)
	}

	copilot := NewAICopilotService(settings, nil)
	_, err = copilot.SuggestComponent(ctx, "t-1", "u-1", domain.AISuggestComponentRequestDTO{
		Prompt: "PyTorch para ejercicios de inteligencia artificial",
	})
	if !errors.Is(err, domain.ErrAIQuotaExceeded) {
		t.Fatalf("esperaba domain.ErrAIQuotaExceeded, got %v", err)
	}
}
