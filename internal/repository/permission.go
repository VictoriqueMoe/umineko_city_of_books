package repository

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	PermissionRepository interface {
		dao.PermissionDAO

		MoveUserVanityRole(ctx context.Context, move spec.VanityRoleMove, tx ...*sql.Tx) error
		InvalidateVanityAssignments(ctx context.Context, userIDs ...string)
		InvalidateVanityRolePermissions(ctx context.Context)
	}

	permissionRepository struct {
		dao.PermissionDAO

		db    *sql.DB
		cache *cache.Manager
	}
)

func NewPermissionRepo(database *sql.DB, d dao.PermissionDAO, c *cache.Manager) PermissionRepository {
	return &permissionRepository{PermissionDAO: d, db: database, cache: c}
}

func (r *permissionRepository) GetRolePermissions(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.PermissionDAO.GetRolePermissions(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.RolePermissions.Key(), load)
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

	return r.cache.Load(ctx, cache.VanityRolePermissions.Key(), load)
}

func (r *permissionRepository) SetVanityRolePermissions(ctx context.Context, s spec.VanityRolePermissionsUpdate, tx ...*sql.Tx) error {
	if err := r.PermissionDAO.SetVanityRolePermissions(ctx, s, tx...); err != nil {
		return err
	}

	r.InvalidateVanityRolePermissions(ctx)

	return nil
}

func (r *permissionRepository) GetVanityRoleIDsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	load := func(ctx context.Context) ([]string, error) {
		return r.PermissionDAO.GetVanityRoleIDsForUser(ctx, userID, tx...)
	}

	return r.cache.Load(ctx, cache.UserVanityRoleIDs.Key(userID.String()), load)
}

func (r *permissionRepository) GetAllVanityRoleAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.PermissionDAO.GetAllVanityRoleAssignments(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.VanityAssignments.Key(), load)
}

func (r *permissionRepository) AssignVanityRoleToUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	if err := r.PermissionDAO.AssignVanityRoleToUser(ctx, s, tx...); err != nil {
		return err
	}

	r.InvalidateVanityAssignments(ctx, s.UserID.String())

	return nil
}

func (r *permissionRepository) UnassignVanityRoleFromUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	if err := r.PermissionDAO.UnassignVanityRoleFromUser(ctx, s, tx...); err != nil {
		return err
	}

	r.InvalidateVanityAssignments(ctx, s.UserID.String())

	return nil
}

func (r *permissionRepository) MoveUserVanityRole(ctx context.Context, move spec.VanityRoleMove, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.PermissionDAO.AssignVanityRoleToUser(ctx, spec.VanityRoleAssignment{UserID: move.UserID, RoleID: move.ToRoleID}, tx); err != nil {
			return err
		}

		return r.PermissionDAO.UnassignVanityRoleFromUser(ctx, spec.VanityRoleAssignment{UserID: move.UserID, RoleID: move.FromRoleID}, tx)
	})
	if err != nil {
		return err
	}

	r.InvalidateVanityAssignments(ctx, move.UserID.String())

	return nil
}

func (r *permissionRepository) InvalidateVanityAssignments(ctx context.Context, userIDs ...string) {
	keys := make([]cache.Key, 0, len(userIDs)+1)
	keys = append(keys, cache.VanityAssignments.Key())
	for _, userID := range userIDs {
		keys = append(keys, cache.UserVanityRoleIDs.Key(userID))
	}

	if err := r.cache.Del(ctx, keys...); err != nil {
		logger.Ctx(ctx).Error().Err(err).Strs("user_ids", userIDs).Msg("failed to invalidate vanity role assignment caches after write")
	}
}

func (r *permissionRepository) InvalidateVanityRolePermissions(ctx context.Context) {
	if err := r.cache.Del(ctx, cache.VanityRolePermissions.Key()); err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to invalidate vanity role permission cache after write")
	}
}
