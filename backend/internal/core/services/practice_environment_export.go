package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"solv-backend/internal/core/domain"
)

// exportBundleMaxConcurrent acota cuántas exportaciones se generan a la vez.
// Un .zip se arma en memoria: dos alumnos exportando a la vez en el nodo de
// 12 GB ya son un pico razonable, y la tercera espera en vez de competir con
// los builds y los laboratorios.
const exportBundleMaxConcurrent = 2

var (
	// ErrPracticeEnvironmentExportExamActive: un examen abierto no se exporta.
	// El material del examen no sale del aula mientras se está rindiendo; una
	// vez cerrado, el alumno sí puede llevarse su trabajo.
	ErrPracticeEnvironmentExportExamActive = errors.New("no se puede exportar el entorno durante un examen activo")
	// ErrPracticeEnvironmentExportBusy: ya hay exportaciones en curso.
	ErrPracticeEnvironmentExportBusy = errors.New("la exportación está ocupada, intente de nuevo en unos segundos")
)

// stackDependencyProfileReader es una capacidad opcional del repositorio de
// plantillas: resuelve, por slug de componente, cómo instala dependencias el
// stack. Sin ella el README explica la instalación en términos generales.
type stackDependencyProfileReader interface {
	GetDependencyProfileForStack(ctx context.Context, slugs []string) (domain.DependencyProfile, bool, error)
}

// PracticeEnvironmentExport es el paquete listo para descargar.
type PracticeEnvironmentExport struct {
	Filename       string
	Content        []byte
	WorkspaceFiles int
	GeneratedPaths []string
	LocalDir       string
}

// ExportPracticeEnvironment arma el proyecto local del alumno a partir del
// volumen de su práctica. Funciona con el entorno corriendo, pausado o
// hibernado: la lectura del volumen no depende del contenedor.
func (s *PracticeEnvironmentService) ExportPracticeEnvironment(ctx context.Context, environmentID, studentID string) (*PracticeEnvironmentExport, error) {
	environmentID = strings.TrimSpace(environmentID)
	studentID = strings.TrimSpace(studentID)
	if environmentID == "" || studentID == "" {
		return nil, fmt.Errorf("%w: environment %s", domain.ErrNotFound, environmentID)
	}

	release, ok := s.acquireExportSlot()
	if !ok {
		return nil, ErrPracticeEnvironmentExportBusy
	}
	defer release()

	environment, err := s.repo.GetByID(ctx, environmentID)
	if err != nil || environment == nil {
		return nil, fmt.Errorf("%w: environment %s", domain.ErrNotFound, environmentID)
	}
	// La propiedad se decide acá y no en el transporte: el endpoint es del
	// alumno y nadie exporta el trabajo de otro, ni siquiera con un id válido.
	if environment.StudentID != studentID {
		return nil, ErrPracticeEnvironmentForbidden
	}
	if environment.EnvironmentProfile.IsExam() &&
		environment.FrozenAt == nil &&
		environment.Status != domain.PracticeEnvironmentStatusFrozen {
		return nil, ErrPracticeEnvironmentExportExamActive
	}

	volumeStore, ok := s.docker.(domain.PracticeEnvironmentVolumeStore)
	if !ok {
		return nil, ErrPracticeEnvironmentVolumeUnavailable
	}
	volumeName := environment.VolumeName()
	if strings.TrimSpace(volumeName) == "" {
		return nil, domain.ErrPracticeEnvironmentPracticeRequired
	}

	// Lectura estricta cuando el adaptador la soporta: un paquete recortado en
	// silencio no es reproducible y el alumno no lo sabría.
	files, err := snapshotForSubmission(ctx, volumeStore, volumeName, domain.ExportBundleMaxWorkspaceBytes)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el proyecto del entorno: %w", err)
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = environment.TenantID
	}
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	exercise := s.exportExercise(ctx, environment, tenantID)
	template := s.exportTemplate(ctx, environment)

	input := domain.ExportBundleInput{
		EnvironmentName: exportEnvironmentName(exercise),
		Files:           files,
	}
	if template != nil {
		input.TemplateName = template.Name
		input.TemplateImage = template.DockerImage
		input.DockerfileContent = template.StackConfig.DockerfileContent
		input.Services = template.ServicesConfig.Services
		input.DependencyProfile = s.exportDependencyProfile(ctx, template)
	}
	if exercise != nil && exercise.Config.IDE != nil {
		input.AppRunCommand = exercise.Config.IDE.AppRunCommand
		input.AppPort = exercise.Config.IDE.AppPort
	}

	bundle, err := domain.BuildExportBundle(input)
	if err != nil {
		return nil, err
	}
	return &PracticeEnvironmentExport{
		Filename:       exportBundleFilename(input.EnvironmentName, environment.ID),
		Content:        bundle.Content,
		WorkspaceFiles: bundle.WorkspaceFiles,
		GeneratedPaths: bundle.GeneratedPaths,
		LocalDir:       bundle.LocalDir,
	}, nil
}

