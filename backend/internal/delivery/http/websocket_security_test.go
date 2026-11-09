package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

// Estas pruebas describen el contrato de seguridad que el motor debe ejecutar
// cuando cambia el handler: una sesión verificable, tenant del claim y un
// origen de navegador permitido. Los headers de identidad nunca sustituyen la
// sesión.
func TestWebSocketSecurityRejectsMissingToken(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()
	handler := NewWebSocketHandler(hub, nil)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err == nil {
		_ = conn.Close()
		t.Fatal("el handshake sin token debe rechazarse, no entregar una conexión")
	}
	if response == nil {
		t.Fatalf("se esperaba una respuesta HTTP de rechazo, err=%v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("handshake sin token: se esperaba 401, se obtuvo %d", response.StatusCode)
	}
}

func TestWebSocketSecurityRejectsIdentityHeaders(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()
	handler := NewWebSocketHandler(hub, nil)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	headers := http.Header{}
	headers.Set("X-User-Id", "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa")
	headers.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), headers)
	if err == nil {
		_ = conn.Close()
		t.Fatal("X-User-Id/X-Tenant-Id no deben sustituir una sesión válida")
	}
	if response == nil {
		t.Fatalf("se esperaba una respuesta HTTP de rechazo, err=%v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("identidad por headers: se esperaba 401, se obtuvo %d", response.StatusCode)
	}
}

func TestWebSocketSecurityRejectsUntrustedOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	hub := NewWebSocketHub()
	go hub.Run()
	handler := NewWebSocketHandler(hub, nil)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Origin", "https://origen-no-permitido.example")
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), headers)
	if err == nil {
		_ = conn.Close()
		t.Fatal("un origen no permitido no debe poder abrir el WebSocket")
	}
	if response == nil {
		t.Fatalf("se esperaba una respuesta HTTP de rechazo, err=%v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("origen no permitido: se esperaba 403, se obtuvo %d", response.StatusCode)
	}
}

func TestWebSocketSecurityAcceptsValidSession(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "websocket-test-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "websocket-test-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/callback")
	t.Setenv("JWT_SECRET", "websocket-test-jwt-secret")

	hub := NewWebSocketHub()
	go hub.Run()
	authService := services.NewAuthService(nil, nil).
		WithSessionRevocationRepository(memory.NewSessionRevocationRepository())
	handler := NewWebSocketHandler(hub, authService)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
		"tenant_id": "00000000-0000-0000-0000-000000000001",
		"role":      "student",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("websocket-test-jwt-secret"))
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+signed)
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/ws/v1/evaluations",
		headers,
	)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("una sesión válida debe abrir el WebSocket: status=%d err=%v", status, err)
	}
	_ = conn.Close()
}
