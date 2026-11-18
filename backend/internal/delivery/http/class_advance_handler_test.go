package httpdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

const (
	advanceHTTPTenantID  = "11111111-1111-1111-1111-111111111111"
	advanceHTTPTeacherID = "22222222-2222-2222-2222-222222222222"
	advanceHTTPSessionID = "33333333-3333-3333-3333-333333333333"
	// advanceHTTPExerciseID es la práctica del aula: el avance pertenece siempre
	// a la práctica que la sesión tiene asociada.
	advanceHTTPExerciseID = "77777777-7777-4777-8777-777777777777"
)

// advanceStoreForHTTP solo guarda lo que miran las rutas de sesión: la
// congelación y la ventana de rezago. El resto del contrato queda vacío.
type advanceStoreForHTTP struct {
	session     *domain.ClassSession
	frozenCalls []bool
	graceCalled []time.Time
	advances    []domain.AdvanceWithRecipients
	delivery    *domain.AdvanceDelivery
	reopened    []string
	decisions   []string
}

func (s *advanceStoreForHTTP) CreateAdvance(context.Context, domain.ClassAdvance, []string) error {
	return nil
}

func (s *advanceStoreForHTTP) SessionParticipants(context.Context, string, string) ([]string, error) {
	return nil, nil
}

func (s *advanceStoreForHTTP) ExercisePurpose(context.Context, string, string) (string, error) {
	return "practice", nil
}

func (s *advanceStoreForHTTP) SetFrozen(_ context.Context, _, _, sessionID string, frozen bool) (*domain.ClassSession, error) {
	if s.session == nil || sessionID != s.session.ID {
		return nil, domain.ErrSessionNotFound
	}
	s.frozenCalls = append(s.frozenCalls, frozen)
	if frozen {
		if s.session.FrozenAt == nil {
			now := time.Now()
			s.session.FrozenAt = &now
		}
		return s.session, nil
	}
	s.session.FrozenAt = nil
	return s.session, nil
}

func (s *advanceStoreForHTTP) SetGrace(_ context.Context, _, _, sessionID string, graceUntil time.Time) (*domain.ClassSession, error) {
	if s.session == nil || sessionID != s.session.ID {
		return nil, domain.ErrSessionNotFound
	}
	s.graceCalled = append(s.graceCalled, graceUntil)
	s.session.GraceUntil = &graceUntil
	return s.session, nil
}

func (s *advanceStoreForHTTP) PendingAdvances(context.Context, string, string) ([]domain.AdvanceDelivery, error) {
	return nil, nil
}

func (s *advanceStoreForHTTP) AdvanceForStudent(context.Context, string, string, string) (*domain.AdvanceDelivery, error) {
	if s.delivery == nil {
		return nil, domain.ErrAdvanceNotFound
	}
	return s.delivery, nil
}

func (s *advanceStoreForHTTP) DecideRecipient(_ context.Context, _, _, _ string, status string, _ *string) error {
	s.decisions = append(s.decisions, status)
	return nil
}

func (s *advanceStoreForHTTP) ReopenRecipient(_ context.Context, _, _, advanceID string) error {
	s.reopened = append(s.reopened, advanceID)
	s.delivery.Status = domain.AdvanceRecipientPending
	return nil
}

func (s *advanceStoreForHTTP) SessionAdvances(context.Context, string, string, string) ([]domain.AdvanceWithRecipients, error) {
	return s.advances, nil
}

// sessionStoreForHTTP devuelve la sesión preparada por la prueba.
type sessionStoreForHTTP struct {
	session *domain.ClassSession
}

func (s *sessionStoreForHTTP) OpenSession(context.Context, string, string, string, *string, int) (*domain.ClassSession, error) {
	return s.session, nil
}

func (s *sessionStoreForHTTP) CloseSession(context.Context, string, string, string) (*domain.ClassSession, error) {
	return s.session, nil
}

func (s *sessionStoreForHTTP) GetOpenSessionForTeacher(context.Context, string, string) (*domain.ClassSession, error) {
	return s.session, nil
}

func (s *sessionStoreForHTTP) GetLastClosedSessionForTeacher(context.Context, string, string) (*domain.ClassSession, error) {
	return s.session, nil
}

func (s *sessionStoreForHTTP) GetSession(_ context.Context, _, _, sessionID string) (*domain.ClassSession, error) {
	if s.session == nil || sessionID != s.session.ID {
		return nil, domain.ErrSessionNotFound
	}
	return s.session, nil
}

func (s *sessionStoreForHTTP) Board(context.Context, string, string, string) (*domain.ClassSessionBoard, error) {
	return nil, nil
}

