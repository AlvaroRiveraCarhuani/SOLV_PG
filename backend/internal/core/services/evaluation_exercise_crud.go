package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
	"strings"
)

func (s *EvaluationService) validateExerciseInputFormat(ex *domain.Exercise) error {
	if ex == nil || s.formatValidator == nil {
		return nil
	}
	var inputFormat json.RawMessage
	if ex.Config.Algorithm != nil && len(ex.Config.Algorithm.InputFormat) > 0 && string(ex.Config.Algorithm.InputFormat) != "null" {
		inputFormat = ex.Config.Algorithm.InputFormat
	} else if len(ex.Config.InputFormat) > 0 && string(ex.Config.InputFormat) != "null" {
		inputFormat = ex.Config.InputFormat
	}

	if len(inputFormat) == 0 || string(inputFormat) == "null" || string(inputFormat) == "{}" {
		return nil
	}

	if err := s.formatValidator.ValidateContract(inputFormat); err != nil {
		return fmt.Errorf("contrato de formato inválido: %w", err)
	}

	if ex.Config.Algorithm != nil {
		for idx, tc := range ex.Config.Algorithm.TestCases {
			ok, msg := s.formatValidator.ValidateCase(inputFormat, tc.Input)
			if !ok {
				return fmt.Errorf("caso %d inválido según contrato de formato: %s", idx+1, msg)
			}
		}
	}
	return nil
}

func (s *EvaluationService) getHostTotalRAM(ctx context.Context) int {
	totalMB := 8192
	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil && v.Total > 0 {
		totalMB = domain.Uint64ToInt(v.Total / (1024 * 1024))
	}
	return totalMB
}

var ErrZeroPublicTestCases = fmt.Errorf("cannot publish exercise with 0 public test cases")

// environmentSnapshotRepository es la capacidad opcional que persiste el pin
// del entorno junto con el estado. El repositorio Postgres la implementa; los
// dobles de prueba que no la implementan conservan la publicación sin pin.
type environmentSnapshotRepository interface {
	UpdateStatusWithEnvironmentSnapshot(ctx context.Context, id, tenantID, status string, snapshot *domain.EnvironmentSnapshot) error
}

// ErrDBInitScriptUnsupported rechaza un db_init_script que el primario declarado por la
// plantilla no sabe aplicar.
var ErrDBInitScriptUnsupported = errors.New("db_init_script no soportado por el motor primario de la plantilla")

// validateExerciseDBInitScript exige que el primario de la plantilla tenga un init
// declarado (sql o mongo-js). Una plantilla sin bindings conserva la ruta heredada de
// PostgreSQL de aula; SQLite, Redis o un primario ausente no admiten init.
func validateExerciseDBInitScript(ex *domain.Exercise, template *domain.AdminTemplateReviewItem) error {
	if ex == nil || template == nil || exerciseDBInitScript(ex) == "" {
		return nil
	}

	bindings := templateServiceBindings(&domain.Template{ServicesConfig: template.ServicesConfig})
	if len(bindings) == 0 {
		return nil
	}
	for _, binding := range bindings {
		if binding.EffectiveRole() != domain.ServiceRolePrimaryDatabase {
			continue
		}
		if kind := binding.InitKind(); kind == domain.ServiceInitSQL || kind == domain.ServiceInitMongoJS {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrDBInitScriptUnsupported, template.Name)
}

func (s *EvaluationService) resolveAndApplyTemplate(ctx context.Context, ex *domain.Exercise) (*domain.AdminTemplateReviewItem, error) {
	if s.adminGovRepo == nil {
		return nil, nil
	}
	if ex.EnvironmentType == "" {
		ex.EnvironmentType = string(domain.EnvironmentTypeJuezEfimero)
	}

	var template *domain.AdminTemplateReviewItem
	var err error

	if ex.TemplateID != "" {
		template, err = s.adminGovRepo.GetTemplateByID(ctx, ex.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve template %s: %w", ex.TemplateID, err)
		}
	} else if ex.Language != "" {
		template, err = s.adminGovRepo.GetTemplateForLanguage(ctx, ex.EnvironmentType, ex.Language)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve default template for language %s: %w", ex.Language, err)
		}
	}

	if template != nil {
		if !strings.EqualFold(template.Status, "approved") && !strings.EqualFold(template.Status, "APROBADA") {
			return nil, domain.ErrTemplateNotApproved
		}
		if !domain.TemplateSupportsEnvironment(template.TargetEnvironment, ex.EnvironmentType) {
			return nil, domain.ErrTemplateEnvironmentMismatch
		}
		if err := validateExerciseDBInitScript(ex, template); err != nil {
			return nil, err
		}

		ex.TemplateID = template.ID
		ramMB := template.BaseRamMB
		if template.TargetEnvironment == domain.TargetEnvironmentAmbos && ex.EnvironmentType == domain.TargetEnvironmentJuezEfimero {
			ramMB = domain.ResolveJudgeRamMB(template.BaseRamMB)
		}
		ex.MemoryLimitMB = ramMB
		if ex.Config.Algorithm != nil {
			ex.Config.Algorithm.MemoryLimitMB = ramMB
		}
		ex.Template = &domain.TemplateSummary{
			ID:                template.ID,
			Name:              template.Name,
			DockerImage:       template.DockerImage,
			Entrypoint:        template.Entrypoint,
			TimeoutMS:         template.TimeoutMS,
			BaseRamMB:         template.BaseRamMB,
			TargetEnvironment: template.TargetEnvironment,
			ServicesConfig:    template.ServicesConfig,
		}
	}
	return template, nil
}

