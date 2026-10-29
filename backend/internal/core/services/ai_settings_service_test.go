package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// mockAITenantRepo implementa domain.TenantRepository con config en memoria.
type mockAITenantRepo struct {
	configs map[string][]byte
}

func (m *mockAITenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return &domain.Tenant{ID: id, Config: cfg}, nil
}

func (m *mockAITenantRepo) UpdateConfig(ctx context.Context, id string, config []byte) error {
	m.configs[id] = config
	return nil
}

func (m *mockAITenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return nil, nil
}

func (m *mockAITenantRepo) GetAll(ctx context.Context) ([]*domain.Tenant, error) {
	return nil, nil
}

func (m *mockAITenantRepo) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	return nil
}

func (m *mockAITenantRepo) GetMaintenance(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	return &domain.MaintenanceStatus{}, nil
}

func newTestAISettingsRepo(config string) *mockAITenantRepo {
	return &mockAITenantRepo{configs: map[string][]byte{"t-1": []byte(config)}}
}

// rewriteAITestTransport redirige cualquier petición (incluida la URL oficial
// cloud forzada por el escudo anti-SSRF) hacia el servidor httptest local.
type rewriteAITestTransport struct {
	base *url.URL
}

func (t *rewriteAITestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.base.Scheme
	clone.URL.Host = t.base.Host
	clone.Host = t.base.Host
	return http.DefaultTransport.RoundTrip(clone)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url inválida %q: %v", raw, err)
	}
	return u
}

func TestAISettingsDefaultsWhenTenantHasNoAISettings(t *testing.T) {
	repo := newTestAISettingsRepo(`{"tenant_primary_color":"#15803D"}`)
	svc := NewAISettingsService(repo)

	got, err := svc.Get(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	def := domain.DefaultAISettings()
	if got.Enabled != false {
		t.Errorf("esperado enabled=false por defecto, got %v", got.Enabled)
	}
	if got.Provider != def.Provider || got.Model != def.Model || got.BaseURL != def.BaseURL {
		t.Errorf("esperado defaults %+v, got %+v", def, got)
	}
	if got.HourlyQuotaPerUser != 15 || got.TimeoutSeconds != 60 {
		t.Errorf("esperado cuota 15 y timeout 60, got %d/%d", got.HourlyQuotaPerUser, got.TimeoutSeconds)
	}
	if got.EncryptedAPIKey != "" || got.APIKeyMasked != "" {
		t.Errorf("esperado sin API Key por defecto, got %q/%q", got.EncryptedAPIKey, got.APIKeyMasked)
	}
}

func TestAISettingsEncryptMaskAndConserveAPIKey(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	plain := "AIzaSyDummyKeyValue0123456789"
	resp, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            false,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		APIKey:             plain,
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.HasAPIKey {
		t.Errorf("esperado has_api_key=true tras guardar la clave")
	}
	wantMasked := plain[:4] + "..." + plain[len(plain)-4:]
	if resp.APIKeyMasked != wantMasked {
		t.Errorf("esperado máscara %q, got %q", wantMasked, resp.APIKeyMasked)
	}

	// La respuesta pública nunca debe filtrar la clave cifrada ni la texto plano.
	rawDTO, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(rawDTO), "encrypted_api_key") || strings.Contains(string(rawDTO), plain) {
		t.Fatalf("la respuesta pública filtra la API Key: %s", string(rawDTO))
	}

	// El almacén guarda ciphertext reversible, no la clave en texto plano.
	stored, err := svc.Get(ctx, "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.EncryptedAPIKey == "" || strings.Contains(stored.EncryptedAPIKey, plain) {
		t.Fatalf("esperado ciphertext AES-GCM en encrypted_api_key, got %q", stored.EncryptedAPIKey)
	}
	decrypted, err := svc.decryptAPIKey(stored.EncryptedAPIKey)
	if err != nil {
		t.Fatalf("unexpected error descifrando: %v", err)
	}
	if decrypted != plain {
		t.Fatalf("el descifrado no recupera la clave: got %q", decrypted)
	}

	// api_key vacía y clear_api_key=false conserva la clave previa.
	kept, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            false,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		APIKey:             "",
		ClearAPIKey:        false,
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !kept.HasAPIKey || kept.APIKeyMasked != wantMasked {
		t.Fatalf("esperaba conservar la clave previa, got has=%v masked=%q", kept.HasAPIKey, kept.APIKeyMasked)
	}

	// clear_api_key=true limpia la clave.
	cleared, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            false,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		ClearAPIKey:        true,
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleared.HasAPIKey || cleared.APIKeyMasked != "" {
		t.Fatalf("esperaba clave limpiada, got has=%v masked=%q", cleared.HasAPIKey, cleared.APIKeyMasked)
	}

	// Enmascaramiento de claves cortas.
	if got := maskAPIKey("12345678"); got != "****" {
		t.Errorf("esperado **** para clave de 8 caracteres, got %q", got)
	}
	if got := maskAPIKey("abcdefghij"); got != "abcd...ghij" {
		t.Errorf("esperado abcd...ghij, got %q", got)
	}
}

func TestAISettingsAntiSSRFShield(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	blocked := []string{
		"http://localhost:11434/v1",
		"http://127.0.0.1:8080",
		"http://169.254.169.254/latest",
		"http://postgres:5432",
	}
	for _, raw := range blocked {
		_, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
			Enabled:            false,
			Provider:           domain.AIProviderOllamaLAN,
			Model:              "qwen2.5-coder:7b",
			BaseURL:            raw,
			HourlyQuotaPerUser: 15,
			TimeoutSeconds:     30,
		})
		if err == nil {
			t.Errorf("esperaba rechazo para %q", raw)
			continue
		}
		var vErr *AISettingsValidationError
		if !errors.As(err, &vErr) {
			t.Errorf("esperaba AISettingsValidationError para %q, got %T: %v", raw, err, err)
		}
	}

	allowed := []string{
		"http://192.168.1.50:11434/v1",
		"https://ollama.universidad.edu.bo/v1",
	}
	for _, raw := range allowed {
		resp, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
			Enabled:            false,
			Provider:           domain.AIProviderOllamaLAN,
			Model:              "qwen2.5-coder:7b",
			BaseURL:            raw,
			HourlyQuotaPerUser: 15,
			TimeoutSeconds:     30,
		})
		if err != nil {
			t.Errorf("esperaba aceptar %q, got %v", raw, err)
			continue
		}
		if resp.BaseURL != raw {
			t.Errorf("esperaba base_url %q, got %q", raw, resp.BaseURL)
		}
	}

	// Proveedor cloud: la URL oficial se fuerza y se ignora cualquier base_url del cliente.
	resp, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            false,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		BaseURL:            "http://evil.example.com/v1",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	official, _ := domain.OfficialAIBaseURL(domain.AIProviderOpenAI)
	if resp.BaseURL != official {
		t.Errorf("esperaba URL oficial cloud %q, got %q", official, resp.BaseURL)
	}

	// Proveedor fuera del catálogo.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider: "azure",
		Model:    "gpt-4o",
	}); err == nil {
		t.Errorf("esperaba rechazo de proveedor inválido")
	}
}

