package httpdelivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"strings"
)

// -----------------------------------------------------------------------------
// Docker Templates Governance (ADR-030)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) GetRuntimeCapabilities(w http.ResponseWriter, r *http.Request) {
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	caps, err := h.govService.GetRuntimeCapabilities(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener capacidades de ejecución")
		return
	}

	SendJSON(w, http.StatusOK, caps, "Capacidades de ejecución obtenidas exitosamente")
}

func (h *AdminAcademicHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores y docentes pueden consultar plantillas")
		return
	}

	tenantID := getTenantFromCtx(r)
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	templates, err := h.govService.ListTemplates(r.Context(), tenantID, status, search)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener listado de plantillas")
		return
	}

	SendJSON(w, http.StatusOK, templates, "Listado de plantillas obtenido exitosamente")
}

func (h *AdminAcademicHandler) ReviewTemplate(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden revisar plantillas")
		return
	}

	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de plantilla requerido")
		return
	}

	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}

	var dto domain.ReviewTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	item, err := h.govService.ReviewTemplate(r.Context(), tenantID, templateID, adminID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrRamExceedsHostCapacity) {
			SendError(w, http.StatusBadRequest, "ram_exceeds_host", err.Error())
			return
		}
		if errors.Is(err, services.ErrInvalidReviewStatus) || errors.Is(err, services.ErrRejectionReasonRequired) {
			SendError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "Plantilla no encontrada")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al revisar la plantilla")
		return
	}

	if h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"template_id":      item.ID,
			"status":           dto.Status,
			"rejection_reason": dto.RejectionReason,
		})
		ipAddr := r.RemoteAddr
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ipAddr = host
		}
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_REVIEWED",
			ResourceType: "lab_template",
			ResourceID:   &item.ID,
			StatusCode:   http.StatusOK,
			Metadata:     meta,
			IPAddress:    ipAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusOK, item, "Plantilla revisada exitosamente")
}

func (h *AdminAcademicHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden registrar plantillas oficiales")
		return
	}

	tenantID := getTenantFromCtx(r)
	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}

	var dto domain.CreateOfficialTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if dto.Name == "" || dto.DockerImage == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "Nombre e imagen Docker son requeridos")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	item, err := h.govService.CreateOfficialTemplate(r.Context(), tenantID, adminID, dto)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNameConflict) {
			SendError(w, http.StatusConflict, "template_exists", "Ya existe una plantilla con este nombre")
			return
		}
		if errors.Is(err, domain.ErrRamExceedsHostCapacity) {
			SendError(w, http.StatusBadRequest, "ram_exceeds_host", err.Error())
			return
		}
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidDockerImage) || errors.Is(err, services.ErrLatestTagForbidden) || strings.Contains(err.Error(), "requerido") {
			status = http.StatusUnprocessableEntity
		}
		SendError(w, status, err.Error(), err.Error())
		return
	}

	if h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"template_id": item.ID,
			"name":        item.Name,
			"image":       item.DockerImage,
		})
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_REGISTERED",
			ResourceType: "lab_template",
			ResourceID:   &item.ID,
			StatusCode:   http.StatusCreated,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusCreated, item, "Plantilla oficial registrada exitosamente")
}

// GetTemplatesCacheStatus retorna el estado de presencia en disco de todas las plantillas activas
func (h *AdminAcademicHandler) GetTemplatesCacheStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	if h.cacheService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "servicio de calentamiento de caché no configurado")
		return
	}

	statuses, err := h.cacheService.GetTemplatesCacheStatus(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	SendJSON(w, http.StatusOK, statuses, "Estados de caché obtenidos exitosamente")
}

// WarmTemplate precalienta una plantilla individual
func (h *AdminAcademicHandler) WarmTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "invalid_id", "ID de plantilla requerido")
		return
	}

	if h.cacheService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "servicio de calentamiento de caché no configurado")
		return
	}

	status, err := h.cacheService.WarmTemplate(r.Context(), tenantID, templateID)
	if err != nil {
		if errors.Is(err, services.ErrTemplateNotFound) {
			SendError(w, http.StatusNotFound, "template_not_found", "plantilla no encontrada")
			return
		}
		SendError(w, http.StatusInternalServerError, "warming_failed", err.Error())
		return
	}

	SendJSON(w, http.StatusOK, status, "Plantilla precalentada exitosamente")
}

// WarmAllTemplates precalienta secuencialmente todas las plantillas activas que no se encuentren en disco
func (h *AdminAcademicHandler) WarmAllTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	if h.cacheService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "servicio de calentamiento de caché no configurado")
		return
	}

	summary, err := h.cacheService.WarmAllActiveTemplates(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "warming_failed", err.Error())
		return
	}

	// Notificación proactiva in-app al administrador
	if h.notificationService != nil {
		adminID := getUserIDFromCtx(r)
		if adminID == "" {
			adminID = r.Header.Get("X-User-Id")
		}
		if adminID != "" {
			msg := fmt.Sprintf("Entornos preparados: %d descargados, %d ya listos en el servidor.", summary.WarmedCount, summary.AlreadyWarm)
			if summary.FailedCount > 0 {
				msg += fmt.Sprintf(" %d presentaron error.", summary.FailedCount)
			}
			h.notificationService.NotifyAsync(tenantID, domain.CreateNotificationDTO{
				RecipientUserID: adminID,
				Title:           "Preparación de entornos completada",
				Message:         msg,
				Severity:        domain.NotificationSeverityInfo,
				Channel:         domain.NotificationChannelInApp,
				EventType:       "TEMPLATE_WARMING_COMPLETED",
				Link:            "/admin/plantillas",
			})
		}
	}

	SendJSON(w, http.StatusOK, summary, "Entornos preparados exitosamente")
}

