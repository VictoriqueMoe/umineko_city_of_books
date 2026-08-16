package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSidebarLastVisitedDAO_Upsert_Insert(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), user.ID, "mysteries"))

	got, err := repos.SidebarVisited.ListForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.NotEmpty(t, got["mysteries"])
}

func TestSidebarLastVisitedDAO_Upsert_Overwrites(t *testing.T) {
	ctx := context.Background()
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	require.NoError(t, repos.SidebarVisited.Upsert(ctx, user.ID, "mysteries"))

	_, err := repos.DB().ExecContext(ctx,
		`UPDATE sidebar_last_visited SET visited_at = NOW() - INTERVAL '1 hour' WHERE user_id = $1 AND key = $2`,
		user.ID, "mysteries",
	)
	require.NoError(t, err)

	first, err := repos.SidebarVisited.ListForUser(ctx, user.ID)
	require.NoError(t, err)
	firstTs := first["mysteries"]
	require.NotEmpty(t, firstTs)

	require.NoError(t, repos.SidebarVisited.Upsert(ctx, user.ID, "mysteries"))

	second, err := repos.SidebarVisited.ListForUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.NotEqual(t, firstTs, second["mysteries"])
}

func TestSidebarLastVisitedDAO_ListForUser_IsolatesUsers(t *testing.T) {
	repos := daotest.NewRepos(t)
	userA := daotest.CreateUser(t, repos)
	userB := daotest.CreateUser(t, repos)

	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), userA.ID, "mysteries"))
	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), userA.ID, "secrets"))
	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), userB.ID, "ships"))

	gotA, err := repos.SidebarVisited.ListForUser(context.Background(), userA.ID)
	require.NoError(t, err)
	assert.Len(t, gotA, 2)
	assert.Contains(t, gotA, "mysteries")
	assert.Contains(t, gotA, "secrets")

	gotB, err := repos.SidebarVisited.ListForUser(context.Background(), userB.ID)
	require.NoError(t, err)
	assert.Len(t, gotB, 1)
	assert.Contains(t, gotB, "ships")
}

func TestSidebarLastVisitedDAO_ListForUser_Empty(t *testing.T) {
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	got, err := repos.SidebarVisited.ListForUser(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}
