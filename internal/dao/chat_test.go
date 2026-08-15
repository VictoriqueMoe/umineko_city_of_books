package dao_test

import (
	"context"
	"sort"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatDAO_CreateRoom_Group(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Room", Description: "desc", Type: "group", IsPublic: true, IsRP: false, CreatedBy: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, room.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Room", row.Name)
	assert.Equal(t, "desc", row.Description)
	assert.Equal(t, dto.RoomTypeGroup, row.Type)
	assert.True(t, row.IsPublic)
	assert.False(t, row.IsRP)
	assert.Equal(t, user.ID, row.CreatedBy)
}

func TestChatDAO_CreateRoom_RPFlag(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "RP", Description: "", Type: "group", IsPublic: false, IsRP: true, CreatedBy: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, room.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsRP)
	assert.False(t, row.IsPublic)
}

func TestChatDAO_CreateSystemRoom(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	_, err := repos.Chat.CreateSystemRoom(ctx, repository.NewChatSystemRoom{ID: roomID, Name: "System", Description: "system room", SystemKind: "announcements", CreatedBy: user.ID})

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsSystem)
	assert.Equal(t, "announcements", row.SystemKind)
	assert.Equal(t, dto.RoomTypeGroup, row.Type)
}

func TestChatDAO_GetSystemRoomID_Found(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	_, err := repos.Chat.CreateSystemRoom(ctx, repository.NewChatSystemRoom{ID: roomID, Name: "Sys", Description: "", SystemKind: "announcements", CreatedBy: user.ID})
	require.NoError(t, err)

	// when
	got, err := repos.Chat.GetSystemRoomID(ctx, "announcements")

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
}

func TestChatDAO_GetSystemRoomID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetSystemRoomID(ctx, "missing")

	// then
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestChatDAO_CreateDMRoomAtomic_New(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)

	// when
	gotRow, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)

	// then
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, gotRow.ID)
	members, err := repos.Chat.GetRoomMembers(ctx, gotRow.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, members)
}

func TestChatDAO_CreateDMRoomAtomic_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	first, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)

	// when
	gotRow, err := repos.Chat.CreateDMRoomAtomic(ctx, b.ID, a.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, first.ID, gotRow.ID)
}

func TestChatDAO_AddMember(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	err = repos.Chat.AddMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	isMember, err := repos.Chat.IsMember(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.True(t, isMember)
}

func TestChatDAO_AddMember_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, joiner.ID))

	// when
	err = repos.Chat.AddMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestChatDAO_AddMemberWithRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	err = repos.Chat.AddMemberWithRole(ctx, roomID, joiner.ID, "host", false)

	// then
	require.NoError(t, err)
	role, err := repos.Chat.GetMemberRole(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", role)
}

func TestChatDAO_SetMemberRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, joiner.ID, "member", false))

	// when
	err = repos.Chat.SetMemberRole(ctx, roomID, joiner.ID, "host")

	// then
	require.NoError(t, err)
	role, err := repos.Chat.GetMemberRole(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", role)
}

func TestChatDAO_GetMemberRole_NotMember(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	role, err := repos.Chat.GetMemberRole(ctx, roomID, other.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "", role)
}

func TestChatDAO_RemoveMember(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, joiner.ID))

	// when
	err = repos.Chat.RemoveMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	isMember, err := repos.Chat.IsMember(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestChatDAO_CountRoomMembers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, a.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, b.ID))

	// when
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestChatDAO_CountRoomMembers_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatDAO_DeleteRoom(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	err = repos.Chat.DeleteRoom(ctx, roomID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatDAO_GetRoomByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	row, err := repos.Chat.GetRoomByID(ctx, uuid.New(), user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatDAO_GetRoomByID_NonMember(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	row, err := repos.Chat.GetRoomByID(ctx, roomID, viewer.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.False(t, row.IsMember)
	assert.Equal(t, "", row.ViewerRole)
}

func TestChatDAO_GetRoomByID_Member(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, owner.ID, "host", false))

	// when
	row, err := repos.Chat.GetRoomByID(ctx, roomID, owner.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsMember)
	assert.Equal(t, "host", row.ViewerRole)
}

func TestChatDAO_GetRoomByID_IncludesTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"lore", "rp"}))

	// when
	row, err := repos.Chat.GetRoomByID(ctx, roomID, owner.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.ElementsMatch(t, []string{"lore", "rp"}, row.Tags)
}

func TestChatDAO_GetRoomMembers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, a.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, b.ID))

	// when
	members, err := repos.Chat.GetRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, members)
}

func TestChatDAO_GetRoomMembers_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	members, err := repos.Chat.GetRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestChatDAO_GetRoomMembersDetailed(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos, daotest.WithDisplayName("Owner"))
	member := daotest.CreateUser(t, repos, daotest.WithDisplayName("Member"))
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, owner.ID, "host", false))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, member.ID, "member", false))

	// when
	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, detailed, 2)
	assert.Equal(t, owner.ID, detailed[0].UserID)
	assert.Equal(t, "host", detailed[0].Role)
	assert.Equal(t, member.ID, detailed[1].UserID)
	assert.Equal(t, "member", detailed[1].Role)
}

func TestChatDAO_IsMember_True(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))

	// when
	ok, err := repos.Chat.IsMember(ctx, roomID, owner.ID)

	// then
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestChatDAO_IsMember_False(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	ok, err := repos.Chat.IsMember(ctx, roomID, other.ID)

	// then
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestChatDAO_SetMuted_And_IsMuted(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))

	// when
	err = repos.Chat.SetMuted(ctx, roomID, owner.ID, true)

	// then
	require.NoError(t, err)
	muted, err := repos.Chat.IsMuted(ctx, roomID, owner.ID)
	require.NoError(t, err)
	assert.True(t, muted)
}

func TestChatDAO_IsMuted_Unmute(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))
	require.NoError(t, repos.Chat.SetMuted(ctx, roomID, owner.ID, true))

	// when
	err = repos.Chat.SetMuted(ctx, roomID, owner.ID, false)

	// then
	require.NoError(t, err)
	muted, err := repos.Chat.IsMuted(ctx, roomID, owner.ID)
	require.NoError(t, err)
	assert.False(t, muted)
}

func TestChatDAO_IsMuted_NotMember(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	muted, err := repos.Chat.IsMuted(ctx, roomID, other.ID)

	// then
	require.NoError(t, err)
	assert.False(t, muted)
}

func TestChatDAO_GetRoomMembersUnmuted(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, a.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, b.ID))
	require.NoError(t, repos.Chat.SetMuted(ctx, roomID, a.ID, true))

	// when
	members, err := repos.Chat.GetRoomMembersUnmuted(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{b.ID}, members)
}

