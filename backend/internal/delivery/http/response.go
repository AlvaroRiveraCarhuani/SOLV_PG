package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
)

type GlobalResponse struct {
	Data    any    `json:"data"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func SendJSON(w http.ResponseWriter, status int, data any, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(GlobalResponse{
		Data:    data,
		Error:   "",
		Message: msg,
	})
}

func SendError(w http.ResponseWriter, status int, err string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(GlobalResponse{
		Data:    nil,
		Error:   err,
		Message: msg,
	})
}

// requestTenantID resuelve el tenant exclusivamente desde el contexto
// autenticado. Los headers no son una fuente de identidad, ni siquiera como
// compatibilidad interna: WithTenant es quien escribe el contexto.
func requestTenantID(r *http.Request) string {
	return strings.TrimSpace(domain.GetTenantID(r.Context()))
}

// requestUserID resuelve el usuario exclusivamente desde el contexto
// autenticado. No hay identidad anónima ni fallback a headers.
func requestUserID(r *http.Request) string {
	return strings.TrimSpace(domain.GetUserID(r.Context()))
}

// requestRole resuelve el rol exclusivamente desde el contexto autenticado.
func requestRole(r *http.Request) string {
	return strings.TrimSpace(domain.GetUserRole(r.Context()))
}

// requestUserEmail resuelve el correo exclusivamente desde el contexto
// autenticado.
func requestUserEmail(r *http.Request) string {
	return strings.TrimSpace(domain.GetUserEmail(r.Context()))
}

func hasAnyRole(r *http.Request, allowed ...string) bool {
	role := requestRole(r)
	if role == "" {
		return false
	}
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

// requireRole aplica una allowlist explícita. Un rol vacío o desconocido no
// pasa: no se usan exclusiones del tipo "todo menos student".
func requireRole(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	if hasAnyRole(r, allowed...) {
		return true
	}
	SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: el rol de la sesión no está autorizado")
	return false
}

func requireAdminRole(w http.ResponseWriter, r *http.Request) bool {
	return requireRole(w, r, "admin")
}

// sessionTokenFromRequest extrae el token de la cookie HttpOnly o del Bearer.
// No acepta tokens por query string.
func sessionTokenFromRequest(r *http.Request) string {
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if cookie, err := r.Cookie("solv_session"); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func getTenantFromCtx(r *http.Request) string {
	return requestTenantID(r)
}

// getUserIDFromCtx conserva el nombre usado por los handlers existentes.
func getUserIDFromCtx(r *http.Request) string {
	return requestUserID(r)
}
