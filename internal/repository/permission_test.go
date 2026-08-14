package repository

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/cache"

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
	dao := NewMockPermissionRepository(t)

	return NewPermissionRepo(dao, cache.NewManagerWithClient(client)), dao, client
}

func expectDel(t *testing.T, client *valkeymock.Client, key string) *gomock.Call {
	t.Helper()

	return client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			assert.Equal(t, []string{"DEL", key}, cmd.Commands())

			return valkeymock.Result(valkeymock.ValkeyInt64(1))
		}).
		Times(1)
}

func TestSetRolePermissions_InvalidatesRolePermissionCache(t *testing.T) {
	// given
	repo, dao, client := newCachedPermissionRepo(t)
	dao.EXPECT().SetRolePermissions(mock.Anything, "moderator", []string{"ban_user"}).Return(nil)
	expectDel(t, client, cache.RolePermissions.Key())

	// when
	err := repo.SetRolePermissions(context.Background(), "moderator", []string{"ban_user"})

	// then
	require.NoError(t, err)
}

func TestSetRolePermissions_DaoErrorSkipsInvalidation(t *testing.T) {
	// given
	repo, dao, client := newCachedPermissionRepo(t)
	dao.EXPECT().SetRolePermissions(mock.Anything, "moderator", []string(nil)).Return(errors.New("db down"))
	client.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := repo.SetRolePermissions(context.Background(), "moderator", nil)

	// then
	require.Error(t, err)
}

func TestSetVanityRolePermissions_InvalidatesVanityPermissionCache(t *testing.T) {
	// given
	repo, dao, client := newCachedPermissionRepo(t)
	dao.EXPECT().SetVanityRolePermissions(mock.Anything, "vanity-a", []string{"use_chatbot"}).Return(nil)
	expectDel(t, client, cache.VanityRolePermissions.Key())

	// when
	err := repo.SetVanityRolePermissions(context.Background(), "vanity-a", []string{"use_chatbot"})

	// then
	require.NoError(t, err)
}

func TestAssignToUser_InvalidatesVanityAssignmentAndUserKeys(t *testing.T) {
	// given
	client := valkeymock.NewClient(gomock.NewController(t))
	dao := NewMockVanityRoleDAO(t)
	repo := NewVanityRoleRepo(nil, dao, cache.NewManagerWithClient(client))
	userID := uuid.New()
	dao.EXPECT().AssignToUser(mock.Anything, userID, "vanity-a").Return(nil)

	var commands []string
	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			commands = cmd.Commands()

			return valkeymock.Result(valkeymock.ValkeyInt64(2))
		}).
		Times(1)

	// when
	err := repo.AssignToUser(context.Background(), userID, "vanity-a")

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"DEL", cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(userID.String())}, commands)
}

func TestUnassignFromUser_InvalidatesVanityAssignmentAndUserKeys(t *testing.T) {
	// given
	client := valkeymock.NewClient(gomock.NewController(t))
	dao := NewMockVanityRoleDAO(t)
	repo := NewVanityRoleRepo(nil, dao, cache.NewManagerWithClient(client))
	userID := uuid.New()
	dao.EXPECT().UnassignFromUser(mock.Anything, userID, "vanity-a").Return(nil)

	var commands []string
	client.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cmd valkey.Completed) valkey.ValkeyResult {
			commands = cmd.Commands()

			return valkeymock.Result(valkeymock.ValkeyInt64(2))
		}).
		Times(1)

	// when
	err := repo.UnassignFromUser(context.Background(), userID, "vanity-a")

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"DEL", cache.VanityAssignments.Key(), cache.UserVanityRoleIDs.Key(userID.String())}, commands)
}

func TestPermissionCacheKeys_DoNotCollide(t *testing.T) {
	// given
	userA := uuid.New()
	userB := uuid.New()

	// when
	keys := []string{
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
		_, dupe := seen[key]
		assert.False(t, dupe, "cache key %q is not unique", key)
		seen[key] = struct{}{}
	}
}

func TestGetRolePermissions_CachesDaoResult(t *testing.T) {
	// given
	repo, dao, client := newCachedPermissionRepo(t)
	dao.EXPECT().GetRolePermissions(mock.Anything).Return(map[string][]string{"moderator": {"ban_user"}}, nil)

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
	assert.Equal(t, []string{"GET", cache.RolePermissions.Key()}, commands[0])
	assert.Equal(t, "SET", commands[1][0])
	assert.Equal(t, cache.RolePermissions.Key(), commands[1][1])
}
