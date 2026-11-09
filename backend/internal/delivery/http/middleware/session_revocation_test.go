package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/storage/memory"
)

func newSessionRevocationAuth(t *testing.T) *services.AuthService {
	t.Helper()
	t.Setenv("GOOGLE_CLIENT_ID", "session-revocation-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "session-revocation-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/callback")
	t.Setenv("JWT_SECRET", "session-revocation-jwt-secret")
	return services.NewAuthService(nil, nil).
		WithSessionRevocationRepository(memory.NewSessionRevocationRepository())
}

func signSessionRevocationToken(t *testing.T) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
		"tenant_id": "tenant-a",
		"role":      "student",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("session-revocation-jwt-secret"))
	if err != nil {
		t.Fatalf("no se pudo firmar el token: %v", err)
	}
	return signed
}

func TestSessionRevocationMiddlewareRejectsRevokedToken(t *testing.T) {
	authService := newSessionRevocationAuth(t)
	token := signSessionRevocationToken(t)
	if err := authService.RevokeSessionToken(context.Background(), token); err != nil {
		t.Fatalf("no se pudo revocar el token de prueba: %v", err)
	}

	called := false
	handler := SessionRevocationMiddleware(authService)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/live-sessions/container-1", nil)
	request.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba 401 para el JWT revocado, se obtuvo %d", recorder.Code)
	}
	if called {
		t.Fatal("la petición revocada no debe llegar al handler")
	}
}

func TestSessionRevocationMiddlewareAllowsActiveToken(t *testing.T) {
	authService := newSessionRevocationAuth(t)
	token := signSessionRevocationToken(t)

	called := false
	handler := SessionRevocationMiddleware(authService)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/live-sessions/container-1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent || !called {
		t.Fatalf("un JWT vigente debe llegar al handler, status=%d called=%v", recorder.Code, called)
	}
}

func TestSessionRevocationMiddlewareFailsClosedWithoutStore(t *testing.T) {
	called := false
	handler := SessionRevocationMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/live-sessions/container-1", nil)
	request.AddCookie(&http.Cookie{Name: "solv_session", Value: "un-token-cualquiera"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("sin repositorio de revocación se esperaba 503, se obtuvo %d", recorder.Code)
	}
	if called {
		t.Fatal("la petición no debe llegar al handler cuando no se puede verificar la revocación")
	}
}
