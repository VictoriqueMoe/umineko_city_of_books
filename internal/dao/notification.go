package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
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

	notificationJoinRow = sqlcgen.GetNotificationByIDRow
)

var (
	collapsibleNotifTypes = []dto.NotificationType{dto.NotifChatRoomMessage, dto.NotifChatMessage}
)

func joinNotifTypes(types []dto.NotificationType) string {
	parts := make([]string, 0, len(types))
	for _, t := range types {
		parts = append(parts, string(t))
	}

	return strings.Join(parts, ",")
}

func toNotificationRow(row notificationJoinRow) (model.NotificationRow, error) {
	out := model.NotificationRow{
		ID:               int(row.ID),
		UserID:           row.UserID,
		Type:             dto.NotificationType(row.Type),
		ReferenceType:    row.ReferenceType,
		Message:          row.Message,
		Read:             row.Read,
		CreatedAt:        row.CreatedAt,
		Count:            1,
		ActorUsername:    row.ActorUsername,
		ActorDisplayName: row.ActorDisplayName,
		ActorAvatarURL:   row.ActorAvatarUrl,
		ActorRole:        row.ActorRole,
	}

	if row.ReferenceID != "" {
		referenceID, err := uuid.Parse(row.ReferenceID)
		if err != nil {
			return model.NotificationRow{}, fmt.Errorf("parse notification reference id: %w", err)
		}

		out.ReferenceID = referenceID
	}

	if row.ActorID != nil {
		out.ActorID = *row.ActorID
	}

	return out, nil
}

func (r *notificationDAO) Create(ctx context.Context, s spec.NewNotification, tx ...*sql.Tx) (*model.NotificationRow, error) {
	var actorID *uuid.UUID
	if s.ActorID != uuid.Nil {
		actorID = &s.ActorID
	}

	created, err := genQueries(r.db, tx).CreateNotification(ctx, sqlcgen.CreateNotificationParams{
		UserID:        s.UserID,
		Type:          string(s.Type),
		ReferenceID:   s.ReferenceID.String(),
		ReferenceType: s.ReferenceType,
		ActorID:       actorID,
		Message:       s.Message,
	})
	if err != nil {
		return nil, fmt.Errorf("insert notification: %w", err)
	}

	n, err := toNotificationRow(notificationJoinRow(created))
	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (r *notificationDAO) ListByUser(ctx context.Context, q spec.NotificationListing, tx ...*sql.Tx) ([]model.NotificationRow, int, error) {
	queries := genQueries(r.db, tx)
	collapsible := joinNotifTypes(collapsibleNotifTypes)

	total, err := queries.CountUserNotifications(ctx, sqlcgen.CountUserNotificationsParams{
		UserID:  q.UserID,
		Column2: collapsible,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	rows, err := queries.ListUserNotifications(ctx, sqlcgen.ListUserNotificationsParams{
		UserID:  q.UserID,
		Column2: collapsible,
		Limit:   int32(q.Limit),
		Offset:  int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}

	var notifications []model.NotificationRow
	for _, row := range rows {
		n, err := toNotificationRow(notificationJoinRow{
			ID:               row.ID,
			UserID:           row.UserID,
			Type:             row.Type,
			ReferenceID:      row.ReferenceID,
			ReferenceType:    row.ReferenceType,
			ActorID:          row.ActorID,
			Message:          row.Message,
			Read:             row.Read,
			CreatedAt:        row.CreatedAt,
			ActorUsername:    row.ActorUsername,
			ActorDisplayName: row.ActorDisplayName,
			ActorAvatarUrl:   row.ActorAvatarUrl,
			ActorRole:        row.ActorRole,
		})
		if err != nil {
			return nil, 0, err
		}

		n.Count = int(row.Count)
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

	return notifications, int(total), nil
}

func (r *notificationDAO) GetByID(ctx context.Context, s spec.NotificationLookup, tx ...*sql.Tx) (*model.NotificationRow, error) {
	row, err := genQueries(r.db, tx).GetNotificationByID(ctx, sqlcgen.GetNotificationByIDParams{
		ID:     int64(s.ID),
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification by id: %w", err)
	}

	n, err := toNotificationRow(row)
	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (r *notificationDAO) MarkRead(ctx context.Context, s spec.NotificationLookup, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	state, err := queries.GetNotificationReadState(ctx, sqlcgen.GetNotificationReadStateParams{
		ID:     int64(s.ID),
		UserID: s.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup notification: %w", err)
	}

	if !state.Read && slices.Contains(collapsibleNotifTypes, dto.NotificationType(state.Type)) {
		err = queries.MarkGroupedNotificationsRead(ctx, sqlcgen.MarkGroupedNotificationsReadParams{
			UserID:      s.UserID,
			Type:        state.Type,
			ReferenceID: state.ReferenceID,
		})
		if err != nil {
			return fmt.Errorf("mark grouped notifications read: %w", err)
		}

		return nil
	}

	err = queries.MarkNotificationRead(ctx, sqlcgen.MarkNotificationReadParams{
		ID:     int64(s.ID),
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}

	return nil
}

func (r *notificationDAO) MarkAllRead(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).MarkAllNotificationsRead(ctx, userID); err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}

	return nil
}

func (r *notificationDAO) MarkReadByReference(ctx context.Context, s spec.NotificationReferenceRead, tx ...*sql.Tx) error {
	if len(s.Types) == 0 {
		return nil
	}

	err := genQueries(r.db, tx).MarkNotificationsReadByReference(ctx, sqlcgen.MarkNotificationsReadByReferenceParams{
		UserID:      s.UserID,
		ReferenceID: s.ReferenceID.String(),
		Column3:     joinNotifTypes(s.Types),
	})
	if err != nil {
		return fmt.Errorf("mark notifications read by reference: %w", err)
	}

	return nil
}

func (r *notificationDAO) DeleteOlderThanBatch(ctx context.Context, s spec.NotificationPruneBatch, tx ...*sql.Tx) (int64, error) {
	affected, err := genQueries(r.db, tx).DeleteNotificationsOlderThanBatch(ctx, sqlcgen.DeleteNotificationsOlderThanBatchParams{
		CreatedAt: s.Cutoff,
		Limit:     int32(s.Limit),
	})
	if err != nil {
		return 0, fmt.Errorf("delete old notifications batch: %w", err)
	}

	return affected, nil
}

func (r *notificationDAO) UnreadCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUnreadNotifications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}

	return int(count), nil
}

func (r *notificationDAO) HasRecentDuplicate(ctx context.Context, q spec.NotificationDuplicateCheck, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountRecentDuplicateNotifications(ctx, sqlcgen.CountRecentDuplicateNotificationsParams{
		UserID:      q.UserID,
		Type:        string(q.Type),
		ReferenceID: q.ReferenceID.String(),
		ActorID:     &q.ActorID,
	})
	if err != nil {
		return false, fmt.Errorf("check duplicate notification: %w", err)
	}

	return count > 0, nil
}

func (r *notificationDAO) HasRecentFromActor(ctx context.Context, q spec.NotificationActorRecency, tx ...*sql.Tx) (bool, error) {
	count, err := genQueries(r.db, tx).CountRecentNotificationsFromActor(ctx, sqlcgen.CountRecentNotificationsFromActorParams{
		Type:    string(q.Type),
		ActorID: &q.ActorID,
		Column3: q.Within.Seconds(),
	})
	if err != nil {
		return false, fmt.Errorf("check recent notification from actor: %w", err)
	}

	return count > 0, nil
}
