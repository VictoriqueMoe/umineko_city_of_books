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
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		userID, notifType, referenceID, referenceType, actorArg, message,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert notification: %w", err)
	}
	return id, nil
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.NotificationRow, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT
		   (SELECT COUNT(DISTINCT reference_id) FROM notifications
		      WHERE user_id = $1 AND type = $2 AND read = FALSE) +
		   (SELECT COUNT(*) FROM notifications
		      WHERE user_id = $3 AND NOT (type = $4 AND read = FALSE))`,
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
		   WHERE user_id = $1 AND type = $2 AND read = FALSE
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
		   WHERE user_id = $3 AND NOT (type = $4 AND read = FALSE)
		 )
		 SELECT c.id, c.user_id, c.type, c.reference_id, c.reference_type, c.actor_id,
		        c.message, c.read, c.created_at, c.count,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM combined c
		 LEFT JOIN users u ON c.actor_id = u.id
		 LEFT JOIN user_roles ur ON c.actor_id = ur.user_id
		 ORDER BY c.created_at DESC
		 LIMIT $5 OFFSET $6`,
		userID, dto.NotifChatRoomMessage, userID, dto.NotifChatRoomMessage, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.NotificationRow
	for rows.Next() {
		var n model.NotificationRow
		var actorID *uuid.UUID
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.ReferenceID, &n.ReferenceType, &actorID, &n.Message, &n.Read, &n.CreatedAt, &n.Count,
			&n.ActorUsername, &n.ActorDisplayName, &n.ActorAvatarURL, &n.ActorRole,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		if actorID != nil {
			n.ActorID = *actorID
		}
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
	var actorID *uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
		        COALESCE(n.message, ''), n.read, n.created_at,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM notifications n
		 LEFT JOIN users u ON n.actor_id = u.id
		 LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
		 WHERE n.id = $1 AND n.user_id = $2`,
		id, userID,
	).Scan(
		&n.ID, &n.UserID, &n.Type, &n.ReferenceID, &n.ReferenceType, &actorID, &n.Message, &n.Read, &n.CreatedAt,
		&n.ActorUsername, &n.ActorDisplayName, &n.ActorAvatarURL, &n.ActorRole,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	if actorID != nil {
		n.ActorID = *actorID
	}
	n.Count = 1
	return &n, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, id int, userID uuid.UUID) error {
	var notifType dto.NotificationType
	var referenceID uuid.UUID
	var read bool
	err := r.db.QueryRowContext(ctx,
		`SELECT type, reference_id, read FROM notifications WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&notifType, &referenceID, &read)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup notification: %w", err)
	}

	if notifType == dto.NotifChatRoomMessage && !read {
		_, err = r.db.ExecContext(ctx,
			`UPDATE notifications SET read = TRUE
			 WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND read = FALSE`,
			userID, notifType, referenceID,
		)
		if err != nil {
			return fmt.Errorf("mark grouped notifications read: %w", err)
		}
		return nil
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID,
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
		 WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND actor_id = $4
		 AND created_at > NOW() - INTERVAL '1 hour'`,
		userID, notifType, referenceID, actorID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check duplicate notification: %w", err)
	}
	return count > 0, nil
}
