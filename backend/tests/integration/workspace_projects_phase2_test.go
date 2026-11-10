package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func TestWorkspaceProjectPhase2ImportAuthorizationAndPublishFlow(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	seed := seedWorkspaceTestBase(t, db.GetDB())
	ctx := context.Background()

	repo := postgres.NewPostgresWorkspaceProjectRepository(db.GetDB())
	exercises := postgres.NewPostgresExerciseRepository(db.GetDB())
	subjects := postgres.NewPostgresSubjectRepository(db.GetDB())
	service := services.NewWorkspaceProjectService(repo, false)
	service.SetAccessRepositories(exercises, subjects)

	if err := service.AuthorizeExercise(ctx, seed.tenantID, seed.teacherID, "teacher", seed.exerciseA); err != nil {
		t.Fatalf("el docente dueño debe autorizar: %v", err)
	}
	if err := service.AuthorizeExercise(ctx, seed.tenantID, uuid.NewString(), "teacher", seed.exerciseA); !errors.Is(err, domain.ErrWorkspaceProjectForbidden) {
		t.Fatalf("otro docente debía quedar fuera, obtuvo %v", err)
	}
	if err := service.AuthorizeExercise(ctx, seed.tenantID, uuid.NewString(), "admin", seed.exerciseA); err != nil {
		t.Fatalf("el administrador del tenant debe autorizar: %v", err)
	}

	var seedTemplateID string
	if err := db.GetDB().Get(&seedTemplateID, `SELECT id FROM lab_templates WHERE tenant_id = $1 LIMIT 1`, seed.tenantID); err != nil {
		t.Fatalf("leer plantilla del seed: %v", err)
	}
	judgeExercise := uuid.NewString()
	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (id, tenant_id, subject_id, title, type, status, environment_type, language, boilerplate, template_id)
		VALUES ($1, $2, $3, 'F2 Judge', 'algorithm', 'draft', 'JUEZ_EFIMERO', 'python', '', $4)
	`, judgeExercise, seed.tenantID, seed.subjectID, seedTemplateID); err != nil {
		t.Fatalf("seed judge exercise: %v", err)
	}
	if err := service.AuthorizeExercise(ctx, seed.tenantID, seed.teacherID, "teacher", judgeExercise); !errors.Is(err, domain.ErrWorkspaceProjectNotIDEPersistent) {
		t.Fatalf("JUEZ_EFIMERO debía quedar fuera, obtuvo %v", err)
	}

	otherTenant := uuid.NewString()
	otherTeacher := uuid.NewString()
	otherSubject := uuid.NewString()
	otherExercise := uuid.NewString()
	if _, err := db.GetDB().Exec(`INSERT INTO tenants (id, name, slug) VALUES ($1, 'F2 Tenant', $2)`, otherTenant, "f2-"+otherTenant[:8]); err != nil {
		t.Fatalf("seed other tenant: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'F2', 'Teacher', $2, 'teacher', $3)
	`, otherTeacher, "f2-"+otherTeacher[:8]+"@example.test", otherTenant); err != nil {
		t.Fatalf("seed other teacher: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'F2 Subject', $3, $4)
	`, otherSubject, otherTenant, "F2-"+otherSubject[:8], otherTeacher); err != nil {
		t.Fatalf("seed other subject: %v", err)
	}
	otherTemplate := uuid.NewString()
	if _, err := db.GetDB().Exec(`
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, status)
		VALUES ($1, $2, 'F2 Other Template', 'solv/ide:test', 512, 'IDE_PERSISTENTE', 'approved')
	`, otherTemplate, otherTenant); err != nil {
		t.Fatalf("seed other template: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		INSERT INTO exercises (id, tenant_id, subject_id, title, type, status, environment_type, language, boilerplate, template_id)
		VALUES ($1, $2, $3, 'F2 Other Exercise', 'project', 'draft', 'IDE_PERSISTENTE', 'python', '', $4)
	`, otherExercise, otherTenant, otherSubject, otherTemplate); err != nil {
		t.Fatalf("seed other exercise: %v", err)
	}
	if err := service.AuthorizeExercise(ctx, seed.tenantID, seed.teacherID, "teacher", otherExercise); !errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
		t.Fatalf("un ejercicio de otro tenant debía quedar fuera, obtuvo %v", err)
	}

	result, err := service.ImportFolder(ctx, seed.tenantID, seed.exerciseA, seed.teacherID, 0, []services.WorkspaceProjectImportFile{
		{Path: "proyecto-go/go.mod", Content: []byte("module ejemplo\n")},
		{Path: "proyecto-go/main.go", Content: []byte("package main\n")},
	})
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}
	if result.FileCount != 2 || result.Draft.Revision != 1 {
		t.Fatalf("importación inesperada: %+v", result)
	}

	var draftCount, versionCount int
	var boilerplate string
	if err := db.GetDB().Get(&draftCount, `SELECT COUNT(*) FROM workspace_project_drafts WHERE tenant_id = $1`, seed.tenantID); err != nil {
		t.Fatalf("contar borradores: %v", err)
	}
	if err := db.GetDB().Get(&versionCount, `SELECT COUNT(*) FROM workspace_project_versions WHERE tenant_id = $1`, seed.tenantID); err != nil {
		t.Fatalf("contar versiones: %v", err)
	}
	if err := db.GetDB().Get(&boilerplate, `SELECT boilerplate FROM exercises WHERE id = $1`, seed.exerciseA); err != nil {
		t.Fatalf("leer boilerplate: %v", err)
	}
	if draftCount != 1 || versionCount != 0 || boilerplate != "" {
		t.Fatalf("importar no debe publicar ni tocar boilerplate: drafts=%d versions=%d boilerplate=%q", draftCount, versionCount, boilerplate)
	}

	archive := phase2Zip(t, map[string]string{
		"proyecto-go/README.md":  "# Proyecto\n",
		"proyecto-go/src/app.py": "print('hola')\n",
	})
	zipped, err := service.ImportZip(ctx, seed.tenantID, seed.exerciseA, seed.teacherID, result.Draft.Revision, archive)
	if err != nil {
		t.Fatalf("ImportZip: %v", err)
	}
	if zipped.Draft.Revision != 2 {
		t.Fatalf("revisión ZIP esperada 2, obtenida %d", zipped.Draft.Revision)
	}
	if _, err := service.ImportFolder(ctx, seed.tenantID, seed.exerciseA, seed.teacherID, 1, []services.WorkspaceProjectImportFile{{Path: "stale.py", Content: []byte("print(1)\n")}}); !errors.Is(err, domain.ErrWorkspaceDraftRevisionConflict) {
		t.Fatalf("la revisión obsoleta debía rechazarse, obtuvo %v", err)
	}

	if _, err := service.Publish(ctx, seed.tenantID, seed.exerciseA, seed.teacherID); !errors.Is(err, domain.ErrWorkspacePublishingDisabled) {
		t.Fatalf("la publicación debía estar cerrada, obtuvo %v", err)
	}

	publishedService := services.NewWorkspaceProjectService(repo, true)
	publishedService.SetAccessRepositories(exercises, subjects)
	version, err := publishedService.Publish(ctx, seed.tenantID, seed.exerciseA, seed.teacherID)
	if err != nil {
		t.Fatalf("publicación autorizada: %v", err)
	}
	if version.VersionNumber != 1 {
		t.Fatalf("versión publicada inesperada: %+v", version)
	}
	if err := db.GetDB().Get(&boilerplate, `SELECT boilerplate FROM exercises WHERE id = $1`, seed.exerciseA); err != nil {
		t.Fatalf("leer boilerplate publicado: %v", err)
	}
	if boilerplate != zipped.Draft.TextSnapshot {
		t.Fatalf("el espejo publicado no coincide con el borrador")
	}

	view, err := publishedService.GetProject(ctx, seed.tenantID, seed.exerciseA)
	if err != nil {
		t.Fatalf("GetProject tras publicar: %v", err)
	}
	if view.Project.CurrentPublishedVersionID == nil || *view.Project.CurrentPublishedVersionID != version.ID || len(view.Versions) != 1 {
		t.Fatalf("puntero o historial inesperado: %+v", view)
	}
	if !view.PublishingEnabled {
		t.Fatal("la vista debe informar el flag de publicación habilitado")
	}

	if _, err := publishedService.ImportFolder(ctx, seed.tenantID, seed.exerciseA, seed.teacherID, zipped.Draft.Revision, []services.WorkspaceProjectImportFile{
		{Path: "proyecto-go/main.go", Content: []byte("package main\n\n// cambio posterior\n")},
	}); err != nil {
		t.Fatalf("importación posterior a publicar: %v", err)
	}
	var storedVersionSnapshot string
	if err := db.GetDB().Get(&storedVersionSnapshot, `SELECT text_snapshot FROM workspace_project_versions WHERE id = $1`, version.ID); err != nil {
		t.Fatalf("leer versión publicada: %v", err)
	}
	if storedVersionSnapshot != zipped.Draft.TextSnapshot {
		t.Fatal("una actualización posterior del borrador alteró la versión publicada")
	}
	if err := db.GetDB().Get(&versionCount, `SELECT COUNT(*) FROM workspace_project_versions WHERE tenant_id = $1`, seed.tenantID); err != nil {
		t.Fatalf("contar versiones finales: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("importar después de publicar creó versiones: %d", versionCount)
	}
}

