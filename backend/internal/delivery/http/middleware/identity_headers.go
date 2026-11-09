package middleware

import "net/http"

// identityHeaders son cabeceras de propagación interna. El backend las fija
// únicamente después de verificar un JWT; cualquier valor que llegue desde el
// cliente se elimina antes de entrar al mux para que no pueda sustituir la
// identidad, el tenant ni el rol.
var identityHeaders = []string{
	"X-User-Id",
	"X-User-Role",
	"X-Tenant-Id",
	"X-User-Email",
	"X-Auth-Request-User",
	"X-Auth-Request-Groups",
	"X-Forwarded-User",
}

// SanitizeIdentityHeaders elimina las cabeceras de identidad del request de
// entrada. Debe envolver el handler raíz, por fuera de la autenticación, para
// que los middleware internos puedan volver a fijarlas desde los claims.
func SanitizeIdentityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range identityHeaders {
			r.Header.Del(header)
		}
		next.ServeHTTP(w, r)
	})
}
