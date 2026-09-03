package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	unreadOnly bool,
	limit, offset int,
) ([]*domain.Notification, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	filterClause := ""
	if unreadOnly {
		filterClause = " AND is_read = false"
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM notifications 
		WHERE tenant_id = $1 AND recipient_user_id = $2 %s
	`, filterClause)

	var total int64
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID, recipientUserID); err != nil {
		return nil, 0, fmt.Errorf("error counting notifications: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT 
			id, tenant_id, recipient_user_id, channel, severity, 
			title, message, event_type, metadata, is_read, 
			read_at, email_sent_at, occurrence_count, link, created_at
		FROM notifications
		WHERE tenant_id = $1 AND recipient_user_id = $2 %s
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, filterClause)

	var list []*domain.Notification
	if err := r.db.SelectContext(ctx, &list, selectQuery, tenantID, recipientUserID, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("error querying notifications: %w", err)
	}
	if list == nil {
		list = []*domain.Notification{}
	}

	return list, total, nil
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