func (s *sessionStoreForHTTP) OpenSessionsForStudent(context.Context, string, string) ([]domain.ClassSession, error) {
	return nil, nil
}

func (s *sessionStoreForHTTP) TouchParticipant(context.Context, string, string, string) error {
	return nil
}

func (s *sessionStoreForHTTP) AttachPracticeEnvironmentToOpenSession(context.Context, string, string, string, string) error {
	return nil
}

func newAdvanceHandlerForHTTP(session *domain.ClassSession) (*ClassAdvanceHandler, *advanceStoreForHTTP) {
	advances := &advanceStoreForHTTP{session: session}
	sessions := &sessionStoreForHTTP{session: session}
	service := services.NewClassAdvanceService(advances, sessions)
	return NewClassAdvanceHandler(service), advances
}

// historyStoreForHTTP entrega el historial preparado y cuenta las lecturas.
type historyStoreForHTTP struct {
	events []domain.ClassHistoryEvent
	calls  int
}

func (h *historyStoreForHTTP) SessionHistory(context.Context, string, string, int) ([]domain.ClassHistoryEvent, error) {
	h.calls++
	return h.events, nil
}

func newAdvanceHandlerWithHistory(session *domain.ClassSession, events []domain.ClassHistoryEvent) (*ClassAdvanceHandler, *historyStoreForHTTP) {
	advances := &advanceStoreForHTTP{session: session}
	sessions := &sessionStoreForHTTP{session: session}
	service := services.NewClassAdvanceService(advances, sessions)
	history := &historyStoreForHTTP{events: events}
	service.SetHistoryStore(history)
	return NewClassAdvanceHandler(service), history
}

// Sin práctica asociada el aula responde un conflicto con el motivo de dominio
// y no un error genérico de almacenamiento.
func TestWriteAdvanceError_MapsMissingPracticeAsConflict(t *testing.T) {
	rec := httptest.NewRecorder()
	(&ClassAdvanceHandler{}).writeAdvanceError(rec, domain.ErrAdvancePracticeRequired, "No se pudo emitir el avance")

	if rec.Code != http.StatusConflict {
		t.Fatalf("esperado 409, hubo %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), domain.ErrAdvancePracticeRequired.Error()) {
		t.Fatalf("la respuesta debe llevar el motivo de dominio, fue: %s", rec.Body.String())
	}
}

func openClassSessionForHTTP() *domain.ClassSession {
	now := time.Now()
	exerciseID := advanceHTTPExerciseID
	return &domain.ClassSession{
		ID:         advanceHTTPSessionID,
		TenantID:   advanceHTTPTenantID,
		SubjectID:  "44444444-4444-4444-4444-444444444444",
		ExerciseID: &exerciseID,
		Status:     domain.ClassSessionStatusOpen,
		OpensAt:    now,
		EndsAt:     now.Add(90 * time.Minute),
	}
}

func openExamSessionForHTTP() *domain.ClassSession {
	session := openClassSessionForHTTP()
	session.Purpose = string(domain.ExercisePurposeExam)
	return session
}

func closedClassSessionForHTTP() *domain.ClassSession {
	session := openClassSessionForHTTP()
	session.Status = domain.ClassSessionStatusClosed
	closedAt := time.Now()
	session.ClosesAt = &closedAt
	return session
}

func advanceSessionRequest(method, role, sessionID, body string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, advanceHTTPTenantID)
	request := httptest.NewRequestWithContext(ctx, method, "/api/v1/teacher/sessions/"+sessionID, reader)
	request.SetPathValue("id", sessionID)
	return withTestIdentity(request, advanceHTTPTeacherID, advanceHTTPTenantID, role)
}

