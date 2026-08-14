package repository

import (
	"context"
	"database/sql"
	"time"
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
			tx ...*sql.Tx,
		) (*model.NotificationRow, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.NotificationRow, int, error)
		GetByID(ctx context.Context, id int, userID uuid.UUID, tx ...*sql.Tx) (*model.NotificationRow, error)
		MarkRead(ctx context.Context, id int, userID uuid.UUID, tx ...*sql.Tx) error
		MarkAllRead(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error
		UnreadCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		HasRecentDuplicate(ctx context.Context, userID uuid.UUID, notifType dto.NotificationType, referenceID uuid.UUID, actorID uuid.UUID, tx ...*sql.Tx) (bool, error)
		HasRecentFromActor(ctx context.Context, notifType dto.NotificationType, actorID uuid.UUID, within time.Duration, tx ...*sql.Tx) (bool, error)
		DeleteOlderThanBatch(ctx context.Context, cutoff time.Time, limit int, tx ...*sql.Tx) (int64, error)
	}
)

type notificationRepository struct {
	dao NotificationRepository
}

func NewNotificationRepo(dao NotificationRepository) NotificationRepository {
	return &notificationRepository{dao: dao}
}

func (r *notificationRepository) Create(
	ctx context.Context,
	userID uuid.UUID,
	notifType dto.NotificationType,
	referenceID uuid.UUID,
	referenceType string,
	actorID uuid.UUID,
	message string,
	tx ...*sql.Tx,
) (*model.NotificationRow, error) {
	return r.dao.Create(ctx, userID, notifType, referenceID, referenceType, actorID, message, tx...)
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, tx ...*sql.Tx) ([]model.NotificationRow, int, error) {
	return r.dao.ListByUser(ctx, userID, limit, offset, tx...)
}

func (r *notificationRepository) GetByID(ctx context.Context, id int, userID uuid.UUID, tx ...*sql.Tx) (*model.NotificationRow, error) {
	return r.dao.GetByID(ctx, id, userID, tx...)
}

func (r *notificationRepository) MarkRead(ctx context.Context, id int, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkRead(ctx, id, userID, tx...)
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.MarkAllRead(ctx, userID, tx...)
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	return r.dao.UnreadCount(ctx, userID, tx...)
}

func (r *notificationRepository) HasRecentDuplicate(ctx context.Context, userID uuid.UUID, notifType dto.NotificationType, referenceID uuid.UUID, actorID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.HasRecentDuplicate(ctx, userID, notifType, referenceID, actorID, tx...)
}

func (r *notificationRepository) HasRecentFromActor(ctx context.Context, notifType dto.NotificationType, actorID uuid.UUID, within time.Duration, tx ...*sql.Tx) (bool, error) {
	return r.dao.HasRecentFromActor(ctx, notifType, actorID, within, tx...)
}

func (r *notificationRepository) DeleteOlderThanBatch(ctx context.Context, cutoff time.Time, limit int, tx ...*sql.Tx) (int64, error) {
	return r.dao.DeleteOlderThanBatch(ctx, cutoff, limit, tx...)
}
