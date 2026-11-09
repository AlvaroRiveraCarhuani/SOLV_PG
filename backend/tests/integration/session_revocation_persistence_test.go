package integration

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

const sessionRevocationTestSecret = "session-revocation-integration-secret"

func openRevocationTestDB(t *testing.T) *database.Database {
	t.Helper()
	dsn := getTestDSN()
	if strings.TrimSpace(dsn) == "" {
		skipOrFailInfra(t, "TEST_DB_DSN o TEST_DATABASE_URL no configurado")
		return nil
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL de tests no disponible: %v", err)
		return nil
	}
	t.Cleanup(func() { _ = db.GetDB().Close() })

	if err := database.RunMigrations(db.GetDB().DB, "../../migrations"); err != nil {
		t.Fatalf("no se pudieron aplicar las migraciones de tests: %v", err)
	}
	return db
}

func newRevocationTestAuth(t *testing.T, repo domain.SessionRevocationRepository) *services.AuthService {
	t.Helper()
	t.Setenv("GOOGLE_CLIENT_ID", "revocation-integration-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "revocation-integration-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/callback")
	t.Setenv("JWT_SECRET", sessionRevocationTestSecret)
	return services.NewAuthService(nil, nil).WithSessionRevocationRepository(repo)
}

func signedSessionRevocationToken(t *testing.T, jti string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"jti":       jti,
		"user_id":   "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
		"tenant_id": "00000000-0000-0000-0000-000000000001",
		"role":      "student",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(sessionRevocationTestSecret))
	if err != nil {
		t.Fatalf("no se pudo firmar el token de integración: %v", err)
	}
	return signed
}

func cleanupRevocationTestRow(t *testing.T, db *database.Database, sessionID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.GetDB().Exec(`DELETE FROM session_revocations WHERE session_id = $1`, sessionID)
	})
}

func TestSessionRevocationPersistsAcrossServiceRestart(t *testing.T) {
	db := openRevocationTestDB(t)
	jti := uuid.NewString()
	sessionID := "jti:" + jti
	cleanupRevocationTestRow(t, db, sessionID)
	token := signedSessionRevocationToken(t, jti)

	first := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	if err := first.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("la primera instancia no pudo persistir la revocación: %v", err)
	}
	var storedID string
	if err := db.GetDB().Get(&storedID, `SELECT session_id FROM session_revocations WHERE session_id = $1`, sessionID); err != nil {
		t.Fatalf("no se encontró la revocación persistida: %v", err)
	}
	if storedID != sessionID || strings.Contains(storedID, token) {
		t.Fatalf("se esperaba solo el identificador de sesión, stored=%q", storedID)
	}

	// Un servicio nuevo representa un reinicio: no comparte memoria con first.
	restarted := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	if _, err := restarted.ValidateSessionTokenContext(context.Background(), token); !errors.Is(err, services.ErrJWTRevoked) {
		t.Fatalf("la revocación no sobrevivió al reinicio del servicio: %v", err)
	}
}

func TestSessionRevocationIsSharedAcrossInstances(t *testing.T) {
	db := openRevocationTestDB(t)
	jti := uuid.NewString()
	sessionID := "jti:" + jti
	cleanupRevocationTestRow(t, db, sessionID)
	token := signedSessionRevocationToken(t, jti)

	instanceA := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	instanceB := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))

	if _, err := instanceA.ValidateSessionTokenContext(context.Background(), token); err != nil {
		t.Fatalf("el token vigente no debe rechazarse antes del logout: %v", err)
	}
	if err := instanceA.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("la instancia A no pudo revocar: %v", err)
	}
	if _, err := instanceB.ValidateSessionTokenContext(context.Background(), token); !errors.Is(err, services.ErrJWTRevoked) {
		t.Fatalf("la instancia B no vio la revocación persistida: %v", err)
	}
}

func TestLogoutPersistsRevocationBeforeResponding(t *testing.T) {
	db := openRevocationTestDB(t)
	jti := uuid.NewString()
	sessionID := "jti:" + jti
	cleanupRevocationTestRow(t, db, sessionID)
	token := signedSessionRevocationToken(t, jti)

	instance := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	recorder := httptest.NewRecorder()
	httpdelivery.NewAuthHandler(instance).HandleLogout(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("logout debía responder 200 tras persistir: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	restarted := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	if _, err := restarted.ValidateSessionTokenContext(context.Background(), token); !errors.Is(err, services.ErrJWTRevoked) {
		t.Fatalf("el logout no persistió la revocación antes de responder: %v", err)
	}
}

func TestRevokedSessionRejectedByHTTPMiddleware(t *testing.T) {
	db := openRevocationTestDB(t)
	jti := uuid.NewString()
	sessionID := "jti:" + jti
	cleanupRevocationTestRow(t, db, sessionID)
	token := signedSessionRevocationToken(t, jti)

	instance := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	if err := instance.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("no se pudo revocar: %v", err)
	}

	called := false
	handler := middleware.SessionRevocationMiddleware(instance)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/live-sessions/container-1", nil)
	request.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized || called {
		t.Fatalf("HTTP debe rechazar el token revocado antes del handler: status=%d called=%v", recorder.Code, called)
	}
}