func (s *EvaluationService) attachTemplateSummary(ctx context.Context, ex *domain.Exercise) {
	if s.adminGovRepo == nil || ex == nil {
		return
	}
	if ex.EnvironmentSnapshot != nil && !strings.EqualFold(ex.Status, "draft") {
		ex.Template = ex.EnvironmentSnapshot.AsTemplateSummary()
		return
	}
	var tpl *domain.AdminTemplateReviewItem
	var err error
	if ex.TemplateID != "" {
		tpl, err = s.adminGovRepo.GetTemplateByID(ctx, ex.TemplateID)
	} else if ex.Language != "" {
		envType := ex.EnvironmentType
		if envType == "" {
			envType = string(domain.EnvironmentTypeJuezEfimero)
		}
		tpl, err = s.adminGovRepo.GetTemplateForLanguage(ctx, envType, ex.Language)
	}
	if err == nil && tpl != nil {
		ex.Template = &domain.TemplateSummary{
			ID:                tpl.ID,
			Name:              tpl.Name,
			DockerImage:       tpl.DockerImage,
			Entrypoint:        tpl.Entrypoint,
			TimeoutMS:         tpl.TimeoutMS,
			BaseRamMB:         tpl.BaseRamMB,
			TargetEnvironment: tpl.TargetEnvironment,
			ServicesConfig:    tpl.ServicesConfig,
		}
	}
}

func (s *EvaluationService) GetExerciseByID(ctx context.Context, id string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByID(ctx, id)
	if err != nil || ex == nil {
		return ex, err
	}
	s.attachTemplateSummary(ctx, ex)
	return ex, nil
}

func (s *EvaluationService) GetExerciseByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil || ex == nil {
		return ex, err
	}
	s.attachTemplateSummary(ctx, ex)
	return ex, nil
}

// assertExerciseSubjectWritable acota una escritura de ejercicio a la materia
// que la contiene. Un ejercicio sin materia no tiene pertenencia que violar; uno
// que vive en la materia de un colega responde igual que si no existiera.
func (s *EvaluationService) assertExerciseSubjectWritable(ctx context.Context, subjectID *string) error {
	if subjectID == nil || *subjectID == "" {
		return nil
	}
	return assertActorWritesSubject(ctx, s.ownership, *subjectID, domain.ErrNotFound)
}

// assertExerciseUpdateWritable revisa la materia actual y la de destino: mover un
// ejercicio a la materia de un colega es la misma escritura cruzada que editarlo.
func (s *EvaluationService) assertExerciseUpdateWritable(ctx context.Context, ex *domain.Exercise) error {
	teacherID, _ := domain.TeacherWriteScope(ctx)
	if teacherID == "" {
		return nil
	}

	current, err := s.exerciseRepo.GetByIDAndTenant(ctx, ex.ID, ex.TenantID)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	if err := s.assertExerciseSubjectWritable(ctx, current.SubjectID); err != nil {
		return err
	}
	if ex.SubjectID != nil && (current.SubjectID == nil || *current.SubjectID != *ex.SubjectID) {
		return s.assertExerciseSubjectWritable(ctx, ex.SubjectID)
	}
	return nil
}

