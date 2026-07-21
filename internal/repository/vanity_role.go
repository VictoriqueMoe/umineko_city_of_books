package repository

import (
	"context"
	"fmt"
	"strings"

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
