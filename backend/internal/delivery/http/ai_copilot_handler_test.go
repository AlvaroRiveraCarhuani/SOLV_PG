package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// aiTestRewriteTransport redirige cualquier petición (incluida la URL oficial
// cloud forzada por el escudo anti-SSRF) hacia el servidor httptest local.
type aiTestRewriteTransport struct {
	base *url.URL
}

func (t *aiTestRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.base.Scheme
	clone.URL.Host = t.base.Host
	clone.Host = t.base.Host
	return http.DefaultTransport.RoundTrip(clone)
}

// TestAICopilotHandlerRepairTemplateTimeoutIs504 ejercita el camino completo:
// el motor agota el plazo del contexto, el servicio lo envuelve en
// domain.ErrAITimeout y el handler responde 504 con código ai_timeout.
func TestAICopilotHandlerRepairTemplateTimeoutIs504(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(300 * time.Millisecond):
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	repo := &mockAISettingsTenantRepo{configs: map[string][]byte{"t-test": []byte(`{}`)}}
	settingsSvc := services.NewAISettingsService(repo).WithTransport(&aiTestRewriteTransport{base: mustParseAITestURL(t, server.URL)})

	ctx := context.Background()
	if _, err := settingsSvc.Update(ctx, "t-test", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}

	handler := NewAICopilotHandler(services.NewAICopilotService(settingsSvc, nil))

	body := `{"name":"Laboratorio","docker_image":"python:3.12-slim","error_message":"el contenedor superó el tiempo de arranque"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/repair-template", strings.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "u-admin")

	reqCtx, cancel := context.WithTimeout(context.WithValue(req.Context(), domain.TenantIDKey, "t-test"), 60*time.Millisecond)
	defer cancel()
	req = req.WithContext(reqCtx)

	rec := httptest.NewRecorder()
	handler.RepairTemplate(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("esperaba HTTP 504, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope GlobalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error deserializando la respuesta: %v", err)
	}
	if envelope.Error != "ai_timeout" {
		t.Errorf("esperaba el código ai_timeout, got %q", envelope.Error)
	}
	if !strings.Contains(envelope.Message, "tiempo límite") {
		t.Errorf("mensaje poco claro para el timeout: %q", envelope.Message)
	}
}

// TestAICopilotHandlerRepairTemplateSuccess ejercita el endpoint completo:
// el motor devuelve el manifiesto reparado y el handler responde 200.
func TestAICopilotHandlerRepairTemplateSuccess(t *testing.T) {
	repaired := `{"docker_image":"solv-local/stack:estadistica","dockerfile_content":"FROM python:3.12-slim-bookworm\nRUN apt-get update && apt-get install -y --no-install-recommends gcc","setup_script":"set -euo pipefail","tools_declared":["python3","gcc"],"entrypoint":"python3 /opt/judge/run.py","base_ram_mb":2048,"rationale":"Faltaba gcc en la imagen; se agrega la capa de build."}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + strconv.Quote(repaired) + `}}]}`))
	}))
	defer server.Close()

	repo := &mockAISettingsTenantRepo{configs: map[string][]byte{"t-test": []byte(`{}`)}}
	settingsSvc := services.NewAISettingsService(repo).WithTransport(&aiTestRewriteTransport{base: mustParseAITestURL(t, server.URL)})
	if _, err := settingsSvc.Update(context.Background(), "t-test", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-123",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     60,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}

	handler := NewAICopilotHandler(services.NewAICopilotService(settingsSvc, nil))
	body := `{"name":"Estadística","docker_image":"python:3.12-slim","dockerfile_content":"FROM python:3.12-slim","tools_declared":["python3"],"base_ram_mb":1024,"error_message":"gcc: not found"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/repair-template", strings.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Id", "u-admin")
	req = req.WithContext(context.WithValue(req.Context(), domain.TenantIDKey, "t-test"))
	rec := httptest.NewRecorder()

	handler.RepairTemplate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data domain.AIRepairTemplateResponseDTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error deserializando la respuesta: %v", err)
	}
	if envelope.Data.DockerImage != "solv-local/stack:estadistica" {
		t.Errorf("docker_image inesperada: %q", envelope.Data.DockerImage)
	}
	if strings.TrimSpace(envelope.Data.Rationale) == "" {
		t.Errorf("esperaba rationale con la causa raíz")
	}
	if envelope.Data.BaseRamMB != 2048 {
		t.Errorf("base_ram_mb inesperada: %d", envelope.Data.BaseRamMB)
	}
}

// TestAICopilotHandlerRepairTemplateRejectsNonAdmin verifica el rol admin.
func TestAICopilotHandlerRepairTemplateRejectsNonAdmin(t *testing.T) {
	repo := &mockAISettingsTenantRepo{configs: map[string][]byte{"t-test": []byte(`{}`)}}
	handler := NewAICopilotHandler(services.NewAICopilotService(services.NewAISettingsService(repo), nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/repair-template", strings.NewReader(`{}`))
	req.Header.Set("X-User-Role", "teacher")
	rec := httptest.NewRecorder()

	handler.RepairTemplate(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperaba HTTP 403 para el rol teacher, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestAICopilotHandlerErrorMappingKeepsTechnicalReason garantiza que la rama
// 500 por defecto nunca oculte la causa real del fallo.
func TestAICopilotHandlerErrorMappingKeepsTechnicalReason(t *testing.T) {
	rec := httptest.NewRecorder()
	sendAICopilotError(rec, errors.New("dial tcp 10.20.1.50:11434: connect: connection refused"), "Error al reparar la plantilla con el Copiloto IA")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("esperaba HTTP 500, got %d", rec.Code)
	}
	var envelope GlobalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error deserializando la respuesta: %v", err)
	}
	if !strings.Contains(envelope.Message, "Error al reparar la plantilla con el Copiloto IA") {
		t.Errorf("el mensaje debía conservar el contexto funcional: %q", envelope.Message)
	}
	if !strings.Contains(envelope.Message, "connection refused") {
		t.Errorf("el mensaje debía incluir el motivo técnico real: %q", envelope.Message)
	}

	// La IA deshabilitada y la cuota agotada conservan sus códigos propios.
	disabled := httptest.NewRecorder()
	sendAICopilotError(disabled, domain.ErrAIDisabled, "fallback")
	if disabled.Code != http.StatusConflict {
		t.Errorf("esperaba HTTP 409 para ai_disabled, got %d", disabled.Code)
	}
	quota := httptest.NewRecorder()
	sendAICopilotError(quota, domain.ErrAIQuotaExceeded, "fallback")
	if quota.Code != http.StatusTooManyRequests {
		t.Errorf("esperaba HTTP 429 para ai_quota_exceeded, got %d", quota.Code)
	}
}

// TestAISettingsMySettingsRejectsStudent cubre el 403 de la cascada BYOK
// /api/v1/ai/my-settings* para estudiantes y el acceso permitido a docentes.
func TestAISettingsMySettingsRejectsStudent(t *testing.T) {
	repo := &mockAISettingsTenantRepo{configs: map[string][]byte{"t-test": []byte(`{}`)}}
	handler := NewAISettingsHandler(services.NewAISettingsService(repo))

	endpoints := map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/v1/ai/my-settings":                  handler.GetMySettings,
		"PUT /api/v1/ai/my-settings":                  handler.UpdateMySettings,
		"POST /api/v1/ai/my-settings/test-connection": handler.TestMyConnection,
		"POST /api/v1/ai/my-settings/models":          handler.ListMyModels,
	}
	for name, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/my-settings", strings.NewReader(`{}`))
		req.Header.Set("X-User-Role", "student")
		req.Header.Set("X-User-Id", "u-student")
		req = req.WithContext(context.WithValue(req.Context(), domain.TenantIDKey, "t-test"))
		rec := httptest.NewRecorder()

		endpoint(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: esperaba HTTP 403 para student, got %d: %s", name, rec.Code, rec.Body.String())
		}
	}

	// El docente sí consulta su propia configuración.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/my-settings", nil)
	req.Header.Set("X-User-Role", "teacher")
	req.Header.Set("X-User-Id", "u-teacher")
	req = req.WithContext(context.WithValue(req.Context(), domain.TenantIDKey, "t-test"))
	rec := httptest.NewRecorder()
	handler.GetMySettings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba HTTP 200 para teacher, got %d: %s", rec.Code, rec.Body.String())
	}
}

func mustParseAITestURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url inválida %q: %v", raw, err)
	}
	return u
}