func (h *AdminAcademicHandler) DuplicateTemplate(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden duplicar plantillas")
		return
	}

	tenantID := getTenantFromCtx(r)
	templateID := r.PathValue("id")
	if templateID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de plantilla requerido")
		return
	}

	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	item, err := h.govService.DuplicateTemplate(r.Context(), tenantID, templateID, adminID)
	if err != nil {
		if strings.Contains(err.Error(), "no encontrada") {
			SendError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al duplicar la plantilla")
		return
	}

	if h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"source_template_id": templateID,
			"duplicated_id":      item.ID,
			"name":               item.Name,
		})
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_DUPLICATED",
			ResourceType: "lab_template",
			ResourceID:   &item.ID,
			StatusCode:   http.StatusCreated,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusCreated, item, "Plantilla duplicada exitosamente en cola de auditoría")
}

// -----------------------------------------------------------------------------
// Emergency Actions (ADR-032)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ExecuteEmergencyAction(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden ejecutar acciones de emergencia")
		return
	}

	tenantID := getTenantFromCtx(r)
	action := r.PathValue("action")
	if action == "" {
		SendError(w, http.StatusBadRequest, "missing_action", "Acción de emergencia requerida")
		return
	}

	adminID := r.Header.Get("X-User-Id")
	if adminID == "" {
		adminID = domain.DefaultTenantID
	}

	var req domain.EmergencyActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	result, err := h.govService.ExecuteEmergencyAction(r.Context(), tenantID, adminID, action, req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidConfirmationPhrase) {
			SendError(w, http.StatusUnprocessableEntity, "invalid_confirmation_phrase", "La frase de confirmación no coincide exactamente")
			return
		}
		if errors.Is(err, services.ErrUnknownEmergencyAction) {
			SendError(w, http.StatusUnprocessableEntity, "unknown_action", "Acción de emergencia no reconocida")
			return
		}
		if errors.Is(err, services.ErrEmergencyExecutorUnavailable) {
			SendError(w, http.StatusServiceUnavailable, "executor_unavailable", "El executor de esta acción de emergencia no está disponible en esta instancia")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar acción de emergencia")
		return
	}

	SendJSON(w, http.StatusOK, result, result.Message)
}

// -----------------------------------------------------------------------------
// Image Verification & Local Images (ADR-030)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListLocalImages(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "Servicio de verificación de imágenes no disponible")
		return
	}

	images, err := h.imageService.ListLocalImages(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, "docker_error", err.Error())
		return
	}

	tenantID := getTenantFromCtx(r)
	var usageMap map[string]int
	if h.govService != nil {
		usageMap, _ = h.govService.GetImageUsageCounts(r.Context(), tenantID)
	}
	if usageMap == nil {
		usageMap = make(map[string]int)
	}

	for i := range images {
		images[i].UsageCount = usageMap[images[i].RepoTag]
	}

	result := domain.LocalImagesResult{
		Images:   images,
		UsageMap: usageMap,
	}

	SendJSON(w, http.StatusOK, result, "Imágenes locales recuperadas exitosamente")
}

func (h *AdminAcademicHandler) VerifyImage(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "Servicio de verificación de imágenes no disponible")
		return
	}

	var req domain.VerifyImageRequest
	if r.Method == http.MethodGet {
		req.Image = r.URL.Query().Get("image")
		req.Force = r.URL.Query().Get("force") == "true"
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
			return
		}
	}

	if strings.TrimSpace(req.Image) == "" {
		SendError(w, http.StatusUnprocessableEntity, "validation_failed", "El campo image es obligatorio")
		return
	}

	result, err := h.imageService.VerifyImage(r.Context(), req.Image, req.Force)
	if err != nil {
		if errors.Is(err, services.ErrImageFormatInvalid) || errors.Is(err, services.ErrLatestTagForbiddenVerif) || errors.Is(err, services.ErrRegistryNotAllowed) {
			SendError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, "verification_failed", err.Error())
		return
	}

	// Registro de auditoría si se utilizó bypass manual force=true (SEC-04)
	if req.Force && h.auditLogRepo != nil {
		meta, _ := json.Marshal(map[string]any{
			"image_ref": req.Image,
			"bypass":    "force_verification",
			"reason":    "Bypass manual de verificación de imagen Docker",
		})
		adminID := r.Header.Get("X-User-Id")
		if adminID == "" {
			adminID = domain.DefaultTenantID
		}
		_ = h.auditLogRepo.Create(r.Context(), &domain.AuditLog{
			TenantID:     getTenantFromCtx(r),
			ActorID:      adminID,
			Action:       "IMAGE_VERIFICATION_FORCE_BYPASS",
			ResourceType: "docker_image",
			StatusCode:   http.StatusOK,
			Metadata:     meta,
			IPAddress:    r.RemoteAddr,
			UserAgent:    r.UserAgent(),
		})
	}

	SendJSON(w, http.StatusOK, result, "Verificación de imagen completada")
}
