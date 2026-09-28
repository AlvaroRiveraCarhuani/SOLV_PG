package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"solv-backend/internal/core/domain"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/storage/postgres"

	"github.com/google/uuid"
)

// TestSlice147_AuditActorTimeline verifica el submódulo 14.7:
//  1. El timeline por actor devuelve solo los eventos del actor, en orden desc.
//  2. El email institucional se resuelve vía JOIN a users.
//  3. Actores sin fila en users conservan trazabilidad (fallback actor_id).
//  4. El endpoint HTTP del timeline aplica el filtro por tenant del contexto.
func TestSlice147_AuditActorTimeline(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	tenantID := "00000000-0000-0000-0000-000000000001"
	ctx := context.Background()

	// Actor con fila de usuario (email resuelto) y actor huérfano (fallback).
	actorWithUser := uuid.NewString()
	actorOrphan := uuid.NewString()
	email := fmt.Sprintf("auditoria_f1_%s@uab.edu.bo", actorWithUser[:8])

	_, err = db.GetDB().Exec(`
		INSERT INTO users (id, email, role, tenant_id)
		VALUES ($1, $2, 'teacher', $3)
		ON CONFLICT (id) DO NOTHING
	`, actorWithUser, email, tenantID)
	if err != nil {
		t.Fatalf("Failed to seed actor user: %v", err)
	}

	auditRepo := postgres.NewAuditLogRepository(db.GetDB())

	for i := 0; i < 3; i++ {
		err := auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      actorWithUser,
			Action:       "SUBJECT_CREATED",
			ResourceType: "subject",
			StatusCode:   http.StatusCreated,
			Metadata:     []byte(fmt.Sprintf(`{"seq": %d}`, i)),
		})
		if err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		err := auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      actorOrphan,
			Action:       "WORKSPACE_HIBERNATED",
			ResourceType: "workspace_instance",
			StatusCode:   http.StatusOK,
		})
		if err != nil {
			t.Fatalf("Failed to create orphan audit log: %v", err)
		}
	}

	// --- 1. Timeline del actor con usuario: 3 eventos, email resuelto ---
	timeline, err := auditRepo.ListByActorTimeline(ctx, tenantID, actorWithUser, 200)
	if err != nil {
		t.Fatalf("ListByActorTimeline failed: %v", err)
	}
	if len(timeline) != 3 {
		t.Fatalf("Expected 3 timeline events for actor with user, got %d", len(timeline))
	}
	for _, log := range timeline {
		if log.ActorEmail != email {
			t.Errorf("Expected resolved email %q, got %q", email, log.ActorEmail)
		}
		if log.Action != "SUBJECT_CREATED" {
			t.Errorf("Unexpected action in timeline: %s", log.Action)
		}
	}
	for i := 1; i < len(timeline); i++ {
		if timeline[i-1].CreatedAt.Before(timeline[i].CreatedAt) {
			t.Errorf("Timeline not in descending order at index %d", i)
		}
	}

	// --- 2. Timeline del actor huérfano: fallback actor_id como email ---
	orphanTimeline, err := auditRepo.ListByActorTimeline(ctx, tenantID, actorOrphan, 200)
	if err != nil {
		t.Fatalf("ListByActorTimeline (orphan) failed: %v", err)
	}
	if len(orphanTimeline) != 2 {
		t.Fatalf("Expected 2 timeline events for orphan actor, got %d", len(orphanTimeline))
	}
	for _, log := range orphanTimeline {
		if log.ActorEmail != actorOrphan {
			t.Errorf("Expected fallback actor_id as email, got %q", log.ActorEmail)
		}
	}

	// --- 3. Endpoint HTTP del timeline con middleware de tenant ---
	adminHandler := httpdelivery.NewAdminHandler(auditRepo, nil, nil, nil, nil)
	mux := http.NewServeMux()
	handlers := httpdelivery.Handlers{
		AdminHandler: adminHandler,
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tenantCtx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
				next.ServeHTTP(w, r.WithContext(tenantCtx))
			})
		},
	}
	httpdelivery.SetupRoutes(mux, &handlers)
	server := httptest.NewServer(mux)
	defer server.Close()

	timelineURL := fmt.Sprintf("%s/api/v1/admin/audit-logs/actors/%s/timeline", server.URL, actorWithUser)
	resp, err := http.Get(timelineURL)
	if err != nil {
		t.Fatalf("Failed to call timeline endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK from timeline endpoint, got %d", resp.StatusCode)
	}

	var respBody struct {
		Count int               `json:"count"`
		Data  []domain.AuditLog `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("Failed to decode timeline response: %v", err)
	}
	if respBody.Count != 3 {
		t.Errorf("Expected count=3 in timeline endpoint, got %d", respBody.Count)
	}
	if len(respBody.Data) != 3 {
		t.Errorf("Expected 3 rows in timeline endpoint, got %d", len(respBody.Data))
	}
	for _, log := range respBody.Data {
		if log.ActorEmail != email {
			t.Errorf("Expected resolved email in endpoint response, got %q", log.ActorEmail)
		}
	}
}

func TestSlice147_AuditUnifiedSearchAndCount(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
	}

	tenantID := "00000000-0000-0000-0000-000000000001"
	ctx := context.Background()
	actorID := uuid.NewString()
	email := fmt.Sprintf("audit_search_%s@uab.edu.bo", actorID[:8])
	resourceID := uuid.NewString()
	actionMarker := "search-action-" + actorID[:8]
	resourceTypeMarker := "search-resource-type-" + actorID[:8]

	_, err = db.GetDB().Exec(`
		INSERT INTO users (id, email, role, tenant_id)
		VALUES ($1, $2, 'teacher', $3)
	`, actorID, email, tenantID)
	if err != nil {
		t.Fatalf("Failed to seed actor user: %v", err)
	}

	repo := postgres.NewAuditLogRepository(db.GetDB())
	logs := []domain.AuditLog{
		{TenantID: tenantID, ActorID: actorID, Action: "POST /api/v1/" + actionMarker, ResourceType: resourceTypeMarker, ResourceID: &resourceID, StatusCode: http.StatusCreated},
		{TenantID: tenantID, ActorID: uuid.NewString(), Action: "DELETE /api/v1/other", ResourceType: "other", StatusCode: http.StatusOK},
	}
	for i := range logs {
		if err := repo.Create(ctx, &logs[i]); err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}
	}

	searchTerms := []string{email, actorID, actionMarker, resourceTypeMarker, resourceID}
	for _, search := range searchTerms {
		t.Run(search, func(t *testing.T) {
			results, err := repo.ListFiltered(ctx, tenantID, search, "", 1, 0)
			if err != nil {
				t.Fatalf("ListFiltered failed for %q: %v", search, err)
			}
			total, err := repo.CountFiltered(ctx, tenantID, search, "")
			if err != nil {
				t.Fatalf("CountFiltered failed for %q: %v", search, err)
			}
			if total != 1 || len(results) != 1 {
				t.Fatalf("Expected one matching result and count for %q, got results=%d total=%d", search, len(results), total)
			}
			if results[0].ActorID != actorID {
				t.Errorf("Search %q returned unexpected actor %q", search, results[0].ActorID)
			}
		})
	}

	adminHandler := httpdelivery.NewAdminHandler(repo, nil, nil, nil, nil)
	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &httpdelivery.Handlers{
		AdminHandler: adminHandler,
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tenantCtx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
				next.ServeHTTP(w, r.WithContext(tenantCtx))
			})
		},
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	for _, query := range []url.Values{
		{"search": []string{resourceTypeMarker}},
		{"actor_id": []string{email}},
	} {
		requestURL := fmt.Sprintf("%s/api/v1/admin/audit-logs?%s", server.URL, query.Encode())
		response, err := http.Get(requestURL)
		if err != nil {
			t.Fatalf("Failed to call audit log endpoint: %v", err)
		}
		var body struct {
			Total int               `json:"total"`
			Data  []domain.AuditLog `json:"data"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK from audit log endpoint, got %d", response.StatusCode)
		}
		if decodeErr != nil {
			t.Fatalf("Failed to decode audit log response: %v", decodeErr)
		}
		if body.Total != 1 || len(body.Data) != 1 || body.Data[0].ActorID != actorID {
			t.Errorf("Unexpected endpoint results for %v: total=%d rows=%d", query, body.Total, len(body.Data))
		}
	}
}
