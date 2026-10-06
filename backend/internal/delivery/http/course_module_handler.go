package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/dto"
	"solv-backend/internal/delivery/http/middleware"
)

type CourseModuleHandler struct {
	service *services.CourseModuleService
}

func NewCourseModuleHandler(service *services.CourseModuleService) *CourseModuleHandler {
	return &CourseModuleHandler{service: service}
}

func (h *CourseModuleHandler) CreateModule(w http.ResponseWriter, r *http.Request) {
	courseID := r.PathValue("courseId")
	if courseID == "" {
		courseID = r.PathValue("id")
	}
	if courseID == "" {
		SendError(w, http.StatusBadRequest, "Course ID required", "ID del curso requerido")
		return
	}

	var req dto.CreateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	if req.Title == "" {
		SendError(w, http.StatusUnprocessableEntity, "Title is required", "El título del módulo es obligatorio")
		return
	}

	module, err := h.service.CreateModule(r.Context(), courseID, req.Title, req.Description, req.OrderIndex, req.PassScore)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al crear el módulo")
		return
	}

	SendJSON(w, http.StatusCreated, module, "Módulo creado exitosamente")
}

func (h *CourseModuleHandler) ListModules(w http.ResponseWriter, r *http.Request) {
	courseID := r.PathValue("courseId")
	if courseID == "" {
		courseID = r.PathValue("id")
	}
	if courseID == "" {
		SendError(w, http.StatusBadRequest, "Course ID required", "ID del curso requerido")
		return
	}

	modules, err := h.service.ListModulesBySubject(r.Context(), courseID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al listar los módulos")
		return
	}

	SendJSON(w, http.StatusOK, modules, "Módulos obtenidos exitosamente")
}

func (h *CourseModuleHandler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	moduleID := r.PathValue("moduleId")
	if moduleID == "" {
		moduleID = r.PathValue("id")
	}
	if moduleID == "" {
		SendError(w, http.StatusBadRequest, "Module ID required", "ID del módulo requerido")
		return
	}

	var req dto.UpdateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	module, err := h.service.UpdateModule(r.Context(), moduleID, req.Title, req.Description, req.OrderIndex, req.PassScore)
	if err != nil {
		if errors.Is(err, domain.ErrModuleNotFound) {
			SendError(w, http.StatusNotFound, "MODULE_NOT_FOUND", "Módulo no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar el módulo")
		return
	}

	SendJSON(w, http.StatusOK, module, "Módulo actualizado exitosamente")
}

func (h *CourseModuleHandler) DeleteModule(w http.ResponseWriter, r *http.Request) {
	moduleID := r.PathValue("moduleId")
	if moduleID == "" {
		moduleID = r.PathValue("id")
	}
	if moduleID == "" {
		SendError(w, http.StatusBadRequest, "Module ID required", "ID del módulo requerido")
		return
	}

	err := h.service.DeleteModule(r.Context(), moduleID)
	if err != nil {
		if errors.Is(err, domain.ErrModuleNotFound) {
			SendError(w, http.StatusNotFound, "MODULE_NOT_FOUND", "Módulo no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al eliminar el módulo")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"deleted_id": moduleID}, "Módulo eliminado exitosamente")
}

func (h *CourseModuleHandler) SetPrerequisites(w http.ResponseWriter, r *http.Request) {
	moduleID := r.PathValue("moduleId")
	if moduleID == "" {
		moduleID = r.PathValue("id")
	}
	if moduleID == "" {
		SendError(w, http.StatusBadRequest, "Module ID required", "ID del módulo requerido")
		return
	}

	var req dto.SetPrerequisitesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	err := h.service.SetPrerequisites(r.Context(), moduleID, req.PrerequisiteModuleIDs)
	if err != nil {
		if errors.Is(err, domain.ErrCyclicPrerequisite) {
			SendError(w, http.StatusUnprocessableEntity, "CYCLIC_PREREQUISITE", "No se pueden crear ciclos de prerrequisitos entre módulos")
			return
		}
		if errors.Is(err, domain.ErrPrerequisiteCrossCourse) {
			SendError(w, http.StatusUnprocessableEntity, "CROSS_COURSE_PREREQUISITE", "El módulo prerrequisito debe pertenecer al mismo curso")
			return
		}
		if errors.Is(err, domain.ErrSelfPrerequisite) {
			SendError(w, http.StatusUnprocessableEntity, "SELF_PREREQUISITE", "Un módulo no puede ser prerrequisito de sí mismo")
			return
		}
		if errors.Is(err, domain.ErrModuleNotFound) {
			SendError(w, http.StatusNotFound, "MODULE_NOT_FOUND", "Módulo no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar prerrequisitos")
		return
	}

	SendJSON(w, http.StatusOK, map[string]any{"module_id": moduleID, "prerequisites": req.PrerequisiteModuleIDs}, "Prerrequisitos actualizados exitosamente")
}

func (h *CourseModuleHandler) DeletePrerequisites(w http.ResponseWriter, r *http.Request) {
	moduleID := r.PathValue("moduleId")
	if moduleID == "" {
		moduleID = r.PathValue("id")
	}
	if moduleID == "" {
		SendError(w, http.StatusBadRequest, "Module ID required", "ID del módulo requerido")
		return
	}

	err := h.service.SetPrerequisites(r.Context(), moduleID, []string{})
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al limpiar prerrequisitos")
		return
	}

	SendJSON(w, http.StatusOK, map[string]any{"module_id": moduleID, "prerequisites": []string{}}, "Prerrequisitos limpiados exitosamente")
}

func (h *CourseModuleHandler) AssignExerciseModule(w http.ResponseWriter, r *http.Request) {
	exerciseID := r.PathValue("exerciseId")
	if exerciseID == "" {
		exerciseID = r.PathValue("id")
	}
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Exercise ID required", "ID del ejercicio requerido")
		return
	}

	var req dto.AssignExerciseModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	err := h.service.AssignExerciseModule(r.Context(), exerciseID, req.ModuleID)
	if err != nil {
		if errors.Is(err, domain.ErrModuleNotFound) {
			SendError(w, http.StatusNotFound, "MODULE_NOT_FOUND", "Módulo no encontrado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al asignar módulo a ejercicio")
		return
	}

	SendJSON(w, http.StatusOK, map[string]any{"exercise_id": exerciseID, "module_id": req.ModuleID}, "Ejercicio asignado al módulo exitosamente")
}

func (h *CourseModuleHandler) GetStudentCurricularMap(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		if ctxTenantID, ok := r.Context().Value(domain.TenantIDKey).(string); ok && ctxTenantID != "" {
			tenantID = ctxTenantID
		}
	}
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "Tenant ID missing in context", "Tenant no identificado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		if ctxUserID, ok := r.Context().Value(domain.UserIDKey).(string); ok && ctxUserID != "" {
			userID = ctxUserID
		}
	}
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "User ID missing in request", "Usuario no autenticado")
		return
	}

	courseID := r.PathValue("courseId")
	if courseID == "" {
		courseID = r.PathValue("id")
	}
	if courseID == "" {
		SendError(w, http.StatusBadRequest, "Course ID required", "ID del curso requerido")
		return
	}

	curMap, err := h.service.GetStudentCurricularMap(r.Context(), tenantID, courseID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener mapa curricular")
		return
	}

	SendJSON(w, http.StatusOK, curMap, "Mapa curricular obtenido exitosamente")
}
