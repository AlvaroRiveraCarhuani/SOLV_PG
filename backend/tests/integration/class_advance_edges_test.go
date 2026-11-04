package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// advanceEdgeSnapshots guarda los snapshots en memoria y no permite pisar una
// clave: la emisión crea un objeto nuevo por avance, nunca reutiliza el anterior.
type advanceEdgeSnapshots struct {
	mu      sync.Mutex
	objects map[string]map[string]string
	deleted []string
}

func newAdvanceEdgeSnapshots() *advanceEdgeSnapshots {
	return &advanceEdgeSnapshots{objects: make(map[string]map[string]string)}
}

func (s *advanceEdgeSnapshots) Put(ctx context.Context, key string, body io.Reader) (string, int64, error) {
	content, err := io.ReadAll(body)
	if err != nil {
		return "", 0, err
	}
	hash, size, _, err := s.PutFiles(ctx, key, map[string]string{"snapshot": string(content)})
	return hash, size, err
}

func (s *advanceEdgeSnapshots) PutFiles(ctx context.Context, key string, files map[string]string) (string, int64, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.objects[key]; exists {
		return "", 0, 0, fmt.Errorf("el snapshot %s ya existe", key)
	}
	copied := make(map[string]string, len(files))
	var size int64
	for path, content := range files {
		copied[path] = content
		size += int64(len(content))
	}
	s.objects[key] = copied
	hash := sha256.New()
	paths := make([]string, 0, len(copied))
	for path := range copied {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(copied[path]))
	}
	return hex.EncodeToString(hash.Sum(nil)), size, len(files), nil
}

func (s *advanceEdgeSnapshots) Files(ctx context.Context, key, expectedSHA256 string, maxFiles int) ([]domain.PracticeEnvironmentVolumeFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, ok := s.objects[key]
	if !ok {
		return nil, fmt.Errorf("snapshot %s no encontrado", key)
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]domain.PracticeEnvironmentVolumeFile, 0, len(paths))
	for _, path := range paths {
		out = append(out, domain.PracticeEnvironmentVolumeFile{Path: path, Content: []byte(files[path])})
	}
	return out, nil
}

func (s *advanceEdgeSnapshots) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *advanceEdgeSnapshots) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.objects)
}

func (s *advanceEdgeSnapshots) deletedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.deleted)
}

// advanceEdgeVolumes simula los volúmenes del docente y de los alumnos. Cada
// escritura se guarda para comprobar qué versión llegó al alumno y cuándo.
type advanceEdgeVolumes struct {
	mu     sync.Mutex
	files  map[string]map[string]string
	writes int
}

func newAdvanceEdgeVolumes() *advanceEdgeVolumes {
	return &advanceEdgeVolumes{files: make(map[string]map[string]string)}
}

func (v *advanceEdgeVolumes) seed(volume string, files map[string]string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	copied := make(map[string]string, len(files))
	for path, content := range files {
		copied[path] = content
	}
	v.files[volume] = copied
}

func (v *advanceEdgeVolumes) replace(volume string, files map[string]string) {
	v.seed(volume, files)
}

func (v *advanceEdgeVolumes) SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	files := v.files[volumeName]
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]domain.PracticeEnvironmentVolumeFile, 0, len(paths))
	for _, path := range paths {
		out = append(out, domain.PracticeEnvironmentVolumeFile{Path: path, Content: []byte(files[path])})
	}
	return out, nil
}

func (v *advanceEdgeVolumes) WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.files[volumeName] == nil {
		v.files[volumeName] = make(map[string]string)
	}
	copied := make(map[string]string, len(files))
	for path, content := range files {
		v.files[volumeName][path] = content
		copied[path] = content
	}
	v.writes++
	return nil
}

func (v *advanceEdgeVolumes) contents(volume string) map[string]string {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make(map[string]string, len(v.files[volume]))
	for path, content := range v.files[volume] {
		out[path] = content
	}
	return out
}

func (v *advanceEdgeVolumes) writeCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.writes
}

// advanceEdgeFixture arma un aula real sobre PostgreSQL, con el handler HTTP y
// el servicio completos. El único doble es el almacenamiento de snapshots y los
// volúmenes, que evita depender de Docker y SeaweedFS en una prueba de reglas.
type advanceEdgeFixture struct {
	db            *database.Database
	server        *httptest.Server
	volumes       *advanceEdgeVolumes
	snapshots     *advanceEdgeSnapshots
	sessions      *postgres.PostgresClassSessionRepository
	tenantID      string
	teacherID     string
	otherTeacher  string
	subjectID     string
	practiceID    string
	examID        string
	sessionID     string
	// noPracticeSessionID es un aula abierta sin práctica asociada: no puede
	// emitir ni recibir avances. Vive en su propia materia porque solo hay un
	// aula abierta por materia.
	noPracticeSessionID string
	noPracticeSubjectID string
	students            []string
	outsiderID    string
	teacherVolume string
}

