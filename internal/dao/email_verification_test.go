package dao_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/dao/daotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailVerification_CreateAndGet(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("verifyuser"))
	expiresAt := time.Now().Add(24 * time.Hour)

	// when
	err := repos.EmailVerification.Create(context.Background(), "vhash-abc", user.ID, expiresAt)
	require.NoError(t, err)
	got, err := repos.EmailVerification.GetByTokenHash(context.Background(), "vhash-abc")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, user.ID, got.UserID)
	assert.Nil(t, got.UsedAt)
	assert.WithinDuration(t, expiresAt, got.ExpiresAt, time.Second)
}

func TestEmailVerification_GetMissingReturnsNil(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	got, err := repos.EmailVerification.GetByTokenHash(context.Background(), "nope")

	// then
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestEmailVerification_MarkUsed(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("verifyused"))
	require.NoError(t, repos.EmailVerification.Create(context.Background(), "vhash-used", user.ID, time.Now().Add(time.Hour)))

	// when
	err := repos.EmailVerification.MarkUsed(context.Background(), "vhash-used")
	require.NoError(t, err)
	got, err := repos.EmailVerification.GetByTokenHash(context.Background(), "vhash-used")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotNil(t, got.UsedAt)
}

func TestEmailVerification_DeleteUnusedForUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("verifyclear"))
	require.NoError(t, repos.EmailVerification.Create(context.Background(), "vhash-old", user.ID, time.Now().Add(time.Hour)))
	require.NoError(t, repos.EmailVerification.MarkUsed(context.Background(), "vhash-old"))
	require.NoError(t, repos.EmailVerification.Create(context.Background(), "vhash-new", user.ID, time.Now().Add(time.Hour)))

	// when
	err := repos.EmailVerification.DeleteUnusedForUser(context.Background(), user.ID)
	require.NoError(t, err)

	// then
	used, err := repos.EmailVerification.GetByTokenHash(context.Background(), "vhash-old")
	require.NoError(t, err)
	assert.NotNil(t, used)

	unused, err := repos.EmailVerification.GetByTokenHash(context.Background(), "vhash-new")
	require.NoError(t, err)
	assert.Nil(t, unused)
}

func TestUserDAO_SetEmailAndVerify(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("emailflow"))

	// when
	require.NoError(t, repos.User.SetEmail(context.Background(), user.ID, "flow@example.com"))
	afterSet, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)

	require.NoError(t, repos.User.MarkEmailVerified(context.Background(), user.ID))
	afterVerify, err := repos.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)

	// then
	assert.Equal(t, "flow@example.com", afterSet.Email)
	assert.False(t, afterSet.EmailVerified)
	assert.True(t, afterVerify.EmailVerified)
}

func TestUserDAO_EmailInUse(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("taken"), daotest.WithEmail("taken@example.com"))

	// when / then
	inUse, err := repos.User.EmailInUse(context.Background(), "Taken@example.com", uuid.Nil)
	require.NoError(t, err)
	assert.True(t, inUse)

	excludedSelf, err := repos.User.EmailInUse(context.Background(), "taken@example.com", user.ID)
	require.NoError(t, err)
	assert.False(t, excludedSelf)

	other, err := repos.User.EmailInUse(context.Background(), "free@example.com", uuid.Nil)
	require.NoError(t, err)
	assert.False(t, other)
}

func TestUserDAO_RequiresEmailVerification(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithUsername("graceuser"))

	// when
	blocked, err := repos.User.RequiresEmailVerification(context.Background(), user.ID)
	require.NoError(t, err)

	require.NoError(t, repos.User.MarkEmailVerified(context.Background(), user.ID))
	afterVerify, err := repos.User.RequiresEmailVerification(context.Background(), user.ID)
	require.NoError(t, err)

	// then
	assert.True(t, blocked)
	assert.False(t, afterVerify)
}
