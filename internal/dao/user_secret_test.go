package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserSecretDAO_UnlockAndListForUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	// when
	require.NoError(t, repos.UserSecret.Unlock(ctx, user.ID, "alpha"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, user.ID, "beta"))

	// then
	got, err := repos.UserSecret.ListForUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"}, got)
}

func TestUserSecretDAO_Unlock_DuplicateIsNoop(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.UserSecret.Unlock(ctx, user.ID, "dup"))

	// when
	err := repos.UserSecret.Unlock(ctx, user.ID, "dup")

	// then
	require.NoError(t, err)
	got, err := repos.UserSecret.ListForUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"dup"}, got)
}

func TestUserSecretDAO_ListForUser_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	got, err := repos.UserSecret.ListForUser(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestUserSecretDAO_GetUserIDsWithSecret(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	c := daotest.CreateUser(t, repos)
	require.NoError(t, repos.UserSecret.Unlock(ctx, a.ID, "shared"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, b.ID, "shared"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, c.ID, "other"))

	// when
	got, err := repos.UserSecret.GetUserIDsWithSecret(ctx, "shared")

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	holders := map[uuid.UUID]bool{}
	for i := range got {
		holders[got[i]] = true
	}
	assert.True(t, holders[a.ID])
	assert.True(t, holders[b.ID])
	assert.False(t, holders[c.ID])
}

func TestUserSecretDAO_GetUserIDsWithAnyPiece(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	c := daotest.CreateUser(t, repos)
	require.NoError(t, repos.UserSecret.Unlock(ctx, a.ID, "p1"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, a.ID, "p2"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, b.ID, "p2"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, c.ID, "p3"))

	// when
	got, err := repos.UserSecret.GetUserIDsWithAnyPiece(ctx, []string{"p1", "p2"})

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	holders := map[uuid.UUID]bool{}
	for i := range got {
		holders[got[i]] = true
	}
	assert.True(t, holders[a.ID])
	assert.True(t, holders[b.ID])
	assert.False(t, holders[c.ID])
}

func TestUserSecretDAO_GetUserIDsWithAnyPiece_EmptyInput(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.UserSecret.GetUserIDsWithAnyPiece(context.Background(), nil)

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUserSecretDAO_IsSolvedByAnyone(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)

	solved, err := repos.UserSecret.IsSolvedByAnyone(ctx, "no-one-yet")
	require.NoError(t, err)
	assert.False(t, solved)

	// when
	require.NoError(t, repos.UserSecret.Unlock(ctx, user.ID, "x"))

	// then
	solved, err = repos.UserSecret.IsSolvedByAnyone(ctx, "x")
	require.NoError(t, err)
	assert.True(t, solved)
}

func TestUserSecretDAO_DeleteSecrets(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	require.NoError(t, repos.UserSecret.Unlock(ctx, a.ID, "keep"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, a.ID, "drop1"))
	require.NoError(t, repos.UserSecret.Unlock(ctx, b.ID, "drop2"))

	// when
	require.NoError(t, repos.UserSecret.DeleteSecrets(ctx, []string{"drop1", "drop2"}))

	// then
	aGot, err := repos.UserSecret.ListForUser(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"keep"}, aGot)
	bGot, err := repos.UserSecret.ListForUser(ctx, b.ID)
	require.NoError(t, err)
	assert.Empty(t, bGot)
}

func TestUserSecretDAO_DeleteSecrets_EmptyInputIsNoop(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	ctx := context.Background()
	user := daotest.CreateUser(t, repos)
	require.NoError(t, repos.UserSecret.Unlock(ctx, user.ID, "intact"))

	// when
	err := repos.UserSecret.DeleteSecrets(ctx, nil)

	// then
	require.NoError(t, err)
	got, err := repos.UserSecret.ListForUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"intact"}, got)
}
