package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationDAO_Create(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	refID := uuid.New()

	// when
	created, err := repos.Notification.Create(context.Background(), user.ID, dto.NotifTheoryUpvote, refID, "theory", actor.ID, "Liked your theory")

	// then
	require.NoError(t, err)
	assert.Greater(t, created.ID, 0)
}

func TestNotificationDAO_ListByUser_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	rows, total, err := repos.Notification.ListByUser(context.Background(), user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestNotificationDAO_ListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos, daotest.WithUsername("actor_user"), daotest.WithDisplayName("Actor"))
	refID := uuid.New()
	ctx := context.Background()
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, refID, "theory", actor.ID, "Mentioned you")
	require.NoError(t, err)

	// when
	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	row := rows[0]
	assert.Equal(t, user.ID, row.UserID)
	assert.Equal(t, dto.NotifMention, row.Type)
	assert.Equal(t, refID, row.ReferenceID)
	assert.Equal(t, "theory", row.ReferenceType)
	assert.Equal(t, actor.ID, row.ActorID)
	assert.Equal(t, "Mentioned you", row.Message)
	assert.False(t, row.Read)
	assert.Equal(t, "actor_user", row.ActorUsername)
	assert.Equal(t, "Actor", row.ActorDisplayName)
	assert.NotEmpty(t, row.CreatedAt)
}

func TestNotificationDAO_ListByUser_FiltersByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "for user")
	require.NoError(t, err)
	_, err = repos.Notification.Create(ctx, other.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "for other")
	require.NoError(t, err)

	// when
	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, "for user", rows[0].Message)
}

func TestNotificationDAO_ListByUser_OrderedDesc(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	ids := make([]int, 3)
	for i := range 3 {
		created, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
		require.NoError(t, err)
		ids[i] = created.ID
	}

	// when
	rows, _, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 3)
	gotIDs := []int{rows[0].ID, rows[1].ID, rows[2].ID}
	assert.Contains(t, gotIDs, int(ids[0]))
	assert.Contains(t, gotIDs, int(ids[1]))
	assert.Contains(t, gotIDs, int(ids[2]))
	for i := 0; i < len(rows)-1; i++ {
		assert.GreaterOrEqual(t, rows[i].CreatedAt, rows[i+1].CreatedAt)
	}
}

func TestNotificationDAO_ListByUser_Pagination(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	for range 5 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
		require.NoError(t, err)
	}

	// when
	page1, total, err := repos.Notification.ListByUser(ctx, user.ID, 2, 0)
	require.NoError(t, err)
	page2, _, err := repos.Notification.ListByUser(ctx, user.ID, 2, 2)
	require.NoError(t, err)
	page3, _, err := repos.Notification.ListByUser(ctx, user.ID, 2, 4)
	require.NoError(t, err)

	// then
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.Len(t, page3, 1)
	seen := map[int]bool{}
	for _, r := range append(append(page1, page2...), page3...) {
		assert.False(t, seen[r.ID])
		seen[r.ID] = true
	}
}

func TestNotificationDAO_MarkRead(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	created, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
	require.NoError(t, err)
	id := created.ID

	// when
	err = repos.Notification.MarkRead(ctx, int(id), user.ID)

	// then
	require.NoError(t, err)
	rows, _, listErr := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, listErr)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].Read)
}

func TestNotificationDAO_MarkRead_OnlyOwner(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	created, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
	require.NoError(t, err)
	id := created.ID

	// when
	err = repos.Notification.MarkRead(ctx, int(id), other.ID)

	// then
	require.NoError(t, err)
	rows, _, listErr := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, listErr)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Read)
}

func TestNotificationDAO_DeleteOlderThanBatch_DeletesOnlyOlder(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	for range 3 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "m")
		require.NoError(t, err)
	}

	// when: a cutoff in the past leaves the fresh rows untouched
	deleted, err := repos.Notification.DeleteOlderThanBatch(ctx, time.Now().Add(-time.Hour), 100)

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	remaining, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, remaining)

	// when: a cutoff in the future treats every row as old
	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, time.Now().Add(time.Hour), 100)

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	remaining, err = repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}