func (s *EvaluationService) CreateExercise(ctx context.Context, ex *domain.Exercise) error {
	if err := s.assertExerciseSubjectWritable(ctx, ex.SubjectID); err != nil {
		return err
	}
	if ex.ID == "" {
		ex.ID = uuid.NewString()
	}
	s.normalizeTimeLimit(ex)
	if err := ex.Validate(); err != nil {
		return err
	}
	if err := s.validateExerciseInputFormat(ex); err != nil {
		return err
	}
	if _, err := s.resolveAndApplyTemplate(ctx, ex); err != nil {
		return err
	}
	if ex.Config.Algorithm != nil {
		for i := range ex.Config.Algorithm.TestCases {
			ex.Config.Algorithm.TestCases[i].Normalize()
			if err := ex.Config.Algorithm.TestCases[i].Validate(); err != nil {
				return err
			}
		}
	}
	if ex.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(ex.MemoryLimitMB, maxAllowed); err != nil {
			return err
		}
	}
	if err := s.exerciseRepo.Create(ctx, ex); err != nil {
		return err
	}
	s.recordExamNetworkException(ctx, ex)
	return nil
}

// recordExamNetworkException deja en auditoría la excepción de salida del
// examen con el motivo que escribió el docente. Best-effort, como el resto.
func (s *EvaluationService) recordExamNetworkException(ctx context.Context, ex *domain.Exercise) {
	if ex == nil || ex.ExamNetwork != domain.ExamNetworkEgress || s.purposeAudit == nil {
		return
	}
	actorID, _ := domain.TeacherWriteScope(ctx)
	if strings.TrimSpace(actorID) == "" {
		actorID = domain.GetUserID(ctx)
	}
	tenantID := ex.TenantID
	if strings.TrimSpace(tenantID) == "" {
		tenantID = domain.GetTenantID(ctx)
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(tenantID) == "" {
		return
	}
	meta, err := json.Marshal(map[string]any{
		"reason":  ex.ExamNetworkReason,
		"purpose": ex.Purpose,
	})
	if err != nil {
		return
	}
	resourceID := ex.ID
	_ = s.purposeAudit.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       "EXAM_NETWORK_EGRESS_GRANTED",
		ResourceType: "exercise",
		ResourceID:   &resourceID,
		Metadata:     meta,
	})
}

// normalizeTimeLimit establece la columna time_limit_ms como fuente de verdad
// del límite por caso y la sincroniza en config.algorithm.time_limit_ms, que
// es lo que consume el runner (evaluation_algorithm_runner) y el dry-run.
// Así un payload con config.algorithm.time_limit_ms divergente queda cubierto
// por la misma validación de rango (100..30.000 ms, ver ErrTimeLimitOutOfRange).
func (s *EvaluationService) normalizeTimeLimit(ex *domain.Exercise) {
	if ex.TimeLimitMS <= 0 {
		ex.TimeLimitMS = 1000
	}
	if ex.Config.Algorithm != nil {
		ex.Config.Algorithm.TimeLimitMS = ex.TimeLimitMS
	}
}

// PurposeChangeConfirmation acompaña la edición cuando el docente cambia el
// propósito con entornos ya cerrados. No forma parte del ejercicio: el cuerpo
// del ejercicio no gana campos que no se guardan.
type PurposeChangeConfirmation struct {
	Confirmed bool
	Reason    string
}

var (
	// ErrExercisePurposeChangeBlocked impide cambiar el propósito con entornos
	// abiertos: el entorno abierto conserva el perfil con el que nació, así que el
	// alumno quedaría en un examen sin las reglas del examen, o al revés.
	ErrExercisePurposeChangeBlocked = errors.New("no se puede cambiar el propósito con entornos abiertos")
	// ErrExercisePurposeChangeConfirmationNeeded pide confirmación cuando el
	// cambio alcanza entornos cerrados.
	ErrExercisePurposeChangeConfirmationNeeded = errors.New("cambiar el propósito con entornos cerrados pide confirmación")
	// ErrExercisePurposeChangeReasonNeeded pide el motivo de esa confirmación.
	ErrExercisePurposeChangeReasonNeeded = errors.New("la confirmación del cambio de propósito pide motivo")
)

