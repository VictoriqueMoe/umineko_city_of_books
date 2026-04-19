package secret

import (
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	secretRepo     *repository.MockSecretRepository
	userSecretRepo *repository.MockUserSecretRepository
	userRepo       *repository.MockUserRepository
	authz          *authz.MockService
	blockSvc       *block.MockService
	notif          *notification.MockService
	settings       *settings.MockService
	upload         *upload.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	secretRepo := repository.NewMockSecretRepository(t)
	userSecretRepo := repository.NewMockUserSecretRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notif := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)

	hub := ws.NewHub()
	mediaProc := &media.Processor{}
	svc := NewService(secretRepo, userSecretRepo, userRepo, authzSvc, blockSvc, notif, settingsSvc, uploadSvc, mediaProc, hub, contentfilter.New()).(*service)

	return svc, &testMocks{
		secretRepo:     secretRepo,
		userSecretRepo: userSecretRepo,
		userRepo:       userRepo,
		authz:          authzSvc,
		blockSvc:       blockSvc,
		notif:          notif,
		settings:       settingsSvc,
		upload:         uploadSvc,
	}
}

func TestList_ReturnsListedSecretsWithStatus(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.secretRepo.EXPECT().
		CountCommentsBySecret(mock.Anything, mock.Anything).
		Return(map[string]int{"witchHunter": 4}, nil)
	m.secretRepo.EXPECT().
		GetFirstSolver(mock.Anything, "witchHunter").
		Return(nil, nil)
	m.secretRepo.EXPECT().
		GetPieceCountForUser(mock.Anything, viewer, mock.Anything).
		Return(3, nil)
	m.secretRepo.EXPECT().
		GetSolversLeaderboard(mock.Anything, mock.Anything).
		Return(nil, nil)

	// when
	got, err := svc.List(context.Background(), viewer)

	// then
	require.NoError(t, err)
	require.Len(t, got.Secrets, 1)
	s := got.Secrets[0]
	assert.Equal(t, "witchHunter", s.ID)
	assert.False(t, s.Solved)
	assert.Equal(t, 3, s.ViewerProgress)
	assert.Equal(t, 12, s.TotalPieces)
	assert.Equal(t, 4, s.CommentCount)
}

func TestList_ShowsSolver(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	winnerID := uuid.New()
	m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, mock.Anything).Return(map[string]int{}, nil)
	m.secretRepo.EXPECT().
		GetFirstSolver(mock.Anything, "witchHunter").
		Return(&repository.SecretSolver{
			UserID:      winnerID,
			Username:    "winner",
			DisplayName: "Winner",
			UnlockedAt:  "2026-01-01T00:00:00Z",
		}, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, viewer, mock.Anything).Return(0, nil)
	m.secretRepo.EXPECT().GetSolversLeaderboard(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.List(context.Background(), viewer)

	// then
	require.NoError(t, err)
	require.Len(t, got.Secrets, 1)
	assert.True(t, got.Secrets[0].Solved)
	require.NotNil(t, got.Secrets[0].Solver)
	assert.Equal(t, winnerID, got.Secrets[0].Solver.ID)
	assert.Equal(t, "2026-01-01T00:00:00Z", got.Secrets[0].SolvedAt)
}

func TestGet_UnknownSecret(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.Get(context.Background(), "nonsense", uuid.Nil)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGet_AssemblesLeaderboardAndComments(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	hunterID := uuid.New()
	m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, []string{"witchHunter"}).Return(map[string]int{}, nil)
	m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, "witchHunter").Return(nil, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, viewer, mock.Anything).Return(0, nil)
	m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return([]repository.SecretLeaderboardRow{
		{UserID: hunterID, Username: "hunter", DisplayName: "Hunter", Pieces: 5},
	}, nil)
	m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return(nil, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.secretRepo.EXPECT().GetComments(mock.Anything, "witchHunter", viewer, []uuid.UUID(nil)).Return(nil, nil)

	// when
	got, err := svc.Get(context.Background(), "witchHunter", viewer)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "The Witch's Epitaph", got.Title)
	assert.NotEmpty(t, got.Riddle)
	require.Len(t, got.Leaderboard, 1)
	assert.Equal(t, hunterID, got.Leaderboard[0].User.ID)
	assert.Equal(t, 5, got.Leaderboard[0].Pieces)
	assert.False(t, got.Leaderboard[0].Solved)
	assert.Empty(t, got.Comments)
}

func TestGet_MarksSolversOnLeaderboard(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	hunterID := uuid.New()
	m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, mock.Anything).Return(map[string]int{}, nil)
	m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, "witchHunter").Return(nil, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, viewer, mock.Anything).Return(0, nil)
	m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return([]repository.SecretLeaderboardRow{
		{UserID: hunterID, Username: "hunter", DisplayName: "Hunter", Pieces: 12},
	}, nil)
	m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return([]uuid.UUID{hunterID}, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.secretRepo.EXPECT().GetComments(mock.Anything, "witchHunter", viewer, []uuid.UUID(nil)).Return(nil, nil)

	// when
	got, err := svc.Get(context.Background(), "witchHunter", viewer)

	// then
	require.NoError(t, err)
	require.Len(t, got.Leaderboard, 1)
	assert.True(t, got.Leaderboard[0].Solved)
}

func TestCreateComment_UnknownSecret(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), "nonsense", uuid.New(), dto.CreateSecretCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), "witchHunter", uuid.New(), dto.CreateSecretCommentRequest{Body: "   "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_Persists(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	m.secretRepo.EXPECT().
		CreateComment(mock.Anything, mock.AnythingOfType("uuid.UUID"), "witchHunter", (*uuid.UUID)(nil), user, "hello").
		Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("skip")).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), "witchHunter", user, dto.CreateSecretCommentRequest{Body: "hello"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestLikeComment_BlocksIfBlocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	authorID := uuid.New()
	commentID := uuid.New()
	m.secretRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, user, authorID).Return(true, nil)

	// when
	err := svc.LikeComment(context.Background(), user, commentID)

	// then
	assert.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikeComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	authorID := uuid.New()
	commentID := uuid.New()
	m.secretRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, user, authorID).Return(false, nil)
	m.secretRepo.EXPECT().LikeComment(mock.Anything, user, commentID).Return(nil)

	// when
	err := svc.LikeComment(context.Background(), user, commentID)

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AuthorPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	commentID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, user, authz.PermEditAnyComment).Return(false)
	m.secretRepo.EXPECT().UpdateComment(mock.Anything, commentID, user, "new body").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), commentID, user, dto.UpdateSecretCommentRequest{Body: "new body"})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AdminPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	commentID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, user, authz.PermEditAnyComment).Return(true)
	m.secretRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, commentID, "admin edit").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), commentID, user, dto.UpdateSecretCommentRequest{Body: "admin edit"})

	// then
	require.NoError(t, err)
}

func TestDeleteComment_AdminPath(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	commentID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, user, authz.PermDeleteAnyComment).Return(true)
	m.secretRepo.EXPECT().DeleteCommentAsAdmin(mock.Anything, commentID).Return(nil)

	// when
	err := svc.DeleteComment(context.Background(), commentID, user)

	// then
	require.NoError(t, err)
}

func TestBroadcastProgress_UnknownSecretIsNoop(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when (should not panic and make no repo calls)
	svc.BroadcastProgress(context.Background(), "nonsense", uuid.New())
}
