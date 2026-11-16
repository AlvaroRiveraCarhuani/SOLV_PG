package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Resultado de la verificación previa a publicar un examen sin salida. El
// estado vive en el ejercicio para que la pantalla de publicación lo muestre
// sin repetir la corrida.
const (
	ExamOfflinePassed  = "passed"
	ExamOfflineFailed  = "failed"
	ExamOfflineSkipped = "skipped"
)

// ExamProjectOfflineCheck es lo que hay que correr para saber si el proyecto
// base funciona sin red: la imagen, el script de preparación, el comando de
// arranque declarado y los archivos de la versión publicada.
type ExamProjectOfflineCheck struct {
	Image         string
	SetupScript   string
	RunCommand    string
	Files         map[string]string
	MemoryLimitMB int
}

// ExamProjectOfflineCheckResult describe qué pasó. Step nombra el paso que
// falló (setup, run, proyecto, tiempo) para que el mensaje diga dónde cortó.
type ExamProjectOfflineCheckResult struct {
	Status     string
	Step       string
	Output     string
	DurationMS int64
}

// ExamOfflineVerifier corre un proyecto base en un contenedor sin salida. La
// infraestructura decide cómo; el servicio solo interpreta el resultado.
type ExamOfflineVerifier interface {
	VerifyExamProjectOffline(ctx context.Context, check ExamProjectOfflineCheck) (*ExamProjectOfflineCheckResult, error)
}

// ErrExamOfflineVerification marca el rechazo de publicación por un proyecto
// base que necesita red. El handler lo traduce a 422 con el detalle del paso.
type ExamOfflineVerificationError struct {
	Step   string
	Output string
}

func (e *ExamOfflineVerificationError) Error() string {
	step := strings.TrimSpace(e.Step)
	if step == "" {
		step = "proyecto"
	}
	output := strings.TrimSpace(e.Output)
	if output == "" {
		return fmt.Sprintf("el proyecto base no funciona sin red: falló el paso %q", step)
	}
	if len(output) > 600 {
		output = "…" + output[len(output)-600:]
	}
	return fmt.Sprintf("el proyecto base no funciona sin red: falló el paso %q. Últimas líneas: %s", step, output)
}

// ErrExamOfflineUnavailable deja constancia de que la verificación no se pudo
// ejecutar. No bloquea la publicación: un fallo de infraestructura no es prueba
// de que el proyecto necesite red.
var ErrExamOfflineUnavailable = errors.New("exam offline verification unavailable")
