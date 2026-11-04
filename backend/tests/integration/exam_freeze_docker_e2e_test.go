package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/docker"
)

// TestExamFreezeDockerE2E recorre el cierre del examen sobre un contenedor y un
// volumen reales: el alumno escribe, entrega, y el sistema de archivos deja de
// aceptar escrituras tanto desde el editor como desde un contenedor efímero con
// el volumen montado en solo lectura.
//
// Está apagada por defecto porque necesita el daemon de Docker, la imagen
// gitpod/openvscode-server:1.96.0 y la red solv-traefik-net. Se activa con:
//
//	EXAM_FREEZE_E2E=1 go test ./tests/integration -run TestExamFreezeDockerE2E -v
func TestExamFreezeDockerE2E(t *testing.T) {
	if os.Getenv("EXAM_FREEZE_E2E") != "1" {
		t.Skip("prueba real de Docker: definir EXAM_FREEZE_E2E=1")
	}
	if testing.Short() {
		t.Skip("prueba de punta a punta omitida con -short")
	}

	ctx := context.WithValue(t.Context(), domain.TenantIDKey, domain.DefaultTenantID)
	tenantID := domain.DefaultTenantID

	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	raw := client.GetRawClient()
	t.Cleanup(func() { _ = raw.Close() })
	if _, err := client.Info(ctx); err != nil {
		t.Fatalf("el daemon de Docker no responde: %v", err)
	}

	studentID := uuid.NewString()
	subjectID := uuid.NewString()
	environmentID := uuid.NewString()
	exerciseID := uuid.NewString()
	volumeName := fmt.Sprintf("solv_workspace_%s_%s_%s", studentID, subjectID, exerciseID)
	containerName := fmt.Sprintf("solv-environment-%s", environmentID)

	instance := &domain.PracticeEnvironment{
		ID:            environmentID,
		TenantID:      tenantID,
		StudentID:     studentID,
		SubjectID:     subjectID,
		ExerciseID:    &exerciseID,
		Type:          domain.PracticeEnvironmentTypeIDEPersistente,
		Status:        domain.PracticeEnvironmentStatusRunning,
		MemoryLimitMB: 512,
	}
	repo := &e2ePracticeEnvironmentRepo{instance: instance}
	submissions := &e2eSubmissionRepo{}
	environments := services.NewPracticeEnvironmentService(repo, client, nil).
		WithExerciseSubmission(&e2eExamExerciseRepo{subjectID: subjectID}, submissions)

	store := &e2eFreezeStore{instance: instance}
	freezer := services.NewExamFreezeService(store, &e2eFreezeSessions{}, repo, environments)
	environments.SetExamSubmissionFreezer(freezer)

	if err := client.EnsureVolumeExists(ctx, volumeName); err != nil {
		t.Fatalf("no se pudo crear el volumen %s: %v", volumeName, err)
	}
	t.Cleanup(func() {
		_ = raw.ContainerRemove(context.WithoutCancel(ctx), containerName, container.RemoveOptions{Force: true})
		_ = raw.VolumeRemove(context.WithoutCancel(ctx), volumeName, false)
	})

	// 1. El alumno arranca con el volumen en escritura y guarda su trabajo.
	if err := environments.SetPracticeEnvironmentMount(ctx, instance, false); err != nil {
		t.Fatalf("no se pudo montar el volumen en escritura: %v", err)
	}
	editableContainer := *instance.ContainerID
	waitContainerRunning(t, raw, editableContainer)

	execInPracticeEnvironment(t, raw, editableContainer, "printf 'package main\\n' > /home/workspace/main.go")
	if got := execInPracticeEnvironment(t, raw, editableContainer, "cat /home/workspace/main.go"); !strings.Contains(got, "package main") {
		t.Fatalf("el alumno debía poder escribir antes de entregar, se leyó %q", got)
	}

	// 2. La entrega es la que dispara el congelamiento: solo ese volumen.
	submission, err := environments.SubmitPracticeEnvironmentProject(ctx, environmentID, studentID, exerciseID, map[string]string{
		"notas.txt": "entrega del examen",
	})
	if err != nil {
		t.Fatalf("la entrega del examen falló: %v", err)
	}
	if submissions.created == nil || submissions.created.ID != submission.ID {
		t.Fatal("la entrega debía quedar registrada")
	}
	if !strings.Contains(submission.Code, "# === FILE: main.go ===") {
		t.Fatalf("el snapshot debía incluir el archivo del alumno: %s", submission.Code)
	}
	if instance.FrozenAt == nil || instance.Status != domain.PracticeEnvironmentStatusFrozen {
		t.Fatalf("la entrega debía marcar el volumen como congelado: %+v", instance)
	}
	if instance.ContainerID == nil || *instance.ContainerID == editableContainer {
		t.Fatal("el contenedor debía reconstruirse en solo lectura")
	}
	frozenContainer := *instance.ContainerID
	waitContainerRunning(t, raw, frozenContainer)

	// 3. El editor ya no escribe: el montaje en :ro lo impide aunque el proceso
	// corra como root. La lectura sigue disponible para consultar lo entregado.
	_, stderr, code, err := execResult(ctx, raw, frozenContainer, "echo intruso > /home/workspace/main.go")
	if err != nil {
		t.Fatalf("no se pudo ejecutar la escritura de prueba: %v", err)
	}
	if code == 0 {
		t.Fatalf("el editor no debía poder escribir con el examen entregado (stderr=%q)", stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "read-only file system") {
		t.Fatalf("la escritura debía fallar por el sistema de archivos, no por otra causa: %q", stderr)
	}
	if got := execInPracticeEnvironment(t, raw, frozenContainer, "cat /home/workspace/main.go"); !strings.Contains(got, "package main") {
		t.Fatalf("el contenido entregado debía seguir legible: %q", got)
	}

	// 4. Un contenedor efímero con el volumen en solo lectura, que es como lo lee
	// el juez, tampoco puede escribir.
	if got := runEphemeralVolumeCommand(t, ctx, raw, volumeName, true, "cat /environment/main.go"); !strings.Contains(got, "package main") {
		t.Fatalf("el contenedor efímero debía leer lo entregado: %q", got)
	}
	_, stderr, code, err = runEphemeralVolumeCommandResult(ctx, raw, volumeName, true, "echo intruso > /environment/efimero.txt")
	if err != nil {
		t.Fatalf("no se pudo ejecutar la escritura efímera: %v", err)
	}
	if code == 0 || !strings.Contains(strings.ToLower(stderr), "read-only file system") {
		t.Fatalf("el contenedor efímero no debía escribir en un volumen congelado (code=%d, stderr=%q)", code, stderr)
	}

	// 5. Volver a entregar tampoco es una vía para escribir: el servidor rechaza
	// la entrega antes de tocar el volumen.
	_, err = environments.SubmitPracticeEnvironmentProject(ctx, environmentID, studentID, exerciseID, map[string]string{
		"tarde.txt": "fuera de plazo",
	})
	if !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("una entrega sobre un volumen congelado debía rechazarse, se obtuvo %v", err)
	}
	_, _, code, err = runEphemeralVolumeCommandResult(ctx, raw, volumeName, true, "test ! -e /environment/tarde.txt")
	if err != nil {
		t.Fatalf("no se pudo comprobar el volumen: %v", err)
	}
	if code != 0 {
		t.Fatal("la entrega rechazada no debía escribir archivos en el volumen")
	}

	// 6. Reabrir vuelve a habilitar la escritura en el mismo volumen.
	if err := environments.SetPracticeEnvironmentMount(ctx, instance, false); err != nil {
		t.Fatalf("no se pudo reabrir el volumen: %v", err)
	}
	reopened := *instance.ContainerID
	waitContainerRunning(t, raw, reopened)
	execInPracticeEnvironment(t, raw, reopened, "printf 'reenviado\\n' > /home/workspace/tarde.txt")
	if got := execInPracticeEnvironment(t, raw, reopened, "cat /home/workspace/tarde.txt"); !strings.Contains(got, "reenviado") {
		t.Fatalf("tras reabrir el alumno debía poder escribir otra vez: %q", got)
	}
}