func decodeSessionResponse(t *testing.T, recorder *httptest.ResponseRecorder) domain.ClassSession {
	t.Helper()
	var payload struct {
		Data domain.ClassSession `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar la sesión: %v", err)
	}
	return payload.Data
}

func TestClassAdvanceHandler_FreezeAndUnfreezeSession(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(openExamSessionForHTTP())

	freeze := httptest.NewRecorder()
	handler.FreezeSession(freeze, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, ""))
	if freeze.Code != http.StatusOK {
		t.Fatalf("congelar: esperado 200, hubo %d: %s", freeze.Code, freeze.Body.String())
	}
	frozen := decodeSessionResponse(t, freeze)
	if frozen.FrozenAt == nil {
		t.Fatal("la sesión congelada debía traer frozen_at")
	}

	unfreeze := httptest.NewRecorder()
	handler.UnfreezeSession(unfreeze, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, ""))
	if unfreeze.Code != http.StatusOK {
		t.Fatalf("descongelar: esperado 200, hubo %d: %s", unfreeze.Code, unfreeze.Body.String())
	}
	thawed := decodeSessionResponse(t, unfreeze)
	if thawed.FrozenAt != nil {
		t.Fatalf("la sesión descongelada no debía traer frozen_at: %v", thawed.FrozenAt)
	}

	if len(store.frozenCalls) != 2 || !store.frozenCalls[0] || store.frozenCalls[1] {
		t.Fatalf("el almacén debía recibir congelar y descongelar en orden: %v", store.frozenCalls)
	}
}

// Pausar una clase es una acción válida: la sesión sigue abierta y queda
// marcada para que los alumnos vean el aviso de solo lectura (ADR-005 §5.2).
func TestClassAdvanceHandler_FreezeClassSession(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.FreezeSession(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("pausar una clase: esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	frozen := decodeSessionResponse(t, recorder)
	if frozen.FrozenAt == nil {
		t.Fatal("la clase pausada debía traer frozen_at")
	}
	if len(store.frozenCalls) != 1 || !store.frozenCalls[0] {
		t.Fatalf("la clase debía marcarse como pausada: %v", store.frozenCalls)
	}
}

func TestClassAdvanceHandler_FreezeRejectsStudent(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.FreezeSession(recorder, advanceSessionRequest(http.MethodPatch, "student", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.frozenCalls) != 0 {
		t.Fatalf("un alumno no debía llegar al almacén: %v", store.frozenCalls)
	}
}

func TestClassAdvanceHandler_FreezeUnknownSession(t *testing.T) {
	handler, _ := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.FreezeSession(recorder, advanceSessionRequest(http.MethodPatch, "teacher", "no-es-un-uuid", ""))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestClassAdvanceHandler_SetGraceUntilWritesTheExactHour(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())
	until := time.Now().Add(45 * time.Minute).UTC().Truncate(time.Second)

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID,
		fmt.Sprintf(`{"until":%q}`, until.Format(time.RFC3339))))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.graceCalled) != 1 || !store.graceCalled[0].Equal(until) {
		t.Fatalf("el almacén debía recibir la hora final %v: %v", until, store.graceCalled)
	}
}

func TestClassAdvanceHandler_ExtendGraceRequiresClosedSession(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	until := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID,
		fmt.Sprintf(`{"until":%q}`, until.Format(time.RFC3339))))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("esperado 409, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.graceCalled) != 0 {
		t.Fatalf("con el aula abierta no se fija ventana: %v", store.graceCalled)
	}
}

func TestClassAdvanceHandler_GraceRejectsABodyWithoutAction(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, `{}`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.graceCalled) != 0 {
		t.Fatalf("un cuerpo sin acción no debía escribir la ventana: %v", store.graceCalled)
	}
}

func TestClassAdvanceHandler_ExtendGraceRejectsStudent(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	until := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "student", advanceHTTPSessionID,
		fmt.Sprintf(`{"until":%q}`, until.Format(time.RFC3339))))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.graceCalled) != 0 {
		t.Fatalf("un alumno no debía fijar la ventana: %v", store.graceCalled)
	}
}

func TestClassAdvanceHandler_SessionAdvancesRejectsStudent(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.GetSessionAdvances(recorder, advanceSessionRequest(http.MethodGet, "student", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if store.advances != nil {
		t.Fatal("un alumno no debía llegar al almacén")
	}
}

func TestClassAdvanceHandler_SessionAdvancesUnknownAula(t *testing.T) {
	handler, _ := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.GetSessionAdvances(recorder, advanceSessionRequest(http.MethodGet, "teacher", "otra-sesion", ""))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestClassAdvanceHandler_SessionAdvancesReturnsRecipientState(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(openClassSessionForHTTP())
	store.advances = []domain.AdvanceWithRecipients{{
		ID:            "99999999-9999-9999-9999-999999999999",
		ExerciseTitle: "Recorridos",
		Counts: domain.AdvanceRecipientCounts{
			Total: 2, Applied: 1, Pending: 1,
		},
		Recipients: []domain.AdvanceRecipientSummary{
			{StudentID: "11111111-1111-1111-1111-111111111111", Name: "Ana", Status: domain.AdvanceRecipientApplied},
			{StudentID: "22222222-2222-2222-2222-222222222222", Name: "Beto", Status: domain.AdvanceRecipientPending},
		},
	}}

	recorder := httptest.NewRecorder()
	handler.GetSessionAdvances(recorder, advanceSessionRequest(http.MethodGet, "teacher", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data []domain.AdvanceWithRecipients `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].Counts.Applied != 1 || payload.Data[0].Recipients[1].Name != "Beto" {
		t.Fatalf("la respuesta debía traer el estado de cada destinatario: %+v", payload.Data)
	}
}

