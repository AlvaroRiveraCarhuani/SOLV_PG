package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// CourseRosterHandler expone el padrón de una materia. Es la misma superficie
// para administración y docencia: el servicio decide si el actor puede operar
// sobre esa materia, y el origen de la matrícula sale del rol.
type CourseRosterHandler struct {
	service *services.CourseRosterService
}

func NewCourseRosterHandler(service *services.CourseRosterService) *CourseRosterHandler {
	return &CourseRosterHandler{service: service}
}

func (h *CourseRosterHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, subjectID, role, userID, ok := courseRosterScope(w, r)
	if !ok {
		return
	}

	students, err := h.service.List(r.Context(), tenantID, subjectID, role, userID)
	if err != nil {
		sendCourseRosterError(w, err, "Error al obtener los estudiantes de la materia")
		return
	}
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"subject_id": subjectID,
		"students":   students,
	}, "Estudiantes de la materia obtenidos exitosamente")
}

func (h *CourseRosterHandler) Available(w http.ResponseWriter, r *http.Request) {
	tenantID, subjectID, role, userID, ok := courseRosterScope(w, r)
	if !ok {
		return
	}

	students, err := h.service.Available(r.Context(), tenantID, subjectID, role, userID, r.URL.Query().Get("q"))
	if err != nil {
		sendCourseRosterError(w, err, "Error al buscar estudiantes disponibles")
		return
	}
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"subject_id": subjectID,
		"students":   students,
	}, "Estudiantes disponibles obtenidos exitosamente")
}

func (h *CourseRosterHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	tenantID, subjectID, role, userID, ok := courseRosterScope(w, r)
	if !ok {
		return
	}

	var req struct {
		StudentID string `json:"student_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "Cuerpo de solicitud inválido")
		return
	}

	student, err := h.service.Enroll(r.Context(), tenantID, subjectID, role, userID, strings.TrimSpace(req.StudentID))
	if err != nil {
		sendCourseRosterError(w, err, "Error al matricular al estudiante")
		return
	}
	SendJSON(w, http.StatusCreated, student, "Estudiante matriculado exitosamente")
}

func (h *CourseRosterHandler) Remove(w http.ResponseWriter, r *http.Request) {
	tenantID, subjectID, role, userID, ok := courseRosterScope(w, r)
	if !ok {
		return
	}

	studentID := r.PathValue("studentId")
	if err := h.service.Remove(r.Context(), tenantID, subjectID, role, userID, studentID); err != nil {
		sendCourseRosterError(w, err, "Error al dar de baja al estudiante")
		return
	}
	SendJSON(w, http.StatusOK, map[string]string{
		"subject_id": subjectID,
		"student_id": studentID,
	}, "Estudiante dado de baja de la materia")
}

// ParseRoster lee la planilla y devuelve los correos normalizados sin tocar la
// base. El panel reutiliza este resultado antes de confirmar la nómina.
func (h *CourseRosterHandler) ParseRoster(w http.ResponseWriter, r *http.Request) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: solo docentes y administradores")
		return
	}
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

	parsed, err := services.NewRosterParser().Parse(file, header.Filename)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al analizar el contenido del archivo")
		return
	}
	SendJSON(w, http.StatusOK, parsed, "Planilla procesada exitosamente")
}

// Roster aplica la nómina completa: crea las cuentas que falten y matricula.
// Acepta multipart (archivo) o JSON con correos.
func (h *CourseRosterHandler) Roster(w http.ResponseWriter, r *http.Request) {
	tenantID, subjectID, role, userID, ok := courseRosterScope(w, r)
	if !ok {
		return
	}

	var (
		entries      []domain.RosterEntry
		duplicates   []string
		invalid      []string
		sourceSystem string
		reconcile    bool
	)

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
		duplicates = parsed.DuplicateEmails
		invalid = parsed.InvalidEmails
		sourceSystem = strings.TrimSpace(r.FormValue("source_system"))
		reconcile = isAffirmative(r.FormValue("reconcile"))
		entries = make([]domain.RosterEntry, 0, len(parsed.ValidEmails))
		for _, email := range parsed.ValidEmails {
			entries = append(entries, domain.RosterEntry{Email: email})
		}
	} else {
		var req struct {
			Emails       []string             `json:"emails"`
			Entries      []domain.RosterEntry `json:"entries"`
			SourceSystem string               `json:"source_system"`
			Reconcile    bool                 `json:"reconcile"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "invalid_payload", "Cuerpo de solicitud inválido")
			return
		}
		sourceSystem = req.SourceSystem
		reconcile = req.Reconcile
		entries = req.Entries
		if len(entries) == 0 {
			entries = make([]domain.RosterEntry, 0, len(req.Emails))
			for _, email := range req.Emails {
				entries = append(entries, domain.RosterEntry{Email: email})
			}
		}
	}

	if len(entries) == 0 {
		SendError(w, http.StatusBadRequest, "empty_emails", "No se encontraron correos válidos en la nómina")
		return
	}

	result, err := h.service.Roster(r.Context(), tenantID, subjectID, role, userID, entries, sourceSystem, reconcile)
	if err != nil {
		sendCourseRosterError(w, err, "Error al procesar la nómina del curso")
		return
	}
	result.DuplicateEmails = duplicates
	result.InvalidEmails = invalid
	SendJSON(w, http.StatusCreated, result, "Nómina procesada exitosamente")
}

func courseRosterScope(w http.ResponseWriter, r *http.Request) (tenantID, subjectID, role, userID string, ok bool) {
	if !hasAnyRole(r, "teacher", "admin") {
		SendError(w, http.StatusForbidden, "forbidden", "Acceso denegado: solo docentes y administradores pueden gestionar el padrón")
		return "", "", "", "", false
	}
	tenantID = getTenantFromCtx(r)
	if tenantID == "" {
		SendError(w, http.StatusUnauthorized, "tenant_required", "No se pudo resolver la institución de la sesión")
		return "", "", "", "", false
	}
	subjectID = r.PathValue("id")
	if subjectID == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "El identificador de la materia es requerido")
		return "", "", "", "", false
	}
	role = requestRole(r)
	if role == "" {
		role = domain.GetUserRole(r.Context())
	}
	userID = requestUserID(r)
	return tenantID, subjectID, role, userID, true
}

func sendCourseRosterError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, domain.ErrCourseForbidden):
		SendError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, "not_found", "La materia o el estudiante no existen en la institución")
	default:
		SendError(w, http.StatusBadRequest, "invalid_request", fallback)
	}
}
