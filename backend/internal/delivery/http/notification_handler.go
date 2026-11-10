package httpdelivery

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type NotificationHandler struct {
	service *services.NotificationService
}

func NewNotificationHandler(service *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	userID := getUserIDFromCtx(r)

	filter := domain.NotificationListFilter{
		UnreadOnly: r.URL.Query().Get("unread_only") == "true",
		Severity:   strings.TrimSpace(r.URL.Query().Get("severity")),
		EventType:  strings.TrimSpace(r.URL.Query().Get("event_type")),
	}
	if filter.Severity != "" && !domain.IsValidNotificationSeverity(filter.Severity) {
		SendError(w, http.StatusBadRequest, "invalid_severity", "La severidad debe ser info, warning, error o critical")
		return
	}
	if filter.EventType != "" && !domain.IsValidNotificationEventType(filter.EventType) {
		SendError(w, http.StatusBadRequest, "invalid_event_type", "El tipo de evento no tiene un formato válido")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	items, total, err := h.service.List(r.Context(), tenantID, userID, filter, page, limit)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener notificaciones")
		return
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	SendJSON(w, http.StatusOK, items, "Notificaciones obtenidas exitosamente")
}

func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	userID := getUserIDFromCtx(r)

	res, err := h.service.GetUnreadCount(r.Context(), tenantID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener contador de notificaciones no leídas")
		return
	}

	SendJSON(w, http.StatusOK, res, "Contador de notificaciones no leídas obtenido exitosamente")
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	userID := getUserIDFromCtx(r)
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de notificación requerido")
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_id", "id de notificación inválido")
		return
	}

	err := h.service.MarkRead(r.Context(), tenantID, userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SendError(w, http.StatusNotFound, "not_found", "Notificación no encontrada o no pertenece al usuario")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al marcar notificación como leída")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "marked_as_read", "id": id}, "Notificación marcada como leída")
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)
	userID := getUserIDFromCtx(r)

	res, err := h.service.MarkAllRead(r.Context(), tenantID, userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al marcar todas las notificaciones como leídas")
		return
	}

	SendJSON(w, http.StatusOK, res, "Todas las notificaciones fueron marcadas como leídas")
}
