package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	ChatRoomBanRepository interface {
		Ban(ctx context.Context, roomID, userID uuid.UUID, bannedBy *uuid.UUID, reason string) error
		Unban(ctx context.Context, roomID, userID uuid.UUID) error
		IsBanned(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
		ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatRoomBanRow, error)
		BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	}

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
