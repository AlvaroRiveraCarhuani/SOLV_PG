package httpdelivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"strings"
)

// -----------------------------------------------------------------------------
// Student Directory & Reset OOM (ADR-033)
// -----------------------------------------------------------------------------

func (h *AdminAcademicHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: rol student no autorizado")
		return
	}

	tenantID := getTenantFromCtx(r)
	search := r.URL.Query().Get("search")
	subjectID := r.URL.Query().Get("subject_id")
	status := r.URL.Query().Get("status")
	periodID := r.URL.Query().Get("period_id")

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	students, err := h.govService.ListStudents(r.Context(), tenantID, search, subjectID, status, periodID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener directorio de estudiantes")
		return
	}

	if students == nil {
		students = []*domain.AdminStudentDirectoryItem{}
	}

	SendJSON(w, http.StatusOK, students, "Directorio de estudiantes obtenido exitosamente")
}

func (h *AdminAcademicHandler) GetStudentCourses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: rol student no autorizado")
		return
	}

	tenantID := getTenantFromCtx(r)
	studentID := r.PathValue("id")
	if studentID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de estudiante requerido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	courses, err := h.govService.GetStudentCourses(r.Context(), tenantID, studentID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener materias del estudiante")
		return
	}

	if courses == nil {
		courses = []*domain.AdminStudentCourseItem{}
	}

	SendJSON(w, http.StatusOK, courses, "Materias del estudiante obtenidas exitosamente")
}

func (h *AdminAcademicHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden registrar estudiantes")
		return
	}

	tenantID := getTenantFromCtx(r)
	var dto domain.CreateStudentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	student, err := h.govService.CreateStudent(r.Context(), tenantID, dto)
	if err != nil {
		if strings.Contains(err.Error(), "already_exists") {
			SendError(w, http.StatusConflict, "already_exists", "Ya existe un estudiante registrado con ese correo institucional")
			return
		}
		SendError(w, http.StatusBadRequest, err.Error(), err.Error())
		return
	}

	SendJSON(w, http.StatusCreated, student, "Estudiante registrado exitosamente")
}

func (h *AdminAcademicHandler) UpdateStudentStatus(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden suspender o reactivar cuentas")
		return
	}

	tenantID := getTenantFromCtx(r)
	studentID := r.PathValue("id")
	if studentID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de estudiante requerido")
		return
	}

	var dto domain.UpdateStudentStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	if err := h.govService.UpdateStudentStatus(r.Context(), tenantID, studentID, dto); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al actualizar estado del estudiante")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"id": studentID, "status": dto.Status}, "Estado del estudiante actualizado exitosamente")
}

func (h *AdminAcademicHandler) ResetStudentOOM(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo administradores pueden resetear penalizaciones OOM")
		return
	}

	tenantID := getTenantFromCtx(r)
	studentID := r.PathValue("id")
	if studentID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de estudiante requerido")
		return
	}

	var dto domain.ResetOOMDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	if len(dto.Reason) < 10 {
		SendError(w, http.StatusUnprocessableEntity, "reason_too_short", "El motivo de justificación debe contener al menos 10 caracteres")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	result, err := h.govService.ResetStudentOOM(r.Context(), tenantID, studentID, dto)
	if err != nil {
		if errors.Is(err, services.ErrReasonTooShort) {
			SendError(w, http.StatusUnprocessableEntity, "reason_too_short", "El motivo de justificación debe contener al menos 10 caracteres")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "not_found", "Estudiante no encontrado en el tenant")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al resetear penalizaciones OOM")
		return
	}

	SendJSON(w, http.StatusOK, result, "Penalizaciones OOM reseteadas exitosamente")
}

// requireAdmin es un guard de rol compartido: solo administradores pueden
// ejecutar operaciones de nómina y matrícula masiva.
func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if role := r.Header.Get("X-User-Role"); role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: se requiere rol administrador")
		return false
	}
	return true
}

