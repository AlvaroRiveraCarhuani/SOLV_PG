package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockAISettingsTenantRepo struct {
	configs map[string][]byte
}

func (m *mockAISettingsTenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return &domain.Tenant{ID: id, Config: cfg}, nil
}

func (m *mockAISettingsTenantRepo) UpdateConfig(ctx context.Context, id string, config []byte) error {
	m.configs[id] = config
	return nil
}

func (m *mockAISettingsTenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return nil, nil
}

func (m *mockAISettingsTenantRepo) GetAll(ctx context.Context) ([]*domain.Tenant, error) {
	return nil, nil
}

func (m *mockAISettingsTenantRepo) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	return nil
}

func (m *mockAISettingsTenantRepo) GetMaintenance(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	return &domain.MaintenanceStatus{}, nil
}

func TestAISettingsHandlerStatusReturnsProviderLabel(t *testing.T) {
	repo := &mockAISettingsTenantRepo{configs: map[string][]byte{"t-test": []byte(`{}`)}}
	svc := services.NewAISettingsService(repo)
	handler := NewAISettingsHandler(svc)

	// 1. GET /api/v1/ai/status con proveedor por defecto Gemini
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/status", nil)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, "t-test")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Data domain.AIStatusDTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error deserializando respuesta: %v", err)
	}
	if envelope.Data.Provider != "gemini" || envelope.Data.ProviderLabel != "Google Gemini" {
		t.Errorf("status response inesperada: %+v", envelope.Data)
	}

	// 2. Configurar proveedor custom y consultar status
	_, err := svc.Update(context.Background(), "t-test", domain.UpdateAISettingsDTO{
		Provider: "custom_mistral",
		CustomProviders: []domain.AICustomProviderDTO{
			{
				ID:      "custom_mistral",
				Label:   "Mistral AI",
				BaseURL: "https://api.mistral.ai/v1",
				Model:   "mistral-small",
			},
		},
	})
	if err != nil {
		t.Fatalf("error configurando custom provider: %v", err)
	}

	rec2 := httptest.NewRecorder()
	handler.Status(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("esperaba HTTP 200, got %d", rec2.Code)
	}
	var envelope2 struct {
		Data domain.AIStatusDTO `json:"data"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &envelope2); err != nil {
		t.Fatalf("error deserializando respuesta 2: %v", err)
	}
	if envelope2.Data.Provider != "custom_mistral" || envelope2.Data.ProviderLabel != "Mistral AI" {
		t.Errorf("status con custom provider inesperado: %+v", envelope2.Data)
	}
}

func TestAISettingsHandlerGetAndUpdateSettingsWithAdminRole(t *testing.T) {
	repo := &mockAISettingsTenantRepo{configs: map[string][]byte{"t-test": []byte(`{}`)}}
	svc := services.NewAISettingsService(repo)
	handler := NewAISettingsHandler(svc)

	// PUT /api/v1/admin/ai/settings
	body := `{
		"enabled": false,
		"provider": "custom_groq",
		"custom_providers": [
			{
				"id": "custom_groq",
				"label": "Groq Cloud",
				"base_url": "https://api.groq.com/openai/v1",
				"model": "llama-3.3-70b-versatile",
				"api_key": "gsk-test-12345678"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/ai/settings", strings.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, "t-test")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba HTTP 200 en UpdateSettings, got %d: %s", rec.Code, rec.Body.String())
	}

	var respEnvelope struct {
		Data domain.AISettingsResponseDTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &respEnvelope); err != nil {
		t.Fatalf("deserialización falló: %v", err)
	}
	if respEnvelope.Data.Provider != "custom_groq" || respEnvelope.Data.ProviderLabel != "Groq Cloud" {
		t.Errorf("datos inesperados: %+v", respEnvelope.Data)
	}
	if len(respEnvelope.Data.CustomProviders) != 1 || !respEnvelope.Data.CustomProviders[0].HasAPIKey {
		t.Errorf("custom provider no retuvo API key: %+v", respEnvelope.Data.CustomProviders)
	}

	// GET /api/v1/admin/ai/settings
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai/settings", nil)
	getReq.Header.Set("X-User-Role", "admin")
	getReq = getReq.WithContext(ctx)
	getRec := httptest.NewRecorder()

	handler.GetSettings(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("esperaba HTTP 200 en GetSettings, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var getEnvelope struct {
		Data domain.AISettingsResponseDTO `json:"data"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getEnvelope); err != nil {
		t.Fatalf("deserialización GetSettings falló: %v", err)
	}
	if getEnvelope.Data.Provider != "custom_groq" || getEnvelope.Data.ProviderLabel != "Groq Cloud" {
		t.Errorf("datos GetSettings inesperados: %+v", getEnvelope.Data)
	}
	if _, ok := getEnvelope.Data.ProviderProfiles[domain.AIProviderGemini]; !ok {
		t.Errorf("provider_profiles debía incluir gemini")
	}
}
