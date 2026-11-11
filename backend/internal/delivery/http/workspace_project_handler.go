package httpdelivery

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type WorkspaceProjectHandler struct {
	service *services.WorkspaceProjectService
}

func NewWorkspaceProjectHandler(service *services.WorkspaceProjectService) *WorkspaceProjectHandler {
	return &WorkspaceProjectHandler{service: service}
}

func (h *WorkspaceProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	tenantID := workspaceTenantID(r)
	exerciseID := r.PathValue("id")
	if !h.authorize(w, r, tenantID, exerciseID) {
		return
	}
	view, err := h.service.GetProject(r.Context(), tenantID, exerciseID)
	if err != nil {
		h.writeError(w, err, "No se pudo leer el proyecto")
		return
	}
	SendJSON(w, http.StatusOK, view, "Proyecto consultado")
}

func (h *WorkspaceProjectHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	exerciseID := r.PathValue("id")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "exercise_id_required", "Falta el identificador del ejercicio")
		return
	}
	if !h.authorize(w, r, workspaceTenantID(r), exerciseID) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024+64*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, http.StatusRequestEntityTooLarge, "project_too_large", "El proyecto excede el tamaño permitido")
		return
	}
	var req services.SaveWorkspaceDraftRequest
	if err := json.Unmarshal(body, &req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_json", "Cuerpo de la petición inválido")
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(string)
	draft, err := h.service.SaveDraft(r.Context(), workspaceTenantID(r), exerciseID, userID, req)
	if err != nil {
		h.writeError(w, err, "No se pudo guardar el borrador")
		return
	}
	SendJSON(w, http.StatusOK, draft, "Borrador guardado")
}

func (h *WorkspaceProjectHandler) Publish(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	exerciseID := r.PathValue("id")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "exercise_id_required", "Falta el identificador del ejercicio")
		return
	}
	if !h.authorize(w, r, workspaceTenantID(r), exerciseID) {
		return
	}
	userID, _ := r.Context().Value(domain.UserIDKey).(string)
	version, err := h.service.Publish(r.Context(), workspaceTenantID(r), exerciseID, userID)
	if err != nil {
		h.writeError(w, err, "No se pudo publicar el proyecto")
		return
	}
	SendJSON(w, http.StatusCreated, version, "Proyecto publicado")
}

// RetryPreparation vuelve a encolar la preparación de la versión publicada
// vigente. La autorización es la de publicar: un reintento consume el mismo
// worker del nodo y el mismo catálogo que la publicación.
func (h *WorkspaceProjectHandler) RetryPreparation(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	exerciseID := r.PathValue("id")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "exercise_id_required", "Falta el identificador del ejercicio")
		return
	}
	tenantID := workspaceTenantID(r)
	if !h.authorize(w, r, tenantID, exerciseID) {
		return
	}
	preparation, err := h.service.RetryPreparation(r.Context(), tenantID, exerciseID)
	if err != nil {
		h.writeError(w, err, "No se pudo reintentar la preparación")
		return
	}
	SendJSON(w, http.StatusAccepted, preparation, "Preparación encolada de nuevo")
}

type workspaceImportManifest struct {
	Files []workspaceImportManifestFile `json:"files"`
}

type workspaceImportManifestFile struct {
	Path  string `json:"path"`
	Field string `json:"field"`
	Size  *int64 `json:"size,omitempty"`
}

