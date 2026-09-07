package spec

import (
	"time"

	"github.com/google/uuid"
)

type (
	NewEmailVerification struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
	}
)
