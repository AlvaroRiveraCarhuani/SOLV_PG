package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func setupSlice16BackupsServer(t *testing.T) (*httptest.Server, *database.Database, *services.BackupService, *services.NotificationService, string) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil, nil, nil, ""
	}

	_ = db.RunInitialMigrations()

	tmpBackupDir := fmt.Sprintf("/tmp/solv_test_backups_%d", time.Now().UnixNano())
	_ = os.MkdirAll(tmpBackupDir, 0750)

	notifRepo := postgres.NewPostgresNotificationRepository(db.GetDB())
	notifService := services.NewNotificationService(notifRepo, 256)

	backupRepo := postgres.NewPostgresBackupRepository(db.GetDB())
	backupService := services.NewBackupService(backupRepo, notifService, tmpBackupDir)
	backupHandler := httpdelivery.NewBackupHandler(backupService)
	notifHandler := httpdelivery.NewNotificationHandler(notifService)

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
		BackupHandler:       backupHandler,
		NotificationHandler: notifHandler,
		TenantMiddleware:    tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db, backupService, notifService, tmpBackupDir
}

func TestSlice16_Backups_CompleteSuite(t *testing.T) {
	server, db, _, notifService, tmpDir := setupSlice16BackupsServer(t)
	if server == nil {
		return
	}
	defer server.Close()
	defer notifService.Stop()
	defer os.RemoveAll(tmpDir)

	tenantID := uuid.NewString()
	adminID := uuid.NewString()
	studentID := uuid.NewString()
	client := &http.Client{}

	// Seed Tenant, Admin y Student
	_, err := db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug, allowed_domains)
		VALUES ($1, 'UAB Backups Tenant', $2, '["@uab.edu.bo"]'::jsonb)
		ON CONFLICT (id) DO NOTHING;
	`, tenantID, fmt.Sprintf("uab-backup-%s", tenantID[:8]))
	if err != nil {
		t.Fatalf("Failed seeding tenant: %v", err)
	}

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES 
			($1, 'Admin', 'Backup', $3, 'admin', $5),
			($2, 'Student', 'NoPerms', $4, 'student', $5)
		ON CONFLICT (id) DO NOTHING;
	`, adminID, studentID,
		fmt.Sprintf("admin_%s@uab.edu.bo", adminID[:6]),
		fmt.Sprintf("student_%s@uab.edu.bo", studentID[:6]),
		tenantID)

	var createdExecutionID string

	// =========================================================================
	// 1. TEST Configuración de Políticas de Respaldo (GET & PUT)
	// =========================================================================
	t.Run("1. Configuración de Políticas - Consulta y Modificación de Retención", func(t *testing.T) {
		// 1.1 GET Config inicial por defecto
		reqGet, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/backups/config", nil)
		reqGet.Header.Set("X-User-Id", adminID)
		reqGet.Header.Set("X-User-Role", "admin")
		reqGet.Header.Set("X-Tenant-Id", tenantID)

		respGet, err := client.Do(reqGet)
		if err != nil || respGet.StatusCode != http.StatusOK {
			t.Fatalf("Failed GET backup config: status=%d, err=%v", respGet.StatusCode, err)
		}

		var getBody map[string]interface{}
		json.NewDecoder(respGet.Body).Decode(&getBody)
		data := getBody["data"].(map[string]interface{})
		if int(data["local_frequency_hours"].(float64)) != 6 {
			t.Errorf("Expected default local_frequency_hours=6, got %v", data["local_frequency_hours"])
		}

		// 1.2 PUT Actualizar retención y parámetros
		remoteEnabled := true
		provider := "backblaze_b2"
		bucket := "solv-institutional-backups"
		updateDTO := domain.UpdateBackupConfigDTO{
			LocalFrequencyHours: 12,
			LocalRetentionDays:  14,
			RemoteEnabled:       &remoteEnabled,
			RemoteProvider:      &provider,
			RemoteBucketName:    &bucket,
		}
		raw, _ := json.Marshal(updateDTO)

		reqPut, _ := http.NewRequest("PUT", server.URL+"/api/v1/admin/backups/config", bytes.NewBuffer(raw))
		reqPut.Header.Set("X-User-Id", adminID)
		reqPut.Header.Set("X-User-Role", "admin")
		reqPut.Header.Set("X-Tenant-Id", tenantID)
		reqPut.Header.Set("Content-Type", "application/json")

		respPut, err := client.Do(reqPut)
		if err != nil || respPut.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			if respPut != nil {
				json.NewDecoder(respPut.Body).Decode(&errBody)
			}
			t.Fatalf("Failed PUT backup config: status=%d, err=%v, body=%v", respPut.StatusCode, err, errBody)
		}

		var putBody map[string]interface{}
		json.NewDecoder(respPut.Body).Decode(&putBody)
		putData := putBody["data"].(map[string]interface{})
		if int(putData["local_frequency_hours"].(float64)) != 12 {
			t.Errorf("Expected updated local_frequency_hours=12, got %v", putData["local_frequency_hours"])
		}
		if int(putData["local_retention_days"].(float64)) != 14 {
			t.Errorf("Expected updated local_retention_days=14, got %v", putData["local_retention_days"])
		}
		if putData["remote_bucket_name"] != "solv-institutional-backups" {
			t.Errorf("Expected remote_bucket_name='solv-institutional-backups', got %v", putData["remote_bucket_name"])
		}
	})

	// =========================================================================
	// 2. TEST Disparo Manual de Respaldo con Checksum SHA-256 (POST /trigger)
	// =========================================================================
	t.Run("2. Ejecución Manual de Respaldo - Generación Gzip y Checksum SHA-256", func(t *testing.T) {
		reqTrigger, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/backups/trigger", nil)
		reqTrigger.Header.Set("X-User-Id", adminID)
		reqTrigger.Header.Set("X-User-Role", "admin")
		reqTrigger.Header.Set("X-Tenant-Id", tenantID)

		respTrigger, err := client.Do(reqTrigger)
		if err != nil || respTrigger.StatusCode != http.StatusCreated {
			t.Fatalf("Failed trigger backup: status=%d, err=%v", respTrigger.StatusCode, err)
		}

		var body map[string]interface{}
		json.NewDecoder(respTrigger.Body).Decode(&body)
		data := body["data"].(map[string]interface{})

		createdExecutionID = data["id"].(string)
		if createdExecutionID == "" {
			t.Fatalf("Expected non-empty backup execution ID")
		}

		if data["status"] != domain.BackupStatusSuccess {
			t.Errorf("Expected status = 'success', got %v", data["status"])
		}

		checksum := data["sha256_checksum"].(string)
		if len(checksum) != 64 {
			t.Errorf("Expected 64-char hex SHA-256 checksum, got %v", checksum)
		}

		size := int64(data["file_size_bytes"].(float64))
		if size <= 0 {
			t.Errorf("Expected positive file_size_bytes, got %d", size)
		}
	})

	// =========================================================================
	// 3. TEST Integración con Notificaciones Proactivas (Slice 15 + Slice 16)
	// =========================================================================
	t.Run("3. Integración Proactiva - Alerta Automática in-app al Administrador", func(t *testing.T) {
		// Esperar que el worker de notificaciones procese el canal de forma resiliente
		var items []interface{}
		for attempt := 0; attempt < 10; attempt++ {
			time.Sleep(50 * time.Millisecond)
			reqNotif, _ := http.NewRequest("GET", server.URL+"/api/v1/notifications?limit=5", nil)
			reqNotif.Header.Set("X-User-Id", adminID)
			reqNotif.Header.Set("X-Tenant-Id", tenantID)

			respNotif, err := client.Do(reqNotif)
			if err == nil && respNotif.StatusCode == http.StatusOK {
				var body map[string]interface{}
				json.NewDecoder(respNotif.Body).Decode(&body)
				if rawItems, ok := body["data"].([]interface{}); ok && len(rawItems) > 0 {
					items = rawItems
					break
				}
			}
		}

		if len(items) == 0 {
			t.Fatalf("Expected at least 1 notification generated for admin upon backup completion")
		}

		first := items[0].(map[string]interface{})
		if first["event_type"] != "backup_created" {
			t.Errorf("Expected event_type='backup_created', got %v", first["event_type"])
		}
	})

	// =========================================================================
	// 4. TEST Historial de Respaldos (GET /api/v1/admin/backups)
	// =========================================================================
	t.Run("4. Historial de Respaldos - Listado Cronológico y Totalizador", func(t *testing.T) {
		reqList, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/backups?page=1&limit=10", nil)
		reqList.Header.Set("X-User-Id", adminID)
		reqList.Header.Set("X-User-Role", "admin")
		reqList.Header.Set("X-Tenant-Id", tenantID)

		respList, err := client.Do(reqList)
		if err != nil || respList.StatusCode != http.StatusOK {
			t.Fatalf("Failed listing backups: status=%d, err=%v", respList.StatusCode, err)
		}

		var body map[string]interface{}
		json.NewDecoder(respList.Body).Decode(&body)
		items := body["data"].([]interface{})
		if len(items) == 0 {
			t.Fatalf("Expected at least 1 backup in history")
		}

		if respList.Header.Get("X-Total-Count") == "" {
			t.Errorf("Expected X-Total-Count header in backups listing")
		}
	})

	// =========================================================================
	// 5. TEST Verificación Criptográfica de Integridad (POST /verify)
	// =========================================================================
	t.Run("5. Verificación Criptográfica - Recálculo de SHA-256 contra Archivo en Disco", func(t *testing.T) {
		reqVerify, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/backups/%s/verify", server.URL, createdExecutionID), nil)
		reqVerify.Header.Set("X-User-Id", adminID)
		reqVerify.Header.Set("X-User-Role", "admin")
		reqVerify.Header.Set("X-Tenant-Id", tenantID)

		respVerify, err := client.Do(reqVerify)
		if err != nil || respVerify.StatusCode != http.StatusOK {
			t.Fatalf("Failed verifying backup: status=%d, err=%v", respVerify.StatusCode, err)
		}

		var body map[string]interface{}
		json.NewDecoder(respVerify.Body).Decode(&body)
		data := body["data"].(map[string]interface{})

		if data["is_valid"] != true {
			t.Errorf("Expected is_valid=true, got %v (msg: %v)", data["is_valid"], data["message"])
		}
		if data["computed_checksum"] != data["database_checksum"] {
			t.Errorf("Expected computed checksum (%v) to match database checksum (%v)", data["computed_checksum"], data["database_checksum"])
		}
	})

	// =========================================================================
	// 6. TEST Descarga de Archivo Comprimido (GET /download)
	// =========================================================================
	t.Run("6. Descarga Directa - Headers de Attachment y Stream Gzip", func(t *testing.T) {
		reqDown, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/admin/backups/%s/download", server.URL, createdExecutionID), nil)
		reqDown.Header.Set("X-User-Id", adminID)
		reqDown.Header.Set("X-User-Role", "admin")
		reqDown.Header.Set("X-Tenant-Id", tenantID)

		respDown, err := client.Do(reqDown)
		if err != nil || respDown.StatusCode != http.StatusOK {
			t.Fatalf("Failed downloading backup: status=%d, err=%v", respDown.StatusCode, err)
		}

		if respDown.Header.Get("Content-Type") != "application/gzip" {
			t.Errorf("Expected Content-Type 'application/gzip', got %s", respDown.Header.Get("Content-Type"))
		}
		if respDown.Header.Get("Content-Disposition") == "" {
			t.Errorf("Expected Content-Disposition header with filename")
		}

		downloadedBytes, err := io.ReadAll(respDown.Body)
		if err != nil || len(downloadedBytes) == 0 {
			t.Fatalf("Expected non-empty downloaded gzip stream, len=%d", len(downloadedBytes))
		}
	})

	// =========================================================================
	// 7. TEST Seguridad y Autorización (403 Forbidden para Alumnos/Docentes)
	// =========================================================================
	t.Run("7. Control de Acceso por Rol - 403 Forbidden para No Administradores", func(t *testing.T) {
		reqNoAuth, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/backups/trigger", nil)
		reqNoAuth.Header.Set("X-User-Id", studentID)
		reqNoAuth.Header.Set("X-User-Role", "student") // Rol estudiante
		reqNoAuth.Header.Set("X-Tenant-Id", tenantID)

		respNoAuth, err := client.Do(reqNoAuth)
		if err != nil || respNoAuth.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for non-admin user, got %d", respNoAuth.StatusCode)
		}
	})
}
