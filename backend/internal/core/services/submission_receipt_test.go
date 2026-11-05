package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func receiptFor(t *testing.T, tenantID, exerciseID, studentID, verdict string, issuedAt time.Time) string {
	t.Helper()
	return receiptForCode(t, tenantID, exerciseID, studentID, verdict, "print(1)", issuedAt)
}

func receiptForCode(t *testing.T, tenantID, exerciseID, studentID, verdict, code string, issuedAt time.Time) string {
	t.Helper()
	token, err := services.SignEvaluationReceipt(services.EvaluationReceipt{
		TenantID:        tenantID,
		ExerciseID:      exerciseID,
		StudentID:       studentID,
		Verdict:         verdict,
		ExecutionTimeMS: 12,
		MemoryUsedMB:    24,
		CodeHash:        services.HashEvaluatedCode(code),
		IssuedAt:        issuedAt.Unix(),
	})
	if err != nil {
		t.Fatalf("no se pudo firmar el recibo: %v", err)
	}
	return token
}

// Un proyecto de IDE no recibe veredictos por el endpoint genérico de
// entregas, aunque el navegador envíe uno y traiga un recibo válido de otro
// flujo.
func TestCreateSubmission_RejectsIDEExercise(t *testing.T) {
	repo := &fakeSubmissionRepo{}
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-ide": {
			ID:              "ex-ide",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
	}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(exerciseRepo)

	_, err := svc.CreateSubmission(context.Background(), "tenant-1", services.CreateSubmissionDTO{
		ExerciseID:        "ex-ide",
		StudentID:         "student-1",
		Verdict:           "AC",
		EvaluationReceipt: receiptFor(t, "tenant-1", "ex-ide", "student-1", "AC", time.Now()),
	})
	if !errors.Is(err, domain.ErrEvaluationRequiresJudge) {
		t.Fatalf("esperado ErrEvaluationRequiresJudge, obtenido %v", err)
	}
	if repo.created != nil {
		t.Fatalf("una entrega de IDE no debe persistirse por esta vía: %+v", repo.created)
	}
}

// Un veredicto declarado por el navegador, sin recibo del servidor, no se
// convierte en resultado verificado ni se guarda.
func TestCreateSubmission_RequiresServerReceipt(t *testing.T) {
	repo := &fakeSubmissionRepo{}
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-judge": {
			ID:              "ex-judge",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
	}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(exerciseRepo)

	for name, receipt := range map[string]string{
		"sin recibo":       "",
		"recibo inventado": "v1.eyJ2ZXJkaWN0IjoiQUMifQ.firma",
		"recibo vencido":   receiptFor(t, "tenant-1", "ex-judge", "student-1", "AC", time.Now().Add(-time.Hour)),
	} {
		t.Run(name, func(t *testing.T) {
			repo.created = nil
			_, err := svc.CreateSubmission(context.Background(), "tenant-1", services.CreateSubmissionDTO{
				ExerciseID:        "ex-judge",
				StudentID:         "student-1",
				Code:              "print(1)",
				Verdict:           "AC",
				EvaluationReceipt: receipt,
			})
			if !errors.Is(err, domain.ErrInvalidEvaluationReceipt) {
				t.Fatalf("esperado ErrInvalidEvaluationReceipt, obtenido %v", err)
			}
			if repo.created != nil {
				t.Fatalf("un veredicto sin recibo válido no debe persistirse: %+v", repo.created)
			}
		})
	}
}

// El recibo manda sobre el payload: el veredicto y las métricas guardados son
// los que emitió el servidor.
func TestCreateSubmission_UsesReceiptVerdict(t *testing.T) {
	repo := &fakeSubmissionRepo{}
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-judge": {
			ID:              "ex-judge",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
	}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(exerciseRepo)

	receipt := receiptFor(t, "tenant-1", "ex-judge", "student-1", "AC", time.Now())
	sub, err := svc.CreateSubmission(context.Background(), "tenant-1", services.CreateSubmissionDTO{
		ExerciseID:        "ex-judge",
		StudentID:         "student-1",
		Code:              "print(1)",
		Verdict:           "WA",
		ExecutionTimeMS:   999,
		MemoryUsedMB:      999,
		EvaluationReceipt: receipt,
	})
	if err != nil {
		t.Fatalf("la entrega válida del juez falló: %v", err)
	}
	if sub.Verdict != "AC" {
		t.Errorf("el veredicto debe salir del recibo: esperado AC, obtenido %q", sub.Verdict)
	}
	if sub.VerdictOrigin != domain.VerdictOriginJudge {
		t.Errorf("origen esperado judge, obtenido %q", sub.VerdictOrigin)
	}
	if sub.ExecutionTimeMS != 12 || sub.MemoryUsedMB != 24 {
		t.Errorf("las métricas deben salir del recibo, obtenidas %d ms / %d MB", sub.ExecutionTimeMS, sub.MemoryUsedMB)
	}
	if repo.created == nil || repo.created.ID != sub.ID {
		t.Fatal("la entrega válida debe persistirse")
	}
}

