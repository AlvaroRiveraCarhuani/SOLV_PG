package httpdelivery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type stubPairHTTPRepo struct {
	*mockTeacherAnalyticsRepo
	rows []*domain.PlagiarismPairSubmission
}

func (r *stubPairHTTPRepo) GetPlagiarismPairSubmissions(_ context.Context, _, _, _, _ string) ([]*domain.PlagiarismPairSubmission, error) {
	return r.rows, nil
}

func pairRequest(submissionA, submissionB, role string) *http.Request {
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/v1/teacher/plagiarism/pairs/"+submissionA+"/"+submissionB,
		nil,
	)
	req.SetPathValue("submission_a", submissionA)
	req.SetPathValue("submission_b", submissionB)
	req.Header.Set("X-User-Role", role)
	req.Header.Set("X-User-Id", "teacher-1")
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	return req
}

func TestGetPlagiarismPairHandlerRejectsForeignSubmission(t *testing.T) {
	repo := &stubPairHTTPRepo{
		mockTeacherAnalyticsRepo: &mockTeacherAnalyticsRepo{},
		rows: []*domain.PlagiarismPairSubmission{
			{SubmissionID: "sub-a", ExerciseID: "ex-1", EnvironmentType: "JUEZ_EFIMERO", Language: "python", Code: "def solve():\n    return 1\n"},
		},
	}
	handler := httpdelivery.NewTeacherHandler(services.NewTeacherService(repo, nil), nil)

	w := httptest.NewRecorder()
	handler.GetPlagiarismPair(w, pairRequest("sub-a", "sub-b", "teacher"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("esperado 403 para una entrega ajena, obtenido %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetPlagiarismPairHandlerRejectsDifferentExercise(t *testing.T) {
	repo := &stubPairHTTPRepo{
		mockTeacherAnalyticsRepo: &mockTeacherAnalyticsRepo{},
		rows: []*domain.PlagiarismPairSubmission{
			{SubmissionID: "sub-a", ExerciseID: "ex-1", EnvironmentType: "JUEZ_EFIMERO", Language: "python", Code: "def solve():\n    return 1\n"},
			{SubmissionID: "sub-b", ExerciseID: "ex-2", EnvironmentType: "JUEZ_EFIMERO", Language: "python", Code: "def solve():\n    return 1\n"},
		},
	}
	handler := httpdelivery.NewTeacherHandler(services.NewTeacherService(repo, nil), nil)

	w := httptest.NewRecorder()
	handler.GetPlagiarismPair(w, pairRequest("sub-a", "sub-b", "teacher"))

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422 para prácticas distintas, obtenido %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetPlagiarismPairHandlerReturnsBothSides(t *testing.T) {
	code := "def solve(arr):\n    total = 0\n    for v in arr:\n        total += v\n    return total\n"
	repo := &stubPairHTTPRepo{
		mockTeacherAnalyticsRepo: &mockTeacherAnalyticsRepo{},
		rows: []*domain.PlagiarismPairSubmission{
			{SubmissionID: "sub-a", ExerciseID: "ex-1", ExerciseTitle: "Suma", SubjectID: "subj-1", SubjectName: "SIS-210", StudentID: "std-1", StudentName: "Ana", EnvironmentType: "JUEZ_EFIMERO", Language: "python", Code: code, Verdict: "AC"},
			{SubmissionID: "sub-b", ExerciseID: "ex-1", ExerciseTitle: "Suma", SubjectID: "subj-1", SubjectName: "SIS-210", StudentID: "std-2", StudentName: "Beto", EnvironmentType: "JUEZ_EFIMERO", Language: "python", Code: code, Verdict: "AC"},
		},
	}
	handler := httpdelivery.NewTeacherHandler(services.NewTeacherService(repo, nil), nil)

	w := httptest.NewRecorder()
	handler.GetPlagiarismPair(w, pairRequest("sub-a", "sub-b", "teacher"))

	if w.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data domain.PlagiarismPair `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("respuesta no es JSON: %v", err)
	}
	if resp.Data.A.SubmissionID != "sub-a" || resp.Data.B.SubmissionID != "sub-b" {
		t.Errorf("lados inesperados: A=%s B=%s", resp.Data.A.SubmissionID, resp.Data.B.SubmissionID)
	}
	if len(resp.Data.Ranges) == 0 {
		t.Errorf("el par idéntico debe traer tramos")
	}
}

func TestGetPlagiarismPairHandlerForbidsStudents(t *testing.T) {
	handler := httpdelivery.NewTeacherHandler(services.NewTeacherService(&mockTeacherAnalyticsRepo{}, nil), nil)

	w := httptest.NewRecorder()
	handler.GetPlagiarismPair(w, pairRequest("sub-a", "sub-b", "student"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("esperado 403 para estudiante, obtenido %d", w.Code)
	}
}
