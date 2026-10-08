package httpdelivery

import (
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

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.authService.GetLoginURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		h.recordAuthFailure(r)
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	token, err := h.authService.CallbackGoogle(r.Context(), code)
	if err != nil {
		h.recordAuthFailure(r)
		if errors.Is(err, services.ErrUnauthorizedDomain) || strings.Contains(err.Error(), "unauthorized: email domain not allowed") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "failed to authenticate: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Seteo de la cookie HttpOnly solv_session adaptable a entorno local vs producción
	cookieDomain := getCookieDomain(r)
	isLocal := strings.Contains(r.Host, "localhost") || strings.Contains(r.Host, "127.0.0.1")
	isSecure := !isLocal && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")

	cookie := &http.Cookie{
		Name:     "solv_session",
		Value:    token,
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 horas
	}
	http.SetCookie(w, cookie)

	// Si un cliente API o test solicita JSON explícitamente
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})
		return
	}

	// Extraer el rol del usuario desde los claims del token
	targetRole := "student"
	if claims, err := h.authService.ValidateSessionToken(token); err == nil {
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

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
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

	claims, err := h.authService.ValidateSessionToken(tokenStr)
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
	cookieDomain := getCookieDomain(r)
	isLocal := strings.Contains(r.Host, "localhost") || strings.Contains(r.Host, "127.0.0.1")
	isSecure := !isLocal && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
	cookie := &http.Cookie{
		Name:     "solv_session",
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, cookie)

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

