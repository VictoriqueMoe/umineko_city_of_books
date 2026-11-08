package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
	valkeymock "github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func newCachedPermissionRepo(t *testing.T) (PermissionRepository, *MockPermissionRepository, *valkeymock.Client) {
	t.Helper()

	client := valkeymock.NewClient(gomock.NewController(t))
	permissionDAO := NewMockPermissionRepository(t)

	return NewPermissionRepo(nil, permissionDAO, cache.NewManager(engines.NewValkeyWithClient(client))), permissionDAO, client
}

func expectDel(t *testing.T, client *valkeymock.Client, key cache.Key) *gomock.Call {
	t.Helper()

	return client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			assert.Equal(t, []string{"DEL", key.String()}, cmd.Commands())

			return valkeymock.Result(valkeymock.ValkeyInt64(1))
		}).
		Times(1)
}

func TestSetRolePermissions_InvalidatesRolePermissionCache(t *testing.T) {
	// given
	repo, permissionDAO, client := newCachedPermissionRepo(t)
	permissionDAO.EXPECT().SetRolePermissions(mock.Anything, spec.RolePermissionsUpdate{RoleName: "moderator", Permissions: []string{"ban_user"}}).Return(nil)
	expectDel(t, client, cache.RolePermissions.Key())

	// when
	err := repo.SetRolePermissions(context.Background(), spec.RolePermissionsUpdate{RoleName: "moderator", Permissions: []string{"ban_user"}})

	// then
	require.NoError(t, err)
}

func TestSetRolePermissions_DaoErrorSkipsInvalidation(t *testing.T) {
	// given
	repo, permissionDAO, client := newCachedPermissionRepo(t)
	permissionDAO.EXPECT().SetRolePermissions(mock.Anything, spec.RolePermissionsUpdate{RoleName: "moderator", Permissions: nil}).Return(errors.New("db down"))
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.SetRolePermissions(context.Background(), spec.RolePermissionsUpdate{RoleName: "moderator", Permissions: nil})

	// then
	require.Error(t, err)
}

func TestSetVanityRolePermissions_InvalidatesVanityPermissionCache(t *testing.T) {
	// given
	repo, permissionDAO, client := newCachedPermissionRepo(t)
	permissionDAO.EXPECT().SetVanityRolePermissions(mock.Anything, spec.VanityRolePermissionsUpdate{VanityRoleID: "vanity-a", Permissions: []string{"use_chatbot"}}).Return(nil)
	expectDel(t, client, cache.VanityRolePermissions.Key())

	// when
	err := repo.SetVanityRolePermissions(context.Background(), spec.VanityRolePermissionsUpdate{VanityRoleID: "vanity-a", Permissions: []string{"use_chatbot"}})

	// then
	require.NoError(t, err)
}

func TestPermissionRepository_VanityAssignmentWritesInvalidateAssignmentAndUserKeys(t *testing.T) {
	userID := uuid.New()
	assignment := spec.VanityRoleAssignment{UserID: userID, RoleID: "vanity-a"}
	tx := new(sql.Tx)

	tests := []struct {
		name  string
		write func(repo PermissionRepository, permissionDAO *MockPermissionRepository) error
	}{
		{
			name: "assigning a vanity role",
			write: func(repo PermissionRepository, permissionDAO *MockPermissionRepository) error {
				permissionDAO.EXPECT().AssignVanityRoleToUser(mock.Anything, assignment).Return(nil).Once()

				return repo.AssignVanityRoleToUser(context.Background(), assignment)
			},
		},
		{
			name: "unassigning a vanity role",
			write: func(repo PermissionRepository, permissionDAO *MockPermissionRepository) error {
				permissionDAO.EXPECT().UnassignVanityRoleFromUser(mock.Anything, assignment).Return(nil).Once()

				return repo.UnassignVanityRoleFromUser(context.Background(), assignment)
			},
		},
		{
			name: "moving a user between vanity roles",
			write: func(repo PermissionRepository, permissionDAO *MockPermissionRepository) error {
				permissionDAO.EXPECT().AssignVanityRoleToUser(mock.Anything, spec.VanityRoleAssignment{UserID: userID, RoleID: "vanity-b"}, []*sql.Tx{tx}).Return(nil).Once()
				permissionDAO.EXPECT().UnassignVanityRoleFromUser(mock.Anything, assignment, []*sql.Tx{tx}).Return(nil).Once()

				return repo.MoveUserVanityRole(context.Background(), spec.VanityRoleMove{UserID: userID, FromRoleID: "vanity-a", ToRoleID: "vanity-b"}, tx)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repo, permissionDAO, client := newCachedPermissionRepo(t)

			var commands []string
			client.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
					commands = cmd.Commands()

					return valkeymock.Result(valkeymock.ValkeyInt64(2))
				}).
				Times(1)

			// when
			err := tc.write(repo, permissionDAO)

			// then
			require.NoError(t, err)
			assert.Equal(t, []string{"DEL", cache.VanityAssignments.Key().String(), cache.UserVanityRoleIDs.Key(userID.String()).String()}, commands)
		})
	}
}

