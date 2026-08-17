package repository

import (
	"context"
	"database/sql"
	"strings"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/dao/utils"
	"umineko_city_of_books/internal/db"

	"github.com/google/uuid"
)

type (
	VanityRoleDAO interface {
		List(ctx context.Context, tx ...*sql.Tx) ([]VanityRoleRow, error)
		GetByID(ctx context.Context, id string, tx ...*sql.Tx) (*VanityRoleRow, error)
		Create(ctx context.Context, id, label, color string, sortOrder int, tx ...*sql.Tx) error
		Update(ctx context.Context, id, label, color string, sortOrder int, tx ...*sql.Tx) error
		Delete(ctx context.Context, id string, tx ...*sql.Tx) error
		AssignToUser(ctx context.Context, userID uuid.UUID, roleID string, tx ...*sql.Tx) error
		UnassignFromUser(ctx context.Context, userID uuid.UUID, roleID string, tx ...*sql.Tx) error
		GetUsersForRole(ctx context.Context, roleID string, search string, limit, offset int, tx ...*sql.Tx) ([]VanityRoleUserRow, int, error)
		GetRolesForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]VanityRoleRow, error)
		GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]VanityRoleRow, error)
		GetAllAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error)
	}

	VanityRoleRepository interface {
		VanityRoleDAO

		MoveUserRole(ctx context.Context, spec VanityRoleMove, tx ...*sql.Tx) error
	}

	VanityRoleMove struct {
		UserID     uuid.UUID
		FromRoleID string
		ToRoleID   string
	}

	VanityRoleRow struct {
		ID        string
		Label     string
		Color     string
		IsSystem  bool
		SortOrder int
	}

	VanityRoleUserRow struct {
		UserID      uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
	}
)

func ExcludeVanityRoleIDs(ids []string, startIndex int) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}

	placeholders, args := utils.PlaceholderArgs(ids, startIndex)

	return " AND id NOT IN (" + strings.Join(placeholders, ", ") + ")", args
}

type vanityRoleRepository struct {
	db    *sql.DB
	dao   VanityRoleDAO
	cache *cache.Manager
}

func NewVanityRoleRepo(database *sql.DB, dao VanityRoleDAO, c *cache.Manager) VanityRoleRepository {
	return &vanityRoleRepository{db: database, dao: dao, cache: c}
}

func (r *vanityRoleRepository) MoveUserRole(ctx context.Context, spec VanityRoleMove, tx ...*sql.Tx) error {
	err := db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error {
		if err := r.dao.AssignToUser(ctx, spec.UserID, spec.ToRoleID, tx); err != nil {
			return err
		}

		return r.dao.UnassignFromUser(ctx, spec.UserID, spec.FromRoleID, tx)
	})
	if err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(spec.UserID.String()))
}

func (r *vanityRoleRepository) List(ctx context.Context, tx ...*sql.Tx) ([]VanityRoleRow, error) {
	return r.dao.List(ctx, tx...)
}

func (r *vanityRoleRepository) GetByID(ctx context.Context, id string, tx ...*sql.Tx) (*VanityRoleRow, error) {
	return r.dao.GetByID(ctx, id, tx...)
}

func (r *vanityRoleRepository) Create(ctx context.Context, id, label, color string, sortOrder int, tx ...*sql.Tx) error {
	return r.dao.Create(ctx, id, label, color, sortOrder, tx...)
}

func (r *vanityRoleRepository) Update(ctx context.Context, id, label, color string, sortOrder int, tx ...*sql.Tx) error {
	return r.dao.Update(ctx, id, label, color, sortOrder, tx...)
}

func (r *vanityRoleRepository) Delete(ctx context.Context, id string, tx ...*sql.Tx) error {
	if err := r.dao.Delete(ctx, id, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.VanityRolePermissions.Key())
}

func (r *vanityRoleRepository) AssignToUser(ctx context.Context, userID uuid.UUID, roleID string, tx ...*sql.Tx) error {
	if err := r.dao.AssignToUser(ctx, userID, roleID, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(userID.String()))
}

func (r *vanityRoleRepository) UnassignFromUser(ctx context.Context, userID uuid.UUID, roleID string, tx ...*sql.Tx) error {
	if err := r.dao.UnassignFromUser(ctx, userID, roleID, tx...); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(userID.String()))
}

func (r *vanityRoleRepository) GetUsersForRole(ctx context.Context, roleID string, search string, limit, offset int, tx ...*sql.Tx) ([]VanityRoleUserRow, int, error) {
	return r.dao.GetUsersForRole(ctx, roleID, search, limit, offset, tx...)
}

func (r *vanityRoleRepository) GetRolesForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]VanityRoleRow, error) {
	return r.dao.GetRolesForUser(ctx, userID, tx...)
}

func (r *vanityRoleRepository) GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]VanityRoleRow, error) {
	return r.dao.GetRolesForUsersBatch(ctx, userIDs, tx...)
}

func (r *vanityRoleRepository) GetAllAssignments(ctx context.Context, tx ...*sql.Tx) (map[string][]string, error) {
	key := cache.VanityAssignments.Key()

	if v, err := cache.Get[map[string][]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetAllAssignments(ctx, tx...)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.VanityAssignments.TTL)
	return v, nil
}