func TestClassAdvanceHandler_SessionHistoryReturnsEvents(t *testing.T) {
	handler, store := newAdvanceHandlerWithHistory(openClassSessionForHTTP(), []domain.ClassHistoryEvent{{
		ID:        "evento-1",
		Action:    domain.AuditClassFrozen,
		ActorName: "Docente",
		ActorRole: "teacher",
	}})

	recorder := httptest.NewRecorder()
	handler.GetSessionHistory(recorder, advanceSessionRequest(http.MethodGet, "teacher", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data []domain.ClassHistoryEvent `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar el historial: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].Action != domain.AuditClassFrozen {
		t.Fatalf("historial inesperado: %+v", payload.Data)
	}
	if store.calls != 1 {
		t.Fatalf("se esperaba una lectura del historial, hubo %d", store.calls)
	}
}

func TestClassAdvanceHandler_SessionHistoryRejectsStudent(t *testing.T) {
	handler, store := newAdvanceHandlerWithHistory(openClassSessionForHTTP(), nil)

	recorder := httptest.NewRecorder()
	handler.GetSessionHistory(recorder, advanceSessionRequest(http.MethodGet, "student", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if store.calls != 0 {
		t.Fatalf("un alumno no debía llegar al historial: %d llamadas", store.calls)
	}
}

func TestClassAdvanceHandler_SessionHistoryUnknownAula(t *testing.T) {
	handler, store := newAdvanceHandlerWithHistory(openClassSessionForHTTP(), nil)

	recorder := httptest.NewRecorder()
	handler.GetSessionHistory(recorder, advanceSessionRequest(http.MethodGet, "teacher", "otra-sesion", ""))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if store.calls != 0 {
		t.Fatalf("un aula ajena no debía leer el historial: %d llamadas", store.calls)
	}
}

func TestClassAdvanceHandler_PreviewWithoutSnapshotStore(t *testing.T) {
	// Sin SeaweedFS configurado el aviso sigue visible, pero no se puede leer
	// qué archivos trae: el alumno recibe un 503 explicable, no un 500.
	handler, _ := newAdvanceHandlerForHTTP(openClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.PreviewAdvance(recorder, advanceSessionRequest(http.MethodGet, "student", advanceHTTPSessionID, ""))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperado 503, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestClassAdvanceHandler_ExtendGraceInvalidBody(t *testing.T) {
	handler, _ := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, `{"until":`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestClassAdvanceHandler_SetGraceDeadline(t *testing.T) {
	handler, _ := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())
	deadline := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID,
		fmt.Sprintf(`{"until":%q}`, deadline.Format(time.RFC3339))))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	session := decodeSessionResponse(t, recorder)
	if session.GraceUntil == nil || !session.GraceUntil.Equal(deadline) {
		t.Fatalf("la fecha límite debía quedar en %v, quedó en %v", deadline, session.GraceUntil)
	}
}

func TestClassAdvanceHandler_SetGraceDeadlineBeyondMax(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())
	deadline := time.Now().Add(8 * 24 * time.Hour).UTC().Truncate(time.Second)

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID,
		fmt.Sprintf(`{"until":%q}`, deadline.Format(time.RFC3339))))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.graceCalled) != 0 {
		t.Fatalf("una fecha fuera del máximo no debía escribir la ventana: %v", store.graceCalled)
	}
}

func TestClassAdvanceHandler_CloseGrace(t *testing.T) {
	handler, _ := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())
	before := time.Now()

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, `{"close":true}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	session := decodeSessionResponse(t, recorder)
	if session.GraceUntil == nil {
		t.Fatal("cerrar la ventana debía devolver grace_until")
	}
	if session.GraceUntil.Before(before.Add(-time.Second)) || session.GraceUntil.After(time.Now().Add(time.Second)) {
		t.Fatalf("la ventana debía cerrarse ahora, quedó en %v", session.GraceUntil)
	}
}

func TestClassAdvanceHandler_GraceRejectsAmbiguousAction(t *testing.T) {
	handler, store := newAdvanceHandlerForHTTP(closedClassSessionForHTTP())

	recorder := httptest.NewRecorder()
	handler.ExtendGrace(recorder, advanceSessionRequest(http.MethodPatch, "teacher", advanceHTTPSessionID, `{"until":"2026-11-18T23:00:00Z","close":true}`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.graceCalled) != 0 {
		t.Fatalf("un cuerpo ambiguo no debía escribir la ventana: %v", store.graceCalled)
	}
}
