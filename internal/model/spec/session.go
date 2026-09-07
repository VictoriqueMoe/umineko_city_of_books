package spec

import (
	"time"

	"github.com/google/uuid"
)

type (
	NewSession struct {
		Token     string
		UserID    uuid.UUID
		ExpiresAt time.Time
	}

	SessionDeletionExcept struct {
		UserID    uuid.UUID
		KeepToken string
	}
)