func TestAISettingsActivationRequiresCloudAPIKey(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	_, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err == nil {
		t.Fatalf("esperaba error de activación sin API Key")
	}
	if err.Error() != "Se requiere una API Key para habilitar un proveedor en la nube" {
		t.Errorf("mensaje de activación inesperado: %v", err)
	}

	// Con API Key la activación procede.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		APIKey:             "AIzaSyDummyKeyValue0123456789",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	}); err != nil {
		t.Fatalf("esperaba activación con API Key, got %v", err)
	}

	// El servidor local LAN no exige API Key para activarse.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOllamaLAN,
		Model:              "qwen2.5-coder:7b",
		BaseURL:            "http://10.20.1.50:11434/v1",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	}); err != nil {
		t.Fatalf("esperaba activación de ollama_lan sin API Key, got %v", err)
	}

	// Rangos de cuota y timeout.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider:           domain.AIProviderOllamaLAN,
		Model:              "qwen2.5-coder:7b",
		BaseURL:            "http://10.20.1.50:11434/v1",
		HourlyQuotaPerUser: 500,
		TimeoutSeconds:     30,
	}); err == nil {
		t.Errorf("esperaba rechazo de cuota fuera de rango (1..200)")
	}
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider:           domain.AIProviderOllamaLAN,
		Model:              "qwen2.5-coder:7b",
		BaseURL:            "http://10.20.1.50:11434/v1",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     2,
	}); err == nil {
		t.Errorf("esperaba rechazo de timeout fuera de rango (5..120)")
	}
}

func TestAISettingsConsumeQuotaBlocksOverLimit(t *testing.T) {
	svc := NewAISettingsService(newTestAISettingsRepo(`{}`))

	for i := 0; i < 2; i++ {
		if err := svc.ConsumeQuota("t-1", "u-1", 2); err != nil {
			t.Fatalf("invocación %d debía permitirse: %v", i+1, err)
		}
	}
	if err := svc.ConsumeQuota("t-1", "u-1", 2); !errors.Is(err, domain.ErrAIQuotaExceeded) {
		t.Fatalf("esperaba domain.ErrAIQuotaExceeded en la tercera invocación, got %v", err)
	}

	// Otro usuario y otro tenant tienen su propia ventana.
	if err := svc.ConsumeQuota("t-1", "u-2", 2); err != nil {
		t.Errorf("otro usuario debía tener cuota propia: %v", err)
	}
	if err := svc.ConsumeQuota("t-2", "u-1", 2); err != nil {
		t.Errorf("otro tenant debía tener cuota propia: %v", err)
	}
}

