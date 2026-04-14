package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatRepository_CreateRoom_Group(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	err := repos.Chat.CreateRoom(ctx, roomID, "Room", "desc", "group", true, false, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Room", row.Name)
	assert.Equal(t, "desc", row.Description)
	assert.Equal(t, "group", row.Type)
	assert.True(t, row.IsPublic)
	assert.False(t, row.IsRP)
	assert.Equal(t, user.ID, row.CreatedBy)
}

func TestChatRepository_CreateRoom_RPFlag(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	err := repos.Chat.CreateRoom(ctx, roomID, "RP", "", "group", false, true, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsRP)
	assert.False(t, row.IsPublic)
}

func TestChatRepository_CreateSystemRoom(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	err := repos.Chat.CreateSystemRoom(ctx, roomID, "System", "system room", "announcements", user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsSystem)
	assert.Equal(t, "announcements", row.SystemKind)
	assert.Equal(t, "group", row.Type)
}

func TestChatRepository_GetSystemRoomID_Found(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateSystemRoom(ctx, roomID, "Sys", "", "announcements", user.ID))

	// when
	got, err := repos.Chat.GetSystemRoomID(ctx, "announcements")

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
}

func TestChatRepository_GetSystemRoomID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetSystemRoomID(ctx, "missing")

	// then
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestChatRepository_CreateDMRoomAtomic_New(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	got, err := repos.Chat.CreateDMRoomAtomic(ctx, roomID, a.ID, b.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
	members, err := repos.Chat.GetRoomMembers(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, members)
}

func TestChatRepository_CreateDMRoomAtomic_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	first := uuid.New()
	_, err := repos.Chat.CreateDMRoomAtomic(ctx, first, a.ID, b.ID)
	require.NoError(t, err)

	// when
	got, err := repos.Chat.CreateDMRoomAtomic(ctx, uuid.New(), b.ID, a.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, first, got)
}

func TestChatRepository_AddMember(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	joiner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	err := repos.Chat.AddMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	isMember, err := repos.Chat.IsMember(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.True(t, isMember)
}

func TestChatRepository_AddMember_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	joiner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, joiner.ID))

	// when
	err := repos.Chat.AddMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestChatRepository_AddMemberWithRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	joiner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	err := repos.Chat.AddMemberWithRole(ctx, roomID, joiner.ID, "host")

	// then
	require.NoError(t, err)
	role, err := repos.Chat.GetMemberRole(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", role)
}

func TestChatRepository_SetMemberRole(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	joiner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, joiner.ID, "member"))

	// when
	err := repos.Chat.SetMemberRole(ctx, roomID, joiner.ID, "host")

	// then
	require.NoError(t, err)
	role, err := repos.Chat.GetMemberRole(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, "host", role)
}

func TestChatRepository_GetMemberRole_NotMember(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	role, err := repos.Chat.GetMemberRole(ctx, roomID, other.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "", role)
}

func TestChatRepository_RemoveMember(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	joiner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, joiner.ID))

	// when
	err := repos.Chat.RemoveMember(ctx, roomID, joiner.ID)

	// then
	require.NoError(t, err)
	isMember, err := repos.Chat.IsMember(ctx, roomID, joiner.ID)
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestChatRepository_CountRoomMembers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, a.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, b.ID))

	// when
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestChatRepository_CountRoomMembers_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	count, err := repos.Chat.CountRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatRepository_DeleteRoom(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	err := repos.Chat.DeleteRoom(ctx, roomID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, owner.ID)
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatRepository_GetRoomByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	// when
	row, err := repos.Chat.GetRoomByID(ctx, uuid.New(), user.ID)

	// then
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatRepository_GetRoomByID_NonMember(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))

	// when
	row, err := repos.Chat.GetRoomByID(ctx, roomID, viewer.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.False(t, row.IsMember)
	assert.Equal(t, "", row.ViewerRole)
}

func TestChatRepository_GetRoomByID_Member(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, owner.ID, "host"))

	// when
	row, err := repos.Chat.GetRoomByID(ctx, roomID, owner.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsMember)
	assert.Equal(t, "host", row.ViewerRole)
}

