package repository

import (
	"context"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	RoleRepository interface {
		GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error)
		GetRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]role.Role, error)
		HasRole(ctx context.Context, userID uuid.UUID, r role.Role) (bool, error)
		SetRole(ctx context.Context, userID uuid.UUID, r role.Role) error
		RemoveRole(ctx context.Context, userID uuid.UUID, r role.Role) error
		GetUsersByRoles(ctx context.Context, roles []role.Role) ([]uuid.UUID, error)
	}

	roleRepository struct {
		dao   RoleRepository
		cache *cache.Manager
	}
)

func NewRoleRepo(dao RoleRepository, c *cache.Manager) RoleRepository {
	return &roleRepository{dao: dao, cache: c}
}

func (r *roleRepository) GetRole(ctx context.Context, userID uuid.UUID) (role.Role, error) {
	key := cache.UserRole.Key(userID.String())

	if cached, err := cache.Get[string](ctx, r.cache, key); err == nil {
		return role.Role(cached), nil
	}

	rl, err := r.dao.GetRole(ctx, userID)
	if err != nil {
		return "", err
	}

	_ = cache.Set(ctx, r.cache, key, string(rl), cache.UserRole.TTL)
	return rl, nil
}

func (r *roleRepository) SetRole(ctx context.Context, userID uuid.UUID, rl role.Role) error {
	if err := r.dao.SetRole(ctx, userID, rl); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.UserRole.Key(userID.String()))
}

func (r *roleRepository) RemoveRole(ctx context.Context, userID uuid.UUID, rl role.Role) error {
	if err := r.dao.RemoveRole(ctx, userID, rl); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.UserRole.Key(userID.String()))
}

func (r *roleRepository) GetRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]role.Role, error) {
	return r.dao.GetRoles(ctx, userIDs)
}

func (r *roleRepository) HasRole(ctx context.Context, userID uuid.UUID, rl role.Role) (bool, error) {
	return r.dao.HasRole(ctx, userID, rl)
}

func (r *roleRepository) GetUsersByRoles(ctx context.Context, roles []role.Role) ([]uuid.UUID, error) {
	return r.dao.GetUsersByRoles(ctx, roles)
}
