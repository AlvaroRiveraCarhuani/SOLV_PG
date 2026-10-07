package httpdelivery

import (
	"encoding/json"
	"fmt"
	"net/http"

	"solv-backend/internal/core/services"
)

type TeacherInvitationHandler struct {
	service *services.TeacherInvitationService
}

func NewTeacherInvitationHandler(service *services.TeacherInvitationService) *TeacherInvitationHandler {
	return &TeacherInvitationHandler{service: service}
}

func (h *TeacherInvitationHandler) ListTeachers(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	origin := r.URL.Query().Get("origin")

	teachers, err := h.service.ListTeachers(r.Context(), tenantID, search, status, origin)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener la lista de docentes")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":    teachers,
		"total":   len(teachers),
		"message": "Docentes obtenidos exitosamente",
	})
}

func (h *TeacherInvitationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	var req struct {
		Email         string `json:"email"`
		Origin        string `json:"origin"`
		RoleType      string `json:"role_type"`
		DurationHours int    `json:"duration_hours"`
		SendEmail     bool   `json:"send_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request payload", "Cuerpo de solicitud inválido")
		return
	}

	if req.DurationHours <= 0 {
		req.DurationHours = 72 // ADR-025: TTL 72h
	}
	if req.RoleType == "" {
		req.RoleType = "titular"
	}
	if req.Origin == "" {
		req.Origin = "manual"
	}

	inv, err := h.service.CreateInvitation(r.Context(), tenantID, req.Email, req.Origin, req.RoleType, req.DurationHours)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al emitir la invitación para el docente")
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	inviteURL := fmt.Sprintf("%s://%s/login?invite=%s", scheme, host, inv.Token)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         inv.ID,
		"email":      inv.Email,
		"token":      inv.Token,
		"invite_url": inviteURL,
		"role_type":  inv.RoleType,
		"origin":     inv.Origin,
		"expires_at": inv.ExpiresAt,
		"message":    "Invitación emitida exitosamente (válida por 72h)",
	})
}

func (h *TeacherInvitationHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	invID := r.PathValue("id")
	if invID == "" {
		SendError(w, http.StatusBadRequest, "Missing invitation ID", "Identificador de invitación requerido")
		return
	}

	inv, err := h.service.ResendInvitation(r.Context(), tenantID, invID)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al reenviar la invitación")
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	inviteURL := fmt.Sprintf("%s://%s/login?invite=%s", scheme, host, inv.Token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         inv.ID,
		"email":      inv.Email,
		"token":      inv.Token,
		"invite_url": inviteURL,
		"expires_at": inv.ExpiresAt,
		"message":    "Invitación reenviada exitosamente",
	})
}

func (h *TeacherInvitationHandler) RenewInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	invID := r.PathValue("id")
	if invID == "" {
		SendError(w, http.StatusBadRequest, "Missing invitation ID", "Identificador de invitación requerido")
		return
	}

	inv, err := h.service.RenewInvitation(r.Context(), tenantID, invID)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al renovar la invitación")
		return
	}

	host := r.Host
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	inviteURL := fmt.Sprintf("%s://%s/login?invite=%s", scheme, host, inv.Token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         inv.ID,
		"email":      inv.Email,
		"token":      inv.Token,
		"invite_url": inviteURL,
		"expires_at": inv.ExpiresAt,
		"message":    "Invitación renovada por 72h con nuevo token transaccional",
	})
}

func (h *TeacherInvitationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	userID := r.Header.Get("X-User-Id")
	userEmail := r.Header.Get("X-User-Email")

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request payload", "Cuerpo de solicitud inválido")
		return
	}

	if err := h.service.AcceptInvitation(r.Context(), tenantID, req.Token, userID, userEmail); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "No se pudo aceptar la invitación")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "Rol actualizado a docente exitosamente",
	})
}

func (h *TeacherInvitationHandler) GetTeacherCourses(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	teacherID := r.PathValue("id")
	if teacherID == "" {
		SendError(w, http.StatusBadRequest, "Missing teacher ID", "Identificador de docente requerido")
		return
	}

	courses, err := h.service.GetTeacherCourses(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener materias del docente")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":    courses,
		"total":   len(courses),
		"message": "Materias del docente obtenidas exitosamente",
	})
}

func (h *TeacherInvitationHandler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	invID := r.PathValue("id")
	if invID == "" {
		SendError(w, http.StatusBadRequest, "Missing invitation ID", "Identificador de invitación requerido")
		return
	}

	if err := h.service.DeleteInvitation(r.Context(), tenantID, invID); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "No se pudo revocar la invitación")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "deleted",
		"message": "Invitación revocada exitosamente",
	})
}

func (h *TeacherInvitationHandler) DeleteTeacher(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	teacherID := r.PathValue("id")
	if teacherID == "" {
		SendError(w, http.StatusBadRequest, "Missing teacher ID", "Identificador de docente requerido")
		return
	}

	if err := h.service.DeleteTeacher(r.Context(), tenantID, teacherID); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "No se pudo dar de baja al docente")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "deleted",
		"message": "Docente dado de baja exitosamente",
	})
}
