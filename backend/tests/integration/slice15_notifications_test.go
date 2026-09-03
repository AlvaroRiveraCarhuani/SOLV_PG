package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupSlice15NotificationsServer(t *testing.T) (*httptest.Server, *database.Database, *services.NotificationService) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil, nil
	}

	_ = db.RunInitialMigrations()

	notificationRepo := postgres.NewPostgresNotificationRepository(db.GetDB())
	notificationService := services.NewNotificationService(notificationRepo, 256)
	notificationHandler := httpdelivery.NewNotificationHandler(notificationService)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = "00000000-0000-0000-0000-000000000001"
			}
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handlers := httpdelivery.Handlers{
		NotificationHandler: notificationHandler,
		TenantMiddleware:    tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db, notificationService
}

func TestSlice15_Notifications_CompleteSuite(t *testing.T) {
	server, db, notifService := setupSlice15NotificationsServer(t)
	if server == nil {
		return
	}
	defer server.Close()
	defer notifService.Stop()

	tenantID := "00000000-0000-0000-0000-000000000001"
	userA_ID := uuid.NewString()
	userB_ID := uuid.NewString()
	client := &http.Client{}

	// 1. Seed Usuarios
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES 
			($1, 'Carlos', 'NotifA', $3, 'student', $5),
			($2, 'Ana', 'NotifB', $4, 'student', $5)
		ON CONFLICT (id) DO NOTHING;
	`, userA_ID, userB_ID,
		fmt.Sprintf("carlos_%s@uab.edu.bo", userA_ID[:6]),
		fmt.Sprintf("ana_%s@uab.edu.bo", userB_ID[:6]),
		tenantID)

	// 2. Sembrar 5 notificaciones para Usuario A
	var notifIDs []string
	for i := 1; i <= 5; i++ {
		severity := domain.NotificationSeverityInfo
		if i == 4 {
			severity = domain.NotificationSeverityWarning
		} else if i == 5 {
			severity = domain.NotificationSeverityCritical
		}

		n, err := notifService.Notify(context.Background(), tenantID, domain.CreateNotificationDTO{
			RecipientUserID: userA_ID,
			Title:           fmt.Sprintf("Notificación %d", i),
			Message:         fmt.Sprintf("Mensaje descriptivo del evento %d", i),
			Severity:        severity,
			EventType:       "grade_published",
			Link:            fmt.Sprintf("/student/labs/%d", i),
		})
		if err != nil {
			t.Fatalf("Failed seeding notification %d: %v", i, err)
		}
		notifIDs = append(notifIDs, n.ID)
		time.Sleep(10 * time.Millisecond) // Asegurar orden temporal estricto
	}

	// =========================================================================
	// 1. TEST Paginación y Orden Temporal (GET /api/v1/notifications)
	// =========================================================================
	t.Run("1. Paginación y Orden Temporal - Consulta Paginada y Metadatos", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/notifications?page=1&limit=2", nil)
		req.Header.Set("X-User-Id", userA_ID)
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed list notifications request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errBody)
			t.Fatalf("Expected 200 OK, got %d. Body: %v", resp.StatusCode, errBody)
		}

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)

		items := body["data"].([]interface{})
		if len(items) != 2 {
			t.Errorf("Expected 2 items in page 1, got %d", len(items))
		}

		if resp.Header.Get("X-Total-Count") != "5" {
			t.Errorf("Expected X-Total-Count = 5, got %s", resp.Header.Get("X-Total-Count"))
		}

		// La primera notificación debe ser la más reciente (Notificación 5 - Critical)
		first := items[0].(map[string]interface{})
		if first["title"] != "Notificación 5" {
			t.Errorf("Expected first item to be 'Notificación 5', got %v", first["title"])
		}
		if first["severity"] != domain.NotificationSeverityCritical {
			t.Errorf("Expected severity = critical, got %v", first["severity"])
		}
	})

	// =========================================================================
	// 2. TEST Contador de la Campana (GET /api/v1/notifications/unread-count)
	// =========================================================================
	t.Run("2. Contador Ultraliviano de No Leídas para Campana UI", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/notifications/unread-count", nil)
		req.Header.Set("X-User-Id", userA_ID)
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed unread count request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		data := body["data"].(map[string]interface{})
		if int(data["unread_count"].(float64)) != 5 {
			t.Errorf("Expected unread_count = 5, got %v", data["unread_count"])
		}
	})

	// =========================================================================
	// 3. TEST Lectura Individual y Seguridad Cruzada (PATCH /notifications/{id}/read)
	// =========================================================================
	t.Run("3. Lectura Individual - Timestamp de Lectura y Aislamiento Cruzado", func(t *testing.T) {
		targetID := notifIDs[4] // Notificación 5

		// 3.1 Usuario B intenta marcar como leída la notificación de Usuario A -> 404 Not Found
		reqCross, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/api/v1/notifications/%s/read", server.URL, targetID), nil)
		reqCross.Header.Set("X-User-Id", userB_ID) // Usuario incorrecto
		reqCross.Header.Set("X-Tenant-Id", tenantID)

		respCross, _ := client.Do(reqCross)
		if respCross.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found for cross-user notification read, got %d", respCross.StatusCode)
		}

		// 3.2 Usuario A marca su propia notificación -> 200 OK
		reqValid, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/api/v1/notifications/%s/read", server.URL, targetID), nil)
		reqValid.Header.Set("X-User-Id", userA_ID)
		reqValid.Header.Set("X-Tenant-Id", tenantID)

		respValid, err := client.Do(reqValid)
		if err != nil {
			t.Fatalf("Failed mark read request: %v", err)
		}
		if respValid.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK marking read, got %d", respValid.StatusCode)
		}

		// 3.3 Verificar en BD que is_read = true y read_at tiene fecha
		var isRead bool
		var readAt *time.Time
		_ = db.GetDB().QueryRow("SELECT is_read, read_at FROM notifications WHERE id = $1", targetID).Scan(&isRead, &readAt)
		if !isRead {
			t.Errorf("Expected is_read = true in DB")
		}
		if readAt == nil {
			t.Errorf("Expected read_at timestamp in DB")
		}

		// 3.4 Verificar que el contador de no leídas bajó a 4
		count, _ := notifService.GetUnreadCount(context.Background(), tenantID, userA_ID)
		if count.UnreadCount != 4 {
			t.Errorf("Expected unread count = 4 after reading one, got %d", count.UnreadCount)
		}
	})

	// =========================================================================
	// 4. TEST Lectura Masiva (POST /api/v1/notifications/mark-all-read)
	// =========================================================================
	t.Run("4. Lectura Masiva - Marcar Todas como Leídas en Una Sola Operación", func(t *testing.T) {
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/notifications/mark-all-read", nil)
		req.Header.Set("X-User-Id", userA_ID)
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed mark all read request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK marking all read, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		data := body["data"].(map[string]interface{})
		if int(data["marked_count"].(float64)) != 4 {
			t.Errorf("Expected 4 notifications marked as read, got %v", data["marked_count"])
		}

		// Comprobar que el contador ahora es 0
		count, _ := notifService.GetUnreadCount(context.Background(), tenantID, userA_ID)
		if count.UnreadCount != 0 {
			t.Errorf("Expected unread count = 0 after mark-all-read, got %d", count.UnreadCount)
		}
	})

	// =========================================================================
	// 5. TEST Despacho Asíncrono no Bloqueante (NotifyAsync)
	// =========================================================================
	t.Run("5. Despacho Asíncrono - Worker Pool Go Channel", func(t *testing.T) {
		asyncUser := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Async', 'Worker', $2, 'student', $3)
			ON CONFLICT (id) DO NOTHING;
		`, asyncUser, fmt.Sprintf("async_%s@uab.edu.bo", asyncUser[:6]), tenantID)

		notifService.NotifyAsync(tenantID, domain.CreateNotificationDTO{
			RecipientUserID: asyncUser,
			Title:           "Alerta Asíncrona de Juez Virtual",
			Message:         "Tu entrega fue evaluada con veredicto ACCEPTED",
			Severity:        domain.NotificationSeverityInfo,
			EventType:       "judge_verdict",
		})

		// Esperar que el worker procese el canal
		time.Sleep(50 * time.Millisecond)

		count, err := notifService.GetUnreadCount(context.Background(), tenantID, asyncUser)
		if err != nil || count.UnreadCount != 1 {
			t.Errorf("Expected async notification to be saved by worker, count=%d, err=%v", count.UnreadCount, err)
		}
	})

	// =========================================================================
	// 6. TEST Inserción Masiva por Lotes (CreateBatch)
	// =========================================================================
	t.Run("6. Inserción Masiva por Lotes - Transacción Batch para Cursos", func(t *testing.T) {
		batchDTOs := make([]domain.CreateNotificationDTO, 10)
		for i := 0; i < 10; i++ {
			batchDTOs[i] = domain.CreateNotificationDTO{
				RecipientUserID: userB_ID,
				Title:           fmt.Sprintf("Aviso Masivo %d", i+1),
				Message:         "El docente extendió la fecha de entrega del laboratorio",
				Severity:        domain.NotificationSeverityInfo,
			}
		}

		createdList, err := notifService.NotifyBatch(context.Background(), tenantID, batchDTOs)
		if err != nil {
			t.Fatalf("Failed batch notification insert: %v", err)
		}
		if len(createdList) != 10 {
			t.Fatalf("Expected 10 batch items created, got %d", len(createdList))
		}

		count, _ := notifService.GetUnreadCount(context.Background(), tenantID, userB_ID)
		if count.UnreadCount != 10 {
			t.Errorf("Expected User B to have 10 unread notifications, got %d", count.UnreadCount)
		}
	})
}
