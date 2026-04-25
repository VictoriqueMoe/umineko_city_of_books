package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGiphyFavouriteRepository_Add_Insert(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	err := repos.GiphyFavourite.Add(context.Background(), user.ID, repository.GiphyFavourite{
		GiphyID:    "abc",
		URL:        "https://media.giphy.com/abc.gif",
		Title:      "cat",
		PreviewURL: "https://media.giphy.com/abc-preview.gif",
		Width:      200,
		Height:     150,
	})
	require.NoError(t, err)

	list, total, err := repos.GiphyFavourite.List(context.Background(), user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, list, 1)
	assert.Equal(t, "abc", list[0].GiphyID)
	assert.Equal(t, "https://media.giphy.com/abc.gif", list[0].URL)
	assert.Equal(t, "cat", list[0].Title)
	assert.Equal(t, "https://media.giphy.com/abc-preview.gif", list[0].PreviewURL)
	assert.Equal(t, 200, list[0].Width)
	assert.Equal(t, 150, list[0].Height)
	assert.False(t, list[0].CreatedAt.IsZero())
}

func TestGiphyFavouriteRepository_Add_Upsert(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	require.NoError(t, repos.GiphyFavourite.Add(context.Background(), user.ID, repository.GiphyFavourite{
		GiphyID: "abc",
		URL:     "old-url",
		Title:   "old",
	}))
	require.NoError(t, repos.GiphyFavourite.Add(context.Background(), user.ID, repository.GiphyFavourite{
		GiphyID: "abc",
		URL:     "new-url",
		Title:   "new",
	}))

	list, total, err := repos.GiphyFavourite.List(context.Background(), user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, list, 1)
	assert.Equal(t, "new-url", list[0].URL)
	assert.Equal(t, "new", list[0].Title)
}

func TestGiphyFavouriteRepository_Remove(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	require.NoError(t, repos.GiphyFavourite.Add(context.Background(), user.ID, repository.GiphyFavourite{
		GiphyID: "abc",
		URL:     "u",
	}))
	require.NoError(t, repos.GiphyFavourite.Remove(context.Background(), user.ID, "abc"))

	_, total, err := repos.GiphyFavourite.List(context.Background(), user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestGiphyFavouriteRepository_Remove_Missing_IsNoOp(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	err := repos.GiphyFavourite.Remove(context.Background(), user.ID, "does-not-exist")
	assert.NoError(t, err)
}

func TestGiphyFavouriteRepository_List_IsolatesUsers(t *testing.T) {
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos)
	bob := repotest.CreateUser(t, repos)

	require.NoError(t, repos.GiphyFavourite.Add(context.Background(), alice.ID, repository.GiphyFavourite{GiphyID: "a", URL: "u"}))
	require.NoError(t, repos.GiphyFavourite.Add(context.Background(), bob.ID, repository.GiphyFavourite{GiphyID: "b", URL: "u"}))

	aliceList, aliceTotal, err := repos.GiphyFavourite.List(context.Background(), alice.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, aliceTotal)
	require.Len(t, aliceList, 1)
	assert.Equal(t, "a", aliceList[0].GiphyID)

	bobList, bobTotal, err := repos.GiphyFavourite.List(context.Background(), bob.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, bobTotal)
	require.Len(t, bobList, 1)
	assert.Equal(t, "b", bobList[0].GiphyID)
}

func TestGiphyFavouriteRepository_List_OrdersByCreatedDesc(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()

	for _, id := range []string{"first", "second", "third"} {
		require.NoError(t, repos.GiphyFavourite.Add(ctx, user.ID, repository.GiphyFavourite{GiphyID: id, URL: "u"}))
	}
	_, err := repos.DB().ExecContext(ctx,
		`UPDATE giphy_favourites SET created_at =
			CASE giphy_id
				WHEN 'first'  THEN TIMESTAMPTZ '2026-01-01 10:00:00'
				WHEN 'second' THEN TIMESTAMPTZ '2026-01-01 11:00:00'
				WHEN 'third'  THEN TIMESTAMPTZ '2026-01-01 12:00:00'
			END
		 WHERE user_id = $1`, user.ID)
	require.NoError(t, err)

	list, total, err := repos.GiphyFavourite.List(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, list, 3)
	assert.Equal(t, "third", list[0].GiphyID)
	assert.Equal(t, "second", list[1].GiphyID)
	assert.Equal(t, "first", list[2].GiphyID)
}

func TestGiphyFavouriteRepository_List_LimitAndOffset(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()

	for _, id := range []string{"a", "b", "c", "d", "e"} {
		require.NoError(t, repos.GiphyFavourite.Add(ctx, user.ID, repository.GiphyFavourite{GiphyID: id, URL: "u"}))
	}

	list, total, err := repos.GiphyFavourite.List(ctx, user.ID, 2, 1)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, list, 2)
}

func TestGiphyFavouriteRepository_List_DefaultsLimit(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()

	require.NoError(t, repos.GiphyFavourite.Add(ctx, user.ID, repository.GiphyFavourite{GiphyID: "a", URL: "u"}))

	list, _, err := repos.GiphyFavourite.List(ctx, user.ID, 0, -5)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestGiphyFavouriteRepository_ListIDs(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()

	for _, id := range []string{"x", "y", "z"} {
		require.NoError(t, repos.GiphyFavourite.Add(ctx, user.ID, repository.GiphyFavourite{GiphyID: id, URL: "u"}))
	}

	ids, err := repos.GiphyFavourite.ListIDs(ctx, user.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"x", "y", "z"}, ids)
}

func TestGiphyFavouriteRepository_ListIDs_Empty(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	ids, err := repos.GiphyFavourite.ListIDs(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestGiphyFavouriteRepository_CascadesOnUserDelete(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	ctx := context.Background()

	require.NoError(t, repos.GiphyFavourite.Add(ctx, user.ID, repository.GiphyFavourite{GiphyID: "a", URL: "u"}))
	_, err := repos.DB().ExecContext(ctx, `DELETE FROM users WHERE id = $1`, user.ID)
	require.NoError(t, err)

	_, total, err := repos.GiphyFavourite.List(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}
