package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupDraftTestServer(t *testing.T) (*httptest.Server, *database.Database) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil
	}

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())

	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)

	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = "00000000-0000-0000-0000-000000000001"
			}
			userID := r.Header.Get("X-User-Id")
			if userID == "" {
				userID = "00000000-0000-0000-0000-000000000001"
			}
			role := r.Header.Get("X-User-Role")
			if role == "" {
				role = "admin"
			}
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, domain.UserIDKey, userID)
			ctx = context.WithValue(ctx, domain.UserRoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handlers := httpdelivery.Handlers{
		AdminAcademicHandler: adminAcademicHandler,
		TenantMiddleware:     tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

func TestTemplateDraftLifecycleAndIsolation(t *testing.T) {
	server, db := setupDraftTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	userA := "11111111-1111-1111-1111-111111111111"
	userB := "22222222-2222-2222-2222-222222222222"
	tenantID := "00000000-0000-0000-0000-000000000001"

	_, errTenant := db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) 
		VALUES ($1, 'Tenant Test Draft', 'test-draft')
		ON CONFLICT (id) DO NOTHING;
	`, tenantID)
	if errTenant != nil {
		t.Fatalf("failed to insert tenant: %v", errTenant)
	}

	_, errUser := db.GetDB().Exec(`
		INSERT INTO users (id, tenant_id, email, name, role)
		VALUES 
			($1, $3, 'usera_draft@uab.edu.bo', 'User A Draft', 'admin'),
			($2, $3, 'userb_draft@uab.edu.bo', 'User B Draft', 'admin')
		ON CONFLICT (id) DO NOTHING;
	`, userA, userB, tenantID)
	if errUser != nil {
		t.Fatalf("failed to insert users: %v", errUser)
	}

	// Limpieza previa de borradores de prueba
	_, _ = db.GetDB().Exec(`DELETE FROM template_drafts WHERE tenant_id = $1 AND user_id IN ($2, $3)`, tenantID, userA, userB)

	// 1. GET sin borrador retorna 404
	t.Run("1. GET sin borrador retorna 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/templates/drafts", nil)
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", userA)
		req.Header.Set("X-User-Role", "admin")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed GET /drafts: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for non-existent draft, got %d", resp.StatusCode)
		}
	})

	// 2. POST con form_data y GET devuelve form_data idéntico
	t.Run("2. POST y GET devuelve form_data idéntico", func(t *testing.T) {
		payload := map[string]interface{}{
			"form_data": map[string]interface{}{
				"name":               "Draft Plantilla Python",
				"docker_image":       "python:3.12-slim",
				"target_environment": "ide",
				"base_ram_mb":        1024,
				"description":        "Borrador en progreso para laboratorio Python",
			},
		}
		bodyBytes, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/admin/templates/drafts", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", userA)
		req.Header.Set("X-User-Role", "admin")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed POST /drafts: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 on SaveDraft, got %d: %s", resp.StatusCode, string(body))
		}

		// Leer borrador con GET
		getReq, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/templates/drafts", nil)
		getReq.Header.Set("X-Tenant-Id", tenantID)
		getReq.Header.Set("X-User-Id", userA)
		getReq.Header.Set("X-User-Role", "admin")

		getResp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			t.Fatalf("failed GET /drafts: %v", err)
		}
		defer getResp.Body.Close()

		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on GetDraftByUser, got %d", getResp.StatusCode)
		}

		var apiResponse struct {
			Data struct {
				ID        string                 `json:"id"`
				UserID    string                 `json:"user_id"`
				FormData  map[string]interface{} `json:"form_data"`
				UpdatedAt string                 `json:"updated_at"`
			} `json:"data"`
		}

		if err := json.NewDecoder(getResp.Body).Decode(&apiResponse); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if apiResponse.Data.UserID != userA {
			t.Fatalf("expected user_id %s, got %s", userA, apiResponse.Data.UserID)
		}
		if apiResponse.Data.FormData["name"] != "Draft Plantilla Python" {
			t.Fatalf("expected name 'Draft Plantilla Python', got %v", apiResponse.Data.FormData["name"])
		}
		if apiResponse.Data.FormData["docker_image"] != "python:3.12-slim" {
			t.Fatalf("expected docker_image 'python:3.12-slim', got %v", apiResponse.Data.FormData["docker_image"])
		}
	})

	// 3. Usuario B no puede leer borrador de Usuario A (aislamiento por user_id)
	t.Run("3. Usuario B no puede leer borrador de Usuario A", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/templates/drafts", nil)
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", userB)
		req.Header.Set("X-User-Role", "admin")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed GET /drafts as User B: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for User B, got %d", resp.StatusCode)
		}
	})

	// 4. Borrar borrador y volver a leer retorna 404
	t.Run("4. DELETE borrador y posterior GET retorna 404", func(t *testing.T) {
		delReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/v1/admin/templates/drafts", nil)
		delReq.Header.Set("X-Tenant-Id", tenantID)
		delReq.Header.Set("X-User-Id", userA)
		delReq.Header.Set("X-User-Role", "admin")

		delResp, err := http.DefaultClient.Do(delReq)
		if err != nil {
			t.Fatalf("failed DELETE /drafts: %v", err)
		}
		defer delResp.Body.Close()

		if delResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on DELETE /drafts, got %d", delResp.StatusCode)
		}

		// Verificar que ya no existe
		getReq, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/templates/drafts", nil)
		getReq.Header.Set("X-Tenant-Id", tenantID)
		getReq.Header.Set("X-User-Id", userA)
		getReq.Header.Set("X-User-Role", "admin")

		getResp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			t.Fatalf("failed GET /drafts after delete: %v", err)
		}
		defer getResp.Body.Close()

		if getResp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", getResp.StatusCode)
		}
	})

	// 5. POST con form_data vacío guarda {} sin error
	t.Run("5. POST con form_data vacío guarda {} sin error", func(t *testing.T) {
		payload := map[string]interface{}{}
		bodyBytes, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/admin/templates/drafts", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", userB)
		req.Header.Set("X-User-Role", "admin")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed POST /drafts with empty body: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 for empty form_data, got %d: %s", resp.StatusCode, string(body))
		}

		// Leer borrador de User B
		getReq, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/templates/drafts", nil)
		getReq.Header.Set("X-Tenant-Id", tenantID)
		getReq.Header.Set("X-User-Id", userB)
		getReq.Header.Set("X-User-Role", "admin")

		getResp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			t.Fatalf("failed GET /drafts for User B: %v", err)
		}
		defer getResp.Body.Close()

		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on GetDraftByUser for User B, got %d", getResp.StatusCode)
		}

		var apiResponse struct {
			Data struct {
				FormData map[string]interface{} `json:"form_data"`
			} `json:"data"`
		}
		if err := json.NewDecoder(getResp.Body).Decode(&apiResponse); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(apiResponse.Data.FormData) != 0 {
			t.Fatalf("expected empty form_data {}, got %v", apiResponse.Data.FormData)
		}
	})
}
