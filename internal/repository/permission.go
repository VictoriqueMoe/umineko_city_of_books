package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/logger"

	"github.com/google/uuid"
)

type (
	PermissionRepository interface {
		GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error)
		SetRolePermissions(ctx context.Context, roleName string, perms []string, tx ...*sql.Tx) error
		GetVanityRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error)
		SetVanityRolePermissions(ctx context.Context, vanityRoleID string, perms []string, tx ...*sql.Tx) error
		GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	permissionRepository struct {
		dao   PermissionRepository
		cache *cache.Manager
	}
)

func NewPermissionRepo(dao PermissionRepository, c *cache.Manager) PermissionRepository {
	return &permissionRepository{dao: dao, cache: c}
}

func cachedRead[T any](ctx context.Context, m *cache.Manager, ns cache.Namespace, load func(context.Context) (T, error), parts ...string) (T, error) {
	key := ns.Key(parts...)

	if v, err := cache.Get[T](ctx, m, key); err == nil {
		return v, nil
	}

	v, err := load(ctx)
	if err != nil {
		var zero T
		return zero, err
	}

	_ = cache.Set(ctx, m, key, v, ns.TTL)

	return v, nil
}

func (r *permissionRepository) GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.dao.GetRolePermissions(ctx, tx...)
	}

	return cachedRead(ctx, r.cache, cache.RolePermissions, load)
}

func (r *permissionRepository) SetRolePermissions(ctx context.Context, roleName string, perms []string, tx ...*sql.Tx) error {
	if err := r.dao.SetRolePermissions(ctx, roleName, perms, tx...); err != nil {
		return err
	}

	if err := r.cache.Del(ctx, cache.RolePermissions.Key()); err != nil {
		logger.Log.Error().Err(err).Str("role", roleName).Msg("failed to invalidate role permission cache after write")
	}

	return nil
}

func (r *permissionRepository) GetVanityRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.dao.GetVanityRolePermissions(ctx, tx...)
	}

	return cachedRead(ctx, r.cache, cache.VanityRolePermissions, load)
}

func (r *permissionRepository) SetVanityRolePermissions(ctx context.Context, vanityRoleID string, perms []string, tx ...*sql.Tx) error {
	if err := r.dao.SetVanityRolePermissions(ctx, vanityRoleID, perms, tx...); err != nil {
		return err
	}

	if err := r.cache.Del(ctx, cache.VanityRolePermissions.Key()); err != nil {
		logger.Log.Error().Err(err).Str("vanity_role_id", vanityRoleID).Msg("failed to invalidate vanity role permission cache after write")
	}

	return nil
}

func (r *permissionRepository) GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	load := func(ctx context.Context) ([]string, error) {
		return r.dao.GetVanityRoleIDsForUser(ctx, userID, tx...)
	}

	return cachedRead(ctx, r.cache, cache.UserVanityRoleIDs, load, userID.String())
}
