package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	notificationSummary struct {
		Type        dto.NotificationType
		ReferenceID uuid.UUID
		Count       int
		Message     string
		Read        bool
	}
)

func notificationMention(userID, actorID uuid.UUID) spec.NewNotification {
	return spec.NewNotification{UserID: userID, Type: dto.NotifMention, ReferenceID: uuid.New(), ReferenceType: "theory", ActorID: actorID, Message: "msg"}
}

func notificationSeed(t *testing.T, repos *repository.Repositories, count int, s spec.NewNotification) []int {
	t.Helper()
	ids := make([]int, 0, count)
	for range count {
		created, err := repos.Notification.Create(context.Background(), s)
		require.NoError(t, err)
		ids = append(ids, created.ID)
	}

	return ids
}

func notificationBackdate(t *testing.T, repos *repository.Repositories, id int, createdAt string) {
	t.Helper()
	_, err := repos.DB().ExecContext(context.Background(), `UPDATE notifications SET created_at = $1 WHERE id = $2`, createdAt, id)
	require.NoError(t, err)
}

func notificationList(t *testing.T, repos *repository.Repositories, userID uuid.UUID) ([]model.NotificationRow, int) {
	t.Helper()
	rows, total, err := repos.Notification.ListByUser(context.Background(), spec.NotificationListing{UserID: userID, Limit: 10, Offset: 0})
	require.NoError(t, err)

	return rows, total
}

func notificationUnread(t *testing.T, repos *repository.Repositories, userID uuid.UUID) int {
	t.Helper()
	count, err := repos.Notification.UnreadCount(context.Background(), userID)
	require.NoError(t, err)

	return count
}

func notificationSummaries(rows []model.NotificationRow) []notificationSummary {
	out := make([]notificationSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, notificationSummary{Type: r.Type, ReferenceID: r.ReferenceID, Count: r.Count, Message: r.Message, Read: r.Read})
	}

	return out
}

func TestNotificationDAO_CreateAndListByUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos, daotest.WithUsername("actor_user"), daotest.WithDisplayName("Actor"))
	refID := uuid.New()
	notificationSeed(t, repos, 1, notificationMention(other.ID, actor.ID))

	// when
	created, err := repos.Notification.Create(context.Background(), spec.NewNotification{UserID: user.ID, Type: dto.NotifMention, ReferenceID: refID, ReferenceType: "theory", ActorID: actor.ID, Message: "Mentioned you"})

	// then
	require.NoError(t, err)
	assert.Greater(t, created.ID, 0)

	rows, total := notificationList(t, repos, user.ID)
	assert.Equal(t, 1, total, "another user's notification is never listed")
	require.Len(t, rows, 1)
	row := rows[0]
	assert.Equal(t, created.ID, row.ID)
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

func TestNotificationDAO_ListByUser_NewestFirstAcrossPages(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ids := notificationSeed(t, repos, 5, notificationMention(user.ID, actor.ID))
	createdAts := []string{"2026-01-04", "2026-01-02", "2026-01-05", "2026-01-01", "2026-01-03"}
	for i, id := range ids {
		notificationBackdate(t, repos, id, createdAts[i])
	}

	pages := []struct {
		name   string
		offset int
		want   []int
	}{
		{name: "the first page holds the two newest", offset: 0, want: []int{ids[2], ids[0]}},
		{name: "the second page continues without overlap", offset: 2, want: []int{ids[4], ids[1]}},
		{name: "the last page holds the remainder", offset: 4, want: []int{ids[3]}},
	}

	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			// when
			rows, total, err := repos.Notification.ListByUser(context.Background(), spec.NotificationListing{UserID: user.ID, Limit: 2, Offset: page.offset})

			// then
			require.NoError(t, err)
			assert.Equal(t, 5, total)

			got := make([]int, 0, len(rows))
			for _, r := range rows {
				got = append(got, r.ID)
			}
			assert.Equal(t, page.want, got)
		})
	}
}