type advanceEdgeEmitResponse struct {
	Data struct {
		Advance    domain.ClassAdvance `json:"advance"`
		Recipients []string            `json:"recipients"`
	} `json:"data"`
}

type advanceEdgeListResponse struct {
	Data []domain.AdvanceWithRecipients `json:"data"`
}

func setupAdvanceEdgeFixture(t *testing.T) *advanceEdgeFixture {
	t.Helper()

	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
		return nil
	}
	ctx := context.Background()
	conn := db.GetDB()

	var templateID string
	if err := conn.GetContext(ctx, &templateID,
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status='approved' LIMIT 1"); err != nil {
		t.Skipf("se omite la prueba: no hay plantilla de juez aprobada: %v", err)
		return nil
	}

	f := &advanceEdgeFixture{
		db:           db,
		volumes:      newAdvanceEdgeVolumes(),
		snapshots:    newAdvanceEdgeSnapshots(),
		tenantID:     "00000000-0000-0000-0000-000000000001",
		teacherID:    uuid.NewString(),
		otherTeacher: uuid.NewString(),
		subjectID:    uuid.NewString(),
		practiceID:   uuid.NewString(),
		examID:       uuid.NewString(),
		sessionID:           uuid.NewString(),
		noPracticeSessionID: uuid.NewString(),
		noPracticeSubjectID: uuid.NewString(),
		students:            []string{uuid.NewString(), uuid.NewString()},
		outsiderID:   uuid.NewString(),
	}

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("fallo al preparar datos: %v", err)
		}
	}

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Borde', $2, 'teacher', $3)`,
		f.teacherID, fmt.Sprintf("docente_borde_%s@uab.edu.bo", f.teacherID[:8]), f.tenantID)
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Otro', 'Docente', $2, 'teacher', $3)`,
		f.otherTeacher, fmt.Sprintf("otro_docente_borde_%s@uab.edu.bo", f.otherTeacher[:8]), f.tenantID)

	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Avances de Borde', $3, $4)`,
		f.subjectID, f.tenantID, "AVB-"+f.subjectID[:8], f.teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Práctica de borde', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		f.practiceID, f.tenantID, f.subjectID, templateID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Examen de borde', 'published', 'JUEZ_EFIMERO', 'exam', $4)`,
		f.examID, f.tenantID, f.subjectID, templateID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		f.sessionID, f.tenantID, f.subjectID, f.practiceID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Materia sin práctica', $3, $4)`,
		f.noPracticeSubjectID, f.tenantID, "AVS-"+f.noPracticeSubjectID[:8], f.teacherID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, NULL, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		f.noPracticeSessionID, f.tenantID, f.noPracticeSubjectID)

	for i, studentID := range f.students {
		mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Alumno', $3, 'student', $4)`,
			studentID, fmt.Sprintf("AlumnoBorde%d", i+1),
			fmt.Sprintf("alumno_borde_%d_%s@uab.edu.bo", i+1, studentID[:8]), f.tenantID)
		mustExec(`INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)`,
			f.tenantID, studentID, f.subjectID)
		mustExec(`INSERT INTO class_session_participants (session_id, student_id, tenant_id)
			VALUES ($1, $2, $3)`, f.sessionID, studentID, f.tenantID)
		mustExec(`INSERT INTO environments (id, student_id, subject_id, exercise_id, tenant_id, status, type, access_url)
			VALUES ($1, $2, $3, $4, $5, 'running', 'IDE_PERSISTENTE', 'http://127.0.0.1:3000')`,
			uuid.NewString(), studentID, f.subjectID, f.practiceID, f.tenantID)
	}
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alumno', 'Ajeno', $2, 'student', $3)`,
		f.outsiderID, fmt.Sprintf("alumno_ajeno_borde_%s@uab.edu.bo", f.outsiderID[:8]), f.tenantID)
	mustExec(`INSERT INTO environments (id, student_id, subject_id, exercise_id, tenant_id, status, type, access_url)
		VALUES ($1, $2, $3, $4, $5, 'running', 'IDE_PERSISTENTE', 'http://127.0.0.1:3000')`,
		uuid.NewString(), f.teacherID, f.subjectID, f.practiceID, f.tenantID)

	f.teacherVolume = fmt.Sprintf("solv_workspace_%s_%s_%s", f.teacherID, f.subjectID, f.practiceID)
	f.volumes.seed(f.teacherVolume, map[string]string{"main.py": "version-1\n"})

	sessions := postgres.NewPostgresClassSessionRepository(conn)
	advances := postgres.NewPostgresClassAdvanceRepository(conn, sessions)
	environments := postgres.NewPostgresPracticeEnvironmentRepository(conn)
	service := services.NewClassAdvanceService(advances, sessions).
		WithApplier(f.snapshots, environments, f.volumes)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = f.tenantID
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.TenantIDKey, tenantID)))
		})
	}
	handlers := httpdelivery.Handlers{
		ClassAdvanceHandler: httpdelivery.NewClassAdvanceHandler(service),
		TenantMiddleware:    tenantMiddleware,
	}
	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	f.sessions = sessions
	f.server = httptest.NewServer(mux)
	t.Cleanup(func() {
		f.server.Close()
		_ = conn.Close()
	})
	return f
}

