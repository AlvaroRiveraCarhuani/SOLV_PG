package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionCookieSecurityAttributes(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:3000/api/v1/auth/callback", nil)
		cookie := sessionCookie(req, "jwt-token", 86400)
		if cookie.Name != "solv_session" || cookie.Value != "jwt-token" {
			t.Fatalf("cookie inesperada: %+v", cookie)
		}
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
			t.Fatalf("faltan atributos de seguridad: %+v", cookie)
		}
		if cookie.Secure {
			t.Fatal("localhost no debe exigir Secure para desarrollo")
		}
		if cookie.MaxAge != 86400 {
			t.Fatalf("MaxAge inesperado: %d", cookie.MaxAge)
		}
	})

	t.Run("produccion", func(t *testing.T) {
		t.Setenv("COOKIE_DOMAIN", ".solv.example")
		req := httptest.NewRequest(http.MethodGet, "https://app.solv.example/api/v1/auth/callback", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		cookie := sessionCookie(req, "jwt-token", 86400)
		if !cookie.Secure || !cookie.HttpOnly {
			t.Fatalf("producción debe usar Secure y HttpOnly: %+v", cookie)
		}
		if cookie.Domain != ".solv.example" {
			t.Fatalf("dominio de cookie inesperado: %q", cookie.Domain)
		}
	})

	t.Run("logout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://localhost:3000/api/v1/auth/logout", nil)
		cookie := sessionCookie(req, "", -1)
		if cookie.MaxAge != -1 || cookie.Value != "" {
			t.Fatalf("la cookie de logout debe expirar: %+v", cookie)
		}
		if !cookie.Expires.Before(time.Now()) {
			t.Fatalf("Expires debe estar en el pasado: %v", cookie.Expires)
		}
	})
}

func TestOAuthStateCookieSecurityAttributes(t *testing.T) {
	local := httptest.NewRequest(http.MethodGet, "http://localhost:3000/api/v1/auth/login", nil)
	localCookie := oauthStateCookie(local, "state-value", oauthStateCookieMaxAge)
	if !localCookie.HttpOnly || localCookie.SameSite != http.SameSiteLaxMode || localCookie.Path != "/" {
		t.Fatalf("cookie de state local sin atributos seguros: %+v", localCookie)
	}

	t.Setenv("COOKIE_DOMAIN", ".solv.example")
	secure := httptest.NewRequest(http.MethodGet, "https://app.solv.example/api/v1/auth/login", nil)
	secure.Header.Set("X-Forwarded-Proto", "https")
	secureCookie := oauthStateCookie(secure, "state-value", oauthStateCookieMaxAge)
	if !secureCookie.Secure || secureCookie.Domain != ".solv.example" {
		t.Fatalf("cookie de state de producción sin Secure/Domain: %+v", secureCookie)
	}
}

func TestGoogleCallbackRejectsMissingCodeWithoutSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)
	rec := httptest.NewRecorder()

	NewAuthHandler(nil).HandleGoogleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("callback sin code debe responder 400, se obtuvo %d", rec.Code)
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "solv_session" {
			t.Fatal("un callback fallido no debe emitir cookie de sesión")
		}
	}
}