func TestPermissionRepository_FailedVanityAssignmentKeepsTheCache(t *testing.T) {
	// given
	repo, permissionDAO, client := newCachedPermissionRepo(t)
	assignment := spec.VanityRoleAssignment{UserID: uuid.New(), RoleID: "vanity-a"}
	permissionDAO.EXPECT().AssignVanityRoleToUser(mock.Anything, assignment).Return(errors.New("db down")).Once()
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.AssignVanityRoleToUser(context.Background(), assignment)

	// then
	require.Error(t, err)
}

func TestPermissionRepository_FailedCacheDeleteDoesNotFailACommittedAssignment(t *testing.T) {
	// given an assignment that commits while the cache is unreachable
	repo, permissionDAO, client := newCachedPermissionRepo(t)
	assignment := spec.VanityRoleAssignment{UserID: uuid.New(), RoleID: "vanity-a"}
	permissionDAO.EXPECT().AssignVanityRoleToUser(mock.Anything, assignment).Return(nil).Once()
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(valkeymock.ErrorResult(errors.New("valkey down"))).Times(1)

	// when
	err := repo.AssignVanityRoleToUser(context.Background(), assignment)

	// then the committed assignment is not reported as failed
	require.NoError(t, err)
}

func TestPermissionCacheKeys_DoNotCollide(t *testing.T) {
	// given
	userA := uuid.New()
	userB := uuid.New()

	// when
	keys := []cache.Key{
		cache.RolePermissions.Key(),
		cache.VanityRolePermissions.Key(),
		cache.UserVanityRoleIDs.Key(userA.String()),
		cache.UserVanityRoleIDs.Key(userB.String()),
		cache.VanityAssignments.Key(),
		cache.UserRole.Key(userA.String()),
	}

	// then
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		_, dupe := seen[key.String()]
		assert.False(t, dupe, "cache key %q is not unique", key)
		seen[key.String()] = struct{}{}
	}
}

func TestGetRolePermissions_CachesDaoResult(t *testing.T) {
	// given
	repo, permissionDAO, client := newCachedPermissionRepo(t)
	permissionDAO.EXPECT().GetRolePermissions(mock.Anything).Return(map[string][]string{"moderator": {"ban_user"}}, nil)

	var commands [][]string
	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			commands = append(commands, cmd.Commands())

			if cmd.Commands()[0] == "GET" {
				return valkeymock.ErrorResult(valkey.Nil)
			}

			return valkeymock.Result(valkeymock.ValkeyString("OK"))
		}).
		Times(2)

	// when
	got, err := repo.GetRolePermissions(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"moderator": {"ban_user"}}, got)
	require.Len(t, commands, 2)
	assert.Equal(t, []string{"GET", cache.RolePermissions.Key().String()}, commands[0])
	assert.Equal(t, "SET", commands[1][0])
	assert.Equal(t, cache.RolePermissions.Key().String(), commands[1][1])
}