func (f *advanceEdgeFixture) do(t *testing.T, method, path, userID, role string, payload any) *http.Response {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("no se pudo codificar el cuerpo: %v", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, f.server.URL+path, body)
	if err != nil {
		t.Fatalf("no se pudo armar la petición: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", f.tenantID)
	req.Header.Set("X-User-Id", userID)
	req.Header.Set("X-User-Role", role)
	res, err := f.server.Client().Do(req)
	if err != nil {
		t.Fatalf("la petición %s %s falló: %v", method, path, err)
	}
	return res
}

func advanceEdgeExpectStatus(t *testing.T, res *http.Response, want int, label string) {
	t.Helper()
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("%s: se esperaba %d, hubo %d: %s", label, want, res.StatusCode, string(body))
	}
}

func (f *advanceEdgeFixture) emit(t *testing.T, userID, role, scope string, studentIDs []string) advanceEdgeEmitResponse {
	t.Helper()
	payload := map[string]any{"scope": scope}
	if len(studentIDs) > 0 {
		payload["student_ids"] = studentIDs
	}
	res := f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.sessionID+"/advances", userID, role, payload)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("emitir: se esperaba 201, hubo %d: %s", res.StatusCode, string(body))
	}
	var out advanceEdgeEmitResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("no se pudo decodificar la emisión: %v", err)
	}
	return out
}