func TestAISettingsTestConnectionWithHTTPTestServer(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotResponseFormat string
	var gotTemperature float64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		var body struct {
			Model          string  `json:"model"`
			Temperature    float64 `json:"temperature"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotResponseFormat = body.ResponseFormat.Type
		gotTemperature = body.Temperature

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	resp, err := svc.TestConnection(context.Background(), "t-1", domain.TestAIConnectionRequestDTO{
		Provider: domain.AIProviderOpenAI,
		Model:    "gpt-4o-mini",
		APIKey:   "sk-test-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.OK {
		t.Fatalf("esperaba ok=true, got message=%q", resp.Message)
	}
	if resp.Provider != domain.AIProviderOpenAI || resp.Model != "gpt-4o-mini" {
		t.Errorf("provider/model inesperados: %+v", resp)
	}
	if resp.LatencyMS < 0 {
		t.Errorf("latency_ms inválida: %d", resp.LatencyMS)
	}
	if resp.Message != "Conexión verificada correctamente con el motor de IA." {
		t.Errorf("mensaje inesperado: %q", resp.Message)
	}
	if gotAuth != "Bearer sk-test-123" {
		t.Errorf("esperaba Authorization Bearer sk-test-123, got %q", gotAuth)
	}
	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Errorf("esperaba POST a /chat/completions, got %q", gotPath)
	}
	if gotResponseFormat != "json_object" {
		t.Errorf("esperaba response_format json_object, got %q", gotResponseFormat)
	}
	if gotTemperature != 0 {
		t.Errorf("esperaba temperature 0 en la prueba de conexión, got %v", gotTemperature)
	}

	// Fallo del motor remoto: ok=false sin error de validación local.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream boom", http.StatusInternalServerError)
	}))
	defer failing.Close()

	svcFailing := NewAISettingsService(repo).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, failing.URL)})
	failed, err := svcFailing.TestConnection(context.Background(), "t-1", domain.TestAIConnectionRequestDTO{
		Provider: domain.AIProviderOpenAI,
		Model:    "gpt-4o-mini",
		APIKey:   "sk-test-123",
	})
	if err != nil {
		t.Fatalf("un fallo del motor no debe ser error de validación: %v", err)
	}
	if failed.OK {
		t.Errorf("esperaba ok=false ante HTTP 500 del motor")
	}
	if strings.TrimSpace(failed.Message) == "" {
		t.Errorf("esperaba mensaje de fallo descriptivo")
	}
}

func TestAISettingsCompleteJSONStructuredAndQuota(t *testing.T) {
	var gotBody struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"verdict\":\"ok\"}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 2,
		TimeoutSeconds:     10,
	}); err != nil {
		t.Fatalf("unexpected error configurando IA: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	content, latencyMS, err := svc.CompleteJSON(ctx, "t-1", "u-1", "feedback_rubrica", "Sos un corrector.", "Evalua esto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != `{"verdict":"ok"}` {
		t.Errorf("contenido inesperado: %q", content)
	}
	if latencyMS < 0 {
		t.Errorf("latencia inválida: %d", latencyMS)
	}
	if gotBody.Model != "gpt-4o-mini" || gotBody.Temperature != 0.2 || gotBody.ResponseFormat.Type != "json_object" {
		t.Errorf("payload OpenAI-incompatible: %+v", gotBody)
	}
	if len(gotBody.Messages) != 2 || gotBody.Messages[0].Role != "system" || gotBody.Messages[1].Role != "user" {
		t.Errorf("mensajes inesperados: %+v", gotBody.Messages)
	}

	// Segunda llamada dentro de la cuota; tercera excedida.
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-1", "feedback_rubrica", "sys", "user"); err != nil {
		t.Fatalf("segunda llamada debía permitirse: %v", err)
	}
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-1", "feedback_rubrica", "sys", "user"); !errors.Is(err, domain.ErrAIQuotaExceeded) {
		t.Fatalf("esperaba domain.ErrAIQuotaExceeded, got %v", err)
	}
}

func TestAISettingsProviderCatalogOpenRouterDeepSeek(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	cases := []struct {
		provider string
		official string
	}{
		{domain.AIProviderOpenRouter, "https://openrouter.ai/api/v1"},
		{domain.AIProviderDeepSeek, "https://api.deepseek.com"},
	}
	for _, tc := range cases {
		if !domain.IsValidAIProvider(tc.provider) {
			t.Errorf("esperaba %q válido en el catálogo", tc.provider)
		}
		if !domain.IsCloudAIProvider(tc.provider) {
			t.Errorf("esperaba %q como proveedor cloud", tc.provider)
		}
		official, ok := domain.OfficialAIBaseURL(tc.provider)
		if !ok || official != tc.official {
			t.Errorf("URL oficial de %q inesperada: %q (ok=%v)", tc.provider, official, ok)
		}

		resp, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
			Enabled:            true,
			Provider:           tc.provider,
			Model:              "m",
			APIKey:             "sk-test-123",
			HourlyQuotaPerUser: 15,
			TimeoutSeconds:     30,
		})
		if err != nil {
			t.Errorf("Update(%q) falló: %v", tc.provider, err)
			continue
		}
		if resp.BaseURL != tc.official {
			t.Errorf("esperaba base_url %q, got %q", tc.official, resp.BaseURL)
		}
	}

	// groq fue retirado del catálogo.
	if domain.IsValidAIProvider("groq") {
		t.Errorf("groq no debe ser válido tras su retiro")
	}
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{Provider: "groq", Model: "m"}); err == nil {
		t.Errorf("esperaba rechazo de groq retirado")
	}
}

func TestListAIModelsOpenRouterSortingAndTiers(t *testing.T) {
	catalog := `{
		"data": [
			{"id": "anthropic/claude-3-haiku", "name": "Claude 3 Haiku", "pricing": {"prompt": "0.00025", "completion": "0.00125"}},
			{"id": "google/gemini-flash-1.5:free", "name": "Gemini Flash 1.5 Free", "pricing": {"prompt": "0", "completion": "0"}},
			{"id": "meta-llama/llama-3-8b-instruct:free", "name": "Llama 3 8B Free"},
			{"id": "openai/gpt-4o", "name": "GPT-4o", "pricing": {"prompt": "0.005", "completion": "0.015"}},
			{"id": "mistralai/mistral-7b-instruct", "name": "Mistral 7B", "pricing": {"prompt": "0.0", "completion": "0.0"}},
			{"id": "openai/text-embedding-3-small", "name": "Embedding"}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/models") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(catalog))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	resp, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenRouter,
		APIKey:   "sk-or-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Provider != domain.AIProviderOpenRouter {
		t.Errorf("esperaba provider openrouter, got %q", resp.Provider)
	}
	if resp.Cached {
		t.Errorf("esperaba cached=false en la primera llamada")
	}
	// Deben quedar 5 (text-embedding-3-small excluido)
	if resp.Count != 5 || len(resp.Models) != 5 {
		t.Fatalf("esperaba 5 modelos, got count=%d len=%d", resp.Count, len(resp.Models))
	}

	// Orden esperado: free primero (alfabético), luego paid (alfabético)
	expectedIDs := []string{
		"google/gemini-flash-1.5:free",
		"meta-llama/llama-3-8b-instruct:free",
		"mistralai/mistral-7b-instruct",
		"anthropic/claude-3-haiku",
		"openai/gpt-4o",
	}

	for i, exp := range expectedIDs {
		if resp.Models[i].ID != exp {
			t.Errorf("posición %d: esperaba id %q, got %q", i, exp, resp.Models[i].ID)
		}
	}

	// Verificar tiers y hints
	for i := 0; i < 3; i++ {
		if resp.Models[i].Tier != "free" || resp.Models[i].PricingHint != "Gratis" {
			t.Errorf("modelo %s debía ser free/Gratis, got tier=%q hint=%q", resp.Models[i].ID, resp.Models[i].Tier, resp.Models[i].PricingHint)
		}
	}
	for i := 3; i < 5; i++ {
		if resp.Models[i].Tier != "paid" || resp.Models[i].PricingHint != "De pago" {
			t.Errorf("modelo %s debía ser paid/De pago, got tier=%q hint=%q", resp.Models[i].ID, resp.Models[i].Tier, resp.Models[i].PricingHint)
		}
	}
}

func TestListAIModelsGeminiTrimmingAndExclusion(t *testing.T) {
	catalog := `{
		"data": [
			{"id": "models/gemini-2.5-pro", "name": "Gemini 2.5 Pro"},
			{"id": "models/gemini-2.5-flash", "name": "Gemini 2.5 Flash"},
			{"id": "models/text-embedding-004"},
			{"id": "models/imagen-3.0-generate-002"},
			{"id": "models/veo-2.0-generate-001"},
			{"id": "models/aqa"},
			{"id": "models/whisper-large"}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/models") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(catalog))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	resp, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderGemini,
		APIKey:   "AIzaSy-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Count != 2 || len(resp.Models) != 2 {
		t.Fatalf("esperaba 2 modelos filtrados, got count=%d", resp.Count)
	}

	// Orden alfabético
	if resp.Models[0].ID != "gemini-2.5-flash" || resp.Models[1].ID != "gemini-2.5-pro" {
		t.Errorf("orden alfabético o recorte de models/ falló: %+v", resp.Models)
	}

	for _, m := range resp.Models {
		if m.Tier != "by_account" || m.PricingHint != "Según plan de la cuenta" {
			t.Errorf("esperaba tier by_account y hint según plan, got tier=%q hint=%q", m.Tier, m.PricingHint)
		}
	}
}

func TestListAIModelsCachingAndInvalidation(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": [{"id": "gpt-4o-mini"}]}`))
	}))
	defer server.Close()

	// Tenant con API Key configurada
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	// Guardar configuración para tener EncryptedAPIKey persistida
	_, err := svc.Update(context.Background(), "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-stored",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 1. Primera llamada sin APIKey en memoria -> consulta remota (calls=1)
	r1, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenAI,
	})
	if err != nil || r1.Cached || calls != 1 {
		t.Fatalf("llamada 1: err=%v cached=%v calls=%d", err, r1.Cached, calls)
	}

	// 2. Segunda llamada -> viene de caché (calls=1, cached=true)
	r2, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenAI,
	})
	if err != nil || !r2.Cached || calls != 1 {
		t.Fatalf("llamada 2: err=%v cached=%v calls=%d", err, r2.Cached, calls)
	}

	// 3. Tercera llamada con Refresh=true -> bypass caché (calls=2, cached=false)
	r3, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenAI,
		Refresh:  true,
	})
	if err != nil || r3.Cached || calls != 2 {
		t.Fatalf("llamada 3: err=%v cached=%v calls=%d", err, r3.Cached, calls)
	}

	// 4. Cuarta llamada con APIKey explícita -> bypass caché y no escribe en caché (calls=3)
	r4, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenAI,
		APIKey:   "sk-override",
	})
	if err != nil || r4.Cached || calls != 3 {
		t.Fatalf("llamada 4: err=%v cached=%v calls=%d", err, r4.Cached, calls)
	}

	// 5. Quinta llamada usando la clave almacenada -> sigue en caché de la llamada 3 (calls=3, cached=true)
	r5, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenAI,
	})
	if err != nil || !r5.Cached || calls != 3 {
		t.Fatalf("llamada 5: err=%v cached=%v calls=%d", err, r5.Cached, calls)
	}

	// 6. Invalida caché al ejecutar Update()
	_, err = svc.Update(context.Background(), "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	})
	if err != nil {
		t.Fatalf("Update 2 failed: %v", err)
	}

	// 7. Sexta llamada -> caché invalidada, nueva consulta remota (calls=4, cached=false)
	r6, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOpenAI,
	})
	if err != nil || r6.Cached || calls != 4 {
		t.Fatalf("llamada 6: err=%v cached=%v calls=%d", err, r6.Cached, calls)
	}
}

func TestListAIModelsMissingAPIKey(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)

	// Cloud sin APIKey ni almacenada ni en memoria -> api_key_required
	_, err := svc.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderDeepSeek,
	})
	if err == nil {
		t.Fatalf("esperaba error api_key_required para cloud sin key")
	}
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "api_key_required" {
		t.Fatalf("esperaba código api_key_required, got %v", err)
	}

	// Ollama LAN sin APIKey -> permitido
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": [{"id": "qwen2.5-coder:7b"}]}`))
	}))
	defer server.Close()

	svcLAN := NewAISettingsService(repo).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})
	resp, err := svcLAN.ListModels(context.Background(), "t-1", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOllamaLAN,
		BaseURL:  "http://192.168.1.100:11434/v1",
	})
	if err != nil {
		t.Fatalf("ollama_lan falló sin apiKey: %v", err)
	}
	if resp.Count != 1 || resp.Models[0].ID != "qwen2.5-coder:7b" {
		t.Errorf("modelo inesperado: %+v", resp)
	}
	if resp.Models[0].Tier != "local" || resp.Models[0].PricingHint != "Local · Sin costo de API" {
		t.Errorf("tier/hint inesperados para ollama_lan: %+v", resp.Models[0])
	}
}

