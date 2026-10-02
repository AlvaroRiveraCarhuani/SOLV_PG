package httpdelivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type TeacherHandler struct {
	service     *services.TeacherService
	authService *services.AuthService
	upgrader    websocket.Upgrader
}

func NewTeacherHandler(service *services.TeacherService) *TeacherHandler {
	return &TeacherHandler{
		service:  service,
		upgrader: defaultUpgrader,
	}
}

func (h *TeacherHandler) SetAuthService(authService *services.AuthService) {
	h.authService = authService
}


func (h *TeacherHandler) GetCourses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	teacherID := r.Header.Get("X-User-Id")

	courses, err := h.service.GetCoursesSummary(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener cursos del docente")
		return
	}

	SendJSON(w, http.StatusOK, courses, "Cursos del docente obtenidos exitosamente")
}

func (h *TeacherHandler) GetAttention(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	teacherID := r.Header.Get("X-User-Id")

	widget, err := h.service.GetAttentionWidget(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener widget de atencion")
		return
	}

	SendJSON(w, http.StatusOK, widget, "Alertas de atencion obtenidas exitosamente")
}

func (h *TeacherHandler) GetCourseLabs(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	stats, err := h.service.GetCourseLabsStats(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener estadisticas de laboratorios")
		return
	}

	SendJSON(w, http.StatusOK, stats, "Estadisticas de laboratorios obtenidas exitosamente")
}

func (h *TeacherHandler) GetCourseSubmissions(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")
	exerciseID := r.URL.Query().Get("exercise_id")
	verdict := r.URL.Query().Get("verdict")

	items, err := h.service.ListCourseSubmissions(r.Context(), tenantID, teacherID, subjectID, exerciseID, verdict)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener cola de entregas")
		return
	}

	SendJSON(w, http.StatusOK, items, "Cola de entregas obtenida exitosamente")
}

func (h *TeacherHandler) GetSubmissionReview(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: los estudiantes no pueden acceder a la vista de revisión de casos privados")
		return
	}

	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	review, err := h.service.GetTeacherSubmissionReview(r.Context(), tenantID, teacherID, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe o no pertenece a una materia del docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener detalle de revision de entrega")
		return
	}

	SendJSON(w, http.StatusOK, review, "Detalle de revisión SpeedGrader obtenido exitosamente")
}

func (h *TeacherHandler) AddSubmissionComment(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden agregar comentarios de feedback")
		return
	}

	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto domain.AddCommentRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON", "Cuerpo de solicitud inválido")
		return
	}

	if strings.TrimSpace(dto.Comment) == "" {
		SendError(w, http.StatusBadRequest, "Comment required", "El contenido del comentario es requerido")
		return
	}

	authorID := r.Header.Get("X-User-Id")
	if authorID == "" {
		authorID = domain.DefaultTenantID
	}

	comment := &domain.SubmissionComment{
		TenantID:     tenantID,
		SubmissionID: submissionID,
		AuthorID:     authorID,
		LineNumber:   dto.LineNumber,
		Comment:      dto.Comment,
	}

	if err := h.service.AddComment(r.Context(), comment); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar comentario")
		return
	}

	SendJSON(w, http.StatusCreated, comment, "Comentario registrado exitosamente")
}

func (h *TeacherHandler) GetSubmissionComments(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	comments, err := h.service.GetCommentsBySubmission(r.Context(), tenantID, submissionID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener comentarios")
		return
	}

	SendJSON(w, http.StatusOK, comments, "Comentarios obtenidos exitosamente")
}

func (h *TeacherHandler) RunEphemeral(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden ejecutar el runner efímero")
		return
	}

	tenantID := getTenantFromCtx(r)

	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	var dto domain.EphemeralRunRequestDTO
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&dto)
	}

	teacherID := r.Header.Get("X-User-Id")

	result, err := h.service.RunEphemeral(r.Context(), tenantID, teacherID, submissionID, dto.Code, dto.Language)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error en ejecución efímera")
		return
	}

	SendJSON(w, http.StatusOK, result, "Ejecución efímera completada exitosamente")
}

func (h *TeacherHandler) ExportGrades(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes pueden exportar calificaciones")
		return
	}

	tenantID := getTenantFromCtx(r)

	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	teacherID := r.Header.Get("X-User-Id")

	csvData, filename, err := h.service.ExportCourseGradesCSV(r.Context(), tenantID, teacherID, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe o no pertenece al docente")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al exportar calificaciones")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvData)
}

