package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"

	coredomain "solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/domain"
)

type identityTenantRepo struct {
	tenants map[string]*coredomain.Tenant
}

func (m *identityTenantRepo) GetByID(_ context.Context, id string) (*coredomain.Tenant, error) {
	tenant, ok := m.tenants[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return tenant, nil
}

func (m *identityTenantRepo) GetBySlug(_ context.Context, slug string) (*coredomain.Tenant, error) {
	for _, tenant := range m.tenants {
		if tenant.Slug == slug {
			return tenant, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *identityTenantRepo) GetAll(context.Context) ([]*coredomain.Tenant, error) {
	return nil, nil
}

func (m *identityTenantRepo) UpdateConfig(context.Context, string, []byte) error { return nil }

func (m *identityTenantRepo) SetMaintenance(context.Context, string, bool, *time.Time, string) error {
	return nil
}

func (m *identityTenantRepo) GetMaintenance(context.Context, string) (*coredomain.MaintenanceStatus, error) {
	return &coredomain.MaintenanceStatus{}, nil
}

type identityUserRepo struct {
	users map[string]domain.UserResponseDTO
}

func (m *identityUserRepo) CreateUser(context.Context, domain.CreateUserDTO) (string, error) {
	return "created", nil
}

func (m *identityUserRepo) GetUserByID(ctx context.Context, id string) (domain.UserResponseDTO, error) {
	user, ok := m.users[id]
	if !ok || coredomain.GetTenantID(ctx) != user.TenantID {
		return domain.UserResponseDTO{}, errors.New("user not found")
	}
	return user, nil
}

func (m *identityUserRepo) GetUserByEmail(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not found")
}

func (m *identityUserRepo) CreateUserFromSSO(context.Context, *domain.User) (string, error) {
	return "created", nil
}

func (m *identityUserRepo) UpdateUserRole(context.Context, string, string) error { return nil }

type identityTemplateRepo struct {
	byTenant map[string][]domain.TemplateResponseDTO
	created  []string
}

func (m *identityTemplateRepo) CreateTemplate(_ context.Context, tenantID string, _ domain.CreateTemplateDTO) (string, error) {
	m.created = append(m.created, tenantID)
	return "template-created", nil
}

func (m *identityTemplateRepo) GetAllTemplates(_ context.Context, tenantID string) ([]domain.TemplateResponseDTO, error) {
	return m.byTenant[tenantID], nil
}

func (m *identityTemplateRepo) GetTemplateByID(context.Context, string) (domain.TemplateResponseDTO, error) {
	return domain.TemplateResponseDTO{}, errors.New("not found")
}

func newIdentitySecurityServer(t *testing.T) (*httptest.Server, *services.AuthService, *identityTenantRepo, *identityUserRepo, *identityTemplateRepo) {
	t.Helper()
	authService := newWebSocketTestAuth(t)
	tenantRepo := &identityTenantRepo{tenants: map[string]*coredomain.Tenant{
		"tenant-a":                 {ID: "tenant-a", Name: "Tenant A", Slug: "tenant-a"},
		"tenant-b":                 {ID: "tenant-b", Name: "Tenant B", Slug: "tenant-b"},
		coredomain.DefaultTenantID: {ID: coredomain.DefaultTenantID, Name: "Default", Slug: "default"},
	}}
	userRepo := &identityUserRepo{users: map[string]domain.UserResponseDTO{
		"user-a": {ID: "user-a", FirstName: "Ada", LastName: "A", Email: "ada@tenant-a.test", Role: "student", TenantID: "tenant-a"},
		"user-b": {ID: "user-b", FirstName: "Bob", LastName: "B", Email: "bob@tenant-b.test", Role: "teacher", TenantID: "tenant-b"},
	}}
	templateRepo := &identityTemplateRepo{byTenant: map[string][]domain.TemplateResponseDTO{
		"tenant-a": {{ID: "template-a", Name: "Template A"}},
		"tenant-b": {{ID: "template-b", Name: "Template B"}},
	}}

	mux := http.NewServeMux()
	SetupRoutes(mux, &Handlers{
		UserHandler:      NewUserHandler(userRepo, validator.New()),
		TemplateHandler:  NewTemplateHandler(templateRepo, validator.New()),
		TeacherHandler:   NewTeacherHandler(nil, nil),
		ConfigHandler:    NewConfigHandler(tenantRepo),
		TenantMiddleware: middleware.WithTenant(tenantRepo, []byte("websocket-gap-jwt-secret")),
	})
	handler := middleware.SanitizeIdentityHeaders(middleware.SessionRevocationMiddleware(authService)(mux))
	return httptest.NewServer(handler), authService, tenantRepo, userRepo, templateRepo
}

func identityRequest(t *testing.T, method, url, token string, headers map[string]string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("no se pudo crear la petición: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req
}

func TestProtectedRouteRejectsIdentityHeadersWithoutSession(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	for name, headers := range map[string]map[string]string{
		"solo usuario": {"X-User-Id": "user-a"},
		"rol admin":    {"X-User-Role": "admin"},
		"tenant ajeno": {
			"X-User-Id":   "user-a",
			"X-User-Role": "admin",
			"X-Tenant-Id": "tenant-b",
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := identityRequest(t, http.MethodGet, server.URL+"/api/v1/users/me", "", headers)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("petición fallida: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("sin sesión y con headers se esperaba 401, se obtuvo %d", resp.StatusCode)
			}
		})
	}
}

func TestValidSessionIgnoresSpoofedIdentityHeaders(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	token := newWebSocketTestToken(t, "user-a", "tenant-a", "student", time.Now().Add(time.Hour))
	req := identityRequest(t, http.MethodGet, server.URL+"/api/v1/users/me", token, map[string]string{
		"X-User-Id":   "user-b",
		"X-User-Role": "admin",
		"X-Tenant-Id": "tenant-b",
	})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200 con la sesión válida, se obtuvo %d", resp.StatusCode)
	}

	var body struct {
		Data domain.UserResponseDTO `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("respuesta inválida: %v", err)
	}
	if body.Data.ID != "user-a" || body.Data.TenantID != "tenant-a" {
		t.Fatalf("los headers sustituyeron la sesión: %+v", body.Data)
	}
}

func TestRevokedSessionIsRejected(t *testing.T) {
	server, authService, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	token := newWebSocketTestToken(t, "user-a", "tenant-a", "student", time.Now().Add(time.Hour))
	if err := authService.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("no se pudo revocar la sesión de prueba: %v", err)
	}

	req := identityRequest(t, http.MethodGet, server.URL+"/api/v1/users/me", token, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("una sesión revocada debe responder 401, se obtuvo %d", resp.StatusCode)
	}
}

func TestRoleAllowlistRejectsEmptyUnknownAndStudent(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	for name, role := range map[string]string{
		"vacío":       "",
		"desconocido": "assistant",
		"student":     "student",
	} {
		t.Run(name, func(t *testing.T) {
			token := newWebSocketTestToken(t, "user-a", "tenant-a", role, time.Now().Add(time.Hour))
			req := identityRequest(t, http.MethodPost, server.URL+"/api/v1/teacher/submissions/sub-1/run-ephemeral", token, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("petición fallida: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("rol %q debe rechazarse con 403, se obtuvo %d", role, resp.StatusCode)
			}
		})
	}
}

func TestTemplatesRequireSessionAndAreFilteredByTenant(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	noSession := identityRequest(t, http.MethodGet, server.URL+"/api/v1/templates", "", map[string]string{
		"X-Tenant-Id": "tenant-b",
		"X-User-Id":   "user-b",
	})
	resp, err := http.DefaultClient.Do(noSession)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("plantillas sin sesión deben responder 401, se obtuvo %d", resp.StatusCode)
	}

	token := newWebSocketTestToken(t, "user-b", "tenant-b", "teacher", time.Now().Add(time.Hour))
	req := identityRequest(t, http.MethodGet, server.URL+"/api/v1/templates", token, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200 para docente autenticado, se obtuvo %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Data []domain.TemplateResponseDTO `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("respuesta inválida: %v", err)
	}
	if len(body.Data.Data) != 1 || body.Data.Data[0].ID != "template-b" {
		t.Fatalf("se filtró una plantilla de otro tenant: %+v", body.Data.Data)
	}
}

func TestStudentCannotCreateTemplate(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	token := newWebSocketTestToken(t, "user-a", "tenant-a", "student", time.Now().Add(time.Hour))
	req := identityRequest(t, http.MethodPost, server.URL+"/api/v1/templates", token, nil)
	req.Body = http.NoBody
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("un estudiante no debe crear plantillas, se obtuvo %d", resp.StatusCode)
	}
}

func TestUserCreationRequiresAdminSession(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	noSession := identityRequest(t, http.MethodPost, server.URL+"/api/v1/users", "", map[string]string{
		"X-User-Id":   "attacker",
		"X-User-Role": "admin",
		"X-Tenant-Id": "tenant-a",
	})
	resp, err := http.DefaultClient.Do(noSession)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("crear usuario sin sesión debe responder 401, se obtuvo %d", resp.StatusCode)
	}

	for name, role := range map[string]string{"vacío": "", "desconocido": "assistant", "student": "student"} {
		t.Run(name, func(t *testing.T) {
			token := newWebSocketTestToken(t, "user-a", "tenant-a", role, time.Now().Add(time.Hour))
			req := identityRequest(t, http.MethodPost, server.URL+"/api/v1/users", token, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("petición fallida: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("rol %q no debe crear usuarios, se obtuvo %d", role, resp.StatusCode)
			}
		})
	}
}

func TestPublicConfigIgnoresTenantHeader(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	req := identityRequest(t, http.MethodGet, server.URL+"/api/v1/config/public", "", map[string]string{
		"X-Tenant-Id": "tenant-b",
	})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("config pública debe responder 200, se obtuvo %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("respuesta inválida: %v", err)
	}
	if body["tenant_id"] != coredomain.DefaultTenantID {
		t.Fatalf("el header eligió otro tenant: %v", body["tenant_id"])
	}
}

func TestPublicConfigSelectsTenantBySlug(t *testing.T) {
	server, _, _, _, _ := newIdentitySecurityServer(t)
	defer server.Close()

	req := identityRequest(t, http.MethodGet, server.URL+"/api/v1/config/public?slug=tenant-b", "", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición fallida: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("config pública por slug debe responder 200, se obtuvo %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("respuesta inválida: %v", err)
	}
	if body["tenant_id"] != "tenant-b" || body["slug"] != "tenant-b" {
		t.Fatalf("el slug debe seleccionar el tenant público: %v", body)
	}
}

func TestPublicConfigOnlyReturnsAllowlistedFields(t *testing.T) {
	tenantRepo := &identityTenantRepo{tenants: map[string]*coredomain.Tenant{
		"tenant-public": {
			ID:   "tenant-public",
			Name: "Institución Pública",
			Slug: "publica",
			Config: []byte(`{
				"institution_name":"Institución Pública",
				"logo_url":"/api/v1/public/branding/logo/publica.png",
				"tenant_primary_color":"#15803D",
				"support_email":"soporte@example.test",
				"font_sans_family":"cat:inter",
				"font_mono_family":"cat:jetbrains-mono",
				"ai_settings":{"encrypted_api_key":"cipher-secret","user_overrides":{"user-1":{"encrypted_api_key":"byok-secret"}}},
				"server_policies":{"ram_limit_mb":512},
				"api_key":"plain-secret"
			}`),
		},
	}}

	h := NewConfigHandler(tenantRepo)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/config/public?slug=publica", nil)
	rec := httptest.NewRecorder()
	h.GetPublicConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta inválida: %v", err)
	}
	for _, field := range []string{"tenant_id", "slug", "institution_name", "logo_url", "tenant_primary_color", "support_email", "font_sans_family", "font_mono_family"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("falta el campo público %q: %v", field, body)
		}
	}
	for _, field := range []string{"ai_settings", "server_policies", "api_key", "user_overrides", "encrypted_api_key"} {
		if _, ok := body[field]; ok {
			t.Fatalf("el campo interno %q no debe salir en config/public: %v", field, body)
		}
	}
	for _, secret := range []string{"cipher-secret", "byok-secret", "plain-secret"} {
		if strings.Contains(rec.Body.String(), secret) {
			t.Fatalf("el secreto %q quedó expuesto en config/public", secret)
		}
	}
}