func TestAISettingsCustomProvidersCRUD(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// 1. Creación de un proveedor personalizado
	resp, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito1",
				Label:   "Paquito AI",
				IconURL: "https://cdn.paquito.ai/logo.svg",
				BaseURL: "https://api.paquito.ai/v1",
				Model:   "paquito-coder-7b",
				APIKey:  "sk-paquito-0123456789",
			},
		},
	})
	if err != nil {
		t.Fatalf("falló creación de custom provider: %v", err)
	}

	if len(resp.CustomProviders) != 1 {
		t.Fatalf("esperaba 1 custom provider, got %d", len(resp.CustomProviders))
	}
	cp := resp.CustomProviders[0]
	if cp.ID != "custom_paquito1" || cp.Label != "Paquito AI" || cp.Model != "paquito-coder-7b" {
		t.Errorf("datos inesperados en custom provider: %+v", cp)
	}
	if !cp.HasAPIKey || cp.APIKeyMasked != "sk-p...6789" {
		t.Errorf("has_api_key o masked inesperado: has=%v masked=%q", cp.HasAPIKey, cp.APIKeyMasked)
	}

	// 2. Edición de custom provider existente y agregado de otro
	resp2, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: "custom_paquito1",
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito1",
				Label:   "Paquito AI Pro",
				IconURL: "https://cdn.paquito.ai/logo.svg",
				BaseURL: "https://api.paquito.ai/v1",
				Model:   "paquito-coder-14b",
				// No envía API Key: debe conservar la previa
			},
			{
				ID:      "custom_mistral",
				Label:   "Mistral AI",
				BaseURL: "https://api.mistral.ai/v1",
				Model:   "mistral-large",
				APIKey:  "sk-mistral-9876543210",
			},
		},
	})
	if err != nil {
		t.Fatalf("falló edición y agregado de custom providers: %v", err)
	}
	if len(resp2.CustomProviders) != 2 {
		t.Fatalf("esperaba 2 custom providers, got %d", len(resp2.CustomProviders))
	}
	p1 := resp2.CustomProviders[0]
	if p1.Label != "Paquito AI Pro" || p1.Model != "paquito-coder-14b" || !p1.HasAPIKey {
		t.Errorf("custom_paquito1 debía conservar la clave y actualizar label/model: %+v", p1)
	}
	p2 := resp2.CustomProviders[1]
	if p2.ID != "custom_mistral" || p2.Label != "Mistral AI" || !p2.HasAPIKey {
		t.Errorf("custom_mistral no se configuró correctamente: %+v", p2)
	}
	// El proveedor activo es custom_paquito1
	if resp2.Provider != "custom_paquito1" || resp2.ProviderLabel != "Paquito AI Pro" {
		t.Errorf("provider o provider_label inesperados: prov=%q label=%q", resp2.Provider, resp2.ProviderLabel)
	}

	// 3. Eliminación: se envía solo custom_mistral y se cambia provider activo a custom_mistral
	resp3, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: "custom_mistral",
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_mistral",
				Label:   "Mistral AI",
				BaseURL: "https://api.mistral.ai/v1",
				Model:   "mistral-large",
			},
		},
	})
	if err != nil {
		t.Fatalf("falló eliminación de custom_paquito1: %v", err)
	}
	if len(resp3.CustomProviders) != 1 || resp3.CustomProviders[0].ID != "custom_mistral" {
		t.Fatalf("esperaba solo custom_mistral tras eliminación, got %+v", resp3.CustomProviders)
	}

	// 4. Validación: más de 20 proveedores
	tooMany := make([]domain.AICustomProviderDTO, 21)
	for i := 0; i < 21; i++ {
		tooMany[i] = domain.AICustomProviderDTO{
			ID:      domain.AIProviderGemini, // will also fail on reserved if count not checked first
			Label:   "P",
			BaseURL: "https://api.test.com/v1",
		}
	}
	_, errLimit := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider:        domain.AIProviderGemini,
		CustomProviders: tooMany,
	})
	if errLimit == nil {
		t.Fatalf("esperaba error por superar límite de 20 proveedores")
	}
	var vErr *AISettingsValidationError
	if !errors.As(errLimit, &vErr) || vErr.Code != "custom_providers_limit_exceeded" {
		t.Errorf("esperaba código custom_providers_limit_exceeded, got %v", errLimit)
	}

	// 5. Validación: ID con prefijo incorrecto o reservado
	invalidIDs := []string{"paquito", "gemini", "custom_", "custom_!@#"}
	for _, badID := range invalidIDs {
		_, errBad := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
			Provider: domain.AIProviderGemini,
			CustomProviders: []domain.AICustomProviderDTO{
				{
					ID:      badID,
					Label:   "Bad",
					BaseURL: "https://api.test.com/v1",
				},
			},
		})
		if errBad == nil {
			t.Errorf("esperaba rechazo para ID inválido %q", badID)
		}
	}

	// 6. Validación: label vacío o superior a 60 caracteres
	_, errEmptyLabel := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_test",
				Label:   "",
				BaseURL: "https://api.test.com/v1",
			},
		},
	})
	if errEmptyLabel == nil {
		t.Errorf("esperaba rechazo para label vacío")
	}

	longLabel := strings.Repeat("a", 61)
	_, errLongLabel := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_test",
				Label:   longLabel,
				BaseURL: "https://api.test.com/v1",
			},
		},
	})
	if errLongLabel == nil {
		t.Errorf("esperaba rechazo para label > 60 chars")
	}

	// 7. Validación: icon_url inválido
	_, errBadIcon := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_test",
				Label:   "Test",
				IconURL: "ftp://invalid-url.com",
				BaseURL: "https://api.test.com/v1",
			},
		},
	})
	if errBadIcon == nil {
		t.Errorf("esperaba rechazo para icon_url con esquema no http/https")
	}
}

func TestAISettingsCustomProvidersSSRFShield(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	blockedURLs := []string{
		"http://localhost:8080/v1",
		"http://127.0.0.1:11434",
		"http://0.0.0.0:8000",
		"http://::1:8000",
		"http://169.254.169.254/latest/meta-data",
		"http://postgres:5432",
		"http://redis/v1",
	}

	for _, badURL := range blockedURLs {
		_, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
			Provider: domain.AIProviderGemini,
			CustomProviders: []domain.AICustomProviderDTO{
				{
					ID:      "custom_blocked",
					Label:   "Blocked Provider",
					BaseURL: badURL,
				},
			},
		})
		if err == nil {
			t.Errorf("SSRF shield debía bloquear %q en custom provider", badURL)
			continue
		}
		var vErr *AISettingsValidationError
		if !errors.As(err, &vErr) {
			t.Errorf("esperaba AISettingsValidationError para %q, got %v", badURL, err)
		}
	}

	allowedURLs := []string{
		"http://192.168.1.100:11434/v1",
		"https://api.groq.com/openai/v1",
		"https://llm.universidad.edu.bo/v1",
	}

	for _, goodURL := range allowedURLs {
		resp, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
			Provider: domain.AIProviderGemini,
			CustomProviders: []domain.AICustomProviderDTO{
				{
					ID:      "custom_allowed",
					Label:   "Allowed Provider",
					BaseURL: goodURL,
				},
			},
		})
		if err != nil {
			t.Errorf("SSRF shield no debía bloquear %q: %v", goodURL, err)
			continue
		}
		if len(resp.CustomProviders) != 1 || resp.CustomProviders[0].BaseURL != goodURL {
			t.Errorf("base_url no preservada para %q: got %+v", goodURL, resp.CustomProviders)
		}
	}
}

