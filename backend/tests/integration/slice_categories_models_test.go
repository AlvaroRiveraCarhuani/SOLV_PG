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

func TestSlice_TemplateCategories_Reorder_And_ModelDeleteGuard(t *testing.T) {
	server, db, _ := setupSliceCategoriesModelsServer(t)
	if server == nil {
		return
	}
	defer server.Close()
	defer func() {
		_, _ = db.GetDB().Exec("DELETE FROM template_models WHERE title LIKE 'Model For Cat Guard %'")
		_, _ = db.GetDB().Exec("DELETE FROM template_categories WHERE name LIKE 'Reorder Cat %'")
	}()

	// 1. Crear 2 categorías
	catName1 := fmt.Sprintf("Reorder Cat A %s", uuid.New().String()[:8])
	catName2 := fmt.Sprintf("Reorder Cat B %s", uuid.New().String()[:8])

	c1Payload := domain.CreateCategoryDTO{Name: catName1, Description: "Cat A"}
	b1, _ := json.Marshal(c1Payload)
	req1, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/template-categories", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User-Role", "admin")
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil || resp1.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create cat A: %v", err)
	}
	var res1 struct{ Data domain.TemplateCategory }
	_ = json.NewDecoder(resp1.Body).Decode(&res1)

	c2Payload := domain.CreateCategoryDTO{Name: catName2, Description: "Cat B"}
	b2, _ := json.Marshal(c2Payload)
	req2, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/template-categories", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Role", "admin")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil || resp2.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create cat B: %v", err)
	}
	var res2 struct{ Data domain.TemplateCategory }
	_ = json.NewDecoder(resp2.Body).Decode(&res2)

	// 2. Reordenar asignando sort_order explícito: B -> 1, A -> 5
	reorderItems := []domain.ReorderCategoryItemDTO{
		{ID: res2.Data.ID, SortOrder: 1},
		{ID: res1.Data.ID, SortOrder: 5},
	}
	reorderBody, _ := json.Marshal(reorderItems)
	rReq, _ := http.NewRequest("PUT", server.URL+"/api/v1/admin/template-categories/reorder", bytes.NewReader(reorderBody))
	rReq.Header.Set("Content-Type", "application/json")
	rReq.Header.Set("X-User-Role", "admin")
	rResp, err := http.DefaultClient.Do(rReq)
	if err != nil || rResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to reorder categories: status %d, err %v", rResp.StatusCode, err)
	}

	// 3. Listar categorías y verificar persistencia del sort_order
	listReq, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/template-categories", nil)
	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil || listResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list categories: %v", err)
	}
	var catListRes struct {
		Data []*domain.TemplateCategory `json:"data"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&catListRes)

	var foundCat1, foundCat2 *domain.TemplateCategory
	for _, c := range catListRes.Data {
		if c.ID == res1.Data.ID {
			foundCat1 = c
		}
		if c.ID == res2.Data.ID {
			foundCat2 = c
		}
	}
	if foundCat1 == nil || foundCat1.SortOrder != 5 {
		t.Fatalf("Expected cat A sort_order 5, got %+v", foundCat1)
	}
	if foundCat2 == nil || foundCat2.SortOrder != 1 {
		t.Fatalf("Expected cat B sort_order 1, got %+v", foundCat2)
	}

	// 4. Asociar un modelo a cat B y verificar que DELETE retorna 409
	modelTitle := fmt.Sprintf("Model For Cat Guard %s", uuid.New().String()[:8])
	_, err = db.GetDB().Exec(`
		INSERT INTO template_models (id, tenant_id, category_id, title, description, target_environment, docker_image, base_ram_mb, tools, created_at, updated_at)
		VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', $1, $2, 'Test Guard Desc', 'IDE_PERSISTENTE', 'python:3.12-slim', 512, '[]'::jsonb, NOW(), NOW())
	`, res2.Data.ID, modelTitle)
	if err != nil {
		t.Fatalf("Failed to insert template model: %v", err)
	}

	delReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/admin/template-categories/%s", server.URL, res2.Data.ID), nil)
	delReq.Header.Set("X-User-Role", "admin")
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil || delResp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict when deleting category with models, got status: %d", delResp.StatusCode)
	}
}

func TestSlice_TemplateModels_Update_Deactivate_Reactivate(t *testing.T) {
	server, db, auditRepo := setupSliceCategoriesModelsServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	modelTitle := fmt.Sprintf("Model Lifecycle Test %s", uuid.New().String()[:8])
	defer func() {
		_, _ = db.GetDB().Exec("DELETE FROM lab_templates WHERE name LIKE 'Child Template For Model %'")
		_, _ = db.GetDB().Exec("DELETE FROM template_models WHERE title LIKE 'Model Lifecycle Test %'")
	}()

	// 1. Insertar modelo inicial
	var modelID string
	err := db.GetDB().QueryRow(`
		INSERT INTO template_models (id, tenant_id, category_id, title, description, target_environment, docker_image, base_ram_mb, tools, is_active, sort_order, created_at, updated_at)
		VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', $1, 'Initial Desc', 'IDE_PERSISTENTE', 'node:20-bookworm-slim', 512, '[]'::jsonb, true, 0, NOW(), NOW())
		RETURNING id
	`, modelTitle).Scan(&modelID)
	if err != nil {
		t.Fatalf("Failed to insert template model: %v", err)
	}

	// 2. Crear plantilla hija asociada a este modelo
	childTplName := fmt.Sprintf("Child Template For Model %s", uuid.New().String()[:8])
	var childTplID string
	err = db.GetDB().QueryRow(`
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, model_id, status, created_at, updated_at)
		VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', $1, 'node:20-bookworm-slim', 512, 'IDE_PERSISTENTE', $2, 'APROBADA', NOW(), NOW())
		RETURNING id
	`, childTplName, modelID).Scan(&childTplID)
	if err != nil {
		t.Fatalf("Failed to insert child template: %v", err)
	}

	// 3. PUT /api/v1/admin/template-models/{id} - actualizar metadatos
	updatedTitle := modelTitle + " (Actualizado)"
	updatedDesc := "Descripción actualizada del modelo"
	upPayload := domain.UpdateTemplateModelDTO{
		Title:       updatedTitle,
		Description: updatedDesc,
		CategoryID:  "c0000000-0000-0000-0000-000000000002",
	}
	upBody, _ := json.Marshal(upPayload)
	upReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/admin/template-models/%s", server.URL, modelID), bytes.NewReader(upBody))
	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("X-User-Role", "admin")
	upResp, err := http.DefaultClient.Do(upReq)
	if err != nil || upResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to update template model: status %d, err %v", upResp.StatusCode, err)
	}

	// 4. POST /api/v1/admin/template-models/{id}/deactivate
	deactReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/template-models/%s/deactivate", server.URL, modelID), nil)
	deactReq.Header.Set("X-User-Role", "admin")
	deactResp, err := http.DefaultClient.Do(deactReq)
	if err != nil || deactResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to deactivate template model: status %d, err %v", deactResp.StatusCode, err)
	}

	// 5. GET /api/v1/admin/template-models (default: include_inactive=false) -> no debe incluir el modelo desactivado
	listActiveReq, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/template-models", nil)
	listActiveResp, err := http.DefaultClient.Do(listActiveReq)
	if err != nil || listActiveResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list active models: %v", err)
	}
	var activeListRes struct {
		Data []*domain.TemplateModelItemDTO `json:"data"`
	}
	_ = json.NewDecoder(listActiveResp.Body).Decode(&activeListRes)
	for _, m := range activeListRes.Data {
		if m.ID == modelID {
			t.Fatalf("Model %s was found in active list, but it should be excluded when inactive", modelID)
		}
	}

	// 6. GET /api/v1/admin/template-models?include_inactive=true -> debe incluir el modelo con is_active = false
	listAllReq, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/template-models?include_inactive=true", nil)
	listAllResp, err := http.DefaultClient.Do(listAllReq)
	if err != nil || listAllResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list all models: %v", err)
	}
	var allListRes struct {
		Data []*domain.TemplateModelItemDTO `json:"data"`
	}
	_ = json.NewDecoder(listAllResp.Body).Decode(&allListRes)
	foundModel := false
	for _, m := range allListRes.Data {
		if m.ID == modelID {
			foundModel = true
			if m.IsActive {
				t.Fatalf("Expected model is_active = false, got true")
			}
			if m.Title != updatedTitle {
				t.Fatalf("Expected updated title %s, got %s", updatedTitle, m.Title)
			}
			if m.UsageCount < 1 {
				t.Fatalf("Expected usage count >= 1 from child template, got %d", m.UsageCount)
			}
			break
		}
	}
	if !foundModel {
		t.Fatalf("Model %s was not found in all models list (with include_inactive=true)", modelID)
	}

	// 7. Verificar que la plantilla hija SIGUE existiendo y referenciando a model_id (no se rompió)
	var existingChildModelID *string
	err = db.GetDB().QueryRow(`SELECT model_id FROM lab_templates WHERE id = $1`, childTplID).Scan(&existingChildModelID)
	if err != nil || existingChildModelID == nil || *existingChildModelID != modelID {
		t.Fatalf("Child template link broken or missing: err %v, model_id: %v", err, existingChildModelID)
	}

	// 8. POST /api/v1/admin/template-models/{id}/reactivate
	reactReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/template-models/%s/reactivate", server.URL, modelID), nil)
	reactReq.Header.Set("X-User-Role", "admin")
	reactResp, err := http.DefaultClient.Do(reactReq)
	if err != nil || reactResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to reactivate template model: status %d, err %v", reactResp.StatusCode, err)
	}

	// 9. Verificar que vuelve a aparecer en la lista activa
	listAgainReq, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/template-models", nil)
	listAgainResp, err := http.DefaultClient.Do(listAgainReq)
	if err != nil || listAgainResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list active models after reactivation: %v", err)
	}
	var activeListAgainRes struct {
		Data []*domain.TemplateModelItemDTO `json:"data"`
	}
	_ = json.NewDecoder(listAgainResp.Body).Decode(&activeListAgainRes)
	foundActiveAgain := false
	for _, m := range activeListAgainRes.Data {
		if m.ID == modelID {
			foundActiveAgain = true
			if !m.IsActive {
				t.Errorf("Expected model is_active = true after reactivation")
			}
			break
		}
	}
	if !foundActiveAgain {
		t.Fatalf("Model %s was not found in active list after reactivation", modelID)
	}

	// 10. Verificar logs de auditoría
	if auditRepo != nil {
		logs, err := auditRepo.ListFiltered(context.Background(), "00000000-0000-0000-0000-000000000001", "", "TEMPLATE_MODEL_DEACTIVATED", 10, 0)
		if err != nil || len(logs) == 0 {
			t.Errorf("Expected audit log TEMPLATE_MODEL_DEACTIVATED, got err: %v", err)
		}
	}
}