// TestClassAdvanceEdges_EmissionRules cubre los rechazos de la emisión: docente
// ajeno, examen, clase congelada, destinatario fuera del aula y aula cerrada.
func TestClassAdvanceEdges_EmissionRules(t *testing.T) {
	f := setupAdvanceEdgeFixture(t)
	if f == nil {
		return
	}

	t.Run("un docente ajeno no emite ni congela", func(t *testing.T) {
		res := f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.otherTeacher, "teacher", map[string]any{"scope": "class"})
		advanceEdgeExpectStatus(t, res, http.StatusNotFound, "emisión ajena")

		res = f.do(t, http.MethodPatch, "/api/v1/teacher/sessions/"+f.sessionID+"/freeze",
			f.otherTeacher, "teacher", nil)
		advanceEdgeExpectStatus(t, res, http.StatusNotFound, "congelación ajena")

		res = f.do(t, http.MethodGet, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.otherTeacher, "teacher", nil)
		advanceEdgeExpectStatus(t, res, http.StatusNotFound, "lectura de avances ajena")

		var count int
		if err := f.db.GetDB().GetContext(context.Background(), &count,
			`SELECT COUNT(*) FROM class_advances WHERE session_id = $1`, f.sessionID); err != nil {
			t.Fatalf("no se pudo contar las emisiones: %v", err)
		}
		if count != 0 {
			t.Fatalf("una emisión ajena no debía crear filas, hubo %d", count)
		}
	})

	t.Run("un alumno no emite avances", func(t *testing.T) {
		res := f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.students[0], "student", map[string]any{"scope": "class"})
		advanceEdgeExpectStatus(t, res, http.StatusForbidden, "emisión de un alumno")
	})

	t.Run("un aula sin práctica responde 409 y no deja snapshot", func(t *testing.T) {
		before := f.snapshots.count()
		res := f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.noPracticeSessionID+"/advances",
			f.teacherID, "teacher", map[string]any{"scope": "class"})
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusConflict {
			t.Fatalf("emisión sin práctica: se esperaba 409, hubo %d: %s", res.StatusCode, string(body))
		}
		if !strings.Contains(string(body), "práctica asociada") {
			t.Fatalf("la respuesta debía nombrar la práctica faltante: %s", string(body))
		}
		if got := f.snapshots.count(); got != before {
			t.Fatalf("el aula sin práctica no debía empaquetar: antes %d, después %d", before, got)
		}
	})

	t.Run("la clase congelada responde 409 y descongelar permite emitir", func(t *testing.T) {
		res := f.do(t, http.MethodPatch, "/api/v1/teacher/sessions/"+f.sessionID+"/freeze",
			f.teacherID, "teacher", nil)
		advanceEdgeExpectStatus(t, res, http.StatusOK, "congelar")

		res = f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.teacherID, "teacher", map[string]any{"scope": "class"})
		advanceEdgeExpectStatus(t, res, http.StatusConflict, "emisión congelada")
		if got := f.snapshots.count(); got != 0 {
			t.Fatalf("con la clase congelada no se empaqueta nada, quedaron %d snapshots", got)
		}

		res = f.do(t, http.MethodPatch, "/api/v1/teacher/sessions/"+f.sessionID+"/unfreeze",
			f.teacherID, "teacher", nil)
		advanceEdgeExpectStatus(t, res, http.StatusOK, "descongelar")

		emitted := f.emit(t, f.teacherID, "teacher", "class", nil)
		if len(emitted.Data.Recipients) != len(f.students) {
			t.Fatalf("la emisión a toda la clase debía llegar a %d alumnos, llegó a %d",
				len(f.students), len(emitted.Data.Recipients))
		}
		if got := f.snapshots.count(); got != 1 {
			t.Fatalf("la emisión correcta debía dejar un snapshot, quedaron %d", got)
		}

		var pending int
		if err := f.db.GetDB().GetContext(context.Background(), &pending,
			`SELECT COUNT(*) FROM class_advance_recipients WHERE advance_id = $1 AND status = 'pending'`,
			emitted.Data.Advance.ID); err != nil {
			t.Fatalf("no se pudo contar los destinatarios: %v", err)
		}
		if pending != len(f.students) {
			t.Fatalf("se esperaban %d pendientes, hubo %d", len(f.students), pending)
		}
	})

	t.Run("un alumno fuera del aula responde 422", func(t *testing.T) {
		before := f.snapshots.count()
		res := f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.teacherID, "teacher", map[string]any{
				"scope":       "selected",
				"student_ids": []string{f.outsiderID},
			})
		advanceEdgeExpectStatus(t, res, http.StatusUnprocessableEntity, "destinatario fuera del aula")
		if got := f.snapshots.count(); got != before {
			t.Fatalf("la emisión rechazada no debía dejar snapshot: antes %d, después %d", before, got)
		}
	})

	t.Run("el aula cerrada no emite", func(t *testing.T) {
		if _, err := f.sessions.CloseSession(context.Background(), f.tenantID, f.teacherID, f.sessionID); err != nil {
			t.Fatalf("no se pudo cerrar el aula: %v", err)
		}

		before := f.snapshots.count()
		res := f.do(t, http.MethodPost, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.teacherID, "teacher", map[string]any{"scope": "class"})
		advanceEdgeExpectStatus(t, res, http.StatusConflict, "emisión con el aula cerrada")
		if got := f.snapshots.count(); got != before {
			t.Fatalf("el aula cerrada no debía empaquetar: antes %d, después %d", before, got)
		}
	})
}

