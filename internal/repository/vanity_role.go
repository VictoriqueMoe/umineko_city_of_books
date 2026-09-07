package repository

import (
	"context"
	"database/sql"
	"strings"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/model/spec"
)

type (
	VanityRoleRepository interface {
		dao.VanityRoleDAO

		MoveUserRole(ctx context.Context, move spec.VanityRoleMove, tx ...*sql.Tx) error
	}

	vanityRoleRepository struct {
		dao.VanityRoleDAO

		db    *sql.DB
		cache *cache.Manager
	}
)

func ExcludeVanityRoleIDs(ids []string, startIndex int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := utils.PlaceholderArgs(ids, startIndex)

	return " AND id NOT IN (" + strings.Join(placeholders, ", ") + ")", args
}

func NewVanityRoleRepo(database *sql.DB, d dao.VanityRoleDAO, c *cache.Manager) VanityRoleRepository {
	return &vanityRoleRepository{VanityRoleDAO: d, db: database, cache: c}
}

func (r *vanityRoleRepository) MoveUserRole(ctx context.Context, move spec.VanityRoleMove, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.VanityRoleDAO.AssignToUser(ctx, spec.VanityRoleAssignment{UserID: move.UserID, RoleID: move.ToRoleID}, tx); err != nil {
			return err
		}

		return r.VanityRoleDAO.UnassignFromUser(ctx, spec.VanityRoleAssignment{UserID: move.UserID, RoleID: move.FromRoleID}, tx)
	})
	if err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(move.UserID.String()))
}

func (r *vanityRoleRepository) Delete(ctx context.Context, id string, tx ...*sql.Tx) error {
	if err := r.VanityRoleDAO.Delete(ctx, id, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.VanityRolePermissions.Key())
}

func (r *vanityRoleRepository) AssignToUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	if err := r.VanityRoleDAO.AssignToUser(ctx, s, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(s.UserID.String()))
}

func (r *vanityRoleRepository) UnassignFromUser(ctx context.Context, s spec.VanityRoleAssignment, tx ...*sql.Tx) error {
	if err := r.VanityRoleDAO.UnassignFromUser(ctx, s, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(s.UserID.String()))
}

func (r *vanityRoleRepository) GetAllAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	load := func(ctx context.Context) (map[string][]string, error) {
		return r.VanityRoleDAO.GetAllAssignments(ctx, tx...)
	}

	return r.cache.Load(ctx, cache.VanityAssignments, load)
}
