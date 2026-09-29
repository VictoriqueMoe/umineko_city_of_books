package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatDAO_SetMuted_MutesAndUnmutes(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID, daotest.WithRoomMembers(a.ID, b.ID))

	// when
	err := repos.Chat.SetMuted(ctx, spec.ChatMemberMuteUpdate{RoomID: roomID, UserID: a.ID, Muted: true})

	// then
	require.NoError(t, err)

	muted, err := repos.Chat.IsMuted(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: a.ID})
	require.NoError(t, err)
	assert.True(t, muted)

	unmuted, err := repos.Chat.GetRoomMembersUnmuted(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{b.ID}, unmuted)

	ownerMuted, err := repos.Chat.IsMuted(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: owner.ID})
	require.NoError(t, err)
	assert.False(t, ownerMuted)

	// and when unmuting
	require.NoError(t, repos.Chat.SetMuted(ctx, spec.ChatMemberMuteUpdate{RoomID: roomID, UserID: a.ID, Muted: false}))

	// then
	mutedAfterUnmute, err := repos.Chat.IsMuted(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: a.ID})
	require.NoError(t, err)
	assert.False(t, mutedAfterUnmute)

	unmutedAfterUnmute, err := repos.Chat.GetRoomMembersUnmuted(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, unmutedAfterUnmute)
}

func TestChatDAO_SetMemberNicknameWithLock_LocksAndUnlocks(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID)
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: owner.ID, Role: "host", Ghost: false}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: member.ID, Role: "member", Ghost: false}))

	// when
	err := repos.Chat.SetMemberNicknameWithLock(ctx, spec.ChatMemberNicknameUpdate{RoomID: roomID, UserID: member.ID, Nickname: "Forced", Locked: true})

	// then
	require.NoError(t, err)

	memberLocked, err := repos.Chat.IsMemberNicknameLocked(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: member.ID})
	require.NoError(t, err)
	assert.True(t, memberLocked)

	ownerLocked, err := repos.Chat.IsMemberNicknameLocked(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: owner.ID})
	require.NoError(t, err)
	assert.False(t, ownerLocked)

	strangerLocked, err := repos.Chat.IsMemberNicknameLocked(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: uuid.New()})
	require.NoError(t, err)
	assert.False(t, strangerLocked)

	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, detailed, 2)
	assert.Equal(t, owner.ID, detailed[0].UserID)
	assert.Equal(t, "", detailed[0].Nickname)
	assert.False(t, detailed[0].NicknameLocked)
	assert.Equal(t, member.ID, detailed[1].UserID)
	assert.Equal(t, "Forced", detailed[1].Nickname)
	assert.True(t, detailed[1].NicknameLocked)

	// and when unlocking
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, spec.ChatMemberNicknameUpdate{RoomID: roomID, UserID: member.ID, Nickname: "", Locked: false}))

	// then
	lockedAfterUnlock, err := repos.Chat.IsMemberNicknameLocked(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: member.ID})
	require.NoError(t, err)
	assert.False(t, lockedAfterUnlock)

	detailedAfterUnlock, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, detailedAfterUnlock, 2)
	assert.Equal(t, member.ID, detailedAfterUnlock[1].UserID)
	assert.Equal(t, "", detailedAfterUnlock[1].Nickname)
	assert.False(t, detailedAfterUnlock[1].NicknameLocked)
}

func TestChatDAO_SetMemberTimeout_SetsAndClears(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID)
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: owner.ID, Role: "host", Ghost: false}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: member.ID, Role: "member", Ghost: false}))

	// when
	err := repos.Chat.SetMemberTimeout(ctx, spec.ChatMemberTimeout{RoomID: roomID, UserID: member.ID, Until: "2099-01-01 00:00:00", ByStaff: true})

	// then
	require.NoError(t, err)

	active, until, byStaff, err := repos.Chat.GetMemberTimeoutState(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: member.ID})
	require.NoError(t, err)
	assert.True(t, active)
	assert.Equal(t, "2099-01-01T00:00:00Z", until)
	assert.True(t, byStaff)

	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, detailed, 2)
	assert.Equal(t, "", detailed[0].TimeoutUntil)
	assert.False(t, detailed[0].TimeoutByStaff)
	assert.Equal(t, member.ID, detailed[1].UserID)
	assert.Equal(t, "2099-01-01T00:00:00Z", detailed[1].TimeoutUntil)
	assert.True(t, detailed[1].TimeoutByStaff)

	// and when clearing
	require.NoError(t, repos.Chat.ClearMemberTimeout(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: member.ID}))

	// then
	activeAfterClear, untilAfterClear, byStaffAfterClear, err := repos.Chat.GetMemberTimeoutState(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: member.ID})
	require.NoError(t, err)
	assert.False(t, activeAfterClear)
	assert.Equal(t, "", untilAfterClear)
	assert.False(t, byStaffAfterClear)
}
