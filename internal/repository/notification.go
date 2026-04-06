package repository

import (
	"context"
	"database/sql"
	"fmt"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

type (
	NotificationRepository interface {
		Create(
			ctx context.Context,
			userID uuid.UUID,
			notifType dto.NotificationType,
			referenceID uuid.UUID,
			referenceType string,
			actorID uuid.UUID,
			message string,
		) (int64, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.NotificationRow, int, error)
		MarkRead(ctx context.Context, id int, userID uuid.UUID) error
		MarkAllRead(ctx context.Context, userID uuid.UUID) error
		UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
		HasRecentDuplicate(ctx context.Context, userID uuid.UUID, notifType dto.NotificationType, referenceID uuid.UUID, actorID uuid.UUID) (bool, error)
	}

	notificationRepository struct {
		db *sql.DB
	}
)

func (r *notificationRepository) Create(
	ctx context.Context,
	userID uuid.UUID,
	notifType dto.NotificationType,
	referenceID uuid.UUID,
	referenceType string,
	actorID uuid.UUID,
	message string,
) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, referenceID, referenceType, actorID, message,
	)
	if err != nil {
		return 0, fmt.Errorf("insert notification: %w", err)
	}

	return result.LastInsertId()
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.NotificationRow, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = ?`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id, COALESCE(n.message, ''), n.read, n.created_at,
		        u.username, u.display_name, u.avatar_url, COALESCE(ur.role, '')
		 FROM notifications n
		 JOIN users u ON n.actor_id = u.id
		 LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
		 WHERE n.user_id = ?
		 ORDER BY n.created_at DESC
		 LIMIT ? OFFSET ?`, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.NotificationRow
	for rows.Next() {
		var n model.NotificationRow
		var readInt int
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.ReferenceID, &n.ReferenceType, &n.ActorID, &n.Message, &readInt, &n.CreatedAt,
			&n.ActorUsername, &n.ActorDisplayName, &n.ActorAvatarURL, &n.ActorRole,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		n.Read = readInt == 1
		notifications = append(notifications, n)
	}

	return notifications, total, rows.Err()
}

func (r *notificationRepository) MarkRead(ctx context.Context, id int, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = 1 WHERE user_id = ?`, userID,
	)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read = 0`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *notificationRepository) HasRecentDuplicate(
	ctx context.Context,
	userID uuid.UUID,
	notifType dto.NotificationType,
	referenceID uuid.UUID,
	actorID uuid.UUID,
) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications
		 WHERE user_id = ? AND type = ? AND reference_id = ? AND actor_id = ?
		 AND created_at > datetime('now', '-1 hour')`,
		userID, notifType, referenceID, actorID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check duplicate notification: %w", err)
	}
	return count > 0, nil
}
