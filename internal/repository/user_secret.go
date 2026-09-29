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
	UserSecretRepository interface {
		dao.UserSecretDAO
	}
)

type userSecretRepository struct {
	dao.UserSecretDAO
	cache *cache.Manager
}

func NewUserSecretRepo(d dao.UserSecretDAO, c *cache.Manager) UserSecretRepository {
	return &userSecretRepository{UserSecretDAO: d, cache: c}
}

func (r *userSecretRepository) Unlock(ctx context.Context, s spec.SecretUnlock, tx ...*sql.Tx) error {
	if err := r.UserSecretDAO.Unlock(ctx, s, tx...); err != nil {
		return err
	}

	if err := r.cache.Del(ctx, cache.SecretHolders.Key(s.SecretID), cache.SecretSolved.Key(s.SecretID)); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("secret_id", s.SecretID).Msg("failed to invalidate secret caches after an unlock")
	}

	return nil
}

func (r *userSecretRepository) GetUserIDsWithSecret(ctx context.Context, secretID string, tx ...*sql.Tx) ([]uuid.UUID, error) {
	load := func(ctx context.Context) ([]uuid.UUID, error) {
		return r.UserSecretDAO.GetUserIDsWithSecret(ctx, secretID, tx...)
	}

	return r.cache.Load(ctx, cache.SecretHolders, load, secretID)
}

func (r *userSecretRepository) IsSolvedByAnyone(ctx context.Context, secretID string, tx ...*sql.Tx) (bool, error) {
	load := func(ctx context.Context) (bool, error) {
		return r.UserSecretDAO.IsSolvedByAnyone(ctx, secretID, tx...)
	}

	return r.cache.Load(ctx, cache.SecretSolved, load, secretID)
}

func (r *userSecretRepository) DeleteSecrets(ctx context.Context, secretIDs []string, tx ...*sql.Tx) error {
	if err := r.UserSecretDAO.DeleteSecrets(ctx, secretIDs, tx...); err != nil {
		return err
	}

	if len(secretIDs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(secretIDs)*2)
	for _, id := range secretIDs {
		keys = append(keys, cache.SecretHolders.Key(id), cache.SecretSolved.Key(id))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Strs("secret_ids", secretIDs).Msg("failed to invalidate secret caches after deleting secrets")
	}

	return nil
}
