package mystery

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetMystery_NonGM_NotSolved_FiltersAttemptsAndClues(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	viewer := uuid.New()
	other := uuid.New()
	row := &model.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []model.MysteryAttemptRow{
		{ID: uuid.New(), UserID: viewer, Body: "mine"},
		{ID: uuid.New(), UserID: other, Body: "not mine"},
	}
	clues := []dto.MysteryClue{
		{ID: 1, Body: "public"},
		{ID: 2, Body: "mine", PlayerID: &viewer},
		{ID: 3, Body: "other", PlayerID: new(uuid.New())},
	}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: id, UserID: viewer}).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, spec.MysteryAttemptQuery{MysteryID: id, ViewerID: viewer}).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

	// when
	got, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.NoError(t, err)
	assert.Len(t, got.Attempts, 1)
	assert.Equal(t, "mine", got.Attempts[0].Body)
	assert.Len(t, got.Clues, 2)
}

func TestGetMystery_FreeForAll_NonGM_SeesAllAttempts(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	viewer := uuid.New()
	other := uuid.New()
	row := &model.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: true}
	attempts := []model.MysteryAttemptRow{
		{ID: uuid.New(), UserID: viewer, Body: "mine"},
		{ID: uuid.New(), UserID: other, Body: "other"},
	}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: id, UserID: viewer}).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, spec.MysteryAttemptQuery{MysteryID: id, ViewerID: viewer}).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

	// when
	got, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.NoError(t, err)
	assert.Len(t, got.Attempts, 2)
}

func TestCreateAttempt_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateAttempt(context.Background(), uuid.New(), uuid.New(), dto.CreateAttemptRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateAttempt_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateAttempt_IsSolvedError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, errors.New("boom"))

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.Error(t, err)
}

func TestCreateAttempt_AlreadySolved(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.ErrorIs(t, err, ErrAlreadySolved)
}

func TestCreateAttempt_PausedBlocksNonAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(true, nil)

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.ErrorIs(t, err, ErrMysteryPaused)
}

func TestCreateAttempt_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateAttempt_ReplyParentNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	parentID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, parentID).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body", ParentID: &parentID})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateAttempt_ReplyByOtherUser_NotAllowed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	parentAuthor := uuid.New()
	parentID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, parentID).Return(parentAuthor, nil)

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body", ParentID: &parentID})

	// then
	require.ErrorIs(t, err, ErrCannotReply)
}

func TestCreateAttempt_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, spec.NewMysteryAttempt{MysteryID: mid, UserID: userID, Body: "body"}).Return(nil, errors.New("boom"))

	// when
	_, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.Error(t, err)
}

func TestCreateAttempt_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	stubAuthor(m, mid, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mid, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, spec.NewMysteryAttempt{MysteryID: mid, UserID: userID, Body: "body"}).Return(&model.MysteryAttemptRow{ID: uuid.New(), AuthorUsername: "u"}, nil)

	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	id, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateAttempt_MentionInTheBodyNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mysteryID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	attemptID := uuid.New()
	mentionedID := uuid.New()
	body := "the culprit is who @alice named"

	stubAuthor(m, mysteryID, authorID)
	m.repo.EXPECT().IsSolved(mock.Anything, mysteryID).Return(false, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: mysteryID, UserID: userID}).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mysteryID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, spec.NewMysteryAttempt{MysteryID: mysteryID, UserID: userID, Body: body}).
		Return(&model.MysteryAttemptRow{ID: attemptID, AuthorUsername: "u"}, nil)
	stubActor(m, userID, "Battler")
	stubMentionOf(m, userID, mentionedID, "alice")
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()

	var wg sync.WaitGroup
	wg.Add(2)

	var mentioned dto.NotifyParams
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			if p.Type == dto.NotifMention {
				mentioned = p
			}
			wg.Done()

			return nil
		})

	// when
	_, err := svc.CreateAttempt(context.Background(), mysteryID, userID, dto.CreateAttemptRequest{Body: body})

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, mysteryID, mentioned.ReferenceID)
	assert.Equal(t, "mystery_attempt:"+attemptID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/mystery/"+mysteryID.String()+"#attempt-"+attemptID.String(), mentioned.EmailLink)
}

func TestDeleteAttempt_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	attemptAuthor := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, id).Return(attemptAuthor, nil)
	m.repo.EXPECT().DeleteAttemptAsAdmin(mock.Anything, id).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionMysteryAttemptDeleteAdmin,
		TargetType: audit.TargetMysteryAttempt,
		TargetID:   id.String(),
		SubjectID:  attemptAuthor,
	}).Return(nil)

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteAttempt_ModeratorDeletingOwnAttempt_WritesNoAuditRow(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, id).Return(userID, nil)
	m.repo.EXPECT().DeleteAttemptAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
	m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestDeleteAttempt_AttemptNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteAttempt_NonAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	m.repo.EXPECT().DeleteAttempt(mock.Anything, spec.MysteryAttemptDeletion{ID: id, UserID: userID}).Return(nil)

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestVoteAttempt_InvalidValue(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.VoteAttempt(context.Background(), uuid.New(), uuid.New(), 2)

	// then
	require.ErrorIs(t, err, ErrInvalidVote)
}

func TestVoteAttempt_AttemptNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	aid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.VoteAttempt(context.Background(), aid, userID, 1)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestVoteAttempt_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	aid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.VoteAttempt(context.Background(), aid, userID, 1)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestVoteAttempt_VoteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	aid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteAttempt(mock.Anything, spec.Vote{UserID: userID, TargetID: aid, Value: 1}).Return(errors.New("boom"))

	// when
	err := svc.VoteAttempt(context.Background(), aid, userID, 1)

	// then
	require.Error(t, err)
}

func TestVoteAttempt_ZeroVote_NoNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	aid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteAttempt(mock.Anything, spec.Vote{UserID: userID, TargetID: aid, Value: 0}).Return(nil)

	// when
	err := svc.VoteAttempt(context.Background(), aid, userID, 0)

	// then
	require.NoError(t, err)
}

func TestVoteAttempt_Upvote_SendsNotification(t *testing.T) {
	synctest.Test(t, testVoteAttemptUpvoteSendsNotification)
}

func testVoteAttemptUpvoteSendsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	aid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	mid := uuid.New()
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteAttempt(mock.Anything, spec.Vote{UserID: userID, TargetID: aid, Value: 1}).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == authorID && p.Type == dto.NotifMysteryVote
	})).Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).Return(nil).Maybe()

	// when
	err := svc.VoteAttempt(context.Background(), aid, userID, 1)

	// then
	require.NoError(t, err)
	wg.Wait()
}
