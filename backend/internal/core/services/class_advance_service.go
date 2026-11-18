package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// ClassAdvanceService gobierna los avances voluntarios del docente en el aula
// (ADR-058). Emitir, congelar y extender ventana son acciones del docente; la
// aplicación en el entorno del alumno vive en otro servicio.
type ClassAdvanceService struct {
	store    domain.ClassAdvanceStore
	sessions domain.ClassSessionStore
	// audit e history son opcionales: sin ellos el aula funciona igual, pero no
	// queda registro ni se puede leer el historial. Se cablean en cmd/api.
	audit     domain.ClassAuditRecorder
	history   domain.ClassHistoryStore
	retention domain.AdvanceSnapshotRetentionStore
	// Snapshots, environments y volumes se cablean con WithApplier: la emisión no
	// los necesita, la aplicación en el entorno del alumno sí.
	snapshots    domain.AdvanceSnapshotStore
	environments domain.PracticeEnvironmentRepository
	volumes      domain.PracticeEnvironmentVolumeStore
	// pause aplica la pausa en vivo al sistema de archivos de los alumnos. Sin
	// él la sesión se marca, pero nadie queda en solo lectura; en producción se
	// cablea con el servicio de congelamiento de examen.
	pause SessionPauseStore
	// applySlots limita cuántas aplicaciones leen y escriben volúmenes a la vez.
	// Las demás esperan su turno en cola; ninguna se rechaza por esto.
	applySlots chan struct{}
	now        func() time.Time
}

// SessionPauseStore aplica y revierte el solo lectura de los laboratorios de un
// examen en curso. Se declara en el consumidor para no atar el servicio de
// avances al mecanismo de Docker.
type SessionPauseStore interface {
	PauseSession(ctx context.Context, tenantID, teacherID, role, sessionID string) (FreezeReport, error)
	ResumeSession(ctx context.Context, tenantID, teacherID, role, sessionID string) (FreezeReport, error)
}

// defaultAdvanceApplyConcurrency es cuántas aplicaciones de avance corren a la
// vez por defecto (ADR-058 §A13). Se ajusta con ADVANCE_APPLY_CONCURRENCY.
const defaultAdvanceApplyConcurrency = 4

func NewClassAdvanceService(store domain.ClassAdvanceStore, sessions domain.ClassSessionStore) *ClassAdvanceService {
	return &ClassAdvanceService{
		store:      store,
		sessions:   sessions,
		applySlots: make(chan struct{}, defaultAdvanceApplyConcurrency),
		now:        time.Now,
	}
}

// SetApplyConcurrency fija cuántas aplicaciones de avance corren en paralelo.
// Valores menores o iguales a cero dejan el valor por defecto.
func (s *ClassAdvanceService) SetApplyConcurrency(n int) {
	if n <= 0 {
		n = defaultAdvanceApplyConcurrency
	}
	s.applySlots = make(chan struct{}, n)
}

// SetAuditRepository engancha el registro de auditoría. Sin él, las acciones
// siguen funcionando y el historial queda vacío; la falta se nota en el aula,
// no se disfraza con una acción fallida.
func (s *ClassAdvanceService) SetAuditRepository(audit domain.ClassAuditRecorder) {
	s.audit = audit
}

// SetHistoryStore engancha la lectura del historial de la clase.
func (s *ClassAdvanceService) SetHistoryStore(history domain.ClassHistoryStore) {
	s.history = history
}

// SetRetentionStore engancha el almacenamiento que usa el barrido de snapshots.
// Sin él, la purga responde que el almacenamiento no está configurado y el aula
// sigue funcionando igual.
func (s *ClassAdvanceService) SetRetentionStore(retention domain.AdvanceSnapshotRetentionStore) {
	s.retention = retention
}

// WithPauseStore engancha la pausa en vivo: el docente detiene la escritura de
// los alumnos sin cerrar el examen (ADR-005 §5.2).
func (s *ClassAdvanceService) WithPauseStore(pause SessionPauseStore) *ClassAdvanceService {
	s.pause = pause
	return s
}

