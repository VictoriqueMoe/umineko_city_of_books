package dao_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/dao/daotest"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretID = "witchHunter"

var testPieceIDs = []string{"piece_01", "piece_02", "piece_03", "piece_04"}

func unlockSecretFor(t *testing.T, repos *repository.Repositories, userID uuid.UUID, secretID string) {
	t.Helper()
	require.NoError(t, repos.UserSecret.Unlock(context.Background(), spec.SecretUnlock{UserID: userID, SecretID: secretID}))
}

func createSecretComment(t *testing.T, repos *repository.Repositories, secretID string, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()

	return createComment(t, repos, spec.CommentTarget{Kind: spec.CommentKindSecret, Key: secretID}, userID, nil, body)
}

func TestSecretDAO_GetFirstSolver_None(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	solver, err := repos.Secret.GetFirstSolver(context.Background(), testSecretID)

	// then
	require.NoError(t, err)
	assert.Nil(t, solver)
}

func TestSecretDAO_GetFirstSolver_Found(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	winner := daotest.CreateUser(t, repos, daotest.WithDisplayName("First"))
	unlockSecretFor(t, repos, winner.ID, testSecretID)

	// when
	solver, err := repos.Secret.GetFirstSolver(context.Background(), testSecretID)

	// then
	require.NoError(t, err)
	require.NotNil(t, solver)
	assert.Equal(t, winner.ID, solver.UserID)
	assert.Equal(t, "First", solver.DisplayName)
	assert.NotEmpty(t, solver.UnlockedAt)
}

func TestSecretDAO_GetProgressLeaderboard_AggregatesAndSorts(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	alice := daotest.CreateUser(t, repos, daotest.WithDisplayName("Alice"))
	bob := daotest.CreateUser(t, repos, daotest.WithDisplayName("Bob"))
	charlie := daotest.CreateUser(t, repos, daotest.WithDisplayName("Charlie"))

	unlockSecretFor(t, repos, alice.ID, "piece_01")
	unlockSecretFor(t, repos, alice.ID, "piece_02")
	unlockSecretFor(t, repos, alice.ID, "piece_03")

	unlockSecretFor(t, repos, bob.ID, "piece_01")

	unlockSecretFor(t, repos, charlie.ID, "piece_02")
	unlockSecretFor(t, repos, charlie.ID, "piece_04")

	// when
	rows, err := repos.Secret.GetProgressLeaderboard(context.Background(), testPieceIDs)

	// then
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, alice.ID, rows[0].UserID)
	assert.Equal(t, 3, rows[0].Pieces)
	assert.Equal(t, 2, rows[1].Pieces)
	assert.Equal(t, 1, rows[2].Pieces)
}

func TestSecretDAO_GetProgressLeaderboard_Empty(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)

	// when
	rows, err := repos.Secret.GetProgressLeaderboard(context.Background(), testPieceIDs)

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestSecretDAO_GetPieceCountForUser(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	unlockSecretFor(t, repos, user.ID, "piece_01")
	unlockSecretFor(t, repos, user.ID, "piece_03")

	// when
	count, err := repos.Secret.GetPieceCountForUser(context.Background(), spec.SecretPieceCount{UserID: user.ID, PieceIDs: testPieceIDs})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestSecretDAO_GetPieceCountForUser_OtherUsersIgnored(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	other := daotest.CreateUser(t, repos)
	unlockSecretFor(t, repos, other.ID, "piece_01")

	// when
	count, err := repos.Secret.GetPieceCountForUser(context.Background(), spec.SecretPieceCount{UserID: user.ID, PieceIDs: testPieceIDs})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSecretDAO_GetUserProgressSummary(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos, daotest.WithDisplayName("Hunter"))
	unlockSecretFor(t, repos, user.ID, "piece_01")
	unlockSecretFor(t, repos, user.ID, "piece_02")

	// when
	summary, err := repos.Secret.GetUserProgressSummary(context.Background(), spec.SecretPieceCount{UserID: user.ID, PieceIDs: testPieceIDs})

	// then
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, user.ID, summary.UserID)
	assert.Equal(t, 2, summary.Pieces)
	assert.Equal(t, "Hunter", summary.DisplayName)
}

func TestSecretDAO_CountCommentsBySecret(t *testing.T) {
	// given
	repos := daotest.NewRepos(t)
	user := daotest.CreateUser(t, repos)
	createSecretComment(t, repos, testSecretID, user.ID, "one")
	createSecretComment(t, repos, testSecretID, user.ID, "two")
	createSecretComment(t, repos, "another", user.ID, "different")

	// when
	counts, err := repos.Secret.CountCommentsBySecret(context.Background(), []string{testSecretID, "another", "unknown"})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, counts[testSecretID])
	assert.Equal(t, 1, counts["another"])
	_, hasUnknown := counts["unknown"]
	assert.False(t, hasUnknown)
}

func TestSecretDAO_GetCommenterIDs(t *testing.T) {
	// given two users commenting on the secret, one of them twice, and a third commenting elsewhere
	repos := daotest.NewRepos(t)
	first := daotest.CreateUser(t, repos)
	second := daotest.CreateUser(t, repos)
	elsewhere := daotest.CreateUser(t, repos)
	createSecretComment(t, repos, testSecretID, first.ID, "one")
	createSecretComment(t, repos, testSecretID, first.ID, "again")
	createSecretComment(t, repos, testSecretID, second.ID, "two")
	createSecretComment(t, repos, "another", elsewhere.ID, "different")

	// when
	ids, err := repos.Secret.GetCommenterIDs(context.Background(), testSecretID)

	// then each commenter on the secret appears exactly once
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{first.ID, second.ID}, ids)
}