func TestChatDAO_FindDMRoom_Found(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	roomID := room.ID

	// when
	got, err := repos.Chat.FindDMRoom(ctx, a.ID, b.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
}

func TestChatDAO_FindDMRoom_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)

	// when
	got, err := repos.Chat.FindDMRoom(ctx, a.ID, b.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestChatDAO_AddRoomTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	err = repos.Chat.AddRoomTags(ctx, roomID, []string{"a", "b"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "b"}, tags)
}

func TestChatDAO_AddRoomTags_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	err = repos.Chat.AddRoomTags(ctx, roomID, nil)

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatDAO_AddRoomTags_SkipEmptyStrings(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	err = repos.Chat.AddRoomTags(ctx, roomID, []string{"valid", "", "also"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"valid", "also"}, tags)
}

func TestChatDAO_AddRoomTags_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"x"}))

	// when
	err = repos.Chat.AddRoomTags(ctx, roomID, []string{"x", "y"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"x", "y"}, tags)
}

func TestChatDAO_ReplaceRoomTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"old1", "old2"}))

	// when
	err = repos.Chat.ReplaceRoomTags(ctx, roomID, []string{"new1", "new2"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"new1", "new2"}, tags)
}

func TestChatDAO_ReplaceRoomTags_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"a"}))

	// when
	err = repos.Chat.ReplaceRoomTags(ctx, roomID, nil)

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatDAO_GetRoomTags_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatDAO_GetRoomTagsBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room1Row, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "r1", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	room1 := room1Row.ID
	room2Row, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "r2", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	room2 := room2Row.ID
	require.NoError(t, repos.Chat.AddRoomTags(ctx, room1, []string{"t1", "t2"}))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, room2, []string{"t3"}))

	// when
	got, err := repos.Chat.GetRoomTagsBatch(ctx, []uuid.UUID{room1, room2})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"t1", "t2"}, got[room1])
	assert.ElementsMatch(t, []string{"t3"}, got[room2])
}

func TestChatDAO_GetRoomTagsBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetRoomTagsBatch(ctx, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChatDAO_GetRoomsByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	r1Row, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R1", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	r1 := r1Row.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, r1, user.ID, "host", false))
	r2Row, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R2", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: other.ID})
	require.NoError(t, err)
	r2 := r2Row.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, r2, other.ID, "host", false))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	assert.Equal(t, r1, rooms[0].ID)
	assert.True(t, rooms[0].IsMember)
}

func TestChatDAO_GetRoomsByUser_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, rooms)
}

func TestChatDAO_GetRoomsByUser_SystemFirst(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	sysID := uuid.New()
	normal, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Normal", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	normalID := normal.ID
	require.NoError(t, repos.Chat.AddMember(ctx, normalID, user.ID))
	_, err = repos.Chat.CreateSystemRoom(ctx, repository.NewChatSystemRoom{ID: sysID, Name: "Sys", Description: "", SystemKind: "announcements", CreatedBy: user.ID})
	require.NoError(t, err)
	require.NoError(t, repos.Chat.AddMember(ctx, sysID, user.ID))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 2)
	assert.True(t, rooms[0].IsSystem)
	assert.Equal(t, sysID, rooms[0].ID)
}

func TestChatDAO_GetRoomsByUser_IncludesTags(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"lore"}))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	assert.ElementsMatch(t, []string{"lore"}, rooms[0].Tags)
}

func TestChatDAO_ListUserGroupRooms_Basic(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Alpha", Description: "about alpha", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, user.ID, "host", false))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "", false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, roomID, rooms[0].ID)
}

func TestChatDAO_ListUserGroupRooms_SearchFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	aRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Apples", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	a := aRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, a, user.ID))
	bRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Bananas", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	b := bRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, b, user.ID))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "Apple", false, "", "", false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, a, rooms[0].ID)
}

func TestChatDAO_ListUserGroupRooms_RPOnlyFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	normalRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Normal", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	normal := normalRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, normal, user.ID))
	rpRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "RP", Description: "", Type: "group", IsPublic: false, IsRP: true, CreatedBy: user.ID})
	require.NoError(t, err)
	rp := rpRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, rp, user.ID))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", true, "", "", false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, rp, rooms[0].ID)
}

func TestChatDAO_ListUserGroupRooms_TagFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	taggedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "T", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	tagged := taggedRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, tagged, user.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, tagged, []string{"lore"}))
	plainRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "P", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	plain := plainRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, plain, user.ID))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "lore", "", false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, tagged, rooms[0].ID)
}

func TestChatDAO_ListUserGroupRooms_HostRoleFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	hostedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "H", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	hosted := hostedRow.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, hosted, user.ID, "host", false))
	joinedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "J", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: other.ID})
	require.NoError(t, err)
	joined := joinedRow.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, joined, user.ID, "member", false))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "host", false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, hosted, rooms[0].ID)
}

func TestChatDAO_ListUserGroupRooms_MemberRoleFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	hostedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "H", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	hosted := hostedRow.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, hosted, user.ID, "host", false))
	joinedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "J", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: other.ID})
	require.NoError(t, err)
	joined := joinedRow.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, joined, user.ID, "member", false))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "member", false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, joined, rooms[0].ID)
}

func TestChatDAO_ListUserGroupRooms_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	for range 3 {
		created, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
		require.NoError(t, err)
		id := created.ID
		require.NoError(t, repos.Chat.AddMember(ctx, id, user.ID))
	}

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "", false, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rooms, 2)
}

func TestChatDAO_ListPublicRooms_Basic(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	publicRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Public", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	public := publicRow.ID
	_, err = repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Private", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, public, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_ExcludesSystem(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	sysID := uuid.New()
	_, err := repos.Chat.CreateSystemRoom(ctx, repository.NewChatSystemRoom{ID: sysID, Name: "Sys", Description: "", SystemKind: "announcements", CreatedBy: owner.ID})
	require.NoError(t, err)

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rooms)
}

func TestChatDAO_ListPublicRooms_ExcludesMembership(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	joinedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Joined", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	joined := joinedRow.ID
	require.NoError(t, repos.Chat.AddMember(ctx, joined, viewer.ID))
	unjoinedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Unjoined", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	unjoined := unjoinedRow.ID

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, unjoined, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_SearchFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	applesRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Apples", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	apples := applesRow.ID
	_, err = repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Bananas", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "Apple", false, "", viewer.ID, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, apples, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_RPOnly(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	_, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "N", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	rpRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "RP", Description: "", Type: "group", IsPublic: true, IsRP: true, CreatedBy: owner.ID})
	require.NoError(t, err)
	rp := rpRow.ID

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", true, "", viewer.ID, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, rp, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_TagFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	taggedRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "T", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	tagged := taggedRow.ID
	require.NoError(t, repos.Chat.AddRoomTags(ctx, tagged, []string{"lore"}))
	_, err = repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "P", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "lore", viewer.ID, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, tagged, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_ExcludeUsers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	ownerA := daotest.CreateUser(t, repos)
	ownerB := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	_, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "A", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: ownerA.ID})
	require.NoError(t, err)
	roomBRow, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "B", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: ownerB.ID})
	require.NoError(t, err)
	roomB := roomBRow.ID

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, []uuid.UUID{ownerA.ID}, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, roomB, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_NilViewer(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", uuid.Nil, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, roomID, rooms[0].ID)
}

