package services_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// fakeNotificationRepo permite observar creación, consolidación y filtros sin
// base de datos.
type fakeNotificationRepo struct {
	mu            sync.Mutex
	created       []*domain.Notification
	list          []*domain.Notification
	consolidate   bool
	incrementArgs []string
	lastFilter    domain.NotificationListFilter
	lastLimit     int
}

func (f *fakeNotificationRepo) Create(_ context.Context, n *domain.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, n)
	return nil
}

func (f *fakeNotificationRepo) CreateBatch(_ context.Context, list []*domain.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, list...)
	return nil
}

func (f *fakeNotificationRepo) ListByRecipient(
	_ context.Context, _ string, _ string, filter domain.NotificationListFilter, limit, _ int,
) ([]*domain.Notification, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastFilter = filter
	f.lastLimit = limit
	return f.list, int64(len(f.list)), nil
}

func (f *fakeNotificationRepo) CountUnread(context.Context, string, string) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationRepo) MarkAsRead(context.Context, string, string, string) error {
	return nil
}

func (f *fakeNotificationRepo) MarkAllAsRead(context.Context, string, string) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationRepo) IncrementRecentOccurrence(
	_ context.Context, _ string, _ string, eventType string, _ time.Time,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.incrementArgs = append(f.incrementArgs, eventType)
	return f.consolidate, nil
}

func (f *fakeNotificationRepo) snapshot() []*domain.Notification {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*domain.Notification, len(f.created))
	copy(out, f.created)
	return out
}

func TestNotificationService_SanitizesLinks(t *testing.T) {
	cases := []struct {
		name string
		link string
		want string
	}{
		{name: "ruta interna", link: "/student/history/abc", want: "/student/history/abc"},
		{name: "externa", link: "https://evil.example/phish", want: ""},
		{name: "protocolo relativo", link: "//evil.example/x", want: ""},
		{name: "traversal", link: "/student/../admin", want: ""},
		{name: "javascript", link: "javascript:alert(1)", want: ""},
		{name: "vacia", link: "   ", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeNotificationRepo{}
			svc := services.NewNotificationService(repo, 4)
			defer svc.Stop()

			if _, err := svc.Notify(context.Background(), "t-1", domain.CreateNotificationDTO{
				RecipientUserID: "u-1",
				Title:           "Aviso",
				Message:         "Mensaje",
				Link:            tc.link,
			}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			created := repo.snapshot()
			if len(created) != 1 {
				t.Fatalf("esperaba 1 aviso, got %d", len(created))
			}
			if created[0].Link != tc.want {
				t.Errorf("enlace %q: esperaba %q, got %q", tc.link, tc.want, created[0].Link)
			}
		})
	}
}

func TestNotificationService_RejectsInvalidContract(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := services.NewNotificationService(repo, 4)
	defer svc.Stop()

	base := domain.CreateNotificationDTO{RecipientUserID: "u-1", Title: "Aviso", Message: "Mensaje"}

	invalidSeverity := base
	invalidSeverity.Severity = "panic"
	if _, err := svc.Notify(context.Background(), "t-1", invalidSeverity); err == nil {
		t.Error("esperaba rechazo de severidad inválida")
	}

	invalidEvent := base
	invalidEvent.EventType = "evento con espacios"
	if _, err := svc.Notify(context.Background(), "t-1", invalidEvent); err == nil {
		t.Error("esperaba rechazo de event_type inválido")
	}

	longTitle := base
	longTitle.Title = strings.Repeat("a", 129)
	if _, err := svc.Notify(context.Background(), "t-1", longTitle); err == nil {
		t.Error("esperaba rechazo de título demasiado largo")
	}

	missingRecipient := base
	missingRecipient.RecipientUserID = " "
	if _, err := svc.Notify(context.Background(), "t-1", missingRecipient); err == nil {
		t.Error("esperaba rechazo sin destinatario")
	}

	if len(repo.snapshot()) != 0 {
		t.Error("ningún aviso inválido debe persistirse")
	}
}

func TestNotificationService_ConsolidatesOnlyWhenRequested(t *testing.T) {
	repo := &fakeNotificationRepo{consolidate: true}
	svc := services.NewNotificationService(repo, 4)
	defer svc.Stop()

	consolidated, err := svc.Notify(context.Background(), "t-1", domain.CreateNotificationDTO{
		RecipientUserID: "u-1",
		Title:           "Falla de respaldo",
		Message:         "Sin espacio",
		Severity:        domain.NotificationSeverityCritical,
		EventType:       "backup_failed",
		Consolidate:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consolidated != nil {
		t.Error("una ocurrencia consolidada no debe crear un aviso nuevo")
	}
	if len(repo.snapshot()) != 0 {
		t.Error("no debía insertarse un aviso cuando la ocurrencia se agrupó")
	}
	if len(repo.incrementArgs) != 1 || repo.incrementArgs[0] != "backup_failed" {
		t.Errorf("consolidación inesperada: %#v", repo.incrementArgs)
	}

	// Sin Consolidate, dos avisos del mismo evento siguen siendo dos.
	repo.consolidate = true
	for i := 0; i < 2; i++ {
		if _, err := svc.Notify(context.Background(), "t-1", domain.CreateNotificationDTO{
			RecipientUserID: "u-1",
			Title:           "Evaluación publicada",
			Message:         "Nota disponible",
			EventType:       "grade_published",
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if len(repo.snapshot()) != 2 {
		t.Errorf("las calificaciones no deben consolidarse: got %d", len(repo.snapshot()))
	}
}

func TestNotificationService_ListSanitizesLegacyLinksAndCapsLimit(t *testing.T) {
	repo := &fakeNotificationRepo{list: []*domain.Notification{
		{ID: "1", Link: "https://evil.example/x"},
		{ID: "2", Link: "/student/history/ok"},
		{ID: "3", Link: "/a/../b"},
	}}
	svc := services.NewNotificationService(repo, 4)
	defer svc.Stop()

	items, _, err := svc.List(context.Background(), "t-1", "u-1", domain.NotificationListFilter{UnreadOnly: true, Severity: "warning"}, 1, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastFilter.UnreadOnly != true || repo.lastFilter.Severity != "warning" {
		t.Errorf("filtro no propagado: %#v", repo.lastFilter)
	}
	if repo.lastLimit != 100 {
		t.Errorf("el límite debe acotarse a 100, got %d", repo.lastLimit)
	}
	if items[0].Link != "" || items[1].Link != "/student/history/ok" || items[2].Link != "" {
		t.Errorf("enlaces no saneados: %#v", []string{items[0].Link, items[1].Link, items[2].Link})
	}
}

func TestNotificationService_NotifyAsyncPersists(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := services.NewNotificationService(repo, 4)
	defer svc.Stop()

	svc.NotifyAsync("t-1", domain.CreateNotificationDTO{
		RecipientUserID: "u-1",
		Title:           "Aviso asíncrono",
		Message:         "Mensaje",
		EventType:       "grade_published",
		Link:            "/student/history/abc",
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		if len(repo.snapshot()) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("el worker no persistió el aviso")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := repo.snapshot()[0]; got.Link != "/student/history/abc" {
		t.Errorf("enlace inesperado: %q", got.Link)
	}
}
