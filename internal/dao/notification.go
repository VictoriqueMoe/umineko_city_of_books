package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

const (
	chatRoomMessagePrefix = "sent a message in "
)

type (
	NotificationDAO interface {
		Create(ctx context.Context, s spec.NewNotification, tx ...*sql.Tx) (*model.NotificationRow, error)
		ListByUser(ctx context.Context, q spec.NotificationListing, tx ...*sql.Tx) ([]model.NotificationRow, int, error)
		GetByID(ctx context.Context, s spec.NotificationLookup, tx ...*sql.Tx) (*model.NotificationRow, error)
		MarkRead(ctx context.Context, s spec.NotificationLookup, tx ...*sql.Tx) error
		MarkAllRead(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		MarkReadByReference(ctx context.Context, s spec.NotificationReferenceRead, tx ...*sql.Tx) error
		UnreadCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		HasRecentDuplicate(ctx context.Context, q spec.NotificationDuplicateCheck, tx ...*sql.Tx) (bool, error)
		HasRecentFromActor(ctx context.Context, q spec.NotificationActorRecency, tx ...*sql.Tx) (bool, error)
		DeleteOlderThanBatch(ctx context.Context, s spec.NotificationPruneBatch, tx ...*sql.Tx) (int64, error)
	}

	notificationDAO struct {
		db *sql.DB
	}
)

var (
	collapsibleNotifTypes = []dto.NotificationType{dto.NotifChatRoomMessage, dto.NotifChatMessage}
)

func (r *notificationDAO) Create(ctx context.Context, s spec.NewNotification, tx ...*sql.Tx) (*model.NotificationRow, error) {
	var actorArg any = s.ActorID
	if s.ActorID == uuid.Nil {
		actorArg = nil
	}

	var n model.NotificationRow
	var createdActorID *uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`WITH n AS (
		     INSERT INTO notifications (user_id, type, reference_id, reference_type, actor_id, message)
		     VALUES ($1, $2, $3, $4, $5, $6)
		     RETURNING id, user_id, type, reference_id, reference_type, actor_id, message, read, created_at
		 )
		 SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
		        COALESCE(n.message, ''), n.read, n.created_at,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM n
		 LEFT JOIN users u ON n.actor_id = u.id
		 LEFT JOIN user_roles ur ON n.actor_id = ur.user_id`,
		s.UserID, s.Type, s.ReferenceID, s.ReferenceType, actorArg, s.Message,
	).Scan(
		&n.ID, &n.UserID, &n.Type, &n.ReferenceID, &n.ReferenceType, &createdActorID, &n.Message, &n.Read, &n.CreatedAt,
		&n.ActorUsername, &n.ActorDisplayName, &n.ActorAvatarURL, &n.ActorRole,
	)
	if err != nil {
		return nil, fmt.Errorf("insert notification: %w", err)
	}

	if createdActorID != nil {
		n.ActorID = *createdActorID
	}
	n.Count = 1

	return &n, nil
}

