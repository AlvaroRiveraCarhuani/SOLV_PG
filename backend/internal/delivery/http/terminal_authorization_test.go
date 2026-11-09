package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

type stubTerminalAuthorizer struct {
	mu          sync.Mutex
	allow       bool
	tenantLimit string
	calls       int
}

func (s *stubTerminalAuthorizer) AuthorizeTerminal(_ context.Context, tenantID, _, containerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if !s.allow || containerID == "" {
		return domain.ErrNotFound
	}
	if s.tenantLimit != "" && tenantID != s.tenantLimit {
		return domain.ErrNotFound
	}
	return nil
}

func (s *stubTerminalAuthorizer) setAllow(allow bool) {
	s.mu.Lock()
	s.allow = allow
	s.mu.Unlock()
}

func (s *stubTerminalAuthorizer) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func newTerminalServer(t *testing.T, authService *services.AuthService, authorizer *stubTerminalAuthorizer) *httptest.Server {
	t.Helper()
	handler := NewTeacherHandler(services.NewTeacherService(nil, nil), nil)
	handler.SetAuthService(authService)
	handler.SetTerminalAuthorizer(authorizer)
	return httptest.NewServer(http.HandlerFunc(handler.HandleTerminalWebSocket))
}

func dialTerminal(t *testing.T, serverURL, token string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	return websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(serverURL, "http")+"/api/v1/teacher/live-sessions/container-1/terminal?container_id=container-1",
		headers,
	)
}

func readTerminalMessage(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var message map[string]interface{}
	if err := conn.ReadJSON(&message); err != nil {
		t.Fatalf("no se pudo leer el mensaje del terminal: %v", err)
	}
	return message
}

func TestTerminalAuthorizationAllowsAuthorizedTeacher(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "eeeeeeee-eeee-4eee-eeee-eeeeeeeeeeee", "tenant-a", "teacher", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: true, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, authService, authorizer)
	defer server.Close()

	conn, response, err := dialTerminal(t, server.URL, token)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("un docente autorizado debe abrir el terminal, status=%d err=%v", status, err)
	}
	defer func() { _ = conn.Close() }()

	init := readTerminalMessage(t, conn)
	if init["type"] != "init" || init["tenant_id"] != "tenant-a" {
		t.Fatalf("frame inicial inesperado: %+v", init)
	}
	if authorizer.callCount() != 1 {
		t.Fatalf("se esperaba una autorización de handshake, se hicieron %d", authorizer.callCount())
	}

	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err := conn.WriteJSON(map[string]string{"type": "exec", "command": "ls -la"}); err != nil {
		t.Fatalf("no se pudo enviar el exec autorizado: %v", err)
	}
	output := readTerminalMessage(t, conn)
	if output["type"] != "stdout" {
		t.Fatalf("se esperaba stdout para un exec autorizado, se obtuvo %+v", output)
	}
	if authorizer.callCount() != 2 {
		t.Fatalf("cada exec debe autorizarse, llamadas=%d", authorizer.callCount())
	}
}

func TestTerminalAuthorizationRejectsUnknownRole(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "ffffffff-ffff-4fff-ffff-ffffffffffff", "tenant-a", "assistant", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: true, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, authService, authorizer)
	defer server.Close()

	conn, response, err := dialTerminal(t, server.URL, token)
	if err == nil {
		_ = conn.Close()
		t.Fatal("un usuario sin rol docente no debe abrir el terminal")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("rol no autorizado: se esperaba 403, response=%v err=%v", response, err)
	}
	if authorizer.callCount() != 0 {
		t.Fatalf("el rol se rechaza antes de consultar permisos, llamadas=%d", authorizer.callCount())
	}
}

func TestTerminalAuthorizationRejectsOtherTenant(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "11111111-1111-4111-8111-111111111111", "tenant-b", "teacher", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: true, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, authService, authorizer)
	defer server.Close()

	conn, response, err := dialTerminal(t, server.URL, token)
	if err == nil {
		_ = conn.Close()
		t.Fatal("un docente de otro tenant no debe abrir el terminal")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("tenant ajeno: se esperaba 403, response=%v err=%v", response, err)
	}
}

