package model

import (
	"github.com/google/uuid"
)

type (
	BlockedUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		BlockedAt   string
	}
)
