package repository

import (
	"context"

	"umineko_city_of_books/internal/cache"

	"github.com/google/uuid"
)

type (
	UserSecretRepository interface {
		Unlock(ctx context.Context, userID uuid.UUID, secretID string) error
		ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error)
		GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error)
		GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string) ([]uuid.UUID, error)
		IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error)
		DeleteSecrets(ctx context.Context, secretIDs []string) error
	}
)

type userSecretRepository struct {
	dao   UserSecretRepository
	cache *cache.Manager
}

func NewUserSecretRepo(dao UserSecretRepository, c *cache.Manager) UserSecretRepository {
	return &userSecretRepository{dao: dao, cache: c}
}

func (r *userSecretRepository) Unlock(ctx context.Context, userID uuid.UUID, secretID string) error {
	if err := r.dao.Unlock(ctx, userID, secretID); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.SecretHolders.Key(secretID), cache.SecretSolved.Key(secretID))
}

func (r *userSecretRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return r.dao.ListForUser(ctx, userID)
}

func (r *userSecretRepository) GetUserIDsWithSecret(ctx context.Context, secretID string) ([]uuid.UUID, error) {
	key := cache.SecretHolders.Key(secretID)

	if v, err := cache.Get[[]uuid.UUID](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetUserIDsWithSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.SecretHolders.TTL)
	return v, nil
}

func (r *userSecretRepository) GetUserIDsWithAnyPiece(ctx context.Context, pieceIDs []string) ([]uuid.UUID, error) {
	return r.dao.GetUserIDsWithAnyPiece(ctx, pieceIDs)
}

func (r *userSecretRepository) IsSolvedByAnyone(ctx context.Context, secretID string) (bool, error) {
	key := cache.SecretSolved.Key(secretID)

	if v, err := cache.Get[bool](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.IsSolvedByAnyone(ctx, secretID)
	if err != nil {
		return false, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.SecretSolved.TTL)
	return v, nil
}

func (r *userSecretRepository) DeleteSecrets(ctx context.Context, secretIDs []string) error {
	if err := r.dao.DeleteSecrets(ctx, secretIDs); err != nil {
		return err
	}

	if len(secretIDs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(secretIDs)*2)
	for _, id := range secretIDs {
		keys = append(keys, cache.SecretHolders.Key(id), cache.SecretSolved.Key(id))
	}

	return r.cache.Del(ctx, keys...)
}