// IssueAdvanceInput agrupa los datos de una emisión.
type IssueAdvanceInput struct {
	// ID permite fijar el identificador antes de subir el snapshot, para que la
	// clave del objeto y la fila usen el mismo. Vacío genera uno nuevo.
	ID         string
	TenantID   string
	TeacherID  string
	Role       string
	SessionID  string
	ExerciseID string
	Title      string
	Scope      string
	StudentIDs []string
	// SnapshotKey, SnapshotSHA256, SnapshotSize y FileCount los produce el
	// empaquetado del snapshot (A3) antes de llamar a Issue.
	SnapshotKey    string
	SnapshotSHA256 string
	SnapshotSize   int64
	FileCount      int
}

// Issue emite un avance: valida la sesión, el examen, la congelación y los
// destinatarios, y registra el avance con una fila por alumno.
func (s *ClassAdvanceService) Issue(ctx context.Context, in IssueAdvanceInput) (*domain.ClassAdvance, []string, error) {
	if strings.TrimSpace(in.TenantID) == "" {
		return nil, nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(in.SessionID)); err != nil {
		return nil, nil, domain.ErrSessionNotFound
	}
	if _, err := uuid.Parse(strings.TrimSpace(in.ExerciseID)); err != nil {
		return nil, nil, domain.ErrInvalidResourceID
	}

	session, err := s.sessions.GetSession(ctx, in.TenantID, teacherScope(in.Role, in.TeacherID), in.SessionID)
	if err != nil {
		return nil, nil, err
	}
	if !session.IsOpen() {
		return nil, nil, domain.ErrAdvanceSessionClosed
	}
	if session.FrozenAt != nil {
		return nil, nil, domain.ErrAdvanceFrozen
	}

	purpose, err := s.store.ExercisePurpose(ctx, in.TenantID, in.ExerciseID)
	if err != nil {
		return nil, nil, err
	}
	if purpose == "exam" {
		return nil, nil, domain.ErrAdvanceExamBlocked
	}

	participants, err := s.store.SessionParticipants(ctx, in.TenantID, in.SessionID)
	if err != nil {
		return nil, nil, err
	}
	recipients, err := resolveRecipients(in.Scope, in.StudentIDs, participants)
	if err != nil {
		return nil, nil, err
	}

	advanceID := strings.TrimSpace(in.ID)
	if advanceID == "" {
		advanceID = uuid.NewString()
	}
	title := strings.TrimSpace(in.Title)
	if len(title) > 255 {
		title = title[:255]
	}
	advance := domain.ClassAdvance{
		ID:             advanceID,
		TenantID:       in.TenantID,
		SessionID:      in.SessionID,
		ExerciseID:     in.ExerciseID,
		Title:          title,
		Scope:          in.Scope,
		SnapshotKey:    in.SnapshotKey,
		SnapshotSHA256: in.SnapshotSHA256,
		SnapshotSize:   in.SnapshotSize,
		FileCount:      in.FileCount,
		CreatedAt:      s.now(),
	}
	if id := strings.TrimSpace(in.TeacherID); id != "" {
		advance.CreatedBy = &id
	}
	if err := s.store.CreateAdvance(ctx, advance, recipients); err != nil {
		return nil, nil, err
	}
	s.recordAudit(ctx, in.TenantID, in.TeacherID, domain.AuditClassAdvanceEmitted, in.SessionID, map[string]any{
		"advance_id":  advance.ID,
		"exercise_id": in.ExerciseID,
		"scope":       in.Scope,
		"recipients":  len(recipients),
		"file_count":  in.FileCount,
	})
	return &advance, recipients, nil
}

