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

type chatRoomBanRepository struct {
	dao ChatRoomBanRepository
}

func NewChatRoomBanRepo(dao ChatRoomBanRepository) ChatRoomBanRepository {
	return &chatRoomBanRepository{dao: dao}
}

func (r *chatRoomBanRepository) Ban(ctx context.Context, roomID, userID uuid.UUID, bannedBy *uuid.UUID, reason string) error {
	return r.dao.Ban(ctx, roomID, userID, bannedBy, reason)
}

func (r *chatRoomBanRepository) Unban(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.dao.Unban(ctx, roomID, userID)
}

func (r *chatRoomBanRepository) IsBanned(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return r.dao.IsBanned(ctx, roomID, userID)
}

func (r *chatRoomBanRepository) ListForRoom(ctx context.Context, roomID uuid.UUID) ([]ChatRoomBanRow, error) {
	return r.dao.ListForRoom(ctx, roomID)
}

func (r *chatRoomBanRepository) BannedRoomIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.BannedRoomIDsForUser(ctx, userID)
}
