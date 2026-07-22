package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBannedGiphyDAO_AddAndList(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	uid := user.ID.String()

	// when
	require.NoError(t, repos.BannedGiphy.Add(context.Background(), "id", "abc123", "spam", &uid))
	require.NoError(t, repos.BannedGiphy.Add(context.Background(), "term", "lewd", "", nil))

	// then
	rows, err := repos.BannedGiphy.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byKey := map[string]int{}
	for i := range rows {
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

func TestBannedGiphyDAO_Add_DuplicateIsNoop(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
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

func TestBannedGiphyDAO_Remove(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
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

func TestBannedGiphyDAO_Remove_NonExistentIsNoop(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	err := repos.BannedGiphy.Remove(context.Background(), "id", "ghost")

	// then
	require.NoError(t, err)
}

func TestBannedGiphyDAO_List_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	rows, err := repos.BannedGiphy.List(context.Background())

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
}
