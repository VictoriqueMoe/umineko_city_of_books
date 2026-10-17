package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	BlockRepository interface {
		dao.BlockDAO
	}

	blockRepository struct {
		dao.BlockDAO

		cache *cache.Manager
	}
)

func NewBlockRepo(d dao.BlockDAO, c *cache.Manager) BlockRepository {
	return &blockRepository{BlockDAO: d, cache: c}
}

func (r *blockRepository) GetBlockedIDs(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	load := func(ctx context.Context) ([]uuid.UUID, error) {
		return r.BlockDAO.GetBlockedIDs(ctx, userID, tx...)
	}

	return r.cache.Load(ctx, cache.UserBlockedIDs.Key(userID.String()), load)
}

func (r *blockRepository) Block(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error {
	if err := r.BlockDAO.Block(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidateBlockedIDs(ctx, s)

	return nil
}

func (r *blockRepository) Unblock(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error {
	if err := r.BlockDAO.Unblock(ctx, s, tx...); err != nil {
		return err
	}

	r.invalidateBlockedIDs(ctx, s)

	return nil
}

func (r *blockRepository) invalidateBlockedIDs(ctx context.Context, s spec.BlockSpec) {
	keys := []cache.Key{
		cache.UserBlockedIDs.Key(s.BlockerID.String()),
		cache.UserBlockedIDs.Key(s.BlockedID.String()),
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("blocker_id", s.BlockerID.String()).Str("blocked_id", s.BlockedID.String()).Msg("failed to invalidate blocked id cache after write")
	}
}