func TestNotificationDAO_DeleteOlderThanBatch_RespectsLimit(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	for range 5 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "m")
		require.NoError(t, err)
	}
	future := time.Now().Add(time.Hour)

	// when: a limit smaller than the backlog deletes only up to the limit
	deleted, err := repos.Notification.DeleteOlderThanBatch(ctx, future, 2)

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// when: repeated batches drain the remainder, the last one short of the limit
	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, future, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, future, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// then: nothing remains
	remaining, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}

func TestNotificationDAO_MarkAllRead(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	for range 3 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "u")
		require.NoError(t, err)
	}
	_, err := repos.Notification.Create(ctx, other.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "o")
	require.NoError(t, err)

	// when
	err = repos.Notification.MarkAllRead(ctx, user.ID)

	// then
	require.NoError(t, err)
	userUnread, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, userUnread)
	otherUnread, err := repos.Notification.UnreadCount(ctx, other.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, otherUnread)
}

func TestNotificationDAO_UnreadCount(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	id1Row, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "a")
	require.NoError(t, err)
	id1 := id1Row.ID
	_, err = repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "b")
	require.NoError(t, err)
	_, err = repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "c")
	require.NoError(t, err)
	require.NoError(t, repos.Notification.MarkRead(ctx, int(id1), user.ID))

	// when
	count, err := repos.Notification.UnreadCount(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestNotificationDAO_UnreadCount_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	count, err := repos.Notification.UnreadCount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestNotificationDAO_HasRecentDuplicate_True(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	refID := uuid.New()
	ctx := context.Background()
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, refID, "theory", actor.ID, "msg")
	require.NoError(t, err)

	// when
	exists, err := repos.Notification.HasRecentDuplicate(ctx, user.ID, dto.NotifMention, refID, actor.ID)

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestNotificationDAO_HasRecentDuplicate_False(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()

	// when
	exists, err := repos.Notification.HasRecentDuplicate(ctx, user.ID, dto.NotifMention, uuid.New(), actor.ID)

	// then
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNotificationDAO_HasRecentDuplicate_DifferentTypeNotMatched(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	refID := uuid.New()
	ctx := context.Background()
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, refID, "theory", actor.ID, "msg")
	require.NoError(t, err)

	// when
	exists, err := repos.Notification.HasRecentDuplicate(ctx, user.ID, dto.NotifPostLiked, refID, actor.ID)

	// then
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNotificationDAO_HasRecentDuplicate_DifferentActorNotMatched(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	otherActor := daotest.CreateUser(t, repos)
	refID := uuid.New()
	ctx := context.Background()
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, refID, "theory", actor.ID, "msg")
	require.NoError(t, err)

	// when
	exists, err := repos.Notification.HasRecentDuplicate(ctx, user.ID, dto.NotifMention, refID, otherActor.ID)

	// then
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNotificationDAO_ListByUser_GroupsUnreadChatRoomMessages(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"), daotest.WithDisplayName("Bob"))
	roomID := uuid.New()
	ctx := context.Background()

	for i := range 5 {
		actor := alice.ID
		if i%2 == 1 {
			actor = bob.ID
		}
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomID, "chat_message:x", actor, "sent a message in General Chat")
		require.NoError(t, err)
	}

	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)

	assert.Equal(t, 1, total, "5 chat messages for one room should collapse to 1 row")
	require.Len(t, rows, 1)
	assert.Equal(t, dto.NotifChatRoomMessage, rows[0].Type)
	assert.Equal(t, 5, rows[0].Count)
	assert.Equal(t, "5 messages sent in General Chat", rows[0].Message)
}

func TestNotificationDAO_ListByUser_DifferentRoomsNotCollapsed(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomA := uuid.New()
	roomB := uuid.New()
	ctx := context.Background()

	for range 3 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomA, "chat_message:x", actor.ID, "sent a message in Room A")
		require.NoError(t, err)
	}
	for range 2 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomB, "chat_message:x", actor.ID, "sent a message in Room B")
		require.NoError(t, err)
	}

	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	require.Len(t, rows, 2)
	byRoom := map[uuid.UUID]int{}
	for _, r := range rows {
		byRoom[r.ReferenceID] = r.Count
	}
	assert.Equal(t, 3, byRoom[roomA])
	assert.Equal(t, 2, byRoom[roomB])
}

