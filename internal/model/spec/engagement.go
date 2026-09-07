package spec

import (
	"github.com/google/uuid"
)

type (
	Like struct {
		UserID   uuid.UUID
		TargetID uuid.UUID
	}

	LikedByQuery struct {
		TargetID       uuid.UUID
		ExcludeUserIDs []uuid.UUID
	}

	Vote struct {
		UserID   uuid.UUID
		TargetID uuid.UUID
		Value    int
	}

	ViewRecord struct {
		TargetID   uuid.UUID
		ViewerHash string
	}

	OwnedDeletion struct {
		ID     uuid.UUID
		UserID uuid.UUID
	}
)