func TestAdminSessionStillCreatesUsersAndTemplates(t *testing.T) {
	server, _, _, _, templates := newIdentitySecurityServer(t)
	defer server.Close()

	adminToken := newWebSocketTestToken(t, "admin-a", "tenant-a", "admin", time.Now().Add(time.Hour))

	userReq := identityRequest(t, http.MethodPost, server.URL+"/api/v1/users", adminToken, nil)
	userReq.Body = io.NopCloser(strings.NewReader(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@uab.edu.bo"}`))
	userReq.Header.Set("Content-Type", "application/json")
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		t.Fatalf("petición de usuario fallida: %v", err)
	}
	defer func() { _ = userResp.Body.Close() }()
	if userResp.StatusCode != http.StatusCreated {
		t.Fatalf("un admin debe poder crear usuarios, se obtuvo %d: %s", userResp.StatusCode, readBody(t, userResp))
	}

	templateReq := identityRequest(t, http.MethodPost, server.URL+"/api/v1/templates", adminToken, nil)
	templateReq.Body = io.NopCloser(strings.NewReader(`{"name":"Lab","docker_image":"python:3.12-slim","base_ram_mb":512}`))
	templateReq.Header.Set("Content-Type", "application/json")
	templateResp, err := http.DefaultClient.Do(templateReq)
	if err != nil {
		t.Fatalf("petición de plantilla fallida: %v", err)
	}
	defer func() { _ = templateResp.Body.Close() }()
	if templateResp.StatusCode != http.StatusCreated {
		t.Fatalf("un admin debe poder crear plantillas, se obtuvo %d: %s", templateResp.StatusCode, readBody(t, templateResp))
	}
	if len(templates.created) != 1 || templates.created[0] != "tenant-a" {
		t.Fatalf("la plantilla debe crearse en el tenant del claim: %v", templates.created)
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

func TestWithAuthRejectsHeaderIdentityWithoutToken(t *testing.T) {
	handler := WithAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(requestUserID(r)))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-User-Id", "attacker")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("WithAuth sin token debe responder 401, se obtuvo %d", rec.Code)
	}
}

func TestMetricsAccessRequiresLoopbackOrToken(t *testing.T) {
	t.Setenv("METRICS_TOKEN", "")
	handler := metricsAccessMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	remote := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	remote.RemoteAddr = "203.0.113.4:1234"
	remoteRec := httptest.NewRecorder()
	handler.ServeHTTP(remoteRec, remote)
	if remoteRec.Code != http.StatusForbidden {
		t.Fatalf("métricas desde una IP no loopback deben rechazarse, se obtuvo %d", remoteRec.Code)
	}

	loopback := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	loopback.RemoteAddr = "127.0.0.1:1234"
	loopbackRec := httptest.NewRecorder()
	handler.ServeHTTP(loopbackRec, loopback)
	if loopbackRec.Code != http.StatusNoContent {
		t.Fatalf("métricas loopback deben pasar, se obtuvo %d", loopbackRec.Code)
	}

	t.Setenv("METRICS_TOKEN", "metrics-test-token")
	tokenHandler := metricsAccessMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	tokenRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	tokenRequest.RemoteAddr = "203.0.113.4:1234"
	tokenRequest.Header.Set("Authorization", "Bearer metrics-test-token")
	tokenRec := httptest.NewRecorder()
	tokenHandler.ServeHTTP(tokenRec, tokenRequest)
	if tokenRec.Code != http.StatusNoContent {
		t.Fatalf("métricas con token deben pasar, se obtuvo %d", tokenRec.Code)
	}
}

func TestSanitizeIdentityHeadersRemovesClientValues(t *testing.T) {
	handler := middleware.SanitizeIdentityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Join([]string{
			r.Header.Get("X-User-Id"),
			r.Header.Get("X-User-Role"),
			r.Header.Get("X-Tenant-Id"),
		}, ",")))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Id", "attacker")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Tenant-Id", "tenant-b")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Body.String() != ",," {
		t.Fatalf("el saneador dejó cabeceras del cliente: %q", rec.Body.String())
	}
}

