package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// stubMediaNotificationRepo embebe la interfaz para no repetir métodos que el
// test no ejercita.
type stubNotificationRepo struct {
	domain.NotificationRepository
	lastFilter domain.NotificationListFilter
	lastPage   int
	items      []*domain.Notification
}

func (s *stubNotificationRepo) ListByRecipient(
	_ context.Context, _, _ string, filter domain.NotificationListFilter, limit, offset int,
) ([]*domain.Notification, int64, error) {
	s.lastFilter = filter
	s.lastPage = offset/limit + 1
	return s.items, int64(len(s.items)), nil
}

func (s *stubNotificationRepo) MarkAsRead(context.Context, string, string, string) error {
	return nil
}

func newNotificationTestRequest(target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := context.WithValue(req.Context(), domain.TenantIDKey, "tenant-1")
	ctx = context.WithValue(ctx, domain.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func TestNotificationHandler_ListPassesFilters(t *testing.T) {
	repo := &stubNotificationRepo{items: []*domain.Notification{{ID: "n-1", Link: "https://evil.example/x"}}}
	svc := services.NewNotificationService(repo, 4)
	defer svc.Stop()
	handler := NewNotificationHandler(svc)

	rec := httptest.NewRecorder()
	handler.List(rec, newNotificationTestRequest("/api/v1/notifications?severity=warning&event_type=grade_published&unread_only=true&page=2&limit=5"))

	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.Severity != "warning" || repo.lastFilter.EventType != "grade_published" || !repo.lastFilter.UnreadOnly {
		t.Errorf("filtros no propagados: %#v", repo.lastFilter)
	}
	if repo.lastPage != 2 {
		t.Errorf("esperaba página 2, got %d", repo.lastPage)
	}

	var body struct {
		Data []domain.Notification `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta ilegible: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Link != "" {
		t.Errorf("el handler debe sanear enlaces: %#v", body.Data)
	}
	if rec.Header().Get("X-Total-Count") != "1" {
		t.Errorf("X-Total-Count inesperado: %q", rec.Header().Get("X-Total-Count"))
	}
}

func TestNotificationHandler_ListRejectsInvalidFilters(t *testing.T) {
	repo := &stubNotificationRepo{}
	svc := services.NewNotificationService(repo, 4)
	defer svc.Stop()
	handler := NewNotificationHandler(svc)

	cases := map[string]string{
		"/api/v1/notifications?severity=panic":      "invalid_severity",
		"/api/v1/notifications?event_type=no%20way": "invalid_event_type",
	}
	for target, wantCode := range cases {
		rec := httptest.NewRecorder()
		handler.List(rec, newNotificationTestRequest(target))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: esperaba 400, got %d", target, rec.Code)
			continue
		}
		var body struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Error != wantCode {
			t.Errorf("%s: esperaba error %q, got %q", target, wantCode, body.Error)
		}
	}
}

func TestNotificationHandler_MarkReadRejectsInvalidID(t *testing.T) {
	svc := services.NewNotificationService(&stubNotificationRepo{}, 4)
	defer svc.Stop()
	handler := NewNotificationHandler(svc)

	req := newNotificationTestRequest("/api/v1/notifications/not-a-uuid/read")
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.MarkRead(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperaba 400, got %d", rec.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error != "invalid_id" {
		t.Errorf("esperaba invalid_id, got %q", body.Error)
	}
}
