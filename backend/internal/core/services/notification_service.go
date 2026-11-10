package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
)

// NotificationEmitter publica avisos in-app desde otros servicios sin
// acoplarlos al worker ni a la persistencia concreta.
type NotificationEmitter interface {
	NotifyAsync(tenantID string, dto domain.CreateNotificationDTO)
}

type NotificationService struct {
	repo         domain.NotificationRepository
	dispatchChan chan *domain.Notification
	stopChan     chan struct{}
	wg           sync.WaitGroup
	nowFn        func() time.Time
}

func NewNotificationService(repo domain.NotificationRepository, bufferSize int) *NotificationService {
	if bufferSize <= 0 {
		bufferSize = 256
	}

	s := &NotificationService{
		repo:         repo,
		dispatchChan: make(chan *domain.Notification, bufferSize),
		stopChan:     make(chan struct{}),
		nowFn:        time.Now,
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := s.persist(ctx, n)
			cancel()
			if err != nil {
				log.Printf("[NotificationWorker] Error dispatching async notification: %v", err)
			}
		case <-s.stopChan:
			// Drenar canal antes de cerrar
			for len(s.dispatchChan) > 0 {
				n := <-s.dispatchChan
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = s.persist(ctx, n)
				cancel()
			}
			return
		}
	}
}

func (s *NotificationService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// Notify emite una notificación de forma síncrona. Devuelve nil cuando la
// ocurrencia se consolidó en un aviso existente dentro de la ventana.
func (s *NotificationService) Notify(ctx context.Context, tenantID string, dto domain.CreateNotificationDTO) (*domain.Notification, error) {
	n, err := buildNotification(tenantID, dto, s.nowFn())
	if err != nil {
		return nil, err
	}

	consolidated, err := s.tryConsolidate(ctx, n)
	if err != nil {
		return nil, err
	}
	if consolidated {
		return nil, nil
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	return n, nil
}

// NotifyAsync encola la notificación en el worker asíncrono sin bloquear la petición
func (s *NotificationService) NotifyAsync(tenantID string, dto domain.CreateNotificationDTO) {
	n, err := buildNotification(tenantID, dto, s.nowFn())
	if err != nil {
		log.Printf("[NotificationService] Aviso descartado: %v", err)
		return
	}

	select {
	case s.dispatchChan <- n:
	default:
		// Fallback si la cola está llena: ejecutar en goroutine efímera
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.persist(ctx, n); err != nil {
				log.Printf("[NotificationService] Error persistiendo aviso: %v", err)
			}
		}()
	}
}

// NotifyBatch emite múltiples notificaciones en una sola transacción eficiente
func (s *NotificationService) NotifyBatch(ctx context.Context, tenantID string, dtos []domain.CreateNotificationDTO) ([]*domain.Notification, error) {
	if len(dtos) == 0 {
		return []*domain.Notification{}, nil
	}

	now := s.nowFn().UTC()
	list := make([]*domain.Notification, 0, len(dtos))
	for _, dto := range dtos {
		n, err := buildNotification(tenantID, dto, now)
		if err != nil {
			return nil, err
		}
		list = append(list, n)
	}

	if err := s.repo.CreateBatch(ctx, list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *NotificationService) List(
	ctx context.Context,
	tenantID, recipientUserID string,
	filter domain.NotificationListFilter,
	page, limit int,
) ([]*domain.Notification, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	items, total, err := s.repo.ListByRecipient(ctx, tenantID, recipientUserID, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	// Los avisos históricos pudieron guardarse antes de validar el enlace: se
	// sanea también en la lectura para que el cliente nunca reciba una URL
	// externa ni una ruta con traversal.
	for _, item := range items {
		item.Link = domain.SanitizeNotificationLink(item.Link)
	}
	return items, total, nil
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

// persist aplica la consolidación y, si no corresponde agrupar, inserta el aviso.
func (s *NotificationService) persist(ctx context.Context, n *domain.Notification) error {
	consolidated, err := s.tryConsolidate(ctx, n)
	if err != nil {
		return err
	}
	if consolidated {
		return nil
	}
	return s.repo.Create(ctx, n)
}

// tryConsolidate agrupa eventos de infraestructura repetidos. Solo se aplica a
// los avisos que lo piden explícitamente y que declaran un evento, para no
// fundir dos calificaciones o dos recordatorios distintos.
func (s *NotificationService) tryConsolidate(ctx context.Context, n *domain.Notification) (bool, error) {
	if n == nil || !n.Consolidate || strings.TrimSpace(n.EventType) == "" {
		return false, nil
	}
	since := n.CreatedAt.Add(-domain.NotificationConsolidationWindow)
	return s.repo.IncrementRecentOccurrence(ctx, n.TenantID, n.RecipientUserID, n.EventType, since)
}

func buildNotification(tenantID string, dto domain.CreateNotificationDTO, now time.Time) (*domain.Notification, error) {
	if strings.TrimSpace(dto.RecipientUserID) == "" {
		return nil, fmt.Errorf("recipient_user_id is required")
	}
	if strings.TrimSpace(dto.Title) == "" || strings.TrimSpace(dto.Message) == "" {
		return nil, fmt.Errorf("title and message are required")
	}
	if len(dto.Title) > 128 {
		return nil, fmt.Errorf("title exceeds 128 characters")
	}

	severity := dto.Severity
	if severity == "" {
		severity = domain.NotificationSeverityInfo
	}
	if !domain.IsValidNotificationSeverity(severity) {
		return nil, fmt.Errorf("invalid notification severity %q", severity)
	}

	channel := dto.Channel
	if channel == "" {
		channel = domain.NotificationChannelInApp
	}

	eventType := strings.TrimSpace(dto.EventType)
	if eventType != "" && !domain.IsValidNotificationEventType(eventType) {
		return nil, fmt.Errorf("invalid notification event_type %q", eventType)
	}

	return &domain.Notification{
		ID:              uuid.NewString(),
		TenantID:        tenantID,
		RecipientUserID: dto.RecipientUserID,
		Channel:         channel,
		Severity:        severity,
		Title:           dto.Title,
		Message:         dto.Message,
		EventType:       eventType,
		Metadata:        dto.Metadata,
		IsRead:          false,
		OccurrenceCount: 1,
		Link:            domain.SanitizeNotificationLink(dto.Link),
		CreatedAt:       now.UTC(),
		Consolidate:     dto.Consolidate,
	}, nil
}