func TestAISettingsCustomProviderExfiltrationShield(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// 1. Guardar proveedor custom con API Key
	originalKey := "sk-secret-paquito-key-12345"
	resp1, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito",
				Label:   "Paquito AI",
				BaseURL: "https://api.paquito.ai/v1",
				APIKey:  originalKey,
			},
		},
	})
	if err != nil {
		t.Fatalf("setup falló: %v", err)
	}
	if !resp1.CustomProviders[0].HasAPIKey {
		t.Fatalf("esperaba has_api_key=true")
	}

	// 2. Modificar host de base_url sin enviar nueva API Key -> DEBE INVALIDAR la clave previa
	resp2, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito",
				Label:   "Paquito AI",
				BaseURL: "https://evil-server.com/v1", // host cambió
				APIKey:  "",                           // sin nueva clave
			},
		},
	})
	if err != nil {
		t.Fatalf("update con cambio de host falló: %v", err)
	}
	if resp2.CustomProviders[0].HasAPIKey || resp2.CustomProviders[0].APIKeyMasked != "" {
		t.Errorf("ESCUDO DE EXFILTRACIÓN FALLÓ: la clave previa no fue invalidada tras cambiar de host")
	}

	// 3. Modificar host de base_url enviando una nueva API Key -> DEBE ACEPTARLA
	newKey := "sk-new-legit-key-99999"
	resp3, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito",
				Label:   "Paquito AI",
				BaseURL: "https://new-host.paquito.ai/v1",
				APIKey:  newKey,
			},
		},
	})
	if err != nil {
		t.Fatalf("update con nuevo host y nueva key falló: %v", err)
	}
	if !resp3.CustomProviders[0].HasAPIKey || resp3.CustomProviders[0].APIKeyMasked != "sk-n...9999" {
		t.Errorf("nueva clave debía ser cifrada y guardada: %+v", resp3.CustomProviders[0])
	}

	// 4. Cambiar solo el path (mismo host) sin enviar API Key -> DEBE CONSERVAR la clave
	resp4, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderGemini,
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito",
				Label:   "Paquito AI",
				BaseURL: "https://new-host.paquito.ai/v2", // mismo host, path v2
				APIKey:  "",                               // sin clave
			},
		},
	})
	if err != nil {
		t.Fatalf("update con mismo host falló: %v", err)
	}
	if !resp4.CustomProviders[0].HasAPIKey || resp4.CustomProviders[0].APIKeyMasked != "sk-n...9999" {
		t.Errorf("clave previa debía ser conservada cuando el host no cambió: %+v", resp4.CustomProviders[0])
	}
}

func TestAISettingsMultiProviderCredentialPreservation(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// 1. Configurar OpenAI con su propia API Key y modelo
	_, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderOpenAI,
		Model:    "gpt-4o-mini",
		APIKey:   "sk-openai-key-00000001",
	})
	if err != nil {
		t.Fatalf("error configurando OpenAI: %v", err)
	}

	// 2. Cambiar de proveedor a Gemini y configurar su propia API Key y modelo
	_, err = svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderGemini,
		Model:    "gemini-2.5-flash",
		APIKey:   "AIzaSyGeminiKey00000002",
	})
	if err != nil {
		t.Fatalf("error configurando Gemini: %v", err)
	}

	// 3. Crear y seleccionar un proveedor personalizado custom_together
	respCustom, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: "custom_together",
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_together",
				Label:   "Together AI",
				BaseURL: "https://api.together.xyz/v1",
				Model:   "meta-llama/Llama-3-70b-chat-hf",
				APIKey:  "tog-secret-key-00000003",
			},
		},
	})
	if err != nil {
		t.Fatalf("error configurando custom provider: %v", err)
	}

	// Verificar que provider_profiles expone ambos proveedores base con sus credenciales intactas
	openaiProf, ok1 := respCustom.ProviderProfiles[domain.AIProviderOpenAI]
	if !ok1 || !openaiProf.HasAPIKey || openaiProf.Model != "gpt-4o-mini" {
		t.Errorf("perfil de OpenAI destruido o mezclado: %+v", openaiProf)
	}
	geminiProf, ok2 := respCustom.ProviderProfiles[domain.AIProviderGemini]
	if !ok2 || !geminiProf.HasAPIKey || geminiProf.Model != "gemini-2.5-flash" {
		t.Errorf("perfil de Gemini destruido o mezclado: %+v", geminiProf)
	}
	if len(respCustom.CustomProviders) != 1 || !respCustom.CustomProviders[0].HasAPIKey {
		t.Errorf("custom provider no retuvo sus credenciales: %+v", respCustom.CustomProviders)
	}

	// 4. Volver a seleccionar OpenAI sin enviar APIKey ni Model:
	// Debe restaurar sus credenciales previas del perfil
	respBack, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  false,
		Provider: domain.AIProviderOpenAI,
		Model:    "",
		APIKey:   "",
	})
	if err != nil {
		t.Fatalf("error volviendo a OpenAI: %v", err)
	}
	if respBack.Provider != domain.AIProviderOpenAI {
		t.Errorf("esperaba provider=openai, got %q", respBack.Provider)
	}
	if !respBack.HasAPIKey || respBack.APIKeyMasked != "sk-o...0001" {
		t.Errorf("OpenAI no restauró su API Key guardada: has=%v masked=%q", respBack.HasAPIKey, respBack.APIKeyMasked)
	}
	if respBack.Model != "gpt-4o-mini" {
		t.Errorf("OpenAI no restauró su Model guardado: got %q", respBack.Model)
	}
}

func TestAISettingsNoFakeDefaultModelAndModelRequiredForEnable(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// DefaultAISettings tiene model=""
	def := domain.DefaultAISettings()
	if def.Model != "" {
		t.Errorf("DefaultAISettings().Model debe iniciar en '', got %q", def.Model)
	}

	// Get() en tenant sin configurar tiene model=""
	got, err := svc.Get(ctx, "t-1")
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	if got.Model != "" {
		t.Errorf("Get() sin configurar debe tener model='', got %q", got.Model)
	}

	// Intentar habilitar enabled=true sin modelo -> BLOQUEADO con model_required
	_, errEmptyModel := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  true,
		Provider: domain.AIProviderOllamaLAN,
		BaseURL:  "http://10.20.1.50:11434/v1",
		Model:    "",
	})
	if errEmptyModel == nil {
		t.Fatalf("esperaba error al habilitar sin modelo")
	}
	var vErr *AISettingsValidationError
	if !errors.As(errEmptyModel, &vErr) || vErr.Code != "model_required" {
		t.Errorf("esperaba código model_required, got %v", errEmptyModel)
	}

	// Con modelo configurado, enabled=true tiene éxito
	resp, errWithModel := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:  true,
		Provider: domain.AIProviderOllamaLAN,
		BaseURL:  "http://10.20.1.50:11434/v1",
		Model:    "qwen2.5-coder:7b",
	})
	if errWithModel != nil {
		t.Fatalf("falló activación con modelo válido: %v", errWithModel)
	}
	if !resp.Enabled || resp.Model != "qwen2.5-coder:7b" {
		t.Errorf("estado inesperado: enabled=%v model=%q", resp.Enabled, resp.Model)
	}
}

