package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresNotificationRepository struct {
	db *sqlx.DB
}

func NewPostgresNotificationRepository(db *sqlx.DB) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{db: db}
}

func (r *PostgresNotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (
			id, tenant_id, recipient_user_id, channel, severity, 
			title, message, event_type, metadata, is_read, 
			read_at, email_sent_at, occurrence_count, link, created_at
		) VALUES (
			:id, :tenant_id, :recipient_user_id, :channel, :severity, 
			:title, :message, :event_type, :metadata, :is_read, 
			:read_at, :email_sent_at, :occurrence_count, :link, :created_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, n)
	if err != nil {
		return fmt.Errorf("error creating notification: %w", err)
	}
	return nil
}

func (r *PostgresNotificationRepository) CreateBatch(ctx context.Context, list []*domain.Notification) error {
	if len(list) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(list))
	valueArgs := make([]interface{}, 0, len(list)*15)
	argIdx := 1

	for _, n := range list {
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4,
			argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9,
			argIdx+10, argIdx+11, argIdx+12, argIdx+13, argIdx+14,
		))

		valueArgs = append(valueArgs,
			n.ID, n.TenantID, n.RecipientUserID, n.Channel, n.Severity,
			n.Title, n.Message, n.EventType, n.Metadata, n.IsRead,
			n.ReadAt, n.EmailSentAt, n.OccurrenceCount, n.Link, n.CreatedAt,
		)
		argIdx += 15
	}

	stmt := fmt.Sprintf(`
		INSERT INTO notifications (
			id, tenant_id, recipient_user_id, channel, severity, 
			title, message, event_type, metadata, is_read, 
			read_at, email_sent_at, occurrence_count, link, created_at
		) VALUES %s
	`, strings.Join(valueStrings, ","))

	_, err := r.db.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return fmt.Errorf("error batch inserting notifications: %w", err)
	}
	return nil
}

func (r *PostgresNotificationRepository) ListByRecipient(
	ctx context.Context,
	tenantID, recipientUserID string,
	filter domain.NotificationListFilter,
	limit, offset int,
) ([]*domain.Notification, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	conditions := []string{"tenant_id = $1", "recipient_user_id = $2"}
	args := []interface{}{tenantID, recipientUserID}
	if filter.UnreadOnly {
		conditions = append(conditions, "is_read = false")
	}
	if filter.Severity != "" {
		args = append(args, filter.Severity)
		conditions = append(conditions, fmt.Sprintf("severity = $%d", len(args)))
	}
	if filter.EventType != "" {
		args = append(args, filter.EventType)
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", len(args)))
	}
	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM notifications 
		WHERE %s
	`, whereClause)

	var total int64
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("error counting notifications: %w", err)
	}

	args = append(args, limit, offset)

	selectQuery := fmt.Sprintf(`
		SELECT 
			id, tenant_id, recipient_user_id, channel, severity, 
			title, message, event_type, COALESCE(metadata, '{}'::jsonb) AS metadata, is_read, 
			read_at, email_sent_at, occurrence_count, COALESCE(link, '') AS link, created_at
		FROM notifications
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)-1, len(args))

	var list []*domain.Notification
	if err := r.db.SelectContext(ctx, &list, selectQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("error querying notifications: %w", err)
	}
	if list == nil {
		list = []*domain.Notification{}
	}

	return list, total, nil
}

// IncrementRecentOccurrence aplica la ventana anti-fatiga de ADR-034 sin
// agregar esquema: reutiliza la fila existente, suma la ocurrencia y vuelve a
// marcarla como no leída para que un evento nuevo no quede escondido.
func (r *PostgresNotificationRepository) IncrementRecentOccurrence(
	ctx context.Context,
	tenantID, recipientUserID, eventType string,
	since time.Time,
) (bool, error) {
	query := `
		UPDATE notifications
		SET occurrence_count = occurrence_count + 1,
		    is_read = false,
		    read_at = NULL,
		    created_at = NOW()
		WHERE id = (
			SELECT id FROM notifications
			WHERE tenant_id = $1
			  AND recipient_user_id = $2
			  AND event_type = $3
			  AND created_at >= $4
			ORDER BY created_at DESC
			LIMIT 1
		)
	`
	res, err := r.db.ExecContext(ctx, query, tenantID, recipientUserID, eventType, since)
	if err != nil {
		return false, fmt.Errorf("error consolidating notification: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, nil
	}
	return rows > 0, nil
}

func (r *PostgresNotificationRepository) CountUnread(ctx context.Context, tenantID, recipientUserID string) (int64, error) {
	query := `
		SELECT COUNT(*) 
		FROM notifications 
		WHERE tenant_id = $1 AND recipient_user_id = $2 AND is_read = false
	`
	var count int64
	err := r.db.GetContext(ctx, &count, query, tenantID, recipientUserID)
	if err != nil {
		return 0, fmt.Errorf("error counting unread notifications: %w", err)
	}
	return count, nil
}

func (r *PostgresNotificationRepository) MarkAsRead(ctx context.Context, tenantID, recipientUserID, notificationID string) error {
	query := `
		UPDATE notifications 
		SET is_read = true, read_at = NOW() 
		WHERE tenant_id = $1 AND recipient_user_id = $2 AND id = $3
	`
	res, err := r.db.ExecContext(ctx, query, tenantID, recipientUserID, notificationID)
	if err != nil {
		return fmt.Errorf("error marking notification as read: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PostgresNotificationRepository) MarkAllAsRead(ctx context.Context, tenantID, recipientUserID string) (int64, error) {
	query := `
		UPDATE notifications 
		SET is_read = true, read_at = NOW() 
		WHERE tenant_id = $1 AND recipient_user_id = $2 AND is_read = false
	`
	res, err := r.db.ExecContext(ctx, query, tenantID, recipientUserID)
	if err != nil {
		return 0, fmt.Errorf("error marking all notifications as read: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}
