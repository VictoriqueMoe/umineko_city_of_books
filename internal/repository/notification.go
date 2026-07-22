package repository

import (
	"context"
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
		) (int64, error)
		ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.NotificationRow, int, error)
		GetByID(ctx context.Context, id int, userID uuid.UUID) (*model.NotificationRow, error)
		MarkRead(ctx context.Context, id int, userID uuid.UUID) error
		MarkAllRead(ctx context.Context, userID uuid.UUID) error
		UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
		HasRecentDuplicate(ctx context.Context, userID uuid.UUID, notifType dto.NotificationType, referenceID uuid.UUID, actorID uuid.UUID) (bool, error)
		DeleteOlderThanBatch(ctx context.Context, cutoff time.Time, limit int) (int64, error)
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
) (int64, error) {
	return r.dao.Create(ctx, userID, notifType, referenceID, referenceType, actorID, message)
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.NotificationRow, int, error) {
	return r.dao.ListByUser(ctx, userID, limit, offset)
}

func (r *notificationRepository) GetByID(ctx context.Context, id int, userID uuid.UUID) (*model.NotificationRow, error) {
	return r.dao.GetByID(ctx, id, userID)
}

func (r *notificationRepository) MarkRead(ctx context.Context, id int, userID uuid.UUID) error {
	return r.dao.MarkRead(ctx, id, userID)
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return r.dao.MarkAllRead(ctx, userID)
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.dao.UnreadCount(ctx, userID)
}

func (r *notificationRepository) HasRecentDuplicate(ctx context.Context, userID uuid.UUID, notifType dto.NotificationType, referenceID uuid.UUID, actorID uuid.UUID) (bool, error) {
	return r.dao.HasRecentDuplicate(ctx, userID, notifType, referenceID, actorID)
}

func (r *notificationRepository) DeleteOlderThanBatch(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	return r.dao.DeleteOlderThanBatch(ctx, cutoff, limit)
}