func TestAIStatusDTOProviderLabel(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// Proveedor base Gemini
	s1, _ := svc.Get(ctx, "t-1")
	status1 := domain.NewAIStatusDTO(*s1)
	if status1.ProviderLabel != "Google Gemini" {
		t.Errorf("esperaba ProviderLabel='Google Gemini', got %q", status1.ProviderLabel)
	}

	// Proveedor base OpenAI
	_, _ = svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider: domain.AIProviderOpenAI,
		Model:    "gpt-4o",
	})
	s2, _ := svc.Get(ctx, "t-1")
	status2 := domain.NewAIStatusDTO(*s2)
	if status2.ProviderLabel != "OpenAI" {
		t.Errorf("esperaba ProviderLabel='OpenAI', got %q", status2.ProviderLabel)
	}

	// Proveedor custom Paquito AI
	_, _ = svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Provider: "custom_paquito1",
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_paquito1",
				Label:   "Paquito AI",
				BaseURL: "https://api.paquito.ai/v1",
				Model:   "paquito-coder-7b",
			},
		},
	})
	s3, _ := svc.Get(ctx, "t-1")
	status3 := domain.NewAIStatusDTO(*s3)
	if status3.Provider != "custom_paquito1" || status3.ProviderLabel != "Paquito AI" {
		t.Errorf("esperaba Provider='custom_paquito1' y ProviderLabel='Paquito AI', got prov=%q label=%q", status3.Provider, status3.ProviderLabel)
	}
}

// TestAISettingsLatencyMeasuredAfterReadAll asegura que latency_ms cubra la
// descarga completa del stream JSON: los headers llegan al instante y el cuerpo
// recién 200 ms después.
func TestAISettingsLatencyMeasuredAfterReadAll(t *testing.T) {
	const bodyDelay = 200 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush() // los headers salen ya, el cuerpo después
		}
		time.Sleep(bodyDelay)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	_, latencyMS, err := svc.CompleteJSON(ctx, "t-1", "u-1", "suggest-component", "sys", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latencyMS < int64(bodyDelay/time.Millisecond)-20 {
		t.Errorf("latency_ms=%d mide antes de descargar el cuerpo: esperaba al menos %d ms", latencyMS, int64(bodyDelay/time.Millisecond)-20)
	}
}

// TestAISettingsTimeoutWrapsErrAITimeout valida el error tipado de plazo
// agotado (el handler HTTP lo traduce a 504 ai_timeout).
func TestAISettingsTimeoutWrapsErrAITimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(500 * time.Millisecond):
		}
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	reqCtx, cancel := context.WithTimeout(ctx, 80*time.Millisecond)
	defer cancel()

	_, _, err := svc.CompleteJSON(reqCtx, "t-1", "u-1", "suggest-component", "sys", "user")
	if err == nil {
		t.Fatalf("esperaba un error de plazo agotado")
	}
	if !errors.Is(err, domain.ErrAITimeout) {
		t.Fatalf("esperaba domain.ErrAITimeout envuelto, got %v", err)
	}
}

// TestAISettingsAdaptiveGovernanceByProfileAndAction cubre la gobernanza
// autoadaptativa: perfil de motor, acción, piso del timeout viejo de 30 s y
// el tope mayor fijado en Modo Experto.
func TestAISettingsAdaptiveGovernanceByProfileAndAction(t *testing.T) {
	profiles := []struct {
		provider string
		model    string
		profile  string
		label    string
		auto     map[string]int
	}{
		{
			provider: domain.AIProviderOpenAI,
			model:    "gpt-4o-mini",
			profile:  domain.AIEngineProfileFastCloud,
			label:    "60s – 120s según complejidad de la tarea",
			auto: map[string]int{
				"suggest-component": 60,
				"suggest-template":  90,
				"repair-template":   90,
				"generate-exercise": 120,
				"test-connection":   30,
			},
		},
		{
			provider: domain.AIProviderOpenRouter,
			model:    "deepseek-r1:free",
			profile:  domain.AIEngineProfileQueuedOrReasoning,
			label:    "90s – 180s según complejidad de la tarea",
			auto: map[string]int{
				"suggest-component": 90,
				"suggest-template":  120,
				"repair-template":   120,
				"generate-exercise": 180,
				"test-connection":   60,
			},
		},
		{
			provider: domain.AIProviderOllamaLAN,
			model:    "qwen2.5-coder:7b",
			profile:  domain.AIEngineProfileLocalGPU,
			label:    "90s – 180s según complejidad de la tarea",
			auto: map[string]int{
				"suggest-component": 90,
				"suggest-template":  120,
				"repair-template":   120,
				"generate-exercise": 180,
				"test-connection":   60,
			},
		},
	}

	for _, tc := range profiles {
		if got := domain.ClassifyAIEngine(tc.provider, tc.model); got != tc.profile {
			t.Errorf("ClassifyAIEngine(%q, %q) = %q, esperaba %q", tc.provider, tc.model, got, tc.profile)
		}
		if got := domain.AIEngineAutoTimeoutRangeLabel(tc.profile); got != tc.label {
			t.Errorf("rango automático de %q = %q, esperaba %q", tc.profile, got, tc.label)
		}

		// Un tope viejo de 30 s guardado en la base jamás recorta la operación.
		stored30 := &domain.AISettingsStored{Provider: tc.provider, Model: tc.model, TimeoutSeconds: 30}
		for action, want := range tc.auto {
			if got := domain.AIEngineAutoTimeoutSeconds(tc.profile, action); got != want {
				t.Errorf("AIEngineAutoTimeoutSeconds(%q, %q) = %d, esperaba %d", tc.profile, action, got, want)
			}
			got := int(effectiveAITimeout(stored30, tc.profile, action) / time.Second)
			if action == "test-connection" {
				if got != want {
					t.Errorf("test-connection en %q = %ds, esperaba %ds", tc.profile, got, want)
				}
				continue
			}
			if got != want {
				t.Errorf("timeout de %q con tope viejo de 30s en %q = %ds, esperaba %ds", tc.profile, action, got, want)
			}
		}

		// Modo Experto: un tope mayor fijado por el administrador se respeta.
		expert := &domain.AISettingsStored{Provider: tc.provider, Model: tc.model, TimeoutSeconds: 240}
		for _, action := range []string{"suggest-component", "suggest-template", "repair-template", "generate-exercise"} {
			if got := int(effectiveAITimeout(expert, tc.profile, action) / time.Second); got != 240 {
				t.Errorf("Modo Experto 240s ignorado en %q/%q: got %ds", tc.profile, action, got)
			}
		}
	}

	// Clasificación complementaria de modelos gratuitos y de razonamiento.
	classified := map[string]string{
		"meta-llama/llama-3.3-70b-instruct:free": domain.AIEngineProfileQueuedOrReasoning,
		"qwen3-30b-a3b-thinking":                 domain.AIEngineProfileQueuedOrReasoning,
		"nvidia/llama-3.1-nemotron-70b":          domain.AIEngineProfileQueuedOrReasoning,
		"deepseek-chat":                          domain.AIEngineProfileFastCloud,
		"gemini-2.5-flash":                       domain.AIEngineProfileFastCloud,
	}
	for model, want := range classified {
		if got := domain.ClassifyAIEngine(domain.AIProviderOpenRouter, model); got != want {
			t.Errorf("ClassifyAIEngine(openrouter, %q) = %q, esperaba %q", model, got, want)
		}
	}
	if got := domain.ClassifyAIEngine("custom_empresa", "llama-3.3-70b"); got != domain.AIEngineProfileLocalGPU {
		t.Errorf("proveedor custom debía clasificarse como local_gpu, got %q", got)
	}

	// Los DTO públicos exponen la política resuelta por el servidor.
	settings := domain.DefaultAISettings()
	settings.Provider = domain.AIProviderOpenAI
	settings.Model = "gpt-4o-mini"
	settingsResp := domain.NewAISettingsResponseDTO(settings)
	if settingsResp.EngineProfile != domain.AIEngineProfileFastCloud || settingsResp.AutoTimeoutRangeLabel == "" {
		t.Errorf("AISettingsResponseDTO sin política autoadaptativa: %+v", settingsResp)
	}
	statusResp := domain.NewAIStatusDTO(settings)
	if statusResp.EngineProfile != domain.AIEngineProfileFastCloud || statusResp.AutoTimeoutRangeLabel == "" {
		t.Errorf("AIStatusDTO sin política autoadaptativa: %+v", statusResp)
	}
}