// environmentStatusCounter es una capacidad opcional del repositorio de
// entornos: cuenta los de un ejercicio por estado.
type environmentStatusCounter interface {
	CountByExerciseAndStatus(ctx context.Context, tenantID, exerciseID, status string) (int, error)
}

// purposeChangeDecision es el cambio aceptado, listo para auditar.
type purposeChangeDecision struct {
	from       string
	to         string
	hibernated int
	reason     string
}

// decidePurposeChange concentra la regla, sin repositorios: con entornos
// abiertos no hay cambio; con entornos cerrados hace falta confirmación y
// motivo; sin entornos, el cambio es libre.
func decidePurposeChange(from, to string, running, hibernated int, confirmation PurposeChangeConfirmation) (*purposeChangeDecision, error) {
	from = strings.ToLower(strings.TrimSpace(from))
	to = strings.ToLower(strings.TrimSpace(to))
	if to == "" || from == to {
		return nil, nil
	}
	if running > 0 {
		return nil, fmt.Errorf("%w: %d entornos abiertos", ErrExercisePurposeChangeBlocked, running)
	}
	if hibernated == 0 {
		return &purposeChangeDecision{from: from, to: to}, nil
	}
	if !confirmation.Confirmed {
		return nil, fmt.Errorf("%w: %d entornos cerrados", ErrExercisePurposeChangeConfirmationNeeded, hibernated)
	}
	if strings.TrimSpace(confirmation.Reason) == "" {
		return nil, ErrExercisePurposeChangeReasonNeeded
	}
	return &purposeChangeDecision{from: from, to: to, hibernated: hibernated, reason: strings.TrimSpace(confirmation.Reason)}, nil
}

func (s *EvaluationService) UpdateExercise(ctx context.Context, ex *domain.Exercise) error {
	return s.UpdateExerciseWithPurposeConfirmation(ctx, ex, PurposeChangeConfirmation{})
}

func (s *EvaluationService) UpdateExerciseWithPurposeConfirmation(ctx context.Context, ex *domain.Exercise, confirmation PurposeChangeConfirmation) error {
	if err := s.assertExerciseUpdateWritable(ctx, ex); err != nil {
		return err
	}
	decision, err := s.checkPurposeChange(ctx, ex, confirmation)
	if err != nil {
		return err
	}
	s.normalizeTimeLimit(ex)
	if err := ex.Validate(); err != nil {
		return err
	}
	if err := s.validateExerciseInputFormat(ex); err != nil {
		return err
	}
	if _, err := s.resolveAndApplyTemplate(ctx, ex); err != nil {
		return err
	}
	if ex.Config.Algorithm != nil {
		for i := range ex.Config.Algorithm.TestCases {
			ex.Config.Algorithm.TestCases[i].Normalize()
			if err := ex.Config.Algorithm.TestCases[i].Validate(); err != nil {
				return err
			}
		}
	}
	if ex.MemoryLimitMB > 0 {
		maxAllowed := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
		if err := domain.ValidateRamAgainstHost(ex.MemoryLimitMB, maxAllowed); err != nil {
			return err
		}
	}

	// El espejo de un ejercicio con proyecto no se puede reemplazar por la vía
	// genérica de edición: la base publicada y las entregas deben coincidir.
	if s.workspaceProjects != nil {
		hasProject, err := s.workspaceProjects.HasProject(ctx, ex.TenantID, ex.ID)
		if err != nil {
			return err
		}
		if hasProject {
			current, err := s.exerciseRepo.GetByIDAndTenant(ctx, ex.ID, ex.TenantID)
			if err != nil {
				return err
			}
			if current != nil && ex.Boilerplate != current.Boilerplate {
				return domain.ErrWorkspaceProjectManagedBoilerplate
			}
		}
	}

	if err := s.exerciseRepo.Update(ctx, ex); err != nil {
		return err
	}
	s.recordPurposeChange(ctx, ex, decision)
	s.recordExamNetworkException(ctx, ex)
	return nil
}

