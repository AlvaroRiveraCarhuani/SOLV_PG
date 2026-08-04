package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"solv-backend/internal/core/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func getCookieDomain() string {
	domain := strings.TrimSpace(os.Getenv("COOKIE_DOMAIN"))
	return domain
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.authService.GetLoginURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	token, err := h.authService.CallbackGoogle(r.Context(), code)
	if err != nil {
		if err.Error() == "unauthorized: email must end with @uab.edu.bo" {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "failed to authenticate: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ajuste 2: Seteo aditivo de la cookie HttpOnly solv_session manteniendo el JSON intacto para Angular
	cookieDomain := getCookieDomain()
	cookie := &http.Cookie{
		Name:     "solv_session",
		Value:    token,
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 horas
	}
	http.SetCookie(w, cookie)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
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
	cookieDomain := getCookieDomain()
	cookie := &http.Cookie{
		Name:     "solv_session",
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, cookie)

	SendJSON(w, http.StatusOK, map[string]string{"status": "logged_out"}, "Sesión cerrada exitosamente")
}