func TestNotificationDAO_ListByUser_ReadChatMessagesNotGrouped(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	ctx := context.Background()

	ids := make([]int, 3)
	for i := range 3 {
		created, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomID, "chat_message:x", actor.ID, "sent a message in General")
		require.NoError(t, err)
		ids[i] = created.ID
	}

	require.NoError(t, repos.Notification.MarkAllRead(ctx, user.ID))

	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)

	assert.Equal(t, 3, total, "read chat_room_message rows should be shown individually")
	assert.Len(t, rows, 3)
	for _, r := range rows {
		assert.Equal(t, 1, r.Count)
		assert.True(t, r.Read)
	}
}

func TestNotificationDAO_ListByUser_MixedTypesPreservesNonChat(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	ctx := context.Background()

	for range 3 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomID, "chat_message:x", actor.ID, "sent a message in General")
		require.NoError(t, err)
	}
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "Mentioned you")
	require.NoError(t, err)
	_, err = repos.Notification.Create(ctx, user.ID, dto.NotifPostLiked, uuid.New(), "post", actor.ID, "")
	require.NoError(t, err)

	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)

	assert.Equal(t, 3, total, "1 grouped chat + 2 individual = 3 rows")
	require.Len(t, rows, 3)

	typeCount := map[dto.NotificationType]int{}
	for _, r := range rows {
		typeCount[r.Type]++
	}
	assert.Equal(t, 1, typeCount[dto.NotifChatRoomMessage])
	assert.Equal(t, 1, typeCount[dto.NotifMention])
	assert.Equal(t, 1, typeCount[dto.NotifPostLiked])
}

func TestNotificationDAO_UnreadCount_SumsRawRowsNotGroups(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	ctx := context.Background()

	for range 50 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomID, "chat_message:x", actor.ID, "sent a message in General")
		require.NoError(t, err)
	}

	count, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, count, "badge counter must reflect raw row count, not grouped count")
}

func TestNotificationDAO_MarkRead_ChatRoomMessageMarksEntireGroup(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	ctx := context.Background()

	ids := make([]int, 4)
	for i := range 4 {
		created, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomID, "chat_message:x", actor.ID, "sent a message in General")
		require.NoError(t, err)
		ids[i] = created.ID
	}

	rows, _, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	representative := rows[0].ID

	require.NoError(t, repos.Notification.MarkRead(ctx, representative, user.ID))

	remaining, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining, "marking the grouped row should mark all 4 underlying rows")
}

func TestNotificationDAO_MarkRead_DifferentRoomUnaffected(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomA := uuid.New()
	roomB := uuid.New()
	ctx := context.Background()

	for range 3 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomA, "chat_message:x", actor.ID, "sent a message in A")
		require.NoError(t, err)
	}
	for range 2 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomMessage, roomB, "chat_message:x", actor.ID, "sent a message in B")
		require.NoError(t, err)
	}

	rows, _, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	var roomARowID int
	for _, r := range rows {
		if r.ReferenceID == roomA {
			roomARowID = r.ID
		}
	}
	require.NotZero(t, roomARowID)

	require.NoError(t, repos.Notification.MarkRead(ctx, roomARowID, user.ID))

	remaining, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, remaining, "only room A should be marked read, room B's 2 messages remain unread")
}

