package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
)

// PracticeSatelliteProvision reúne lo que la prueba de una práctica necesita para
// alcanzar sus bases satélite: credenciales aisladas, red y motores involucrados.
type PracticeSatelliteProvision struct {
	Env         []string
	NetworkName string
	Engines     []string
}

// PracticeSatelliteProvisioner asegura los satélites de una práctica con una
// identidad de prueba estable, sin crear un workspace ni una entrega.
type PracticeSatelliteProvisioner interface {
	ProvisionPracticeSatellites(ctx context.Context, exercise *domain.Exercise, template *domain.Template, subjectID string) (PracticeSatelliteProvision, error)
}

// practiceTestStudentID deriva una identidad de prueba distinta de cualquier UUID de
// alumno. sanitizeIDForDB recorta a 8 caracteres, así que la parte única va primero.
func practiceTestStudentID(exerciseID string) string {
	cleaned := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(exerciseID), "-", ""))
	if len(cleaned) < 7 {
		cleaned += strings.Repeat("0", 7-len(cleaned))
	}
	return "t" + cleaned[:7]
}

// ProvisionPracticeSatellites levanta los motores declarados por la plantilla y aplica
// el db_init_script del ejercicio en modo hard-fail. La base de prueba queda aislada por
// práctica y se reutiliza entre pruebas; nunca se mezcla con la de un alumno.
func (s *WorkspaceService) ProvisionPracticeSatellites(
	ctx context.Context,
	exercise *domain.Exercise,
	template *domain.Template,
	subjectID string,
) (PracticeSatelliteProvision, error) {
	if exercise == nil {
		return PracticeSatelliteProvision{}, errors.New("el ejercicio de la práctica es requerido")
	}
	if strings.TrimSpace(subjectID) == "" {
		return PracticeSatelliteProvision{}, errors.New("la materia de la práctica es requerida")
	}

	provision, err := s.ensureExerciseSatelliteDB(ctx, practiceTestStudentID(exercise.ID), subjectID, template, exercise)
	if err != nil {
		return PracticeSatelliteProvision{}, err
	}

	engines := templateManagedEngines(template)
	if len(engines) == 0 && len(provision.Env) > 0 {
		// Un db_init_script sin servicios declarados conserva el PostgreSQL de aula,
		// ahora como satélite dedicado.
		engines = []string{"postgres"}
	}

	networkName := ""
	if provision.UsesManagedContainer {
		networkName = "solv-satellites-net"
	}

	return PracticeSatelliteProvision{
		Env:         provision.Env,
		NetworkName: networkName,
		Engines:     engines,
	}, nil
}

// practiceSatelliteTools convierte los motores asegurados en filas del reporte.
func practiceSatelliteTools(engines []string) []domain.ToolResult {
	results := make([]domain.ToolResult, 0, len(engines))
	for _, engine := range engines {
		engine = strings.TrimSpace(engine)
		if engine == "" {
			continue
		}
		results = append(results, domain.ToolResult{
			Name:    fmt.Sprintf("satélite %s", engine),
			Present: true,
			Path:    "solv-satellites-net",
			Version: "base aislada por práctica",
		})
	}
	return results
}
