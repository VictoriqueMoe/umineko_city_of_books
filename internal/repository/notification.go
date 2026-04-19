package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
)

const (
	chatRoomMessagePrefix = "sent a message in "
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
		GetByID(ctx context.Context, id int, userID uuid.UUID) (*model.NotificationRow, error)
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
	var actorArg interface{} = actorID
	if actorID == uuid.Nil {
		actorArg = nil
	}
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, referenceID, referenceType, actorArg, message,
	)
	if err != nil {
		return 0, fmt.Errorf("insert notification: %w", err)
	}

	return result.LastInsertId()
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.NotificationRow, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT
		   (SELECT COUNT(DISTINCT reference_id) FROM notifications
		      WHERE user_id = ? AND type = ? AND read = 0) +
		   (SELECT COUNT(*) FROM notifications
		      WHERE user_id = ? AND NOT (type = ? AND read = 0))`,
		userID, dto.NotifChatRoomMessage, userID, dto.NotifChatRoomMessage,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`WITH chat_grouped AS (
		   SELECT
		     id, user_id, type, reference_id, reference_type, actor_id, message, read, created_at,
		     ROW_NUMBER() OVER (PARTITION BY reference_id ORDER BY created_at DESC, id DESC) AS rn,
		     COUNT(*) OVER (PARTITION BY reference_id) AS grp_count
		   FROM notifications
		   WHERE user_id = ? AND type = ? AND read = 0
		 ),
		 combined AS (
		   SELECT id, user_id, type, reference_id, reference_type, actor_id,
		          COALESCE(message, '') AS message, read, created_at, grp_count AS count
		   FROM chat_grouped
		   WHERE rn = 1
		   UNION ALL
		   SELECT id, user_id, type, reference_id, reference_type, actor_id,
		          COALESCE(message, '') AS message, read, created_at, 1 AS count
		   FROM notifications
		   WHERE user_id = ? AND NOT (type = ? AND read = 0)
		 )
		 SELECT c.id, c.user_id, c.type, c.reference_id, c.reference_type, c.actor_id,
		        c.message, c.read, c.created_at, c.count,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM combined c
		 LEFT JOIN users u ON c.actor_id = u.id
		 LEFT JOIN user_roles ur ON c.actor_id = ur.user_id
		 ORDER BY c.created_at DESC
		 LIMIT ? OFFSET ?`,
		userID, dto.NotifChatRoomMessage, userID, dto.NotifChatRoomMessage, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.NotificationRow
	for rows.Next() {
		var n model.NotificationRow
		var readInt int
		var actorID sql.NullString
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.ReferenceID, &n.ReferenceType, &actorID, &n.Message, &readInt, &n.CreatedAt, &n.Count,
			&n.ActorUsername, &n.ActorDisplayName, &n.ActorAvatarURL, &n.ActorRole,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		if actorID.Valid {
			if id, err := uuid.Parse(actorID.String); err == nil {
				n.ActorID = id
			}
		}
		n.Read = readInt == 1
		notifications = append(notifications, n)
	}

	for i := range notifications {
		n := &notifications[i]
		if n.Type == dto.NotifChatRoomMessage && n.Count > 1 {
			roomName := strings.TrimPrefix(n.Message, chatRoomMessagePrefix)
			n.Message = fmt.Sprintf("%d messages sent in %s", n.Count, roomName)
		}
	}

	return notifications, total, rows.Err()
}

func (r *notificationRepository) GetByID(ctx context.Context, id int, userID uuid.UUID) (*model.NotificationRow, error) {
	var n model.NotificationRow
	var readInt int
	var actorID sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
		        COALESCE(n.message, ''), n.read, n.created_at,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM notifications n
		 LEFT JOIN users u ON n.actor_id = u.id
		 LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
		 WHERE n.id = ? AND n.user_id = ?`,
		id, userID,
	).Scan(
		&n.ID, &n.UserID, &n.Type, &n.ReferenceID, &n.ReferenceType, &actorID, &n.Message, &readInt, &n.CreatedAt,
		&n.ActorUsername, &n.ActorDisplayName, &n.ActorAvatarURL, &n.ActorRole,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	if actorID.Valid {
		if parsed, err := uuid.Parse(actorID.String); err == nil {
			n.ActorID = parsed
		}
	}
	n.Read = readInt == 1
	n.Count = 1
	return &n, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, id int, userID uuid.UUID) error {
	var notifType dto.NotificationType
	var referenceID uuid.UUID
	var readInt int
	err := r.db.QueryRowContext(ctx,
		`SELECT type, reference_id, read FROM notifications WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&notifType, &referenceID, &readInt)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup notification: %w", err)
	}

	if notifType == dto.NotifChatRoomMessage && readInt == 0 {
		_, err = r.db.ExecContext(ctx,
			`UPDATE notifications SET read = 1
			 WHERE user_id = ? AND type = ? AND reference_id = ? AND read = 0`,
			userID, notifType, referenceID,
		)
		if err != nil {
			return fmt.Errorf("mark grouped notifications read: %w", err)
		}
		return nil
	}

	_, err = r.db.ExecContext(ctx,
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
