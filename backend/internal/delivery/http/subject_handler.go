package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/middleware"
)

type SubjectHandler struct {
	service *services.SubjectService
}

func NewSubjectHandler(service *services.SubjectService) *SubjectHandler {
	return &SubjectHandler{service: service}
}

func (h *SubjectHandler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Name              string  `json:"name"`
		Code              string  `json:"code"`
		TeacherID         *string `json:"teacher_id,omitempty"`
		AcademicPeriodID  *string `json:"academic_period_id,omitempty"`
		ClassroomCourseID *string `json:"classroom_course_id,omitempty"`
		TemplateID        *string `json:"template_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	subject, err := h.service.CreateSubjectWithDetails(r.Context(), tenantID, req.Name, req.Code, req.TeacherID, req.AcademicPeriodID, req.ClassroomCourseID, req.TemplateID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(subject)
}

func (h *SubjectHandler) ArchiveSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error":"Subject ID required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		IsArchived bool `json:"is_archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.ArchiveSubject(r.Context(), tenantID, id, req.IsArchived); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"is_archived": req.IsArchived})
}

func (h *SubjectHandler) UpdateSubject(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error":"Subject ID required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateSubject(r.Context(), tenantID, id, req.Name, req.Code); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "updated",
		"message": "Materia actualizada exitosamente",
	})
}

func (h *SubjectHandler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	list, err := h.service.ListSubjects(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":"Failed to list subjects"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": list,
	})
}

func (h *SubjectHandler) EnrollStudent(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	subjectID := r.PathValue("id")
	if subjectID == "" {
		http.Error(w, `{"error":"Subject ID missing"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		StudentID string `json:"student_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	enrollment, err := h.service.EnrollStudent(r.Context(), tenantID, req.StudentID, subjectID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(enrollment)
}

func (h *SubjectHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	subjectID := r.PathValue("id")
	if subjectID == "" {
		http.Error(w, `{"error":"Subject ID missing"}`, http.StatusBadRequest)
		return
	}

	students, err := h.service.ListStudents(r.Context(), tenantID, subjectID)
	if err != nil {
		http.Error(w, `{"error":"Failed to list enrolled students"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subject_id": subjectID,
		"students":   students,
	})
}