// e2ePracticeEnvironmentRepo es el repositorio mínimo que el servicio de entornos necesita:
// la fila viva del laboratorio y la actualización del contenedor reconstruido.
type e2ePracticeEnvironmentRepo struct {
	domain.PracticeEnvironmentRepository
	instance *domain.PracticeEnvironment
}

func (r *e2ePracticeEnvironmentRepo) GetByID(context.Context, string) (*domain.PracticeEnvironment, error) {
	return r.instance, nil
}

func (r *e2ePracticeEnvironmentRepo) UpdateContainerID(_ context.Context, _, containerID string) error {
	r.instance.ContainerID = &containerID
	return nil
}

func (r *e2ePracticeEnvironmentRepo) UpdateAccessURL(_ context.Context, _, accessURL string) error {
	r.instance.AccessURL = accessURL
	return nil
}

// e2eSubmissionRepo guarda la entrega en memoria: esta prueba mira el volumen y
// el montaje, no la persistencia.
type e2eSubmissionRepo struct {
	domain.SubmissionRepository
	created *domain.Submission
}

func (r *e2eSubmissionRepo) Create(_ context.Context, submission *domain.Submission) error {
	r.created = submission
	return nil
}

// e2eExamExerciseRepo devuelve siempre un examen de la materia del laboratorio.
type e2eExamExerciseRepo struct {
	domain.ExerciseRepository
	subjectID string
}

func (r *e2eExamExerciseRepo) GetByIDAndTenant(_ context.Context, exerciseID, _ string) (*domain.Exercise, error) {
	subjectID := r.subjectID
	return &domain.Exercise{
		ID:        exerciseID,
		Purpose:   string(domain.ExercisePurposeExam),
		SubjectID: &subjectID,
	}, nil
}

// e2eFreezeStore marca la fila viva para que la misma instancia refleje el
// congelamiento, igual que haría PostgreSQL.
type e2eFreezeStore struct {
	instance *domain.PracticeEnvironment
}

func (s *e2eFreezeStore) SessionIsExam(context.Context, string, string) (bool, error) {
	return true, nil
}