func TestChatDAO_ListPublicRooms_IsMemberFlag(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	_, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)

	// when
	rooms, _, err := repos.Chat.ListPublicRooms(ctx, "", false, "", uuid.Nil, nil, false, 20, 0)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	assert.False(t, rooms[0].IsMember)
	_ = viewer
}

func TestChatDAO_ListPublicRooms_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)
	for range 3 {
		_, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: true, IsRP: false, CreatedBy: owner.ID})
		require.NoError(t, err)
	}

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, false, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rooms, 2)
}

func TestChatDAO_InsertMessage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hello"})

	// then
	require.NoError(t, err)
	got, err := repos.Chat.GetMessageByID(ctx, msg.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "hello", got.Body)
	assert.Equal(t, user.ID, got.SenderID)
}

func TestChatDAO_SearchMessagesForViewer_FindsMatchInMemberRoom(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	match, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "the golden witch beatrice laughs"})
	require.NoError(t, err)
	matchID := match.ID
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "an ordinary mundane lunch"})
	require.NoError(t, err)

	// when
	results, total, err := repos.Chat.SearchMessagesForViewer(ctx, user.ID, uuid.Nil, "beatrice", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, matchID.String(), results[0].ID)
	require.NotNil(t, results[0].ParentID)
	assert.Equal(t, roomID.String(), *results[0].ParentID)
}

func TestChatDAO_SearchMessagesForViewer_ExcludesNonMemberRooms(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	viewer := daotest.CreateUser(t, repos)
	owner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "Private", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: owner.ID, Body: "secret beatrice plans"})
	require.NoError(t, err)

	// when
	results, total, err := repos.Chat.SearchMessagesForViewer(ctx, viewer.ID, uuid.Nil, "beatrice", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, results)
}

func TestChatDAO_SearchMessagesForViewer_RoomFilter(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room1Row, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R1", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	room1 := room1Row.ID
	room2Row, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R2", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	room2 := room2Row.ID
	require.NoError(t, repos.Chat.AddMember(ctx, room1, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, room2, user.ID))
	msg1Row, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: room1, SenderID: user.ID, Body: "phoenix rises"})
	require.NoError(t, err)
	msg1 := msg1Row.ID
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: room2, SenderID: user.ID, Body: "phoenix falls"})
	require.NoError(t, err)

	// when
	scoped, scopedTotal, err := repos.Chat.SearchMessagesForViewer(ctx, user.ID, room1, "phoenix", 20, 0)
	require.NoError(t, err)
	all, allTotal, err := repos.Chat.SearchMessagesForViewer(ctx, user.ID, uuid.Nil, "phoenix", 20, 0)
	require.NoError(t, err)

	// then
	assert.Equal(t, 1, scopedTotal)
	require.Len(t, scoped, 1)
	assert.Equal(t, msg1.String(), scoped[0].ID)
	assert.Equal(t, 2, allTotal)
	assert.Len(t, all, 2)
}

func TestChatDAO_SearchMessagesForViewer_ExcludesSystemMessages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	_, err = repos.Chat.InsertSystemMessage(ctx, roomID, user.ID, "beatrice joined the room")
	require.NoError(t, err)

	// when
	results, total, err := repos.Chat.SearchMessagesForViewer(ctx, user.ID, uuid.Nil, "beatrice", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, results)
}

func TestChatDAO_SearchMessagesForViewer_CreatedAtSupportsJumpCursor(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "unicorn sighting reported"})
	require.NoError(t, err)
	msgID := msg.ID

	// when: the created_at returned by search is used to build a jump cursor
	results, _, err := repos.Chat.SearchMessagesForViewer(ctx, user.ID, uuid.Nil, "unicorn", 20, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	cursor := results[0].CreatedAt + "|ffffffff-ffff-ffff-ffff-ffffffffffff"
	before, err := repos.Chat.GetMessagesBefore(ctx, roomID, uuid.Nil, cursor, 50)
	require.NoError(t, err)

	// then: the target message is inside that cursor window (full-precision round-trip)
	found := false
	for _, m := range before {
		if m.ID == msgID {
			found = true
		}
	}
	assert.True(t, found, "jump cursor built from the search created_at must include the target message")
}

func TestChatDAO_InsertMessage_UpdatesRoomLastMessage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastMessageAt.Valid)
}

func TestChatDAO_InsertMessage_WithReply(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Sender"))
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	parent, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "parent"})
	require.NoError(t, err)
	parentID := parent.ID

	// when
	reply, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "reply", ReplyToID: &parentID})

	// then
	require.NoError(t, err)
	got, err := repos.Chat.GetMessageByID(ctx, reply.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.ReplyToID)
	assert.Equal(t, parentID, *got.ReplyToID)
	require.NotNil(t, got.ReplyToBody)
	assert.Equal(t, "parent", *got.ReplyToBody)
	require.NotNil(t, got.ReplyToSenderName)
	assert.Equal(t, "Sender", *got.ReplyToSenderName)
}

func TestChatDAO_ReplyPreview_UsesRoomAliasWhenSet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	parentAuthor := daotest.CreateUser(t, repos, daotest.WithDisplayName("RealName"))
	replier := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: parentAuthor.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, parentAuthor.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, replier.ID))
	require.NoError(t, repos.Chat.SetMemberNickname(ctx, roomID, parentAuthor.ID, "Battler"))
	parent, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: parentAuthor.ID, Body: "parent"})
	require.NoError(t, err)
	parentID := parent.ID
	reply, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: replier.ID, Body: "reply", ReplyToID: &parentID})
	require.NoError(t, err)
	replyID := reply.ID

	// when
	got, err := repos.Chat.GetMessageByID(ctx, replyID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.ReplyToSenderName)
	assert.Equal(t, "Battler", *got.ReplyToSenderName)
}

func TestChatDAO_ReplyPreview_FallsBackToDisplayNameWhenNoAlias(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	parentAuthor := daotest.CreateUser(t, repos, daotest.WithDisplayName("RealName"))
	replier := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: parentAuthor.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, parentAuthor.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, replier.ID))
	parent, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: parentAuthor.ID, Body: "parent"})
	require.NoError(t, err)
	parentID := parent.ID
	reply, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: replier.ID, Body: "reply", ReplyToID: &parentID})
	require.NoError(t, err)
	replyID := reply.ID

	// when
	got, err := repos.Chat.GetMessageByID(ctx, replyID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.ReplyToSenderName)
	assert.Equal(t, "RealName", *got.ReplyToSenderName)
}

func TestChatDAO_GetMessages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	for range 3 {
		_, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
		require.NoError(t, err)
	}

	// when
	msgs, total, err := repos.Chat.GetMessages(ctx, roomID, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, msgs, 3)
}

