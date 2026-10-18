package httpdelivery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

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

	inviteURL := h.buildInviteURL(r, inv.Token)

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

	inviteURL := h.buildInviteURL(r, inv.Token)

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

	inviteURL := h.buildInviteURL(r, inv.Token)

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

func (h *TeacherInvitationHandler) buildInviteURL(r *http.Request, token string) string {
	// 1. Origen reportado por el frontend del navegador
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return fmt.Sprintf("%s/login?invite=%s", strings.TrimRight(origin, "/"), token)
	}

	// 2. Referer del cliente
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		if u, err := url.Parse(referer); err == nil && u.Host != "" {
			return fmt.Sprintf("%s://%s/login?invite=%s", u.Scheme, u.Host, token)
		}
	}

	// 3. Variable de entorno explícita para la URL base del frontend
	if fe := strings.TrimSpace(os.Getenv("FRONTEND_URL")); fe != "" {
		return fmt.Sprintf("%s/login?invite=%s", strings.TrimRight(fe, "/"), token)
	}

	// 4. En desarrollo local, Angular corre en el puerto 4200
	if strings.Contains(r.Host, "localhost") || strings.Contains(r.Host, "127.0.0.1") {
		return fmt.Sprintf("http://localhost:4200/login?invite=%s", token)
	}

	// 5. Host directo (producción con reverse proxy Traefik unificado)
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/login?invite=%s", scheme, r.Host, token)
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

func (h *TeacherInvitationHandler) RegisterDirectTeacher(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	var req struct {
		Email            string `json:"email"`
		FirstName        string `json:"first_name"`
		LastName         string `json:"last_name"`
		SendNotification bool   `json:"send_notification"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request payload", "Cuerpo de solicitud inválido")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		SendError(w, http.StatusBadRequest, "Email is required", "El correo institucional es obligatorio")
		return
	}

	loginURL := h.buildInviteURL(r, "")
	loginURL = strings.TrimSuffix(loginURL, "?invite=")

	teacher, err := h.service.RegisterDirectTeacher(r.Context(), tenantID, req.Email, req.FirstName, req.LastName, req.SendNotification, loginURL, "Universidad Adventista de Bolivia")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al registrar al docente")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":     teacher,
		"notified": req.SendNotification,
		"message":  "Docente registrado exitosamente con rol docente activo",
	})
}

func (h *TeacherInvitationHandler) ParseRoster(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar el archivo subido")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		SendError(w, http.StatusBadRequest, "Missing file parameter", "Archivo de nómina requerido")
		return
	}
	defer file.Close()

	result, err := h.service.ParseRoster(file, header.Filename)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al analizar el contenido del archivo")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TeacherInvitationHandler) BatchRegisterTeachers(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	var req struct {
		Emails           []string `json:"emails"`
		SendNotification bool     `json:"send_notification"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request payload", "Cuerpo de solicitud inválido")
		return
	}
	if len(req.Emails) == 0 {
		SendError(w, http.StatusBadRequest, "Empty emails list", "Debe incluir al menos un correo")
		return
	}

	loginURL := h.buildInviteURL(r, "")
	loginURL = strings.TrimSuffix(loginURL, "?invite=")

	count, err := h.service.BatchRegisterTeachers(r.Context(), tenantID, req.Emails, req.SendNotification, loginURL, "Universidad Adventista de Bolivia")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al procesar el alta masiva")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_processed":  len(req.Emails),
		"registered_count": count,
		"notified":         req.SendNotification,
		"message":          fmt.Sprintf("%d docentes dados de alta exitosamente", count),
	})
}
