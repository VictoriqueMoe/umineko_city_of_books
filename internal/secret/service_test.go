package secret

import (
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/sidebar"
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
	commentRepo    *repository.MockCommentRepository
	notif          *notification.MockService
	commentSvc     *comment.MockService
	activity       *sidebar.MockActivityBumper
}

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		secretRepo:     repository.NewMockSecretRepository(t),
		userSecretRepo: repository.NewMockUserSecretRepository(t),
		userRepo:       repository.NewMockUserRepository(t),
		commentRepo:    repository.NewMockCommentRepository(t),
		notif:          notification.NewMockService(t),
		commentSvc:     comment.NewMockService(t),
		activity:       sidebar.NewMockActivityBumper(t),
	}

	svc := NewService(m.secretRepo, m.userSecretRepo, m.userRepo, m.commentRepo, m.notif, m.commentSvc, ws.NewHub(), m.activity).(*service)

	return svc, m
}

func witchHunterPieceIDs() []string {
	witchHunter, _ := secrets.Lookup("witchHunter")

	return secrets.PieceIDStrings(witchHunter)
}

func witchHunterTarget() spec.CommentTarget {
	return spec.CommentTarget{Kind: spec.CommentKindSecret, Key: "witchHunter"}
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
		GetPieceCountForUser(mock.Anything, spec.SecretPieceCount{UserID: viewer, PieceIDs: witchHunterPieceIDs()}).
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
		Return(&model.SecretSolver{
			UserID:      winnerID,
			Username:    "winner",
			DisplayName: "Winner",
			UnlockedAt:  "2026-01-01T00:00:00Z",
		}, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, spec.SecretPieceCount{UserID: viewer, PieceIDs: witchHunterPieceIDs()}).Return(0, nil)
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
	thread := []dto.CommentResponse{{ID: uuid.New(), Body: "found the first piece"}}
	m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, []string{"witchHunter"}).Return(map[string]int{}, nil)
	m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, "witchHunter").Return(nil, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, spec.SecretPieceCount{UserID: viewer, PieceIDs: witchHunterPieceIDs()}).Return(0, nil)
	m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return([]model.SecretLeaderboardRow{
		{UserID: hunterID, Username: "hunter", DisplayName: "Hunter", Pieces: 5},
	}, nil)
	m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return(nil, nil)
	m.commentSvc.EXPECT().Thread(mock.Anything, witchHunterTarget(), viewer).Return(thread, nil).Once()

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
	assert.Equal(t, thread, got.Comments)
}

func TestGet_MarksSolversOnLeaderboard(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	hunterID := uuid.New()
	m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, mock.Anything).Return(map[string]int{}, nil)
	m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, "witchHunter").Return(nil, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, spec.SecretPieceCount{UserID: viewer, PieceIDs: witchHunterPieceIDs()}).Return(0, nil)
	m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return([]model.SecretLeaderboardRow{
		{UserID: hunterID, Username: "hunter", DisplayName: "Hunter", Pieces: 12},
	}, nil)
	m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return([]uuid.UUID{hunterID}, nil)
	m.commentSvc.EXPECT().Thread(mock.Anything, witchHunterTarget(), viewer).Return([]dto.CommentResponse{}, nil).Once()

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
	_, err := svc.CreateComment(context.Background(), "nonsense", uuid.New(), dto.CreateCommentRequest{Body: "hi"}, nil)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_AFailedCreateBumpsNothingAndFansNothingOut(t *testing.T) {
	// given
	svc, m := newTestService(t)
	boom := errors.New("boom")
	req := dto.CreateCommentRequest{Body: "hello"}
	m.commentSvc.EXPECT().Create(mock.Anything, witchHunterTarget(), mock.Anything, req, mock.Anything).Return(uuid.Nil, boom).Once()

	// when
	id, err := svc.CreateComment(context.Background(), "witchHunter", uuid.New(), req, nil)

	// then
	require.ErrorIs(t, err, boom)
	assert.Equal(t, uuid.Nil, id)
}

