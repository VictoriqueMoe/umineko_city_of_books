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

func TestChatDAO_GetRoomsByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	stranger := daotest.CreateUser(t, repos)

	sysID := uuid.New()
	_, err := repos.Chat.CreateSystemRoom(ctx, spec.NewChatSystemRoom{ID: sysID, Name: "Sys", Description: "", SystemKind: "announcements", CreatedBy: user.ID})
	require.NoError(t, err)
	require.NoError(t, repos.Chat.AddMember(ctx, spec.ChatMemberRef{RoomID: sysID, UserID: user.ID}))
	_, err = repos.DB().ExecContext(ctx, `UPDATE chat_rooms SET created_at = NOW() - INTERVAL '1 day' WHERE id = $1`, sysID)
	require.NoError(t, err)

	mine := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomName("Mine"), daotest.WithRoomMembers(user.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: mine, Tags: []string{"lore"}}))

	daotest.CreateChatRoom(t, repos, other.ID, daotest.WithRoomMembers(other.ID))

	// when
	rooms, err := repos.Chat.GetRoomsByUser(ctx, user.ID)
	strangerRooms, strangerErr := repos.Chat.GetRoomsByUser(ctx, stranger.ID)

	// then
	require.NoError(t, err)
	require.Len(t, rooms, 2)

	assert.Equal(t, sysID, rooms[0].ID)
	assert.True(t, rooms[0].IsSystem)
	assert.Empty(t, rooms[0].Tags)
	assert.True(t, rooms[0].IsMember)

	assert.Equal(t, mine, rooms[1].ID)
	assert.ElementsMatch(t, []string{"lore"}, rooms[1].Tags)
	assert.True(t, rooms[1].IsMember)

	require.NoError(t, strangerErr)
	assert.Empty(t, strangerRooms)
}

func TestChatDAO_ListUserGroupRooms(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)

	apples := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomName("Apples"))
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: apples, UserID: user.ID, Role: "host", Ghost: false}))

	bananas := daotest.CreateChatRoom(t, repos, other.ID, daotest.WithRoomName("Bananas"), daotest.WithRPRoom())
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: bananas, UserID: user.ID, Role: "member", Ghost: false}))

	cherries := daotest.CreateChatRoom(t, repos, user.ID, daotest.WithRoomName("Cherries"), daotest.WithRoomMembers(user.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: cherries, Tags: []string{"lore"}}))

	elsewhere := daotest.CreateChatRoom(t, repos, other.ID, daotest.WithRoomName("Apples Elsewhere"), daotest.WithRPRoom())
	require.NoError(t, repos.Chat.AddMemberWithRole(ctx, spec.NewChatRoomMember{RoomID: elsewhere, UserID: other.ID, Role: "host", Ghost: false}))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: elsewhere, Tags: []string{"lore"}}))

	tests := []struct {
		name      string
		search    string
		rpOnly    bool
		tag       string
		role      string
		want      []uuid.UUID
		wantTotal int
	}{
		{
			name:      "no filter lists every group room the user belongs to",
			want:      []uuid.UUID{apples, bananas, cherries},
			wantTotal: 3,
		},
		{
			name:      "search matches the room name",
			search:    "Apple",
			want:      []uuid.UUID{apples},
			wantTotal: 1,
		},
		{
			name:      "rp only",
			rpOnly:    true,
			want:      []uuid.UUID{bananas},
			wantTotal: 1,
		},
		{
			name:      "tag",
			tag:       "lore",
			want:      []uuid.UUID{cherries},
			wantTotal: 1,
		},
		{
			name:      "host role",
			role:      "host",
			want:      []uuid.UUID{apples},
			wantTotal: 1,
		},
		{
			name:      "member role covers explicit and default member rows",
			role:      "member",
			want:      []uuid.UUID{bananas, cherries},
			wantTotal: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, spec.ChatUserRoomFilter{ChatRoomFilter: spec.ChatRoomFilter{Search: tc.search, IsRPOnly: tc.rpOnly, Tag: tc.tag, Limit: 20}, UserID: user.ID, Role: tc.role})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)

			ids := make([]uuid.UUID, 0, len(rooms))
			for _, room := range rooms {
				ids = append(ids, room.ID)
				assert.True(t, room.IsMember)
			}
			assert.ElementsMatch(t, tc.want, ids)
		})
	}

	t.Run("limit caps the page but not the total", func(t *testing.T) {
		// when
		rooms, total, err := repos.Chat.ListUserGroupRooms(ctx, spec.ChatUserRoomFilter{ChatRoomFilter: spec.ChatRoomFilter{Limit: 2}, UserID: user.ID})

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		require.Len(t, rooms, 2)

		ids := make([]uuid.UUID, 0, len(rooms))
		for _, room := range rooms {
			ids = append(ids, room.ID)
		}
		assert.Subset(t, []uuid.UUID{apples, bananas, cherries}, ids)
	})
}

