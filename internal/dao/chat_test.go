package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatDAO_CreateGroupRoom_CreatesRoomTagsAndMembers(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	member := daotest.CreateUser(t, repos)
	outsider := daotest.CreateUser(t, repos)

	// when
	room, err := repos.Chat.CreateGroupRoom(ctx, spec.NewChatGroupRoom{
		Name:        "Party",
		Description: "desc",
		IsPublic:    true,
		IsRP:        false,
		CreatedBy:   host.ID,
		Tags:        []string{"lore", "rp"},
		MemberIDs:   []uuid.UUID{member.ID},
	})
	require.NoError(t, err)

	// then
	hostRole, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: room.ID, UserID: host.ID})
	require.NoError(t, err)
	assert.Equal(t, "host", hostRole)

	memberRole, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: room.ID, UserID: member.ID})
	require.NoError(t, err)
	assert.Equal(t, "member", memberRole)

	hostView, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: room.ID, ViewerID: host.ID})
	require.NoError(t, err)
	require.NotNil(t, hostView)
	assert.Equal(t, "Party", hostView.Name)
	assert.Equal(t, "desc", hostView.Description)
	assert.Equal(t, dto.RoomTypeGroup, hostView.Type)
	assert.True(t, hostView.IsPublic)
	assert.False(t, hostView.IsRP)
	assert.False(t, hostView.IsSystem)
	assert.Equal(t, host.ID, hostView.CreatedBy)
	assert.True(t, hostView.IsMember)
	assert.Equal(t, "host", hostView.ViewerRole)
	assert.Equal(t, []string{"lore", "rp"}, hostView.Tags)

	outsiderView, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: room.ID, ViewerID: outsider.ID})
	require.NoError(t, err)
	require.NotNil(t, outsiderView)
	assert.False(t, outsiderView.IsMember)
	assert.Equal(t, "", outsiderView.ViewerRole)

	missing, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: uuid.New(), ViewerID: host.ID})
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestChatDAO_CreateGroupRoom_RollsBackWhenAMemberDoesNotExist(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)

	// when
	_, err := repos.Chat.CreateGroupRoom(ctx, spec.NewChatGroupRoom{
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

func TestChatDAO_CreateSystemRoomWithHost(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	roomID := uuid.New()

	// when
	room, err := repos.Chat.CreateSystemRoomWithHost(ctx, spec.NewChatSystemRoom{
		ID:          roomID,
		Name:        "Announcements",
		Description: "system room",
		SystemKind:  "announcements",
		CreatedBy:   host.ID,
	})
	require.NoError(t, err)

	// then
	assert.Equal(t, roomID, room.ID)

	role, err := repos.Chat.GetMemberRole(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: host.ID})
	require.NoError(t, err)
	assert.Equal(t, "host", role)

	row, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: host.ID})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.True(t, row.IsSystem)
	assert.Equal(t, "announcements", row.SystemKind)
	assert.Equal(t, dto.RoomTypeGroup, row.Type)
	assert.Equal(t, "Announcements", row.Name)
	assert.Equal(t, "system room", row.Description)

	found, err := repos.Chat.GetSystemRoomID(ctx, "announcements")
	require.NoError(t, err)
	assert.Equal(t, roomID, found)

	missing, err := repos.Chat.GetSystemRoomID(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, missing)
}

func TestChatDAO_CreateDMRoomAtomic_CreatesOneRoomPerPair(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	c := daotest.CreateUser(t, repos)

	// when
	room, err := repos.Chat.CreateDMRoomAtomic(ctx, spec.ChatDMPair{UserA: a.ID, UserB: b.ID})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, room.ID)

	// then
	members, err := repos.Chat.GetRoomMembers(ctx, room.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{a.ID, b.ID}, members)

	found, err := repos.Chat.FindDMRoom(ctx, spec.ChatDMPair{UserA: a.ID, UserB: b.ID})
	require.NoError(t, err)
	assert.Equal(t, room.ID, found)

	none, err := repos.Chat.FindDMRoom(ctx, spec.ChatDMPair{UserA: a.ID, UserB: c.ID})
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, none)

	// when the pair is opened again from the other side
	again, err := repos.Chat.CreateDMRoomAtomic(ctx, spec.ChatDMPair{UserA: b.ID, UserB: a.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, room.ID, again.ID)
}

