package repository_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/repotest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretID = "witchHunter"

var testPieceIDs = []string{"piece_01", "piece_02", "piece_03", "piece_04"}

func unlockSecretFor(t *testing.T, repos *repository.Repositories, userID uuid.UUID, secretID string) {
	t.Helper()
	require.NoError(t, repos.UserSecret.Unlock(context.Background(), userID, secretID))
}

func createSecretComment(t *testing.T, repos *repository.Repositories, secretID string, parent *uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, repos.Secret.CreateComment(context.Background(), id, secretID, parent, userID, body))
	return id
}

func TestSecretRepository_GetFirstSolver_None(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	solver, err := repos.Secret.GetFirstSolver(context.Background(), testSecretID)

	// then
	require.NoError(t, err)
	assert.Nil(t, solver)
}

func TestSecretRepository_GetFirstSolver_Found(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	winner := repotest.CreateUser(t, repos, repotest.WithDisplayName("First"))
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

func TestSecretRepository_GetProgressLeaderboard_AggregatesAndSorts(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	alice := repotest.CreateUser(t, repos, repotest.WithDisplayName("Alice"))
	bob := repotest.CreateUser(t, repos, repotest.WithDisplayName("Bob"))
	charlie := repotest.CreateUser(t, repos, repotest.WithDisplayName("Charlie"))

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

func TestSecretRepository_GetProgressLeaderboard_Empty(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)

	// when
	rows, err := repos.Secret.GetProgressLeaderboard(context.Background(), testPieceIDs)

	// then
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestSecretRepository_GetPieceCountForUser(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	unlockSecretFor(t, repos, user.ID, "piece_01")
	unlockSecretFor(t, repos, user.ID, "piece_03")

	// when
	count, err := repos.Secret.GetPieceCountForUser(context.Background(), user.ID, testPieceIDs)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestSecretRepository_GetPieceCountForUser_OtherUsersIgnored(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	unlockSecretFor(t, repos, other.ID, "piece_01")

	// when
	count, err := repos.Secret.GetPieceCountForUser(context.Background(), user.ID, testPieceIDs)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSecretRepository_GetUserProgressSummary(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos, repotest.WithDisplayName("Hunter"))
	unlockSecretFor(t, repos, user.ID, "piece_01")
	unlockSecretFor(t, repos, user.ID, "piece_02")

	// when
	summary, err := repos.Secret.GetUserProgressSummary(context.Background(), user.ID, testPieceIDs)

	// then
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, user.ID, summary.UserID)
	assert.Equal(t, 2, summary.Pieces)
	assert.Equal(t, "Hunter", summary.DisplayName)
}

func TestSecretRepository_CreateComment_AndGet(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)

	// when
	id := createSecretComment(t, repos, testSecretID, nil, user.ID, "first word")
	comments, err := repos.Secret.GetComments(context.Background(), testSecretID, user.ID, nil)

	// then
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, id, comments[0].ID)
	assert.Equal(t, "first word", comments[0].Body)
	assert.Equal(t, user.ID, comments[0].UserID)
	assert.False(t, comments[0].UserLiked)
	assert.Equal(t, 0, comments[0].LikeCount)
}

func TestSecretRepository_UpdateComment_OnlyOwner(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	owner := repotest.CreateUser(t, repos)
	other := repotest.CreateUser(t, repos)
	id := createSecretComment(t, repos, testSecretID, nil, owner.ID, "original")

	// when
	err := repos.Secret.UpdateComment(context.Background(), id, other.ID, "hijacked")

	// then
	assert.Error(t, err)
}

func TestSecretRepository_DeleteComment_AsAdmin(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createSecretComment(t, repos, testSecretID, nil, user.ID, "bye")

	// when
	require.NoError(t, repos.Secret.DeleteCommentAsAdmin(context.Background(), id))

	// then
	comment, err := repos.Secret.GetCommentByID(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, comment)
}

func TestSecretRepository_LikeComment_Idempotent(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	author := repotest.CreateUser(t, repos)
	liker := repotest.CreateUser(t, repos)
	id := createSecretComment(t, repos, testSecretID, nil, author.ID, "nice")

	// when
	require.NoError(t, repos.Secret.LikeComment(context.Background(), liker.ID, id))
	require.NoError(t, repos.Secret.LikeComment(context.Background(), liker.ID, id))

	// then
	comments, err := repos.Secret.GetComments(context.Background(), testSecretID, liker.ID, nil)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, 1, comments[0].LikeCount)
	assert.True(t, comments[0].UserLiked)

	require.NoError(t, repos.Secret.UnlikeComment(context.Background(), liker.ID, id))
	comments, err = repos.Secret.GetComments(context.Background(), testSecretID, liker.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, comments[0].LikeCount)
	assert.False(t, comments[0].UserLiked)
}

func TestSecretRepository_CountCommentsBySecret(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	createSecretComment(t, repos, testSecretID, nil, user.ID, "one")
	createSecretComment(t, repos, testSecretID, nil, user.ID, "two")
	createSecretComment(t, repos, "another", nil, user.ID, "different")

	// when
	counts, err := repos.Secret.CountCommentsBySecret(context.Background(), []string{testSecretID, "another", "unknown"})

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, counts[testSecretID])
	assert.Equal(t, 1, counts["another"])
	_, hasUnknown := counts["unknown"]
	assert.False(t, hasUnknown)
}

func TestSecretRepository_AddCommentMedia(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	user := repotest.CreateUser(t, repos)
	id := createSecretComment(t, repos, testSecretID, nil, user.ID, "with media")

	// when
	mediaID, err := repos.Secret.AddCommentMedia(context.Background(), id, "/u/a.png", "image/png", "/u/a-thumb.png", 0)

	// then
	require.NoError(t, err)
	assert.NotZero(t, mediaID)

	media, err := repos.Secret.GetCommentMedia(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, media, 1)
	assert.Equal(t, "/u/a.png", media[0].MediaURL)
}

func TestSecretRepository_ExcludesBlockedUsers(t *testing.T) {
	// given
	repos := repotest.NewRepos(t)
	viewer := repotest.CreateUser(t, repos)
	blocked := repotest.CreateUser(t, repos)
	friend := repotest.CreateUser(t, repos)
	createSecretComment(t, repos, testSecretID, nil, blocked.ID, "hidden")
	createSecretComment(t, repos, testSecretID, nil, friend.ID, "visible")

	// when
	rows, err := repos.Secret.GetComments(context.Background(), testSecretID, viewer.ID, []uuid.UUID{blocked.ID})

	// then
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, friend.ID, rows[0].UserID)
}
