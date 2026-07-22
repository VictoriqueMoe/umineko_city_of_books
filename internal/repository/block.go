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

type blockRepository struct {
	dao BlockRepository
}

func NewBlockRepo(dao BlockRepository) BlockRepository {
	return &blockRepository{dao: dao}
}

func (r *blockRepository) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	return r.dao.Block(ctx, blockerID, blockedID)
}

func (r *blockRepository) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	return r.dao.Unblock(ctx, blockerID, blockedID)
}

func (r *blockRepository) IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error) {
	return r.dao.IsBlocked(ctx, blockerID, blockedID)
}

func (r *blockRepository) IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error) {
	return r.dao.IsBlockedEither(ctx, userA, userB)
}

func (r *blockRepository) GetBlockedIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return r.dao.GetBlockedIDs(ctx, userID)
}

func (r *blockRepository) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID) ([]BlockedUser, error) {
	return r.dao.GetBlockedUsers(ctx, blockerID)
}