// El recibo está atado a su ejercicio y a su alumno: no se reutiliza en otra
// entrega ni en nombre de otra persona.
func TestCreateSubmission_ReceiptIsBoundToExerciseAndStudent(t *testing.T) {
	repo := &fakeSubmissionRepo{}
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-judge": {
			ID:              "ex-judge",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
	}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(exerciseRepo)

	cases := map[string]services.CreateSubmissionDTO{
		"otro ejercicio": {
			ExerciseID:        "ex-judge",
			StudentID:         "student-1",
			Code:              "print(1)",
			EvaluationReceipt: receiptFor(t, "tenant-1", "ex-otro", "student-1", "AC", time.Now()),
		},
		"otro alumno": {
			ExerciseID:        "ex-judge",
			StudentID:         "student-1",
			Code:              "print(1)",
			EvaluationReceipt: receiptFor(t, "tenant-1", "ex-judge", "student-2", "AC", time.Now()),
		},
		"otra institución": {
			ExerciseID:        "ex-judge",
			StudentID:         "student-1",
			Code:              "print(1)",
			EvaluationReceipt: receiptFor(t, "tenant-2", "ex-judge", "student-1", "AC", time.Now()),
		},
		"otro código": {
			ExerciseID:        "ex-judge",
			StudentID:         "student-1",
			Code:              "print(123456)",
			EvaluationReceipt: receiptForCode(t, "tenant-1", "ex-judge", "student-1", "AC", "print(1)", time.Now()),
		},
	}
	for name, dto := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateSubmission(context.Background(), "tenant-1", dto)
			if !errors.Is(err, domain.ErrInvalidEvaluationReceipt) {
				t.Fatalf("esperado ErrInvalidEvaluationReceipt, obtenido %v", err)
			}
		})
	}
	if repo.created != nil {
		t.Fatal("ningún recibo ajeno debe persistir una entrega")
	}
}

// Un recibo registra una sola entrega: reenviarlo devuelve la entrega creada
// (reintento legítimo) en lugar de duplicarla.
func TestCreateSubmission_ReceiptReplayReturnsSameSubmission(t *testing.T) {
	repo := &fakeSubmissionRepo{stored: map[string]*domain.Submission{}}
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-judge": {
			ID:              "ex-judge",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
	}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(exerciseRepo)

	receipt := receiptFor(t, "tenant-1", "ex-judge", "student-1", "AC", time.Now())
	dto := services.CreateSubmissionDTO{
		ExerciseID:        "ex-judge",
		StudentID:         "student-1",
		Code:              "print(1)",
		Verdict:           "AC",
		EvaluationReceipt: receipt,
	}

	first, err := svc.CreateSubmission(context.Background(), "tenant-1", dto)
	if err != nil {
		t.Fatalf("la primera entrega debe registrarse: %v", err)
	}
	second, err := svc.CreateSubmission(context.Background(), "tenant-1", dto)
	if err != nil {
		t.Fatalf("el reenvío del mismo recibo no debe fallar: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("el reenvío debe devolver la misma entrega: %s != %s", second.ID, first.ID)
	}
	if len(repo.stored) != 1 {
		t.Fatalf("un recibo no debe producir más de una entrega, hay %d", len(repo.stored))
	}
	if repo.creates != 2 {
		t.Fatalf("ambos envíos deben intentar registrar, intentos: %d", repo.creates)
	}
}