func TestIdentityHelpersOnlyReadVerifiedContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-User-Id", "header-user")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Tenant-Id", "header-tenant")
	req.Header.Set("X-User-Email", "header@example.test")

	if got := requestUserID(req); got != "" {
		t.Fatalf("requestUserID no debe leer headers, obtuvo %q", got)
	}
	if got := requestTenantID(req); got != "" {
		t.Fatalf("requestTenantID no debe leer headers, obtuvo %q", got)
	}
	if got := requestRole(req); got != "" {
		t.Fatalf("requestRole no debe leer headers, obtuvo %q", got)
	}
	if got := requestUserEmail(req); got != "" {
		t.Fatalf("requestUserEmail no debe leer headers, obtuvo %q", got)
	}

	req = withTestIdentity(req, "context-user", "context-tenant", "teacher")
	req = withTestEmail(req, "context@example.test")
	if got := requestUserID(req); got != "context-user" {
		t.Fatalf("el contexto verificado debe prevalecer, obtuvo %q", got)
	}
	if got := requestTenantID(req); got != "context-tenant" {
		t.Fatalf("el tenant verificado debe prevalecer, obtuvo %q", got)
	}
	if got := requestRole(req); got != "teacher" {
		t.Fatalf("el rol verificado debe prevalecer, obtuvo %q", got)
	}
	if got := requestUserEmail(req); got != "context@example.test" {
		t.Fatalf("el correo verificado debe prevalecer, obtuvo %q", got)
	}
}

