package integration

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/docker"
)

// TestExamRetakeDockerE2E comprueba sobre volúmenes reales lo que sostiene el
// recuperatorio: el intento nuevo estrena volumen, la siembra escribe sobre ese
// volumen y el intento anterior queda intacto. También confirma que el espacio
// de práctica de la misma práctica conserva su contenido.
//
// Está apagada por defecto porque necesita el daemon de Docker y la imagen del
// contenedor auxiliar. Se activa con:
//
//	EXAM_RETAKE_E2E=1 go test ./tests/integration -run TestExamRetakeDockerE2E -v
func TestExamRetakeDockerE2E(t *testing.T) {
	if os.Getenv("EXAM_RETAKE_E2E") != "1" {
		t.Skip("prueba real de Docker: definir EXAM_RETAKE_E2E=1")
	}
	if testing.Short() {
		t.Skip("prueba de punta a punta omitida con -short")
	}

	ctx := context.Background()
	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	if _, err := client.Info(ctx); err != nil {
		t.Fatalf("el daemon de Docker no responde: %v", err)
	}

	studentID := "e2e" + uuid.NewString()[:8]
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()
	practice := domain.PracticeEnvironmentAttemptVolumeName(studentID, subjectID, &exerciseID, 0)
	firstAttempt := domain.PracticeEnvironmentAttemptVolumeName(studentID, subjectID, &exerciseID, 1)
	secondAttempt := domain.PracticeEnvironmentAttemptVolumeName(studentID, subjectID, &exerciseID, 2)

	for _, volume := range []string{practice, firstAttempt, secondAttempt} {
		volumeName := volume
		if err := client.EnsureVolumeExists(ctx, volumeName); err != nil {
			t.Fatalf("no se pudo crear el volumen %s: %v", volumeName, err)
		}
		t.Cleanup(func() {
			if remover, ok := any(client).(interface {
				RemoveVolume(context.Context, string) error
			}); ok {
				_ = remover.RemoveVolume(context.Background(), volumeName)
			}
		})
	}

	if practice == firstAttempt || firstAttempt == secondAttempt || practice == secondAttempt {
		t.Fatalf("los volúmenes de práctica y de cada intento debían ser distintos: %q %q %q", practice, firstAttempt, secondAttempt)
	}

	// El espacio de práctica ya tiene trabajo del alumno.
	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, practice, map[string]string{
		"main.py": "print('practica')\n",
	}); err != nil {
		t.Fatalf("no se pudo escribir en el volumen de práctica: %v", err)
	}

	// El examen del primer intento nace con el proyecto base.
	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, firstAttempt, map[string]string{
		"main.py": "print('base del examen')\n",
	}); err != nil {
		t.Fatalf("no se pudo sembrar el primer intento: %v", err)
	}

	// El recuperatorio estrena volumen y se siembra otra vez: no ve nada del
	// intento anterior.
	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, secondAttempt, map[string]string{
		"main.py": "print('base del examen')\n",
	}); err != nil {
		t.Fatalf("no se pudo sembrar el recuperatorio: %v", err)
	}
	if err := client.WritePracticeEnvironmentVolumeFiles(ctx, firstAttempt, map[string]string{
		"notas.txt": "intento uno\n",
	}); err != nil {
		t.Fatalf("no se pudo escribir en el primer intento: %v", err)
	}

	second, err := client.SnapshotPracticeEnvironmentVolumeStrict(ctx, secondAttempt, 4*1024*1024)
	if err != nil {
		t.Fatalf("no se pudo leer el recuperatorio: %v", err)
	}
	if !hasVolumeFile(second, "main.py") || hasVolumeFile(second, "notas.txt") {
		t.Fatalf("el recuperatorio debía tener solo el proyecto base: %+v", second)
	}

	first, err := client.SnapshotPracticeEnvironmentVolumeStrict(ctx, firstAttempt, 4*1024*1024)
	if err != nil {
		t.Fatalf("no se pudo leer el primer intento: %v", err)
	}
	if !hasVolumeFile(first, "notas.txt") {
		t.Fatalf("el intento anterior debía conservar su trabajo: %+v", first)
	}

	practiceFiles, err := client.SnapshotPracticeEnvironmentVolumeStrict(ctx, practice, 4*1024*1024)
	if err != nil {
		t.Fatalf("no se pudo leer la práctica: %v", err)
	}
	if !hasVolumeFile(practiceFiles, "main.py") {
		t.Fatalf("la práctica debía conservar su contenido: %+v", practiceFiles)
	}
}

func hasVolumeFile(files []domain.PracticeEnvironmentVolumeFile, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}
