package services

import (
	"context"
	"fmt"
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
}

func NewClassSessionService(store domain.ClassSessionStore) *ClassSessionService {
	return &ClassSessionService{store: store}
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
func (s *ClassSessionService) CloseAula(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, domain.ErrSessionNotFound
	}
	return s.store.CloseSession(ctx, tenantID, teacherScope(role, teacherID), sessionID)
}

// CurrentAula devuelve el aula abierta del docente, o nil si no hay ninguna.
func (s *ClassSessionService) CurrentAula(ctx context.Context, tenantID, teacherID, role string) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	return s.store.GetOpenSessionForTeacher(ctx, tenantID, teacherScope(role, teacherID))
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
