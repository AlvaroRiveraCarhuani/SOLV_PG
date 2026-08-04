package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type MockUserRepository struct{}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*domainUserPlaceholder, error) {
	return nil, nil
}
func (m *MockUserRepository) CreateUserFromSSO(ctx context.Context, user interface{}) (string, error) {
	return "test-user-id", nil
}

type domainUserPlaceholder struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Role      string
}

func setupForwardAuthTestServer(t *testing.T, jwtSecret string) (*http.ServeMux, *services.AuthService) {
	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/auth/callback")
	os.Setenv("JWT_SECRET", jwtSecret)
	os.Setenv("COOKIE_DOMAIN", ".solv.uab.edu.bo")

	authService := services.NewAuthService(nil, nil)
	authHandler := httpdelivery.NewAuthHandler(authService)

	mux := http.NewServeMux()
	handlers := &httpdelivery.Handlers{
		AuthHandler: authHandler,
	}
	httpdelivery.SetupRoutes(mux, handlers)

	return mux, authService
}

func createTestJWT(secret string, expiration time.Duration) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test-student-uuid",
		"email":   "estudiante@uab.edu.bo",
		"role":    "student",
		"exp":     time.Now().Add(expiration).Unix(),
	})
	str, _ := token.SignedString([]byte(secret))
	return str
}

func TestCRIT05ForwardAuthVerification(t *testing.T) {
	secret := "test-super-secret-key-1234567890"
	mux, _ := setupForwardAuthTestServer(t, secret)

	validToken := createTestJWT(secret, 1*time.Hour)
	expiredToken := createTestJWT(secret, -1*time.Hour)
	wrongSecretToken := createTestJWT("wrong-signature-key-9876543210", 1*time.Hour)

	tests := []struct {
		name           string
		cookieValue    string
		expectedStatus int
	}{
		{
			name:           "1. Cookie válida -> Retorna 200 OK",
			cookieValue:    validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "2. Sin cookie -> Retorna 401 Unauthorized",
			cookieValue:    "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "3. Cookie con JWT expirado -> Retorna 403 Forbidden",
			cookieValue:    expiredToken,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "4. Cookie con firma inválida -> Retorna 401 Unauthorized",
			cookieValue:    wrongSecretToken,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/auth/verify", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "solv_session",
					Value: tt.cookieValue,
				})
			}

			rec := httptest.NewRecorder()
			start := time.Now()

			mux.ServeHTTP(rec, req)

			duration := time.Since(start)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			// Performance crítica: respuesta en <50ms
			if duration > 50*time.Millisecond {
				t.Errorf("Performance SLA breached! Request took %v (limit is <50ms)", duration)
			} else {
				t.Logf("PASS: Latency: %v (<50ms SLA met)", duration)
			}
		})
	}
}

func TestCRIT05LogoutClearsSessionCookie(t *testing.T) {
	mux, _ := setupForwardAuthTestServer(t, "test-super-secret-key-1234567890")

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on logout, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	var logoutCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "solv_session" {
			logoutCookie = c
			break
		}
	}

	if logoutCookie == nil {
		t.Fatalf("Assertion failed: solv_session cookie not found in Set-Cookie header on logout")
	}

	if logoutCookie.MaxAge != -1 {
		t.Errorf("Expected Cookie MaxAge -1, got %d", logoutCookie.MaxAge)
	}
	expectedDomain := ".solv.uab.edu.bo"
	if logoutCookie.Domain != expectedDomain && logoutCookie.Domain != "solv.uab.edu.bo" {
		t.Errorf("Expected Cookie Domain '%s', got '%s'", expectedDomain, logoutCookie.Domain)
	}
}
