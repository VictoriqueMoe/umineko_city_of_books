package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveStreamDAO_Lifecycle(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Beatrice"))

	// when
	stream, err := repo.Create(ctx, spec.NewLiveStream{UserID: user.ID, Title: "My Stream", MaxConcurrent: 3})

	// then
	require.NoError(t, err)
	require.NotNil(t, stream)

	id := stream.ID
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
	_, err = repo.Create(ctx, spec.NewLiveStream{UserID: user.ID, Title: "Another", MaxConcurrent: 3})

	// then
	require.ErrorIs(t, err, dao.ErrLiveStreamActiveExists)

	// when
	room := "live_" + id.String()
	require.NoError(t, repo.SetIngress(ctx, spec.LiveStreamIngressUpdate{
		ID:        id,
		IngressID: "ing_1",
		Room:      room,
		WhipURL:   "https://whip/w",
		StreamKey: "key",
	}))
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
	count, ok, err := repo.AdjustViewerCount(ctx, spec.LiveStreamViewerAdjustment{ID: id, Delta: 1})

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
	_, ok, err = repo.AdjustViewerCount(ctx, spec.LiveStreamViewerAdjustment{ID: id, Delta: 1})

	// then
	require.NoError(t, err)
	assert.False(t, ok)

	n, err = repo.CountActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// when
	_, err = repo.Create(ctx, spec.NewLiveStream{UserID: user.ID, Title: "Fresh", MaxConcurrent: 3})

	// then
	require.NoError(t, err)
}

func TestLiveStreamDAO_Capacity(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	u1 := daotest.CreateUser(t, repos)
	u2 := daotest.CreateUser(t, repos)
	u3 := daotest.CreateUser(t, repos)

	_, err := repo.Create(ctx, spec.NewLiveStream{UserID: u1.ID, Title: "a", MaxConcurrent: 2})
	require.NoError(t, err)
	_, err = repo.Create(ctx, spec.NewLiveStream{UserID: u2.ID, Title: "b", MaxConcurrent: 2})
	require.NoError(t, err)

	// when
	_, err = repo.Create(ctx, spec.NewLiveStream{UserID: u3.ID, Title: "c", MaxConcurrent: 2})

	// then
	require.ErrorIs(t, err, dao.ErrLiveStreamCapacity)
}

func TestLiveStreamDAO_ListStartingBefore(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	stream, err := repo.Create(ctx, spec.NewLiveStream{UserID: user.ID, Title: "stale", MaxConcurrent: 5})
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
	assert.Equal(t, stream.ID, got[0].ID)
}

func TestLiveStreamDAO_SetTitle(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	stream, err := repo.Create(ctx, spec.NewLiveStream{UserID: user.ID, Title: "Old Title", MaxConcurrent: 3})
	require.NoError(t, err)
	id := stream.ID

	// when
	err = repo.SetTitle(ctx, spec.LiveStreamTitleUpdate{ID: id, Title: "New Title"})

	// then
	require.NoError(t, err)
	row, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "New Title", row.Title)
}

func TestLiveStreamDAO_GetActiveByUsername(t *testing.T) {
	// given a streamer with a mixed-case username and an active stream
	repos := daotest.NewRepos(t)
	repo := repos.LiveStream
	ctx := context.Background()
	user := daotest.CreateUser(t, repos, daotest.WithUsername("Featherine"), daotest.WithDisplayName("Featherine"))

	stream, err := repo.Create(ctx, spec.NewLiveStream{UserID: user.ID, Title: "Ciconia blind run", MaxConcurrent: 3})
	require.NoError(t, err)
	require.NotNil(t, stream)

	lookups := []struct {
		name     string
		username string
	}{
		{name: "the stored casing", username: "Featherine"},
		{name: "all lower case", username: "featherine"},
		{name: "all upper case", username: "FEATHERINE"},
	}

	for _, tc := range lookups {
		t.Run(tc.name, func(t *testing.T) {
			// when the stable url is resolved
			row, err := repo.GetActiveByUsername(ctx, tc.username)

			// then the same stream comes back regardless of casing
			require.NoError(t, err)
			require.NotNil(t, row)
			assert.Equal(t, stream.ID, row.ID)
			assert.Equal(t, "Featherine", row.Username)
		})
	}

	t.Run("an unknown username resolves to nothing", func(t *testing.T) {
		// when
		row, err := repo.GetActiveByUsername(ctx, "nobody")

		// then
		require.NoError(t, err)
		assert.Nil(t, row)
	})

	t.Run("an ended stream no longer answers the stable url", func(t *testing.T) {
		// given the stream has ended
		ended, err := repo.MarkOffline(ctx, stream.ID)
		require.NoError(t, err)
		require.True(t, ended)

		// when the stable url is resolved again
		row, err := repo.GetActiveByUsername(ctx, "Featherine")

		// then the historical row is not returned
		require.NoError(t, err)
		assert.Nil(t, row)
	})
}
