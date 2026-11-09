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

func newWebSocketTestAuth(t *testing.T) *services.AuthService {
	t.Helper()
	t.Setenv("GOOGLE_CLIENT_ID", "websocket-gap-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "websocket-gap-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/callback")
	t.Setenv("JWT_SECRET", "websocket-gap-jwt-secret")
	return services.NewAuthService(nil, nil).
		WithSessionRevocationRepository(memory.NewSessionRevocationRepository())
}

func newWebSocketTestToken(t *testing.T, userID, tenantID, role string, expiresAt time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"role":      role,
		"exp":       expiresAt.Unix(),
	})
	signed, err := token.SignedString([]byte("websocket-gap-jwt-secret"))
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}
	return signed
}

func dialWebSocket(t *testing.T, serverURL, token string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	return websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(serverURL, "http")+"/ws/v1/evaluations",
		headers,
	)
}

// El hub no debe compartir el índice de destinatarios entre tenants. Dos
// usuarios con el mismo user_id en tenants distintos son dos destinatarios
// diferentes; un mapa global por user_id filtra eventos entre instituciones.
func TestWebSocketSecurityTenantIsolationAtHub(t *testing.T) {
	hub := NewWebSocketHub()
	clientA := &WebSocketClient{
		hub:      hub,
		send:     make(chan []byte, 1),
		userID:   "shared-user",
		tenantID: "tenant-a",
	}
	clientB := &WebSocketClient{
		hub:      hub,
		send:     make(chan []byte, 1),
		userID:   "shared-user",
		tenantID: "tenant-b",
	}
	hub.Register(clientA)
	hub.Register(clientB)
	defer hub.Unregister(clientA)
	defer hub.Unregister(clientB)

	if len(hub.userClients) != 2 {
		t.Fatalf("el hub indexa %d destinatarios para el mismo user_id en dos tenants; se esperaban 2", len(hub.userClients))
	}

	hub.EmitToUserInTenant("tenant-a", "shared-user", WebSocketMessage{Event: "TENANT_A_ONLY"})
	select {
	case <-clientA.send:
	default:
		t.Fatal("el destinatario del tenant correcto no recibió el evento")
	}
	select {
	case payload := <-clientB.send:
		t.Fatalf("un cliente de otro tenant recibió el evento: %s", payload)
	default:
	}
}

// La expiración del JWT debe cortar el handshake. Una conexión nueva con un
// token vencido no puede usar el WebSocket.
func TestWebSocketSecurityRejectsExpiredSession(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(
		t,
		"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
		"00000000-0000-0000-0000-000000000001",
		"student",
		time.Now().Add(-time.Minute),
	)
	hub := NewWebSocketHub()
	go hub.Run()
	handler := NewWebSocketHandler(hub, authService)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	conn, response, err := dialWebSocket(t, server.URL, token)
	if err == nil {
		_ = conn.Close()
		t.Fatal("un token expirado no debe abrir el WebSocket")
	}
	if response == nil {
		t.Fatalf("se esperaba una respuesta HTTP de rechazo, err=%v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token expirado: se esperaba 401, se obtuvo %d", response.StatusCode)
	}
}

// Cerrar sesión debe invalidar la sesión del servidor. Si el logout solo borra
// la cookie del navegador, el token robado o conservado sigue abriendo un
// WebSocket hasta su expiración.
func TestWebSocketSecurityRejectsLoggedOutSession(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(
		t,
		"bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb",
		"00000000-0000-0000-0000-000000000001",
		"student",
		time.Now().Add(time.Hour),
	)

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	NewAuthHandler(authService).HandleLogout(httptest.NewRecorder(), logoutRequest)

	hub := NewWebSocketHub()
	go hub.Run()
	handler := NewWebSocketHandler(hub, authService)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	conn, response, err := dialWebSocket(t, server.URL, token)
	if err == nil {
		_ = conn.Close()
		t.Fatal("una sesión cerrada con logout no debe abrir el WebSocket")
	}
	if response == nil {
		t.Fatalf("se esperaba una respuesta HTTP de rechazo, err=%v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("sesión revocada: se esperaba 401, se obtuvo %d", response.StatusCode)
	}
}

// Una conexión ya abierta debe cerrarse cuando su sesión se revoca. No basta
// con impedir un handshake nuevo: el socket existente seguiría recibiendo
// eventos con un token que el usuario ya cerró.
func TestWebSocketSecurityClosesOpenSessionOnLogout(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(
		t,
		"dddddddd-dddd-4ddd-dddd-dddddddddddd",
		"00000000-0000-0000-0000-000000000001",
		"student",
		time.Now().Add(time.Hour),
	)
	hub := NewWebSocketHub()
	go hub.Run()
	hub.BindSessionRevocations(authService)
	handler := NewWebSocketHandler(hub, authService)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	conn, _, err := dialWebSocket(t, server.URL, token)
	if err != nil {
		t.Fatalf("no se pudo abrir el WebSocket con una sesión válida: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("se esperaba el evento inicial antes del logout: %v", err)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	NewAuthHandler(authService).HandleLogout(httptest.NewRecorder(), logoutRequest)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("el WebSocket abierto siguió recibiendo mensajes después del logout")
	}
}

// El terminal docente ejecuta comandos en el contenedor del estudiante. Un
// estudiante no debe abrir ese WebSocket ni enviar un exec.
func TestWebSocketSecurityStudentCannotOpenTeacherTerminal(t *testing.T) {
	authService := newWebSocketTestAuth(t)
	token := newWebSocketTestToken(
		t,
		"cccccccc-cccc-4ccc-cccc-cccccccccccc",
		"00000000-0000-0000-0000-000000000001",
		"student",
		time.Now().Add(time.Hour),
	)
	teacherHandler := NewTeacherHandler(services.NewTeacherService(nil, nil), nil)
	teacherHandler.SetAuthService(authService)
	server := httptest.NewServer(http.HandlerFunc(teacherHandler.HandleTerminalWebSocket))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/teacher/live-sessions/container-1/terminal",
		headers,
	)
	if err == nil {
		_ = conn.Close()
		t.Fatal("un estudiante no debe abrir el terminal docente")
	}
	if response == nil {
		t.Fatalf("se esperaba una respuesta HTTP de rechazo, err=%v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("terminal docente: se esperaba 403, se obtuvo %d", response.StatusCode)
	}
}
