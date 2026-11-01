package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// FreezeReport cuenta qué pasó al congelar un examen. El cierre del aula no se
// da por hecho: si un alumno falló, queda en Failed y el barrido lo reintenta.
type FreezeReport struct {
	Frozen  int `json:"frozen"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

func (r *FreezeReport) merge(other FreezeReport) {
	r.Frozen += other.Frozen
	r.Skipped += other.Skipped
	r.Failed += other.Failed
}

func (r FreezeReport) empty() bool {
	return r.Frozen == 0 && r.Skipped == 0 && r.Failed == 0
}

// WorkspaceMountSetter es la capacidad del servicio de entornos que el
// congelamiento necesita: reconstruir el contenedor con el volumen en :ro o
// devolverlo a :rw.
type WorkspaceMountSetter interface {
	SetWorkspaceMount(ctx context.Context, instance *domain.WorkspaceInstance, readOnly bool) error
}

// ExamSessionReader es la lectura del aula que el congelamiento necesita.
type ExamSessionReader interface {
	GetSession(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSession, error)
}

// ExamWorkspaceReader resuelve el laboratorio por identificador.
type ExamWorkspaceReader interface {
	GetByID(ctx context.Context, id string) (*domain.WorkspaceInstance, error)
}

// ExamWorkspaceFreezer es la capacidad que la entrega anticipada necesita:
// cerrar un solo volumen sin tocar el resto de la clase.
type ExamWorkspaceFreezer interface {
	FreezeWorkspace(ctx context.Context, tenantID, actorID, workspaceID string) error
}

// ExamAuditWriter registra el congelamiento y la excepción.
type ExamAuditWriter interface {
	Create(ctx context.Context, log *domain.AuditLog) error
}

// ExamFreezeService cierra el examen sobre los volúmenes: al cerrar el aula,
// cada laboratorio de los participantes queda montado en solo lectura y el
// estado se guarda en la base (ADR-005 §5).
type ExamFreezeService struct {
	store      domain.ExamFreezeStore
	sessions   ExamSessionReader
	workspaces ExamWorkspaceReader
	mounts     WorkspaceMountSetter
	audit      ExamAuditWriter
	now        func() time.Time
}

func NewExamFreezeService(
	store domain.ExamFreezeStore,
	sessions ExamSessionReader,
	workspaces ExamWorkspaceReader,
	mounts WorkspaceMountSetter,
) *ExamFreezeService {
	return &ExamFreezeService{
		store:      store,
		sessions:   sessions,
		workspaces: workspaces,
		mounts:     mounts,
		now:        time.Now,
	}
}

// SetAuditRepository habilita el registro de cada congelamiento y de cada
// excepción. Sin él la operación funciona igual: la auditoría es un extra.
func (s *ExamFreezeService) SetAuditRepository(audit ExamAuditWriter) *ExamFreezeService {
	s.audit = audit
	return s
}

// WithClock fija el reloj para las pruebas.
func (s *ExamFreezeService) WithClock(now func() time.Time) *ExamFreezeService {
	if now != nil {
		s.now = now
	}
	return s
}

// FreezeSession congela los laboratorios de un examen ya cerrado. Es
// idempotente: los que ya estaban congelados cuentan como omitidos.
func (s *ExamFreezeService) FreezeSession(ctx context.Context, tenantID, teacherID, role, sessionID string) (FreezeReport, error) {
	var report FreezeReport
	if strings.TrimSpace(tenantID) == "" {
		return report, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return report, domain.ErrSessionNotFound
	}

	session, err := s.sessions.GetSession(ctx, tenantID, teacherScope(role, teacherID), sessionID)
	if err != nil {
		return report, err
	}
	isExam, err := s.store.SessionIsExam(ctx, tenantID, sessionID)
	if err != nil {
		return report, err
	}
	if !isExam {
		return report, domain.ErrExamFreezeNotAllowed
	}
	if session == nil || session.Status != domain.ClassSessionStatusClosed {
		return report, domain.ErrExamSessionOpen
	}

	actor := strings.TrimSpace(teacherID)
	if actor == "" {
		actor = session.CreatedBy
	}
	return s.freezeParticipants(ctx, tenantID, sessionID, actor)
}

// FreezeClosedExams recorre las aulas cerradas de examen que todavía tienen
// laboratorios editables. Cubre el cierre por vencimiento de `ends_at` y lo que
// quedó pendiente mientras el backend estaba detenido.
func (s *ExamFreezeService) FreezeClosedExams(ctx context.Context, limit int) (FreezeReport, error) {
	var report FreezeReport
	refs, err := s.store.ClosedExamSessions(ctx, limit)
	if err != nil {
		return report, err
	}
	for _, ref := range refs {
		partial, freezeErr := s.freezeParticipants(ctx, ref.TenantID, ref.SessionID, "")
		report.merge(partial)
		if freezeErr != nil {
			// Un aula que falla no frena el barrido de las demás.
			log.Printf("[Exam Freeze] no se pudo congelar el examen %s: %v", ref.SessionID, freezeErr)
		}
	}
	return report, nil
}

// UnfreezeStudent devuelve la escritura al laboratorio de un alumno. Es la
// excepción individual del docente y exige motivo: queda en la auditoría.
func (s *ExamFreezeService) UnfreezeStudent(ctx context.Context, tenantID, teacherID, role, sessionID, studentID, reason string) (FreezeReport, error) {
	var report FreezeReport
	if strings.TrimSpace(tenantID) == "" {
		return report, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return report, domain.ErrSessionNotFound
	}
	if strings.TrimSpace(reason) == "" {
		return report, domain.ErrFreezeReasonRequired
	}

	if _, err := s.sessions.GetSession(ctx, tenantID, teacherScope(role, teacherID), sessionID); err != nil {
		return report, err
	}
	isExam, err := s.store.SessionIsExam(ctx, tenantID, sessionID)
	if err != nil {
		return report, err
	}
	if !isExam {
		return report, domain.ErrExamFreezeNotAllowed
	}

	rows, err := s.store.ParticipantWorkspaces(ctx, tenantID, sessionID)
	if err != nil {
		return report, err
	}
	var target *domain.ExamFreezeWorkspace
	for i := range rows {
		if rows[i].StudentID == strings.TrimSpace(studentID) {
			target = &rows[i]
			break
		}
	}
	if target == nil {
		return report, domain.ErrFreezeStudentNotInSession
	}
	if !target.IsFrozen() {
		// Reabrir lo que ya está abierto no es un error: el docente pudo
		// pulsar dos veces.
		report.Skipped++
		return report, nil
	}

	instance, err := s.workspaces.GetByID(withTenant(ctx, tenantID), target.WorkspaceID)
	if err != nil {
		if isMissingRow(err) {
			return report, domain.ErrNotFound
		}
		return report, err
	}
	if instance == nil {
		return report, domain.ErrNotFound
	}
	if err := s.mounts.SetWorkspaceMount(ctx, instance, false); err != nil {
		report.Failed++
		return report, err
	}
	if err := s.store.SetWorkspaceFrozen(ctx, tenantID, target.WorkspaceID, false, s.now()); err != nil {
		report.Failed++
		return report, err
	}
	report.Frozen++

	reason = strings.TrimSpace(reason)
	s.recordAudit(ctx, tenantID, teacherID, domain.AuditExamWorkspaceUnfrozen, sessionID, map[string]any{
		"student_id":   target.StudentID,
		"workspace_id": target.WorkspaceID,
		"reason":       reason,
	})
	return report, nil
}

// FreezeWorkspace cierra el volumen de un alumno sin cerrar el aula. Es lo que
// pasa cuando entrega el examen antes: su entorno queda en solo lectura y el
// resto de la clase sigue escribiendo (ADR-005 §5.2).
func (s *ExamFreezeService) FreezeWorkspace(ctx context.Context, tenantID, actorID, workspaceID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return domain.ErrInvalidTenant
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if _, err := uuid.Parse(workspaceID); err != nil {
		return domain.ErrInvalidResourceID
	}

	instance, err := s.workspaces.GetByID(withTenant(ctx, tenantID), workspaceID)
	if err != nil {
		if isMissingRow(err) {
			return domain.ErrNotFound
		}
		return err
	}
	if instance == nil {
		return domain.ErrNotFound
	}
	if instance.FrozenAt != nil || instance.Status == domain.WorkspaceStatusFrozen {
		// Reabrir y volver a entregar deja el volumen en el mismo estado.
		return nil
	}

	if err := s.mounts.SetWorkspaceMount(ctx, instance, true); err != nil {
		return err
	}
	if err := s.store.SetWorkspaceFrozen(ctx, tenantID, workspaceID, true, s.now()); err != nil {
		// El contenedor ya quedó en solo lectura: el barrido del aula vuelve a
		// pasar y el montaje repetido no rompe nada.
		return err
	}
	s.recordWorkspaceAudit(ctx, tenantID, actorID, workspaceID)
	return nil
}

// recordWorkspaceAudit deja la entrega anticipada en la auditoría. El recurso es
// el laboratorio, no el aula: el examen sigue abierto para los demás.
func (s *ExamFreezeService) recordWorkspaceAudit(ctx context.Context, tenantID, actorID, workspaceID string) {
	if s.audit == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(actorID) == "" {
		return
	}
	raw, err := json.Marshal(map[string]any{"reason": "submission"})
	if err != nil {
		return
	}
	resourceID := workspaceID
	_ = s.audit.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       domain.AuditExamWorkspaceFrozen,
		ResourceType: "workspace",
		ResourceID:   &resourceID,
		StatusCode:   200,
		Metadata:     raw,
	})
}

// freezeParticipants aplica el montaje en solo lectura a cada laboratorio que
// todavía no lo tiene. Un fallo no aborta el resto.
func (s *ExamFreezeService) freezeParticipants(ctx context.Context, tenantID, sessionID, actor string) (FreezeReport, error) {
	var report FreezeReport
	rows, err := s.store.ParticipantWorkspaces(ctx, tenantID, sessionID)
	if err != nil {
		return report, err
	}

	for _, row := range rows {
		if row.IsFrozen() {
			report.Skipped++
			continue
		}
		instance, err := s.workspaces.GetByID(withTenant(ctx, tenantID), row.WorkspaceID)
		if err != nil || instance == nil {
			// El laboratorio ya no existe: no hay volumen que congelar y no es un
			// fallo del cierre. Solo una lectura rota se cuenta como fallo.
			if isMissingRow(err) {
				report.Skipped++
				continue
			}
			report.Failed++
			log.Printf("[Exam Freeze] no se pudo leer el laboratorio %s: %v", row.WorkspaceID, err)
			continue
		}
		if err := s.mounts.SetWorkspaceMount(ctx, instance, true); err != nil {
			report.Failed++
			log.Printf("[Exam Freeze] no se pudo montar en solo lectura el laboratorio %s: %v", row.WorkspaceID, err)
			continue
		}
		if err := s.store.SetWorkspaceFrozen(ctx, tenantID, row.WorkspaceID, true, s.now()); err != nil {
			// El contenedor ya quedó en solo lectura: el barrido vuelve a pasar
			// y el montaje repetido no rompe nada.
			report.Failed++
			log.Printf("[Exam Freeze] el laboratorio %s quedó en solo lectura pero no se pudo marcar: %v", row.WorkspaceID, err)
			continue
		}
		report.Frozen++

		s.recordAudit(ctx, tenantID, actor, domain.AuditExamWorkspaceFrozen, sessionID, map[string]any{
			"student_id":   row.StudentID,
			"workspace_id": row.WorkspaceID,
		})
	}
	return report, nil
}

func (s *ExamFreezeService) recordAudit(ctx context.Context, tenantID, actorID, action, sessionID string, metadata map[string]any) {
	if s.audit == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(actorID) == "" || strings.TrimSpace(sessionID) == "" {
		return
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["session_id"] = sessionID
	raw, err := json.Marshal(metadata)
	if err != nil {
		return
	}
	resourceID := sessionID
	_ = s.audit.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       action,
		ResourceType: "class_session",
		ResourceID:   &resourceID,
		StatusCode:   200,
		Metadata:     raw,
	})
}

// withTenant deja el tenant en el contexto para las lecturas que lo resuelven
// desde ahí, como el repositorio de laboratorios.
func withTenant(ctx context.Context, tenantID string) context.Context {
	if strings.TrimSpace(tenantID) == "" {
		return ctx
	}
	return context.WithValue(ctx, domain.TenantIDKey, tenantID)
}

// ExamFreezeWorker congela al arrancar y después cada 15 minutos. El primer
// ciclo recupera los exámenes que terminaron mientras el backend no estaba.
type ExamFreezeWorker struct {
	service  *ExamFreezeService
	interval time.Duration
	limit    int
}

func NewExamFreezeWorker(service *ExamFreezeService) *ExamFreezeWorker {
	return &ExamFreezeWorker{service: service, interval: 15 * time.Minute, limit: 50}
}

func (w *ExamFreezeWorker) WithInterval(interval time.Duration) *ExamFreezeWorker {
	if interval > 0 {
		w.interval = interval
	}
	return w
}

func (w *ExamFreezeWorker) Start(ctx context.Context) {
	if w.service == nil {
		return
	}
	log.Printf("[Exam Freeze] Worker started (interval=%v)", w.interval)
	w.SweepOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[Exam Freeze] Worker stopped.")
			return
		case <-ticker.C:
			w.SweepOnce(ctx)
		}
	}
}

// SweepOnce ejecuta un ciclo. Un error de consulta no tumba al worker.
func (w *ExamFreezeWorker) SweepOnce(ctx context.Context) {
	if w.service == nil {
		return
	}
	report, err := w.service.FreezeClosedExams(ctx, w.limit)
	if err != nil {
		log.Printf("[Exam Freeze] Error en el barrido: %v", err)
		return
	}
	if !report.empty() {
		log.Printf("[Exam Freeze] congelados=%d omitidos=%d fallos=%d", report.Frozen, report.Skipped, report.Failed)
	}
}

// errFreezeIgnored evita que un aula que no es examen ensucie el log del cierre.
func errFreezeIgnored(err error) bool {
	return errors.Is(err, domain.ErrExamFreezeNotAllowed) || errors.Is(err, domain.ErrExamSessionOpen)
}
