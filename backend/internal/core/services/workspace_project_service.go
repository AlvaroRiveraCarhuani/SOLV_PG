package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

const (
	maxWorkspaceProjectBytes     = 1024 * 1024
	maxWorkspaceProjectFiles     = 300
	maxWorkspaceProjectFileBytes = 512 * 1024
	maxWorkspaceGenerationPrompt = 4000
	maxWorkspaceGenerationOutput = 2 * 1024 * 1024
	maxWorkspaceGenerationJSON   = 2 * 1024 * 1024
)

type WorkspaceProjectService struct {
	repo              domain.WorkspaceProjectRepository
	exercises         domain.ExerciseRepository
	subjects          domain.SubjectRepository
	aiSettings        *AISettingsService
	dependencyQueue   *DependencyPreparationQueue
	publishingEnabled bool
}

func NewWorkspaceProjectService(repo domain.WorkspaceProjectRepository, publishingEnabled bool) *WorkspaceProjectService {
	return &WorkspaceProjectService{repo: repo, publishingEnabled: publishingEnabled}
}

// SetAISettingsService conecta la generación al proveedor institucional ya
// configurado. Las credenciales y el selector de proveedor nunca llegan desde
// la petición del navegador.
func (s *WorkspaceProjectService) SetAISettingsService(service *AISettingsService) {
	if s != nil {
		s.aiSettings = service
	}
}

// SetDependencyPreparationQueue engancha la cola que prepara las dependencias
// del proyecto al publicar. Sin cola, publicar sigue funcionando y la versión
// queda sin preparación, como antes de esta pieza.
func (s *WorkspaceProjectService) SetDependencyPreparationQueue(queue *DependencyPreparationQueue) {
	if s != nil {
		s.dependencyQueue = queue
	}
}

type WorkspaceGenerationStackConfig struct {
	RuntimeID     string   `json:"runtime_id,omitempty"`
	WebPlatformID string   `json:"web_platform_id,omitempty"`
	AddonIDs      []string `json:"addon_ids,omitempty"`
	DatabaseID    string   `json:"database_id,omitempty"`
}

type GenerateWorkspaceProjectRequest struct {
	Description  string                          `json:"description"`
	Purpose      string                          `json:"purpose,omitempty"`
	Technology   string                          `json:"technology,omitempty"`
	Requirements string                          `json:"requirements,omitempty"`
	TemplateID   string                          `json:"template_id,omitempty"`
	StackConfig  *WorkspaceGenerationStackConfig `json:"stack_config,omitempty"`
}

type WorkspaceProjectProposalFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type GenerateWorkspaceProjectProposal struct {
	Files      []WorkspaceProjectProposalFile `json:"files"`
	Summary    string                         `json:"summary"`
	FileCount  int                            `json:"file_count"`
	TotalBytes int                            `json:"total_bytes"`
}