func TestCreateComment_FansOutToOtherCommentersAndBumpsTheSecretsSection(t *testing.T) {
	// given a reply on a hunt that the actor, the parent's author and one other hunter have all commented on
	svc, m := newTestService(t)
	actorID, parentAuthorID, otherID := uuid.New(), uuid.New(), uuid.New()
	parentID, commentID := uuid.New(), uuid.New()
	req := dto.CreateCommentRequest{Body: "the epitaph points north", ParentID: &parentID}

	m.commentSvc.EXPECT().Create(mock.Anything, witchHunterTarget(), actorID, req, mock.Anything).Return(commentID, nil).Once()
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeySecrets).Return().Once()
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "Battler"}, nil)
	m.commentRepo.EXPECT().GetCommentAuthorID(mock.Anything, spec.CommentRef{Kind: spec.CommentKindSecret, CommentID: parentID}).Return(parentAuthorID, nil)
	m.secretRepo.EXPECT().GetCommenterIDs(mock.Anything, "witchHunter").Return([]uuid.UUID{actorID, parentAuthorID, otherID}, nil)

	notified := make(chan dto.NotifyParams, 3)
	m.notif.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			notified <- p
			return nil
		})

	// when
	id, err := svc.CreateComment(context.Background(), "witchHunter", actorID, req, nil)

	// then only the other hunter hears about it, the actor and the parent's author do not
	require.NoError(t, err)
	assert.Equal(t, commentID, id)

	select {
	case p := <-notified:
		assert.Equal(t, otherID, p.RecipientID)
		assert.Equal(t, dto.NotifSecretCommented, p.Type)
		assert.Equal(t, "secret_comment:witchHunter:"+commentID.String(), p.ReferenceType)
		assert.Equal(t, "/secrets/witchHunter#comment-"+commentID.String(), p.EmailLink)
	case <-time.After(5 * time.Second):
		t.Fatal("the commenter fan-out never reached the notification service")
	}

	select {
	case p := <-notified:
		t.Fatalf("an unexpected recipient %s was notified", p.RecipientID)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestList_AFailedReadIsSurfacedInsteadOfShowingZeroes(t *testing.T) {
	steps := []string{"comment counts", "viewer progress", "solvers leaderboard"}

	for _, failAt := range steps {
		t.Run("the "+failAt+" read failing", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewer := uuid.New()
			boom := errors.New("boom")
			errAt := func(step string) error {
				if step == failAt {
					return boom
				}

				return nil
			}
			m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, mock.Anything).Return(map[string]int{}, errAt("comment counts")).Maybe()
			m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
			m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, mock.Anything).Return(0, errAt("viewer progress")).Maybe()
			m.secretRepo.EXPECT().GetSolversLeaderboard(mock.Anything, mock.Anything).Return(nil, errAt("solvers leaderboard")).Maybe()

			// when
			got, err := svc.List(context.Background(), viewer)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestGet_AFailedReadIsSurfacedInsteadOfRenderingAnEmptySection(t *testing.T) {
	steps := []string{"comment counts", "viewer progress", "solved users", "comments"}

	for _, failAt := range steps {
		t.Run("the "+failAt+" read failing", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewer := uuid.New()
			boom := errors.New("boom")
			errAt := func(step string) error {
				if step == failAt {
					return boom
				}

				return nil
			}
			m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, mock.Anything).Return(map[string]int{}, errAt("comment counts")).Maybe()
			m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
			m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, mock.Anything).Return(0, errAt("viewer progress")).Maybe()
			m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
			m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return(nil, errAt("solved users")).Maybe()
			m.commentSvc.EXPECT().Thread(mock.Anything, mock.Anything, viewer).Return(nil, errAt("comments")).Maybe()

			// when
			got, err := svc.Get(context.Background(), "witchHunter", viewer)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestBroadcastProgress_UnknownSecretIsNoop(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when (should not panic and make no repo calls)
	svc.BroadcastProgress(context.Background(), "nonsense", uuid.New())
}