// checkPurposeChange decide si el propósito puede cambiar y con qué recaudos.
func (s *EvaluationService) checkPurposeChange(ctx context.Context, ex *domain.Exercise, confirmation PurposeChangeConfirmation) (*purposeChangeDecision, error) {
	if s.environments == nil || s.exerciseRepo == nil {
		return nil, nil
	}
	current, err := s.exerciseRepo.GetByIDAndTenant(ctx, ex.ID, ex.TenantID)
	if err != nil || current == nil {
		return nil, nil
	}
	if strings.EqualFold(strings.TrimSpace(current.Purpose), strings.TrimSpace(ex.Purpose)) {
		return nil, nil
	}
	// Un ejercicio sin propósito explícito se comporta como práctica: si el
	// cuerpo no lo manda, se compara contra ese valor para no perder el cambio.
	if strings.TrimSpace(ex.Purpose) == "" {
		ex.Purpose = string(domain.ExercisePurposeClass)
	}
	counter, ok := s.environments.(environmentStatusCounter)
	if !ok {
		return nil, nil
	}
	running, err := counter.CountByExerciseAndStatus(ctx, ex.TenantID, ex.ID, string(domain.PracticeEnvironmentStatusRunning))
	if err != nil {
		return nil, err
	}
	hibernated, err := counter.CountByExerciseAndStatus(ctx, ex.TenantID, ex.ID, string(domain.PracticeEnvironmentStatusHibernated))
	if err != nil {
		return nil, err
	}
	return decidePurposeChange(current.Purpose, ex.Purpose, running, hibernated, confirmation)
}

// recordPurposeChange deja el cambio en la auditoría con el motivo que escribió
// el docente. Es best-effort, como el resto del proyecto.
func (s *EvaluationService) recordPurposeChange(ctx context.Context, ex *domain.Exercise, decision *purposeChangeDecision) {
	if decision == nil || s.purposeAudit == nil {
		return
	}
	actorID, _ := domain.TeacherWriteScope(ctx)
	if strings.TrimSpace(actorID) == "" {
		// Un administrador no viene con alcance docente: el actor autenticado es
		// el que queda en la auditoría.
		actorID = domain.GetUserID(ctx)
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(ex.TenantID) == "" {
		return
	}
	meta, err := json.Marshal(map[string]any{
		"from":       decision.from,
		"to":         decision.to,
		"hibernated": decision.hibernated,
		"reason":     decision.reason,
	})
	if err != nil {
		return
	}
	resourceID := ex.ID
	_ = s.purposeAudit.Create(ctx, &domain.AuditLog{
		TenantID:     ex.TenantID,
		ActorID:      actorID,
		Action:       "EXERCISE_PURPOSE_CHANGED",
		ResourceType: "exercise",
		ResourceID:   &resourceID,
		Metadata:     meta,
	})
}

func (s *EvaluationService) BulkAddTestCases(ctx context.Context, exerciseID, tenantID string, testCases []domain.TestCase) error {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return fmt.Errorf("exercise not found: %w", err)
	}
	if err := s.assertExerciseSubjectWritable(ctx, ex.SubjectID); err != nil {
		return err
	}

	if ex.Config.Algorithm == nil {
		ex.Config.Algorithm = &domain.AlgorithmConfig{
			TimeLimitMS:   ex.TimeLimitMS,
			MemoryLimitMB: ex.MemoryLimitMB,
		}
	}
	// Invariante de modalidad: los casos de prueba I/O pertenecen al juez
	// virtual (JUEZ_EFIMERO). Los laboratorios IDE_PERSISTENTE se corrigen con
	// rúbrica en SpeedGrader, no contra casos I/O.
	if ex.EnvironmentType == string(domain.EnvironmentTypeIDEPersistente) {
		return domain.ErrTestCasesRequireJudge
	}
	for i := range testCases {
		testCases[i].Normalize()
		if err := testCases[i].Validate(); err != nil {
			return err
		}
	}
	ex.Config.Algorithm.TestCases = append(ex.Config.Algorithm.TestCases, testCases...)
	if err := s.validateExerciseInputFormat(ex); err != nil {
		return err
	}
	return s.exerciseRepo.Update(ctx, ex)
}