func TestChatDAO_GetMessages_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	msgs, total, err := repos.Chat.GetMessages(ctx, roomID, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, msgs)
}

func TestChatDAO_GetMessages_Limit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	for range 5 {
		_, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
		require.NoError(t, err)
	}

	// when
	msgs, total, err := repos.Chat.GetMessages(ctx, roomID, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, msgs, 2)
}

func TestChatDAO_GetMessagesBefore(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	for range 3 {
		_, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
		require.NoError(t, err)
	}

	// when
	msgs, err := repos.Chat.GetMessagesBefore(ctx, roomID, uuid.Nil, "2099-01-01 00:00:00", 20)

	// then
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestChatDAO_GetMessagesBefore_FiltersOld(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)

	// when
	msgs, err := repos.Chat.GetMessagesBefore(ctx, roomID, uuid.Nil, "2000-01-01 00:00:00", 20)

	// then
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestChatDAO_GetMessagesBefore_RFC3339CursorUsesDatetimeComparison(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	older, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "older"})
	require.NoError(t, err)
	olderID := older.ID
	newer, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "newer"})
	require.NoError(t, err)
	newerID := newer.ID

	_, err = repos.DB().ExecContext(ctx,
		`UPDATE chat_messages SET created_at = $1 WHERE id = $2`,
		"2024-01-01 00:30:00", olderID,
	)
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx,
		`UPDATE chat_messages SET created_at = $1 WHERE id = $2`,
		"2024-01-01 02:00:00", newerID,
	)
	require.NoError(t, err)

	// when
	msgs, err := repos.Chat.GetMessagesBefore(ctx, roomID, uuid.Nil, "2024-01-01T01:00:00Z", 20)

	// then
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, olderID, msgs[0].ID)
}

func TestChatDAO_GetMessagesBefore_CursorWithIDPaginatesSameSecondMessages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	ids := make([]uuid.UUID, 3)
	for i := range ids {
		msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
		require.NoError(t, err)
		ids[i] = msg.ID
		_, err = repos.DB().ExecContext(ctx,
			`UPDATE chat_messages SET created_at = $1 WHERE id = $2`,
			"2024-01-01 00:00:00", ids[i],
		)
		require.NoError(t, err)
	}

	sorted := make([]string, 0, len(ids))
	for i := range ids {
		sorted = append(sorted, ids[i].String())
	}
	sort.Strings(sorted)
	expectedOldestID := sorted[0]

	// when
	firstPage, total, err := repos.Chat.GetMessages(ctx, roomID, 2, 0)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, firstPage, 2)

	cursor := firstPage[0].CreatedAt + "|" + firstPage[0].ID.String()
	secondPage, err := repos.Chat.GetMessagesBefore(ctx, roomID, uuid.Nil, cursor, 2)

	// then
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, expectedOldestID, secondPage[0].ID.String())
}

func TestChatDAO_GetMessageByID_NotFound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetMessageByID(ctx, uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestChatDAO_DeleteMessages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)

	// when
	err = repos.Chat.DeleteMessages(ctx, roomID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Chat.GetMessages(ctx, roomID, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestChatDAO_GetMessageSenderID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	sender, err := repos.Chat.GetMessageSenderID(ctx, msgID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, sender)
}

func TestChatDAO_GetMessageRoomID(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	got, err := repos.Chat.GetMessageRoomID(ctx, msgID)

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
}

func TestChatDAO_AddMessageMedia(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	id, err := repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: msgID, MediaURL: "/url", MediaType: "image", ThumbnailURL: "/thumb", SortOrder: 0})

	// then
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msgID})
	require.NoError(t, err)
	require.Len(t, media[msgID], 1)
	assert.Equal(t, "/url", media[msgID][0].MediaURL)
	assert.Equal(t, "image", media[msgID][0].MediaType)
	assert.Equal(t, "/thumb", media[msgID][0].ThumbnailURL)
}

func TestChatDAO_UpdateMessageMediaURL(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)
	msgID := msg.ID
	id, err := repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: msgID, MediaURL: "/old", MediaType: "image", ThumbnailURL: "", SortOrder: 0})
	require.NoError(t, err)

	// when
	err = repos.Chat.UpdateMessageMediaURL(ctx, id, "/new")

	// then
	require.NoError(t, err)
	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msgID})
	require.NoError(t, err)
	require.Len(t, media[msgID], 1)
	assert.Equal(t, "/new", media[msgID][0].MediaURL)
}

func TestChatDAO_UpdateMessageMediaThumbnail(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)
	msgID := msg.ID
	id, err := repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: msgID, MediaURL: "/u", MediaType: "image", ThumbnailURL: "", SortOrder: 0})
	require.NoError(t, err)

	// when
	err = repos.Chat.UpdateMessageMediaThumbnail(ctx, id, "/newthumb")

	// then
	require.NoError(t, err)
	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msgID})
	require.NoError(t, err)
	require.Len(t, media[msgID], 1)
	assert.Equal(t, "/newthumb", media[msgID][0].ThumbnailURL)
}

func TestChatDAO_GetMessageMediaBatch_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	media, err := repos.Chat.GetMessageMediaBatch(ctx, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, media)
}

func TestChatDAO_GetMessageMediaBatch_SortOrder(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "m"})
	require.NoError(t, err)
	msgID := msg.ID
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: msgID, MediaURL: "/b", MediaType: "image", ThumbnailURL: "", SortOrder: 2})
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: msgID, MediaURL: "/a", MediaType: "image", ThumbnailURL: "", SortOrder: 1})
	require.NoError(t, err)

	// when
	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msgID})

	// then
	require.NoError(t, err)
	require.Len(t, media[msgID], 2)
	assert.Equal(t, "/a", media[msgID][0].MediaURL)
	assert.Equal(t, "/b", media[msgID][1].MediaURL)
}

func TestChatDAO_TouchRoomActivity(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	err = repos.Chat.TouchRoomActivity(ctx, roomID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastMessageAt.Valid)
}

func TestChatDAO_MarkRoomRead(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	err = repos.Chat.MarkRoomRead(ctx, roomID, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastReadAt.Valid)
}

func TestChatDAO_CountUnreadRoomsForUser_Zero(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatDAO_CountUnreadRoomsForUser_DMUnread(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	roomID := room.ID
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: b.ID, Body: "hi"})
	require.NoError(t, err)

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, a.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestChatDAO_CountUnreadRoomsForUser_AfterMarkRead(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	roomID := room.ID
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: b.ID, Body: "hi"})
	require.NoError(t, err)
	require.NoError(t, repos.Chat.MarkRoomRead(ctx, roomID, a.ID))

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, a.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatDAO_CountUnreadRoomsForUser_IgnoresGroups(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})
	require.NoError(t, err)

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatDAO_SetMemberNickname(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	err = repos.Chat.SetMemberNickname(ctx, roomID, user.ID, "Beato")

	// then
	require.NoError(t, err)
	members, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "Beato", members[0].Nickname)
}

