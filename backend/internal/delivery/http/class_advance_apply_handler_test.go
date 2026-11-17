package httpdelivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

const advanceHTTPAdvanceID = "99999999-9999-9999-9999-999999999999"

// snapshotsForHTTP entrega el snapshot preparado por la prueba.
type snapshotsForHTTP struct {
	files []domain.PracticeEnvironmentVolumeFile
}

func (s *snapshotsForHTTP) Put(context.Context, string, io.Reader) (string, int64, error) {
	return "", 0, nil
}

func (s *snapshotsForHTTP) PutFiles(context.Context, string, map[string]string) (string, int64, int, error) {
	return "", 0, 0, nil
}

func (s *snapshotsForHTTP) Files(context.Context, string, string, int) ([]domain.PracticeEnvironmentVolumeFile, error) {
	return s.files, nil
}

func (s *snapshotsForHTTP) Delete(context.Context, string) error {
	return nil
}

// volumesForHTTP registra las escrituras sobre el volumen del alumno.
type volumesForHTTP struct {
	existing []domain.PracticeEnvironmentVolumeFile
	writes   []map[string]string
}

func (v *volumesForHTTP) SnapshotPracticeEnvironmentVolume(context.Context, string, int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	return v.existing, nil
}

func (v *volumesForHTTP) WritePracticeEnvironmentVolumeFiles(_ context.Context, _ string, files map[string]string) error {
	v.writes = append(v.writes, files)
	return nil
}

// stubPracticeEnvironments resuelve solo el entorno del alumno; el resto del contrato del
// repositorio no lo usa este camino.
type stubPracticeEnvironments struct {
	domain.PracticeEnvironmentRepository
	environment *domain.PracticeEnvironment
}

func (s stubPracticeEnvironments) GetByStudentSubjectExercise(context.Context, string, string, string) (*domain.PracticeEnvironment, error) {
	return s.environment, nil
}

func newAdvanceApplierHandler(delivery *domain.AdvanceDelivery, snapshots []domain.PracticeEnvironmentVolumeFile, existing []domain.PracticeEnvironmentVolumeFile) (*ClassAdvanceHandler, *advanceStoreForHTTP, *volumesForHTTP) {
	session := openClassSessionForHTTP()
	store := &advanceStoreForHTTP{session: session, delivery: delivery}
	service := services.NewClassAdvanceService(store, &sessionStoreForHTTP{session: session})
	volumes := &volumesForHTTP{existing: existing}
	environment := &domain.PracticeEnvironment{
		ID:         "66666666-6666-6666-6666-666666666666",
		StudentID:  advanceHTTPTeacherID,
		SubjectID:  session.SubjectID,
		ExerciseID: session.ExerciseID,
		Status:     domain.PracticeEnvironmentStatusRunning,
	}
	service.WithApplier(&snapshotsForHTTP{files: snapshots}, stubPracticeEnvironments{environment: environment}, volumes)
	return NewClassAdvanceHandler(service), store, volumes
}

func advanceApplyRequest(body string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, advanceHTTPTenantID)
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/class-advances/"+advanceHTTPAdvanceID+"/apply", reader)
	request.SetPathValue("id", advanceHTTPAdvanceID)
	return withTestIdentity(request, advanceHTTPTeacherID, advanceHTTPTenantID, "student")
}

func pendingDeliveryForHTTP() *domain.AdvanceDelivery {
	return &domain.AdvanceDelivery{
		ClassAdvance: domain.ClassAdvance{
			ID:             advanceHTTPAdvanceID,
			TenantID:       advanceHTTPTenantID,
			SessionID:      advanceHTTPSessionID,
			SnapshotSHA256: "hash",
			ExerciseID:     advanceHTTPExerciseID,
		},
		Status:        domain.AdvanceRecipientPending,
		SubjectID:     "44444444-4444-4444-4444-444444444444",
		SessionStatus: domain.ClassSessionStatusOpen,
		Purpose:       "practice",
	}
}

