package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBannedGiphyDAO_AddAndList(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	uid := user.ID.String()

	// when
	require.NoError(t, repos.BannedGiphy.Add(context.Background(), spec.NewBannedGiphy{Kind: "id", Value: "abc123", Reason: "spam", CreatedBy: &uid}))
	require.NoError(t, repos.BannedGiphy.Add(context.Background(), spec.NewBannedGiphy{Kind: "term", Value: "lewd", Reason: "", CreatedBy: nil}))

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
	require.NoError(t, repos.BannedGiphy.Add(ctx, spec.NewBannedGiphy{Kind: "id", Value: "dup", Reason: "first", CreatedBy: nil}))

	// when
	err := repos.BannedGiphy.Add(ctx, spec.NewBannedGiphy{Kind: "id", Value: "dup", Reason: "second", CreatedBy: nil})

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
	require.NoError(t, repos.BannedGiphy.Add(ctx, spec.NewBannedGiphy{Kind: "id", Value: "keep", Reason: "", CreatedBy: nil}))
	require.NoError(t, repos.BannedGiphy.Add(ctx, spec.NewBannedGiphy{Kind: "id", Value: "drop", Reason: "", CreatedBy: nil}))

	// when
	require.NoError(t, repos.BannedGiphy.Remove(ctx, spec.BannedGiphyDeletion{Kind: "id", Value: "drop"}))

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
	err := repos.BannedGiphy.Remove(context.Background(), spec.BannedGiphyDeletion{Kind: "id", Value: "ghost"})

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
