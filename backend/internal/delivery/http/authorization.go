package httpdelivery

import "net/http"

// adminOnly exige el rol admin antes de invocar un handler administrativo.
// El tenant y la identidad ya deben estar en el contexto (WithTenant).
func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requireAdminRole(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// staffOnly permite las operaciones compartidas por docentes y administradores.
// Se usa solo donde el docente tiene una capacidad legítima; las operaciones
// exclusivamente administrativas usan adminOnly.
func staffOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requireRole(w, r, "teacher", "admin") {
			return
		}
		next.ServeHTTP(w, r)
	})
}