func TestEvaluationWebSocketIgnoresIdentityHeaders(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	hub := NewWebSocketHub()
	go hub.Run()
	handler := NewWebSocketHandler(hub, authService)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	token := newWebSocketTestToken(t, "user-a", "tenant-a", "student", time.Now().Add(time.Hour))
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("X-User-Id", "user-b")
	headers.Set("X-Tenant-Id", "tenant-b")

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/v1/evaluations", headers)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("la sesión válida debe abrir el WebSocket: status=%d err=%v", status, err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var message struct {
		Event string            `json:"event"`
		Data  map[string]string `json:"data"`
	}
	if err := conn.ReadJSON(&message); err != nil {
		t.Fatalf("no se pudo leer el frame inicial: %v", err)
	}
	if message.Event != "CONNECTION_ESTABLISHED" || message.Data["user_id"] != "user-a" || message.Data["tenant_id"] != "tenant-a" {
		t.Fatalf("los headers sustituyeron la sesión WebSocket: %+v", message)
	}
}

func TestTerminalWebSocketRejectsHeaderOnlyIdentity(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	handler := NewTeacherHandler(nil, nil)
	handler.SetAuthService(authService)
	handler.SetTerminalAuthorizer(&stubTerminalAuthorizer{allow: true})
	server := httptest.NewServer(http.HandlerFunc(handler.HandleTerminalWebSocket))
	defer server.Close()

	headers := http.Header{}
	headers.Set("X-User-Id", "user-a")
	headers.Set("X-User-Role", "teacher")
	headers.Set("X-Tenant-Id", "tenant-a")
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/teacher/live-sessions/container-1/terminal",
		headers,
	)
	if err == nil {
		_ = conn.Close()
		t.Fatal("el terminal no debe abrirse con identidad por headers")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("terminal sin token: se esperaba 401, response=%v err=%v", response, err)
	}
}
