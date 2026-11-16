package services

import (
	"context"
	"log"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// examOfflineOutputLimit acota lo que se guarda del log de la verificación.
// Alcanza para el motivo; el log completo no aporta en la ficha del ejercicio.
const examOfflineOutputLimit = 4000

// examOfflineVerificationRepository persiste el resultado. Es opcional: los
// dobles de prueba del servicio no lo implementan y la publicación sigue.
type examOfflineVerificationRepository interface {
	RecordExamOfflineVerification(ctx context.Context, exerciseID, tenantID, status, output string) error
}

// SetExamOfflineVerifier habilita la verificación previa a publicar un examen
// sin salida. Sin verificador no se rechaza ninguna publicación.
func (s *EvaluationService) SetExamOfflineVerifier(verifier domain.ExamOfflineVerifier) {
	s.examOfflineVerifier = verifier
}

// verifyExamProjectOffline corre el proyecto base en un contenedor sin red
// cuando el examen se publica cerrado. Deja el resultado en el ejercicio y
// rechaza la publicación si el proyecto necesita instalar en línea.
func (s *EvaluationService) verifyExamProjectOffline(ctx context.Context, ex *domain.Exercise, template *domain.AdminTemplateReviewItem) error {
	if s.examOfflineVerifier == nil || ex == nil {
		return nil
	}
	if string(ex.Purpose) != string(domain.ExercisePurposeExam) ||
		domain.NormalizeExamNetwork(ex.ExamNetwork) != domain.ExamNetworkNone ||
		ex.EnvironmentType != string(domain.EnvironmentTypeIDEPersistente) {
		return nil
	}
	if template == nil || strings.TrimSpace(template.DockerImage) == "" {
		s.persistExamOffline(ctx, ex, domain.ExamOfflineSkipped, "La plantilla no declara imagen: no hay proyecto que verificar.")
		return nil
	}

	check := domain.ExamProjectOfflineCheck{
		Image:         template.DockerImage,
		SetupScript:   template.SetupScript,
		MemoryLimitMB: template.BaseRamMB,
	}
	if ex.Config.IDE != nil {
		check.RunCommand = ex.Config.IDE.AppRunCommand
	}

	files, ok := s.examBaseProjectFiles(ctx, ex)
	if !ok {
		s.persistExamOffline(ctx, ex, domain.ExamOfflineSkipped, "El ejercicio no tiene proyecto base publicado: no hay nada que verificar.")
		return nil
	}
	check.Files = files

	result, err := s.examOfflineVerifier.VerifyExamProjectOffline(ctx, check)
	if err != nil {
		log.Printf("[Exam Offline] la verificación de %s no se pudo ejecutar: %v", ex.ID, err)
		s.persistExamOffline(ctx, ex, domain.ExamOfflineSkipped, "No se pudo ejecutar la verificación: "+err.Error())
		return nil
	}
	if result == nil {
		return nil
	}

	output := strings.TrimSpace(result.Output)
	switch result.Status {
	case domain.ExamOfflineFailed:
		s.persistExamOffline(ctx, ex, domain.ExamOfflineFailed, output)
		return &domain.ExamOfflineVerificationError{Step: result.Step, Output: output}
	case domain.ExamOfflinePassed:
		s.persistExamOffline(ctx, ex, domain.ExamOfflinePassed, output)
	default:
		s.persistExamOffline(ctx, ex, domain.ExamOfflineSkipped, output)
	}
	return nil
}

// examBaseProjectFiles toma los archivos de la versión publicada más reciente y
// cae al borrador cuando todavía no hay versión. Sin proyecto, la verificación
// no aplica y no bloquea la publicación.
func (s *EvaluationService) examBaseProjectFiles(ctx context.Context, ex *domain.Exercise) (map[string]string, bool) {
	if s.workspaceProjects == nil {
		return nil, false
	}
	view, err := s.workspaceProjects.GetProject(ctx, ex.TenantID, ex.ID)
	if err != nil || view == nil {
		if err != nil {
			log.Printf("[Exam Offline] no se pudo leer el proyecto base de %s: %v", ex.ID, err)
		}
		return nil, false
	}

	snapshot := ""
	if view.Draft != nil {
		snapshot = view.Draft.TextSnapshot
	}
	highest := 0
	for _, version := range view.Versions {
		if version != nil && version.VersionNumber > highest {
			highest = version.VersionNumber
			snapshot = version.TextSnapshot
		}
	}
	if strings.TrimSpace(snapshot) == "" {
		return nil, false
	}

	parsed, ok := domain.ParseProjectSnapshot(snapshot, "")
	if !ok || len(parsed) == 0 {
		return nil, false
	}
	files := make(map[string]string, len(parsed))
	for _, file := range parsed {
		if strings.TrimSpace(file.Path) == "" {
			continue
		}
		files[file.Path] = file.Content
	}
	if len(files) == 0 {
		return nil, false
	}
	return files, true
}

// persistExamOffline deja el estado en el ejercicio y, si el repositorio lo
// soporta, lo guarda. Un fallo al guardar no cambia el resultado de la corrida.
func (s *EvaluationService) persistExamOffline(ctx context.Context, ex *domain.Exercise, status, output string) {
	output = strings.TrimSpace(output)
	if len(output) > examOfflineOutputLimit {
		output = "…" + output[len(output)-examOfflineOutputLimit:]
	}
	now := time.Now()
	ex.ExamOfflineStatus = status
	ex.ExamOfflineOutput = output
	ex.ExamOfflineCheckedAt = &now

	repo, ok := s.exerciseRepo.(examOfflineVerificationRepository)
	if !ok {
		return
	}
	if err := repo.RecordExamOfflineVerification(ctx, ex.ID, ex.TenantID, status, output); err != nil {
		log.Printf("[Exam Offline] no se pudo guardar el resultado de %s: %v", ex.ID, err)
	}
}
