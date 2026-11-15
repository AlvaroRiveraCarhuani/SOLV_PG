package httpdelivery

import "net/http"

type ClassroomHandler struct{}

func NewClassroomHandler() *ClassroomHandler {
	return &ClassroomHandler{}
}

// ImportRosterManual responde 501 a propósito.
//
// Antes devolvía dos alumnos inventados con un mensaje de éxito. Eso es peor que
// no tener el endpoint: el docente creía haber importado la nómina, la materia
// seguía vacía y nadie se enteraba. La importación real de una nómina existe en
// POST /api/v1/admin/subjects/{id}/roster, y la sincronización con Classroom
// todavía no está construida (ver ADR-022).
func (h *ClassroomHandler) ImportRosterManual(w http.ResponseWriter, r *http.Request) {
	SendError(
		w,
		http.StatusNotImplemented,
		"classroom_import_not_implemented",
		"La importación desde Classroom no está implementada. Cargá la nómina desde la materia.",
	)
}
