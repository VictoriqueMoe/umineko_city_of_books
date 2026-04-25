package repository_test

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/repository/repotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSidebarLastVisitedRepository_Upsert_Insert(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), user.ID, "mysteries"))

	got, err := repos.SidebarVisited.ListForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.NotEmpty(t, got["mysteries"])
}

func TestSidebarLastVisitedRepository_Upsert_Overwrites(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), user.ID, "mysteries"))
	first, err := repos.SidebarVisited.ListForUser(context.Background(), user.ID)
	require.NoError(t, err)
	firstTs := first["mysteries"]
	require.NotEmpty(t, firstTs)

	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, repos.SidebarVisited.Upsert(context.Background(), user.ID, "mysteries"))

	second, err := repos.SidebarVisited.ListForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.NotEqual(t, firstTs, second["mysteries"])
}

func TestSidebarLastVisitedRepository_ListForUser_IsolatesUsers(t *testing.T) {
	repos := repotest.NewRepos(t)
	userA := repotest.CreateUser(t, repos)
	userB := repotest.CreateUser(t, repos)

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

func TestSidebarLastVisitedRepository_ListForUser_Empty(t *testing.T) {
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	got, err := repos.SidebarVisited.ListForUser(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}