// ParseStudentRoster analiza un archivo de nómina (CSV/XLSX) y extrae los
// correos válidos, duplicados e inválidos (mismo contrato que docentes).
func (h *AdminAcademicHandler) ParseStudentRoster(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// Límite explícito de 10 MB para parseo de nómina en formulario multipart.
	//nolint:gosec
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar el archivo subido")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		SendError(w, http.StatusBadRequest, "missing_file", "Archivo de nómina requerido")
		return
	}
	defer func() { _ = file.Close() }()

	result, err := services.NewRosterParser().Parse(file, header.Filename)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al analizar el contenido del archivo")
		return
	}

	SendJSON(w, http.StatusOK, result, "Nómina de estudiantes analizada exitosamente")
}

// BatchCreateStudents da de alta en lote cuentas de estudiante por correo.
func (h *AdminAcademicHandler) BatchCreateStudents(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	tenantID := getTenantFromCtx(r)

	var req struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "Cuerpo de solicitud inválido")
		return
	}
	if len(req.Emails) == 0 {
		SendError(w, http.StatusBadRequest, "empty_emails", "Debe incluir al menos un correo")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	count, err := h.govService.BatchCreateStudents(r.Context(), tenantID, req.Emails)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar el alta masiva")
		return
	}

	SendJSON(w, http.StatusCreated, domain.BatchCreateStudentsResult{
		TotalProcessed: len(req.Emails),
		CreatedCount:    count,
	}, fmt.Sprintf("%d estudiantes dados de alta exitosamente", count))
}

// EnrollStudentsBatch matricula en lote a estudiantes existentes en una materia.
func (h *AdminAcademicHandler) EnrollStudentsBatch(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	tenantID := getTenantFromCtx(r)
	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de materia requerido")
		return
	}

	var req struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "Cuerpo de solicitud inválido")
		return
	}
	if len(req.Emails) == 0 {
		SendError(w, http.StatusBadRequest, "empty_emails", "Debe incluir al menos un correo")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	enrolled, skipped, err := h.govService.EnrollStudentsByEmails(r.Context(), tenantID, subjectID, req.Emails)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar la matrícula masiva")
		return
	}

	SendJSON(w, http.StatusCreated, map[string]interface{}{
		"enrolled_count":     enrolled,
		"skipped_not_found":  skipped,
		"total_processed":    len(req.Emails),
	}, fmt.Sprintf("%d estudiantes matriculados exitosamente", enrolled))
}

// RosterEnroll es la operación combinada de nómina de curso: crea las cuentas
// faltantes y matricula a todos los correos válidos en la materia. Acepta el
// archivo (multipart "file") o un JSON con el arreglo de correos.
func (h *AdminAcademicHandler) RosterEnroll(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	tenantID := getTenantFromCtx(r)
	subjectID := r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "id de materia requerido")
		return
	}

	if h.govService == nil {
		SendError(w, http.StatusInternalServerError, "service_unavailable", "Servicio de gobernanza no configurado")
		return
	}

	var validEmails, duplicateEmails, invalidEmails []string

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		//nolint:gosec
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar el archivo subido")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			SendError(w, http.StatusBadRequest, "missing_file", "Archivo de nómina requerido")
			return
		}
		parsed, err := services.NewRosterParser().Parse(file, header.Filename)
		_ = file.Close()
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error(), "Error al analizar el contenido del archivo")
			return
		}
		validEmails = parsed.ValidEmails
		duplicateEmails = parsed.DuplicateEmails
		invalidEmails = parsed.InvalidEmails
	} else {
		var req struct {
			Emails []string `json:"emails"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "invalid_payload", "Cuerpo de solicitud inválido")
			return
		}
		validEmails = req.Emails
	}

	if len(validEmails) == 0 {
		SendError(w, http.StatusBadRequest, "empty_emails", "No se encontraron correos válidos en la nómina")
		return
	}

	result, err := h.govService.RosterEnroll(r.Context(), tenantID, subjectID, validEmails)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar la nómina del curso")
		return
	}

	result.DuplicateEmails = duplicateEmails
	result.InvalidEmails = invalidEmails
	SendJSON(w, http.StatusCreated, result, "Nómina procesada exitosamente")
}