func TestChatDAO_SetMemberAvatar_Overwrites(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, roomID, user.ID, "/uploads/chat-avatars/first.png"))

	// when
	err = repos.Chat.SetMemberAvatar(ctx, roomID, user.ID, "/uploads/chat-avatars/second.png")

	// then
	require.NoError(t, err)
	members, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "/uploads/chat-avatars/second.png", members[0].MemberAvatarURL)
}

func TestChatDAO_PinAndUnpinMessage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	require.NoError(t, repos.Chat.PinMessage(ctx, msgID, user.ID))
	pinned, err := repos.Chat.ListPinnedMessages(ctx, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, pinned, 1)
	assert.Equal(t, msgID, pinned[0].ID)
	require.NotNil(t, pinned[0].PinnedAt)
	require.NotNil(t, pinned[0].PinnedBy)
	assert.Equal(t, user.ID, *pinned[0].PinnedBy)

	require.NoError(t, repos.Chat.UnpinMessage(ctx, msgID))
	after, err := repos.Chat.ListPinnedMessages(ctx, roomID)
	require.NoError(t, err)
	assert.Len(t, after, 0)
}

func TestChatDAO_ListPinnedMessages_OrdersByPinnedAtDesc(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	firstRow, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "first"})
	require.NoError(t, err)
	first := firstRow.ID
	secondRow, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "second"})
	require.NoError(t, err)
	second := secondRow.ID

	// when
	require.NoError(t, repos.Chat.PinMessage(ctx, first, user.ID))
	_, _ = repos.DB().ExecContext(ctx, `UPDATE chat_messages SET pinned_at = pinned_at - INTERVAL '1 second' WHERE id = $1`, first)
	require.NoError(t, repos.Chat.PinMessage(ctx, second, user.ID))
	pinned, err := repos.Chat.ListPinnedMessages(ctx, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, pinned, 2)
	assert.Equal(t, second, pinned[0].ID)
	assert.Equal(t, first, pinned[1].ID)
}

func TestChatDAO_AddAndRemoveReaction(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	inserted, err := repos.Chat.AddReaction(ctx, msgID, user.ID, "👍")
	require.NoError(t, err)
	assert.True(t, inserted)
	groups, err := repos.Chat.GetReactionsBatch(ctx, []uuid.UUID{msgID}, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, groups[msgID], 1)
	assert.Equal(t, "👍", groups[msgID][0].Emoji)
	assert.Equal(t, 1, groups[msgID][0].Count)
	assert.True(t, groups[msgID][0].ViewerReacted)

	deleted, err := repos.Chat.RemoveReaction(ctx, msgID, user.ID, "👍")
	require.NoError(t, err)
	assert.True(t, deleted)
	after, err := repos.Chat.GetReactionsBatch(ctx, []uuid.UUID{msgID}, user.ID)
	require.NoError(t, err)
	assert.Empty(t, after[msgID])
}

func TestChatDAO_AddReaction_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	firstInserted, err := repos.Chat.AddReaction(ctx, msgID, user.ID, "🎉")
	require.NoError(t, err)
	assert.True(t, firstInserted)
	secondInserted, err := repos.Chat.AddReaction(ctx, msgID, user.ID, "🎉")
	require.NoError(t, err)
	assert.False(t, secondInserted)
	groups, err := repos.Chat.GetReactionsBatch(ctx, []uuid.UUID{msgID}, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, groups[msgID], 1)
	assert.Equal(t, 1, groups[msgID][0].Count)
}

func TestChatDAO_GetReactionsBatch_GroupsByEmoji(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: userA.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, userA.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, userB.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: userA.ID, Body: "hi"})
	require.NoError(t, err)
	msgID := msg.ID
	_, err = repos.Chat.AddReaction(ctx, msgID, userA.ID, "👍")
	require.NoError(t, err)
	_, err = repos.Chat.AddReaction(ctx, msgID, userB.ID, "👍")
	require.NoError(t, err)
	_, err = repos.Chat.AddReaction(ctx, msgID, userA.ID, "😂")
	require.NoError(t, err)

	// when
	groups, err := repos.Chat.GetReactionsBatch(ctx, []uuid.UUID{msgID}, userB.ID)

	// then
	require.NoError(t, err)
	require.Len(t, groups[msgID], 2)
	thumbs := groups[msgID][0]
	assert.Equal(t, "👍", thumbs.Emoji)
	assert.Equal(t, 2, thumbs.Count)
	assert.True(t, thumbs.ViewerReacted)
	laugh := groups[msgID][1]
	assert.Equal(t, "😂", laugh.Emoji)
	assert.Equal(t, 1, laugh.Count)
	assert.False(t, laugh.ViewerReacted)
}

func TestChatDAO_IsMemberNicknameLocked_False(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	locked, err := repos.Chat.IsMemberNicknameLocked(ctx, roomID, user.ID)

	// then
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestChatDAO_IsMemberNicknameLocked_True(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, roomID, user.ID, "Locked", true))

	// when
	locked, err := repos.Chat.IsMemberNicknameLocked(ctx, roomID, user.ID)

	// then
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestChatDAO_IsMemberNicknameLocked_NotMember(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	locked, err := repos.Chat.IsMemberNicknameLocked(ctx, roomID, uuid.New())

	// then
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestChatDAO_SetMemberNicknameWithLock_LocksAndUnlocks(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, roomID, user.ID, "Forced", true))
	lockedAfter, err := repos.Chat.IsMemberNicknameLocked(ctx, roomID, user.ID)

	// then
	require.NoError(t, err)
	assert.True(t, lockedAfter)
	members, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "Forced", members[0].Nickname)
	assert.True(t, members[0].NicknameLocked)

	// and when unlocking
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, roomID, user.ID, "", false))
	lockedAfterUnlock, err := repos.Chat.IsMemberNicknameLocked(ctx, roomID, user.ID)
	require.NoError(t, err)
	assert.False(t, lockedAfterUnlock)
	members2, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, members2, 1)
	assert.Equal(t, "", members2[0].Nickname)
	assert.False(t, members2[0].NicknameLocked)
}

func TestChatDAO_GetRoomMembersDetailed_PopulatesNicknameLocked(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, owner.ID, "host", false))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, other.ID, "member", false))
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, roomID, other.ID, "Pinned", true))

	// when
	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, detailed, 2)
	assert.False(t, detailed[0].NicknameLocked)
	assert.Equal(t, other.ID, detailed[1].UserID)
	assert.True(t, detailed[1].NicknameLocked)
	assert.Equal(t, "Pinned", detailed[1].Nickname)
}