func (s *EvaluationService) PublishExercise(ctx context.Context, exerciseID, tenantID string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}
	if err := s.assertExerciseSubjectWritable(ctx, ex.SubjectID); err != nil {
		return nil, err
	}

	template, err := s.resolveAndApplyTemplate(ctx, ex)
	if err != nil {
		return nil, err
	}

	if ex.Type == domain.ExerciseTypeAlgorithm {
		publicCount := 0
		if ex.Config.Algorithm != nil {
			for i := range ex.Config.Algorithm.TestCases {
				tc := &ex.Config.Algorithm.TestCases[i]
				tc.Normalize()
				if tc.Visibility != domain.TestCaseVisibilityHidden {
					publicCount++
				}
			}
		}

		if publicCount == 0 {
			return nil, ErrZeroPublicTestCases
		}

		if strings.TrimSpace(ex.ReferenceSolution) == "" {
			return nil, domain.ErrMissingReferenceSolution
		}
		if ex.Stale {
			return nil, domain.ErrExerciseStale
		}
	}

	if err := s.verifyExamProjectOffline(ctx, ex, template); err != nil {
		return nil, err
	}

	snapshot := buildEnvironmentSnapshot(template)
	if repo, ok := s.exerciseRepo.(environmentSnapshotRepository); ok {
		if err := repo.UpdateStatusWithEnvironmentSnapshot(ctx, exerciseID, tenantID, "published", snapshot); err != nil {
			return nil, fmt.Errorf("failed to publish exercise: %w", err)
		}
	} else if err := s.exerciseRepo.UpdateStatus(ctx, exerciseID, tenantID, "published"); err != nil {
		return nil, fmt.Errorf("failed to publish exercise: %w", err)
	}

	ex.Status = "published"
	ex.EnvironmentSnapshot = snapshot
	return ex, nil
}

// buildEnvironmentSnapshot congela la decisión efectiva de la plantilla. Los
// bindings se ordenan con la misma regla que usa el arranque (primario primero)
// para que la huella sea estable entre publicaciones equivalentes.
func buildEnvironmentSnapshot(template *domain.AdminTemplateReviewItem) *domain.EnvironmentSnapshot {
	if template == nil {
		return nil
	}
	snapshot := &domain.EnvironmentSnapshot{
		TemplateID:        template.ID,
		TemplateName:      template.Name,
		DockerImage:       template.DockerImage,
		Entrypoint:        template.Entrypoint,
		TimeoutMS:         template.TimeoutMS,
		BaseRAMMB:         template.BaseRamMB,
		TargetEnvironment: template.TargetEnvironment,
		SetupScript:       template.SetupScript,
	}
	bindings := templateServiceBindings(&domain.Template{ServicesConfig: template.ServicesConfig})
	snapshot.Bindings = make([]domain.EnvironmentSnapshotBinding, 0, len(bindings))
	snapshot.InitContract = make([]domain.EnvironmentInitContract, 0, len(bindings))
	for _, binding := range bindings {
		engine := domain.CanonicalServiceEngine(binding.Engine)
		snapshot.Bindings = append(snapshot.Bindings, domain.EnvironmentSnapshotBinding{
			Role:        binding.EffectiveRole(),
			Engine:      engine,
			ComponentID: binding.ComponentID,
			Version:     binding.Version,
			EnvAlias:    binding.EffectiveEnvAlias(),
			InitKind:    binding.InitKind(),
		})
		snapshot.InitContract = append(snapshot.InitContract, domain.EnvironmentInitContract{
			Alias:  binding.EffectiveEnvAlias(),
			Engine: engine,
			Kind:   binding.InitKind(),
		})
	}
	snapshot.VerificationFingerprint = snapshot.ComputeVerificationFingerprint()
	return snapshot
}

// UnpublishExercise devuelve la práctica a borrador. Deja de estar disponible para
// los alumnos sin borrar contenido, entregas ni las bases que ya se abrieron.
func (s *EvaluationService) UnpublishExercise(ctx context.Context, exerciseID, tenantID string) (*domain.Exercise, error) {
	ex, err := s.exerciseRepo.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exercise not found: %w", err)
	}
	if err := s.assertExerciseSubjectWritable(ctx, ex.SubjectID); err != nil {
		return nil, err
	}

	if repo, ok := s.exerciseRepo.(environmentSnapshotRepository); ok {
		if err := repo.UpdateStatusWithEnvironmentSnapshot(ctx, exerciseID, tenantID, "draft", nil); err != nil {
			return nil, fmt.Errorf("failed to unpublish exercise: %w", err)
		}
	} else if err := s.exerciseRepo.UpdateStatus(ctx, exerciseID, tenantID, "draft"); err != nil {
		return nil, fmt.Errorf("failed to unpublish exercise: %w", err)
	}

	ex.Status = "draft"
	ex.EnvironmentSnapshot = nil
	return ex, nil
}