// TestAISettingsLocalGPUQuotaFloor verifica el piso de cuota anti-abuso del
// servidor GPU local (60/h) y que la nube conserve su límite configurado.
func TestAISettingsLocalGPUQuotaFloor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOllamaLAN,
		Model:              "qwen2.5-coder:7b",
		BaseURL:            "http://10.20.1.50:11434/v1",
		HourlyQuotaPerUser: 5,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA local: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	for i := 0; i < 7; i++ {
		if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-1", "suggest-component", "sys", "user"); err != nil {
			t.Fatalf("invocación %d debía permitirse con el piso local de 60/h: %v", i+1, err)
		}
	}

	// Mismo límite de 5 en nube: la sexta invocación queda bloqueada.
	_, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 5,
		TimeoutSeconds:     60,
	})
	if err != nil {
		t.Fatalf("no se pudo pasar a nube: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-2", "suggest-component", "sys", "user"); err != nil {
			t.Fatalf("invocación %d debía permitirse: %v", i+1, err)
		}
	}
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-2", "suggest-component", "sys", "user"); !errors.Is(err, domain.ErrAIQuotaExceeded) {
		t.Fatalf("esperaba domain.ErrAIQuotaExceeded en la sexta invocación, got %v", err)
	}
}

// TestAISettingsUserOverrideCascadeBYOK cubre la cascada personal: la API Key
// del docente prevalece, opera con la institucional deshabilitada y no consume
// la cuota horaria compartida.
func TestAISettingsUserOverrideCascadeBYOK(t *testing.T) {
	var gotAuth []string
	var gotModel []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		gotModel = append(gotModel, body.Model)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// Base institucional: Gemini con cuota horaria de 1 y tope viejo de 30 s.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		APIKey:             "AIzaInstitucional0000001",
		HourlyQuotaPerUser: 1,
		TimeoutSeconds:     30,
	}); err != nil {
		t.Fatalf("no se pudo configurar la base institucional: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	override, err := svc.UpdateUserOverride(ctx, "t-1", "u-teacher", domain.UpdateAIUserOverrideDTO{
		Enabled:  true,
		Provider: domain.AIProviderOpenAI,
		Model:    "gpt-4o-mini",
		APIKey:   "sk-personal-00000001",
	})
	if err != nil {
		t.Fatalf("no se pudo guardar la API Key personal: %v", err)
	}
	if !override.HasAPIKey || override.APIKeyMasked != "sk-p...0001" {
		t.Errorf("cascada personal sin clave cifrada: %+v", override)
	}
	if !override.InstitutionalEnabled || override.InstitutionalProviderLabel != "Google Gemini" || override.InstitutionalModel != "gemini-2.5-flash" {
		t.Errorf("la respuesta no conserva la base institucional: %+v", override)
	}

	// GET devuelve la clave personal sin exponer el ciphertext.
	fetched, err := svc.GetUserOverride(ctx, "t-1", "u-teacher")
	if err != nil {
		t.Fatalf("GetUserOverride falló: %v", err)
	}
	rawDTO, err := json.Marshal(fetched)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}
	if strings.Contains(string(rawDTO), "encrypted_api_key") || strings.Contains(string(rawDTO), "sk-personal-00000001") {
		t.Fatalf("la respuesta personal filtra la API Key: %s", string(rawDTO))
	}

	// El docente opera con su cuenta: modelo y Authorization propios.
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-teacher", "generate-exercise", "sys", "user"); err != nil {
		t.Fatalf("la inferencia con clave personal falló: %v", err)
	}
	if gotAuth[len(gotAuth)-1] != "Bearer sk-personal-00000001" || gotModel[len(gotModel)-1] != "gpt-4o-mini" {
		t.Errorf("no se usó la credencial personal: auth=%q model=%q", gotAuth[len(gotAuth)-1], gotModel[len(gotModel)-1])
	}

	// Segunda invocación: la cuota institucional de 1 no se descuenta aquí.
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-teacher", "generate-exercise", "sys", "user"); err != nil {
		t.Fatalf("la cuota institucional no debe aplicar a la clave personal: %v", err)
	}

	// Otro usuario sigue con la base institucional y su cuota de 1.
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-student", "generate-exercise", "sys", "user"); err != nil {
		t.Fatalf("la base institucional debía seguir operativa: %v", err)
	}
	if gotAuth[len(gotAuth)-1] != "Bearer AIzaInstitucional0000001" || gotModel[len(gotModel)-1] != "gemini-2.5-flash" {
		t.Errorf("no se usó la base institucional: auth=%q model=%q", gotAuth[len(gotAuth)-1], gotModel[len(gotModel)-1])
	}
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-student", "generate-exercise", "sys", "user"); !errors.Is(err, domain.ErrAIQuotaExceeded) {
		t.Fatalf("esperaba domain.ErrAIQuotaExceeded para el usuario sin override, got %v", err)
	}

	// Estado efectivo por usuario.
	teacherStatus, err := svc.GetEffectiveStatus(ctx, "t-1", "u-teacher")
	if err != nil {
		t.Fatalf("GetEffectiveStatus falló: %v", err)
	}
	if !teacherStatus.UsingPersonalKey || !teacherStatus.Enabled || !teacherStatus.InstitutionalEnabled {
		t.Errorf("estado personal inesperado: %+v", teacherStatus)
	}
	if teacherStatus.Provider != domain.AIProviderOpenAI || teacherStatus.EngineProfile != domain.AIEngineProfileFastCloud {
		t.Errorf("motor personal inesperado: %+v", teacherStatus)
	}
	studentStatus, err := svc.GetEffectiveStatus(ctx, "t-1", "u-student")
	if err != nil {
		t.Fatalf("GetEffectiveStatus falló: %v", err)
	}
	if studentStatus.UsingPersonalKey || studentStatus.Provider != domain.AIProviderGemini {
		t.Errorf("estado institucional inesperado: %+v", studentStatus)
	}

	// La actualización institucional del administrador preserva los overrides.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-pro",
		HourlyQuotaPerUser: 1,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("la actualización institucional falló: %v", err)
	}
	afterAdmin, err := svc.Get(ctx, "t-1")
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	kept, ok := afterAdmin.UserOverrides["u-teacher"]
	if !ok || !kept.Enabled || kept.APIKeyMasked != "sk-p...0001" {
		t.Fatalf("Update institucional pisó la cascada BYOK: %+v", afterAdmin.UserOverrides)
	}

	// Con la IA institucional deshabilitada, el docente con clave personal sigue operando.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            false,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-pro",
		HourlyQuotaPerUser: 1,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("la desactivación institucional falló: %v", err)
	}
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-teacher", "generate-exercise", "sys", "user"); err != nil {
		t.Fatalf("la clave personal debe operar con la IA institucional deshabilitada: %v", err)
	}
	if _, _, err := svc.CompleteJSON(ctx, "t-1", "u-student", "generate-exercise", "sys", "user"); !errors.Is(err, domain.ErrAIDisabled) {
		t.Fatalf("esperaba domain.ErrAIDisabled para el usuario sin override, got %v", err)
	}
}

