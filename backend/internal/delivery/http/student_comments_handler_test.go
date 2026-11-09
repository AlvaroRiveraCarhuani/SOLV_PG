package httpdelivery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	httpdelivery "solv-backend/internal/delivery/http"
)

type commentsSubmissionRepo struct {
	*mockSubmissionRepoForStudent
	byID map[string]*domain.Submission
}

func (r *commentsSubmissionRepo) GetByID(_ context.Context, _, id string) (*domain.Submission, error) {
	return r.byID[id], nil
}

type commentsRepo struct {
	comments []*domain.SubmissionComment
}

func (r *commentsRepo) GetCommentsBySubmission(_ context.Context, _, _ string) ([]*domain.SubmissionComment, error) {
	return r.comments, nil
}

func commentsRequest(submissionID, userID string) *http.Request {
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/v1/student/submissions/"+submissionID+"/comments",
		nil,
	)
	req.SetPathValue("id", submissionID)
	return withTestIdentity(req, userID, "tenant-1", "student")
}

func TestStudentGetsCommentsOfOwnSubmission(t *testing.T) {
	path, local := "src/tree.py", 12
	repo := &commentsSubmissionRepo{
		mockSubmissionRepoForStudent: &mockSubmissionRepoForStudent{},
		byID: map[string]*domain.Submission{
			"sub-1": {ID: "sub-1", StudentID: "student-1"},
		},
	}
	reader := &commentsRepo{comments: []*domain.SubmissionComment{
		{ID: "c-1", SubmissionID: "sub-1", AuthorName: "Docente", FilePath: &path, LocalLineNumber: &local, Comment: "Revisar el caso base"},
	}}
	handler := httpdelivery.NewStudentHandler(nil, nil, repo, nil, reader)

	w := httptest.NewRecorder()
	handler.GetSubmissionComments(w, commentsRequest("sub-1", "student-1"))

	if w.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d. Body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []domain.SubmissionComment `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("respuesta no es JSON: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].LocalLineNumber == nil || *resp.Data[0].LocalLineNumber != 12 {
		t.Fatalf("comentario sin ancla local esperada: %+v", resp.Data)
	}
}

func TestStudentCannotReadCommentsOfAnotherSubmission(t *testing.T) {
	repo := &commentsSubmissionRepo{
		mockSubmissionRepoForStudent: &mockSubmissionRepoForStudent{},
		byID: map[string]*domain.Submission{
			"sub-1": {ID: "sub-1", StudentID: "student-2"},
		},
	}
	handler := httpdelivery.NewStudentHandler(nil, nil, repo, nil, &commentsRepo{})

	w := httptest.NewRecorder()
	handler.GetSubmissionComments(w, commentsRequest("sub-1", "student-1"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("esperado 404 para una entrega ajena, obtenido %d", w.Code)
	}
}

func TestStudentCommentsRequireIdentity(t *testing.T) {
	handler := httpdelivery.NewStudentHandler(nil, nil, nil, nil, &commentsRepo{})

	w := httptest.NewRecorder()
	handler.GetSubmissionComments(w, commentsRequest("sub-1", ""))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 sin identidad, obtenido %d", w.Code)
	}
}
