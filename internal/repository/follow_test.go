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

func TestFollowRepository_Follow(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)

	// when
	err := repos.Follow.Follow(context.Background(), follower.ID, target.ID)

	// then
	require.NoError(t, err)
	following, err := repos.Follow.IsFollowing(context.Background(), follower.ID, target.ID)
	require.NoError(t, err)
	assert.True(t, following)
}

func TestFollowRepository_Follow_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, target.ID))

	// when
	err := repos.Follow.Follow(context.Background(), follower.ID, target.ID)

	// then
	require.NoError(t, err)
	count, err := repos.Follow.GetFollowerCount(context.Background(), target.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestFollowRepository_Follow_Self(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	err := repos.Follow.Follow(context.Background(), user.ID, user.ID)

	// then
	require.NoError(t, err)
	following, err := repos.Follow.IsFollowing(context.Background(), user.ID, user.ID)
	require.NoError(t, err)
	assert.True(t, following)
}

func TestFollowRepository_Unfollow(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, target.ID))

	// when
	err := repos.Follow.Unfollow(context.Background(), follower.ID, target.ID)

	// then
	require.NoError(t, err)
	following, err := repos.Follow.IsFollowing(context.Background(), follower.ID, target.ID)
	require.NoError(t, err)
	assert.False(t, following)
}

func TestFollowRepository_Unfollow_NotFollowing(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)

	// when
	err := repos.Follow.Unfollow(context.Background(), follower.ID, target.ID)

	// then
	require.NoError(t, err)
}

func TestFollowRepository_IsFollowing_False(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	target := repotest.CreateUser(t, repos)

	// when
	following, err := repos.Follow.IsFollowing(context.Background(), follower.ID, target.ID)

	// then
	require.NoError(t, err)
	assert.False(t, following)
}

func TestFollowRepository_IsFollowing_DirectionMatters(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), a.ID, b.ID))

	// when
	reverse, err := repos.Follow.IsFollowing(context.Background(), b.ID, a.ID)

	// then
	require.NoError(t, err)
	assert.False(t, reverse)
}

func TestFollowRepository_GetFollowerCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	target := repotest.CreateUser(t, repos)
	for i := 0; i < 3; i++ {
		f := repotest.CreateUser(t, repos)
		require.NoError(t, repos.Follow.Follow(context.Background(), f.ID, target.ID))
	}

	// when
	count, err := repos.Follow.GetFollowerCount(context.Background(), target.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestFollowRepository_GetFollowerCount_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	count, err := repos.Follow.GetFollowerCount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestFollowRepository_GetFollowingCount(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	for i := 0; i < 4; i++ {
		t2 := repotest.CreateUser(t, repos)
		require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, t2.ID))
	}

	// when
	count, err := repos.Follow.GetFollowingCount(context.Background(), follower.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestFollowRepository_GetFollowingCount_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	count, err := repos.Follow.GetFollowingCount(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestFollowRepository_GetFollowers_Ordering(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	target := repotest.CreateUser(t, repos)
	first := repotest.CreateUser(t, repos)
	second := repotest.CreateUser(t, repos)
	third := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), first.ID, target.ID))
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, repos.Follow.Follow(context.Background(), second.ID, target.ID))
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, repos.Follow.Follow(context.Background(), third.ID, target.ID))

	// when
	users, total, err := repos.Follow.GetFollowers(context.Background(), target.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, users, 3)
	assert.Equal(t, third.ID, users[0].ID)
	assert.Equal(t, second.ID, users[1].ID)
	assert.Equal(t, first.ID, users[2].ID)
}

func TestFollowRepository_GetFollowers_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	target := repotest.CreateUser(t, repos)
	for i := 0; i < 5; i++ {
		f := repotest.CreateUser(t, repos)
		require.NoError(t, repos.Follow.Follow(context.Background(), f.ID, target.ID))
	}

	// when
	page1, total1, err1 := repos.Follow.GetFollowers(context.Background(), target.ID, 2, 0)
	page2, total2, err2 := repos.Follow.GetFollowers(context.Background(), target.ID, 2, 2)
	page3, total3, err3 := repos.Follow.GetFollowers(context.Background(), target.ID, 2, 4)

	// then
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	assert.Equal(t, 5, total1)
	assert.Equal(t, 5, total2)
	assert.Equal(t, 5, total3)
	assert.Len(t, page1, 2)
	assert.Len(t, page2, 2)
	assert.Len(t, page3, 1)
}

