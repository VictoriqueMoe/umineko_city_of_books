package repository_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveStreamRepository_Lifecycle(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Beatrice"))

	// when
	id, err := repo.Create(ctx, user.ID, "My Stream", 3)

	// then
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)

	row, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "starting", row.Status)
	assert.Equal(t, "My Stream", row.Title)
	assert.Equal(t, user.ID, row.UserID)
	assert.Equal(t, "Beatrice", row.DisplayName)

	// then
	n, err := repo.CountActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	active, err := repo.GetActiveByUser(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, id, active.ID)

	// when
	_, err = repo.Create(ctx, user.ID, "Another", 3)

	// then
	require.ErrorIs(t, err, repository.ErrLiveStreamActiveExists)

	// when
	room := "live_" + id.String()
	require.NoError(t, repo.SetIngress(ctx, id, "ing_1", room, "https://whip/w", "key"))
	require.NoError(t, repo.MarkLive(ctx, id))

	// then
	byRoom, err := repo.GetByRoom(ctx, room)
	require.NoError(t, err)
	require.NotNil(t, byRoom)
	assert.Equal(t, id, byRoom.ID)
	assert.Equal(t, "ing_1", byRoom.IngressID)
	assert.Equal(t, "live", byRoom.Status)
	assert.True(t, byRoom.StartedAt.Valid)

	live, err := repo.ListLive(ctx)
	require.NoError(t, err)
	require.Len(t, live, 1)
	assert.Equal(t, id, live[0].ID)

	// when
	count, ok, err := repo.AdjustViewerCount(ctx, id, 1)

	// then
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, count)

	// when
	transitioned, err := repo.MarkOffline(ctx, id)
	require.NoError(t, err)

	// then
	assert.True(t, transitioned)

	again, err := repo.MarkOffline(ctx, id)
	require.NoError(t, err)
	assert.False(t, again)

	// when
	_, ok, err = repo.AdjustViewerCount(ctx, id, 1)

	// then
	require.NoError(t, err)
	assert.False(t, ok)

	n, err = repo.CountActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// when
	_, err = repo.Create(ctx, user.ID, "Fresh", 3)

	// then
	require.NoError(t, err)
}

func TestLiveStreamRepository_Capacity(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	u1 := repotest.CreateUser(t, repos)
	u2 := repotest.CreateUser(t, repos)
	u3 := repotest.CreateUser(t, repos)

	_, err := repo.Create(ctx, u1.ID, "a", 2)
	require.NoError(t, err)
	_, err = repo.Create(ctx, u2.ID, "b", 2)
	require.NoError(t, err)

	// when
	_, err = repo.Create(ctx, u3.ID, "c", 2)

	// then
	require.ErrorIs(t, err, repository.ErrLiveStreamCapacity)
}

func TestLiveStreamRepository_ListStartingBefore(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	id, err := repo.Create(ctx, user.ID, "stale", 5)
	require.NoError(t, err)

	// when
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	none, err := repo.ListStartingBefore(ctx, past)

	// then
	require.NoError(t, err)
	assert.Empty(t, none)

	// when
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	got, err := repo.ListStartingBefore(ctx, future)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0].ID)
}
