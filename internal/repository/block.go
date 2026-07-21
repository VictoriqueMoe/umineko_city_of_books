package repository

import (
	"context"

	"github.com/google/uuid"
)

type (
	BlockRepository interface {
		Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
		Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
		IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error)
		IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error)
		GetBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
		GetBlockedUsers(ctx context.Context, blockerID uuid.UUID) ([]BlockedUser, error)
	}

	BlockedUser struct {
		ID          uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		BlockedAt   string
	}
)
