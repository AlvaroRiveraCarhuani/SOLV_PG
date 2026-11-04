package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

type TeacherService struct {
	repo                   domain.TeacherRepository
	subRepo                domain.SubmissionRepository
	keystrokeRepo          domain.KeystrokeRepository
	evalService            *EvaluationService
	mirrorService          *TerminalMirrorService
	metricsSnapshotService *MetricsSnapshotService
	mailer                 domain.Mailer
	tenantRepo             domain.TenantRepository
	reviewStore            domain.TeacherReviewStore
	overviewStore          domain.TeacherOverviewStore
	classSessionStore      domain.ClassSessionStore
	auditRepo              domain.AuditLogRepository
}

func NewTeacherService(
	repo domain.TeacherRepository,
	subRepo domain.SubmissionRepository,
) *TeacherService {
	svc := &TeacherService{
		repo:          repo,
		subRepo:       subRepo,
		mirrorService: NewTerminalMirrorService(nil),
	}
	// El repositorio de revisión se declara en el consumidor: si la
	// implementación lo soporta, se activa sin cambiar el wiring existente.
	if store, ok := repo.(domain.TeacherReviewStore); ok {
		svc.reviewStore = store
	}
	if store, ok := repo.(domain.TeacherOverviewStore); ok {
		svc.overviewStore = store
	}
	return svc
}

func (s *TeacherService) SetReviewStore(store domain.TeacherReviewStore) {
	s.reviewStore = store
}

func (s *TeacherService) SetAuditRepository(auditRepo domain.AuditLogRepository) {
	s.auditRepo = auditRepo
}

// SetClassSessionStore habilita la fila de aula del panel. Sin este almacén el
// resumen sigue funcionando y simplemente no muestra aula abierta.
func (s *TeacherService) SetClassSessionStore(store domain.ClassSessionStore) {
	s.classSessionStore = store
}

func (s *TeacherService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

func (s *TeacherService) SetPracticeEnvironmentOrchestrator(orch domain.PracticeEnvironmentOrchestrator) {
	s.mirrorService = NewTerminalMirrorService(orch)
}

func (s *TeacherService) SetMetricsSnapshotService(metricsSnapshotService *MetricsSnapshotService) {
	s.metricsSnapshotService = metricsSnapshotService
}

func (s *TeacherService) SetEvaluationService(evalService *EvaluationService) {
	s.evalService = evalService
}

func (s *TeacherService) SetMailer(m domain.Mailer) {
	s.mailer = m
}

func (s *TeacherService) SetTenantRepository(tr domain.TenantRepository) {
	s.tenantRepo = tr
}

func (s *TeacherService) GetCoursesSummary(ctx context.Context, tenantID, teacherID, periodID string) ([]*domain.TeacherCourseSummary, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	// Validar period_id si viene vacío se omite el filtro
	// Si viene no vacío, validar que sea un UUID válido
	if periodID != "" {
		if _, err := uuid.Parse(periodID); err != nil {
			return nil, domain.ErrInvalidTenant
		}
	}

	courses, err := s.repo.GetCoursesSummary(ctx, tenantID, teacherID, periodID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener resumen de cursos: %w", err)
	}
	if courses == nil {
		courses = make([]*domain.TeacherCourseSummary, 0)
	}
	return courses, nil
}

// GetTeacherOverview devuelve el read model del panel con los conteos resueltos.
// Es una sola respuesta: la vista no suma ni agrupa nada.
func (s *TeacherService) GetTeacherOverview(ctx context.Context, tenantID, teacherID, periodID string) (*domain.TeacherOverview, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if periodID != "" {
		if _, err := uuid.Parse(periodID); err != nil {
			return nil, domain.ErrInvalidTenant
		}
	}
	if s.overviewStore == nil {
		return nil, fmt.Errorf("el repositorio del docente no implementa el resumen del panel")
	}

	overview, err := s.overviewStore.GetTeacherOverview(ctx, tenantID, teacherID, periodID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener el resumen del panel docente: %w", err)
	}
	if overview == nil {
		return nil, domain.ErrNotFound
	}

	// La fila de aula del panel sale de la misma sesión que usa la pantalla de
	// aula: dos superficies, un solo hecho. Un fallo aquí no tumba el resumen.
	if s.classSessionStore != nil {
		session, err := s.classSessionStore.GetOpenSessionForTeacher(ctx, tenantID, teacherID)
		if err != nil {
			return nil, fmt.Errorf("error al obtener el aula abierta del panel: %w", err)
		}
		if session != nil {
			overview.Session = &domain.TeacherOverviewSession{
				ID:            session.ID,
				SubjectID:     session.SubjectID,
				SubjectCode:   session.SubjectCode,
				SubjectName:   session.SubjectName,
				ExerciseTitle: derefString(session.ExerciseTitle),
				OpensAt:       session.OpensAt,
				EndsAt:        session.EndsAt,
				ClosesAt:      session.ClosesAt,
				Enrolled:      session.Enrolled,
				Present:       session.Present,
			}
		}
	}

	return overview, nil
}

// derefString devuelve el valor del puntero o cadena vacía. Existe para no
// arrastrar punteros al contrato JSON del panel.
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *TeacherService) GetCourseLabsStats(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.TeacherLabStats, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}
	stats, err := s.repo.GetCourseLabsStats(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = make([]*domain.TeacherLabStats, 0)
	}
	return stats, nil
}