// TestClassAdvanceEdges_StudentIsolationWindowAndSnapshot comprueba que un
// alumno ajeno no ve el avance, que la ventana vencida lo deja en expired y que
// el snapshot emitido conserva la versión del docente aunque él siga editando.
func TestClassAdvanceEdges_StudentIsolationWindowAndSnapshot(t *testing.T) {
	f := setupAdvanceEdgeFixture(t)
	if f == nil {
		return
	}
	ctx := context.Background()
	studentVolume := fmt.Sprintf("solv_workspace_%s_%s_%s", f.students[0], f.subjectID, f.practiceID)

	first := f.emit(t, f.teacherID, "teacher", "selected", []string{f.students[0]})
	if len(first.Data.Recipients) != 1 || first.Data.Recipients[0] != f.students[0] {
		t.Fatalf("la emisión seleccionada debía quedar solo para el primer alumno: %v", first.Data.Recipients)
	}

	// El docente sigue editando después de emitir: el avance ya guardado no cambia.
	f.volumes.replace(f.teacherVolume, map[string]string{"main.py": "version-2\n"})

	t.Run("un alumno ajeno recibe 404 y no escribe nada", func(t *testing.T) {
		before := f.volumes.writeCount()
		for _, studentID := range []string{f.students[1], f.outsiderID} {
			res := f.do(t, http.MethodPost, "/api/v1/class-advances/"+first.Data.Advance.ID+"/apply",
				studentID, "student", nil)
			advanceEdgeExpectStatus(t, res, http.StatusNotFound, "aplicación de un avance ajeno")
		}
		if got := f.volumes.writeCount(); got != before {
			t.Fatalf("un avance ajeno no debía escribir en ningún volumen, hubo %d escrituras", got-before)
		}
	})

	t.Run("el snapshot conserva la versión emitida", func(t *testing.T) {
		res := f.do(t, http.MethodPost, "/api/v1/class-advances/"+first.Data.Advance.ID+"/apply",
			f.students[0], "student", nil)
		advanceEdgeExpectStatus(t, res, http.StatusOK, "aplicación del primer avance")

		if got := f.volumes.contents(studentVolume)["main.py"]; got != "version-1\n" {
			t.Fatalf("el alumno debía recibir la versión emitida, recibió %q", got)
		}
	})

	second := f.emit(t, f.teacherID, "teacher", "selected", []string{f.students[0]})
	res := f.do(t, http.MethodPost, "/api/v1/class-advances/"+second.Data.Advance.ID+"/apply",
		f.students[0], "student", nil)
	advanceEdgeExpectStatus(t, res, http.StatusOK, "aplicación del segundo avance")
	if got := f.volumes.contents(studentVolume)["main.py"]; got != "version-2\n" {
		t.Fatalf("el segundo avance debía llevar la versión nueva, llevó %q", got)
	}

	// Dos avances quedan pendientes cuando se cierra el aula: el docente los ve
	// vencidos sin que el alumno abra el laboratorio.
	third := f.emit(t, f.teacherID, "teacher", "selected", []string{f.students[0]})
	fourth := f.emit(t, f.teacherID, "teacher", "selected", []string{f.students[0]})

	if _, err := f.sessions.CloseSession(ctx, f.tenantID, f.teacherID, f.sessionID); err != nil {
		t.Fatalf("no se pudo cerrar el aula: %v", err)
	}
	if _, err := f.db.GetDB().ExecContext(ctx,
		`UPDATE class_sessions SET grace_until = NOW() - INTERVAL '1 minute' WHERE id = $1`,
		f.sessionID); err != nil {
		t.Fatalf("no se pudo vencer la ventana: %v", err)
	}

	t.Run("el docente ve vencidos sin que el alumno abra el laboratorio", func(t *testing.T) {
		res := f.do(t, http.MethodGet, "/api/v1/teacher/sessions/"+f.sessionID+"/advances",
			f.teacherID, "teacher", nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("lectura de avances: se esperaba 200, hubo %d: %s", res.StatusCode, string(body))
		}
		var list advanceEdgeListResponse
		if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
			t.Fatalf("no se pudo decodificar la lista: %v", err)
		}
		expired := map[string]int{}
		for _, advance := range list.Data {
			expired[advance.ID] = advance.Counts.Expired
		}
		for _, id := range []string{third.Data.Advance.ID, fourth.Data.Advance.ID} {
			if expired[id] != 1 {
				t.Fatalf("el avance %s debía quedar vencido sin acción del alumno: %+v", id, expired)
			}
		}
	})

	t.Run("un avance vencido no se aplica ni toca el volumen", func(t *testing.T) {
		before := f.volumes.writeCount()
		res := f.do(t, http.MethodPost, "/api/v1/class-advances/"+third.Data.Advance.ID+"/apply",
			f.students[0], "student", nil)
		advanceEdgeExpectStatus(t, res, http.StatusConflict, "aplicación de un avance vencido")
		if got := f.volumes.writeCount(); got != before {
			t.Fatalf("un avance vencido no debía escribir, hubo %d escrituras", got-before)
		}

		var status string
		if err := f.db.GetDB().GetContext(ctx, &status,
			`SELECT status FROM class_advance_recipients WHERE advance_id = $1 AND student_id = $2`,
			third.Data.Advance.ID, f.students[0]); err != nil {
			t.Fatalf("no se pudo leer el estado del destinatario: %v", err)
		}
		if status != domain.AdvanceRecipientExpired {
			t.Fatalf("el destinatario debía quedar expired, quedó %q", status)
		}
	})
}
