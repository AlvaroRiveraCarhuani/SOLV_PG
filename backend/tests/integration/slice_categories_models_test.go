package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupSliceCategoriesModelsServer(t *testing.T) (*httptest.Server, *database.Database, domain.AuditLogRepository) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil, nil
	}

	if err := db.RunInitialMigrations(); err != nil {
		t.Fatalf("Failed to run initial migrations: %v", err)
	}

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ('00000000-0000-0000-0000-000000000001', 'Admin', 'Root', 'admin_slice@uab.edu.bo', 'admin', '00000000-0000-0000-0000-000000000001')
		ON CONFLICT (id) DO NOTHING;
	`)

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	auditLogRepo := postgres.NewAuditLogRepository(db.GetDB())

	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)
	govService.SetAuditRepo(auditLogRepo)

	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService).
		WithAuditLogRepo(auditLogRepo)

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
		AdminAcademicHandler: adminAcademicHandler,
		TenantMiddleware:     tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)
	server := httptest.NewServer(mux)

	return server, db, auditLogRepo
}

func TestSlice_CreateTemplate_NoSilentOverwrite_409Conflict(t *testing.T) {
	server, db, _ := setupSliceCategoriesModelsServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	uniqueName := fmt.Sprintf("Template Conflict Test %s", uuid.New().String()[:8])
	defer func() {
		_, _ = db.GetDB().Exec("DELETE FROM lab_templates WHERE name = $1", uniqueName)
	}()

	payload := domain.CreateOfficialTemplateDTO{
		Name:              uniqueName,
		DockerImage:       "python:3.12-slim-bookworm",
		BaseRamMB:         1024,
		TargetEnvironment: "IDE_PERSISTENTE",
		Description:       "Original description",
		SampleInput:       "test-input",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		var errRes map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errRes)
		t.Fatalf("Expected 201 Created on first create, got: %d, body: %v", resp.StatusCode, errRes)
	}

	// Segundo intento con el MISMO nombre: debe retornar 409 Conflict, nunca sobreescribir silenciosamente
	req2, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/templates", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Role", "admin")
	req2.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("Failed to execute second create: %v", err)
	}
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict on duplicate name, got: %d", resp2.StatusCode)
	}
}

func TestSlice_DuplicateTemplate_TwiceWithout500(t *testing.T) {
	server, db, _ := setupSliceCategoriesModelsServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	uniqueName := fmt.Sprintf("Template To Duplicate %s", uuid.New().String()[:8])
	defer func() {
		_, _ = db.GetDB().Exec("DELETE FROM lab_templates WHERE name LIKE $1 OR name LIKE $2", "%"+uniqueName+"%", "%Template With Very Long Name Designed To Test Varchar 100 Limits%")
	}()

	payload := domain.CreateOfficialTemplateDTO{
		Name:              uniqueName,
		DockerImage:       "node:20-bookworm-slim",
		BaseRamMB:         768,
		TargetEnvironment: "IDE_PERSISTENTE",
		SampleInput:       "sample-data",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create template to duplicate: status %d, err %v", resp.StatusCode, err)
	}

	var createdRes struct {
		Data domain.AdminTemplateReviewItem `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createdRes)
	tplID := createdRes.Data.ID

	// Primera duplicación -> (Copia) uniqueName
	dupReq1, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/templates/%s/duplicate", server.URL, tplID), nil)
	dupReq1.Header.Set("X-User-Role", "admin")
	dupReq1.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	dupResp1, err := http.DefaultClient.Do(dupReq1)
	if err != nil || dupResp1.StatusCode != http.StatusCreated {
		t.Fatalf("First duplication failed: status %d, err %v", dupResp1.StatusCode, err)
	}

	// Segunda duplicación -> (Copia 2) uniqueName (no debe fallar con 500)
	dupReq2, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/templates/%s/duplicate", server.URL, tplID), nil)
	dupReq2.Header.Set("X-User-Role", "admin")
	dupReq2.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	dupResp2, err := http.DefaultClient.Do(dupReq2)
	if err != nil || dupResp2.StatusCode != http.StatusCreated {
		t.Fatalf("Second duplication failed with status %d (wanted 201 without 500 collision)", dupResp2.StatusCode)
	}

	// Tercera duplicación con plantilla de nombre muy largo (95 caracteres): no debe violar VARCHAR(100)
	longName := "Template With Very Long Name Designed To Test Varchar 100 Limits On Duplicate " + uuid.New().String()[:15]
	longPayload := domain.CreateOfficialTemplateDTO{
		Name:              longName,
		DockerImage:       "node:20-bookworm-slim",
		BaseRamMB:         512,
		TargetEnvironment: "IDE_PERSISTENTE",
	}
	longBody, _ := json.Marshal(longPayload)
	longReq, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/templates", bytes.NewReader(longBody))
	longReq.Header.Set("Content-Type", "application/json")
	longReq.Header.Set("X-User-Role", "admin")
	longReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")
	longResp, err := http.DefaultClient.Do(longReq)
	if err != nil || longResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create long name template: %v", err)
	}
	var longRes struct {
		Data domain.AdminTemplateReviewItem `json:"data"`
	}
	_ = json.NewDecoder(longResp.Body).Decode(&longRes)

	// Duplicar la de nombre largo 3 veces sucesivas
	currID := longRes.Data.ID
	for d := 1; d <= 3; d++ {
		dReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/templates/%s/duplicate", server.URL, currID), nil)
		dReq.Header.Set("X-User-Role", "admin")
		dReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")
		dResp, err := http.DefaultClient.Do(dReq)
		if err != nil || dResp.StatusCode != http.StatusCreated {
			t.Fatalf("Duplication %d of long name template failed: status %d", d, dResp.StatusCode)
		}
		var dRes struct {
			Data domain.AdminTemplateReviewItem `json:"data"`
		}
		_ = json.NewDecoder(dResp.Body).Decode(&dRes)
		if len(dRes.Data.Name) > 100 {
			t.Errorf("Duplicated name exceeded 100 characters: %d chars (%s)", len(dRes.Data.Name), dRes.Data.Name)
		}
		currID = dRes.Data.ID
	}
}

