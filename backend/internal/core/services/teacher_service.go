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
	return svc
}

func (s *TeacherService) SetReviewStore(store domain.TeacherReviewStore) {
	s.reviewStore = store
}

func (s *TeacherService) SetAuditRepository(auditRepo domain.AuditLogRepository) {
	s.auditRepo = auditRepo
}

func (s *TeacherService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

func (s *TeacherService) SetWorkspaceOrchestrator(orch domain.WorkspaceOrchestrator) {
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

func (s *TeacherService) GetAttentionWidget(ctx context.Context, tenantID, teacherID string) (*domain.TeacherAttentionWidget, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	widget, err := s.repo.GetAttentionWidget(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener widget de atencion: %w", err)
	}
	return widget, nil
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

func (s *TeacherService) GetLiveWorkspaceSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LiveWorkspaceSession, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	sessions, err := s.repo.ListLiveWorkspaceSessions(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener sesiones en vivo: %w", err)
	}
	if sessions == nil {
		sessions = make([]*domain.LiveWorkspaceSession, 0)
	}
	return sessions, nil
}