func (r *notificationDAO) ListByUser(ctx context.Context, q spec.NotificationListing, tx ...*sql.Tx) ([]model.NotificationRow, int, error) {
	var total int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT
		   (SELECT COUNT(DISTINCT (type, reference_id)) FROM notifications
		      WHERE user_id = $1 AND type = ANY($2) AND read = FALSE) +
		   (SELECT COUNT(*) FROM notifications
		      WHERE user_id = $3 AND NOT (type = ANY($4) AND read = FALSE))`,
		q.UserID, collapsibleNotifTypes, q.UserID, collapsibleNotifTypes,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	rows, err := txOrDB(r.db, tx).QueryContext(ctx,
		`WITH chat_grouped AS (
		   SELECT
		     id, user_id, type, reference_id, reference_type, actor_id, message, read, created_at,
		     ROW_NUMBER() OVER (PARTITION BY type, reference_id ORDER BY created_at DESC, id DESC) AS rn,
		     COUNT(*) OVER (PARTITION BY type, reference_id) AS grp_count
		   FROM notifications
		   WHERE user_id = $1 AND type = ANY($2) AND read = FALSE
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
		   WHERE user_id = $3 AND NOT (type = ANY($4) AND read = FALSE)
		 )
		 SELECT c.id, c.user_id, c.type, c.reference_id, c.reference_type, c.actor_id,
		        c.message, c.read, c.created_at, c.count,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM combined c
		 LEFT JOIN users u ON c.actor_id = u.id
		 LEFT JOIN user_roles ur ON c.actor_id = ur.user_id
		 ORDER BY c.created_at DESC
		 LIMIT $5 OFFSET $6`,
		q.UserID, collapsibleNotifTypes, q.UserID, collapsibleNotifTypes, q.Limit, q.Offset,
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
		if n.Count > 1 {
			switch n.Type {
			case dto.NotifChatRoomMessage:
				roomName := strings.TrimPrefix(n.Message, chatRoomMessagePrefix)
				n.Message = fmt.Sprintf("%d messages sent in %s", n.Count, roomName)
			case dto.NotifChatMessage:
				n.Message = fmt.Sprintf("has sent you %d messages", n.Count)
			}
		}
		notifications = append(notifications, n)
	}

	return notifications, total, rows.Err()
}

func (r *notificationDAO) GetByID(ctx context.Context, s spec.NotificationLookup, tx ...*sql.Tx) (*model.NotificationRow, error) {
	var n model.NotificationRow
	var actorID *uuid.UUID
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT n.id, n.user_id, n.type, n.reference_id, n.reference_type, n.actor_id,
		        COALESCE(n.message, ''), n.read, n.created_at,
		        COALESCE(u.username, ''), COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), COALESCE(ur.role, '')
		 FROM notifications n
		 LEFT JOIN users u ON n.actor_id = u.id
		 LEFT JOIN user_roles ur ON n.actor_id = ur.user_id
		 WHERE n.id = $1 AND n.user_id = $2`,
		s.ID, s.UserID,
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

func (r *notificationDAO) MarkRead(ctx context.Context, s spec.NotificationLookup, tx ...*sql.Tx) error {
	var notifType dto.NotificationType
	var referenceID uuid.UUID
	var read bool
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT type, reference_id, read FROM notifications WHERE id = $1 AND user_id = $2`,
		s.ID, s.UserID,
	).Scan(&notifType, &referenceID, &read)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup notification: %w", err)
	}

	if !read && slices.Contains(collapsibleNotifTypes, notifType) {
		_, err = txOrDB(r.db, tx).ExecContext(ctx,
			`UPDATE notifications SET read = TRUE
			 WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND read = FALSE`,
			s.UserID, notifType, referenceID,
		)
		if err != nil {
			return fmt.Errorf("mark grouped notifications read: %w", err)
		}
		return nil
	}

	_, err = txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`, s.ID, s.UserID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (r *notificationDAO) MarkAllRead(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID,
	)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (r *notificationDAO) MarkReadByReference(ctx context.Context, s spec.NotificationReferenceRead, tx ...*sql.Tx) error {
	if len(s.Types) == 0 {
		return nil
	}

	_, err := txOrDB(r.db, tx).ExecContext(ctx,
		`UPDATE notifications SET read = TRUE
		 WHERE user_id = $1 AND reference_id = $2 AND read = FALSE AND type = ANY($3)`,
		s.UserID, s.ReferenceID, s.Types,
	)
	if err != nil {
		return fmt.Errorf("mark notifications read by reference: %w", err)
	}

	return nil
}

func (r *notificationDAO) DeleteOlderThanBatch(ctx context.Context, s spec.NotificationPruneBatch, tx ...*sql.Tx) (int64, error) {
	res, err := txOrDB(r.db, tx).ExecContext(ctx,
		`DELETE FROM notifications
		 WHERE id IN (
		     SELECT id FROM notifications
		     WHERE created_at < $1
		     ORDER BY id
		     LIMIT $2
		 )`, s.Cutoff, s.Limit,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old notifications batch: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete old notifications rows affected: %w", err)
	}

	return affected, nil
}

func (r *notificationDAO) UnreadCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *notificationDAO) HasRecentDuplicate(ctx context.Context, q spec.NotificationDuplicateCheck, tx ...*sql.Tx) (bool, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications
		 WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND actor_id = $4
		 AND created_at > NOW() - INTERVAL '1 hour'`,
		q.UserID, q.Type, q.ReferenceID, q.ActorID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check duplicate notification: %w", err)
	}
	return count > 0, nil
}

func (r *notificationDAO) HasRecentFromActor(ctx context.Context, q spec.NotificationActorRecency, tx ...*sql.Tx) (bool, error) {
	var count int
	err := txOrDB(r.db, tx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications
		 WHERE type = $1 AND actor_id = $2
		 AND created_at > NOW() - make_interval(secs => $3)`,
		q.Type, q.ActorID, q.Within.Seconds(),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check recent notification from actor: %w", err)
	}
	return count > 0, nil
}
