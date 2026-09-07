package model

import (
	"time"

	"github.com/google/uuid"
)

type (
	EmailVerificationToken struct {
		TokenHash string
		UserID    uuid.UUID
		ExpiresAt time.Time
		UsedAt    *time.Time
		CreatedAt time.Time
	}
)