func TestNotificationDAO_ListByUser_CollapsesUnreadChatGroups(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	alice := daotest.CreateUser(t, repos, daotest.WithUsername("alice"), daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithUsername("bob"), daotest.WithDisplayName("Bob"))
	generalRoom := uuid.New()
	sharedRef := uuid.New()
	bobDM := uuid.New()
	aliceLoneDM := uuid.New()
	mentionRef := uuid.New()
	likeRef := uuid.New()

	for _, actorID := range []uuid.UUID{alice.ID, bob.ID, alice.ID, bob.ID, alice.ID} {
		notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatRoomMessage, ReferenceID: generalRoom, ReferenceType: "chat_message:x", ActorID: actorID, Message: "sent a message in General Chat"})
	}
	notificationSeed(t, repos, 2, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatRoomMessage, ReferenceID: sharedRef, ReferenceType: "chat_message:x", ActorID: bob.ID, Message: "sent a message in Room B"})
	notificationSeed(t, repos, 2, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatMessage, ReferenceID: sharedRef, ReferenceType: "chat", ActorID: alice.ID})
	notificationSeed(t, repos, 7, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatMessage, ReferenceID: bobDM, ReferenceType: "chat", ActorID: bob.ID})
	notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatMessage, ReferenceID: aliceLoneDM, ReferenceType: "chat", ActorID: alice.ID})
	notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: dto.NotifMention, ReferenceID: mentionRef, ReferenceType: "theory", ActorID: alice.ID, Message: "Mentioned you"})
	notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: dto.NotifPostLiked, ReferenceID: likeRef, ReferenceType: "post", ActorID: alice.ID})

	// when
	rows, total := notificationList(t, repos, user.ID)

	// then
	assert.Equal(t, 7, total, "each unread chat group collapses to one row and every other notification stays individual")
	require.ElementsMatch(t, []notificationSummary{
		{Type: dto.NotifChatRoomMessage, ReferenceID: generalRoom, Count: 5, Message: "5 messages sent in General Chat"},
		{Type: dto.NotifChatRoomMessage, ReferenceID: sharedRef, Count: 2, Message: "2 messages sent in Room B"},
		{Type: dto.NotifChatMessage, ReferenceID: sharedRef, Count: 2, Message: "has sent you 2 messages"},
		{Type: dto.NotifChatMessage, ReferenceID: bobDM, Count: 7, Message: "has sent you 7 messages"},
		{Type: dto.NotifChatMessage, ReferenceID: aliceLoneDM, Count: 1, Message: ""},
		{Type: dto.NotifMention, ReferenceID: mentionRef, Count: 1, Message: "Mentioned you"},
		{Type: dto.NotifPostLiked, ReferenceID: likeRef, Count: 1, Message: ""},
	}, notificationSummaries(rows))

	dmSenders := map[uuid.UUID]*model.User{sharedRef: alice, bobDM: bob, aliceLoneDM: alice}
	for _, r := range rows {
		if r.Type != dto.NotifChatMessage {
			continue
		}
		sender := dmSenders[r.ReferenceID]
		assert.Equal(t, sender.ID, r.ActorID, "the collapsed row keeps the latest sender as its actor")
		assert.Equal(t, sender.DisplayName, r.ActorDisplayName)
	}

	assert.Equal(t, 19, notificationUnread(t, repos, user.ID), "badge counter must reflect raw row count, not grouped count")
}

func TestNotificationDAO_ListByUser_ReadChatRowsNotGrouped(t *testing.T) {
	tests := []struct {
		name        string
		notifType   dto.NotificationType
		message     string
		wantGrouped string
	}{
		{name: "read chat room messages are listed individually beside the unread group", notifType: dto.NotifChatRoomMessage, message: "sent a message in General", wantGrouped: "2 messages sent in General"},
		{name: "read direct messages keep their stored message beside the unread group", notifType: dto.NotifChatMessage, message: "", wantGrouped: "has sent you 2 messages"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			actor := daotest.CreateUser(t, repos)
			refID := uuid.New()
			seed := spec.NewNotification{UserID: user.ID, Type: tc.notifType, ReferenceID: refID, ReferenceType: "chat", ActorID: actor.ID, Message: tc.message}
			notificationSeed(t, repos, 3, seed)
			require.NoError(t, repos.Notification.MarkAllRead(context.Background(), user.ID))
			notificationSeed(t, repos, 2, seed)

			// when
			rows, total := notificationList(t, repos, user.ID)

			// then
			read := notificationSummary{Type: tc.notifType, ReferenceID: refID, Count: 1, Message: tc.message, Read: true}
			unread := notificationSummary{Type: tc.notifType, ReferenceID: refID, Count: 2, Message: tc.wantGrouped, Read: false}
			assert.Equal(t, 4, total, "3 read rows stay individual, the 2 unread ones collapse to 1")
			assert.ElementsMatch(t, []notificationSummary{read, read, read, unread}, notificationSummaries(rows))
		})
	}
}