func TestChatRepository_GetRoomByID_IncludesTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"lore", "rp"}))

	// when
	row, err := repos.Chat.GetRoomByID(ctx, roomID, owner.ID)

	// then
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.ElementsMatch(t, []string{"lore", "rp"}, row.Tags)
}

func TestChatRepository_GetRoomMembers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, a.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, b.ID))

	// when
	members, err := repos.Chat.GetRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, members)
}

func TestChatRepository_GetRoomMembers_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	members, err := repos.Chat.GetRoomMembers(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestChatRepository_GetRoomMembersDetailed(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos, repotest.WithDisplayName("Owner"))
	member := repotest.CreateUser(t, repos, repotest.WithDisplayName("Member"))
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, owner.ID, "host"))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, member.ID, "member"))

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

func TestChatRepository_IsMember_True(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))

	// when
	ok, err := repos.Chat.IsMember(ctx, roomID, owner.ID)

	// then
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestChatRepository_IsMember_False(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	ok, err := repos.Chat.IsMember(ctx, roomID, other.ID)

	// then
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestChatRepository_SetMuted_And_IsMuted(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))

	// when
	err := repos.Chat.SetMuted(ctx, roomID, owner.ID, true)

	// then
	require.NoError(t, err)
	muted, err := repos.Chat.IsMuted(ctx, roomID, owner.ID)
	require.NoError(t, err)
	assert.True(t, muted)
}

func TestChatRepository_IsMuted_Unmute(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, owner.ID))
	require.NoError(t, repos.Chat.SetMuted(ctx, roomID, owner.ID, true))

	// when
	err := repos.Chat.SetMuted(ctx, roomID, owner.ID, false)

	// then
	require.NoError(t, err)
	muted, err := repos.Chat.IsMuted(ctx, roomID, owner.ID)
	require.NoError(t, err)
	assert.False(t, muted)
}

func TestChatRepository_IsMuted_NotMember(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	muted, err := repos.Chat.IsMuted(ctx, roomID, other.ID)

	// then
	require.NoError(t, err)
	assert.False(t, muted)
}

func TestChatRepository_GetRoomMembersUnmuted(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, a.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, b.ID))
	require.NoError(t, repos.Chat.SetMuted(ctx, roomID, a.ID, true))

	// when
	members, err := repos.Chat.GetRoomMembersUnmuted(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{b.ID}, members)
}

func TestChatRepository_FindDMRoom_Found(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	_, err := repos.Chat.CreateDMRoomAtomic(ctx, roomID, a.ID, b.ID)
	require.NoError(t, err)

	// when
	got, err := repos.Chat.FindDMRoom(ctx, a.ID, b.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
}

func TestChatRepository_FindDMRoom_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)

	// when
	got, err := repos.Chat.FindDMRoom(ctx, a.ID, b.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestChatRepository_AddRoomTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	err := repos.Chat.AddRoomTags(ctx, roomID, []string{"a", "b"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "b"}, tags)
}

func TestChatRepository_AddRoomTags_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	err := repos.Chat.AddRoomTags(ctx, roomID, nil)

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatRepository_AddRoomTags_SkipEmptyStrings(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	err := repos.Chat.AddRoomTags(ctx, roomID, []string{"valid", "", "also"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"valid", "also"}, tags)
}

func TestChatRepository_AddRoomTags_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"x"}))

	// when
	err := repos.Chat.AddRoomTags(ctx, roomID, []string{"x", "y"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"x", "y"}, tags)
}

func TestChatRepository_ReplaceRoomTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"old1", "old2"}))

	// when
	err := repos.Chat.ReplaceRoomTags(ctx, roomID, []string{"new1", "new2"})

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"new1", "new2"}, tags)
}

func TestChatRepository_ReplaceRoomTags_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"a"}))

	// when
	err := repos.Chat.ReplaceRoomTags(ctx, roomID, nil)

	// then
	require.NoError(t, err)
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatRepository_GetRoomTags_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, owner.ID))

	// when
	tags, err := repos.Chat.GetRoomTags(ctx, roomID)

	// then
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestChatRepository_GetRoomTagsBatch(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	room1 := uuid.New()
	room2 := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, room1, "r1", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, room2, "r2", "", "group", false, false, owner.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, room1, []string{"t1", "t2"}))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, room2, []string{"t3"}))

	// when
	got, err := repos.Chat.GetRoomTagsBatch(ctx, []uuid.UUID{room1, room2})

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"t1", "t2"}, got[room1])
	assert.ElementsMatch(t, []string{"t3"}, got[room2])
}

