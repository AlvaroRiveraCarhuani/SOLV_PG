package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_GlobalIPLimit(t *testing.T) {
	rl := NewRateLimiter()
	handler := GlobalRateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Enviar peticiones dentro del burst (burst = 20)
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.RemoteAddr = "192.168.1.10:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200 en iteracion %d, obtenido %d", i, rr.Code)
		}
	}

	// La peticion 21 debe ser rechazada con 429 Too Many Requests
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("se esperaba status 429 cuando se excede el burst, obtenido %d", rr.Code)
	}

	retryAfter := rr.Header().Get("Retry-After")
	if retryAfter != "60" {
		t.Errorf("se esperaba header Retry-After: 60, obtenido %s", retryAfter)
	}
}

func TestRateLimiter_AuthFailureBlocking(t *testing.T) {
	rl := NewRateLimiter()
	ip := "10.0.0.5"

	// Registrar 4 fallos (aún no bloqueado)
	for i := 0; i < 4; i++ {
		blocked := rl.RecordAuthFailure(ip)
		if blocked {
			t.Fatalf("ip no deberia estar bloqueada en intento %d", i+1)
		}
	}

	if rl.IsAuthBlocked(ip) {
		t.Fatal("ip no deberia estar bloqueada con 4 fallos")
	}

	// Registrar el quinto fallo
	blocked := rl.RecordAuthFailure(ip)
	if !blocked {
		t.Fatal("quinto fallo deberia retornar bloqueado")
	}

	if !rl.IsAuthBlocked(ip) {
		t.Fatal("ip deberia estar bloqueada despues de 5 fallos")
	}

	// Verificar respuesta del middleware cuando la IP está bloqueada por auth
	handler := GlobalRateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/public", nil)
	req.RemoteAddr = ip + ":54321"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("se esperaba status 429 para IP bloqueada por auth, obtenido %d", rr.Code)
	}

	if rr.Header().Get("Retry-After") != "900" {
		t.Errorf("se esperaba Retry-After: 900, obtenido %s", rr.Header().Get("Retry-After"))
	}
}

func TestRateLimiter_SubmissionRateLimit(t *testing.T) {
	rl := NewRateLimiter()
	handler := SubmissionRateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	studentID := "student-test-123"

	// Burst = 5
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/submissions", nil)
		req.Header.Set("X-User-Id", studentID)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("se esperaba status 200 en submission %d, obtenido %d", i, rr.Code)
		}
	}

	// La peticion 6 debe fallar con 429
	req := httptest.NewRequest("POST", "/api/v1/submissions", nil)
	req.Header.Set("X-User-Id", studentID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("se esperaba status 429 al exceder limite de envios, obtenido %d", rr.Code)
	}
}
