package model

import (
	"github.com/google/uuid"
)

type (
	FollowUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
	}
)