func (s *e2eFreezeStore) ParticipantPracticeEnvironments(context.Context, string, string) ([]domain.ExamFreezePracticeEnvironment, error) {
	return nil, nil
}

func (s *e2eFreezeStore) ClosedExamSessions(context.Context, int) ([]domain.ExamSessionRef, error) {
	return nil, nil
}

func (s *e2eFreezeStore) SetPracticeEnvironmentFrozen(_ context.Context, _, _ string, frozen bool, at time.Time) error {
	if !frozen {
		s.instance.FrozenAt = nil
		s.instance.Status = domain.PracticeEnvironmentStatusRunning
		return nil
	}
	s.instance.FrozenAt = &at
	s.instance.Status = domain.PracticeEnvironmentStatusFrozen
	return nil
}

// e2eFreezeSessions no se usa: esta prueba congela un volumen suelto.
type e2eFreezeSessions struct{}

func (s *e2eFreezeSessions) GetSession(context.Context, string, string, string) (*domain.ClassSession, error) {
	return nil, errors.New("la prueba congela un volumen suelto, no un aula")
}

// waitContainerRunning espera a que el contenedor esté en ejecución: el
// arranque no bloquea y la primera escritura puede llegar antes.
func waitContainerRunning(t *testing.T, raw *dockerclient.Client, containerID string) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for {
		inspect, err := raw.ContainerInspect(context.Background(), containerID)
		if err == nil && inspect.State.Running {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("el contenedor %s no llegó a ejecutarse: %v", containerID, err)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// execInPracticeEnvironment ejecuta un comando como root y devuelve la salida. Root evita
// que un permiso de directorio se confunda con el bloqueo del montaje.
func execInPracticeEnvironment(t *testing.T, cli *dockerclient.Client, containerID, script string) string {
	t.Helper()
	stdout, _, code, err := execResult(context.Background(), cli, containerID, script)
	if err != nil {
		t.Fatalf("exec en %s falló: %v", containerID, err)
	}
	if code != 0 {
		t.Fatalf("el comando %q salió con código %d", script, code)
	}
	return stdout
}

// execResult corre el comando sin exigir éxito: devuelve salida, error y código.
func execResult(ctx context.Context, cli *dockerclient.Client, containerID, script string) (string, string, int, error) {
	create, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		User:         "0",
		Cmd:          []string{"sh", "-c", script},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", "", 0, err
	}
	attached, err := cli.ContainerExecAttach(ctx, create.ID, container.ExecStartOptions{})
	if err != nil {
		return "", "", 0, err
	}
	defer attached.Close()

	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := stdcopy.StdCopy(&stdout, &stderr, attached.Reader)
		done <- copyErr
	}()

	select {
	case copyErr := <-done:
		if copyErr != nil && !errors.Is(copyErr, io.EOF) {
			return stdout.String(), stderr.String(), 0, copyErr
		}
	case <-time.After(60 * time.Second):
		return "", "", 0, errors.New("el comando dentro del contenedor no terminó a tiempo")
	case <-ctx.Done():
		return "", "", 0, ctx.Err()
	}

	inspect, err := cli.ContainerExecInspect(ctx, create.ID)
	if err != nil {
		return stdout.String(), stderr.String(), 0, err
	}
	return stdout.String(), stderr.String(), inspect.ExitCode, nil
}

// runEphemeralVolumeCommand ejecuta un comando en un contenedor alpine que monta
// el volumen del alumno en solo lectura, como el lector del juez.
func runEphemeralVolumeCommand(t *testing.T, ctx context.Context, cli *dockerclient.Client, volumeName string, readOnly bool, script string) string {
	t.Helper()
	stdout, stderr, code, err := runEphemeralVolumeCommandResult(ctx, cli, volumeName, readOnly, script)
	if err != nil {
		t.Fatalf("el contenedor efímero falló: %v", err)
	}
	if code != 0 {
		t.Fatalf("el comando %q salió con código %d: %s", script, code, strings.TrimSpace(stderr))
	}
	return stdout
}

// runEphemeralVolumeCommandResult corre el comando y devuelve también el código y
// el stderr para distinguir un fallo del bloqueo de escritura.
func runEphemeralVolumeCommandResult(ctx context.Context, cli *dockerclient.Client, volumeName string, readOnly bool, script string) (string, string, int, error) {
	mode := "rw"
	if readOnly {
		mode = "ro"
	}
	created, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine:3.20",
		User:  "0",
		Cmd:   []string{"sh", "-c", script},
	}, &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:/environment:%s", volumeName, mode)},
		NetworkMode: "none",
	}, nil, nil, "")
	if err != nil {
		return "", "", 0, err
	}
	defer func() {
		_ = cli.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
	}()

	if err := cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return "", "", 0, err
	}
	statusCh, errCh := cli.ContainerWait(ctx, created.ID, container.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return "", "", 0, err
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}

	logStream, err := cli.ContainerLogs(ctx, created.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", "", 0, err
	}
	defer func() { _ = logStream.Close() }()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, logStream); err != nil {
		return "", "", 0, err
	}
	return stdout.String(), stderr.String(), int(statusCode), nil
}