func TestChatRepository_GetRoomTagsBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetRoomTagsBatch(ctx, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChatRepository_GetRoomsByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	r1 := uuid.New()
	r2 := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, r1, "R1", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, r1, user.ID, "host"))
	require.NoError(t, repos.Chat.CreateRoom(ctx, r2, "R2", "", "group", false, false, other.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, r2, other.ID, "host"))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	assert.Equal(t, r1, rooms[0].ID)
	assert.True(t, rooms[0].IsMember)
}

func TestChatRepository_GetRoomsByUser_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, rooms)
}

func TestChatRepository_GetRoomsByUser_SystemFirst(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	normalID := uuid.New()
	sysID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, normalID, "Normal", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, normalID, user.ID))
	require.NoError(t, repos.Chat.CreateSystemRoom(ctx, sysID, "Sys", "", "announcements", user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, sysID, user.ID))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 2)
	assert.True(t, rooms[0].IsSystem)
	assert.Equal(t, sysID, rooms[0].ID)
}

func TestChatRepository_GetRoomsByUser_IncludesTags(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, roomID, []string{"lore"}))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	assert.ElementsMatch(t, []string{"lore"}, rooms[0].Tags)
}

func TestChatRepository_ListUserGroupRooms_Basic(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "Alpha", "about alpha", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, roomID, user.ID, "host"))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, roomID, rooms[0].ID)
}

func TestChatRepository_ListUserGroupRooms_SearchFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	a := uuid.New()
	b := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, a, "Apples", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, a, user.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, b, "Bananas", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, b, user.ID))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "Apple", false, "", "", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, a, rooms[0].ID)
}

func TestChatRepository_ListUserGroupRooms_RPOnlyFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	normal := uuid.New()
	rp := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, normal, "Normal", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, normal, user.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, rp, "RP", "", "group", false, true, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, rp, user.ID))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", true, "", "", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, rp, rooms[0].ID)
}

func TestChatRepository_ListUserGroupRooms_TagFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	tagged := uuid.New()
	plain := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, tagged, "T", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, tagged, user.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, tagged, []string{"lore"}))
	require.NoError(t, repos.Chat.CreateRoom(ctx, plain, "P", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, plain, user.ID))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "lore", "", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, tagged, rooms[0].ID)
}

func TestChatRepository_ListUserGroupRooms_HostRoleFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	hosted := uuid.New()
	joined := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, hosted, "H", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, hosted, user.ID, "host"))
	require.NoError(t, repos.Chat.CreateRoom(ctx, joined, "J", "", "group", false, false, other.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, joined, user.ID, "member"))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "host", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, hosted, rooms[0].ID)
}

func TestChatRepository_ListUserGroupRooms_MemberRoleFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	hosted := uuid.New()
	joined := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, hosted, "H", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, hosted, user.ID, "host"))
	require.NoError(t, repos.Chat.CreateRoom(ctx, joined, "J", "", "group", false, false, other.ID))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, joined, user.ID, "member"))

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "member", 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, joined, rooms[0].ID)
}

func TestChatRepository_ListUserGroupRooms_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		id := uuid.New()
		require.NoError(t, repos.Chat.CreateRoom(ctx, id, "R", "", "group", false, false, user.ID))
		require.NoError(t, repos.Chat.AddMember(ctx, id, user.ID))
	}

	// when
	rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, user.ID, "", false, "", "", 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rooms, 2)
}

