package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type (
	BlockRepository interface {
		Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) error
		Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) error
		IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) (bool, error)
		IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID, tx ...*sql.Tx) (bool, error)
		GetBlockedIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, tx ...*sql.Tx) ([]BlockedUser, error)
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

func (r *blockRepository) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Block(ctx, blockerID, blockedID, tx...)
}

func (r *blockRepository) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) error {
	return r.dao.Unblock(ctx, blockerID, blockedID, tx...)
}

func (r *blockRepository) IsBlocked(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsBlocked(ctx, blockerID, blockedID, tx...)
}

func (r *blockRepository) IsBlockedEither(ctx context.Context, userA uuid.UUID, userB uuid.UUID, tx ...*sql.Tx) (bool, error) {
	return r.dao.IsBlockedEither(ctx, userA, userB, tx...)
}

func (r *blockRepository) GetBlockedIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	return r.dao.GetBlockedIDs(ctx, userID, tx...)
}

func (r *blockRepository) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, tx ...*sql.Tx) ([]BlockedUser, error) {
	return r.dao.GetBlockedUsers(ctx, blockerID, tx...)
}