func TestTerminalAuthorizationRejectsTeacherWithoutPermission(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "22222222-2222-4222-8222-222222222222", "tenant-a", "teacher", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: false, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, authService, authorizer)
	defer server.Close()

	conn, response, err := dialTerminal(t, server.URL, token)
	if err == nil {
		_ = conn.Close()
		t.Fatal("un docente sin permiso sobre la sesión no debe abrir el terminal")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("sin permiso: se esperaba 403, response=%v err=%v", response, err)
	}
}

func TestTerminalAuthorizationChecksEveryExec(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "33333333-3333-4333-8333-333333333333", "tenant-a", "teacher", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: true, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, authService, authorizer)
	defer server.Close()

	conn, _, err := dialTerminal(t, server.URL, token)
	if err != nil {
		t.Fatalf("no se pudo abrir el terminal autorizado: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = readTerminalMessage(t, conn)

	authorizer.setAllow(false)
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err := conn.WriteJSON(map[string]string{"type": "exec", "command": "rm -rf /tmp/demo"}); err != nil {
		t.Fatalf("no se pudo enviar el exec no autorizado: %v", err)
	}

	message := readTerminalMessage(t, conn)
	if message["type"] != "error" {
		payload, _ := json.Marshal(message)
		t.Fatalf("un exec sin autorización debe devolver error, se obtuvo %s", payload)
	}
	if authorizer.callCount() != 2 {
		t.Fatalf("el exec debe reautorizarse, llamadas=%d", authorizer.callCount())
	}
}

func TestTerminalAuthorizationClosesOpenTerminalOnLogout(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "44444444-4444-4444-8444-444444444444", "tenant-a", "teacher", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: true, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, authService, authorizer)
	defer server.Close()

	conn, _, err := dialTerminal(t, server.URL, token)
	if err != nil {
		t.Fatalf("no se pudo abrir el terminal autorizado: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = readTerminalMessage(t, conn)

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	NewAuthHandler(authService).HandleLogout(httptest.NewRecorder(), logoutRequest)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("el terminal abierto siguió leyendo después del logout")
	}
}

func TestTerminalAuthorizationClosesTerminalRevokedByAnotherInstance(t *testing.T) {
	store := memory.NewSessionRevocationRepository()
	instanceA := newWebSocketTestAuth(t).WithSessionRevocationRepository(store)
	instanceB := newWebSocketTestAuth(t).WithSessionRevocationRepository(store)
	token := newWebSocketTestToken(t, "66666666-6666-4666-8666-666666666666", "tenant-a", "teacher", time.Now().Add(time.Hour))
	authorizer := &stubTerminalAuthorizer{allow: true, tenantLimit: "tenant-a"}
	server := newTerminalServer(t, instanceA, authorizer)
	defer server.Close()

	conn, _, err := dialTerminal(t, server.URL, token)
	if err != nil {
		t.Fatalf("no se pudo abrir el terminal autorizado: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = readTerminalMessage(t, conn)

	if err := instanceB.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("la otra instancia no pudo revocar: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("el terminal debía cerrarse al observar la revocación de otra instancia")
	}
}

func TestTerminalAuthorizationRejectsRevokedHttpExec(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(t, "55555555-5555-4555-8555-555555555555", "tenant-a", "teacher", time.Now().Add(time.Hour))
	handler := NewTeacherHandler(services.NewTeacherService(nil, nil), nil)
	handler.SetAuthService(authService)

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	NewAuthHandler(authService).HandleLogout(httptest.NewRecorder(), logoutRequest)

	ctx := context.WithValue(context.Background(), domain.UserIDKey, "55555555-5555-4555-8555-555555555555")
	ctx = context.WithValue(ctx, domain.TenantIDKey, "tenant-a")
	ctx = context.WithValue(ctx, domain.UserRoleKey, "teacher")
	ctx = context.WithValue(ctx, domain.UserTokenKey, token)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/live-sessions/container-1/exec", strings.NewReader(`{"command":"ls"}`)).WithContext(ctx)
	req.SetPathValue("id", "container-1")
	recorder := httptest.NewRecorder()
	handler.PostTutorExec(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("un exec HTTP con sesión revocada debe responder 401, se obtuvo %d", recorder.Code)
	}
}