func TestNotificationDAO_MarkRead(t *testing.T) {
	tests := []struct {
		name       string
		byOwner    bool
		wantUnread int
	}{
		{name: "the owner marks only that notification read", byOwner: true, wantUnread: 2},
		{name: "another user cannot mark someone else's notification read", byOwner: false, wantUnread: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			other := daotest.CreateUser(t, repos)
			ids := notificationSeed(t, repos, 3, notificationMention(user.ID, other.ID))

			caller := other.ID
			if tc.byOwner {
				caller = user.ID
			}

			// when
			err := repos.Notification.MarkRead(context.Background(), spec.NotificationLookup{ID: ids[0], UserID: caller})

			// then
			require.NoError(t, err)

			rows, _ := notificationList(t, repos, user.ID)
			require.Len(t, rows, 3)
			for _, r := range rows {
				assert.Equal(t, tc.byOwner && r.ID == ids[0], r.Read)
			}
			assert.Equal(t, tc.wantUnread, notificationUnread(t, repos, user.ID))
		})
	}
}

func TestNotificationDAO_MarkRead_GroupedRowMarksEntireGroup(t *testing.T) {
	tests := []struct {
		name      string
		markType  dto.NotificationType
		otherType dto.NotificationType
		sameRef   bool
	}{
		{name: "a chat room group clears as a whole and another room is untouched", markType: dto.NotifChatRoomMessage, otherType: dto.NotifChatRoomMessage, sameRef: false},
		{name: "a dm conversation clears as a whole and another conversation is untouched", markType: dto.NotifChatMessage, otherType: dto.NotifChatMessage, sameRef: false},
		{name: "a dm conversation clears as a whole and the same reference under the room type is untouched", markType: dto.NotifChatMessage, otherType: dto.NotifChatRoomMessage, sameRef: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			alice := daotest.CreateUser(t, repos)
			bob := daotest.CreateUser(t, repos)
			markRef := uuid.New()
			otherRef := uuid.New()
			if tc.sameRef {
				otherRef = markRef
			}

			notificationSeed(t, repos, 4, spec.NewNotification{UserID: user.ID, Type: tc.markType, ReferenceID: markRef, ReferenceType: "chat", ActorID: alice.ID})
			notificationSeed(t, repos, 3, spec.NewNotification{UserID: user.ID, Type: tc.otherType, ReferenceID: otherRef, ReferenceType: "chat", ActorID: bob.ID})

			rows, _ := notificationList(t, repos, user.ID)
			require.Len(t, rows, 2)
			var groupRowID int
			for _, r := range rows {
				if r.Type == tc.markType && r.ReferenceID == markRef {
					groupRowID = r.ID
				}
			}
			require.NotZero(t, groupRowID)

			// when
			err := repos.Notification.MarkRead(context.Background(), spec.NotificationLookup{ID: groupRowID, UserID: user.ID})

			// then
			require.NoError(t, err)
			assert.Equal(t, 3, notificationUnread(t, repos, user.ID), "one click clears all 4 and leaves the other group alone")

			after, total := notificationList(t, repos, user.ID)
			assert.Equal(t, 5, total, "4 read rows plus 1 grouped unread row")
			require.Len(t, after, 5)

			var survivors int
			for _, r := range after {
				if r.Read {
					assert.Equal(t, tc.markType, r.Type)
					continue
				}
				survivors++
				assert.Equal(t, tc.otherType, r.Type)
				assert.Equal(t, 3, r.Count)
			}
			assert.Equal(t, 1, survivors)
		})
	}
}

func TestNotificationDAO_MarkAllRead(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	notificationSeed(t, repos, 3, notificationMention(user.ID, other.ID))
	notificationSeed(t, repos, 1, notificationMention(other.ID, user.ID))

	// when
	err := repos.Notification.MarkAllRead(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, notificationUnread(t, repos, user.ID))
	assert.Equal(t, 1, notificationUnread(t, repos, other.ID))
}