func TestNotificationDAO_ListByUser_CollapsesUnreadDirectMessages(t *testing.T) {
	tests := []struct {
		name        string
		messages    int
		wantCount   int
		wantMessage string
	}{
		{name: "two unread dms collapse", messages: 2, wantCount: 2, wantMessage: "has sent you 2 messages"},
		{name: "seven unread dms collapse", messages: 7, wantCount: 7, wantMessage: "has sent you 7 messages"},
		{name: "a lone dm keeps its stored message", messages: 1, wantCount: 1, wantMessage: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
			dmRoom := uuid.New()
			ctx := context.Background()

			for range tt.messages {
				_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatMessage, dmRoom, "chat", alice.ID, "")
				require.NoError(t, err)
			}

			// when
			rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)

			// then
			require.NoError(t, err)
			assert.Equal(t, 1, total, "one conversation is one row")
			require.Len(t, rows, 1)
			assert.Equal(t, dto.NotifChatMessage, rows[0].Type)
			assert.Equal(t, dmRoom, rows[0].ReferenceID)
			assert.Equal(t, tt.wantCount, rows[0].Count)
			assert.Equal(t, tt.wantMessage, rows[0].Message)
			assert.Equal(t, alice.ID, rows[0].ActorID, "the collapsed row keeps the latest sender as its actor")
			assert.Equal(t, "Alice", rows[0].ActorDisplayName)
		})
	}
}

func TestNotificationDAO_ListByUser_DirectMessagesCollapsePerConversation(t *testing.T) {
	tests := []struct {
		name         string
		otherType    dto.NotificationType
		otherMessage string
		sameRoom     bool
		wantMessages []string
	}{
		{
			name:         "a second sender is a separate conversation",
			otherType:    dto.NotifChatMessage,
			otherMessage: "",
			sameRoom:     false,
			wantMessages: []string{"has sent you 2 messages", "has sent you 3 messages"},
		},
		{
			name:         "the same reference under the room type is a separate group",
			otherType:    dto.NotifChatRoomMessage,
			otherMessage: "sent a message in General Chat",
			sameRoom:     true,
			wantMessages: []string{"has sent you 2 messages", "3 messages sent in General Chat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
			bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"), daotest.WithDisplayName("Bob"))
			dmRoom := uuid.New()
			otherRoom := uuid.New()
			if tt.sameRoom {
				otherRoom = dmRoom
			}
			ctx := context.Background()

			for range 2 {
				_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatMessage, dmRoom, "chat", alice.ID, "")
				require.NoError(t, err)
			}
			for range 3 {
				_, err := repos.Notification.Create(ctx, user.ID, tt.otherType, otherRoom, "chat", bob.ID, tt.otherMessage)
				require.NoError(t, err)
			}

			// when
			rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)

			// then
			require.NoError(t, err)
			assert.Equal(t, 2, total, "the two groups must not merge into one row")
			require.Len(t, rows, 2)
			gotMessages := make([]string, 0, len(rows))
			for _, r := range rows {
				gotMessages = append(gotMessages, r.Message)
			}
			assert.ElementsMatch(t, tt.wantMessages, gotMessages)
		})
	}
}

func TestNotificationDAO_ListByUser_ReadDirectMessagesNotGrouped(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
	dmRoom := uuid.New()
	ctx := context.Background()

	for range 3 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatMessage, dmRoom, "chat", alice.ID, "")
		require.NoError(t, err)
	}
	require.NoError(t, repos.Notification.MarkAllRead(ctx, user.ID))

	for range 2 {
		_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatMessage, dmRoom, "chat", alice.ID, "")
		require.NoError(t, err)
	}

	// when
	rows, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, total, "3 read rows stay individual, the 2 unread ones collapse to 1")
	require.Len(t, rows, 4)

	var readRows, unreadRows int
	for _, r := range rows {
		if r.Read {
			readRows++
			assert.Equal(t, 1, r.Count)
			assert.Empty(t, r.Message, "a read dm row keeps its stored message")
			continue
		}
		unreadRows++
		assert.Equal(t, 2, r.Count)
		assert.Equal(t, "has sent you 2 messages", r.Message)
	}
	assert.Equal(t, 3, readRows)
	assert.Equal(t, 1, unreadRows)
}

