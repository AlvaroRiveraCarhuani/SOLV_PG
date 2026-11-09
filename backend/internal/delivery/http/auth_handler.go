package httpdelivery

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"solv-backend/internal/core/services"
)

type AuthFailureRecorder interface {
	RecordAuthFailure(ip string) bool
}

type AuthHandler struct {
	authService *services.AuthService
	rateLimiter AuthFailureRecorder
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) WithRateLimiter(rl AuthFailureRecorder) *AuthHandler {
	h.rateLimiter = rl
	return h
}

func getCookieDomain(r *http.Request) string {
	if strings.Contains(r.Host, "localhost") || strings.Contains(r.Host, "127.0.0.1") {
		return ""
	}
	return strings.TrimSpace(os.Getenv("COOKIE_DOMAIN"))
}

const oauthStateCookieName = "solv_oauth_state"
const oauthStateCookieMaxAge = 600

func secureCookieRequest(r *http.Request) bool {
	isLocal := strings.Contains(r.Host, "localhost") || strings.Contains(r.Host, "127.0.0.1")
	return !isLocal && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
}

// sessionCookie construye la cookie de sesión con los atributos de seguridad
// que comparten login y logout.
func sessionCookie(r *http.Request, token string, maxAge int) *http.Cookie {
	cookie := &http.Cookie{
		Name:     "solv_session",
		Value:    token,
		Path:     "/",
		Domain:   getCookieDomain(r),
		HttpOnly: true,
		Secure:   secureCookieRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
	if maxAge < 0 {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(0, 0)
	}
	return cookie
}

func oauthStateCookie(r *http.Request, state string, maxAge int) *http.Cookie {
	cookie := &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		Domain:   getCookieDomain(r),
		HttpOnly: true,
		Secure:   secureCookieRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
	if maxAge < 0 {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(0, 0)
	}
	return cookie
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url, state, err := h.authService.BeginGoogleLogin()
	if err != nil {
		http.Error(w, "failed to start oauth login", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, oauthStateCookie(r, state, oauthStateCookieMaxAge))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// La cookie de state se invalida en toda respuesta del callback, incluso si
	// el state no coincide o el código falta.
	http.SetCookie(w, oauthStateCookie(r, "", -1))

	code := r.URL.Query().Get("code")
	if code == "" {
		h.recordAuthFailure(r)
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	if err := h.validateOAuthState(r); err != nil {
		h.recordAuthFailure(r)
		http.Error(w, "invalid oauth state", http.StatusForbidden)
		return
	}

	token, err := h.authService.CallbackGoogle(r.Context(), code)
	if err != nil {
		h.recordAuthFailure(r)
		if errors.Is(err, services.ErrUnauthorizedDomain) ||
			errors.Is(err, services.ErrEmailNotVerified) ||
			strings.Contains(err.Error(), "unauthorized: email domain not allowed") ||
			strings.Contains(err.Error(), "google email is not verified") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "failed to authenticate: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Seteo de la cookie HttpOnly solv_session adaptable a entorno local vs producción
	http.SetCookie(w, sessionCookie(r, token, 86400))

	// Si un cliente API o test solicita JSON explícitamente
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})
		return
	}

	// Extraer el rol del usuario desde los claims del token
	targetRole := "student"
	if claims, err := h.authService.ValidateSessionTokenContext(r.Context(), token); err == nil {
		if rVal, ok := claims["role"].(string); ok && rVal != "" {
			targetRole = rVal
		}
	}

	// Redirección dinámica según el rol y entorno
	frontendBase := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	var redirectURL string
	if frontendBase != "" {
		redirectURL = strings.TrimSuffix(frontendBase, "/") + "/" + targetRole
	} else if strings.Contains(r.Host, "localhost") || strings.Contains(r.Host, "127.0.0.1") {
		redirectURL = fmt.Sprintf("http://localhost:4200/%s?token=%s", targetRole, token)
	} else {
		redirectURL = "/" + targetRole
	}

	// Redireccion construida desde FRONTEND_URL de configuracion o paths fijos, r.Host no se concatena.
	//nolint:gosec
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// validateOAuthState compara el state del callback con el valor guardado en la
// cookie HttpOnly y consume el state del almacén de un solo uso. La comparación
// es de tiempo constante.
func (h *AuthHandler) validateOAuthState(r *http.Request) error {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	cookie, err := r.Cookie(oauthStateCookieName)
	if state == "" || err != nil || strings.TrimSpace(cookie.Value) == "" {
		return services.ErrOAuthStateMissing
	}
	if subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		return services.ErrOAuthStateInvalid
	}
	if h.authService == nil {
		return services.ErrOAuthStateInvalid
	}
	return h.authService.ConsumeOAuthState(state)
}

func (h *AuthHandler) VerifyAuth(w http.ResponseWriter, r *http.Request) {
	var tokenStr string
	cookie, err := r.Cookie("solv_session")
	if err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	} else {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		SendError(w, http.StatusUnauthorized, "missing session cookie or authorization token", "Sesión no iniciada o cookie inexistente")
		return
	}

	claims, err := h.authService.ValidateSessionTokenContext(r.Context(), tokenStr)
	if err != nil {
		if errors.Is(err, services.ErrJWTExpired) {
			SendError(w, http.StatusForbidden, "session expired", "La sesión ha expirado")
			return
		}
		SendError(w, http.StatusUnauthorized, "invalid session token", "Token de sesión inválido")
		return
	}

	// Inyectar headers que Traefik ForwardAuth puede propagar a servicios aguas abajo
	if userID, ok := claims["user_id"].(string); ok {
		w.Header().Set("X-User-Id", userID)
	}
	if role, ok := claims["role"].(string); ok {
		w.Header().Set("X-User-Role", role)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var tokenString string
	if cookie, err := r.Cookie("solv_session"); err == nil && cookie.Value != "" {
		tokenString = cookie.Value
	} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if tokenString != "" {
		if h.authService == nil {
			SendError(w, http.StatusServiceUnavailable, "session_revocation_unavailable", "No se pudo revocar la sesión; intenta de nuevo")
			return
		}
		if err := h.authService.RevokeSessionToken(r.Context(), tokenString); err != nil {
			SendError(w, http.StatusServiceUnavailable, "session_revocation_unavailable", "No se pudo revocar la sesión; intenta de nuevo")
			return
		}
	}

	http.SetCookie(w, sessionCookie(r, "", -1))

	SendJSON(w, http.StatusOK, map[string]string{"status": "logged_out"}, "Sesión cerrada exitosamente")
}

func (h *AuthHandler) recordAuthFailure(r *http.Request) {
	if h.rateLimiter == nil {
		return
	}
	ip := r.RemoteAddr
	if forward := r.Header.Get("X-Forwarded-For"); forward != "" {
		ip = strings.TrimSpace(strings.Split(forward, ",")[0])
	}
	h.rateLimiter.RecordAuthFailure(ip)
}