func (h *TeacherHandler) AnalyzePlagiarism(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	subjectID := r.URL.Query().Get("subject_id")
	if subjectID == "" {
		subjectID = r.PathValue("id")
	}
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "Missing subject ID", "El identificador de la materia es requerido")
		return
	}

	exerciseID := r.URL.Query().Get("exercise_id")

	report, err := h.service.AnalyzePlagiarism(r.Context(), tenantID, teacherID, subjectID, exerciseID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Subject not found", "La materia solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar análisis de plagio")
		return
	}

	SendJSON(w, http.StatusOK, report, "Análisis de similitud estructural AST completado exitosamente")
}

func (h *TeacherHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")
	submissionID := r.PathValue("id")
	if submissionID == "" {
		SendError(w, http.StatusBadRequest, "Missing submission ID", "El identificador de la entrega es requerido")
		return
	}

	timeline, err := h.service.GetSubmissionTimeline(r.Context(), tenantID, teacherID, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "Submission not found", "La entrega solicitada no existe")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener timeline de la entrega")
		return
	}

	SendJSON(w, http.StatusOK, timeline, "Telemetría de Time-Travel Replay obtenida exitosamente")
}

func (h *TeacherHandler) GetLiveSessions(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	sessions, err := h.service.ListLiveSessions(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al listar sesiones de estudiantes en vivo")
		return
	}

	SendJSON(w, http.StatusOK, sessions, "Sesiones en vivo obtenidas exitosamente")
}

func (h *TeacherHandler) PostTutorExec(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	containerID := r.PathValue("id")
	if containerID == "" {
		SendError(w, http.StatusBadRequest, "Missing container ID", "El identificador del contenedor es requerido")
		return
	}

	var req domain.TutorCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "El cuerpo de la solicitud no es válido")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	resp, err := h.service.ExecuteTutorCommand(r.Context(), tenantID, teacherID, containerID, req.Command)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al ejecutar comando tutor en el contenedor")
		return
	}

	SendJSON(w, http.StatusOK, resp, "Comando ejecutado exitosamente en el contenedor")
}

func (h *TeacherHandler) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		containerID = r.URL.Query().Get("container_id")
	}

	var tokenStr string
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		tokenStr = qToken
	} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	} else if cookie, err := r.Cookie("solv_session"); err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}

	userID := "anonymous"
	tenantID := domain.DefaultTenantID

	if tokenStr != "" && h.authService != nil {
		claims, err := h.authService.ValidateSessionToken(tokenStr)
		if err != nil {
			SendError(w, http.StatusUnauthorized, "invalid session token", "Token de sesión inválido para WebSocket")
			return
		}
		if uID, ok := claims["user_id"].(string); ok && uID != "" {
			userID = uID
		}
		if tID, ok := claims["tenant_id"].(string); ok && tID != "" {
			tenantID = tID
		}
	} else {
		if hUID := r.Header.Get("X-User-Id"); hUID != "" {
			userID = hUID
		}
		if hTID := r.Header.Get("X-Tenant-Id"); hTID != "" {
			tenantID = hTID
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TerminalMirror] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 1. Enviar frame inicial de bienvenida y buffer reciente
	initBuffer, _ := h.service.GetInitialTerminalBuffer(r.Context(), containerID, 100)

	welcomePayload := map[string]interface{}{
		"type":         "init",
		"container_id": containerID,
		"user_id":      userID,
		"tenant_id":    tenantID,
		"buffer":       initBuffer,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"mode":         "SHADOW_MIRROR",
	}
	_ = conn.WriteJSON(welcomePayload)

	// 2. Loop de lectura para mensajes del docente (ping, resize, exec)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[TerminalMirror] Client closed: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			var incoming struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			}
			if err := json.Unmarshal(message, &incoming); err == nil {
				if incoming.Type == "ping" {
					_ = conn.WriteJSON(map[string]string{"type": "pong"})
				} else if incoming.Type == "exec" && incoming.Command != "" {
					resp, execErr := h.service.ExecuteTutorCommand(r.Context(), tenantID, userID, containerID, incoming.Command)
					if execErr != nil {
						_ = conn.WriteJSON(map[string]interface{}{
							"type":  "error",
							"error": execErr.Error(),
						})
					} else {
						_ = conn.WriteJSON(map[string]interface{}{
							"type":   "stdout",
							"output": fmt.Sprintf("\n%s\n", resp.Output),
						})
					}
				}
			}
		}
	}
}