func TestNotificationDAO_MarkRead_DirectMessageMarksEntireConversation(t *testing.T) {
	tests := []struct {
		name         string
		otherType    dto.NotificationType
		otherMessage string
		sameRoom     bool
	}{
		{
			name:         "another conversation is untouched",
			otherType:    dto.NotifChatMessage,
			otherMessage: "",
			sameRoom:     false,
		},
		{
			name:         "the same reference under the room type is untouched",
			otherType:    dto.NotifChatRoomMessage,
			otherMessage: "sent a message in General Chat",
			sameRoom:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
			bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"), daotest.WithDisplayName("Bob"))
			dmRoom := uuid.New()
			otherRoom := uuid.New()
			if tt.sameRoom {
				otherRoom = dmRoom
			}
			ctx := context.Background()

			for range 4 {
				_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatMessage, dmRoom, "chat", alice.ID, "")
				require.NoError(t, err)
			}
			for range 3 {
				_, err := repos.Notification.Create(ctx, user.ID, tt.otherType, otherRoom, "chat", bob.ID, tt.otherMessage)
				require.NoError(t, err)
			}

			rows, _, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
			require.NoError(t, err)
			require.Len(t, rows, 2)

			var dmRowID int
			for _, r := range rows {
				if r.Type == dto.NotifChatMessage && r.ReferenceID == dmRoom {
					dmRowID = r.ID
				}
			}
			require.NotZero(t, dmRowID)

			// when
			require.NoError(t, repos.Notification.MarkRead(ctx, dmRowID, user.ID))

			// then
			remaining, err := repos.Notification.UnreadCount(ctx, user.ID)
			require.NoError(t, err)
			assert.Equal(t, 3, remaining, "one click clears all 4 dms and leaves the other group alone")

			after, total, err := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
			require.NoError(t, err)
			assert.Equal(t, 5, total, "4 read dm rows plus 1 grouped unread row")
			require.Len(t, after, 5)

			var survivors int
			for _, r := range after {
				if r.Read {
					assert.Equal(t, dto.NotifChatMessage, r.Type)
					continue
				}
				survivors++
				assert.Equal(t, tt.otherType, r.Type)
				assert.Equal(t, 3, r.Count)
			}
			assert.Equal(t, 1, survivors)
		})
	}
}

func TestNotificationDAO_MarkReadByReference(t *testing.T) {
	// given one thread's four chat notification types, an invite for the same thread and a mention from another thread
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	roomID := uuid.New()
	otherRoomID := uuid.New()

	chatTypes := []dto.NotificationType{dto.NotifChatMessage, dto.NotifChatRoomMessage, dto.NotifChatMention, dto.NotifChatReply}
	for _, notifType := range chatTypes {
		_, err := repos.Notification.Create(ctx, user.ID, notifType, roomID, "chat_message:x", actor.ID, "")
		require.NoError(t, err)
	}
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatRoomInvite, roomID, "chat_room", actor.ID, "")
	require.NoError(t, err)
	_, err = repos.Notification.Create(ctx, user.ID, dto.NotifChatMention, otherRoomID, "chat_message:y", actor.ID, "")
	require.NoError(t, err)

	// when the thread is marked read
	err = repos.Notification.MarkReadByReference(ctx, user.ID, roomID, chatTypes)

	// then only that thread's message notifications clear, and the invite and the other thread survive
	require.NoError(t, err)
	unread, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, unread)
}

func TestNotificationDAO_MarkReadByReference_OnlyOwnerAndNeverEmptyTypes(t *testing.T) {
	// given two people each holding a chat notification for the same thread
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ctx := context.Background()
	roomID := uuid.New()
	_, err := repos.Notification.Create(ctx, user.ID, dto.NotifChatMessage, roomID, "chat_message:x", actor.ID, "")
	require.NoError(t, err)
	_, err = repos.Notification.Create(ctx, other.ID, dto.NotifChatMessage, roomID, "chat_message:x", actor.ID, "")
	require.NoError(t, err)

	// when one of them marks the thread read, and when the type list is empty
	require.NoError(t, repos.Notification.MarkReadByReference(ctx, user.ID, roomID, []dto.NotificationType{dto.NotifChatMessage}))
	require.NoError(t, repos.Notification.MarkReadByReference(ctx, other.ID, roomID, nil))

	// then only the caller's row clears and an empty type list is a no-op rather than a wildcard
	mine, err := repos.Notification.UnreadCount(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, mine)
	theirs, err := repos.Notification.UnreadCount(ctx, other.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, theirs)
}
