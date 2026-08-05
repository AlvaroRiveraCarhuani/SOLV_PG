package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/delivery/http/middleware"
)

type ClassroomHandler struct{}

func NewClassroomHandler() *ClassroomHandler {
	return &ClassroomHandler{}
}

func (h *ClassroomHandler) ImportRosterManual(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == "" {
		http.Error(w, `{"error":"Tenant ID missing in context"}`, http.StatusUnauthorized)
		return
	}

	courseID := r.URL.Query().Get("course_id")
	if courseID == "" {
		http.Error(w, `{"error":"Query parameter course_id is required"}`, http.StatusBadRequest)
		return
	}

	// Simulación estructurada según D6 (Importación manual unidireccional)
	importedData := map[string]interface{}{
		"tenant_id":        tenantID,
		"course_id":        courseID,
		"sync_type":        "unidirectional_manual_import",
		"imported_students": []map[string]string{
			{"email": "alumno1.classroom@uab.edu.bo", "name": "Alumno Classroom 1", "status": "imported"},
			{"email": "alumno2.classroom@uab.edu.bo", "name": "Alumno Classroom 2", "status": "imported"},
		},
		"message": "Manual roster import executed successfully (D6 compliant)",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(importedData)
}
