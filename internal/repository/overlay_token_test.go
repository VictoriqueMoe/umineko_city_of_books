package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlayTokenRepository_UpsertAndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.OverlayToken.Upsert(ctx, user.ID, "tok_abc")

	// then
	require.NoError(t, err)
	token, err := repos.OverlayToken.GetByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "tok_abc", token)

	gotUser, err := repos.OverlayToken.GetUserByToken(ctx, "tok_abc")
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotUser)
}

func TestOverlayTokenRepository_GetByUserEmptyWhenMissing(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)

	// when
	token, err := repos.OverlayToken.GetByUser(ctx, user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, "", token)
}

func TestOverlayTokenRepository_GetUserByTokenNilWhenMissing(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()

	// when
	gotUser, err := repos.OverlayToken.GetUserByToken(ctx, "does-not-exist")

	// then
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, gotUser)
}

func TestOverlayTokenRepository_UpsertReplacesToken(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.OverlayToken.Upsert(ctx, user.ID, "tok_first"))

	// when
	err := repos.OverlayToken.Upsert(ctx, user.ID, "tok_second")

	// then
	require.NoError(t, err)
	token, err := repos.OverlayToken.GetByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "tok_second", token)

	staleUser, err := repos.OverlayToken.GetUserByToken(ctx, "tok_first")
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, staleUser)
}

func TestOverlayTokenRepository_Delete(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	ctx := context.Background()
	user := repotest.CreateUser(t, repos)
	require.NoError(t, repos.OverlayToken.Upsert(ctx, user.ID, "tok_del"))

	// when
	err := repos.OverlayToken.Delete(ctx, user.ID)

	// then
	require.NoError(t, err)
	token, err := repos.OverlayToken.GetByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "", token)
}
