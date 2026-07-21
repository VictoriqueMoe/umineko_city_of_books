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