func TestChatDAO_ListPublicRooms(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	ownerA := daotest.CreateUser(t, repos)
	ownerB := daotest.CreateUser(t, repos)
	viewer := daotest.CreateUser(t, repos)

	apples := daotest.CreateChatRoom(t, repos, ownerA.ID, daotest.WithRoomName("Apples"), daotest.WithPublicRoom())
	rp := daotest.CreateChatRoom(t, repos, ownerB.ID, daotest.WithRoomName("Bananas"), daotest.WithPublicRoom(), daotest.WithRPRoom())

	tagged := daotest.CreateChatRoom(t, repos, ownerB.ID, daotest.WithRoomName("Cherries"), daotest.WithPublicRoom())
	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: tagged, Tags: []string{"lore"}}))

	joined := daotest.CreateChatRoom(t, repos, ownerA.ID, daotest.WithRoomName("Apple Club"), daotest.WithPublicRoom(), daotest.WithRPRoom(), daotest.WithRoomMembers(viewer.ID))
	require.NoError(t, repos.Chat.AddRoomTags(ctx, spec.ChatRoomTags{RoomID: joined, Tags: []string{"lore"}}))

	daotest.CreateChatRoom(t, repos, ownerA.ID, daotest.WithRoomName("Private"))

	sysID := uuid.New()
	_, err := repos.Chat.CreateSystemRoom(ctx, spec.NewChatSystemRoom{ID: sysID, Name: "Sys", Description: "", SystemKind: "announcements", CreatedBy: ownerA.ID})
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx, `UPDATE chat_rooms SET is_public = TRUE WHERE id = $1`, sysID)
	require.NoError(t, err)

	tests := []struct {
		name      string
		search    string
		rpOnly    bool
		tag       string
		anonymous bool
		exclude   []uuid.UUID
		want      []uuid.UUID
		wantTotal int
	}{
		{
			name:      "lists public rooms the viewer has not joined",
			want:      []uuid.UUID{apples, rp, tagged},
			wantTotal: 3,
		},
		{
			name:      "search",
			search:    "Apple",
			want:      []uuid.UUID{apples},
			wantTotal: 1,
		},
		{
			name:      "rp only",
			rpOnly:    true,
			want:      []uuid.UUID{rp},
			wantTotal: 1,
		},
		{
			name:      "tag",
			tag:       "lore",
			want:      []uuid.UUID{tagged},
			wantTotal: 1,
		},
		{
			name:      "exclude creators",
			exclude:   []uuid.UUID{ownerA.ID},
			want:      []uuid.UUID{rp, tagged},
			wantTotal: 2,
		},
		{
			name:      "anonymous viewer skips the membership exclusion",
			anonymous: true,
			want:      []uuid.UUID{apples, rp, tagged, joined},
			wantTotal: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			viewerID := viewer.ID
			if tc.anonymous {
				viewerID = uuid.Nil
			}

			// when
			rooms, total, err := repos.Chat.ListPublicRooms(ctx, spec.ChatPublicRoomFilter{ChatRoomFilter: spec.ChatRoomFilter{Search: tc.search, IsRPOnly: tc.rpOnly, Tag: tc.tag, Limit: 20}, ViewerID: viewerID, ExcludeUserIDs: tc.exclude})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, total)

			ids := make([]uuid.UUID, 0, len(rooms))
			for _, room := range rooms {
				ids = append(ids, room.ID)
				assert.False(t, room.IsMember)
			}
			assert.ElementsMatch(t, tc.want, ids)
		})
	}

	t.Run("limit caps the page but not the total", func(t *testing.T) {
		// when
		rooms, total, err := repos.Chat.ListPublicRooms(ctx, spec.ChatPublicRoomFilter{ChatRoomFilter: spec.ChatRoomFilter{Limit: 2}, ViewerID: viewer.ID})

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		require.Len(t, rooms, 2)

		ids := make([]uuid.UUID, 0, len(rooms))
		for _, room := range rooms {
			ids = append(ids, room.ID)
		}
		assert.Subset(t, []uuid.UUID{apples, rp, tagged}, ids)
	})
}