// resolveRecipients fija los destinatarios: toda la clase o un subconjunto que
// debe estar entre los participantes de la sesión.
func resolveRecipients(scope string, selected, participants []string) ([]string, error) {
	switch scope {
	case domain.AdvanceScopeClass:
		if len(participants) == 0 {
			return nil, domain.ErrAdvanceNoRecipients
		}
		return participants, nil
	case domain.AdvanceScopeSelected:
		if len(selected) == 0 {
			return nil, domain.ErrAdvanceScopeInvalid
		}
		inSession := make(map[string]bool, len(participants))
		for _, id := range participants {
			inSession[id] = true
		}
		seen := make(map[string]bool, len(selected))
		out := make([]string, 0, len(selected))
		for _, raw := range selected {
			id := strings.TrimSpace(raw)
			if _, err := uuid.Parse(id); err != nil {
				return nil, domain.ErrAdvanceRecipientNotInSession
			}
			if !inSession[id] {
				return nil, domain.ErrAdvanceRecipientNotInSession
			}
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		return out, nil
	default:
		return nil, domain.ErrAdvanceScopeInvalid
	}
}

// Freeze pausa un examen en curso: pone en solo lectura los laboratorios de los
// alumnos y deja la sesión abierta. Una clase no se pausa.
func (s *ClassAdvanceService) Freeze(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSession, error) {
	return s.setFrozen(ctx, tenantID, teacherID, role, sessionID, true)
}

// Unfreeze reanuda el examen pausado: los laboratorios vuelven a escritura.
func (s *ClassAdvanceService) Unfreeze(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSession, error) {
	return s.setFrozen(ctx, tenantID, teacherID, role, sessionID, false)
}

// setFrozen pausa o reanuda un examen. Primero se toca el montaje y después la
// fila de la sesión: el alumno nunca ve "pausado" mientras todavía puede
// escribir. Reanudar hace lo inverso por la misma razón.
func (s *ClassAdvanceService) setFrozen(ctx context.Context, tenantID, teacherID, role, sessionID string, frozen bool) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, domain.ErrSessionNotFound
	}
	session, err := s.sessions.GetSession(ctx, tenantID, teacherScope(role, teacherID), sessionID)
	if err != nil {
		return nil, err
	}
	if !session.IsOpen() {
		return nil, domain.ErrSessionNotOpen
	}
	if session.Purpose != string(domain.ExercisePurposeExam) {
		return nil, domain.ErrExamPauseNotAllowed
	}

	if s.pause != nil {
		var report FreezeReport
		if frozen {
			report, err = s.pause.PauseSession(ctx, tenantID, teacherID, role, sessionID)
		} else {
			report, err = s.pause.ResumeSession(ctx, tenantID, teacherID, role, sessionID)
		}
		if err != nil {
			return nil, err
		}
		if frozen && report.Frozen == 0 && report.Failed > 0 {
			return nil, domain.ErrExamPauseFailed
		}
	}

	session, err = s.store.SetFrozen(ctx, tenantID, teacherScope(role, teacherID), sessionID, frozen)
	if err != nil {
		return nil, err
	}
	action := domain.AuditClassUnfrozen
	if frozen {
		action = domain.AuditClassFrozen
	}
	s.recordAudit(ctx, tenantID, teacherID, action, sessionID, map[string]any{
		"frozen": frozen,
	})
	return session, nil
}

// SessionAdvances devuelve las emisiones del aula con el estado de cada
// destinatario. Para un docente de otra materia el aula no existe.
func (s *ClassAdvanceService) SessionAdvances(ctx context.Context, tenantID, teacherID, role, sessionID string) ([]domain.AdvanceWithRecipients, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, domain.ErrSessionNotFound
	}
	scope := teacherScope(role, teacherID)
	if _, err := s.sessions.GetSession(ctx, tenantID, scope, sessionID); err != nil {
		return nil, err
	}
	return s.store.SessionAdvances(ctx, tenantID, scope, sessionID)
}

