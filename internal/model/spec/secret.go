package spec

import (
	"github.com/google/uuid"
)

type (
	SecretPieceCount struct {
		UserID   uuid.UUID
		PieceIDs []string
	}

	SecretUnlock struct {
		UserID   uuid.UUID
		SecretID string
	}
)