func TestChatDAO_CreateDMRoomAtomic_RestoresALeaverToTheRoomTheStayerStillResolves(t *testing.T) {
	// given a pair with history that one party has left
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	stayer := daotest.CreateUser(t, repos)
	leaver := daotest.CreateUser(t, repos)
	roomID := daotest.CreateDMRoom(t, repos, stayer.ID, leaver.ID)
	daotest.SendChatMessage(t, repos, roomID, stayer.ID, "hello")

	require.NoError(t, repos.Chat.RemoveMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: leaver.ID}))

	left, err := repos.Chat.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: leaver.ID})
	require.NoError(t, err)
	require.False(t, left, "leaving must actually remove membership")

	// when each side resolves the pair
	forStayer, err := repos.Chat.FindDMRoom(ctx, spec.ChatDMPair{UserA: stayer.ID, UserB: leaver.ID})
	require.NoError(t, err)

	forLeaver, err := repos.Chat.FindDMRoom(ctx, spec.ChatDMPair{UserA: leaver.ID, UserB: stayer.ID})
	require.NoError(t, err)

	byPair, err := repos.Chat.FindDMRoomByPair(ctx, spec.ChatDMPair{UserA: stayer.ID, UserB: leaver.ID})
	require.NoError(t, err)

	// then the side that still holds the thread resolves the very room the next send attaches to
	require.NotNil(t, byPair)
	assert.Equal(t, roomID, byPair.ID)
	assert.Equal(t, roomID, forStayer, "resolve must not report a fresh conversation when the history is still on screen")
	assert.Equal(t, uuid.Nil, forLeaver, "the party who left has no thread to resolve until they open a new one")

	// when the leaver reopens the pair
	again, err := repos.Chat.CreateDMRoomAtomic(ctx, spec.ChatDMPair{UserA: leaver.ID, UserB: stayer.ID})

	// then
	require.NoError(t, err)
	assert.Equal(t, roomID, again.ID, "the pair keeps its room")

	rejoined, err := repos.Chat.IsMember(ctx, spec.ChatMemberRef{RoomID: roomID, UserID: leaver.ID})
	require.NoError(t, err)
	assert.True(t, rejoined, "messaging the pair must put the sender back in the room")
}

func TestChatDAO_UpdateGroupRoom(t *testing.T) {
	tests := []struct {
		name        string
		initialTags []string
		update      spec.UpdateChatRoom
	}{
		{
			name:        "writes every editable field and replaces the whole tag set",
			initialTags: []string{"old-one", "old-two"},
			update: spec.UpdateChatRoom{
				Name:        "New name",
				Description: "new description",
				IsPublic:    true,
				IsRP:        false,
				Tags:        []string{"new-one", "new-two", "new-three"},
			},
		},
		{
			name:        "a tags-only edit that leaves every field as it was clears the tags",
			initialTags: []string{"keep-me"},
			update: spec.UpdateChatRoom{
				Name:        "Old name",
				Description: "old description",
				IsPublic:    false,
				IsRP:        true,
				Tags:        nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			host := daotest.CreateUser(t, repos)

			room, err := repos.Chat.CreateGroupRoom(ctx, spec.NewChatGroupRoom{
				Name:        "Old name",
				Description: "old description",
				IsPublic:    false,
				IsRP:        true,
				CreatedBy:   host.ID,
				Tags:        tc.initialTags,
			})
			require.NoError(t, err)

			before, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: room.ID, ViewerID: host.ID})
			require.NoError(t, err)
			require.NotNil(t, before)
			require.True(t, before.IsRP)
			require.False(t, before.IsPublic)

			// when
			update := tc.update
			update.RoomID = room.ID
			err = repos.Chat.UpdateGroupRoom(ctx, update)
			require.NoError(t, err)

			// then
			after, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: room.ID, ViewerID: host.ID})
			require.NoError(t, err)
			require.NotNil(t, after)
			assert.Equal(t, tc.update.Name, after.Name)
			assert.Equal(t, tc.update.Description, after.Description)
			assert.Equal(t, tc.update.IsPublic, after.IsPublic)
			assert.Equal(t, tc.update.IsRP, after.IsRP)
			assert.Equal(t, dto.RoomTypeGroup, after.Type)
			assert.False(t, after.IsSystem)
			assert.Equal(t, before.CreatedBy, after.CreatedBy)
			assert.Equal(t, before.CreatedAt, after.CreatedAt, "an edit must never restamp created_at")

			tags, err := repos.Chat.GetRoomTags(ctx, room.ID)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.update.Tags, tags)
		})
	}
}

