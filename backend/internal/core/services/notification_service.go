package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
)

type NotificationService struct {
	repo         domain.NotificationRepository
	dispatchChan chan *domain.Notification
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

func NewNotificationService(repo domain.NotificationRepository, bufferSize int) *NotificationService {
	if bufferSize <= 0 {
		bufferSize = 256
	}

	s := &NotificationService{
		repo:         repo,
		dispatchChan: make(chan *domain.Notification, bufferSize),
		stopChan:     make(chan struct{}),
	}

	// Iniciar worker en background para despacho asíncrono
	s.wg.Add(1)
	go s.worker()

	return s
}

func (s *NotificationService) worker() {
	defer s.wg.Done()
	for {
		select {
		case n := <-s.dispatchChan:
			if err := s.repo.Create(context.Background(), n); err != nil {
				log.Printf("[NotificationWorker] Error dispatching async notification: %v", err)
			}
		case <-s.stopChan:
			// Drenar canal antes de cerrar
			for len(s.dispatchChan) > 0 {
				n := <-s.dispatchChan
				_ = s.repo.Create(context.Background(), n)
			}
			return
		}
	}
}

func (s *NotificationService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// Notify emite una notificación de forma síncrona o asíncrona
func (s *NotificationService) Notify(ctx context.Context, tenantID string, dto domain.CreateNotificationDTO) (*domain.Notification, error) {
	if dto.RecipientUserID == "" {
		return nil, fmt.Errorf("recipient_user_id is required")
	}
	if dto.Title == "" || dto.Message == "" {
		return nil, fmt.Errorf("title and message are required")
	}

	severity := dto.Severity
	if severity == "" {
		severity = domain.NotificationSeverityInfo
	}

	channel := dto.Channel
	if channel == "" {
		channel = domain.NotificationChannelInApp
	}

	n := &domain.Notification{
		ID:              uuid.NewString(),
		TenantID:        tenantID,
		RecipientUserID: dto.RecipientUserID,
		Channel:         channel,
		Severity:        severity,
		Title:           dto.Title,
		Message:         dto.Message,
		EventType:       dto.EventType,
		Metadata:        dto.Metadata,
		IsRead:          false,
		OccurrenceCount: 1,
		Link:            dto.Link,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	return n, nil
}

// NotifyAsync encola la notificación en el worker asíncrono sin bloquear la petición
func (s *NotificationService) NotifyAsync(tenantID string, dto domain.CreateNotificationDTO) {
	severity := dto.Severity
	if severity == "" {
		severity = domain.NotificationSeverityInfo
	}

	channel := dto.Channel
	if channel == "" {
		channel = domain.NotificationChannelInApp
	}

	n := &domain.Notification{
		ID:              uuid.NewString(),
		TenantID:        tenantID,
		RecipientUserID: dto.RecipientUserID,
		Channel:         channel,
		Severity:        severity,
		Title:           dto.Title,
		Message:         dto.Message,
		EventType:       dto.EventType,
		Metadata:        dto.Metadata,
		IsRead:          false,
		OccurrenceCount: 1,
		Link:            dto.Link,
		CreatedAt:       time.Now().UTC(),
	}

	select {
	case s.dispatchChan <- n:
	default:
		// Fallback si la cola está llena: ejecutar en goroutine efímera
		go func() {
			_ = s.repo.Create(context.Background(), n)
		}()
	}
}

// NotifyBatch emite múltiples notificaciones en una sola transacción eficiente
func (s *NotificationService) NotifyBatch(ctx context.Context, tenantID string, dtos []domain.CreateNotificationDTO) ([]*domain.Notification, error) {
	if len(dtos) == 0 {
		return []*domain.Notification{}, nil
	}

	list := make([]*domain.Notification, len(dtos))
	now := time.Now().UTC()

	for i, dto := range dtos {
		severity := dto.Severity
		if severity == "" {
			severity = domain.NotificationSeverityInfo
		}

		channel := dto.Channel
		if channel == "" {
			channel = domain.NotificationChannelInApp
		}

		list[i] = &domain.Notification{
			ID:              uuid.NewString(),
			TenantID:        tenantID,
			RecipientUserID: dto.RecipientUserID,
			Channel:         channel,
			Severity:        severity,
			Title:           dto.Title,
			Message:         dto.Message,
			EventType:       dto.EventType,
			Metadata:        dto.Metadata,
			IsRead:          false,
			OccurrenceCount: 1,
			Link:            dto.Link,
			CreatedAt:       now,
		}
	}

	if err := s.repo.CreateBatch(ctx, list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *NotificationService) List(
	ctx context.Context,
	tenantID, recipientUserID string,
	unreadOnly bool,
	page, limit int,
) ([]*domain.Notification, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	return s.repo.ListByRecipient(ctx, tenantID, recipientUserID, unreadOnly, limit, offset)
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, tenantID, recipientUserID string) (*domain.UnreadCountResponse, error) {
	count, err := s.repo.CountUnread(ctx, tenantID, recipientUserID)
	if err != nil {
		return nil, err
	}
	return &domain.UnreadCountResponse{UnreadCount: count}, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, tenantID, recipientUserID, notificationID string) error {
	return s.repo.MarkAsRead(ctx, tenantID, recipientUserID, notificationID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, tenantID, recipientUserID string) (*domain.MarkAllReadResult, error) {
	count, err := s.repo.MarkAllAsRead(ctx, tenantID, recipientUserID)
	if err != nil {
		return nil, err
	}
	return &domain.MarkAllReadResult{MarkedCount: count}, nil
}