// GetTeacherExerciseBank devuelve el banco de ejercicios del docente con
// semántica de revisión por entorno (juez vs IDE).
func (s *TeacherService) GetTeacherExerciseBank(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherExerciseBankItem, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	items, err := s.repo.GetTeacherExerciseBank(ctx, tenantID, teacherID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]*domain.TeacherExerciseBankItem, 0)
	}
	return items, nil
}

// DuplicateExercise copia un laboratorio del docente a otra materia suya. La
// copia nace en borrador, sin entregas ni marca de validación del original. La
// persistencia pasa por EvaluationService.CreateExercise para reusar validaciones
// y casos de prueba.
func (s *TeacherService) DuplicateExercise(ctx context.Context, tenantID, teacherID, exerciseID, targetSubjectID string) (*domain.Exercise, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if exerciseID == "" || targetSubjectID == "" {
		return nil, domain.ErrNotFound
	}
	if s.evalService == nil {
		return nil, fmt.Errorf("el servicio de evaluación no está configurado")
	}

	source, err := s.evalService.GetExerciseByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, err
	}
	if source == nil || source.SubjectID == nil {
		return nil, domain.ErrNotFound
	}

	// Propiedad: GetCourseLabsStats devuelve ErrNotFound si la materia no es del docente.
	if _, err := s.repo.GetCourseLabsStats(ctx, tenantID, teacherID, *source.SubjectID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetCourseLabsStats(ctx, tenantID, teacherID, targetSubjectID); err != nil {
		return nil, err
	}

	copied := *source
	copied.ID = uuid.NewString()
	copied.SubjectID = &targetSubjectID
	copied.Status = "draft"
	copied.Stale = false
	copied.LastValidDryRunAt = nil
	copied.TenantID = tenantID
	copied.Tags = append(copied.Tags[:0:0], source.Tags...)
	if source.Config.Algorithm != nil {
		algo := *source.Config.Algorithm
		algo.TestCases = append([]domain.TestCase(nil), source.Config.Algorithm.TestCases...)
		copied.Config.Algorithm = &algo
	}

	if err := s.evalService.CreateExercise(ctx, &copied); err != nil {
		return nil, err
	}
	return &copied, nil
}

func (s *TeacherService) GetLivePracticeEnvironmentSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LivePracticeEnvironmentSession, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	sessions, err := s.repo.ListLivePracticeEnvironmentSessions(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener sesiones en vivo: %w", err)
	}
	if sessions == nil {
		sessions = make([]*domain.LivePracticeEnvironmentSession, 0)
	}
	return sessions, nil
}
