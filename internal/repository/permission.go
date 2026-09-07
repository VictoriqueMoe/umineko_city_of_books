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
	PermissionRepository interface {
		dao.PermissionDAO
	}

	permissionRepository struct {
		dao.PermissionDAO

		cache *cache.Manager
	}
)

func NewPermissionRepo(d dao.PermissionDAO, c *cache.Manager) PermissionRepository {
	return &permissionRepository{PermissionDAO: d, cache: c}
}

func (r *permissionRepository) GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.PermissionDAO.GetRolePermissions(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.RolePermissions, load)
}

func (r *permissionRepository) SetRolePermissions(ctx context.Context, s spec.RolePermissionsUpdate, tx ...*sql.Tx) error {
	if err := r.PermissionDAO.SetRolePermissions(ctx, s, tx...); err != nil {
		return err
	}

	if err := r.cache.Del(ctx, cache.RolePermissions.Key()); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("role", s.RoleName).Msg("failed to invalidate role permission cache after write")
	}

	return nil
}

func (r *permissionRepository) GetVanityRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.PermissionDAO.GetVanityRolePermissions(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.VanityRolePermissions, load)
}

func (r *permissionRepository) SetVanityRolePermissions(ctx context.Context, s spec.VanityRolePermissionsUpdate, tx ...*sql.Tx) error {
	if err := r.PermissionDAO.SetVanityRolePermissions(ctx, s, tx...); err != nil {
		return err
	}

	if err := r.cache.Del(ctx, cache.VanityRolePermissions.Key()); err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("vanity_role_id", s.VanityRoleID).Msg("failed to invalidate vanity role permission cache after write")
	}

	return nil
}

func (r *permissionRepository) GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	load := func(ctx context.Context) ([]string, error) {
		return r.PermissionDAO.GetVanityRoleIDsForUser(ctx, userID, tx...)
	}

	return r.cache.Load(ctx, cache.UserVanityRoleIDs, load, userID.String())
}