func TestFollowRepository_GetFollowers_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	users, total, err := repos.Follow.GetFollowers(context.Background(), user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, users)
}

func TestFollowRepository_GetFollowers_PopulatesUserFields(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	target := repotest.CreateUser(t, repos)
	follower := repotest.CreateUser(t, repos, repotest.WithUsername("alice_"+uuid.New().String()[:6]), repotest.WithDisplayName("Alice"))
	require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, target.ID))

	// when
	users, _, err := repos.Follow.GetFollowers(context.Background(), target.ID, 10, 0)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, follower.ID, users[0].ID)
	assert.Equal(t, follower.Username, users[0].Username)
	assert.Equal(t, "Alice", users[0].DisplayName)
	assert.Equal(t, "", users[0].Role)
}

func TestFollowRepository_GetFollowing_Ordering(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, a.ID))
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, b.ID))

	// when
	users, total, err := repos.Follow.GetFollowing(context.Background(), follower.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, users, 2)
	assert.Equal(t, b.ID, users[0].ID)
	assert.Equal(t, a.ID, users[1].ID)
}

func TestFollowRepository_GetFollowing_Pagination(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	follower := repotest.CreateUser(t, repos)
	for i := 0; i < 4; i++ {
		t2 := repotest.CreateUser(t, repos)
		require.NoError(t, repos.Follow.Follow(context.Background(), follower.ID, t2.ID))
	}

	// when
	page, total, err := repos.Follow.GetFollowing(context.Background(), follower.ID, 2, 1)

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, page, 2)
}

func TestFollowRepository_GetFollowing_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	users, total, err := repos.Follow.GetFollowing(context.Background(), user.ID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, users)
}

func TestFollowRepository_GetMutualFollowers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	mutualA := repotest.CreateUser(t, repos, repotest.WithDisplayName("Beatrice"))
	mutualB := repotest.CreateUser(t, repos, repotest.WithDisplayName("Ange"))
	oneWay := repotest.CreateUser(t, repos, repotest.WithDisplayName("Zepar"))
	require.NoError(t, repos.Follow.Follow(context.Background(), user.ID, mutualA.ID))
	require.NoError(t, repos.Follow.Follow(context.Background(), mutualA.ID, user.ID))
	require.NoError(t, repos.Follow.Follow(context.Background(), user.ID, mutualB.ID))
	require.NoError(t, repos.Follow.Follow(context.Background(), mutualB.ID, user.ID))
	require.NoError(t, repos.Follow.Follow(context.Background(), user.ID, oneWay.ID))

	// when
	users, err := repos.Follow.GetMutualFollowers(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, mutualB.ID, users[0].ID)
	assert.Equal(t, mutualA.ID, users[1].ID)
}

func TestFollowRepository_GetMutualFollowers_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), user.ID, other.ID))

	// when
	users, err := repos.Follow.GetMutualFollowers(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestFollowRepository_GetMutualFollowers_NoFollows(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	users, err := repos.Follow.GetMutualFollowers(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestFollowRepository_FollowerAndFollowingCount_Independent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	a := repotest.CreateUser(t, repos)
	b := repotest.CreateUser(t, repos)
	c := repotest.CreateUser(t, repos)
	require.NoError(t, repos.Follow.Follow(context.Background(), a.ID, user.ID))
	require.NoError(t, repos.Follow.Follow(context.Background(), b.ID, user.ID))
	require.NoError(t, repos.Follow.Follow(context.Background(), user.ID, c.ID))

	// when
	followers, errF := repos.Follow.GetFollowerCount(context.Background(), user.ID)
	following, errG := repos.Follow.GetFollowingCount(context.Background(), user.ID)

	// then
	require.NoError(t, errF)
	require.NoError(t, errG)
	assert.Equal(t, 2, followers)
	assert.Equal(t, 1, following)
}