// SessionHistory devuelve los movimientos de la clase, del más nuevo al más
// viejo. Para un docente de otra materia el aula no existe, igual que en el
// resto de las lecturas del aula.
func (s *ClassAdvanceService) SessionHistory(ctx context.Context, tenantID, teacherID, role, sessionID string) ([]domain.ClassHistoryEvent, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, domain.ErrSessionNotFound
	}
	scope := teacherScope(role, teacherID)
	if _, err := s.sessions.GetSession(ctx, tenantID, scope, sessionID); err != nil {
		return nil, err
	}
	if s.history == nil {
		return []domain.ClassHistoryEvent{}, nil
	}
	events, err := s.history.SessionHistory(ctx, tenantID, sessionID, 200)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []domain.ClassHistoryEvent{}
	}
	return events, nil
}

// SetGraceUntil fija la hora en que termina la ventana de una sesión cerrada.
//
// Recibe la hora final, no una cantidad de minutos: un doble clic, un reintento
// o dos pestañas abiertas mandan la misma hora y la ventana no se corre dos
// veces. Si la hora pedida ya es la vigente no se escribe nada ni se agrega
// otra fila al historial. Debe ser futura y no superar el máximo desde el
// cierre de la clase.
func (s *ClassAdvanceService) SetGraceUntil(ctx context.Context, tenantID, teacherID, role, sessionID string, until time.Time) (*domain.ClassSession, error) {
	session, err := s.closedSession(ctx, tenantID, teacherID, role, sessionID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if !until.After(now) || !insideAdvanceWindow(session, until, now) {
		return nil, domain.ErrAdvanceGraceDeadline
	}
	if session.GraceUntil != nil && session.GraceUntil.Equal(until) {
		return session, nil
	}
	updated, err := s.store.SetGrace(ctx, tenantID, teacherScope(role, teacherID), sessionID, until)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, tenantID, teacherID, domain.AuditClassAdvanceGraceSet, sessionID, map[string]any{
		"previous_until": session.GraceUntil,
		"until":          graceUntilOf(updated),
	})
	return updated, nil
}

// CloseGrace cierra la ventana en el momento. Los pendientes pasan a vencidos
// y los vencidos pueden volver a pendientes si el docente suma minutos después.
func (s *ClassAdvanceService) CloseGrace(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSession, error) {
	session, err := s.closedSession(ctx, tenantID, teacherID, role, sessionID)
	if err != nil {
		return nil, err
	}
	updated, err := s.store.SetGrace(ctx, tenantID, teacherScope(role, teacherID), sessionID, s.now())
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, tenantID, teacherID, domain.AuditClassAdvanceGraceClosed, sessionID, map[string]any{
		"previous_until": session.GraceUntil,
		"until":          graceUntilOf(updated),
	})
	return updated, nil
}

// closedSession valida la identidad y devuelve una sesión cerrada del docente.
func (s *ClassAdvanceService) closedSession(ctx context.Context, tenantID, teacherID, role, sessionID string) (*domain.ClassSession, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, domain.ErrSessionNotFound
	}
	session, err := s.sessions.GetSession(ctx, tenantID, teacherScope(role, teacherID), sessionID)
	if err != nil {
		return nil, err
	}
	if session.IsOpen() {
		return nil, domain.ErrAdvanceNotCloseable
	}
	return session, nil
}

// graceUntilOf tolera un almacén que no devuelva la sesión: el registro no
// debe romper una acción que ya quedó guardada.
func graceUntilOf(session *domain.ClassSession) *time.Time {
	if session == nil {
		return nil
	}
	return session.GraceUntil
}

// recordAudit guarda un movimiento del aula. Es best-effort, como el resto del
// proyecto: un fallo de auditoría no revierte la acción que el docente ya hizo.
func (s *ClassAdvanceService) recordAudit(ctx context.Context, tenantID, actorID, action, sessionID string, metadata map[string]any) {
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

// insideAdvanceWindow acota la ventana al máximo desde el cierre. Si la sesión
// no tiene closes_at (dato viejo), se toma ahora como referencia.
func insideAdvanceWindow(session *domain.ClassSession, until, now time.Time) bool {
	base := now
	if session.ClosesAt != nil {
		base = *session.ClosesAt
	}
	return !until.After(base.Add(time.Duration(domain.MaxAdvanceGraceDays) * 24 * time.Hour))
}
