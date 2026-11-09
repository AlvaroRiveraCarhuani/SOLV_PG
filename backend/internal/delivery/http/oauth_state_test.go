package httpdelivery

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"solv-backend/internal/core/services"
)

func beginOAuthLoginForTest(t *testing.T) (*AuthHandler, *httptest.ResponseRecorder, *http.Cookie, string) {
	t.Helper()
	authService := newWebSocketTestAuth(t)
	handler := NewAuthHandler(authService)
	req := httptest.NewRequest(http.MethodPost, "http://localhost:3000/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()

	handler.HandleGoogleLogin(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("el login OAuth debe redirigir, se obtuvo %d: %s", rec.Code, rec.Body.String())
	}
	var stateCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == oauthStateCookieName {
			stateCookie = cookie
			break
		}
	}
	if stateCookie == nil || strings.TrimSpace(stateCookie.Value) == "" {
		t.Fatal("el login debe emitir una cookie de state")
	}
	if !stateCookie.HttpOnly || stateCookie.SameSite != http.SameSiteLaxMode || stateCookie.Path != "/" {
		t.Fatalf("cookie de state sin atributos seguros: %+v", stateCookie)
	}
	if stateCookie.MaxAge != oauthStateCookieMaxAge {
		t.Fatalf("MaxAge de state inesperado: %d", stateCookie.MaxAge)
	}

	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Location inválida: %v", err)
	}
	state := location.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		t.Fatalf("el state de la URL y la cookie deben coincidir: url=%q cookie=%q", state, stateCookie.Value)
	}
	return handler, rec, stateCookie, state
}

func callbackStateRequest(state string, cookie *http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?code=fake&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

func TestOAuthStateValidAndSingleUse(t *testing.T) {
	handler, _, cookie, state := beginOAuthLoginForTest(t)

	req := callbackStateRequest(state, cookie)
	if err := handler.validateOAuthState(req); err != nil {
		t.Fatalf("un state válido debe aceptarse: %v", err)
	}
	if err := handler.validateOAuthState(req); !errors.Is(err, services.ErrOAuthStateInvalid) {
		t.Fatalf("un state reutilizado debe rechazarse, se obtuvo %v", err)
	}
}

func TestOAuthStateRejectsMissingCookieOrState(t *testing.T) {
	handler, _, cookie, state := beginOAuthLoginForTest(t)

	if err := handler.validateOAuthState(callbackStateRequest(state, nil)); !errors.Is(err, services.ErrOAuthStateMissing) {
		t.Fatalf("sin cookie de state debe rechazarse, se obtuvo %v", err)
	}
	if err := handler.validateOAuthState(callbackStateRequest("", cookie)); !errors.Is(err, services.ErrOAuthStateMissing) {
		t.Fatalf("sin state en la URL debe rechazarse, se obtuvo %v", err)
	}
}

func TestOAuthStateRejectsIncorrectValueWithoutConsumingValidState(t *testing.T) {
	handler, _, cookie, state := beginOAuthLoginForTest(t)

	wrong := callbackStateRequest("incorrect-state", cookie)
	if err := handler.validateOAuthState(wrong); !errors.Is(err, services.ErrOAuthStateInvalid) {
		t.Fatalf("un state distinto al de la cookie debe rechazarse, se obtuvo %v", err)
	}

	valid := callbackStateRequest(state, cookie)
	if err := handler.validateOAuthState(valid); err != nil {
		t.Fatalf("un intento incorrecto no debe consumir el state legítimo: %v", err)
	}
}

func TestOAuthCallbackRejectsMissingOrInvalidStateBeforeExchange(t *testing.T) {
	handler, _, cookie, _ := beginOAuthLoginForTest(t)

	missing := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?code=fake", nil)
	missingRec := httptest.NewRecorder()
	handler.HandleGoogleCallback(missingRec, missing)
	if missingRec.Code != http.StatusForbidden {
		t.Fatalf("callback sin state debe responder 403, se obtuvo %d", missingRec.Code)
	}

	invalid := callbackStateRequest("incorrect-state", cookie)
	invalidRec := httptest.NewRecorder()
	handler.HandleGoogleCallback(invalidRec, invalid)
	if invalidRec.Code != http.StatusForbidden {
		t.Fatalf("callback con state incorrecto debe responder 403, se obtuvo %d", invalidRec.Code)
	}

	for _, rec := range []*httptest.ResponseRecorder{missingRec, invalidRec} {
		stateCleared := false
		for _, responseCookie := range rec.Result().Cookies() {
			if responseCookie.Name == "solv_session" {
				t.Fatal("un callback rechazado no debe emitir sesión")
			}
			if responseCookie.Name == oauthStateCookieName {
				stateCleared = responseCookie.MaxAge < 0
			}
		}
		if !stateCleared {
			t.Fatal("el callback debe invalidar la cookie de state")
		}
	}
}
