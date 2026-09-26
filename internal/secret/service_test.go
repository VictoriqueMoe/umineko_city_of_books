package secret

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/secrets"
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
	secretComments *dao.MockCommentDAO[string]
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
	secretComments := dao.NewMockCommentDAO[string](t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notif, dao.CommentDAOs{
		BySlug: map[string]dao.CommentDAO[string]{string(mention.KindSecretComment): secretComments},
	})
	svc := NewService(secretRepo, userSecretRepo, userRepo, authzSvc, blockSvc, notif, mentionSvc, settingsSvc, uploadSvc, mediaProc, hub, contentfilter.New()).(*service)

	secretRepo.EXPECT().GetCommentEntityID(mock.Anything, mock.Anything).Return("", nil).Maybe()
	userRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).Return("").Maybe()
	notif.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	return svc, &testMocks{
		secretRepo:     secretRepo,
		secretComments: secretComments,
		userSecretRepo: userSecretRepo,
		userRepo:       userRepo,
		authz:          authzSvc,
		blockSvc:       blockSvc,
		notif:          notif,
		settings:       settingsSvc,
		upload:         uploadSvc,
	}
}

func witchHunterPieceIDs() []string {
	witchHunter, _ := secrets.Lookup("witchHunter")

	return secrets.PieceIDStrings(witchHunter)
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
	m.secretRepo.EXPECT().CountCommentsBySecret(mock.Anything, []string{"witchHunter"}).Return(map[string]int{}, nil)
	m.secretRepo.EXPECT().GetFirstSolver(mock.Anything, "witchHunter").Return(nil, nil)
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, spec.SecretPieceCount{UserID: viewer, PieceIDs: witchHunterPieceIDs()}).Return(0, nil)
	m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return([]model.SecretLeaderboardRow{
		{UserID: hunterID, Username: "hunter", DisplayName: "Hunter", Pieces: 5},
	}, nil)
	m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return(nil, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.secretRepo.EXPECT().GetComments(mock.Anything, spec.CommentQuery[string]{
		TargetID: "witchHunter",
		ViewerID: viewer,
		Limit:    500,
		Offset:   0,
	}).Return(nil, 0, nil)

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
	m.secretRepo.EXPECT().GetPieceCountForUser(mock.Anything, spec.SecretPieceCount{UserID: viewer, PieceIDs: witchHunterPieceIDs()}).Return(0, nil)
	m.secretRepo.EXPECT().GetProgressLeaderboard(mock.Anything, mock.Anything).Return([]model.SecretLeaderboardRow{
		{UserID: hunterID, Username: "hunter", DisplayName: "Hunter", Pieces: 12},
	}, nil)
	m.userSecretRepo.EXPECT().GetUserIDsWithSecret(mock.Anything, "witchHunter").Return([]uuid.UUID{hunterID}, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.secretRepo.EXPECT().GetComments(mock.Anything, spec.CommentQuery[string]{
		TargetID: "witchHunter",
		ViewerID: viewer,
		Limit:    500,
		Offset:   0,
	}).Return(nil, 0, nil)

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
	m.secretComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[string]{TargetID: "witchHunter", UserID: user, Body: "hello"}).
		Return(&model.CommentRow{ID: uuid.New()}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("skip")).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), "witchHunter", user, dto.CreateSecretCommentRequest{Body: "hello"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	// given a service whose notification mock captures rather than discards
	secretRepo := repository.NewMockSecretRepository(t)
	userSecretRepo := repository.NewMockUserSecretRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notif := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	secretComments := dao.NewMockCommentDAO[string](t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notif, dao.CommentDAOs{
		BySlug: map[string]dao.CommentDAO[string]{string(mention.KindSecretComment): secretComments},
	})
	svc := NewService(secretRepo, userSecretRepo, userRepo, authzSvc, blockSvc, notif, mentionSvc, settingsSvc, uploadSvc, &media.Processor{}, ws.NewHub(), contentfilter.New())

	userID := uuid.New()
	commentID := uuid.New()
	mentionedID := uuid.New()

	secretComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[string]{TargetID: "witchHunter", UserID: userID, Body: "look at this @alice"}).
		Return(&model.CommentRow{ID: commentID}, nil)
	var fanout sync.WaitGroup
	fanout.Add(1)

	secretRepo.EXPECT().GetCommenterIDs(mock.Anything, "witchHunter").
		RunAndReturn(func(context.Context, string, ...*sql.Tx) ([]uuid.UUID, error) {
			fanout.Done()

			return nil, nil
		})
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
	userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	var mentioned dto.NotifyParams
	notif.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			mentioned = p
			wg.Done()

			return nil
		})

	// when
	_, err := svc.CreateComment(context.Background(), "witchHunter", userID, dto.CreateSecretCommentRequest{Body: "look at this @alice"})

	// then
	require.NoError(t, err)
	wg.Wait()
	fanout.Wait()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, uuid.Nil, mentioned.ReferenceID)
	assert.Equal(t, "secret_comment:witchHunter:"+commentID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/secrets/witchHunter#comment-"+commentID.String(), mentioned.EmailLink)
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
	steps := []string{"comment counts", "viewer progress", "solved users", "blocked users", "comments", "comment media"}

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
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, errAt("blocked users")).Maybe()
			m.secretRepo.EXPECT().GetComments(mock.Anything, mock.Anything).Return([]model.CommentRow{{ID: uuid.New()}}, 1, errAt("comments")).Maybe()
			m.secretRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, errAt("comment media")).Maybe()

			// when
			got, err := svc.Get(context.Background(), "witchHunter", viewer)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestSecretCommentLookupFailures(t *testing.T) {
	boom := errors.New("boom")
	missing := errors.Join(errors.New("no row"), dao.ErrNotFound)
	cases := []struct {
		name        string
		lookupErr   error
		blockErr    error
		existingErr error
		call        func(s *service, commentID, userID uuid.UUID) error
		wantErr     error
	}{
		{name: "liking a missing comment is not found", lookupErr: missing, call: secretLike, wantErr: ErrNotFound},
		{name: "a failed lookup before a like is surfaced", lookupErr: boom, call: secretLike, wantErr: boom},
		{name: "a failed block check refuses the like", blockErr: boom, call: secretLike, wantErr: boom},
		{name: "attaching media to a missing comment is not found", lookupErr: missing, call: secretAttach, wantErr: ErrNotFound},
		{name: "a failed lookup before an attachment is surfaced, not reported as not found", lookupErr: boom, call: secretAttach, wantErr: boom},
		{name: "a failed existing media lookup is surfaced instead of reusing a sort position", existingErr: boom, call: secretAttach, wantErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID := uuid.New()
			userID := uuid.New()
			m.secretRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, tc.lookupErr)
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, tc.blockErr).Maybe()
			m.secretRepo.EXPECT().GetCommentMedia(mock.Anything, commentID).Return(nil, tc.existingErr).Maybe()

			// when
			err := tc.call(svc, commentID, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func secretLike(s *service, commentID, userID uuid.UUID) error {
	return s.LikeComment(context.Background(), userID, commentID)
}

func secretAttach(s *service, commentID, userID uuid.UUID) error {
	_, err := s.UploadCommentMedia(context.Background(), commentID, userID, "image/png", "p.png", 3, strings.NewReader("img"), false)

	return err
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
	m.secretRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: user, CommentID: commentID}).Return(nil)

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
	m.secretRepo.EXPECT().
		UpdateCommentBody(mock.Anything, spec.CommentUpdate{
			CommentID: commentID,
			UserID:    user,
			Body:      "new body",
		}).
		Return(nil)

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
	m.secretRepo.EXPECT().
		UpdateCommentBody(mock.Anything, spec.CommentUpdate{
			CommentID: commentID,
			UserID:    user,
			Body:      "admin edit",
			AsAdmin:   true,
		}).
		Return(nil)

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
	m.secretRepo.EXPECT().
		DeleteCommentWithAudit(mock.Anything, spec.CommentDeletion{CommentID: commentID, UserID: user, AsAdmin: true}).
		Return([]string{"/uploads/secrets/a.png", "/uploads/secrets/a-thumb.png"}, nil)
	m.upload.EXPECT().Delete([]string{"/uploads/secrets/a.png", "/uploads/secrets/a-thumb.png"}).Return()

	// when
	err := svc.DeleteComment(context.Background(), commentID, user)

	// then
	require.NoError(t, err)
}

func TestDeleteComment_KeepsFilesWhenTransactionFails(t *testing.T) {
	// given
	svc, m := newTestService(t)
	user := uuid.New()
	commentID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, user, authz.PermDeleteAnyComment).Return(false)
	m.secretRepo.EXPECT().
		DeleteCommentWithAudit(mock.Anything, spec.CommentDeletion{CommentID: commentID, UserID: user}).
		Return(nil, errors.New("rolled back"))

	// when
	err := svc.DeleteComment(context.Background(), commentID, user)

	// then
	require.Error(t, err)
}

func TestBroadcastProgress_UnknownSecretIsNoop(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when (should not panic and make no repo calls)
	svc.BroadcastProgress(context.Background(), "nonsense", uuid.New())
}