func TestSlice_CategoryCRUD_And_DeleteInUse_409(t *testing.T) {
	server, db, auditRepo := setupSliceCategoriesModelsServer(t)
	if server == nil {
		return
	}
	defer server.Close()
	defer func() {
		_, _ = db.GetDB().Exec("DELETE FROM lab_templates WHERE category_id IN (SELECT id FROM template_categories WHERE name LIKE 'Category Test %' OR name LIKE 'Orphan Cat %')")
		_, _ = db.GetDB().Exec("DELETE FROM template_models WHERE category_id IN (SELECT id FROM template_categories WHERE name LIKE 'Category Test %' OR name LIKE 'Orphan Cat %')")
		_, _ = db.GetDB().Exec("DELETE FROM template_categories WHERE name LIKE 'Category Test %' OR name LIKE 'Orphan Cat %'")
	}()

	// 1. Crear categoría
	catPayload := domain.CreateCategoryDTO{
		Name:        fmt.Sprintf("Category Test %s", uuid.New().String()[:8]),
		Description: "Descripción para test",
	}
	catBody, _ := json.Marshal(catPayload)
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/template-categories", bytes.NewReader(catBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create category: status %d, err %v", resp.StatusCode, err)
	}

	var catRes struct {
		Data domain.TemplateCategory `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&catRes)
	catID := catRes.Data.ID

	// 2. Crear una plantilla asociada a esta categoría
	tplPayload := domain.CreateOfficialTemplateDTO{
		Name:              fmt.Sprintf("Template with Cat %s", uuid.New().String()[:8]),
		DockerImage:       "python:3.12-slim-bookworm",
		BaseRamMB:         512,
		CategoryID:        &catID,
		TargetEnvironment: "IDE_PERSISTENTE",
	}
	tplBody, _ := json.Marshal(tplPayload)
	tplReq, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/templates", bytes.NewReader(tplBody))
	tplReq.Header.Set("Content-Type", "application/json")
	tplReq.Header.Set("X-User-Role", "admin")
	tplReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	tplResp, err := http.DefaultClient.Do(tplReq)
	if err != nil || tplResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create template with category: status %d", tplResp.StatusCode)
	}

	// 3. Intentar eliminar categoría en uso -> debe retornar 409 Conflict
	delReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/admin/template-categories/%s", server.URL, catID), nil)
	delReq.Header.Set("X-User-Role", "admin")
	delReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil || delResp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict when deleting category in use, got: %d", delResp.StatusCode)
	}

	// 4. Crear categoría huérfana y eliminarla exitosamente -> debe registrar audit log
	orphanPayload := domain.CreateCategoryDTO{
		Name: fmt.Sprintf("Orphan Cat %s", uuid.New().String()[:8]),
	}
	orphanBody, _ := json.Marshal(orphanPayload)
	oReq, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/template-categories", bytes.NewReader(orphanBody))
	oReq.Header.Set("Content-Type", "application/json")
	oReq.Header.Set("X-User-Role", "admin")
	oReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")
	oResp, _ := http.DefaultClient.Do(oReq)
	var oRes struct {
		Data domain.TemplateCategory `json:"data"`
	}
	_ = json.NewDecoder(oResp.Body).Decode(&oRes)

	delOrphanReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/admin/template-categories/%s", server.URL, oRes.Data.ID), nil)
	delOrphanReq.Header.Set("X-User-Role", "admin")
	delOrphanReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")
	delOrphanResp, err := http.DefaultClient.Do(delOrphanReq)
	if err != nil || delOrphanResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK deleting unused category, got: %d", delOrphanResp.StatusCode)
	}

	// Verificar audit log
	if auditRepo != nil {
		logs, err := auditRepo.ListFiltered(context.Background(), "00000000-0000-0000-0000-000000000001", "", "TEMPLATE_CATEGORY_DELETED", 10, 0)
		if err != nil || len(logs) == 0 {
			t.Errorf("Expected audit log for category deletion, got err: %v, count: %d", err, len(logs))
		}
	}
}

func TestSlice_PromoteToModel_Guards_And_UsageCount(t *testing.T) {
	server, db, auditRepo := setupSliceCategoriesModelsServer(t)
	if server == nil {
		return
	}
	defer server.Close()
	defer func() {
		_, _ = db.GetDB().Exec("DELETE FROM template_models WHERE title LIKE 'Promoted Go Model %'")
		_, _ = db.GetDB().Exec("DELETE FROM lab_templates WHERE name LIKE 'Template For Model %'")
	}()

	// 1. Crear una plantilla en estado PENDIENTE_AUDITORIA
	tplPayload := domain.CreateOfficialTemplateDTO{
		Name:              fmt.Sprintf("Template For Model %s", uuid.New().String()[:8]),
		DockerImage:       "golang:1.22-bookworm",
		BaseRamMB:         512,
		TargetEnvironment: "IDE_PERSISTENTE",
		SampleInput:       "echo 42",
	}
	tplBody, _ := json.Marshal(tplPayload)
	tplReq, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/templates", bytes.NewReader(tplBody))
	tplReq.Header.Set("Content-Type", "application/json")
	tplReq.Header.Set("X-User-Role", "admin")
	tplReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	tplResp, _ := http.DefaultClient.Do(tplReq)
	var tplRes struct {
		Data domain.AdminTemplateReviewItem `json:"data"`
	}
	_ = json.NewDecoder(tplResp.Body).Decode(&tplRes)
	tplID := tplRes.Data.ID

	// 2. Intentar promover antes de aprobar -> debe fallar con 422
	promotePayload := domain.PromoteTemplateToModelDTO{
		CategoryID:  "c0000000-0000-0000-0000-000000000002",
		Title:       fmt.Sprintf("Promoted Go Model %s", uuid.New().String()[:8]),
		Description: "Modelo promovido para test",
	}
	pBody, _ := json.Marshal(promotePayload)
	pReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/templates/%s/promote-to-model", server.URL, tplID), bytes.NewReader(pBody))
	pReq.Header.Set("Content-Type", "application/json")
	pReq.Header.Set("X-User-Role", "admin")
	pReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	pResp, _ := http.DefaultClient.Do(pReq)
	if pResp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 422 Unprocessable when promoting unapproved template, got: %d", pResp.StatusCode)
	}

	// 3. Forzar estado APROBADA (o approved) en la plantilla directamente
	_, err := db.GetDB().Exec(`UPDATE lab_templates SET status = 'APROBADA' WHERE id = $1`, tplID)
	if err != nil {
		t.Fatalf("Failed to update template status: %v", err)
	}

	// 4. Promover ahora -> debe responder 201 Created y retornar el modelo
	pReq2, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/templates/%s/promote-to-model", server.URL, tplID), bytes.NewReader(pBody))
	pReq2.Header.Set("Content-Type", "application/json")
	pReq2.Header.Set("X-User-Role", "admin")
	pReq2.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")

	pResp2, err := http.DefaultClient.Do(pReq2)
	if err != nil || pResp2.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created when promoting approved template, got %d, err: %v", pResp2.StatusCode, err)
	}

	// 5. Consultar GET /api/v1/admin/template-models y validar que el modelo aparece con usage_count >= 1
	mReq, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/template-models", nil)
	mReq.Header.Set("X-Tenant-Id", "00000000-0000-0000-0000-000000000001")
	mResp, err := http.DefaultClient.Do(mReq)
	if err != nil || mResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list template models: %v", err)
	}

	var mListRes struct {
		Data []*domain.TemplateModelItemDTO `json:"data"`
	}
	_ = json.NewDecoder(mResp.Body).Decode(&mListRes)
	found := false
	for _, m := range mListRes.Data {
		if m.Title == promotePayload.Title {
			found = true
			if m.UsageCount < 1 {
				t.Errorf("Expected usage_count >= 1, got %d", m.UsageCount)
			}
			if m.SourceTemplateID == nil || *m.SourceTemplateID != tplID {
				t.Errorf("Expected source_template_id %s, got %v", tplID, m.SourceTemplateID)
			}
			break
		}
	}
	if !found {
		t.Errorf("Promoted model not found in GET /api/v1/admin/template-models")
	}

	// 6. Validar audit log de promoción
	if auditRepo != nil {
		logs, err := auditRepo.ListFiltered(context.Background(), "00000000-0000-0000-0000-000000000001", "", "TEMPLATE_PROMOTED_TO_MODEL", 10, 0)
		if err != nil || len(logs) == 0 {
			t.Errorf("Expected audit log TEMPLATE_PROMOTED_TO_MODEL, got: %v", err)
		}
	}
}
