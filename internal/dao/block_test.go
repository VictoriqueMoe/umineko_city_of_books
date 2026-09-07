package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockDAO_Block(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	blocker := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)

	// when
	err := repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID})

	// then
	require.NoError(t, err)
	isBlocked, err := repos.Block.IsBlocked(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID})
	require.NoError(t, err)
	assert.True(t, isBlocked)
}

func TestBlockDAO_Block_Idempotent(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	blocker := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID}))

	// when
	err := repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID})

	// then
	require.NoError(t, err)
	users, err := repos.Block.GetBlockedUsers(context.Background(), blocker.ID)
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestBlockDAO_Unblock(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	blocker := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID}))

	// when
	err := repos.Block.Unblock(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID})

	// then
	require.NoError(t, err)
	isBlocked, err := repos.Block.IsBlocked(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID})
	require.NoError(t, err)
	assert.False(t, isBlocked)
}

func TestBlockDAO_Unblock_NonExistingNoop(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	blocker := daotest.CreateUser(t, repos)
	blocked := daotest.CreateUser(t, repos)

	// when
	err := repos.Block.Unblock(context.Background(), spec.BlockSpec{BlockerID: blocker.ID, BlockedID: blocked.ID})

	// then
	require.NoError(t, err)
}

func TestBlockDAO_IsBlocked_False(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)

	// when
	isBlocked, err := repos.Block.IsBlocked(context.Background(), spec.BlockSpec{BlockerID: a.ID, BlockedID: b.ID})

	// then
	require.NoError(t, err)
	assert.False(t, isBlocked)
}

func TestBlockDAO_IsBlocked_DirectionMatters(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: a.ID, BlockedID: b.ID}))

	// when
	reverse, err := repos.Block.IsBlocked(context.Background(), spec.BlockSpec{BlockerID: b.ID, BlockedID: a.ID})

	// then
	require.NoError(t, err)
	assert.False(t, reverse)
}

func TestBlockDAO_IsBlockedEither_AblocksB(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: a.ID, BlockedID: b.ID}))

	// when
	forward, err := repos.Block.IsBlockedEither(context.Background(), spec.BlockPairSpec{UserA: a.ID, UserB: b.ID})
	reverse, err2 := repos.Block.IsBlockedEither(context.Background(), spec.BlockPairSpec{UserA: b.ID, UserB: a.ID})

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.True(t, forward)
	assert.True(t, reverse)
}

func TestBlockDAO_IsBlockedEither_BblocksA(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: b.ID, BlockedID: a.ID}))

	// when
	forward, err := repos.Block.IsBlockedEither(context.Background(), spec.BlockPairSpec{UserA: a.ID, UserB: b.ID})
	reverse, err2 := repos.Block.IsBlockedEither(context.Background(), spec.BlockPairSpec{UserA: b.ID, UserB: a.ID})

	// then
	require.NoError(t, err)
	require.NoError(t, err2)
	assert.True(t, forward)
	assert.True(t, reverse)
}

func TestBlockDAO_IsBlockedEither_None(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	a := daotest.CreateUser(t, repos)
	b := daotest.CreateUser(t, repos)

	// when
	either, err := repos.Block.IsBlockedEither(context.Background(), spec.BlockPairSpec{UserA: a.ID, UserB: b.ID})

	// then
	require.NoError(t, err)
	assert.False(t, either)
}

func TestBlockDAO_GetBlockedIDs_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	ids, err := repos.Block.GetBlockedIDs(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestBlockDAO_GetBlockedIDs_BothDirections(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	blockedByUser := daotest.CreateUser(t, repos)
	blockerOfUser := daotest.CreateUser(t, repos)
	unrelated := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: user.ID, BlockedID: blockedByUser.ID}))
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: blockerOfUser.ID, BlockedID: user.ID}))
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: unrelated.ID, BlockedID: blockedByUser.ID}))

	// when
	ids, err := repos.Block.GetBlockedIDs(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{blockedByUser.ID, blockerOfUser.ID}, ids)
}

func TestBlockDAO_GetBlockedUsers_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)

	// when
	users, err := repos.Block.GetBlockedUsers(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestBlockDAO_GetBlockedUsers_OnlyOutbound(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	target := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: user.ID, BlockedID: target.ID}))
	require.NoError(t, repos.Block.Block(context.Background(), spec.BlockSpec{BlockerID: other.ID, BlockedID: user.ID}))

	// when
	users, err := repos.Block.GetBlockedUsers(context.Background(), user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, target.ID, users[0].ID)
	assert.Equal(t, target.Username, users[0].Username)
	assert.NotEmpty(t, users[0].BlockedAt)
}

func TestBlockDAO_GetBlockedUsers_OrderedByBlockedAtDesc(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	first := daotest.CreateUser(t, repos)
	second := daotest.CreateUser(t, repos)
	third := daotest.CreateUser(t, repos)
	ctx := context.Background()
	insert := `INSERT INTO blocks (blocker_id, blocked_id, created_at) VALUES ($1, $2, $3)`
	_, err := repos.DB().ExecContext(ctx, insert, user.ID, first.ID, "2026-01-01 10:00:00")
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx, insert, user.ID, second.ID, "2026-01-02 10:00:00")
	require.NoError(t, err)
	_, err = repos.DB().ExecContext(ctx, insert, user.ID, third.ID, "2026-01-03 10:00:00")
	require.NoError(t, err)

	// when
	users, err := repos.Block.GetBlockedUsers(ctx, user.ID)

	// then
	require.NoError(t, err)
	require.Len(t, users, 3)
	assert.Equal(t, third.ID, users[0].ID)
	assert.Equal(t, second.ID, users[1].ID)
	assert.Equal(t, first.ID, users[2].ID)
}