// ImportFolder recibe un manifiesto y un archivo multipart por entrada. El
// manifiesto relaciona la ruta relativa con el campo; el campo nunca se usa
// como ruta del proyecto.
func (h *WorkspaceProjectHandler) ImportFolder(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	exerciseID := r.PathValue("id")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "exercise_id_required", "Falta el identificador del ejercicio")
		return
	}
	tenantID := workspaceTenantID(r)
	if !h.authorize(w, r, tenantID, exerciseID) {
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		SendError(w, http.StatusUnsupportedMediaType, "multipart_required", "La importación debe usar multipart/form-data")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4*1024*1024)
	if err := r.ParseMultipartForm(4 * 1024 * 1024); err != nil {
		SendError(w, http.StatusRequestEntityTooLarge, "project_import_too_large", "La carpeta supera el cuerpo HTTP permitido")
		return
	}
	defer r.MultipartForm.RemoveAll()

	var manifest workspaceImportManifest
	if err := json.Unmarshal([]byte(r.FormValue("manifest")), &manifest); err != nil {
		SendError(w, http.StatusUnprocessableEntity, "invalid_import_manifest", "El manifiesto de la carpeta no es válido")
		return
	}
	if len(manifest.Files) == 0 {
		SendError(w, http.StatusUnprocessableEntity, "empty_import_manifest", "La carpeta no contiene archivos")
		return
	}
	if len(manifest.Files) > 300 {
		SendError(w, http.StatusRequestEntityTooLarge, "project_import_too_large", "La carpeta supera el máximo de 300 archivos")
		return
	}

	expectedRevision, err := parseWorkspaceRevision(r.FormValue("expected_revision"))
	if err != nil {
		SendError(w, http.StatusUnprocessableEntity, "invalid_expected_revision", err.Error())
		return
	}

	files := make([]services.WorkspaceProjectImportFile, 0, len(manifest.Files))
	usedFields := make(map[string]struct{}, len(manifest.Files))
	for _, entry := range manifest.Files {
		if entry.Field == "" || entry.Path == "" {
			SendError(w, http.StatusUnprocessableEntity, "invalid_import_manifest", "Cada archivo debe declarar ruta y campo")
			return
		}
		if _, exists := usedFields[entry.Field]; exists {
			SendError(w, http.StatusUnprocessableEntity, "invalid_import_manifest", "El manifiesto reutiliza un campo de archivo")
			return
		}
		usedFields[entry.Field] = struct{}{}
		data, err := readMultipartFile(r, entry.Field, 512*1024)
		if err != nil {
			h.writeImportReadError(w, err, entry.Path)
			return
		}
		if entry.Size != nil && *entry.Size != int64(len(data)) {
			SendError(w, http.StatusUnprocessableEntity, "import_size_mismatch", "El tamaño declarado de "+entry.Path+" no coincide con el archivo recibido")
			return
		}
		files = append(files, services.WorkspaceProjectImportFile{Path: entry.Path, Content: data})
	}
	for field := range r.MultipartForm.File {
		if _, declared := usedFields[field]; !declared {
			SendError(w, http.StatusUnprocessableEntity, "unexpected_import_file", "El formulario contiene un archivo que no está declarado en el manifiesto")
			return
		}
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(string)
	result, err := h.service.ImportFolder(r.Context(), tenantID, exerciseID, userID, expectedRevision, files)
	if err != nil {
		h.writeError(w, err, "No se pudo importar la carpeta")
		return
	}
	SendJSON(w, http.StatusCreated, result, "Carpeta importada al borrador")
}

// ImportZip acepta un único ZIP en memoria y lo entrega al extractor protegido.
func (h *WorkspaceProjectHandler) ImportZip(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	exerciseID := r.PathValue("id")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "exercise_id_required", "Falta el identificador del ejercicio")
		return
	}
	tenantID := workspaceTenantID(r)
	if !h.authorize(w, r, tenantID, exerciseID) {
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		SendError(w, http.StatusUnsupportedMediaType, "multipart_required", "La importación debe usar multipart/form-data")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 11*1024*1024)
	if err := r.ParseMultipartForm(11 * 1024 * 1024); err != nil {
		SendError(w, http.StatusRequestEntityTooLarge, "project_import_too_large", "El ZIP supera el cuerpo HTTP permitido")
		return
	}
	defer r.MultipartForm.RemoveAll()

	expectedRevision, err := parseWorkspaceRevision(r.FormValue("expected_revision"))
	if err != nil {
		SendError(w, http.StatusUnprocessableEntity, "invalid_expected_revision", err.Error())
		return
	}
	archive, err := readMultipartFile(r, "archive", 10*1024*1024)
	if err != nil {
		h.writeImportReadError(w, err, "ZIP")
		return
	}
	for field := range r.MultipartForm.File {
		if field != "archive" {
			SendError(w, http.StatusUnprocessableEntity, "unexpected_import_file", "El formulario del ZIP contiene archivos adicionales")
			return
		}
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(string)
	result, err := h.service.ImportZip(r.Context(), tenantID, exerciseID, userID, expectedRevision, archive)
	if err != nil {
		h.writeError(w, err, "No se pudo importar el ZIP")
		return
	}
	SendJSON(w, http.StatusCreated, result, "ZIP importado al borrador")
}

func (h *WorkspaceProjectHandler) authorize(w http.ResponseWriter, r *http.Request, tenantID, exerciseID string) bool {
	userID, _ := r.Context().Value(domain.UserIDKey).(string)
	if err := h.service.AuthorizeExercise(r.Context(), tenantID, userID, requestRole(r), exerciseID); err != nil {
		h.writeError(w, err, "No tiene permisos sobre el proyecto de este ejercicio")
		return false
	}
	return true
}

// GenerateProject pide al proveedor de IA configurado una propuesta de
// archivos. La respuesta es solo una propuesta: esta ruta no crea el proyecto,
// no guarda borrador y no publica. La credencial y el proveedor salen de la
// configuración institucional, nunca del cuerpo de la petición.
func (h *WorkspaceProjectHandler) GenerateProject(w http.ResponseWriter, r *http.Request) {
	if !workspaceTeacherOrAdmin(w, r) {
		return
	}
	exerciseID := r.PathValue("id")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "exercise_id_required", "Falta el identificador del ejercicio")
		return
	}
	tenantID := workspaceTenantID(r)
	if !h.authorize(w, r, tenantID, exerciseID) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, http.StatusRequestEntityTooLarge, "generation_input_too_large", "La descripción del proyecto excede el tamaño permitido")
		return
	}
	var req services.GenerateWorkspaceProjectRequest
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_json", "Cuerpo de la petición inválido")
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(string)
	proposal, err := h.service.GenerateProposal(r.Context(), tenantID, userID, exerciseID, req)
	if err != nil {
		h.writeGenerateError(w, err)
		return
	}
	SendJSON(w, http.StatusOK, proposal, "Propuesta generada para revisar")
}