func TestChatRepository_ListPublicRooms_Basic(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	public := uuid.New()
	private := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, public, "Public", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, private, "Private", "", "group", false, false, owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, public, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_ExcludesSystem(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	sysID := uuid.New()
	require.NoError(t, repos.Chat.CreateSystemRoom(ctx, sysID, "Sys", "", "announcements", owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rooms)
}

func TestChatRepository_ListPublicRooms_ExcludesMembership(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	joined := uuid.New()
	unjoined := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, joined, "Joined", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, joined, viewer.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, unjoined, "Unjoined", "", "group", true, false, owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, unjoined, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_SearchFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	apples := uuid.New()
	bananas := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, apples, "Apples", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, bananas, "Bananas", "", "group", true, false, owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "Apple", false, "", viewer.ID, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, apples, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_RPOnly(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	normal := uuid.New()
	rp := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, normal, "N", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, rp, "RP", "", "group", true, true, owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", true, "", viewer.ID, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, rp, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_TagFilter(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	tagged := uuid.New()
	plain := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, tagged, "T", "", "group", true, false, owner.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, tagged, []string{"lore"}))
	require.NoError(t, repos.Chat.CreateRoom(ctx, plain, "P", "", "group", true, false, owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "lore", viewer.ID, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, tagged, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_ExcludeUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	ownerA := repotest.CreateUser(t, repos)
	ownerB := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	roomA := uuid.New()
	roomB := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomA, "A", "", "group", true, false, ownerA.ID))
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomB, "B", "", "group", true, false, ownerB.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, []uuid.UUID{ownerA.ID}, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, roomB, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_NilViewer(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", uuid.Nil, nil, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rooms, 1)
	assert.Equal(t, roomID, rooms[0].ID)
}

func TestChatRepository_ListPublicRooms_IsMemberFlag(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", true, false, owner.ID))

	// when
	rooms, _, err := repos.Chat.ListPublicRooms(ctx, "", false, "", uuid.Nil, nil, 20, 0)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 1)
	assert.False(t, rooms[0].IsMember)
	_ = viewer
}

func TestChatRepository_ListPublicRooms_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	owner := repotest.CreateUser(t, repos)
	viewer := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		id := uuid.New()
		require.NoError(t, repos.Chat.CreateRoom(ctx, id, "R", "", "group", true, false, owner.ID))
	}

	// when
	rooms, total, err := repos.Chat.ListPublicRooms(ctx, "", false, "", viewer.ID, nil, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rooms, 2)
}

func TestChatRepository_InsertMessage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()

	// when
	err := repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "hello", nil)

	// then
	require.NoError(t, err)
	got, err := repos.Chat.GetMessageByID(ctx, msgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "hello", got.Body)
	assert.Equal(t, user.ID, got.SenderID)
}

func TestChatRepository_InsertMessage_UpdatesRoomLastMessage(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	err := repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "hi", nil)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastMessageAt.Valid)
}

func TestChatRepository_InsertMessage_WithReply(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Sender"))
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	parentID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, parentID, roomID, user.ID, "parent", nil))
	replyID := uuid.New()

	// when
	err := repos.Chat.InsertMessage(ctx, replyID, roomID, user.ID, "reply", &parentID)

	// then
	require.NoError(t, err)
	got, err := repos.Chat.GetMessageByID(ctx, replyID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.ReplyToID)
	assert.Equal(t, parentID, *got.ReplyToID)
	require.NotNil(t, got.ReplyToBody)
	assert.Equal(t, "parent", *got.ReplyToBody)
	require.NotNil(t, got.ReplyToSenderName)
	assert.Equal(t, "Sender", *got.ReplyToSenderName)
}

func TestChatRepository_GetMessages(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	for i := 0; i < 3; i++ {
		require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "m", nil))
	}

	// when
	msgs, total, err := repos.Chat.GetMessages(ctx, roomID, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, msgs, 3)
}

func TestChatRepository_GetMessages_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))

	// when
	msgs, total, err := repos.Chat.GetMessages(ctx, roomID, 20, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, msgs)
}

func TestChatRepository_GetMessages_Limit(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	for i := 0; i < 5; i++ {
		require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "m", nil))
	}

	// when
	msgs, total, err := repos.Chat.GetMessages(ctx, roomID, 2, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, msgs, 2)
}

