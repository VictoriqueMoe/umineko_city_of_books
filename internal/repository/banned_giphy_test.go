package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBannedGiphyRepository_AddAndList(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	uid := user.ID.String()

	// when
	require.NoError(t, repos.BannedGiphy.Add(context.Background(), "id", "abc123", "spam", &uid))
	require.NoError(t, repos.BannedGiphy.Add(context.Background(), "term", "lewd", "", nil))

	// then
	rows, err := repos.BannedGiphy.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byKey := map[string]int{}
	for i := 0; i < len(rows); i++ {
		byKey[rows[i].Kind+":"+rows[i].Value] = i
	}

	idRow := rows[byKey["id:abc123"]]
	assert.Equal(t, "spam", idRow.Reason)
	require.NotNil(t, idRow.CreatedBy)
	assert.Equal(t, uid, *idRow.CreatedBy)

	termRow := rows[byKey["term:lewd"]]
	assert.Equal(t, "", termRow.Reason)
	assert.Nil(t, termRow.CreatedBy)
}

func TestBannedGiphyRepository_Add_DuplicateIsNoop(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.BannedGiphy.Add(ctx, "id", "dup", "first", nil))

	// when
	err := repos.BannedGiphy.Add(ctx, "id", "dup", "second", nil)

	// then
	require.NoError(t, err)
	rows, err := repos.BannedGiphy.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "first", rows[0].Reason)
}

func TestBannedGiphyRepository_Remove(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	require.NoError(t, repos.BannedGiphy.Add(ctx, "id", "keep", "", nil))
	require.NoError(t, repos.BannedGiphy.Add(ctx, "id", "drop", "", nil))

	// when
	require.NoError(t, repos.BannedGiphy.Remove(ctx, "id", "drop"))

	// then
	rows, err := repos.BannedGiphy.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "keep", rows[0].Value)
}

func TestBannedGiphyRepository_Remove_NonExistentIsNoop(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	err := repos.BannedGiphy.Remove(context.Background(), "id", "ghost")

	// then
	require.NoError(t, err)
}

func TestBannedGiphyRepository_List_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, err := repos.BannedGiphy.List(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
}
