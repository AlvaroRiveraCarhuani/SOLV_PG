package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"solv-backend/internal/core/domain"
	"strings"
)

// ListAdministrators devuelve todos los operadores con rol admin del tenant actual.
func (h *AdminAcademicHandler) ListAdministrators(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: se requiere rol de administrador")
		return
	}

	tenantID := getTenantFromCtx(r)
	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	admins, err := h.govService.ListAdministrators(r.Context(), tenantID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener lista de administradores")
		return
	}

	if admins == nil {
		admins = []*domain.AdminUserItem{}
	}

	SendJSON(w, http.StatusOK, admins, "Administradores obtenidos exitosamente")
}

// CreateAdministrator procesa el alta o promoción de un usuario con privilegios administrativos.
func (h *AdminAcademicHandler) CreateAdministrator(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: se requiere rol de administrador")
		return
	}

	tenantID := getTenantFromCtx(r)
	actorID := getUserIDFromCtx(r)

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	var dto domain.CreateAdminDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "Cuerpo de solicitud inválido")
		return
	}

	created, err := h.govService.CreateAdministrator(r.Context(), tenantID, actorID, dto)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidAdminEmail):
			SendError(w, http.StatusBadRequest, "invalid_email", "El correo institucional ingresado no es válido")
		case errors.Is(err, domain.ErrAdminNamesRequired):
			SendError(w, http.StatusBadRequest, "names_required", "Nombre y apellido son campos obligatorios")
		default:
			SendError(w, http.StatusInternalServerError, err.Error(), "Error al registrar administrador")
		}
		return
	}

	SendJSON(w, http.StatusCreated, created, "Administrador pre-aprovisionado exitosamente")
}

// RevokeAdministrator procesa la revocación de privilegios administrativos de un usuario.
func (h *AdminAcademicHandler) RevokeAdministrator(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "admin" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: se requiere rol de administrador")
		return
	}

	targetUserID := strings.TrimSpace(r.PathValue("id"))
	if targetUserID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "Identificador de administrador requerido")
		return
	}

	tenantID := getTenantFromCtx(r)
	actorID := getUserIDFromCtx(r)

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	err := h.govService.RevokeAdministrator(r.Context(), tenantID, targetUserID, actorID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCannotRevokeSelf):
			SendError(w, http.StatusUnprocessableEntity, "cannot_revoke_self", "No es posible revocar los propios privilegios de administrador")
		case errors.Is(err, domain.ErrLastAdministrator):
			SendError(w, http.StatusUnprocessableEntity, "last_administrator", "No se puede revocar al único administrador activo de la institución")
		case errors.Is(err, domain.ErrAdminNotFound):
			SendError(w, http.StatusNotFound, "admin_not_found", "El administrador especificado no existe o no tiene rol administrativo")
		default:
			SendError(w, http.StatusInternalServerError, err.Error(), "Error al revocar administrador")
		}
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{
		"id":     targetUserID,
		"status": "revoked",
	}, "Privilegios de administrador revocados exitosamente")
}
