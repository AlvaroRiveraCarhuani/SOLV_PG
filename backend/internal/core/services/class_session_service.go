package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// ClassSessionService gobierna el aula: abrirla, cerrarla, mostrar quién está y
// quién falta, y recibir la señal de presencia del alumno.
//
// El aula es opcional por diseño: una práctica puede trabajarse sin sesión y el
// panel no se rompe. Lo que agrega la sesión es el conjunto contra el que se
// puede contar — faltantes, sin empezar, trabados.
type ClassSessionService struct {
	store domain.ClassSessionStore
	// examFreezer congela los volúmenes cuando el aula cerrada es un examen.
	// Es opcional: sin él, el aula se cierra igual y el barrido lo resuelve
	// después.
	examFreezer ExamFreezer
}

// ExamFreezer es la capacidad que el aula consume al cerrar un examen y al
// reabrir a un alumno puntual.
type ExamFreezer interface {
	FreezeSession(ctx context.Context, tenantID, teacherID, role, sessionID string) (FreezeReport, error)
	UnfreezeStudent(ctx context.Context, tenantID, teacherID, role, sessionID, studentID, reason string) (FreezeReport, error)
}

func NewClassSessionService(store domain.ClassSessionStore) *ClassSessionService {
	return &ClassSessionService{store: store}
}

// WithExamFreezer habilita el congelamiento al cerrar. Sin esta inyección el
// cierre funciona como antes y los volúmenes los congela el barrido.
func (s *ClassSessionService) WithExamFreezer(freezer ExamFreezer) *ClassSessionService {
	s.examFreezer = freezer
	return s
}

// teacherScope devuelve el filtro de propiedad del aula: el docente solo opera
// sus materias; un administrador no se limita por docente asignado.
func teacherScope(role, userID string) string {
	if strings.TrimSpace(role) == "admin" {
		return ""
	}
	return strings.TrimSpace(userID)
}

// OpenAula abre el aula de una materia con una duración. Dos aulas abiertas en la
// misma materia son un conflicto, no una segunda fila.
func (s *ClassSessionService) OpenAula(ctx context.Context, tenantID, teacherID, role, subjectID string, exerciseID *string, durationMinutes *int) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(subjectID)); err != nil {
		return nil, domain.ErrNotFound
	}
	if exerciseID != nil && strings.TrimSpace(*exerciseID) != "" {
		if _, err := uuid.Parse(strings.TrimSpace(*exerciseID)); err != nil {
			return nil, domain.ErrNotFound
		}
	}

	duration := domain.DefaultAulaDurationMinutes
	if durationMinutes != nil {
		duration = *durationMinutes
	}
	if duration < domain.MinAulaDurationMinutes || duration > domain.MaxAulaDurationMinutes {
		return nil, domain.ErrSessionDurationInvalid
	}

	return s.store.OpenSession(ctx, tenantID, teacherScope(role, teacherID), subjectID, exerciseID, duration)
}

// CloseAula cierra el aula. Cerrar dos veces devuelve el mismo estado.
//
// Devuelve además el conteo del congelamiento para que el cierre no se dé por
// hecho: el aula reporta cuántos volúmenes quedaron en solo lectura y cuántos
// fallaron (ADR-005 §5.5).
func (s *ClassSessionService) CloseAula(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSession, FreezeReport, error) {
	var report FreezeReport
	if strings.TrimSpace(tenantID) == "" {
		return nil, report, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, report, domain.ErrSessionNotFound
	}
	session, err := s.store.CloseSession(ctx, tenantID, teacherScope(role, teacherID), sessionID)
	if err != nil {
		return nil, report, err
	}

	// El examen se cierra con los volúmenes en solo lectura. Un aula de clase
	// devuelve "no es examen" y no pasa nada. Si el congelamiento falla, el
	// cierre ya ocurrió: el barrido lo reintenta y el aula no se reabre.
	if s.examFreezer != nil && session != nil {
		frozen, freezeErr := s.examFreezer.FreezeSession(ctx, tenantID, teacherID, role, sessionID)
		report = frozen
		if freezeErr != nil && !errFreezeIgnored(freezeErr) {
			log.Printf("[Exam Freeze] el aula %s se cerró pero no se pudo congelar: %v", sessionID, freezeErr)
		}
	}
	return session, report, nil
}

// FreezeExam congela los volúmenes de un examen ya cerrado. Es la acción que
// se puede repetir: los que ya estaban en solo lectura cuentan como omitidos.
func (s *ClassSessionService) FreezeExam(ctx context.Context, tenantID, teacherID, role, sessionID string) (FreezeReport, error) {
	if s.examFreezer == nil {
		return FreezeReport{}, domain.ErrExamFreezeNotAllowed
	}
	return s.examFreezer.FreezeSession(ctx, tenantID, teacherID, role, sessionID)
}

// UnfreezeExamStudent devuelve la escritura a un alumno. Es la excepción
// individual del docente y exige motivo.
func (s *ClassSessionService) UnfreezeExamStudent(ctx context.Context, tenantID, teacherID, role, sessionID, studentID, reason string) (FreezeReport, error) {
	if s.examFreezer == nil {
		return FreezeReport{}, domain.ErrExamFreezeNotAllowed
	}
	return s.examFreezer.UnfreezeStudent(ctx, tenantID, teacherID, role, sessionID, studentID, reason)
}

// CurrentAula devuelve el aula abierta del docente, o nil si no hay ninguna.
func (s *ClassSessionService) CurrentAula(ctx context.Context, tenantID, teacherID, role string) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	return s.store.GetOpenSessionForTeacher(ctx, tenantID, teacherScope(role, teacherID))
}

// LastClosedAula devuelve el aula cerrada más reciente del docente, o nil. La
// pantalla la usa al recargar para retomar el historial y la ventana.
func (s *ClassSessionService) LastClosedAula(ctx context.Context, tenantID, teacherID, role string) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	return s.store.GetLastClosedSessionForTeacher(ctx, tenantID, teacherScope(role, teacherID))
}

// Board resuelve el aula completa: sesión, marcador y alumnos por urgencia.
func (s *ClassSessionService) Board(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSessionBoard, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, domain.ErrSessionNotFound
	}
	return s.store.Board(ctx, tenantID, teacherScope(role, teacherID), sessionID)
}

// StudentAulas lista las aulas abiertas de las materias del alumno. El cliente
// las sondea para saber si tiene que emitir su señal de presencia.
func (s *ClassSessionService) StudentAulas(ctx context.Context, tenantID, studentID string) ([]domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(studentID)); err != nil {
		return nil, domain.ErrNotFound
	}
	aulas, err := s.store.OpenSessionsForStudent(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener las aulas abiertas del alumno: %w", err)
	}
	if aulas == nil {
		aulas = make([]domain.ClassSession, 0)
	}
	return aulas, nil
}

// Heartbeat registra la presencia del alumno en el aula. Es la señal que
// convierte una inscripción en una fila "dentro" del tablero.
func (s *ClassSessionService) Heartbeat(ctx context.Context, tenantID, studentID, sessionID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(studentID)); err != nil {
		return domain.ErrNotFound
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return domain.ErrSessionNotFound
	}
	return s.store.TouchParticipant(ctx, tenantID, studentID, sessionID)
}
