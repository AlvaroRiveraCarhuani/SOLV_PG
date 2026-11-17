package services

import (
	"context"
	"errors"
	"strings"

	"solv-backend/internal/core/domain"
)

// CourseRosterService gobierna el padrón de una materia para los dos roles que
// lo operan: el administrador (cualquier materia del tenant) y el docente
// titular (solo las suyas). La matrícula nace marcada con su origen para que
// la reconciliación posterior de la nómina oficial no borre altas de cátedra.
type CourseRosterService struct {
	repo   domain.CourseRosterRepository
	syncer domain.RosterSyncer
}

func NewCourseRosterService(repo domain.CourseRosterRepository, syncer domain.RosterSyncer) *CourseRosterService {
	return &CourseRosterService{repo: repo, syncer: syncer}
}

// authorize valida la materia y, cuando el actor es docente, que la dicte.
func (s *CourseRosterService) authorize(ctx context.Context, tenantID, subjectID, role, userID string) error {
	exists, err := s.repo.SubjectExists(ctx, tenantID, subjectID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}
	if role == "admin" {
		return nil
	}
	owns, err := s.repo.TeacherOwnsSubject(ctx, tenantID, subjectID, userID)
	if err != nil {
		return err
	}
	if !owns {
		return domain.ErrCourseForbidden
	}
	return nil
}

func (s *CourseRosterService) List(ctx context.Context, tenantID, subjectID, role, userID string) ([]*domain.CourseStudentItem, error) {
	if err := s.authorize(ctx, tenantID, subjectID, role, userID); err != nil {
		return nil, err
	}
	return s.repo.ListCourseStudents(ctx, tenantID, subjectID)
}

func (s *CourseRosterService) Available(ctx context.Context, tenantID, subjectID, role, userID, query string) ([]*domain.CourseAvailableStudent, error) {
	if err := s.authorize(ctx, tenantID, subjectID, role, userID); err != nil {
		return nil, err
	}
	return s.repo.ListAvailableStudents(ctx, tenantID, subjectID, query, 20)
}

// Enroll da de alta una matrícula individual. El origen se deriva del rol que
// la ejecuta: una alta del docente es de cátedra y una del administrador se
// considera institucional.
func (s *CourseRosterService) Enroll(ctx context.Context, tenantID, subjectID, role, userID, studentID string) (*domain.CourseStudentItem, error) {
	if strings.TrimSpace(studentID) == "" {
		return nil, errors.New("el identificador del estudiante es obligatorio")
	}
	if err := s.authorize(ctx, tenantID, subjectID, role, userID); err != nil {
		return nil, err
	}
	source := domain.EnrollmentSourceInstitutional
	if role == "teacher" {
		source = domain.EnrollmentSourceTeacherManual
	}
	if err := s.repo.EnrollStudentWithSource(ctx, tenantID, subjectID, studentID, source); err != nil {
		return nil, err
	}
	students, err := s.repo.ListCourseStudents(ctx, tenantID, subjectID)
	if err != nil {
		// La matrícula ya quedó escrita. Un fallo de lectura posterior no puede
		// presentarse como un alta fallida: eso invita a reintentar y duplica
		// altas de distintos alumnos.
		return &domain.CourseStudentItem{StudentID: studentID, SourceSystem: source}, nil
	}
	for _, item := range students {
		if item.StudentID == studentID {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *CourseRosterService) Remove(ctx context.Context, tenantID, subjectID, role, userID, studentID string) error {
	if strings.TrimSpace(studentID) == "" {
		return errors.New("el identificador del estudiante es obligatorio")
	}
	if err := s.authorize(ctx, tenantID, subjectID, role, userID); err != nil {
		return err
	}
	return s.repo.RemoveStudentFromSubject(ctx, tenantID, subjectID, studentID)
}

// Roster aplica una nómina completa. El docente nunca reconcilia: su planilla
// agrega altas de cátedra, pero no es la fuente que decide quién sigue
// matriculado en la materia.
func (s *CourseRosterService) Roster(
	ctx context.Context,
	tenantID, subjectID, role, userID string,
	entries []domain.RosterEntry,
	requestedSource string,
	reconcile bool,
) (*domain.RosterEnrollResult, error) {
	if err := s.authorize(ctx, tenantID, subjectID, role, userID); err != nil {
		return nil, err
	}
	if s.syncer == nil {
		return nil, errors.New("la sincronización de nómina no está disponible")
	}

	source := strings.TrimSpace(requestedSource)
	if role == "teacher" {
		source = domain.EnrollmentSourceTeacherManual
		reconcile = false
	} else if source == "" {
		source = domain.EnrollmentSourceInstitutional
	}

	return s.syncer.RosterEnrollFromSource(ctx, tenantID, subjectID, domain.RosterSyncRequest{
		Entries:      entries,
		SourceSystem: source,
		Reconcile:    reconcile,
	})
}
