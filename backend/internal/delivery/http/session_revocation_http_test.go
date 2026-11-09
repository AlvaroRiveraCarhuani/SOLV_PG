package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/services"
)

type failingSessionRevocationStore struct{}

func (f *failingSessionRevocationStore) RevokeSession(context.Context, string, time.Time) error {
	return errors.New("postgres unavailable")
}

func (f *failingSessionRevocationStore) IsSessionRevoked(context.Context, string) (bool, error) {
	return false, errors.New("postgres unavailable")
}

func (f *failingSessionRevocationStore) ListRevokedSessions(context.Context) ([]string, error) {
	return nil, errors.New("postgres unavailable")
}

func (f *failingSessionRevocationStore) DeleteExpiredSessions(context.Context) error {
	return errors.New("postgres unavailable")
}

func TestLogoutFailsWhenRevocationCannotBePersisted(t *testing.T) {
	authService := newWebSocketTestAuth(t).
		WithSessionRevocationRepository(&failingSessionRevocationStore{})
	token := newWebSocketTestToken(t, "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", "tenant-a", "student", time.Now().Add(time.Hour))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "solv_session", Value: token})
	recorder := httptest.NewRecorder()
	NewAuthHandler(authService).HandleLogout(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("logout sin persistencia debe fallar de forma segura: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatal("el logout no debe confirmar el cierre ni borrar la cookie cuando falla la persistencia")
	}
}

func TestAuthenticationFailsClosedWhenRevocationStoreIsUnavailable(t *testing.T) {
	authService := newWebSocketTestAuth(t).
		WithSessionRevocationRepository(&failingSessionRevocationStore{})
	token := newWebSocketTestToken(t, "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb", "tenant-a", "student", time.Now().Add(time.Hour))

	if _, err := authService.ValidateSessionTokenContext(context.Background(), token); err == nil {
		t.Fatal("un token no verificable no debe aceptarse cuando el repositorio falla")
	}
}

func TestNoRevocationStoreFailsClosed(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "no-store-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "no-store-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/callback")
	t.Setenv("JWT_SECRET", "no-store-jwt-secret")

	authService := newWebSocketTestAuthWithoutStore(t)
	token := newWebSocketTestToken(t, "cccccccc-cccc-4ccc-cccc-cccccccccccc", "tenant-a", "student", time.Now().Add(time.Hour))
	if _, err := authService.ValidateSessionTokenContext(context.Background(), token); err == nil {
		t.Fatal("sin repositorio de revocación la autenticación debe fallar cerrada")
	}
}

func newWebSocketTestAuthWithoutStore(t *testing.T) *services.AuthService {
	t.Helper()
	return services.NewAuthService(nil, nil)
}
