package model

import (
	"time"

	"github.com/google/uuid"
)

type (
	SessionAuthState struct {
		UserID                    uuid.UUID
		ExpiresAt                 time.Time
		Banned                    bool
		Locked                    bool
		RequiresEmailVerification bool
	}
)
