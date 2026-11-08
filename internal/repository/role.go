package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	RoleRepository interface {
		dao.RoleDAO

		InvalidateRole(ctx context.Context, userID uuid.UUID)
	}

	roleRepository struct {
		dao.RoleDAO

		cache *cache.Manager
	}
)

func NewRoleRepo(d dao.RoleDAO, c *cache.Manager) RoleRepository {
	return &roleRepository{RoleDAO: d, cache: c}
}

func (r *roleRepository) GetRole(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (role.Role, error) {
	load := func(ctx context.Context) (string, error) {
		rl, err := r.RoleDAO.GetRole(ctx, userID, tx...)

		return string(rl), err
	}

	cached, err := r.cache.Load(ctx, cache.UserRole.Key(userID.String()), load)
	if err != nil {
		return "", err
	}

	return role.Role(cached), nil
}

func (r *roleRepository) SetRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) error {
	if err := r.RoleDAO.SetRole(ctx, s, tx...); err != nil {
		return err
	}

	r.InvalidateRole(ctx, s.UserID)

	return nil
}

func (r *roleRepository) RemoveRole(ctx context.Context, s spec.UserRoleSpec, tx ...*sql.Tx) error {
	if err := r.RoleDAO.RemoveRole(ctx, s, tx...); err != nil {
		return err
	}

	r.InvalidateRole(ctx, s.UserID)

	return nil
}

func (r *roleRepository) InvalidateRole(ctx context.Context, userID uuid.UUID) {
	if err := r.cache.Del(ctx, cache.UserRole.Key(userID.String())); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("user_id", userID.String()).Msg("failed to invalidate the cached role after write")
	}
}