func TestChatRepository_GetMessagesBefore(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	for i := 0; i < 3; i++ {
		require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "m", nil))
	}

	// when
	msgs, err := repos.Chat.GetMessagesBefore(ctx, roomID, "2099-01-01 00:00:00", 20)

	// then
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestChatRepository_GetMessagesBefore_FiltersOld(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "m", nil))

	// when
	msgs, err := repos.Chat.GetMessagesBefore(ctx, roomID, "2000-01-01 00:00:00", 20)

	// then
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestChatRepository_GetMessageByID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	// when
	got, err := repos.Chat.GetMessageByID(ctx, uuid.New())

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestChatRepository_DeleteMessages(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "m", nil))

	// when
	err := repos.Chat.DeleteMessages(ctx, roomID)

	// then
	require.NoError(t, err)
	_, total, err := repos.Chat.GetMessages(ctx, roomID, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestChatRepository_GetMessageSenderID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "m", nil))

	// when
	sender, err := repos.Chat.GetMessageSenderID(ctx, msgID)

	// then
	require.NoError(t, err)
	assert.Equal(t, user.ID, sender)
}

func TestChatRepository_GetMessageRoomID(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "m", nil))

	// when
	got, err := repos.Chat.GetMessageRoomID(ctx, msgID)

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, got)
}

func TestChatRepository_AddMessageMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "m", nil))

	// when
	id, err := repos.Chat.AddMessageMedia(ctx, msgID, "/url", "image", "/thumb", 0)

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

func TestChatRepository_UpdateMessageMediaURL(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "m", nil))
	id, err := repos.Chat.AddMessageMedia(ctx, msgID, "/old", "image", "", 0)
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

func TestChatRepository_UpdateMessageMediaThumbnail(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "m", nil))
	id, err := repos.Chat.AddMessageMedia(ctx, msgID, "/u", "image", "", 0)
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

func TestChatRepository_GetMessageMediaBatch_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	// when
	media, err := repos.Chat.GetMessageMediaBatch(ctx, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, media)
}

func TestChatRepository_GetMessageMediaBatch_SortOrder(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	msgID := uuid.New()
	require.NoError(t, repos.Chat.InsertMessage(ctx, msgID, roomID, user.ID, "m", nil))
	_, err := repos.Chat.AddMessageMedia(ctx, msgID, "/b", "image", "", 2)
	require.NoError(t, err)
	_, err = repos.Chat.AddMessageMedia(ctx, msgID, "/a", "image", "", 1)
	require.NoError(t, err)

	// when
	media, err := repos.Chat.GetMessageMediaBatch(ctx, []uuid.UUID{msgID})

	// then
	require.NoError(t, err)
	require.Len(t, media[msgID], 2)
	assert.Equal(t, "/a", media[msgID][0].MediaURL)
	assert.Equal(t, "/b", media[msgID][1].MediaURL)
}

func TestChatRepository_TouchRoomActivity(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	err := repos.Chat.TouchRoomActivity(ctx, roomID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastMessageAt.Valid)
}

func TestChatRepository_MarkRoomRead(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))

	// when
	err := repos.Chat.MarkRoomRead(ctx, roomID, user.ID)

	// then
	require.NoError(t, err)
	row, err := repos.Chat.GetRoomByID(ctx, roomID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.LastReadAt.Valid)
}

func TestChatRepository_CountUnreadRoomsForUser_Zero(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatRepository_CountUnreadRoomsForUser_DMUnread(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	_, err := repos.Chat.CreateDMRoomAtomic(ctx, roomID, a.ID, b.ID)
	require.NoError(t, err)
	require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, b.ID, "hi", nil))

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, a.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestChatRepository_CountUnreadRoomsForUser_AfterMarkRead(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	_, err := repos.Chat.CreateDMRoomAtomic(ctx, roomID, a.ID, b.ID)
	require.NoError(t, err)
	require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, b.ID, "hi", nil))
	require.NoError(t, repos.Chat.MarkRoomRead(ctx, roomID, a.ID))

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, a.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChatRepository_CountUnreadRoomsForUser_IgnoresGroups(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	roomID := uuid.New()
	require.NoError(t, repos.Chat.CreateRoom(ctx, roomID, "R", "", "group", false, false, user.ID))
	require.NoError(t, repos.Chat.AddMember(ctx, roomID, user.ID))
	require.NoError(t, repos.Chat.InsertMessage(ctx, uuid.New(), roomID, user.ID, "hi", nil))

	// when
	count, err := repos.Chat.CountUnreadRoomsForUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
