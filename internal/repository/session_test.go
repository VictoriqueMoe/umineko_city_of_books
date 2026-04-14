package repository_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRepository_CreateAndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	token := uuid.NewString()
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	// when
	err := repos.Session.Create(context.Background(), token, user.ID, expiresAt)

	// then
	require.NoError(t, err)
	gotID, gotExpires, err := repos.Session.GetUserID(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotID)
	assert.WithinDuration(t, expiresAt, gotExpires, time.Second)
}

func TestSessionRepository_GetUserID_NotFound(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	_, _, err := repos.Session.GetUserID(context.Background(), "does-not-exist")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	token := repotest.CreateSession(t, repos, user.ID)

	// when
	err := repos.Session.Delete(context.Background(), token)

	// then
	require.NoError(t, err)
	_, _, getErr := repos.Session.GetUserID(context.Background(), token)
	require.Error(t, getErr)
}

func TestSessionRepository_DeleteAllForUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	tokenA := repotest.CreateSession(t, repos, user.ID)
	tokenB := repotest.CreateSession(t, repos, user.ID)
	tokenC := repotest.CreateSession(t, repos, other.ID)

	// when
	err := repos.Session.DeleteAllForUser(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	_, _, errA := repos.Session.GetUserID(context.Background(), tokenA)
	_, _, errB := repos.Session.GetUserID(context.Background(), tokenB)
	_, _, errC := repos.Session.GetUserID(context.Background(), tokenC)
	assert.Error(t, errA)
	assert.Error(t, errB)
	assert.NoError(t, errC)
}

func TestSessionRepository_CleanExpired(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	freshToken := uuid.NewString()
	staleToken := uuid.NewString()
	require.NoError(t, repos.Session.Create(context.Background(), freshToken, user.ID, time.Now().Add(time.Hour)))
	require.NoError(t, repos.Session.Create(context.Background(), staleToken, user.ID, time.Now().Add(-time.Hour)))

	// when
	err := repos.Session.CleanExpired(context.Background())

	// then
	require.NoError(t, err)
	_, _, freshErr := repos.Session.GetUserID(context.Background(), freshToken)
	_, _, staleErr := repos.Session.GetUserID(context.Background(), staleToken)
	assert.NoError(t, freshErr)
	assert.Error(t, staleErr)
}
