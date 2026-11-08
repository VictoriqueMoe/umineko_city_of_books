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

		InvalidateBlockedIDs(ctx context.Context, userIDs ...uuid.UUID)
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

	r.InvalidateBlockedIDs(ctx, s.BlockerID, s.BlockedID)

	return nil
}

func (r *blockRepository) Unblock(ctx context.Context, s spec.BlockSpec, tx ...*sql.Tx) error {
	if err := r.BlockDAO.Unblock(ctx, s, tx...); err != nil {
		return err
	}

	r.InvalidateBlockedIDs(ctx, s.BlockerID, s.BlockedID)

	return nil
}

func (r *blockRepository) InvalidateBlockedIDs(ctx context.Context, userIDs ...uuid.UUID) {
	if len(userIDs) == 0 {
		return
	}

	keys := make([]cache.Key, 0, len(userIDs))
	ids := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, cache.UserBlockedIDs.Key(userID.String()))
		ids = append(ids, userID.String())
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Strs("user_ids", ids).Msg("failed to invalidate blocked id cache after write")
	}
}
