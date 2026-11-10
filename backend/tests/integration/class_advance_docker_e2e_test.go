package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/docker"
	"solv-backend/internal/infrastructure/storage/postgres"
	"solv-backend/internal/infrastructure/storage/seaweedfs"
)

// TestClassAdvanceDockerE2E recorre el avance contra Docker y SeaweedFS reales:
// el docente emite desde su volumen, el alumno ve el aviso, previsualiza los
// archivos, aplica con respaldo y la ventana vence. No usa dobles de volumen ni
// de almacenamiento.
//
// Está apagada por defecto porque necesita el daemon de Docker, el servicio
// `seaweedfs` de Compose y las credenciales del .env. Se activa con:
//
//	ADVANCE_E2E=1 SEAWEEDFS_ENDPOINT=http://127.0.0.1:8333 go test ./tests/integration -run TestClassAdvanceDockerE2E -v
//
// El endpoint del .env apunta a `seaweedfs:8333`, que es el nombre dentro de la
// red de Compose; desde el host hay que usar 127.0.0.1. Usar un bucket de
// prueba, no el de producción: la prueba crea y borra objetos en él.
func TestClassAdvanceDockerE2E(t *testing.T) {
	if os.Getenv("ADVANCE_E2E") != "1" {
		t.Skip("prueba real de Docker y SeaweedFS: definir ADVANCE_E2E=1 y las variables SEAWEEDFS_*")
	}
	if testing.Short() {
		t.Skip("prueba de punta a punta omitida con -short")
	}

	ctx := context.WithValue(t.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	tenantID := domain.DefaultTenantID

	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}
	conn := db.GetDB()
	t.Cleanup(func() { _ = conn.Close() })

	var templateID string
	if err := conn.GetContext(ctx, &templateID,
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status IN ('approved','APROBADA') LIMIT 1"); err != nil {
		t.Fatalf("no hay plantilla de juez aprobada para la práctica: %v", err)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	if _, err := dockerClient.Info(ctx); err != nil {
		t.Fatalf("el daemon de Docker no responde: %v", err)
	}
	t.Cleanup(func() { _ = dockerClient.GetRawClient().Close() })

	endpoint := strings.TrimSpace(os.Getenv("SEAWEEDFS_ENDPOINT"))
	bucket := strings.TrimSpace(os.Getenv("SEAWEEDFS_BUCKET"))
	accessKey := strings.TrimSpace(os.Getenv("SEAWEEDFS_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("SEAWEEDFS_SECRET_KEY"))
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		t.Fatalf("faltan SEAWEEDFS_ENDPOINT, SEAWEEDFS_BUCKET, SEAWEEDFS_ACCESS_KEY o SEAWEEDFS_SECRET_KEY; cargar el .env y apuntar el endpoint a http://127.0.0.1:8333")
	}

	maxBytes := int64(4 << 20)
	if raw := strings.TrimSpace(os.Getenv("ADVANCE_SNAPSHOT_MAX_BYTES")); raw != "" {
		if parsed, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && parsed > 0 {
			maxBytes = parsed
		}
	}
	maxFiles := 512
	if raw := strings.TrimSpace(os.Getenv("ADVANCE_SNAPSHOT_MAX_FILES")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			maxFiles = parsed
		}
	}
	excludes := []string{".vscode/", ".solv_*", ".git/", "node_modules/"}
	if raw := strings.TrimSpace(os.Getenv("ADVANCE_SNAPSHOT_EXCLUDES")); raw != "" {
		excludes = strings.Split(raw, ",")
	}

	store, err := seaweedfs.NewAdvanceSnapshotStore(ctx, seaweedfs.Config{
		Endpoint:  endpoint,
		Region:    strings.TrimSpace(os.Getenv("SEAWEEDFS_REGION")),
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		MaxBytes:  maxBytes,
		Archive:   seaweedfs.ArchiveOptions{ExcludePrefixes: excludes, MaxFiles: maxFiles},
	})
	if err != nil {
		t.Fatalf("no se pudo abrir el almacén de SeaweedFS en %s: %v", endpoint, err)
	}

	teacherID := uuid.NewString()
	studentID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	sessionID := uuid.NewString()

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := conn.ExecContext(ctx, query, args...); execErr != nil {
			t.Fatalf("fallo al preparar datos: %v", execErr)
		}
	}

	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'E2E', $2, 'teacher', $3)`,
		teacherID, fmt.Sprintf("docente_e2e_%s@uab.edu.bo", teacherID[:8]), tenantID)
	mustExec(`INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alumno', 'E2E', $2, 'student', $3)`,
		studentID, fmt.Sprintf("alumno_e2e_%s@uab.edu.bo", studentID[:8]), tenantID)
	mustExec(`INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Avance E2E', $3, $4)`,
		subjectID, tenantID, "E2E-"+subjectID[:8], teacherID)
	mustExec(`INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, purpose, template_id)
		VALUES ($1, $2, $3, 'Práctica E2E', 'published', 'JUEZ_EFIMERO', 'class', $4)`,
		exerciseID, tenantID, subjectID, templateID)
	mustExec(`INSERT INTO class_sessions (id, tenant_id, subject_id, exercise_id, status, opens_at, ends_at)
		VALUES ($1, $2, $3, $4, 'open', NOW(), NOW() + INTERVAL '90 minutes')`,
		sessionID, tenantID, subjectID, exerciseID)
	mustExec(`INSERT INTO enrollments (tenant_id, student_id, subject_id) VALUES ($1, $2, $3)`,
		tenantID, studentID, subjectID)
	mustExec(`INSERT INTO class_session_participants (session_id, student_id, tenant_id)
		VALUES ($1, $2, $3)`, sessionID, studentID, tenantID)
	mustExec(`INSERT INTO environments (id, student_id, subject_id, exercise_id, tenant_id, status, type, access_url)
		VALUES ($1, $2, $3, $4, $5, 'running', 'IDE_PERSISTENTE', 'http://127.0.0.1:3000')`,
		uuid.NewString(), teacherID, subjectID, exerciseID, tenantID)
	mustExec(`INSERT INTO environments (id, student_id, subject_id, exercise_id, tenant_id, status, type, access_url)
		VALUES ($1, $2, $3, $4, $5, 'running', 'IDE_PERSISTENTE', 'http://127.0.0.1:3000')`,
		uuid.NewString(), studentID, subjectID, exerciseID, tenantID)

	teacherVolume := fmt.Sprintf("solv_workspace_%s_%s_%s", teacherID, subjectID, exerciseID)
	studentVolume := fmt.Sprintf("solv_workspace_%s_%s_%s", studentID, subjectID, exerciseID)
	for _, volumeName := range []string{teacherVolume, studentVolume} {
		if err := dockerClient.EnsureVolumeExists(ctx, volumeName); err != nil {
			t.Fatalf("no se pudo crear el volumen %s: %v", volumeName, err)
		}
	}

	teacherCode := "def solve():\n    return 42\n"
	studentCode := "def solve():\n    return 0  # intento del alumno\n"
	if err := dockerClient.WritePracticeEnvironmentVolumeFiles(ctx, teacherVolume, map[string]string{
		"main.py":                    teacherCode,
		"README.md":                  "Solución del docente\n",
		".vscode/settings.json":      `{"editor.tabSize": 2}`,
		"node_modules/demo/index.js": "module.exports = 1\n",
	}); err != nil {
		t.Fatalf("no se pudo sembrar el volumen del docente: %v", err)
	}
	if err := dockerClient.WritePracticeEnvironmentVolumeFiles(ctx, studentVolume, map[string]string{
		"main.py":    studentCode,
		"alumno.txt": "trabajo previo\n",
	}); err != nil {
		t.Fatalf("no se pudo sembrar el volumen del alumno: %v", err)
	}

	var emittedKeys []string
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for _, key := range emittedKeys {
			_ = store.Delete(cleanupCtx, key)
		}
		raw := dockerClient.GetRawClient()
		_ = raw.VolumeRemove(cleanupCtx, teacherVolume, true)
		_ = raw.VolumeRemove(cleanupCtx, studentVolume, true)
		_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM class_sessions WHERE id = $1`, sessionID)
		_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM environments WHERE subject_id = $1`, subjectID)
		_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM enrollments WHERE subject_id = $1`, subjectID)
		_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM exercises WHERE id = $1`, exerciseID)
		_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM subjects WHERE id = $1`, subjectID)
		_, _ = conn.ExecContext(cleanupCtx, `DELETE FROM users WHERE id IN ($1, $2)`, teacherID, studentID)
	})

	sessions := postgres.NewPostgresClassSessionRepository(conn)
	advances := postgres.NewPostgresClassAdvanceRepository(conn, sessions)
	environments := postgres.NewPostgresPracticeEnvironmentRepository(conn)
	svc := services.NewClassAdvanceService(advances, sessions).
		WithApplier(store, environments, dockerClient)

	advance, recipients, err := svc.Emit(ctx, services.EmitAdvanceInput{
		TenantID:  tenantID,
		TeacherID: teacherID,
		Role:      "teacher",
		SessionID: sessionID,
		Scope:     domain.AdvanceScopeClass,
	})
	if err != nil {
		t.Fatalf("Emit falló contra Docker y SeaweedFS: %v", err)
	}
	if advance == nil || advance.SnapshotKey == "" {
		t.Fatal("la emisión no devolvió el snapshot")
	}
	emittedKeys = append(emittedKeys, advance.SnapshotKey)
	if len(recipients) != 1 || recipients[0] != studentID {
		t.Fatalf("la emisión debía quedar para el alumno del aula, quedó para %v", recipients)
	}
	if !strings.HasPrefix(advance.SnapshotKey, "class-advances/"+tenantID+"/") {
		t.Fatalf("la clave del snapshot no lleva el tenant: %s", advance.SnapshotKey)
	}
	if advance.FileCount != 2 {
		t.Fatalf("el snapshot debía llevar main.py y README.md, llevó %d archivos", advance.FileCount)
	}
	if len(advance.SnapshotSHA256) != 64 {
		t.Fatalf("hash de snapshot inesperado: %q", advance.SnapshotSHA256)
	}
	t.Logf("emisión real: avance=%s snapshot=%s archivos=%d", advance.ID, advance.SnapshotKey, advance.FileCount)

	// El objeto está en SeaweedFS y el hash guardado se verifica al leerlo.
	snapshotFiles, err := store.Files(ctx, advance.SnapshotKey, advance.SnapshotSHA256, 10)
	if err != nil {
		t.Fatalf("el snapshot real no se pudo leer de SeaweedFS: %v", err)
	}
	snapshotContent := make(map[string]string, len(snapshotFiles))
	for _, file := range snapshotFiles {
		snapshotContent[file.Path] = string(file.Content)
	}
	if snapshotContent["main.py"] != teacherCode || snapshotContent["README.md"] != "Solución del docente\n" {
		t.Fatalf("contenido del snapshot inesperado: %v", snapshotContent)
	}
	if _, ok := snapshotContent[".vscode/settings.json"]; ok {
		t.Fatal(".vscode/ no debía viajar en el snapshot")
	}
	if _, ok := snapshotContent["node_modules/demo/index.js"]; ok {
		t.Fatal("node_modules/ no debía viajar en el snapshot")
	}

	// Aviso: el alumno ve el avance pendiente y la confirmación separa lo que
	// reemplaza de lo que agrega.
	pending, err := svc.PendingAdvances(ctx, tenantID, studentID)
	if err != nil {
		t.Fatalf("PendingAdvances falló: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != advance.ID || pending[0].FileCount != 2 {
		t.Fatalf("el aviso pendiente no es el esperado: %+v", pending)
	}
	preview, err := svc.Preview(ctx, tenantID, studentID, advance.ID)
	if err != nil {
		t.Fatalf("Preview falló contra el volumen real: %v", err)
	}
	if len(preview.Overwritten) != 1 || preview.Overwritten[0] != "main.py" {
		t.Fatalf("el aviso debía reemplazar main.py: %+v", preview.Overwritten)
	}
	if len(preview.Added) != 1 || preview.Added[0] != "README.md" {
		t.Fatalf("el aviso debía agregar README.md: %+v", preview.Added)
	}
	if dirs, listErr := dockerClient.ListPracticeEnvironmentBackupDirs(ctx, studentVolume, ".solv_backups"); listErr != nil {
		t.Fatalf("no se pudieron listar los respaldos antes de aplicar: %v", listErr)
	} else if len(dirs) != 0 {
		t.Fatalf("la previsualización no debía crear respaldos: %v", dirs)
	}

	// Aplicación: el volumen del alumno recibe la versión emitida y guarda el
	// estado previo de main.py en .solv_backups/<avance>/.
	result, err := svc.Apply(ctx, tenantID, studentID, advance.ID)
	if err != nil {
		t.Fatalf("Apply falló contra el volumen real: %v", err)
	}
	if result.FilesWritten != 2 {
		t.Fatalf("se esperaban 2 archivos escritos, hubo %d", result.FilesWritten)
	}
	if len(result.Overwritten) != 1 || result.Overwritten[0] != "main.py" {
		t.Fatalf("archivos reemplazados inesperados: %v", result.Overwritten)
	}
	if len(result.Added) != 1 || result.Added[0] != "README.md" {
		t.Fatalf("archivos agregados inesperados: %v", result.Added)
	}
	wantBackup := ".solv_backups/" + advance.ID
	if result.BackupPath == nil || *result.BackupPath != wantBackup {
		t.Fatalf("ruta de respaldo inesperada: %v", result.BackupPath)
	}

	studentFiles, err := dockerClient.SnapshotPracticeEnvironmentVolume(ctx, studentVolume, maxBytes)
	if err != nil {
		t.Fatalf("no se pudo leer el volumen del alumno después de aplicar: %v", err)
	}
	studentContent := make(map[string]string, len(studentFiles))
	for _, file := range studentFiles {
		studentContent[file.Path] = string(file.Content)
	}
	if studentContent["main.py"] != teacherCode {
		t.Fatalf("main.py del alumno no quedó con la versión emitida: %q", studentContent["main.py"])
	}
	if studentContent["README.md"] != "Solución del docente\n" {
		t.Fatalf("README.md no se agregó al volumen: %q", studentContent["README.md"])
	}
	if studentContent["alumno.txt"] != "trabajo previo\n" {
		t.Fatalf("el archivo previo del alumno se perdió: %q", studentContent["alumno.txt"])
	}
	if got := studentContent[wantBackup+"/main.py"]; got != studentCode {
		t.Fatalf("el respaldo debía guardar el main.py previo, guardó %q", got)
	}
	if _, ok := studentContent[".vscode/settings.json"]; ok {
		t.Fatal(".vscode/ del docente no debía llegar al volumen del alumno")
	}

	var first struct {
		Status     string  `db:"status"`
		BackupPath *string `db:"backup_path"`
	}
	if err := conn.GetContext(ctx, &first,
		`SELECT status, backup_path FROM class_advance_recipients WHERE advance_id = $1 AND student_id = $2`,
		advance.ID, studentID); err != nil {
		t.Fatalf("no se pudo leer el destinatario después de aplicar: %v", err)
	}
	if first.Status != domain.AdvanceRecipientApplied {
		t.Fatalf("el destinatario debía quedar applied, quedó %q", first.Status)
	}
	if first.BackupPath == nil || *first.BackupPath != wantBackup {
		t.Fatalf("backup_path en la base inesperado: %v", first.BackupPath)
	}

	// Segundo aviso, esta vez sin aplicar: al cerrar y vencer la ventana el
	// sistema lo marca vencido y no vuelve a tocar el volumen.
	second, secondRecipients, err := svc.Emit(ctx, services.EmitAdvanceInput{
		TenantID:  tenantID,
		TeacherID: teacherID,
		Role:      "teacher",
		SessionID: sessionID,
		Scope:     domain.AdvanceScopeClass,
	})
	if err != nil {
		t.Fatalf("la segunda emisión falló: %v", err)
	}
	if second == nil || len(secondRecipients) != 1 || secondRecipients[0] != studentID {
		t.Fatalf("la segunda emisión no quedó para el alumno: %v", secondRecipients)
	}
	emittedKeys = append(emittedKeys, second.SnapshotKey)

	if _, err := sessions.CloseSession(ctx, tenantID, teacherID, sessionID); err != nil {
		t.Fatalf("no se pudo cerrar el aula: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		`UPDATE class_sessions SET grace_until = NOW() - INTERVAL '1 minute' WHERE id = $1`,
		sessionID); err != nil {
		t.Fatalf("no se pudo vencer la ventana: %v", err)
	}
	if _, err := svc.Apply(ctx, tenantID, studentID, second.ID); !errors.Is(err, domain.ErrAdvanceGraceExpired) {
		t.Fatalf("un avance pendiente fuera de la ventana debía devolver ErrAdvanceGraceExpired, fue: %v", err)
	}

	var secondStatus string
	if err := conn.GetContext(ctx, &secondStatus,
		`SELECT status FROM class_advance_recipients WHERE advance_id = $1 AND student_id = $2`,
		second.ID, studentID); err != nil {
		t.Fatalf("no se pudo leer el destinatario después del vencimiento: %v", err)
	}
	if secondStatus != domain.AdvanceRecipientExpired {
		t.Fatalf("el destinatario pendiente debía quedar expired, quedó %q", secondStatus)
	}
	if pending, pendingErr := svc.PendingAdvances(ctx, tenantID, studentID); pendingErr != nil {
		t.Fatalf("PendingAdvances falló después del vencimiento: %v", pendingErr)
	} else if len(pending) != 0 {
		t.Fatalf("un avance vencido no debía seguir como pendiente: %+v", pending)
	}

	afterExpiry, err := dockerClient.SnapshotPracticeEnvironmentVolume(ctx, studentVolume, maxBytes)
	if err != nil {
		t.Fatalf("no se pudo releer el volumen después del vencimiento: %v", err)
	}
	afterContent := make(map[string]string, len(afterExpiry))
	for _, file := range afterExpiry {
		afterContent[file.Path] = string(file.Content)
	}
	if afterContent["main.py"] != teacherCode || afterContent["README.md"] != "Solución del docente\n" {
		t.Fatal("el vencimiento no debía cambiar el volumen del alumno")
	}

	// Los objetos siguen en SeaweedFS: el vencimiento de la ventana no borra el
	// código. La retención lo hace recién a los 8 días del cierre.
	for _, current := range []*domain.ClassAdvance{advance, second} {
		if _, err := store.Files(ctx, current.SnapshotKey, current.SnapshotSHA256, 10); err != nil {
			t.Fatalf("el snapshot %s debía seguir disponible después del vencimiento: %v", current.ID, err)
		}
	}

	// Integridad del adaptador real: un objeto distinto del emitido no se
	// entrega. Se usa una clave aparte para no tocar los avances de la prueba.
	corruptKey := "class-advances/" + tenantID + "/e2e-" + uuid.NewString() + ".tar.gz"
	if _, _, err := store.Put(ctx, corruptKey, strings.NewReader("contenido alterado")); err != nil {
		t.Fatalf("no se pudo subir el objeto de integridad: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), corruptKey) })
	if _, err := store.Files(ctx, corruptKey, strings.Repeat("0", 64), 10); !errors.Is(err, seaweedfs.ErrChecksumMismatch) {
		t.Fatalf("un objeto alterado debía fallar por hash, fue: %v", err)
	}

	t.Logf("aplicación real: %d archivos, respaldo=%s, primer estado=%s, segundo estado=%s",
		result.FilesWritten, wantBackup, first.Status, secondStatus)
}
