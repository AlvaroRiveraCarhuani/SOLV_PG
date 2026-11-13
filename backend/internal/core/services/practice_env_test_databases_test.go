package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

const practiceTestExerciseID = "aaaaaaaa-1111-4111-8111-111111111111"

func TestPracticeEnvTest_ListStudentDatabases_DevuelveLasBasesDeLaPractica(t *testing.T) {
	svc, _ := newPracticeTestService(t)

	databases, err := svc.ListStudentDatabases(
		context.Background(), "teacher-1", domain.DefaultTenantID, "teacher", "subj-1", practiceTestExerciseID,
	)
	if err != nil {
		t.Fatalf("listado inesperado: %v", err)
	}
	if len(databases) != 1 || databases[0].StudentID != "student-1" {
		t.Fatalf("se esperaba la base de student-1, se obtuvo %+v", databases)
	}
}

func TestPracticeEnvTest_ResetStudentDatabase_ReiniciaUnaBaseExistente(t *testing.T) {
	svc, fixture := newPracticeTestService(t)

	err := svc.ResetStudentDatabase(
		context.Background(), "teacher-1", domain.DefaultTenantID, "teacher", "subj-1", practiceTestExerciseID, "student-1",
	)
	if err != nil {
		t.Fatalf("reinicio inesperado: %v", err)
	}
	if fixture.satellites.resets != 1 {
		t.Fatalf("se esperaba un reinicio, se ejecutaron %d", fixture.satellites.resets)
	}
	if fixture.satellites.resetID != "student-1" {
		t.Fatalf("se reinició %q en lugar de student-1", fixture.satellites.resetID)
	}
}

func TestPracticeEnvTest_ResetStudentDatabase_RechazaAlumnoQueNoAbrioLaPractica(t *testing.T) {
	svc, fixture := newPracticeTestService(t)

	err := svc.ResetStudentDatabase(
		context.Background(), "teacher-1", domain.DefaultTenantID, "teacher", "subj-1", practiceTestExerciseID, "student-9",
	)
	if !errors.Is(err, services.ErrStudentPracticeNotStarted) {
		t.Fatalf("se esperaba ErrStudentPracticeNotStarted, se obtuvo %v", err)
	}
	if fixture.satellites.resets != 0 {
		t.Fatalf("no debía reiniciarse ninguna base, se ejecutaron %d", fixture.satellites.resets)
	}
}

func TestPracticeEnvTest_ResetStudentDatabase_RechazaADocenteAjeno(t *testing.T) {
	svc, _ := newPracticeTestService(t)

	err := svc.ResetStudentDatabase(
		context.Background(), "teacher-2", domain.DefaultTenantID, "teacher", "subj-1", practiceTestExerciseID, "student-1",
	)
	if !errors.Is(err, services.ErrPracticeTestForbidden) {
		t.Fatalf("se esperaba ErrPracticeTestForbidden, se obtuvo %v", err)
	}
}