func TestRevokedSessionRejectedByWebSocketHandshake(t *testing.T) {
	db := openRevocationTestDB(t)
	jti := uuid.NewString()
	sessionID := "jti:" + jti
	cleanupRevocationTestRow(t, db, sessionID)
	token := signedSessionRevocationToken(t, jti)

	instance := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	if err := instance.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("no se pudo revocar: %v", err)
	}

	hub := httpdelivery.NewWebSocketHub()
	go hub.Run()
	handler := httpdelivery.NewWebSocketHandler(hub, instance)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/ws/v1/evaluations",
		headers,
	)
	if err == nil {
		_ = conn.Close()
		t.Fatal("el handshake WebSocket no debe aceptar una sesión revocada")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("handshake revocado: response=%v err=%v", response, err)
	}
}

func TestRevocationFailsClosedWhenPostgresIsUnavailable(t *testing.T) {
	_ = openRevocationTestDB(t)

	raw, err := sql.Open("postgres", getTestDSN())
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión de prueba: %v", err)
	}
	if err := raw.Ping(); err != nil {
		t.Fatalf("PostgreSQL de tests no disponible: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("no se pudo cerrar la conexión de prueba: %v", err)
	}

	instance := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(sqlx.NewDb(raw, "postgres")))
	token := signedSessionRevocationToken(t, uuid.NewString())
	if _, err := instance.ValidateSessionTokenContext(context.Background(), token); err == nil || errors.Is(err, services.ErrJWTRevoked) {
		t.Fatalf("con PostgreSQL caído la validación debe devolver un error de almacenamiento, no una sesión válida ni revocada: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	httpdelivery.NewAuthHandler(instance).HandleLogout(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("logout con PostgreSQL caído: status=%d, se esperaba 503", recorder.Code)
	}
	if cookies := recorder.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("el logout no debe borrar la cookie si no pudo persistir la revocación: %v", cookies)
	}

	nextCalled := false
	protected := middleware.SessionRevocationMiddleware(instance)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	middlewareRecorder := httptest.NewRecorder()
	middlewareRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
	middlewareRequest.Header.Set("Authorization", "Bearer "+token)
	protected.ServeHTTP(middlewareRecorder, middlewareRequest)
	if middlewareRecorder.Code != http.StatusServiceUnavailable || nextCalled {
		t.Fatalf("middleware con PostgreSQL caído: status=%d next=%v, se esperaba 503 y no continuar", middlewareRecorder.Code, nextCalled)
	}
}

func TestRevocationClosesOpenSocketFromAnotherInstance(t *testing.T) {
	db := openRevocationTestDB(t)
	jti := uuid.NewString()
	sessionID := "jti:" + jti
	cleanupRevocationTestRow(t, db, sessionID)
	token := signedSessionRevocationToken(t, jti)

	instanceA := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))
	instanceB := newRevocationTestAuth(t, postgres.NewPostgresSessionRevocationRepository(db.GetDB()))

	hub := httpdelivery.NewWebSocketHub()
	go hub.Run()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub.StartRevocationSync(ctx, postgres.NewPostgresSessionRevocationRepository(db.GetDB()), 50*time.Millisecond)

	handler := httpdelivery.NewWebSocketHandler(hub, instanceA)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleEvaluationWS))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/ws/v1/evaluations",
		headers,
	)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("no se pudo abrir el socket antes de la revocación: status=%d err=%v", status, err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("se esperaba el evento inicial: %v", err)
	}

	if err := instanceB.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("la otra instancia no pudo revocar: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("el sondeo persistente debía cerrar la conexión abierta en la otra instancia")
	}
}

func TestRevocationExpirationAllowsCleanup(t *testing.T) {
	db := openRevocationTestDB(t)
	repo := postgres.NewPostgresSessionRevocationRepository(db.GetDB())
	expiredID := "jti:" + uuid.NewString()
	activeID := "jti:" + uuid.NewString()
	cleanupRevocationTestRow(t, db, expiredID)
	cleanupRevocationTestRow(t, db, activeID)

	if err := repo.RevokeSession(context.Background(), expiredID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("no se pudo insertar la revocación vencida: %v", err)
	}
	if err := repo.RevokeSession(context.Background(), activeID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("no se pudo insertar la revocación vigente: %v", err)
	}
	revoked, err := repo.IsSessionRevoked(context.Background(), expiredID)
	if err != nil || revoked {
		t.Fatalf("una revocación vencida no debe considerarse activa: revoked=%v err=%v", revoked, err)
	}
	if err := repo.DeleteExpiredSessions(context.Background()); err != nil {
		t.Fatalf("no se pudo limpiar: %v", err)
	}
	var expiredCount int
	if err := db.GetDB().Get(&expiredCount, `SELECT COUNT(*) FROM session_revocations WHERE session_id = $1`, expiredID); err != nil {
		t.Fatalf("no se pudo verificar la limpieza: %v", err)
	}
	if expiredCount != 0 {
		t.Fatalf("la revocación vencida no se eliminó, filas=%d", expiredCount)
	}
}
