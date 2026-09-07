package spec

import (
	"time"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	NewNotification struct {
		UserID        uuid.UUID
		Type          dto.NotificationType
		ReferenceID   uuid.UUID
		ReferenceType string
		ActorID       uuid.UUID
		Message       string
	}

	NotificationListing struct {
		UserID uuid.UUID
		Limit  int
		Offset int
	}

	NotificationLookup struct {
		ID     int
		UserID uuid.UUID
	}

	NotificationReferenceRead struct {
		UserID      uuid.UUID
		ReferenceID uuid.UUID
		Types       []dto.NotificationType
	}

	NotificationDuplicateCheck struct {
		UserID      uuid.UUID
		Type        dto.NotificationType
		ReferenceID uuid.UUID
		ActorID     uuid.UUID
	}

	NotificationActorRecency struct {
		Type    dto.NotificationType
		ActorID uuid.UUID
		Within  time.Duration
	}

	NotificationPruneBatch struct {
		Cutoff time.Time
		Limit  int
	}
)
