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

type stubComponentRepoForTesting struct {
	comps []*domain.TemplateComponent
}

func (s *stubComponentRepoForTesting) UpdateOfficialTemplate(ctx context.Context, tenantID, templateID, adminID string, dto domain.UpdateOfficialTemplateDTO) (*domain.AdminTemplateReviewItem, error) {
	return nil, nil
}
func (s *stubComponentRepoForTesting) ListTemplateComponents(ctx context.Context, tenantID, layer string, includeInactive bool) ([]*domain.TemplateComponent, error) {
	return s.comps, nil
}
func (s *stubComponentRepoForTesting) GetTemplateComponentByIDOrSlug(ctx context.Context, tenantID, idOrSlug string) (*domain.TemplateComponent, error) {
	return nil, nil
}
func (s *stubComponentRepoForTesting) CreateTemplateComponent(ctx context.Context, tenantID string, comp *domain.TemplateComponent) (*domain.TemplateComponent, error) {
	return comp, nil
}
func (s *stubComponentRepoForTesting) UpdateTemplateComponent(ctx context.Context, tenantID, id string, dto domain.UpdateTemplateComponentDTO, synthesizedLayer string) (*domain.TemplateComponent, error) {
	return nil, nil
}
func (s *stubComponentRepoForTesting) DeleteTemplateComponent(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestAICopilot_SuggestTemplate_ReturnsProposalWithSanitizedSlugs(t *testing.T) {
	templateJSON := `{"name":"Algoritmos con Python y PostgreSQL","description":"Juez automático para estructuras de datos","runtime_slug":"python-312","web_platform_slug":"","addon_slugs":["fastapi","python-sqlalchemy"],"database_slug":"sqlite","setup_script":"export PIP_NO_CACHE=1","base_ram_mb":768,"rationale":"Python 3.12 con FastAPI y SQLAlchemy.","missing_components":[{"name":"Celery","layer":"addon","category":"toolkit","reason":"Falta soporte asíncrono"}]}`

	server := newCopilotLLMServer(t, "```json\n"+templateJSON+"\n```")
	svc := newEnabledCopilotSettings(t, server)
	mockRepo := &stubComponentRepoForTesting{
		comps: []*domain.TemplateComponent{
			{ID: "rt-1", Slug: "python-312", Name: "Python 3.12", Layer: "runtime", SubCategory: "backend", IsActive: true},
			{ID: "ad-1", Slug: "fastapi", Name: "FastAPI + Uvicorn", Layer: "addon", SubCategory: "framework", CompatibleRuntimes: []string{"python"}, IsActive: true},
			{ID: "ad-2", Slug: "python-sqlalchemy", Name: "SQLAlchemy + Drivers", Layer: "addon", SubCategory: "orm", CompatibleRuntimes: []string{"python"}, IsActive: true},
		},
	}
	copilot := NewAICopilotService(svc, nil, mockRepo)

	resp, err := copilot.SuggestTemplate(context.Background(), "t-1", "u-1", domain.AISuggestTemplateRequestDTO{
		Prompt:            "Entorno para estructuras de datos con Python, FastAPI y SQLite",
		TargetEnvironment: "JUEZ_EFIMERO",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RuntimeSlug != "python-312" {
		t.Errorf("runtime_slug inesperado: %q", resp.RuntimeSlug)
	}
	// Slugs saneados: minúsculas y sin duplicados.
	if len(resp.AddonSlugs) != 2 || resp.AddonSlugs[0] != "fastapi" || resp.AddonSlugs[1] != "python-sqlalchemy" {
		t.Errorf("addon_slugs saneado inesperado: %#v", resp.AddonSlugs)
	}
	if len(resp.MissingComponents) != 1 || resp.MissingComponents[0].Name != "Celery" {
		t.Errorf("missing_components inesperado: %#v", resp.MissingComponents)
	}
	if resp.DatabaseSlug != "sqlite" {
		t.Errorf("database_slug inesperado: %q", resp.DatabaseSlug)
	}
	if resp.BaseRamMB != 768 {
		t.Errorf("base_ram_mb inesperado: %d", resp.BaseRamMB)
	}
	if strings.TrimSpace(resp.Rationale) == "" {
		t.Errorf("esperaba una rationale técnica en español")
	}

	// Entorno objetivo inválido -> error de validación.
	if _, err := copilot.SuggestTemplate(context.Background(), "t-1", "u-1", domain.AISuggestTemplateRequestDTO{
		Prompt:            "un entorno de prueba",
		TargetEnvironment: "MODO_RARO",
	}); err == nil {
		t.Errorf("esperaba rechazo de target_environment inválido")
	}

	// Prompt fuera de rango -> error de validación.
	if _, err := copilot.SuggestTemplate(context.Background(), "t-1", "u-1", domain.AISuggestTemplateRequestDTO{
		Prompt: "ab",
	}); err == nil {
		t.Errorf("esperaba rechazo de prompt menor a 3 caracteres")
	}
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

// TestAICopilot_RepairTemplateRepairsDockerfileAndTools cubre la autorreparación
// diagnóstica: corrige el Dockerfile, fuerza el build local cuando hay capas
// RUN y devuelve la causa raíz en español.
func TestAICopilot_RepairTemplateRepairsDockerfileAndTools(t *testing.T) {
	repaired := `{"docker_image":"python:3.12-slim-bookworm","dockerfile_content":"FROM python:3.12-slim-bookworm\nRUN apt-get update && apt-get install -y --no-install-recommends gcc make\nCMD [\"python3\"]","setup_script":"set -euo pipefail\npip install --no-cache-dir numpy==2.1.0","tools_declared":["python3","pip","gcc","pip","  "],"entrypoint":"python3 /opt/judge/run.py","base_ram_mb":5120,"rationale":"El contenedor falló porque gcc no estaba instalado y la imagen llegaba a 8 GB de RAM."}`
	server := newCopilotLLMServer(t, repaired)
	svc := newEnabledCopilotSettings(t, server)
	copilot := NewAICopilotService(svc, nil)

	resp, err := copilot.RepairTemplate(context.Background(), "t-1", "u-1", domain.AIRepairTemplateRequestDTO{
		Name:              "Laboratorio de Estadística",
		Description:       "Python con NumPy",
		TargetEnvironment: "python",
		DockerImage:       "python:3.12-slim",
		DockerfileContent: "FROM python:3.12-slim\nCMD [\"python3\"]",
		SetupScript:       "set -euo pipefail",
		ToolsDeclared:     []string{"python3", "pip"},
		Entrypoint:        "python /app/main.py",
		BaseRamMB:         4096,
		ErrorMessage:      "exec format error: gcc: not found",
		FailedTools:       []string{"gcc"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.DockerImage != "solv-local/stack:laboratorio-de-estadistica" {
		t.Errorf("con capas RUN la imagen debe forzar build local, got %q", resp.DockerImage)
	}
	if strings.Contains(strings.ToLower(resp.DockerfileContent), ":latest") {
		t.Errorf("el Dockerfile reparado no debe llevar :latest: %q", resp.DockerfileContent)
	}
	if !strings.Contains(resp.DockerfileContent, "FROM python:3.12-slim-bookworm") || !strings.Contains(resp.DockerfileContent, "gcc") {
		t.Errorf("Dockerfile reparado inesperado: %q", resp.DockerfileContent)
	}
	if resp.SetupScript == "" || !strings.Contains(resp.SetupScript, "numpy") {
		t.Errorf("setup_script inesperado: %q", resp.SetupScript)
	}
	wantTools := []string{"python3", "pip", "gcc"}
	if strings.Join(resp.ToolsDeclared, ",") != strings.Join(wantTools, ",") {
		t.Errorf("tools_declared depurado inesperado: %v", resp.ToolsDeclared)
	}
	if resp.Entrypoint != "python3 /opt/judge/run.py" {
		t.Errorf("entrypoint inesperado: %q", resp.Entrypoint)
	}
	if resp.BaseRamMB != 4096 {
		t.Errorf("base_ram_mb debe quedar dentro de 256..4096, got %d", resp.BaseRamMB)
	}
	if strings.TrimSpace(resp.Rationale) == "" {
		t.Errorf("esperaba una rationale con la causa raíz")
	}
	if resp.LatencyMS < 0 {
		t.Errorf("latency_ms inválida: %d", resp.LatencyMS)
	}
}

// TestAICopilot_RepairTemplateGuardrails verifica los guardrails deterministas.
func TestAICopilot_RepairTemplateGuardrails(t *testing.T) {
	req := domain.AIRepairTemplateRequestDTO{
		Name:              "Stack Web",
		DockerImage:       "node:20-bookworm-slim",
		DockerfileContent: "FROM node:20-bookworm-slim\nCMD [\"node\"]",
		ToolsDeclared:     []string{"node"},
		Entrypoint:        "node server.js",
		BaseRamMB:         512,
		ErrorMessage:      "no arranca",
	}

	// 1. Imagen :latest -> rechazo.
	latest := newCopilotLLMServer(t, `{"docker_image":"node:latest","dockerfile_content":"FROM node:20-bookworm-slim","rationale":"cambio de imagen"}`)
	copilot := NewAICopilotService(newEnabledCopilotSettings(t, latest), nil)
	_, err := copilot.RepairTemplate(context.Background(), "t-1", "u-1", req)
	if err == nil {
		t.Fatalf("esperaba rechazo de imagen :latest")
	}
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "base_image_latest" {
		t.Errorf("esperaba AISettingsValidationError base_image_latest, got %v", err)
	}

	// 2. Dockerfile con FROM :latest -> rechazo.
	latestFROM := newCopilotLLMServer(t, `{"docker_image":"node:20-bookworm-slim","dockerfile_content":"FROM node:latest","rationale":"sin cambios"}`)
	copilotFROM := NewAICopilotService(newEnabledCopilotSettings(t, latestFROM), nil)
	_, err = copilotFROM.RepairTemplate(context.Background(), "t-1", "u-1", req)
	if !errors.As(err, &vErr) || vErr.Code != "base_image_latest" {
		t.Errorf("esperaba base_image_latest desde el Dockerfile, got %v", err)
	}

	// 3. Sin rationale -> rechazo.
	withoutRationale := newCopilotLLMServer(t, `{"docker_image":"node:20-bookworm-slim","dockerfile_content":"FROM node:20-bookworm-slim"}`)
	copilotNoRationale := NewAICopilotService(newEnabledCopilotSettings(t, withoutRationale), nil)
	_, err = copilotNoRationale.RepairTemplate(context.Background(), "t-1", "u-1", req)
	if !errors.As(err, &vErr) || vErr.Code != "ai_response_invalid" {
		t.Errorf("esperaba ai_response_invalid sin rationale, got %v", err)
	}

	// 4. Sin capas RUN la imagen fijada se conserva tal cual.
	withoutRUN := newCopilotLLMServer(t, `{"docker_image":"node:20-bookworm-slim","dockerfile_content":"FROM node:20-bookworm-slim\nCMD [\"node\"]","rationale":"solo se corrigió el entrypoint","tools_declared":["node"]}`)
	copilotNoRUN := NewAICopilotService(newEnabledCopilotSettings(t, withoutRUN), nil)
	resp, err := copilotNoRUN.RepairTemplate(context.Background(), "t-1", "u-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DockerImage != "node:20-bookworm-slim" {
		t.Errorf("la imagen fijada debía conservarse, got %q", resp.DockerImage)
	}
	if resp.BaseRamMB != 512 {
		t.Errorf("base_ram_mb debía conservar el valor del snapshot, got %d", resp.BaseRamMB)
	}

	// 5. Memoria fuera de rango -> recorte determinista a 256..4096.
	outOfRange := newCopilotLLMServer(t, `{"docker_image":"node:20-bookworm-slim","dockerfile_content":"FROM node:20-bookworm-slim","base_ram_mb":99999,"rationale":"memoria desajustada"}`)
	copilotRAM := NewAICopilotService(newEnabledCopilotSettings(t, outOfRange), nil)
	respRAM, err := copilotRAM.RepairTemplate(context.Background(), "t-1", "u-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respRAM.BaseRamMB != 4096 {
		t.Errorf("esperaba recorte a 4096, got %d", respRAM.BaseRamMB)
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