func phase2Zip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("crear %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatalf("escribir %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar ZIP: %v", err)
	}
	return buffer.Bytes()
}

func TestWorkspaceProjectPhase2HTTPImportContracts(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	seed := seedWorkspaceTestBase(t, db.GetDB())
	repo := postgres.NewPostgresWorkspaceProjectRepository(db.GetDB())
	service := services.NewWorkspaceProjectService(repo, false)
	service.SetAccessRepositories(postgres.NewPostgresExerciseRepository(db.GetDB()), postgres.NewPostgresSubjectRepository(db.GetDB()))
	handler := httpdelivery.NewWorkspaceProjectHandler(service)
	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &httpdelivery.Handlers{
		WorkspaceProjectHandler: handler,
		TenantMiddleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, withTestIdentityFromHeaders(r))
			})
		},
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	folderBody, folderType := phase2FolderMultipart(t, map[string]string{
		"proyecto-go/main.go":          "package main\n",
		"proyecto-go/internal/calc.go": "package internal\n",
	})
	resp := phase2Request(t, server.URL, http.MethodPost, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/import/folder", seed.tenantID, seed.teacherID, "teacher", folderType, folderBody)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("importar carpeta devolvió %d: %s", resp.StatusCode, body)
	}
	_ = resp.Body.Close()

	zipBody, contentType := phase2ZipMultipart(t, map[string]string{"proyecto-go/README.md": "# Proyecto\n"}, 1)
	resp = phase2Request(t, server.URL, http.MethodPost, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/import/zip", seed.tenantID, seed.teacherID, "teacher", contentType, zipBody)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("importar ZIP devolvió %d: %s", resp.StatusCode, body)
	}
	_ = resp.Body.Close()

	resp = phase2Request(t, server.URL, http.MethodGet, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project", seed.tenantID, seed.teacherID, "teacher", "", nil)
	payload, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(payload), "proyecto-go/README.md") || !strings.Contains(string(payload), "\"publishing_enabled\":false") {
		t.Fatalf("GET del proyecto inesperado (%d): %s", resp.StatusCode, payload)
	}

	resp = phase2Request(t, server.URL, http.MethodPost, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/import/folder", seed.tenantID, seed.teacherID, "student", folderType, folderBody)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("student debía recibir 403, recibió %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = phase2Request(t, server.URL, http.MethodGet, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project", seed.tenantID, seed.teacherID, "admin", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin debía leer el proyecto, recibió %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = phase2Request(t, server.URL, http.MethodGet, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project", uuid.NewString(), seed.teacherID, "teacher", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("otro tenant debía recibir 404, recibió %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	var mismatch bytes.Buffer
	mismatchWriter := multipart.NewWriter(&mismatch)
	_ = mismatchWriter.WriteField("manifest", `{"files":[{"path":"main.go","field":"file_0","size":999}]}`)
	_ = mismatchWriter.WriteField("expected_revision", "2")
	mismatchPart, err := mismatchWriter.CreateFormFile("file_0", "main.go")
	if err != nil {
		t.Fatalf("crear archivo incompleto: %v", err)
	}
	_, _ = mismatchPart.Write([]byte("x\n"))
	_ = mismatchWriter.Close()
	resp = phase2Request(t, server.URL, http.MethodPost, "/api/v1/teacher/exercises/"+seed.exerciseA+"/project/import/folder", seed.tenantID, seed.teacherID, "teacher", mismatchWriter.FormDataContentType(), mismatch.Bytes())
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("un tamaño inconsistente debía recibir 422, recibió %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func phase2Request(t *testing.T, baseURL, method, path, tenantID, userID, role, contentType string, body []byte) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, baseURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("X-Test-Tenant-Id", tenantID)
	request.Header.Set("X-Test-User-Id", userID)
	request.Header.Set("X-Test-User-Role", role)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do %s: %v", path, err)
	}
	return response
}

func phase2FolderMultipart(t *testing.T, files map[string]string) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	manifestFiles := make([]map[string]any, 0, len(files))
	index := 0
	for path, content := range files {
		field := fmt.Sprintf("file_%d", index)
		manifestFiles = append(manifestFiles, map[string]any{"path": path, "field": field, "size": len(content)})
		part, err := writer.CreateFormFile(field, path[strings.LastIndex(path, "/")+1:])
		if err != nil {
			t.Fatalf("crear parte %s: %v", field, err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("escribir parte %s: %v", field, err)
		}
		index++
	}
	manifest, err := json.Marshal(map[string]any{"files": manifestFiles})
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if err := writer.WriteField("manifest", string(manifest)); err != nil {
		t.Fatalf("manifest field: %v", err)
	}
	if err := writer.WriteField("expected_revision", "0"); err != nil {
		t.Fatalf("revision field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func phase2ZipMultipart(t *testing.T, files map[string]string, expectedRevision int) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("expected_revision", strconv.Itoa(expectedRevision)); err != nil {
		t.Fatalf("revision field: %v", err)
	}
	part, err := writer.CreateFormFile("archive", "proyecto.zip")
	if err != nil {
		t.Fatalf("crear ZIP: %v", err)
	}
	if _, err := part.Write(phase2Zip(t, files)); err != nil {
		t.Fatalf("escribir ZIP: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart ZIP: %v", err)
	}
	return body.Bytes(), writer.FormDataContentType()
}