// writeGenerateError distingue el fallo del proveedor del fallo de permiso:
// el docente necesita saber si falta configurar la IA, si el motor tardó
// demasiado o si la respuesta no cumplió el contrato.
func (h *WorkspaceProjectHandler) writeGenerateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAITimeout):
		SendError(w, http.StatusGatewayTimeout, "ai_timeout", "El motor de IA superó el tiempo límite de espera. Probá nuevamente o describí un proyecto más acotado.")
	case errors.Is(err, domain.ErrAIDisabled):
		SendError(w, http.StatusConflict, "ai_disabled", err.Error())
	case errors.Is(err, domain.ErrAIQuotaExceeded):
		SendError(w, http.StatusTooManyRequests, "ai_quota_exceeded", err.Error())
	default:
		var vErr *services.AISettingsValidationError
		if errors.As(err, &vErr) {
			status := http.StatusBadRequest
			if vErr.Code == "ai_response_invalid" {
				status = http.StatusUnprocessableEntity
			}
			SendError(w, status, vErr.Code, vErr.Message)
			return
		}
		h.writeError(w, err, "No se pudo generar la propuesta de proyecto")
	}
}

func parseWorkspaceRevision(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	revision, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || revision < 0 {
		return 0, errors.New("La revisión esperada no es válida")
	}
	return revision, nil
}

func readMultipartFile(r *http.Request, field string, maxBytes int64) ([]byte, error) {
	headers := r.MultipartForm.File[field]
	if len(headers) == 0 {
		return nil, errors.New("missing_file")
	}
	if len(headers) != 1 {
		return nil, errors.New("duplicate_file")
	}
	file, err := headers[0].Open()
	if err != nil {
		return nil, errors.New("unreadable_file")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, errors.New("unreadable_file")
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("file_too_large")
	}
	return data, nil
}

func (h *WorkspaceProjectHandler) writeImportReadError(w http.ResponseWriter, err error, target string) {
	if err.Error() == "file_too_large" {
		SendError(w, http.StatusRequestEntityTooLarge, "project_import_too_large", "El archivo "+target+" excede el tamaño permitido")
		return
	}
	SendError(w, http.StatusUnprocessableEntity, "invalid_import_file", "No se pudo leer el archivo "+target+" completo")
}

func (h *WorkspaceProjectHandler) writeError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, domain.ErrWorkspaceProjectForbidden):
		SendError(w, http.StatusForbidden, "workspace_project_forbidden", err.Error())
	case errors.Is(err, domain.ErrWorkspaceProjectNotIDEPersistent):
		SendError(w, http.StatusUnprocessableEntity, "workspace_project_not_ide", err.Error())
	case errors.Is(err, domain.ErrWorkspaceProjectImportTooLarge):
		SendError(w, http.StatusRequestEntityTooLarge, "project_import_too_large", err.Error())
	case errors.Is(err, domain.ErrWorkspaceProjectImportInvalid):
		SendError(w, http.StatusUnprocessableEntity, "invalid_project_import", err.Error())
	case errors.Is(err, domain.ErrWorkspacePublishingDisabled):
		SendError(w, http.StatusForbidden, "workspace_publishing_disabled", err.Error())
	case errors.Is(err, domain.ErrDependencyPreparationNotRetryable):
		SendError(w, http.StatusConflict, "preparation_not_retryable", err.Error())
	case errors.Is(err, domain.ErrDependencyPreparationDisabled):
		SendError(w, http.StatusConflict, "preparation_unavailable", err.Error())
	case errors.Is(err, domain.ErrDependencyPreparationJobNotFound):
		SendError(w, http.StatusNotFound, "preparation_not_found", "La versión no tiene preparación registrada")
	case errors.Is(err, domain.ErrWorkspaceDraftRevisionConflict):
		SendError(w, http.StatusConflict, "draft_revision_conflict", err.Error())
	case errors.Is(err, domain.ErrWorkspaceDraftEmpty), errors.Is(err, domain.ErrInvalidResourceID):
		SendError(w, http.StatusUnprocessableEntity, "invalid_project", err.Error())
	case errors.Is(err, domain.ErrWorkspaceProjectNotFound), errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, "project_not_found", "Proyecto no encontrado")
	default:
		SendError(w, http.StatusInternalServerError, "workspace_project_error", fallback)
	}
}

func workspaceTeacherOrAdmin(w http.ResponseWriter, r *http.Request) bool {
	role := requestRole(r)
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "teacher_role_required", "No tiene permisos para gestionar proyectos")
		return false
	}
	return true
}

func workspaceTenantID(r *http.Request) string {
	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		return domain.DefaultTenantID
	}
	return tenantID
}