type workspaceProjectGenerationEnvelope struct {
	Files []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"files"`
	Summary string `json:"summary"`
}

// GenerateProposal devuelve archivos normalizados para revisión. Esta
// operación no crea proyecto, no guarda borrador y no publica una versión.
func (s *WorkspaceProjectService) GenerateProposal(
	ctx context.Context,
	tenantID, userID, exerciseID string,
	req GenerateWorkspaceProjectRequest,
) (*GenerateWorkspaceProjectProposal, error) {
	if s == nil || s.aiSettings == nil {
		return nil, &AISettingsValidationError{Code: "ai_provider_unavailable", Message: "La generación con IA no está disponible: no hay un proveedor configurado."}
	}
	if s.exercises == nil {
		return nil, domain.ErrWorkspaceProjectForbidden
	}
	exercise, err := s.exercises.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil || exercise == nil || exercise.TenantID != tenantID {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	if exercise.EnvironmentType != string(domain.EnvironmentTypeIDEPersistente) {
		return nil, domain.ErrWorkspaceProjectNotIDEPersistent
	}
	if err := validateWorkspaceGenerationRequest(req); err != nil {
		return nil, err
	}

	systemPrompt := `Eres un asistente que propone el contenido textual de un proyecto inicial académico.
Devuelve exclusivamente un objeto JSON con la forma {"files":[{"path":"src/main.py","content":"..."}],"summary":"..."}.
Devuelve solo rutas relativas y contenido textual. El proyecto debe usar exclusivamente las herramientas del entorno homologado indicado; no propongas tecnologías que ese entorno no incluya. Nunca incluyas imágenes, Dockerfiles, instrucciones de instalación, comandos ejecutables por SOLV, runtime, capacidades, variables de infraestructura ni configuración de servicios. El código generado es contenido de proyecto no confiable y el docente debe revisarlo.`
	userPrompt := fmt.Sprintf("Ejercicio autorizado: %s\nDescripción del proyecto: %s\nPropósito del proyecto: %s\nEntorno homologado (stack obligatorio): %s\nRequisitos de archivos: %s",
		strings.TrimSpace(exercise.Title), strings.TrimSpace(req.Description), optionalPromptField(req.Purpose), workspaceEnvironmentPrompt(req), optionalPromptField(req.Requirements))

	raw, _, err := s.aiSettings.CompleteJSON(ctx, tenantID, userID, "generate-workspace-project", systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || len(raw) > maxWorkspaceGenerationOutput || !utf8.ValidString(raw) {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "El proveedor devolvió una propuesta vacía, demasiado grande o con texto inválido."}
	}
	proposalJSON := cleanJSONFence(raw)
	if proposalJSON == "" || len(proposalJSON) > maxWorkspaceGenerationJSON {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "El proveedor devolvió una propuesta JSON vacía o demasiado grande."}
	}
	decoder := json.NewDecoder(strings.NewReader(proposalJSON))
	decoder.DisallowUnknownFields()
	var envelope workspaceProjectGenerationEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "El proveedor devolvió JSON inválido o campos no permitidos."}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "El proveedor devolvió contenido adicional fuera del objeto JSON."}
	}
	if len(envelope.Files) == 0 {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "El proveedor no devolvió archivos para revisar."}
	}
	if strings.TrimSpace(envelope.Summary) == "" || utf8.RuneCountInString(envelope.Summary) > 500 {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "La propuesta no incluye un resumen breve válido."}
	}
	if len(envelope.Files) > maxWorkspaceProjectFiles {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "La propuesta supera el máximo de 300 archivos."}
	}

	rawFiles := make([]WorkspaceProjectImportFile, 0, len(envelope.Files))
	for _, file := range envelope.Files {
		rawFiles = append(rawFiles, WorkspaceProjectImportFile{Path: file.Path, Content: []byte(file.Content)})
	}
	files, totalBytes, err := normalizeWorkspaceProjectFiles(rawFiles)
	if err != nil {
		// La propuesta se valida con las mismas reglas que un ZIP o una carpeta:
		// una ruta insegura o un binario la descartan completa en vez de entrar
		// al editor como si fuera contenido confiable.
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "La propuesta no cumple las reglas de proyecto: " + err.Error()}
	}
	if len(files) == 0 || totalBytes == 0 {
		return nil, &AISettingsValidationError{Code: "ai_response_invalid", Message: "La propuesta no contiene un proyecto textual válido."}
	}
	proposalFiles := make([]WorkspaceProjectProposalFile, 0, len(files))
	for _, file := range files {
		proposalFiles = append(proposalFiles, WorkspaceProjectProposalFile{Path: file.Path, Content: file.Content})
	}
	return &GenerateWorkspaceProjectProposal{
		Files:      proposalFiles,
		Summary:    strings.TrimSpace(envelope.Summary),
		FileCount:  len(proposalFiles),
		TotalBytes: totalBytes,
	}, nil
}

func validateWorkspaceGenerationRequest(req GenerateWorkspaceProjectRequest) error {
	fields := []struct {
		name  string
		value string
		max   int
	}{
		{"description", req.Description, 2000},
		{"purpose", req.Purpose, 500},
		{"technology", req.Technology, 300},
		{"requirements", req.Requirements, 1200},
	}
	total := 0
	for _, field := range fields {
		count := utf8.RuneCountInString(field.value)
		if count > field.max {
			return &AISettingsValidationError{Code: "generation_input_too_large", Message: fmt.Sprintf("El campo %s supera el límite de %d caracteres.", field.name, field.max)}
		}
		total += len(field.value)
	}
	if strings.TrimSpace(req.Description) == "" {
		return &AISettingsValidationError{Code: "generation_description_required", Message: "Describe el proyecto que quieres preparar."}
	}
	if total > maxWorkspaceGenerationPrompt {
		return &AISettingsValidationError{Code: "generation_input_too_large", Message: "La descripción total supera el límite permitido."}
	}
	return validateWorkspaceGenerationStack(req)
}

// validateWorkspaceGenerationStack acota los identificadores que describen el
// entorno. El navegador los toma de la plantilla seleccionada, pero el backend
// no confía en su tamaño ni en la cantidad de complementos.
func validateWorkspaceGenerationStack(req GenerateWorkspaceProjectRequest) error {
	fields := []struct {
		name  string
		value string
		max   int
	}{
		{"template_id", req.TemplateID, 100},
	}
	if req.StackConfig != nil {
		fields = append(fields,
			struct {
				name  string
				value string
				max   int
			}{"stack_config.runtime_id", req.StackConfig.RuntimeID, 100},
			struct {
				name  string
				value string
				max   int
			}{"stack_config.web_platform_id", req.StackConfig.WebPlatformID, 100},
			struct {
				name  string
				value string
				max   int
			}{"stack_config.database_id", req.StackConfig.DatabaseID, 100},
		)
		if len(req.StackConfig.AddonIDs) > 20 {
			return &AISettingsValidationError{Code: "generation_stack_too_large", Message: "El stack del entorno supera el máximo de 20 complementos."}
		}
		for _, id := range req.StackConfig.AddonIDs {
			if utf8.RuneCountInString(id) > 100 {
				return &AISettingsValidationError{Code: "generation_stack_too_large", Message: "Un complemento del entorno supera el límite de 100 caracteres."}
			}
		}
	}
	for _, field := range fields {
		if utf8.RuneCountInString(field.value) > field.max {
			return &AISettingsValidationError{Code: "generation_stack_too_large", Message: fmt.Sprintf("El campo %s supera el límite de %d caracteres.", field.name, field.max)}
		}
	}
	return nil
}

// workspaceEnvironmentPrompt describe el entorno homologado con las piezas
// estructuradas que envía el editor. La etiqueta legible queda como respaldo
// para clientes anteriores que solo mandan technology.
func workspaceEnvironmentPrompt(req GenerateWorkspaceProjectRequest) string {
	parts := make([]string, 0, 6)
	if value := strings.TrimSpace(req.TemplateID); value != "" {
		parts = append(parts, "plantilla="+value)
	}
	if config := req.StackConfig; config != nil {
		if value := strings.TrimSpace(config.RuntimeID); value != "" {
			parts = append(parts, "runtime="+value)
		}
		if value := strings.TrimSpace(config.WebPlatformID); value != "" {
			parts = append(parts, "plataforma_web="+value)
		}
		if len(config.AddonIDs) > 0 {
			parts = append(parts, "complementos="+strings.Join(config.AddonIDs, ","))
		}
		if value := strings.TrimSpace(config.DatabaseID); value != "" {
			parts = append(parts, "base_de_datos="+value)
		}
	}
	if value := strings.TrimSpace(req.Technology); value != "" {
		parts = append(parts, "tecnología="+value)
	}
	if len(parts) == 0 {
		return "No especificado"
	}
	return strings.Join(parts, "; ")
}

func optionalPromptField(value string) string {
	if strings.TrimSpace(value) == "" {
		return "No especificado"
	}
	return strings.TrimSpace(value)
}

type WorkspaceProjectView struct {
	Project  *domain.WorkspaceProject                `json:"project"`
	Draft    *domain.WorkspaceProjectDraft           `json:"draft,omitempty"`
	Versions []*domain.WorkspaceProjectVersion       `json:"versions"`
	Legacy   *domain.WorkspaceLegacyProjectReference `json:"legacy,omitempty"`
	// PublishingEnabled refleja el flag del servidor para que la interfaz no
	// ofrezca una acción que el backend rechazará. No autoriza por sí mismo.
	PublishingEnabled bool `json:"publishing_enabled"`
	// Preparation describe la preparación de dependencias de la versión
	// publicada vigente. Sin trabajo queda ausente: la versión se publicó antes
	// de que la cola existiera.
	Preparation *WorkspacePreparationView `json:"preparation,omitempty"`
}

// SetAccessRepositories engancha los repositorios con los que se comprueba que
// el ejercicio es IDE_PERSISTENTE y pertenece al docente autenticado. Sin ellos
// el servicio conserva el comportamiento de Fase 1 para las pruebas unitarias.
func (s *WorkspaceProjectService) SetAccessRepositories(exercises domain.ExerciseRepository, subjects domain.SubjectRepository) {
	if s == nil {
		return
	}
	s.exercises = exercises
	s.subjects = subjects
}

type SaveWorkspaceDraftRequest struct {
	TextSnapshot     string  `json:"text_snapshot"`
	ExpectedRevision int64   `json:"expected_revision"`
	BaseVersionID    *string `json:"base_version_id,omitempty"`
}

func (s *WorkspaceProjectService) GetProject(ctx context.Context, tenantID, exerciseID string) (*WorkspaceProjectView, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("workspace project repository is not configured")
	}
	project, err := s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
			legacy, legacyErr := s.repo.GetLegacyReference(ctx, tenantID, exerciseID)
			if legacyErr == nil {
				return &WorkspaceProjectView{Legacy: legacy, Versions: []*domain.WorkspaceProjectVersion{}, PublishingEnabled: s.publishingEnabled}, nil
			}
			// Un ejercicio IDE todavía sin proyecto es un estado válido: la
			// interfaz necesita saber si puede publicar y ofrecer la importación.
			return &WorkspaceProjectView{Versions: []*domain.WorkspaceProjectVersion{}, PublishingEnabled: s.publishingEnabled}, nil
		}
		return nil, err
	}
	draft, err := s.repo.GetDraftByProject(ctx, tenantID, project.ID)
	if err != nil && !errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
		return nil, err
	}
	versions, err := s.repo.ListVersions(ctx, tenantID, project.ID)
	if err != nil {
		return nil, err
	}
	view := &WorkspaceProjectView{Project: project, Draft: draft, Versions: versions, PublishingEnabled: s.publishingEnabled}
	if err := s.attachPreparation(ctx, tenantID, view); err != nil {
		return nil, err
	}
	return view, nil
}

// attachPreparation suma el estado de la versión publicada vigente. Solo esa:
// el historial no se prepara y la vista no necesita una consulta por versión.
func (s *WorkspaceProjectService) attachPreparation(
	ctx context.Context,
	tenantID string,
	view *WorkspaceProjectView,
) error {
	if s == nil || s.dependencyQueue == nil || view == nil || view.Project == nil {
		return nil
	}
	if view.Project.CurrentPublishedVersionID == nil {
		return nil
	}
	versionID := strings.TrimSpace(*view.Project.CurrentPublishedVersionID)
	if versionID == "" {
		return nil
	}
	preparation, err := s.dependencyQueue.Preparation(ctx, tenantID, versionID)
	if err != nil {
		return err
	}
	view.Preparation = preparation
	return nil
}

// RetryPreparation vuelve a encolar la preparación de la versión publicada
// vigente. Solo esa: preparar una versión vieja deja un árbol que ningún alumno
// va a recibir. La autorización la hizo el llamador con `AuthorizeExercise`.
func (s *WorkspaceProjectService) RetryPreparation(
	ctx context.Context,
	tenantID, exerciseID string,
) (*WorkspacePreparationView, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("workspace project repository is not configured")
	}
	if s.dependencyQueue == nil {
		return nil, domain.ErrDependencyPreparationDisabled
	}
	project, err := s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
	if err != nil {
		return nil, err
	}
	if project.CurrentPublishedVersionID == nil || strings.TrimSpace(*project.CurrentPublishedVersionID) == "" {
		return nil, fmt.Errorf("%w: el proyecto todavía no tiene una versión publicada",
			domain.ErrDependencyPreparationNotRetryable)
	}
	versionID := *project.CurrentPublishedVersionID
	if _, err := s.dependencyQueue.Retry(ctx, tenantID, exerciseID, versionID); err != nil {
		return nil, err
	}
	return s.dependencyQueue.Preparation(ctx, tenantID, versionID)
}

// AuthorizeExercise resuelve desde el contexto autenticado la propiedad del
// ejercicio. Un docente solo opera sobre materias que tiene asignadas; un
// administrador puede operar sobre el tenant, pero nunca fuera de él.
func (s *WorkspaceProjectService) AuthorizeExercise(ctx context.Context, tenantID, userID, role, exerciseID string) error {
	if s == nil || s.exercises == nil {
		return nil
	}
	if role != "teacher" && role != "admin" {
		return domain.ErrWorkspaceProjectForbidden
	}
	exercise, err := s.exercises.GetByIDAndTenant(ctx, exerciseID, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrNotFound) {
			return domain.ErrWorkspaceProjectNotFound
		}
		return err
	}
	if exercise == nil || exercise.TenantID != tenantID {
		return domain.ErrWorkspaceProjectNotFound
	}
	if exercise.EnvironmentType != string(domain.EnvironmentTypeIDEPersistente) {
		return domain.ErrWorkspaceProjectNotIDEPersistent
	}
	if role == "admin" {
		return nil
	}
	if s.subjects == nil || exercise.SubjectID == nil || strings.TrimSpace(*exercise.SubjectID) == "" {
		return domain.ErrWorkspaceProjectForbidden
	}
	subject, err := s.subjects.GetByID(ctx, tenantID, *exercise.SubjectID)
	if err != nil || subject == nil || subject.TeacherID == nil || *subject.TeacherID != userID {
		return domain.ErrWorkspaceProjectForbidden
	}
	return nil
}

func (s *WorkspaceProjectService) SaveDraft(ctx context.Context, tenantID, exerciseID, userID string, req SaveWorkspaceDraftRequest) (*domain.WorkspaceProjectDraft, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("workspace project repository is not configured")
	}
	if _, err := uuid.Parse(strings.TrimSpace(exerciseID)); err != nil {
		return nil, domain.ErrInvalidResourceID
	}
	if err := validateWorkspaceProjectSnapshot(req.TextSnapshot); err != nil {
		return nil, err
	}

	project, err := s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
	if errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
		project, err = s.repo.CreateProject(ctx, &domain.WorkspaceProject{
			TenantID:   tenantID,
			ExerciseID: exerciseID,
			CreatedBy:  optionalString(userID),
		})
		if errors.Is(err, domain.ErrWorkspaceProjectExists) {
			project, err = s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
		}
	}
	if err != nil {
		return nil, err
	}

	baseVersionID := req.BaseVersionID
	if baseVersionID == nil {
		baseVersionID = project.CurrentPublishedVersionID
	}
	draft := &domain.WorkspaceProjectDraft{
		TenantID:      tenantID,
		ProjectID:     project.ID,
		BaseVersionID: baseVersionID,
		TextSnapshot:  req.TextSnapshot,
		UpdatedBy:     optionalString(userID),
	}
	return s.repo.SaveDraft(ctx, draft, req.ExpectedRevision)
}

func (s *WorkspaceProjectService) Publish(ctx context.Context, tenantID, exerciseID, userID string) (*domain.WorkspaceProjectVersion, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("workspace project repository is not configured")
	}
	if !s.publishingEnabled {
		return nil, domain.ErrWorkspacePublishingDisabled
	}
	project, err := s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
	if err != nil {
		return nil, err
	}
	draft, err := s.repo.GetDraftByProject(ctx, tenantID, project.ID)
	if err != nil {
		return nil, err
	}
	if err := validateWorkspaceProjectSnapshot(draft.TextSnapshot); err != nil {
		return nil, err
	}
	version, err := s.repo.Publish(ctx, tenantID, project.ID, userID)
	if err != nil {
		return nil, err
	}
	if s.dependencyQueue != nil {
		if _, enqueueErr := s.dependencyQueue.Enqueue(ctx, tenantID, exerciseID, version.ID); enqueueErr != nil {
			// La versión ya está publicada y no se deshace por un encolado fallido:
			// el reconciliador de la cola la recupera en el siguiente arranque.
			log.Printf("aviso: la versión %s se publicó sin encolar su preparación: %v", version.ID, enqueueErr)
		}
	}
	return version, nil
}

// HasProject es la comprobación que usa el catálogo de ejercicios para impedir
// que un update lateral reemplace el espejo publicado.
func (s *WorkspaceProjectService) HasProject(ctx context.Context, tenantID, exerciseID string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	_, err := s.repo.GetProjectByExercise(ctx, tenantID, exerciseID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
		return false, nil
	}
	return false, err
}

func validateWorkspaceProjectSnapshot(snapshot string) error {
	if strings.TrimSpace(snapshot) == "" {
		return domain.ErrWorkspaceDraftEmpty
	}
	if len(snapshot) > maxWorkspaceProjectBytes {
		return fmt.Errorf("el proyecto excede el tamaño máximo de %d bytes", maxWorkspaceProjectBytes)
	}
	files, ok := domain.ParseProjectSnapshot(snapshot, "")
	if !ok || len(files) == 0 {
		return fmt.Errorf("el proyecto no contiene un snapshot válido")
	}
	if len(files) > maxWorkspaceProjectFiles {
		return fmt.Errorf("el proyecto excede el máximo de %d archivos", maxWorkspaceProjectFiles)
	}
	for _, file := range files {
		if len(file.Content) > maxWorkspaceProjectFileBytes {
			return fmt.Errorf("el archivo %s excede el tamaño máximo de %d bytes", file.Path, maxWorkspaceProjectFileBytes)
		}
	}
	return nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
