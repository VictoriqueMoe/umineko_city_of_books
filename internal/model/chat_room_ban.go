package model

import (
	"github.com/google/uuid"
)

type (
	ChatRoomBanRow struct {
		RoomID            uuid.UUID
		UserID            uuid.UUID
		Username          string
		DisplayName       string
		AvatarURL         string
		Role              string
		BannedByID        *uuid.UUID
		BannedByUsername  string
		BannedByDisplay   string
		BannedByAvatarURL string
		Reason            string
		CreatedAt         string
	}
)