// Un recibo distinto (otra evaluación del mismo alumno y ejercicio) sí produce
// una entrega nueva.
func TestCreateSubmission_NewReceiptCreatesAnotherSubmission(t *testing.T) {
	repo := &fakeSubmissionRepo{stored: map[string]*domain.Submission{}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(&mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-judge": {ID: "ex-judge", EnvironmentType: string(domain.EnvironmentTypeJuezEfimero)},
	}})

	now := time.Now()
	first, err := svc.CreateSubmission(context.Background(), "tenant-1", services.CreateSubmissionDTO{
		ExerciseID:        "ex-judge",
		StudentID:         "student-1",
		Code:              "print(1)",
		EvaluationReceipt: receiptForCode(t, "tenant-1", "ex-judge", "student-1", "AC", "print(1)", now),
	})
	if err != nil {
		t.Fatalf("la primera entrega debe registrarse: %v", err)
	}
	second, err := svc.CreateSubmission(context.Background(), "tenant-1", services.CreateSubmissionDTO{
		ExerciseID:        "ex-judge",
		StudentID:         "student-1",
		Code:              "print(2)",
		EvaluationReceipt: receiptForCode(t, "tenant-1", "ex-judge", "student-1", "WA", "print(2)", now.Add(time.Minute)),
	})
	if err != nil {
		t.Fatalf("una evaluación nueva debe registrar otra entrega: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("dos evaluaciones distintas no pueden compartir entrega")
	}
	if len(repo.stored) != 2 {
		t.Fatalf("esperadas 2 entregas, hay %d", len(repo.stored))
	}
}

// Dos evaluaciones idénticas en el mismo segundo no comparten identidad: el
// recibo lleva un jti propio y cada una registra su entrega.
func TestSignEvaluationReceipt_EachEvaluationHasOwnIdentity(t *testing.T) {
	issuedAt := time.Now()
	build := func() string {
		t.Helper()
		token, err := services.SignEvaluationReceipt(services.EvaluationReceipt{
			TenantID:        "tenant-1",
			ExerciseID:      "ex-judge",
			StudentID:       "student-1",
			Verdict:         "AC",
			ExecutionTimeMS: 12,
			MemoryUsedMB:    24,
			CodeHash:        services.HashEvaluatedCode("print(1)"),
			IssuedAt:        issuedAt.Unix(),
		})
		if err != nil {
			t.Fatalf("no se pudo firmar el recibo: %v", err)
		}
		return token
	}

	firstToken := build()
	secondToken := build()
	if firstToken == secondToken {
		t.Fatal("dos emisiones en el mismo segundo no pueden producir el mismo token")
	}

	first, err := services.ParseEvaluationReceipt(firstToken, issuedAt)
	if err != nil {
		t.Fatalf("el primer recibo debe ser válido: %v", err)
	}
	second, err := services.ParseEvaluationReceipt(secondToken, issuedAt)
	if err != nil {
		t.Fatalf("el segundo recibo debe ser válido: %v", err)
	}
	if first.JTI == "" || second.JTI == "" {
		t.Fatal("cada recibo debe llevar su identificador")
	}
	if first.JTI == second.JTI {
		t.Fatal("dos evaluaciones distintas no pueden compartir identificador")
	}

	repo := &fakeSubmissionRepo{stored: map[string]*domain.Submission{}}
	svc := services.NewSubmissionService(repo)
	svc.SetExerciseRepository(&mockExerciseRepo{exercises: map[string]*domain.Exercise{
		"ex-judge": {ID: "ex-judge", EnvironmentType: string(domain.EnvironmentTypeJuezEfimero)},
	}})
	base := services.CreateSubmissionDTO{
		ExerciseID: "ex-judge",
		StudentID:  "student-1",
		Code:       "print(1)",
		Verdict:    "AC",
	}
	base.EvaluationReceipt = firstToken
	firstSub, err := svc.CreateSubmission(context.Background(), "tenant-1", base)
	if err != nil {
		t.Fatalf("la primera entrega debe registrarse: %v", err)
	}
	base.EvaluationReceipt = secondToken
	secondSub, err := svc.CreateSubmission(context.Background(), "tenant-1", base)
	if err != nil {
		t.Fatalf("la segunda evaluación debe registrar su entrega: %v", err)
	}
	if firstSub.ID == secondSub.ID {
		t.Fatal("dos evaluaciones del mismo segundo no pueden compartir entrega")
	}
}