// acquireExportSlot limita las exportaciones simultáneas sin bloquear al
// alumno: la tercera recibe un aviso en vez de una espera larga.
func (s *PracticeEnvironmentService) acquireExportSlot() (func(), bool) {
	s.exportMu.Lock()
	if s.exportInFlight >= exportBundleMaxConcurrent {
		s.exportMu.Unlock()
		return nil, false
	}
	s.exportInFlight++
	s.exportMu.Unlock()
	return func() {
		s.exportMu.Lock()
		if s.exportInFlight > 0 {
			s.exportInFlight--
		}
		s.exportMu.Unlock()
	}, true
}

// exportExercise resuelve la práctica del entorno para tomar su título, su
// puerto de aplicación y el comando de arranque declarado por el docente.
func (s *PracticeEnvironmentService) exportExercise(ctx context.Context, environment *domain.PracticeEnvironment, tenantID string) *domain.Exercise {
	if environment == nil || environment.ExerciseID == nil || s.exerciseRepo == nil {
		return nil
	}
	exerciseID := strings.TrimSpace(*environment.ExerciseID)
	if exerciseID == "" {
		return nil
	}
	exercise, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil || exercise == nil {
		log.Printf("[Export] entorno %s: no se pudo leer la práctica %s: %v", environment.ID, exerciseID, err)
		return nil
	}
	return exercise
}

// exportTemplate resuelve la plantilla viva del entorno. Es opcional: sin ella
// el paquete igual contiene el código y un README sin detalles del stack.
func (s *PracticeEnvironmentService) exportTemplate(ctx context.Context, environment *domain.PracticeEnvironment) *domain.Template {
	if environment == nil || s.templateRepo == nil {
		return nil
	}
	if environment.TemplateID != nil && strings.TrimSpace(*environment.TemplateID) != "" {
		if template, err := s.templateRepo.GetTemplateByID(ctx, strings.TrimSpace(*environment.TemplateID)); err == nil && template != nil {
			return template
		}
	}
	return s.resolveTemplate(ctx, environment.SubjectID)
}

// exportDependencyProfile busca el perfil de la familia tecnológica del stack.
// Un repositorio sin esa capacidad no rompe la exportación: el README cae a
// instrucciones generales.
func (s *PracticeEnvironmentService) exportDependencyProfile(ctx context.Context, template *domain.Template) *domain.DependencyProfile {
	if template == nil {
		return nil
	}
	reader, ok := s.templateRepo.(stackDependencyProfileReader)
	if !ok {
		return nil
	}
	var slugs []string
	stack := template.StackConfig
	for _, slug := range append([]string{stack.RuntimeID, stack.WebPlatformID}, stack.AddonIDs...) {
		if trimmed := strings.TrimSpace(slug); trimmed != "" {
			slugs = append(slugs, trimmed)
		}
	}
	if len(slugs) == 0 {
		return nil
	}
	profile, found, err := reader.GetDependencyProfileForStack(ctx, slugs)
	if err != nil {
		log.Printf("[Export] plantilla %s: no se pudo leer el perfil de dependencias: %v", template.ID, err)
		return nil
	}
	if !found {
		return nil
	}
	return &profile
}

// exportEnvironmentName elige el nombre del paquete: el título de la práctica
// manda, porque es lo que el alumno reconoce.
func exportEnvironmentName(exercise *domain.Exercise) string {
	if exercise != nil {
		if title := strings.TrimSpace(exercise.Title); title != "" {
			return title
		}
	}
	return "Proyecto de práctica"
}

// exportBundleFilename arma un nombre estable y seguro: slug del título más un
// sufijo del entorno para que dos prácticas homónimas no se pisen al descargar.
func exportBundleFilename(title, environmentID string) string {
	slug := domain.SanitizeExportFilename(title)
	if slug == "" {
		slug = "entorno-solv"
	}
	if len(environmentID) >= 8 {
		slug += "-" + environmentID[:8]
	}
	return "solv-" + slug + ".zip"
}
