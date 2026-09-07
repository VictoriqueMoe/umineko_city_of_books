package spec

import (
	"time"

	"github.com/google/uuid"
)

type (
	NewPasswordReset struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
	}
)