// TestAIUserListModelsRequiresPersonalCredential es la regresión del bug de
// cascada: el catálogo BYOK del docente no debe caer en la credencial
// institucional. Sin clave personal, listar modelos debe rechazarse.
func TestAIUserListModelsRequiresPersonalCredential(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	// Base institucional con clave guardada: si la cascada filtrara esa
	// credencial, ListUserModels la usaría y traería el catálogo del motor
	// institucional.
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderGemini,
		Model:              "gemini-2.5-flash",
		APIKey:             "AIzaInstitucional0000002",
		HourlyQuotaPerUser: 5,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo configurar la base institucional: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": [{"id": "gemini-2.5-pro"}]}`))
	}))
	defer server.Close()
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	// Sin proveedor ni clave personal: rechazo, no lista el catálogo institucional.
	_, err := svc.ListUserModels(ctx, "t-1", "u-teacher", domain.ListAIModelsRequestDTO{})
	if err == nil {
		t.Fatalf("esperaba rechazo sin credencial personal")
	}
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "provider_required" {
		t.Errorf("esperaba provider_required, got %v", err)
	}

	// Proveedor cloud sin clave personal: api_key_required.
	_, err = svc.ListUserModels(ctx, "t-1", "u-teacher", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderGemini,
	})
	if err == nil {
		t.Fatalf("esperaba api_key_required para cloud sin clave personal")
	}
	if !errors.As(err, &vErr) || vErr.Code != "api_key_required" {
		t.Errorf("esperaba api_key_required, got %v", err)
	}

	// LAN sin URL base: base_url_required.
	_, err = svc.ListUserModels(ctx, "t-1", "u-teacher", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderOllamaLAN,
	})
	if err == nil {
		t.Fatalf("esperaba base_url_required para LAN sin URL")
	}
	if !errors.As(err, &vErr) || vErr.Code != "base_url_required" {
		t.Errorf("esperaba base_url_required, got %v", err)
	}

	// Con clave personal sí lista y usa la credencial del docente.
	resp, err := svc.ListUserModels(ctx, "t-1", "u-teacher", domain.ListAIModelsRequestDTO{
		Provider: domain.AIProviderGemini,
		APIKey:   "AIzaPersonal0000003",
	})
	if err != nil {
		t.Fatalf("con clave personal debía listar: %v", err)
	}
	if resp.Count != 1 || resp.Models[0].ID != "gemini-2.5-pro" {
		t.Errorf("modelo inesperado: %+v", resp)
	}
}

// TestAISettingsUserOverrideValidation aplica al override personal las mismas
// garantías que la configuración institucional.
func TestAISettingsUserOverrideValidation(t *testing.T) {
	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()

	cases := []struct {
		name string
		dto  domain.UpdateAIUserOverrideDTO
		code string
	}{
		{
			name: "proveedor fuera de catálogo",
			dto:  domain.UpdateAIUserOverrideDTO{Enabled: true, Provider: "azure", Model: "gpt-4o"},
			code: "provider_invalid",
		},
		{
			name: "cloud sin api key",
			dto:  domain.UpdateAIUserOverrideDTO{Enabled: true, Provider: domain.AIProviderOpenAI, Model: "gpt-4o-mini"},
			code: "api_key_required",
		},
		{
			name: "sin modelo al activar",
			dto:  domain.UpdateAIUserOverrideDTO{Enabled: true, Provider: domain.AIProviderOllamaLAN, BaseURL: "http://10.20.1.50:11434/v1"},
			code: "model_required",
		},
		{
			name: "escudo anti-SSRF",
			dto: domain.UpdateAIUserOverrideDTO{
				Enabled:  false,
				Provider: domain.AIProviderOllamaLAN,
				Model:    "qwen2.5-coder:7b",
				BaseURL:  "http://localhost:11434/v1",
			},
			code: "base_url_blocked",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpdateUserOverride(ctx, "t-1", "u-teacher", tc.dto)
			if err == nil {
				t.Fatalf("esperaba rechazo")
			}
			var vErr *AISettingsValidationError
			if !errors.As(err, &vErr) || vErr.Code != tc.code {
				t.Errorf("esperaba código %q, got %v", tc.code, err)
			}
		})
	}

	// Un override sin clave en la nube no califica: cae en la base institucional.
	stored, err := svc.Get(ctx, "t-1")
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	if _, ok := resolveUserOverride(stored, "u-teacher"); ok {
		t.Errorf("un override incompleto no debe calificar")
	}
}

// TestAISettingsResponseFormatFallbackOnBadRequest cubre el blindaje de
// transporte: 400 por response_format se reintenta una vez sin él.
func TestAISettingsResponseFormatFallbackOnBadRequest(t *testing.T) {
	var withFormat []bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sentFormat := body.ResponseFormat.Type != ""
		withFormat = append(withFormat, sentFormat)
		if sentFormat {
			time.Sleep(80 * time.Millisecond)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"response_format json_object no soportado"}`))
			return
		}
		time.Sleep(80 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	repo := newTestAISettingsRepo(`{}`)
	svc := NewAISettingsService(repo)
	ctx := context.Background()
	if _, err := svc.Update(ctx, "t-1", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenRouter,
		Model:              "deepseek-r1:free",
		APIKey:             "sk-or-test",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}
	svc.WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

	content, latencyMS, err := svc.CompleteJSON(ctx, "t-1", "u-1", "generate-exercise", "sys", "user")
	if err != nil {
		t.Fatalf("el reintento sin response_format debía resolver el 400: %v", err)
	}
	if content != `{"ok":true}` {
		t.Errorf("contenido inesperado: %q", content)
	}
	if len(withFormat) != 2 || !withFormat[0] || withFormat[1] {
		t.Errorf("esperaba 2 intentos (con y sin response_format), got %v", withFormat)
	}
	if latencyMS < 150 {
		t.Errorf("latency_ms=%d debe acumular los dos intentos (80 ms + 80 ms)", latencyMS)
	}
}

// TestAISettingsTestConnectionToleratesReasoningWrap cubre modelos de
// razonamiento que envuelven el JSON en prosa o en fences Markdown.
func TestAISettingsTestConnectionToleratesReasoningWrap(t *testing.T) {
	contents := []string{
		"Pensando en cómo verificar...\n```json\n{\"ok\":true}\n``` ",
		"Verificando el endpoint... {\"ok\":true}",
		"{\"ok\":true}",
	}
	for _, content := range contents {
		server := newCopilotLLMServer(t, content)
		svc := NewAISettingsService(newTestAISettingsRepo(`{}`)).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})

		resp, err := svc.TestConnection(context.Background(), "t-1", domain.TestAIConnectionRequestDTO{
			Provider: domain.AIProviderOpenRouter,
			Model:    "deepseek-r1:free",
			APIKey:   "sk-or-test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.OK {
			t.Errorf("esperaba ok=true con contenido %q, got message=%q", content, resp.Message)
		}
	}

	// Sin objeto JSON alguno la prueba sigue fallando con ok=false.
	server := newCopilotLLMServer(t, "no puedo responder ahora")
	svc := NewAISettingsService(newTestAISettingsRepo(`{}`)).WithTransport(&rewriteAITestTransport{base: mustParseURL(t, server.URL)})
	resp, err := svc.TestConnection(context.Background(), "t-1", domain.TestAIConnectionRequestDTO{
		Provider: domain.AIProviderOpenRouter,
		Model:    "deepseek-r1:free",
		APIKey:   "sk-or-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OK {
		t.Errorf("esperaba ok=false sin objeto JSON")
	}
}