func TestClassAdvanceHandler_ApplySelectionWritesChosenFiles(t *testing.T) {
	handler, store, volumes := newAdvanceApplierHandler(
		pendingDeliveryForHTTP(),
		[]domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
			{Path: "utils.py", Content: []byte("helper\n")},
		},
		[]domain.PracticeEnvironmentVolumeFile{{Path: "main.py", Content: []byte("viejo\n")}},
	)

	recorder := httptest.NewRecorder()
	handler.ApplyAdvance(recorder, advanceApplyRequest(`{"selected_files":["main.py"]}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data services.AdvanceApplyResult `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	if payload.Data.FilesWritten != 1 {
		t.Errorf("esperado 1 archivo escrito, obtenido %d", payload.Data.FilesWritten)
	}
	if len(volumes.writes) != 2 {
		t.Fatalf("esperado respaldo y copia, obtenidas %d escrituras", len(volumes.writes))
	}
	if _, ok := volumes.writes[1]["utils.py"]; ok {
		t.Errorf("un archivo fuera de la selección no se escribe: %v", volumes.writes[1])
	}
	if len(store.decisions) != 1 || store.decisions[0] != domain.AdvanceRecipientApplied {
		t.Errorf("el destinatario debía quedar aplicado: %v", store.decisions)
	}
}

func TestClassAdvanceHandler_ApplyWithoutBodyAppliesAll(t *testing.T) {
	handler, _, volumes := newAdvanceApplierHandler(
		pendingDeliveryForHTTP(),
		[]domain.PracticeEnvironmentVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ApplyAdvance(recorder, advanceApplyRequest(""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200 sin cuerpo, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(volumes.writes) != 1 || volumes.writes[0]["main.py"] != "nuevo\n" {
		t.Fatalf("sin cuerpo se aplica el avance completo, obtenido: %v", volumes.writes)
	}
}

func TestClassAdvanceHandler_ApplyRejectsUnknownFiles(t *testing.T) {
	handler, _, volumes := newAdvanceApplierHandler(
		pendingDeliveryForHTTP(),
		[]domain.PracticeEnvironmentVolumeFile{{Path: "main.py", Content: []byte("nuevo\n")}},
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ApplyAdvance(recorder, advanceApplyRequest(`{"selected_files":["otro.py"]}`))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(volumes.writes) != 0 {
		t.Error("una ruta ajena al avance no debe escribir el volumen")
	}
}

func TestClassAdvanceHandler_ApplyRejectsInvalidBody(t *testing.T) {
	handler, _, _ := newAdvanceApplierHandler(pendingDeliveryForHTTP(), nil, nil)

	recorder := httptest.NewRecorder()
	handler.ApplyAdvance(recorder, advanceApplyRequest(`{"selected_files":`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestClassAdvanceHandler_RollbackRestoresBackup(t *testing.T) {
	delivery := pendingDeliveryForHTTP()
	delivery.Status = domain.AdvanceRecipientApplied
	handler, store, volumes := newAdvanceApplierHandler(
		delivery,
		nil,
		[]domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("avance\n")},
			{Path: ".solv_backups/" + advanceHTTPAdvanceID + "/main.py", Content: []byte("version previa\n")},
		},
	)

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, advanceHTTPTenantID)
	request := httptest.NewRequestWithContext(ctx, http.MethodPost,
		"/api/v1/class-advances/"+advanceHTTPAdvanceID+"/rollback", nil)
	request.SetPathValue("id", advanceHTTPAdvanceID)
	request = withTestIdentity(request, advanceHTTPTeacherID, advanceHTTPTenantID, "student")

	recorder := httptest.NewRecorder()
	handler.RollbackAdvance(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data services.AdvanceRollbackResult `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	if payload.Data.FilesRestored != 1 || payload.Data.Restored[0] != "main.py" {
		t.Fatalf("resultado inesperado: %+v", payload.Data)
	}
	if len(volumes.writes) != 1 || volumes.writes[0]["main.py"] != "version previa\n" {
		t.Fatalf("el rollback debe restaurar el respaldo, obtenido: %v", volumes.writes)
	}
	if len(store.reopened) != 1 {
		t.Errorf("el destinatario debía volver a pendiente: %v", store.reopened)
	}
}

func TestClassAdvanceHandler_RollbackRequiresAppliedAdvance(t *testing.T) {
	handler, _, volumes := newAdvanceApplierHandler(pendingDeliveryForHTTP(), nil, nil)

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, advanceHTTPTenantID)
	request := httptest.NewRequestWithContext(ctx, http.MethodPost,
		"/api/v1/class-advances/"+advanceHTTPAdvanceID+"/rollback", nil)
	request.SetPathValue("id", advanceHTTPAdvanceID)
	request = withTestIdentity(request, advanceHTTPTeacherID, advanceHTTPTenantID, "student")

	recorder := httptest.NewRecorder()
	handler.RollbackAdvance(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("esperado 409, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(volumes.writes) != 0 {
		t.Error("un aviso pendiente no se deshace")
	}
}

func TestClassAdvanceHandler_RollbackRequiresIdentity(t *testing.T) {
	handler, _, _ := newAdvanceApplierHandler(pendingDeliveryForHTTP(), nil, nil)

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/api/v1/class-advances/"+advanceHTTPAdvanceID+"/rollback", nil)
	request.SetPathValue("id", advanceHTTPAdvanceID)

	recorder := httptest.NewRecorder()
	handler.RollbackAdvance(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
}

// rollbackRequest arma la petición de deshacer con la selección de archivos.
func rollbackRequest(method, body string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, advanceHTTPTenantID)
	request := httptest.NewRequestWithContext(ctx, method,
		"/api/v1/class-advances/"+advanceHTTPAdvanceID+"/rollback", reader)
	request.SetPathValue("id", advanceHTTPAdvanceID)
	return withTestIdentity(request, advanceHTTPTeacherID, advanceHTTPTenantID, "student")
}

func TestClassAdvanceHandler_RollbackSelectionKeepsRemainingBackup(t *testing.T) {
	delivery := pendingDeliveryForHTTP()
	delivery.Status = domain.AdvanceRecipientApplied
	handler, store, volumes := newAdvanceApplierHandler(
		delivery,
		nil,
		[]domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("avance\n")},
			{Path: "utils.py", Content: []byte("avance helper\n")},
			{Path: ".solv_backups/" + advanceHTTPAdvanceID + "/main.py", Content: []byte("version previa\n")},
			{Path: ".solv_backups/" + advanceHTTPAdvanceID + "/utils.py", Content: []byte("helper previo\n")},
		},
	)

	recorder := httptest.NewRecorder()
	handler.RollbackAdvance(recorder, rollbackRequest(http.MethodPost, `{"paths":["main.py"]}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data services.AdvanceRollbackResult `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	if payload.Data.FullyRestored {
		t.Error("un deshacer parcial no puede quedar completo")
	}
	if len(payload.Data.Remaining) != 1 || payload.Data.Remaining[0] != "utils.py" {
		t.Fatalf("el resto debe seguir respaldado, obtenido: %+v", payload.Data)
	}
	if len(volumes.writes) != 1 || volumes.writes[0]["main.py"] != "version previa\n" {
		t.Fatalf("solo se restaura el archivo elegido, obtenido: %v", volumes.writes)
	}
	if _, ok := volumes.writes[0]["utils.py"]; ok {
		t.Error("un archivo fuera de la selección no se restaura")
	}
	if len(store.reopened) != 0 {
		t.Error("el avance sigue aplicado mientras quede respaldo")
	}
}

func TestClassAdvanceHandler_RollbackRejectsFilesWithoutBackup(t *testing.T) {
	delivery := pendingDeliveryForHTTP()
	delivery.Status = domain.AdvanceRecipientApplied
	handler, _, volumes := newAdvanceApplierHandler(
		delivery,
		nil,
		[]domain.PracticeEnvironmentVolumeFile{
			{Path: ".solv_backups/" + advanceHTTPAdvanceID + "/main.py", Content: []byte("version previa\n")},
		},
	)

	recorder := httptest.NewRecorder()
	handler.RollbackAdvance(recorder, rollbackRequest(http.MethodPost, `{"paths":["otro.py"]}`))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, hubo %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(volumes.writes) != 0 {
		t.Error("una ruta sin respaldo no debe escribir el volumen")
	}
}

func TestClassAdvanceHandler_RollbackInventoryListsFiles(t *testing.T) {
	delivery := pendingDeliveryForHTTP()
	delivery.Status = domain.AdvanceRecipientApplied
	handler, _, _ := newAdvanceApplierHandler(
		delivery,
		nil,
		[]domain.PracticeEnvironmentVolumeFile{
			{Path: ".solv_backups/" + advanceHTTPAdvanceID + "/utils.py", Content: []byte("helper\n")},
			{Path: ".solv_backups/" + advanceHTTPAdvanceID + "/main.py", Content: []byte("main\n")},
		},
	)

	recorder := httptest.NewRecorder()
	handler.RollbackInventory(recorder, rollbackRequest(http.MethodGet, ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, hubo %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data services.AdvanceRollbackInventory `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	if len(payload.Data.Files) != 2 || payload.Data.Files[0] != "main.py" || payload.Data.Files[1] != "utils.py" {
		t.Fatalf("inventario inesperado: %v", payload.Data.Files)
	}
}