func TestChatDAO_UpdateRoom(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, repos *repository.Repositories) spec.ChatRoomViewer
		exists bool
	}{
		{
			name: "a DM",
			setup: func(t *testing.T, repos *repository.Repositories) spec.ChatRoomViewer {
				a := daotest.CreateUser(t, repos)
				b := daotest.CreateUser(t, repos)

				return spec.ChatRoomViewer{RoomID: daotest.CreateDMRoom(t, repos, a.ID, b.ID), ViewerID: a.ID}
			},
			exists: true,
		},
		{
			name: "a system room",
			setup: func(t *testing.T, repos *repository.Repositories) spec.ChatRoomViewer {
				host := daotest.CreateUser(t, repos)
				id := uuid.New()

				_, err := repos.Chat.CreateSystemRoom(context.Background(), spec.NewChatSystemRoom{
					ID:          id,
					Name:        "Announcements",
					Description: "system room",
					SystemKind:  "announcements",
					CreatedBy:   host.ID,
				})
				require.NoError(t, err)

				return spec.ChatRoomViewer{RoomID: id, ViewerID: host.ID}
			},
			exists: true,
		},
		{
			name: "a missing row",
			setup: func(t *testing.T, repos *repository.Repositories) spec.ChatRoomViewer {
				viewer := daotest.CreateUser(t, repos)

				return spec.ChatRoomViewer{RoomID: uuid.New(), ViewerID: viewer.ID}
			},
			exists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			target := tc.setup(t, repos)

			before, err := repos.Chat.GetRoomByID(ctx, target)
			require.NoError(t, err)
			require.Equal(t, tc.exists, before != nil)

			// when
			err = repos.Chat.UpdateRoom(ctx, spec.UpdateChatRoom{
				RoomID:      target.RoomID,
				Name:        "Hijacked",
				Description: "hijacked",
				IsPublic:    true,
				IsRP:        true,
			})

			// then
			require.ErrorIs(t, err, dao.ErrNotFound)

			after, err := repos.Chat.GetRoomByID(ctx, target)
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}

func TestChatDAO_DeleteRoomWithMessages_ReturnsEveryOrphanedFile(t *testing.T) {
	// given a room whose messages carry media and whose members carry room avatars
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	host := daotest.CreateUser(t, repos)
	guest := daotest.CreateUser(t, repos)
	bare := daotest.CreateUser(t, repos)
	roomID := daotest.CreateChatRoom(t, repos, host.ID, daotest.WithRoomMembers(host.ID, guest.ID, bare.ID))

	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, spec.ChatMemberAvatarUpdate{RoomID: roomID, UserID: host.ID, AvatarURL: "/uploads/chat-avatars/host.webp"}))
	require.NoError(t, repos.Chat.SetMemberAvatar(ctx, spec.ChatMemberAvatarUpdate{RoomID: roomID, UserID: guest.ID, AvatarURL: "/uploads/chat-avatars/guest.webp"}))

	first := daotest.SendChatMessage(t, repos, roomID, host.ID, "look")
	_, err := repos.Chat.AddMessageMedia(ctx, spec.NewChatMessageMedia{NewMedia: spec.NewMedia{TargetID: first, MediaURL: "/uploads/chat/one.webp", MediaType: "image", ThumbnailURL: "/uploads/chat/one_thumb.webp", SortOrder: 0}})
	require.NoError(t, err)

	second := daotest.SendChatMessage(t, repos, roomID, guest.ID, "clip")
	_, err = repos.Chat.AddMessageMedia(ctx, spec.NewChatMessageMedia{NewMedia: spec.NewMedia{TargetID: second, MediaURL: "/uploads/chat/two.mp4", MediaType: "video", ThumbnailURL: "", SortOrder: 0}})
	require.NoError(t, err)

	daotest.SendChatMessage(t, repos, roomID, bare.ID, "bye")

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

	row, err := repos.Chat.GetRoomByID(ctx, spec.ChatRoomViewer{RoomID: roomID, ViewerID: host.ID})
	require.NoError(t, err)
	assert.Nil(t, row)
}

func TestChatDAO_AddRoomTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		add      []string
		want     []string
	}{
		{
			name: "adds every tag",
			add:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "no tags is a no-op",
			add:  nil,
			want: nil,
		},
		{
			name: "skips empty strings",
			add:  []string{"valid", "", "also"},
			want: []string{"valid", "also"},
		},
		{
			name:     "is idempotent",
			existing: []string{"x"},
			add:      []string{"x", "y"},
			want:     []string{"x", "y"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			ctx := context.Background()
			owner := daotest.CreateUser(t, repos)
			roomID := daotest.CreateChatRoom(t, repos, owner.ID)

			if len(tc.existing) > 0 {
				require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: roomID, Tags: tc.existing}))
			}

			// when
			err := repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: roomID, Tags: tc.add})

			// then
			require.NoError(t, err)

			tags, err := repos.Chat.GetRoomTags(ctx, roomID)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, tags)
		})
	}
}

func TestChatDAO_GetRoomTagsBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	owner := daotest.CreateUser(t, repos)
	room1 := daotest.CreateChatRoom(t, repos, owner.ID, daotest.WithRoomName("r1"))
	room2 := daotest.CreateChatRoom(t, repos, owner.ID, daotest.WithRoomName("r2"))

	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: room1, Tags: []string{"t1", "t2"}}))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: room2, Tags: []string{"t3"}}))

	// when
	got, err := repos.Chat.GetRoomTagsBatch(ctx, []uuid.UUID{room1, room2})

	// then
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID][]string{
		room1: {"t1", "t2"},
		room2: {"t3"},
	}, got)

	// when no room ids are given
	empty, err := repos.Chat.GetRoomTagsBatch(ctx, nil)

	// then
	require.NoError(t, err)
	assert.Empty(t, empty)
}
