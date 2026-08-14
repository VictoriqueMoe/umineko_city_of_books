package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordReset_CreateAndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("resetuser"))
	expiresAt := time.Now().Add(time.Hour)

	// when
	err := repos.PasswordReset.Create(context.Background(), "hash-abc", user.ID, expiresAt)
	require.NoError(t, err)
	got, err := repos.PasswordReset.GetByTokenHash(context.Background(), "hash-abc")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "hash-abc", got.TokenHash)
	assert.Equal(t, user.ID, got.UserID)
	assert.Nil(t, got.UsedAt)
	assert.WithinDuration(t, expiresAt, got.ExpiresAt, time.Second)
}

func TestPasswordReset_GetMissingReturnsNil(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.PasswordReset.GetByTokenHash(context.Background(), "does-not-exist")

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPasswordReset_MarkUsed(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("usedtoken"))
	require.NoError(t, repos.PasswordReset.Create(context.Background(), "hash-used", user.ID, time.Now().Add(time.Hour)))

	// when
	err := repos.PasswordReset.MarkUsed(context.Background(), "hash-used")
	require.NoError(t, err)
	got, err := repos.PasswordReset.GetByTokenHash(context.Background(), "hash-used")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotNil(t, got.UsedAt)
}

func TestPasswordReset_DeleteUnusedForUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("cleartokens"))
	require.NoError(t, repos.PasswordReset.Create(context.Background(), "hash-old", user.ID, time.Now().Add(time.Hour)))
	require.NoError(t, repos.PasswordReset.MarkUsed(context.Background(), "hash-old"))
	require.NoError(t, repos.PasswordReset.Create(context.Background(), "hash-new", user.ID, time.Now().Add(time.Hour)))

	// when
	err := repos.PasswordReset.DeleteUnusedForUser(context.Background(), user.ID)
	require.NoError(t, err)

	// then
	used, err := repos.PasswordReset.GetByTokenHash(context.Background(), "hash-old")
	require.NoError(t, err)
	assert.NotNil(t, used)

	unused, err := repos.PasswordReset.GetByTokenHash(context.Background(), "hash-new")
	require.NoError(t, err)
	assert.Nil(t, unused)
}

func TestPasswordReset_IssueReplacesUnusedTokens(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("resetissue"))
	require.NoError(t, repos.PasswordReset.Create(context.Background(), "hash-stale", user.ID, time.Now().Add(time.Hour)))

	// when
	err := repos.PasswordReset.Issue(context.Background(), repository.NewPasswordReset{
		TokenHash: "hash-fresh",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	// then
	require.NoError(t, err)

	stale, err := repos.PasswordReset.GetByTokenHash(context.Background(), "hash-stale")
	require.NoError(t, err)
	assert.Nil(t, stale)

	fresh, err := repos.PasswordReset.GetByTokenHash(context.Background(), "hash-fresh")
	require.NoError(t, err)
	require.NotNil(t, fresh)
	assert.Equal(t, user.ID, fresh.UserID)
}
