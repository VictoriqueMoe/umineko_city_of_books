package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatRoomBanRepository_BanIsBannedListAndUnban(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos, repotest.WithDisplayName("Owner"))
	target := repotest.CreateUser(t, repos, repotest.WithDisplayName("Target"))
	mod := repotest.CreateUser(t, repos, repotest.WithDisplayName("Mod"))

	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "Room", "", "group", true, false, owner.ID))

	banned, err := repos.ChatRoomBan.IsBanned(ctx, roomID, target.ID)
	require.NoError(t, err)
	assert.False(t, banned)

	// when
	require.NoError(t, repos.ChatRoomBan.Ban(ctx, roomID, target.ID, &mod.ID, "trolling"))

	// then
	banned, err = repos.ChatRoomBan.IsBanned(ctx, roomID, target.ID)
	require.NoError(t, err)
	assert.True(t, banned)

	rows, err := repos.ChatRoomBan.ListForRoom(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, target.ID, rows[0].UserID)
	assert.Equal(t, "Target", rows[0].DisplayName)
	require.NotNil(t, rows[0].BannedByID)
	assert.Equal(t, mod.ID, *rows[0].BannedByID)
	assert.Equal(t, "Mod", rows[0].BannedByDisplay)
	assert.Equal(t, "trolling", rows[0].Reason)

	// when (unban)
	require.NoError(t, repos.ChatRoomBan.Unban(ctx, roomID, target.ID))

	// then
	banned, err = repos.ChatRoomBan.IsBanned(ctx, roomID, target.ID)
	require.NoError(t, err)
	assert.False(t, banned)
	rows, err = repos.ChatRoomBan.ListForRoom(ctx, roomID)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestChatRoomBanRepository_Ban_UpsertsExisting(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)

	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "Room", "", "group", true, false, owner.ID))
	require.NoError(t, repos.ChatRoomBan.Ban(ctx, roomID, target.ID, nil, "first reason"))

	// when
	require.NoError(t, repos.ChatRoomBan.Ban(ctx, roomID, target.ID, &owner.ID, "updated reason"))

	// then
	rows, err := repos.ChatRoomBan.ListForRoom(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "updated reason", rows[0].Reason)
	require.NotNil(t, rows[0].BannedByID)
	assert.Equal(t, owner.ID, *rows[0].BannedByID)
}

func TestChatRoomBanRepository_BannedRoomIDsForUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)

	roomA := uuid.New()
	roomB := uuid.New()
	roomC := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomA, "A", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomB, "B", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomC, "C", "", "group", true, false, owner.ID))

	require.NoError(t, repos.ChatRoomBan.Ban(ctx, roomA, target.ID, nil, ""))
	require.NoError(t, repos.ChatRoomBan.Ban(ctx, roomC, target.ID, nil, ""))

	// when
	ids, err := repos.ChatRoomBan.BannedRoomIDsForUser(ctx, target.ID)

	// then
	require.NoError(t, err)
	require.Len(t, ids, 2)
	got := map[uuid.UUID]bool{}
	for i := 0; i < len(ids); i++ {
		got[ids[i]] = true
	}
	assert.True(t, got[roomA])
	assert.False(t, got[roomB])
	assert.True(t, got[roomC])
}

func TestChatRoomBanRepository_Unban_NonExistentIsNoop(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))

	// when
	err := repos.ChatRoomBan.Unban(ctx, roomID, target.ID)

	// then
	require.NoError(t, err)
}