func TestChatDAO_SetMemberTimeoutAndGetMemberTimeoutState(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, member.ID))

	until := "2099-01-01 00:00:00"

	// when
	err = repos.Chat.SetMemberTimeout(ctx, roomID, member.ID, until, true)

	// then
	require.NoError(t, err)
	active, gotUntil, byStaff, err := repos.Chat.GetMemberTimeoutState(ctx, roomID, member.ID)
	require.NoError(t, err)
	assert.True(t, active)
	assert.Contains(t, gotUntil, "2099-01-01")
	assert.True(t, byStaff)
}

func TestChatDAO_ClearMemberTimeout(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, member.ID))
	require.NoError(t, repos.Chat.SetMemberTimeout(ctx, roomID, member.ID, "2099-01-01 00:00:00", true))

	// when
	err = repos.Chat.ClearMemberTimeout(ctx, roomID, member.ID)

	// then
	require.NoError(t, err)
	active, gotUntil, byStaff, err := repos.Chat.GetMemberTimeoutState(ctx, roomID, member.ID)
	require.NoError(t, err)
	assert.False(t, active)
	assert.Equal(t, "", gotUntil)
	assert.False(t, byStaff)
}

func TestChatDAO_GetRoomMembersDetailed_ShowsOnlyActiveTimeout(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, owner.ID, "host", false))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, member.ID, "member", false))
	require.NoError(t, repos.Chat.SetMemberTimeout(ctx, roomID, member.ID, "2099-01-01 00:00:00", true))

	// when
	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)

	// then
	require.NoError(t, err)
	require.Len(t, detailed, 2)
	assert.Equal(t, "", detailed[0].TimeoutUntil)
	assert.False(t, detailed[0].TimeoutByStaff)
	assert.Equal(t, member.ID, detailed[1].UserID)
	assert.Equal(t, "2099-01-01T00:00:00Z", detailed[1].TimeoutUntil)
	assert.True(t, detailed[1].TimeoutByStaff)
}

