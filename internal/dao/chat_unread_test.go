package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatDAO_TouchRoomActivity(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID, daotest.WithRoomMembers(owner.ID))

	// when
	err := repos.Chat.TouchRoomActivity(ctx, roomID)

	// then
	require.NoError(t, err)

	row, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: owner.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastMessageAt.Valid)
}

func TestChatDAO_CountUnreadRoomsForUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	viewer := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	dmID := daotest.CreateDMRoom(t, repos, viewer.ID, other.ID)
	groupID := daotest.CreateChatRoom(t, repos, other.ID, daotest.WithRoomMembers(viewer.ID, other.ID))
	daotest.SendChatMessage(t, repos, dmID, other.ID, "hi")
	daotest.SendChatMessage(t, repos, groupID, other.ID, "hi")

	// when
	before, err := repos.Chat.CountUnreadRoomsForUser(ctx, viewer.ID)
	require.NoError(t, err)

	require.NoError(t, repos.Chat.MarkRoomRead(ctx, spec.ChatMemberRef{RoomID: dmID, UserID: viewer.ID}))

	after, err := repos.Chat.CountUnreadRoomsForUser(ctx, viewer.ID)
	require.NoError(t, err)

	// then
	assert.Equal(t, 1, before)
	assert.Equal(t, 0, after)

	row, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: dmID, ViewerID: viewer.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastReadAt.Valid)
}
