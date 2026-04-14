package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationRepository_Create(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	refID := uuid.New()

	// when
	id, err := repos.Notification.Create(context.Background(), user.ID, dto.NotifTheoryUpvote, refID, "theory", actor.ID, "Liked your theory")

	// then
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
}

func TestNotificationRepository_ListByUser_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	rows, total, err := repos.Notification.ListByUser(context.Background(), user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}

func TestNotificationRepository_ListByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos, repotest.WithUsername("actor_user"), repotest.WithDisplayName("Actor"))
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

func TestNotificationRepository_ListByUser_FiltersByUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
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

func TestNotificationRepository_ListByUser_OrderedDesc(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()
	ids := make([]int64, 3)
	for i := 0; i < 3; i++ {
		id, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
		require.NoError(t, err)
		ids[i] = id
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

func TestNotificationRepository_ListByUser_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
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

func TestNotificationRepository_MarkRead(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
	require.NoError(t, err)

	// when
	err = repos.Notification.MarkRead(ctx, int(id), user.ID)

	// then
	require.NoError(t, err)
	rows, _, listErr := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, listErr)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].Read)
}

func TestNotificationRepository_MarkRead_OnlyOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "msg")
	require.NoError(t, err)

	// when
	err = repos.Notification.MarkRead(ctx, int(id), other.ID)

	// then
	require.NoError(t, err)
	rows, _, listErr := repos.Notification.ListByUser(ctx, user.ID, 10, 0)
	require.NoError(t, listErr)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Read)
}

func TestNotificationRepository_MarkAllRead(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
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

func TestNotificationRepository_UnreadCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()
	id1, err := repos.Notification.Create(ctx, user.ID, dto.NotifMention, uuid.New(), "theory", actor.ID, "a")
	require.NoError(t, err)
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

func TestNotificationRepository_UnreadCount_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	count, err := repos.Notification.UnreadCount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestNotificationRepository_HasRecentDuplicate_True(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
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

func TestNotificationRepository_HasRecentDuplicate_False(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	ctx := context.Background()

	// when
	exists, err := repos.Notification.HasRecentDuplicate(ctx, user.ID, dto.NotifMention, uuid.New(), actor.ID)

	// then
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNotificationRepository_HasRecentDuplicate_DifferentTypeNotMatched(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
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

func TestNotificationRepository_HasRecentDuplicate_DifferentActorNotMatched(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	actor := repotest.CreateUser(t, repos)
	otherActor := repotest.CreateUser(t, repos)
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
