package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

type SubmissionService struct {
	repo          domain.SubmissionRepository
	keystrokeRepo domain.KeystrokeRepository
	snapshotRepo  domain.GeneratedCaseSnapshotRepository
	exerciseRepo  domain.ExerciseRepository
}

func NewSubmissionService(repo domain.SubmissionRepository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

// SetGeneratedCaseSnapshotRepository conecta el snapshot autoritativo de los
// casos generados por semilla. Permite que la entrega se persista con la
// salida esperada real sin que el rol student la reciba nunca.
func (s *SubmissionService) SetGeneratedCaseSnapshotRepository(repo domain.GeneratedCaseSnapshotRepository) {
	s.snapshotRepo = repo
}

func (s *SubmissionService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

// SetExerciseRepository conecta el catálogo de ejercicios. La entrega del juez
// necesita saber la modalidad antes de aceptar un veredicto.
func (s *SubmissionService) SetExerciseRepository(repo domain.ExerciseRepository) {
	s.exerciseRepo = repo
}

func (s *SubmissionService) SaveKeystrokeEvents(ctx context.Context, tenantID, submissionID string, events []domain.SubmissionKeystrokeEvent) (int, error) {
	if submissionID == "" {
		return 0, errors.New("submission_id is required")
	}
	if len(events) == 0 {
		return 0, nil
	}
	if len(events) > 10000 {
		return 0, domain.ErrTooManyKeystrokeEvents
	}
	if s.keystrokeRepo == nil {
		return len(events), nil
	}
	if _, err := s.repo.GetByID(ctx, tenantID, submissionID); err != nil {
		// El repositorio ya distingue "no existe" de "id mal formado"; el
		// transporte necesita ese error intacto para responder 404 o 400.
		return 0, err
	}
	if err := s.keystrokeRepo.SaveBatch(ctx, submissionID, events); err != nil {
		return 0, fmt.Errorf("failed to save keystroke events: %w", err)
	}
	return len(events), nil
}

type CreateSubmissionDTO struct {
	ExerciseID    string  `json:"exercise_id"`
	StudentID     string  `json:"student_id"`
	EnvironmentID *string `json:"environment_id,omitempty"`
	Code          string  `json:"code"`
	// Verdict se conserva para clientes anteriores, pero el servidor no lo usa:
	// el veredicto sale del recibo firmado por la evaluación.
	Verdict string `json:"verdict"`
	// EvaluationReceipt viaja en la respuesta de POST /api/v1/evaluations.
	EvaluationReceipt  string          `json:"evaluation_receipt,omitempty"`
	ASTResult          json.RawMessage `json:"ast_result"`
	GeneratedCases     json.RawMessage `json:"generated_cases,omitempty"`
	ExecutionTimeMS    int             `json:"execution_time_ms"`
	MemoryUsedMB       int             `json:"memory_used_mb"`
	ComplexityAnalysis json.RawMessage `json:"complexity_analysis,omitempty"`
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, tenantID string, dto CreateSubmissionDTO) (*domain.Submission, error) {
	if dto.ExerciseID == "" || dto.StudentID == "" {
		return nil, errors.New("exercise_id and student_id are required")
	}
	if s.exerciseRepo == nil {
		return nil, errors.New("submission service has no exercise repository")
	}

	// La modalidad manda antes que el payload: un proyecto de IDE no recibe
	// veredictos por esta vía, ni siquiera si el navegador los inventa.
	exercise, err := s.exerciseRepo.GetByIDAndTenant(ctx, dto.ExerciseID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load exercise: %w", err)
	}
	if exercise == nil {
		return nil, domain.ErrNotFound
	}
	if exercise.EnvironmentType == string(domain.EnvironmentTypeIDEPersistente) {
		return nil, domain.ErrEvaluationRequiresJudge
	}

	// El veredicto y las métricas salen del recibo firmado al evaluar. El
	// payload del navegador no puede alterarlos.
	receipt, err := ParseEvaluationReceipt(dto.EvaluationReceipt, time.Now())
	if err != nil {
		return nil, err
	}
	if receipt.ExerciseID != dto.ExerciseID || receipt.StudentID != dto.StudentID {
		return nil, domain.ErrInvalidEvaluationReceipt
	}
	if receipt.TenantID != "" && tenantID != "" && receipt.TenantID != tenantID {
		return nil, domain.ErrInvalidEvaluationReceipt
	}
	if receipt.CodeHash != HashEvaluatedCode(dto.Code) {
		return nil, domain.ErrInvalidEvaluationReceipt
	}

	if len(dto.ASTResult) == 0 {
		dto.ASTResult = json.RawMessage("{}")
	}
	// La entrega se identifica por la evaluación que la respalda: reenviar el
	// mismo recibo devuelve la entrega ya registrada, no crea otra.
	subID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(dto.EvaluationReceipt)).String()
	sub := &domain.Submission{
		ID:            subID,
		TenantID:      tenantID,
		ExerciseID:    dto.ExerciseID,
		StudentID:     dto.StudentID,
		EnvironmentID: dto.EnvironmentID,
		Code:          dto.Code,
		Verdict:       receipt.Verdict,
		// El origen lo respalda la evaluación del servidor que firmó el recibo.
		VerdictOrigin:      domain.VerdictOriginJudge,
		ReviewStatus:       string(domain.ReviewNotRequired),
		ASTResult:          dto.ASTResult,
		GeneratedCases:     dto.GeneratedCases,
		ComplexityAnalysis: dto.ComplexityAnalysis,
		ExecutionTimeMS:    receipt.ExecutionTimeMS,
		MemoryUsedMB:       receipt.MemoryUsedMB,
	}

	// El snapshot del servidor manda sobre el payload del cliente: es la única
	// fuente de verdad de los casos generados por semilla (GOB-SEGURO-04).
	if authoritative := s.authoritativeGeneratedCases(ctx, dto.ExerciseID, dto.StudentID); authoritative != nil {
		payload, err := json.Marshal(authoritative)
		if err == nil {
			sub.GeneratedCases = json.RawMessage(payload)
		}
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		if errors.Is(err, domain.ErrDuplicateSubmission) {
			// Reintento del mismo envío: la respuesta es la entrega que ya
			// existe, para que el cliente no pierda el identificador.
			existing, getErr := s.repo.GetByID(ctx, tenantID, subID)
			if getErr != nil {
				return nil, getErr
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}
	return sub, nil
}

// authoritativeGeneratedCases recupera del servidor los casos generados por
// semilla para el ejercicio y el estudiante indicados. Devuelve nil cuando no
// existe snapshot (el ejercicio no usa semilla o todavía no se evaluó).
func (s *SubmissionService) authoritativeGeneratedCases(ctx context.Context, exerciseID, studentID string) []domain.TestCase {
	if s.snapshotRepo == nil || exerciseID == "" || studentID == "" {
		return nil
	}
	cases, err := s.snapshotRepo.GetGeneratedCaseSnapshot(ctx, exerciseID, studentID)
	if err != nil || len(cases) == 0 {
		return nil
	}
	return cases
}

func (s *SubmissionService) GetSubmissionsForExercise(ctx context.Context, tenantID, exerciseID, userID, userRole string) ([]*domain.Submission, error) {
	if userRole == "student" {
		return s.repo.ListByExerciseAndStudent(ctx, tenantID, exerciseID, userID)
	}
	return s.repo.ListByExercise(ctx, tenantID, exerciseID)
}

func (s *SubmissionService) GetSubmissionByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *SubmissionService) OverrideSubmission(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	if verdict == "" {
		return errors.New("verdict is required")
	}
	if reason == "" {
		return errors.New("override_reason is required")
	}
	return s.repo.UpdateOverride(ctx, tenantID, id, verdict, reason, score, gradedBy)
}
