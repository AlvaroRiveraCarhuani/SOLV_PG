package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type stubPlagiarismPairRepo struct {
	*stubTeacherRepoForKeystroke
	rows []*domain.PlagiarismPairSubmission
	err  error
}

func (r *stubPlagiarismPairRepo) GetPlagiarismPairSubmissions(_ context.Context, _, _, _, _ string) ([]*domain.PlagiarismPairSubmission, error) {
	return r.rows, r.err
}

func pairSubmission(id, student, code string) *domain.PlagiarismPairSubmission {
	return &domain.PlagiarismPairSubmission{
		SubmissionID:    id,
		ExerciseID:      "ex-1",
		ExerciseTitle:   "Pilas y Colas",
		SubjectID:       "subj-1",
		SubjectName:     "SIS-210",
		StudentID:       student,
		StudentName:     student,
		StudentEmail:    student + "@solv.test",
		EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		Language:        "python",
		Code:            code,
		Verdict:         "AC",
		SubmittedAt:     time.Now(),
	}
}

func TestGetPlagiarismPairReturnsRangesAndMetrics(t *testing.T) {
	code := "def solve(arr):\n    total = 0\n    for v in arr:\n        total += v\n    return total\n"
	repo := &stubPlagiarismPairRepo{
		stubTeacherRepoForKeystroke: &stubTeacherRepoForKeystroke{},
		rows: []*domain.PlagiarismPairSubmission{
			pairSubmission("sub-a", "student-1", code),
			pairSubmission("sub-b", "student-2", code),
		},
	}

	keystrokes := newMemoryKeystrokeRepo()
	_ = keystrokes.SaveBatch(context.Background(), "sub-a", []domain.SubmissionKeystrokeEvent{
		{EventType: domain.KeystrokeEventInsert, Content: "abcd", TimestampMS: 60000},
		{EventType: domain.KeystrokeEventPaste, Content: "efghij", TimestampMS: 90000},
	})
	service := NewTeacherService(repo, newStubSubmissionRepo())
	service.SetKeystrokeRepository(keystrokes)

	pair, err := service.GetPlagiarismPair(context.Background(), "tenant-1", "teacher-1", "sub-a", "sub-b")
	if err != nil {
		t.Fatalf("par inesperado: %v", err)
	}

	if pair.Similarity != 100 {
		t.Errorf("esperado 100%% entre entregas idénticas, obtenido %.1f", pair.Similarity)
	}
	if len(pair.Ranges) == 0 {
		t.Fatalf("el par debe traer tramos compartidos")
	}
	if pair.A.SubmissionID != "sub-a" || pair.B.StudentID != "student-2" {
		t.Errorf("lados mal armados: A=%s B=%s", pair.A.SubmissionID, pair.B.SubmissionID)
	}
	if !pair.A.Metrics.EventsAvailable || pair.A.Metrics.PasteEvents != 1 {
		t.Errorf("métricas forenses de A incompletas: %+v", pair.A.Metrics)
	}
	if pair.B.Metrics.EventsAvailable {
		t.Errorf("B no tiene eventos y no debe inventarlos")
	}
	if pair.A.Verdict != "AC" {
		t.Errorf("el veredicto debe viajar con el lado, obtenido %q", pair.A.Verdict)
	}
}

func TestGetPlagiarismPairRejectsDifferentExercise(t *testing.T) {
	other := pairSubmission("sub-b", "student-2", "def solve():\n    return 1\n")
	other.ExerciseID = "ex-2"
	repo := &stubPlagiarismPairRepo{
		stubTeacherRepoForKeystroke: &stubTeacherRepoForKeystroke{},
		rows: []*domain.PlagiarismPairSubmission{
			pairSubmission("sub-a", "student-1", "def solve():\n    return 1\n"),
			other,
		},
	}
	service := NewTeacherService(repo, newStubSubmissionRepo())

	_, err := service.GetPlagiarismPair(context.Background(), "tenant-1", "teacher-1", "sub-a", "sub-b")
	if !errors.Is(err, domain.ErrPlagiarismPairDifferentExercise) {
		t.Fatalf("esperado ErrPlagiarismPairDifferentExercise, obtenido %v", err)
	}
}

func TestGetPlagiarismPairHidesForeignSubmission(t *testing.T) {
	repo := &stubPlagiarismPairRepo{
		stubTeacherRepoForKeystroke: &stubTeacherRepoForKeystroke{},
		rows: []*domain.PlagiarismPairSubmission{
			pairSubmission("sub-a", "student-1", "def solve():\n    return 1\n"),
		},
	}
	service := NewTeacherService(repo, newStubSubmissionRepo())

	_, err := service.GetPlagiarismPair(context.Background(), "tenant-1", "teacher-1", "sub-a", "sub-b")
	if !errors.Is(err, domain.ErrPlagiarismPairForbidden) {
		t.Fatalf("esperado ErrPlagiarismPairForbidden, obtenido %v", err)
	}
}

func TestGetPlagiarismPairSeparatesIDEFiles(t *testing.T) {
	snapshot := "# === FILE: base.py ===\ndef helper(x):\n    return x + 1\n\n# === FILE: main.py ===\ndef solve(arr):\n    return arr[0]\n"
	first := pairSubmission("sub-a", "student-1", snapshot)
	first.EnvironmentType = string(domain.EnvironmentTypeIDEPersistente)
	first.Boilerplate = "# === FILE: base.py ===\ndef helper(x):\n    return x + 1\n"
	second := pairSubmission("sub-b", "student-2", snapshot)
	second.EnvironmentType = string(domain.EnvironmentTypeIDEPersistente)
	second.Boilerplate = first.Boilerplate

	repo := &stubPlagiarismPairRepo{
		stubTeacherRepoForKeystroke: &stubTeacherRepoForKeystroke{},
		rows:                        []*domain.PlagiarismPairSubmission{first, second},
	}
	service := NewTeacherService(repo, newStubSubmissionRepo())

	pair, err := service.GetPlagiarismPair(context.Background(), "tenant-1", "teacher-1", "sub-a", "sub-b")
	if err != nil {
		t.Fatalf("par IDE inesperado: %v", err)
	}
	if !pair.A.FilesAvailable || len(pair.A.Files) != 2 {
		t.Fatalf("el lado IDE debe traer sus archivos, obtenidos %d", len(pair.A.Files))
	}
	if pair.A.Code != "" {
		t.Errorf("en IDE no se duplica el snapshot en code")
	}
	if !pair.Exercise.BaseFilesAvailable {
		t.Errorf("el ejercicio debe traer el proyecto base para descartar archivos")
	}
	if len(pair.Ranges) == 0 || pair.Ranges[0].SourceA.FilePath != "main.py" {
		t.Errorf("los tramos deben ubicarse en main.py: %+v", pair.Ranges)
	}
}
