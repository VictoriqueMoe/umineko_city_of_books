package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatDAO_AddMember_IsIdempotentAndSeenByEveryMemberLookup(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	outsider := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID)

	// when
	require.NoError(t, repos.Chat.AddMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: a.ID}))
	require.NoError(t, repos.Chat.AddMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: b.ID}))
	require.NoError(t, repos.Chat.AddMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: a.ID}))

	// then
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	members, err := repos.Chat.GetRoomMembers(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, members)

	isMember, err := repos.Chat.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: a.ID})
	require.NoError(t, err)
	assert.True(t, isMember)

	outsiderIsMember, err := repos.Chat.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: outsider.ID})
	require.NoError(t, err)
	assert.False(t, outsiderIsMember)

	outsiderRole, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: outsider.ID})
	require.NoError(t, err)
	assert.Equal(t, "", outsiderRole)
}

func TestChatDAO_RemoveMember_SoftDeletesSoARejoinKeepsTheNicknameAndAvatar(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID)
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: joiner.ID, Role: "member", Ghost: false}))
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, spec.ChatMemberNicknameUpdate{RoomID: roomID, UserID: joiner.ID, Nickname: "Beato", Locked: true}))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, spec.ChatMemberAvatarUpdate{RoomID: roomID, UserID: joiner.ID, AvatarURL: "/custom.png"}))

	// when
	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: joiner.ID}))

	stillMember, err := repos.Chat.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: joiner.ID})
	require.NoError(t, err)

	var softDeleted int
	require.NoError(t, repos.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NOT NULL`,
		roomID, joiner.ID,
	).Scan(&softDeleted))

	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: joiner.ID, Role: "member", Ghost: false}))

	// then
	assert.False(t, stillMember)
	assert.Equal(t, 1, softDeleted)

	rejoined, err := repos.Chat.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: joiner.ID})
	require.NoError(t, err)
	assert.True(t, rejoined)

	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, detailed, 1)
	assert.Equal(t, joiner.ID, detailed[0].UserID)
	assert.Equal(t, "Beato", detailed[0].Nickname)
	assert.True(t, detailed[0].NicknameLocked)
	assert.Equal(t, "/custom.png", detailed[0].MemberAvatarURL)
}

func TestChatDAO_GetRoomMembersDetailed(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, owner.ID)
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: member.ID, Role: "member", Ghost: false}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: roomID, UserID: owner.ID, Role: "host", Ghost: false}))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, spec.ChatMemberAvatarUpdate{RoomID: roomID, UserID: member.ID, AvatarURL: "/uploads/chat-avatars/first.png"}))

	// when
	require.NoError(t, repos.Chat.SetMemberNickname(ctx, spec.ChatMemberNicknameUpdate{RoomID: roomID, UserID: member.ID, Nickname: "Beato"}))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, spec.ChatMemberAvatarUpdate{RoomID: roomID, UserID: member.ID, AvatarURL: "/uploads/chat-avatars/second.png"}))

	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, detailed, 2)
	assert.Equal(t, owner.ID, detailed[0].UserID)
	assert.Equal(t, "host", detailed[0].Role)
	assert.Equal(t, "", detailed[0].Nickname)
	assert.Equal(t, "", detailed[0].MemberAvatarURL)
	assert.Equal(t, member.ID, detailed[1].UserID)
	assert.Equal(t, "member", detailed[1].Role)
	assert.Equal(t, "Beato", detailed[1].Nickname)
	assert.Equal(t, "/uploads/chat-avatars/second.png", detailed[1].MemberAvatarURL)
}

func TestChatDAO_SyncSystemRoomMembership_JoinsLeavesAndUpgrades(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	joinRoomID := uuid.New()
	leaveRoomID := uuid.New()
	upgradeRoomID := uuid.New()
	require.NoError(t, repos.Chat.CreateSystemRooms(ctx, []spec.NewChatSystemRoom{
		{ID: joinRoomID, Name: "Mods", SystemKind: "mods", CreatedBy: user.ID},
		{ID: leaveRoomID, Name: "Admins", SystemKind: "admins", CreatedBy: user.ID},
		{ID: upgradeRoomID, Name: "Announcements", SystemKind: "announcements", CreatedBy: user.ID},
	}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: leaveRoomID, UserID: user.ID, Role: "member", Ghost: false}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: upgradeRoomID, UserID: user.ID, Role: "member", Ghost: false}))

	// when
	changes, err := repos.Chat.SyncSystemRoomMembership(ctx, []spec.SystemRoomMembership{
		{RoomID: joinRoomID, UserID: user.ID, ShouldBeMember: true, DesiredRole: "host"},
		{RoomID: leaveRoomID, UserID: user.ID, ShouldBeMember: false, DesiredRole: "host"},
		{RoomID: upgradeRoomID, UserID: user.ID, ShouldBeMember: true, DesiredRole: "host"},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, []model.SystemRoomMembershipChange{
		{RoomID: joinRoomID, Joined: true},
		{RoomID: leaveRoomID, Left: true},
	}, changes, "a role change is not a join or a leave, so the hub is left alone")

	joined, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: joinRoomID, UserID: user.ID})
	require.NoError(t, err)
	assert.Equal(t, "host", joined)

	left, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: leaveRoomID, UserID: user.ID})
	require.NoError(t, err)
	assert.Equal(t, "", left)

	upgraded, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: upgradeRoomID, UserID: user.ID})
	require.NoError(t, err)
	assert.Equal(t, "host", upgraded)
}

func TestChatDAO_AddMemberWithSystemMessage(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		ghost       bool
		wantMessage bool
	}{
		{name: "announces the join", body: "Test User joined the room.", ghost: false, wantMessage: true},
		{name: "skips the message when the body is empty", body: "", ghost: true, wantMessage: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			host := daotest.CreateUser(t, repos)
			joiner := daotest.CreateUser(t, repos)
			roomID := daotest.CreateChatRoom(t, repos, host.ID)

			// when
			msg, err := repos.Chat.AddMemberWithSystemMessage(ctx, spec.ChatMemberJoinAnnouncement{
				Member:  spec.NewChatRoomMember{RoomID: roomID, UserID: joiner.ID, Role: "member", Ghost: tc.ghost},
				Message: spec.NewChatMessage{RoomID: roomID, SenderID: joiner.ID, Body: tc.body, IsSystem: true},
			})

			// then
			require.NoError(t, err)

			role, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: joiner.ID})
			require.NoError(t, err)
			assert.Equal(t, "member", role)

			if !tc.wantMessage {
				assert.Nil(t, msg, "a ghost join is silent")
				return
			}

			require.NotNil(t, msg)
			assert.Equal(t, tc.body, msg.Body)
			assert.True(t, msg.IsSystem)
		})
	}
}

func TestChatDAO_SoftLeaveHidesTheLeaversHistoryAndKeepsTheOthers(t *testing.T) {
	// given a pair with history, backdated so a stepping container clock cannot reorder it
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	stayer := daotest.CreateUser(t, repos)
	leaver := daotest.CreateUser(t, repos)
	roomID := daotest.CreateDMRoom(t, repos, stayer.ID, leaver.ID)
	daotest.BackdateChatRoomJoins(t, repos, roomID, "2024-01-01 00:00:00")
	oldID := daotest.SendChatMessage(t, repos, roomID, stayer.ID, "before the leave")
	daotest.BackdateChatMessage(t, repos, oldID, "2024-01-01 01:00:00")

	// when the leaver leaves and later reopens the same pair
	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: leaver.ID}))

	remaining, err := repos.Chat.CountRoomMembers(ctx, roomID)
	require.NoError(t, err)

	reopened, err := repos.Chat.CreateDMRoomAtomic(ctx, spec.ChatDMPair{UserA: leaver.ID, UserB: stayer.ID})
	require.NoError(t, err)

	// then the pair is reused, the leaver's thread is blank and the other party keeps everything
	assert.Equal(t, 1, remaining, "one member left is what stops the service hard-deleting the pair")
	assert.Equal(t, roomID, reopened.ID)

	leaverView, leaverTotal, err := repos.Chat.GetMessagesForViewer(ctx, spec.ChatMessagePage{RoomID: roomID, ViewerID: leaver.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Empty(t, leaverView, "the history clip is the load-bearing half of soft-leave")
	assert.Equal(t, 0, leaverTotal)

	stayerView, stayerTotal, err := repos.Chat.GetMessagesForViewer(ctx, spec.ChatMessagePage{RoomID: roomID, ViewerID: stayer.ID, Limit: 20, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, stayerTotal)
	require.Len(t, stayerView, 1)
	assert.Equal(t, "before the leave", stayerView[0].Body)
}

func TestChatDAO_BothPartiesLeavingEmptiesThePair(t *testing.T) {
	// given a pair with history
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	roomID := daotest.CreateDMRoom(t, repos, a.ID, b.ID)
	daotest.SendChatMessage(t, repos, roomID, a.ID, "hi")

	// when both sides leave
	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: a.ID}))
	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: b.ID}))

	// then the count the service hard-deletes on reaches zero
	remaining, err := repos.Chat.CountRoomMembers(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}