func TestChatDAO_RemoveMember_SoftDeletes(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, joiner.ID))
	require.NoError(t, repos.Chat.SetMemberNickname(ctx, roomID, joiner.ID, "Beato"))

	// when
	err = repos.Chat.RemoveMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	isMember, err := repos.Chat.IsMember(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.False(t, isMember)

	var count int
	require.NoError(t, repos.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_room_members WHERE room_id = $1 AND user_id = $2 AND left_at IS NOT NULL`,
		roomID, joiner.ID,
	).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestChatDAO_AddMember_Rejoin_PreservesNickname(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: owner.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, joiner.ID, "member", false))
	require.NoError(t, repos.Chat.SetMemberNicknameWithLock(ctx, roomID, joiner.ID, "Beato", true))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, roomID, joiner.ID, "/custom.png"))
	require.NoError(t, repos.Chat.RemoveMember(ctx, roomID, joiner.ID))

	// when
	err = repos.Chat.AddMemberWithRole(ctx, roomID, joiner.ID, "member", false)

	// then
	require.NoError(t, err)
	isMember, err := repos.Chat.IsMember(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.True(t, isMember)

	detailed, err := repos.Chat.GetRoomMembersDetailed(ctx, roomID)
	require.NoError(t, err)
	var found *repository.ChatRoomMemberRow
	for i := range detailed {
		if detailed[i].UserID == joiner.ID {
			found = &detailed[i]
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Beato", found.Nickname)
	assert.True(t, found.NicknameLocked)
	assert.Equal(t, "/custom.png", found.MemberAvatarURL)
}

func TestChatDAO_DeleteMessage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})
	require.NoError(t, err)
	msgID := msg.ID

	// when
	err = repos.Chat.DeleteMessage(ctx, msgID)

	// then
	require.NoError(t, err)
	got, err := repos.Chat.GetMessageByID(ctx, msgID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestChatDAO_EditMessage_UpdatesBodyAndStampsEditedAt(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "original"})
	require.NoError(t, err)
	msgID := msg.ID
	before, err := repos.Chat.GetMessageByID(ctx, msgID)
	require.NoError(t, err)
	require.NotNil(t, before)
	assert.Nil(t, before.EditedAt, "new message should have no edited_at")

	// when
	err = repos.Chat.EditMessage(ctx, msgID, "updated body")

	// then
	require.NoError(t, err)
	after, err := repos.Chat.GetMessageByID(ctx, msgID)
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.Equal(t, "updated body", after.Body)
	require.NotNil(t, after.EditedAt)
	assert.NotEmpty(t, *after.EditedAt)
}

func TestChatDAO_EditMessage_SurfacesInListQueries(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msg, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hello"})
	require.NoError(t, err)
	msgID := msg.ID
	require.NoError(t, repos.Chat.EditMessage(ctx, msgID, "hello world"))

	// when
	messages, _, err := repos.Chat.GetMessages(ctx, roomID, 10, 0)

	// then
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "hello world", messages[0].Body)
	require.NotNil(t, messages[0].EditedAt)
}

func TestChatDAO_EditMessage_UnknownIDIsNoop(t *testing.T) {
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	err := repos.Chat.EditMessage(ctx, uuid.New(), "noop")

	// then
	require.NoError(t, err)
}

func TestChatDAO_GetMessages_UsesPerRoomOverrides(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.SetMemberNickname(ctx, roomID, user.ID, "Beato"))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, roomID, user.ID, "/custom.png"))
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "hi"})
	require.NoError(t, err)

	// when
	msgs, _, err := repos.Chat.GetMessages(ctx, roomID, 10, 0)

	// then
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Beato", msgs[0].SenderNickname)
	assert.Equal(t, "/custom.png", msgs[0].SenderMemberAvatar)
}

func TestChatDAO_InsertSystemMessage_SetsSystemFlag(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateRoom(ctx, repository.NewChatRoom{Name: "R", Description: "", Type: "group", IsPublic: false, IsRP: false, CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	msg, err := repos.Chat.InsertSystemMessage(ctx, roomID, user.ID, "System test")

	// then
	require.NoError(t, err)
	got, err := repos.Chat.GetMessageByID(ctx, msg.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsSystem)
}

func TestChatDAO_CreateDMRoomAtomic_RestoresAMemberWhoLeft(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)

	room, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.RemoveMember(ctx, roomID, a.ID))

	left, err := repos.Chat.IsMember(ctx, roomID, a.ID)
	require.NoError(t, err)
	require.False(t, left, "leaving must actually remove membership")

	// when
	againRow, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, againRow.ID, "the pair keeps its room")

	rejoined, err := repos.Chat.IsMember(ctx, roomID, a.ID)
	require.NoError(t, err)
	assert.True(t, rejoined, "messaging againRow.ID must put the sender back in the room")
}

func TestChatDAO_GetMessagesForMember_StartsFreshAfterDeletingADM(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)

	room, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	roomID := room.ID
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: a.ID, Body: "before"})
	require.NoError(t, err)
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: b.ID, Body: "also before"})
	require.NoError(t, err)

	require.NoError(t, repos.Chat.RemoveMember(ctx, roomID, a.ID))
	_, err = repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: a.ID, Body: "after"})
	require.NoError(t, err)

	// when
	mine, err := repos.Chat.GetMessagesForMember(ctx, roomID, a.ID, 50)
	require.NoError(t, err)

	theirs, err := repos.Chat.GetMessagesForMember(ctx, roomID, b.ID, 50)
	require.NoError(t, err)

	// then
	require.Len(t, mine, 1, "the member who deleted the chat only sees what came after")
	assert.Equal(t, "after", mine[0].Body)
	assert.Len(t, theirs, 3, "the other member keeps the whole conversation")
}

func TestChatDAO_CreateGroupRoom_CreatesRoomTagsAndMembers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)

	// when
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{
		Name:        "Party",
		Description: "desc",
		IsPublic:    true,
		CreatedBy:   host.ID,
		Tags:        []string{"tag"},
		MemberIDs:   []uuid.UUID{member.ID},
	})

	// then
	require.NoError(t, err)
	hostRole, err := repos.Chat.GetMemberRole(ctx, room.ID, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", hostRole)
	memberRole, err := repos.Chat.GetMemberRole(ctx, room.ID, member.ID)
	require.NoError(t, err)
	assert.Equal(t, "member", memberRole)
	tags, err := repos.Chat.GetRoomTags(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"tag"}, tags)
}

func TestChatDAO_CreateGroupRoom_RollsBackWhenAMemberDoesNotExist(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)

	// when
	_, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{
		Name:      "Party",
		CreatedBy: host.ID,
		MemberIDs: []uuid.UUID{uuid.New()},
	})

	// then
	require.Error(t, err)
	rooms, err := repos.Chat.GetRoomsByUser(ctx, host.ID)
	require.NoError(t, err)
	assert.Empty(t, rooms, "the half built room must not survive the failed member insert")
}

func TestChatDAO_UpdateRoom_WritesEveryEditableFieldAndLeavesTheRestAlone(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{
		Name:        "Old name",
		Description: "old description",
		IsPublic:    false,
		IsRP:        true,
		CreatedBy:   host.ID,
	})
	require.NoError(t, err)
	before, err := repos.Chat.GetRoomByID(ctx, room.ID, host.ID)
	require.NoError(t, err)
	require.NotNil(t, before)

	// when
	err = repos.Chat.UpdateRoom(ctx, repository.UpdateChatRoom{
		RoomID:      room.ID,
		Name:        "New name",
		Description: "new description",
		IsPublic:    true,
		IsRP:        false,
	})

	// then
	require.NoError(t, err)
	after, err := repos.Chat.GetRoomByID(ctx, room.ID, host.ID)
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.Equal(t, "New name", after.Name)
	assert.Equal(t, "new description", after.Description)
	assert.True(t, after.IsPublic)
	assert.False(t, after.IsRP)
	assert.Equal(t, dto.RoomTypeGroup, after.Type)
	assert.False(t, after.IsSystem)
	assert.Equal(t, before.CreatedBy, after.CreatedBy)
	assert.Equal(t, before.CreatedAt, after.CreatedAt, "an edit must never restamp created_at")
}

func TestChatDAO_UpdateGroupRoom_ReplacesTheWholeTagSet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{
		Name:      "Tagged",
		CreatedBy: host.ID,
		Tags:      []string{"old-one", "old-two"},
	})
	require.NoError(t, err)

	// when
	err = repos.Chat.UpdateGroupRoom(ctx, repository.UpdateChatRoom{
		RoomID:      room.ID,
		Name:        "Tagged again",
		Description: "desc",
		Tags:        []string{"new-one", "new-two", "new-three"},
		IsPublic:    true,
		IsRP:        false,
	})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, room.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"new-one", "new-two", "new-three"}, tags)
	after, err := repos.Chat.GetRoomByID(ctx, room.ID, host.ID)
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.Equal(t, "Tagged again", after.Name)
	assert.True(t, after.IsPublic)
}

func TestChatDAO_UpdateGroupRoom_ClearsTheTagsWhenNoneAreGiven(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{
		Name:      "Tagged",
		CreatedBy: host.ID,
		Tags:      []string{"keep-me"},
	})
	require.NoError(t, err)

	// when
	err = repos.Chat.UpdateGroupRoom(ctx, repository.UpdateChatRoom{
		RoomID: room.ID,
		Name:   "Tagged",
		Tags:   nil,
	})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, room.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatDAO_UpdateRoom_RefusesToTouchADM(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateDMRoomAtomic(ctx, a.ID, b.ID)
	require.NoError(t, err)
	before, err := repos.Chat.GetRoomByID(ctx, room.ID, a.ID)
	require.NoError(t, err)
	require.NotNil(t, before)

	// when
	err = repos.Chat.UpdateRoom(ctx, repository.UpdateChatRoom{
		RoomID:      room.ID,
		Name:        "Renamed DM",
		Description: "hijacked",
		IsPublic:    true,
		IsRP:        true,
	})

	// then
	require.Error(t, err, "the WHERE clause carries type = 'group' so a DM must never match")
	after, err := repos.Chat.GetRoomByID(ctx, room.ID, a.ID)
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.Equal(t, before.Name, after.Name)
	assert.Equal(t, before.Description, after.Description)
	assert.Equal(t, before.IsPublic, after.IsPublic)
	assert.Equal(t, before.IsRP, after.IsRP)
}

func TestChatDAO_UpdateRoom_RefusesToTouchASystemRoom(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	_, err := repos.Chat.CreateSystemRoom(ctx, repository.NewChatSystemRoom{ID: roomID, Name: "Announcements", Description: "system room", SystemKind: "announcements", CreatedBy: host.ID})
	require.NoError(t, err)

	// when
	err = repos.Chat.UpdateRoom(ctx, repository.UpdateChatRoom{
		RoomID:      roomID,
		Name:        "Renamed system room",
		Description: "hijacked",
		IsPublic:    true,
		IsRP:        true,
	})

	// then
	require.Error(t, err, "the WHERE clause carries is_system = FALSE so a system room must never match")
	after, err := repos.Chat.GetRoomByID(ctx, roomID, host.ID)
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.Equal(t, "Announcements", after.Name)
	assert.Equal(t, "system room", after.Description)
	assert.True(t, after.IsSystem)
}

func TestChatDAO_UpdateRoom_MissingRowIsAnError(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()

	// when
	err := repos.Chat.UpdateRoom(ctx, repository.UpdateChatRoom{
		RoomID: uuid.New(),
		Name:   "Ghost room",
	})

	// then
	require.Error(t, err)
}

func TestChatDAO_CreateSystemRoomWithHost(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	room, err := repos.Chat.CreateSystemRoomWithHost(ctx, repository.NewChatSystemRoom{
		ID:         roomID,
		Name:       "Stream",
		SystemKind: "live_stream",
		CreatedBy:  host.ID,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, room.ID)
	got, err := repos.Chat.GetMemberRole(ctx, roomID, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", got)
}

func TestChatDAO_SyncSystemRoomMembership_JoinsAndLeaves(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	joinRoomID := uuid.New()
	leaveRoomID := uuid.New()
	require.NoError(t, repos.Chat.CreateSystemRooms(ctx, []repository.NewChatSystemRoom{
		{ID: joinRoomID, Name: "Mods", SystemKind: "mods", CreatedBy: user.ID},
		{ID: leaveRoomID, Name: "Admins", SystemKind: "admins", CreatedBy: user.ID},
	}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, leaveRoomID, user.ID, "member", false))

	// when
	changes, err := repos.Chat.SyncSystemRoomMembership(ctx, []repository.SystemRoomMembership{
		{RoomID: joinRoomID, UserID: user.ID, ShouldBeMember: true, DesiredRole: "host"},
		{RoomID: leaveRoomID, UserID: user.ID, ShouldBeMember: false, DesiredRole: "host"},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, []repository.SystemRoomMembershipChange{
		{RoomID: joinRoomID, Joined: true},
		{RoomID: leaveRoomID, Left: true},
	}, changes)
	joined, err := repos.Chat.GetMemberRole(ctx, joinRoomID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", joined)
	left, err := repos.Chat.GetMemberRole(ctx, leaveRoomID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "", left)
}

func TestChatDAO_SyncSystemRoomMembership_UpgradesAnExistingRole(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateSystemRooms(ctx, []repository.NewChatSystemRoom{
		{ID: roomID, Name: "Mods", SystemKind: "mods", CreatedBy: user.ID},
	}))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, user.ID, "member", false))

	// when
	changes, err := repos.Chat.SyncSystemRoomMembership(ctx, []repository.SystemRoomMembership{
		{RoomID: roomID, UserID: user.ID, ShouldBeMember: true, DesiredRole: "host"},
	})

	// then
	require.NoError(t, err)
	assert.Empty(t, changes, "a role change is not a join or a leave, so the hub is left alone")
	got, err := repos.Chat.GetMemberRole(ctx, roomID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", got)
}

func TestChatDAO_AddMemberWithSystemMessage(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{Name: "R", CreatedBy: host.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	msg, err := repos.Chat.AddMemberWithSystemMessage(ctx,
		repository.NewChatRoomMember{RoomID: roomID, UserID: joiner.ID, Role: "member"},
		repository.NewChatMessage{RoomID: roomID, SenderID: joiner.ID, Body: "Test User joined the room.", IsSystem: true},
	)

	// then
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, "Test User joined the room.", msg.Body)
	assert.True(t, msg.IsSystem)
	role, err := repos.Chat.GetMemberRole(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, "member", role)
}

func TestChatDAO_AddMemberWithSystemMessage_SkipsTheMessageWhenTheBodyIsEmpty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	joiner := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{Name: "R", CreatedBy: host.ID})
	require.NoError(t, err)
	roomID := room.ID

	// when
	msg, err := repos.Chat.AddMemberWithSystemMessage(ctx,
		repository.NewChatRoomMember{RoomID: roomID, UserID: joiner.ID, Role: "member", Ghost: true},
		repository.NewChatMessage{RoomID: roomID, SenderID: joiner.ID, IsSystem: true},
	)

	// then
	require.NoError(t, err)
	assert.Nil(t, msg, "a ghost join is silent")
	role, err := repos.Chat.GetMemberRole(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, "member", role)
}

func TestChatDAO_DeleteRoomWithMessages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{Name: "R", CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID
	_, err = repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "bye"})
	require.NoError(t, err)

	// when
	paths, err := repos.Chat.DeleteRoomWithMessages(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Empty(t, paths)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatDAO_DeleteRoomWithMessages_ReturnsEveryOrphanedFile(t *testing.T) {
	// given a room whose messages carry media and whose members carry room avatars
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	guest := daotest.CreateUser(t, repos)
	bare := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{Name: "R", CreatedBy: host.ID, MemberIDs: []uuid.UUID{guest.ID, bare.ID}})
	require.NoError(t, err)
	roomID := room.ID
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, roomID, host.ID, "/uploads/chat-avatars/host.webp"))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, roomID, guest.ID, "/uploads/chat-avatars/guest.webp"))

	first, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: host.ID, Body: "look"})
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: first.ID, MediaURL: "/uploads/chat/one.webp", MediaType: "image", ThumbnailURL: "/uploads/chat/one_thumb.webp", SortOrder: 0})
	require.NoError(t, err)

	second, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: guest.ID, Body: "clip"})
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: second.ID, MediaURL: "/uploads/chat/two.mp4", MediaType: "video", ThumbnailURL: "", SortOrder: 0})
	require.NoError(t, err)

	// when
	paths, err := repos.Chat.DeleteRoomWithMessages(ctx, roomID)

	// then every file the cascade orphaned comes back, and the blank thumbnail is skipped
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/chat/one.webp",
		"/uploads/chat/one_thumb.webp",
		"/uploads/chat/two.mp4",
		"/uploads/chat-avatars/host.webp",
		"/uploads/chat-avatars/guest.webp",
	}, paths)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, host.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatDAO_DeleteMessageWithMedia_ReturnsThatMessagesFiles(t *testing.T) {
	// given two messages in the same room, each carrying media
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	room, err := repos.Chat.CreateGroupRoom(ctx, repository.NewChatGroupRoom{Name: "R", CreatedBy: user.ID})
	require.NoError(t, err)
	roomID := room.ID

	target, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "mine"})
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: target.ID, MediaURL: "/uploads/chat/target.webp", MediaType: "image", ThumbnailURL: "/uploads/chat/target_thumb.webp", SortOrder: 0})
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: target.ID, MediaURL: "/uploads/chat/target2.mp4", MediaType: "video", ThumbnailURL: "", SortOrder: 1})
	require.NoError(t, err)

	other, err := repos.Chat.InsertMessageAndMarkRead(ctx, repository.NewChatMessage{RoomID: roomID, SenderID: user.ID, Body: "theirs"})
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, repository.NewChatMessageMedia{MessageID: other.ID, MediaURL: "/uploads/chat/other.webp", MediaType: "image", ThumbnailURL: "/uploads/chat/other_thumb.webp", SortOrder: 0})
	require.NoError(t, err)

	// when
	paths, err := repos.Chat.DeleteMessageWithMedia(ctx, target.ID)

	// then only the deleted message's files come back
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/uploads/chat/target.webp",
		"/uploads/chat/target_thumb.webp",
		"/uploads/chat/target2.mp4",
	}, paths)
	gone, err := repos.Chat.GetMessageByID(ctx, target.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)
	survivor, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{other.ID})
	require.NoError(t, err)
	require.Len(t, survivor[other.ID], 1)
	assert.Equal(t, "/uploads/chat/other.webp", survivor[other.ID][0].MediaURL)
}
