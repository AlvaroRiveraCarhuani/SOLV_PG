package httpdelivery

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"solv-backend/internal/core/services"
)

var errWebSocketUnauthorized = errors.New("websocket session required")

type websocketPrincipal struct {
	userID    string
	tenantID  string
	role      string
	sessionID string
	expiresAt time.Time
}

// websocketSession extrae una sesión válida del encabezado Authorization o de
// la cookie HttpOnly. No acepta tokens por query string: la política prohíbe
// JWT en URLs y un handshake no puede convertir cabeceras de identidad en sesión.
func websocketSession(r *http.Request, authService *services.AuthService) (websocketPrincipal, error) {
	if authService == nil {
		return websocketPrincipal{}, errWebSocketUnauthorized
	}

	token := ""
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if token == "" {
		if cookie, err := r.Cookie("solv_session"); err == nil {
			token = strings.TrimSpace(cookie.Value)
		}
	}
	if token == "" {
		return websocketPrincipal{}, errWebSocketUnauthorized
	}

	claims, err := authService.ValidateSessionTokenContext(r.Context(), token)
	if err != nil {
		return websocketPrincipal{}, errWebSocketUnauthorized
	}

	userID, _ := claims["user_id"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	role, _ := claims["role"].(string)
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(tenantID) == "" {
		return websocketPrincipal{}, errWebSocketUnauthorized
	}

	principal := websocketPrincipal{
		userID:    userID,
		tenantID:  tenantID,
		role:      role,
		sessionID: services.SessionIdentifierFromClaims(token, claims),
	}
	if exp, ok := claims["exp"].(float64); ok && exp > 0 {
		principal.expiresAt = time.Unix(int64(exp), 0)
	}

	return principal, nil
}

// websocketOriginAllowed aplica la misma allowlist que el CORS HTTP. Un
// handshake sin Origin (cliente nativo o test) se acepta porque no hay contexto
// de navegador que falsificar; un Origin presente debe ser el mismo host o una
// entrada explícita de CORS_ALLOWED_ORIGINS/ALLOWED_ORIGINS.
func websocketOriginAllowed(r *http.Request) bool {
	rawOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if rawOrigin == "" {
		return true
	}

	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Host == "" || origin.User != nil {
		return false
	}
	if origin.Scheme != "http" && origin.Scheme != "https" {
		return false
	}

	if sameWebSocketHost(origin, r.Host) {
		return true
	}

	for _, allowed := range websocketAllowedOrigins() {
		parsed, err := url.Parse(allowed)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		if sameOrigin(origin, parsed) {
			return true
		}
	}

	return isLoopbackDevelopmentOrigin(origin)
}

func websocketAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	}
	if raw == "" {
		return nil
	}

	origins := make([]string, 0)
	for _, entry := range strings.Split(raw, ",") {
		if candidate := strings.TrimSpace(entry); candidate != "" {
			origins = append(origins, candidate)
		}
	}
	return origins
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(normalizeOriginHost(a), normalizeOriginHost(b)) &&
		strings.EqualFold(a.Scheme, b.Scheme)
}

func sameWebSocketHost(origin *url.URL, requestHost string) bool {
	if requestHost == "" {
		return false
	}
	return strings.EqualFold(
		normalizeHost(origin.Host, origin.Scheme),
		normalizeHost(requestHost, origin.Scheme),
	)
}

func normalizeOriginHost(origin *url.URL) string {
	return normalizeHost(origin.Host, origin.Scheme)
}

func normalizeHost(host, scheme string) string {
	if hostname, port, err := splitHostPort(host); err == nil {
		if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
			return hostname
		}
		return hostname + ":" + port
	}
	return host
}

func splitHostPort(host string) (string, string, error) {
	// net.SplitHostPort falla cuando el host no declara puerto; en ese caso se
	// usa el host completo y la comparación sigue siendo exacta.
	parsed, err := url.Parse("//" + host)
	if err != nil || parsed.Hostname() == "" {
		return "", "", errWebSocketUnauthorized
	}
	return parsed.Hostname(), parsed.Port(), nil
}

func isLoopbackDevelopmentOrigin(origin *url.URL) bool {
	hostname := strings.ToLower(origin.Hostname())
	if hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
		return false
	}
	switch origin.Port() {
	case "", "3000", "4200":
		return true
	default:
		return false
	}
}
