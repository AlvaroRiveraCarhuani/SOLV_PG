package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
	"github.com/golang-jwt/jwt/v5"
)

func TestSlice11OperabilityB2B(t *testing.T) {
	t.Run("1. Rate Limiting on Workspaces Start (5 req/min)", func(t *testing.T) {
		limiter := middleware.NewUserRateLimiter(5, 5)
		dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"started"}`))
		})
		rateLimitedHandler := middleware.RateLimitMiddleware(limiter)(dummyHandler)

		userID := "user-slice11-rate-limit-test"

		// Enviar 5 requests permitidos
		for i := 1; i <= 5; i++ {
			req := httptest.NewRequest("POST", "/api/v1/workspaces/start", nil)
			ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			rateLimitedHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("Request %d expected HTTP 200 OK, got %d", i, rr.Code)
			}

			if limitHeader := rr.Header().Get("X-RateLimit-Limit"); limitHeader != "5" {
				t.Errorf("Request %d missing or invalid X-RateLimit-Limit header: %s", i, limitHeader)
			}
			if remHeader := rr.Header().Get("X-RateLimit-Remaining"); remHeader == "" {
				t.Errorf("Request %d missing X-RateLimit-Remaining header", i)
			}
		}

		// El 6to request DEBE ser rechazado con 429
		req6 := httptest.NewRequest("POST", "/api/v1/workspaces/start", nil)
		ctx6 := context.WithValue(req6.Context(), domain.UserIDKey, userID)
		req6 = req6.WithContext(ctx6)

		rr6 := httptest.NewRecorder()
		rateLimitedHandler.ServeHTTP(rr6, req6)

		if rr6.Code != http.StatusTooManyRequests {
			t.Fatalf("6th request expected HTTP 429 Too Many Requests, got %d", rr6.Code)
		}

		if retryAfter := rr6.Header().Get("Retry-After"); retryAfter != "60" {
			t.Errorf("Expected Retry-After header '60', got '%s'", retryAfter)
		}

		var respErr map[string]string
		if err := json.Unmarshal(rr6.Body.Bytes(), &respErr); err != nil || respErr["error"] == "" {
			t.Errorf("Expected rate limit error payload in JSON, got: %s", rr6.Body.String())
		}

		t.Logf("PASS: Rate Limiter correctly enforced 429 on 6th request with RFC 6585 headers!")
	})

	t.Run("2. Audit Log Async Worker Pool for Teacher/Admin", func(t *testing.T) {
		db, err := setupTestDB()
		if err != nil {
			t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
		}

		auditRepo := postgres.NewAuditLogRepository(db.GetDB())
		auditPool := middleware.NewAuditWorkerPool(auditRepo, 1000, 5)
		defer auditPool.Shutdown()

		// Handler simple para crear materia
		subjectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"subj-123","name":"Sistemas Operativos"}`))
		})

		auditedHandler := middleware.AuditMiddleware(auditPool)(subjectHandler)

		teacherID := "11111111-1111-1111-1111-111111111111"
		tenantID := domain.DefaultTenantID

		reqBody := bytes.NewBufferString(`{"name":"Sistemas Operativos","code":"SO-2026"}`)
		req := httptest.NewRequest("POST", "/api/v1/subjects", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "SOLV-IntegrationTest/1.0")

		ctx := context.WithValue(req.Context(), domain.TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, domain.UserIDKey, teacherID)
		ctx = context.WithValue(ctx, domain.UserRoleKey, "teacher")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		auditedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("Expected HTTP 201 Created, got %d", rr.Code)
		}

		// Dar 200ms al worker pool para procesar el evento asíncrono
		time.Sleep(200 * time.Millisecond)

		logs, err := auditRepo.ListByTenant(context.Background(), tenantID, 10)
		if err != nil {
			t.Fatalf("Failed to query audit logs from DB: %v", err)
		}

		found := false
		for _, l := range logs {
			if l.ActorID == teacherID && l.Action == "POST /api/v1/subjects" {
				found = true
				if l.StatusCode != http.StatusCreated {
					t.Errorf("Expected status_code %d in audit log, got %d", http.StatusCreated, l.StatusCode)
				}
				if l.ResourceType != "subjects" {
					t.Errorf("Expected resource_type 'subjects', got '%s'", l.ResourceType)
				}
				break
			}
		}

		if !found {
			t.Fatalf("Audit log entry for action 'POST /api/v1/subjects' was not persisted by worker pool!")
		}

		t.Logf("PASS: Teacher action correctly audited and persisted in DB via AuditWorkerPool with status_code!")
	})

	t.Run("3. Concurrent Migration Advisory Lock (pg_advisory_lock)", func(t *testing.T) {
		db, err := setupTestDB()
		if err != nil {
			t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 2)

		// Lanzar 2 goroutines simultáneas ejecutando RunInitialMigrations()
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if err := db.RunInitialMigrations(); err != nil {
					errChan <- err
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		for mErr := range errChan {
			t.Fatalf("Concurrent migration failed with error: %v", mErr)
		}

		t.Logf("PASS: Concurrent RunInitialMigrations executed cleanly with pg_advisory_lock(1337)!")
	})
}

func generateTestJWT(userID, tenantID, role string) string {
	claims := jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"role":      role,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("una_cadena_aleatoria_y_muy_larga_para_solv"))
	return tokenStr
}

func setupTestDB() (*database.Database, error) {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbDSN = "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	}
	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		return nil, err
	}
	if err := db.RunInitialMigrations(); err != nil {
		return nil, err
	}
	return db, nil
}