func TestNotificationDAO_MarkReadByReference(t *testing.T) {
	// given one thread's four chat notification types, an invite for the same thread and a mention from another thread, and a second person holding a chat notification for the same thread
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	roomID := uuid.New()
	chatTypes := []dto.NotificationType{dto.NotifChatMessage, dto.NotifChatRoomMessage, dto.NotifChatMention, dto.NotifChatReply}

	for _, notifType := range chatTypes {
		notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: notifType, ReferenceID: roomID, ReferenceType: "chat_message:x", ActorID: actor.ID})
	}
	notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatRoomInvite, ReferenceID: roomID, ReferenceType: "chat_room", ActorID: actor.ID})
	notificationSeed(t, repos, 1, spec.NewNotification{UserID: user.ID, Type: dto.NotifChatMention, ReferenceID: uuid.New(), ReferenceType: "chat_message:y", ActorID: actor.ID})
	notificationSeed(t, repos, 1, spec.NewNotification{UserID: other.ID, Type: dto.NotifChatMessage, ReferenceID: roomID, ReferenceType: "chat_message:x", ActorID: actor.ID})

	// when the thread is marked read, and when the second person marks it read with an empty type list
	require.NoError(t, repos.Notification.MarkReadByReference(ctx, spec.NotificationReferenceRead{UserID: user.ID, ReferenceID: roomID, Types: chatTypes}))
	require.NoError(t, repos.Notification.MarkReadByReference(ctx, spec.NotificationReferenceRead{UserID: other.ID, ReferenceID: roomID, Types: nil}))

	// then only the caller's message notifications for that thread clear, the invite and the other thread survive, and an empty type list is a no-op rather than a wildcard
	assert.Equal(t, 2, notificationUnread(t, repos, user.ID))
	assert.Equal(t, 1, notificationUnread(t, repos, other.ID))
}

func TestNotificationDAO_DeleteOlderThanBatch(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	actor := daotest.CreateUser(t, repos)
	ids := notificationSeed(t, repos, 7, notificationMention(user.ID, actor.ID))
	for _, id := range ids[:5] {
		notificationBackdate(t, repos, id, "2020-01-01")
	}
	past := time.Now().Add(-time.Hour)

	// when: a limit smaller than the backlog deletes only up to the limit
	deleted, err := repos.Notification.DeleteOlderThanBatch(ctx, spec.NotificationPruneBatch{Cutoff: past, Limit: 2})

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// when: repeated batches drain the remainder, the last one short of the limit
	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, spec.NotificationPruneBatch{Cutoff: past, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, spec.NotificationPruneBatch{Cutoff: past, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// when: a cutoff in the past leaves the fresh rows untouched
	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, spec.NotificationPruneBatch{Cutoff: past, Limit: 100})

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.Equal(t, 2, notificationUnread(t, repos, user.ID))

	// when: a cutoff in the future treats every row as old
	deleted, err = repos.Notification.DeleteOlderThanBatch(ctx, spec.NotificationPruneBatch{Cutoff: time.Now().Add(time.Hour), Limit: 100})

	// then: nothing remains
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	assert.Equal(t, 0, notificationUnread(t, repos, user.ID))
}

func TestNotificationDAO_HasRecentDuplicate(t *testing.T) {
	tests := []struct {
		name       string
		notifType  dto.NotificationType
		otherRef   bool
		otherActor bool
		want       bool
	}{
		{name: "the same recipient, type, reference and actor within the hour is a duplicate", notifType: dto.NotifMention, otherRef: false, otherActor: false, want: true},
		{name: "a different reference is not a duplicate", notifType: dto.NotifMention, otherRef: true, otherActor: false, want: false},
		{name: "a different type is not a duplicate", notifType: dto.NotifPostLiked, otherRef: false, otherActor: false, want: false},
		{name: "a different actor is not a duplicate", notifType: dto.NotifMention, otherRef: false, otherActor: true, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			repos := daotest.NewRepos(t)
			user := daotest.CreateUser(t, repos)
			actor := daotest.CreateUser(t, repos)
			otherActor := daotest.CreateUser(t, repos)
			seeded := notificationMention(user.ID, actor.ID)
			notificationSeed(t, repos, 1, seeded)

			check := spec.NotificationDuplicateCheck{UserID: user.ID, Type: tc.notifType, ReferenceID: seeded.ReferenceID, ActorID: actor.ID}
			if tc.otherRef {
				check.ReferenceID = uuid.New()
			}
			if tc.otherActor {
				check.ActorID = otherActor.ID
			}

			// when
			exists, err := repos.Notification.HasRecentDuplicate(context.Background(), check)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, exists)
		})
	}
}
