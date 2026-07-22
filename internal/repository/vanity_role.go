package repository

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/cache"

	"github.com/google/uuid"
)

type (
	VanityRoleRepository interface {
		List(ctx context.Context) ([]VanityRoleRow, error)
		GetByID(ctx context.Context, id string) (*VanityRoleRow, error)
		Create(ctx context.Context, id, label, color string, sortOrder int) error
		Update(ctx context.Context, id, label, color string, sortOrder int) error
		Delete(ctx context.Context, id string) error
		AssignToUser(ctx context.Context, userID uuid.UUID, roleID string) error
		UnassignFromUser(ctx context.Context, userID uuid.UUID, roleID string) error
		GetUsersForRole(ctx context.Context, roleID string, search string, limit, offset int) ([]VanityRoleUserRow, int, error)
		GetRolesForUser(ctx context.Context, userID uuid.UUID) ([]VanityRoleRow, error)
		GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]VanityRoleRow, error)
		GetAllAssignments(ctx context.Context) (map[string][]string, error)
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
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
		args[i] = id
	}
	return " AND id NOT IN (" + strings.Join(placeholders, ", ") + ")", args
}

type vanityRoleRepository struct {
	dao   VanityRoleRepository
	cache *cache.Manager
}

func NewVanityRoleRepo(dao VanityRoleRepository, c *cache.Manager) VanityRoleRepository {
	return &vanityRoleRepository{dao: dao, cache: c}
}

func (r *vanityRoleRepository) List(ctx context.Context) ([]VanityRoleRow, error) {
	return r.dao.List(ctx)
}

func (r *vanityRoleRepository) GetByID(ctx context.Context, id string) (*VanityRoleRow, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *vanityRoleRepository) Create(ctx context.Context, id, label, color string, sortOrder int) error {
	return r.dao.Create(ctx, id, label, color, sortOrder)
}

func (r *vanityRoleRepository) Update(ctx context.Context, id, label, color string, sortOrder int) error {
	return r.dao.Update(ctx, id, label, color, sortOrder)
}

func (r *vanityRoleRepository) Delete(ctx context.Context, id string) error {
	if err := r.dao.Delete(ctx, id); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key())
}

func (r *vanityRoleRepository) AssignToUser(ctx context.Context, userID uuid.UUID, roleID string) error {
	if err := r.dao.AssignToUser(ctx, userID, roleID); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key())
}

func (r *vanityRoleRepository) UnassignFromUser(ctx context.Context, userID uuid.UUID, roleID string) error {
	if err := r.dao.UnassignFromUser(ctx, userID, roleID); err != nil {
		return err
	}

	return r.cache.Del(ctx, cache.VanityAssignments.Key())
}

func (r *vanityRoleRepository) GetUsersForRole(ctx context.Context, roleID string, search string, limit, offset int) ([]VanityRoleUserRow, int, error) {
	return r.dao.GetUsersForRole(ctx, roleID, search, limit, offset)
}

func (r *vanityRoleRepository) GetRolesForUser(ctx context.Context, userID uuid.UUID) ([]VanityRoleRow, error) {
	return r.dao.GetRolesForUser(ctx, userID)
}

func (r *vanityRoleRepository) GetRolesForUsersBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]VanityRoleRow, error) {
	return r.dao.GetRolesForUsersBatch(ctx, userIDs)
}

func (r *vanityRoleRepository) GetAllAssignments(ctx context.Context) (map[string][]string, error) {
	key := cache.VanityAssignments.Key()

	if v, err := cache.Get[map[string][]string](ctx, r.cache, key); err == nil {
		return v, nil
	}

	v, err := r.dao.GetAllAssignments(ctx)
	if err != nil {
		return nil, err
	}

	_ = cache.Set(ctx, r.cache, key, v, cache.VanityAssignments.TTL)
	return v, nil
}
