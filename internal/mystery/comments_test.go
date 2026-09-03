package mystery

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetMystery_Solved_LoadsCommentsAndWinner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	viewer := uuid.New()
	winnerID := uuid.New()
	row := &repository.MysteryRow{
		ID:                id,
		UserID:            author,
		Solved:            true,
		WinnerID:          &winnerID,
		WinnerUsername:    new("win"),
		WinnerDisplayName: new("Winner"),
		WinnerAvatarURL:   new(""),
		WinnerRole:        new("user"),
	}
	commentID := uuid.New()
	comments := []repository.CommentRow{{ID: commentID, UserID: author, Body: "post"}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, id, viewer).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(nil, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().GetComments(mock.Anything, id, viewer, 500, 0, []uuid.UUID(nil)).Return(comments, 1, nil)
	m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{commentID}).Return(nil, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

	// when
	got, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.Winner)
	assert.Equal(t, winnerID, got.Winner.ID)
	assert.Len(t, got.Comments, 1)
}

func TestCreateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_IsSolvedError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, errors.New("boom"))

	// when
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_NotSolved(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)

	// when
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotSolved)
}

func TestCreateComment_AuthorError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)
	stubAuthor(m, mid, authorID)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)
	stubAuthor(m, mid, authorID)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.comments.EXPECT().CreateComment(mock.Anything, mid, (*uuid.UUID)(nil), userID, "hi").Return(nil, errors.New("boom"))

	// when
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestCreateComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)
	stubAuthor(m, mid, authorID)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.comments.EXPECT().CreateComment(mock.Anything, mid, (*uuid.UUID)(nil), userID, "hi").Return(&repository.CommentRow{ID: uuid.New()}, nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "D"}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	commentID := uuid.New()
	mentionedID := uuid.New()

	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)
	stubAuthor(m, mid, authorID)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.comments.EXPECT().
		CreateComment(mock.Anything, mid, (*uuid.UUID)(nil), userID, "look at this @alice").
		Return(&repository.CommentRow{ID: commentID}, nil)
	stubActor(m, userID, "Battler")
	stubMentionOf(m, userID, mentionedID, "alice")

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
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "look at this @alice"})

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, "mystery_comment:"+commentID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/mystery/"+mid.String()+"#comment-"+commentID.String(), mentioned.EmailLink)
}

func TestCreateComment_Reply_NotifiesParentAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	parentID := uuid.New()
	parentAuthor := uuid.New()
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(true, nil)
	stubAuthor(m, mid, authorID)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.comments.EXPECT().CreateComment(mock.Anything, mid, &parentID, userID, "hi").Return(&repository.CommentRow{ID: uuid.New()}, nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "D"}, nil).Maybe()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, parentID).Return(parentAuthor, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	_, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi", ParentID: &parentID})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateComment_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	commentAuthor := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(commentAuthor, nil)
	m.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "new").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryCommentUpdateAdmin,
		TargetType: repository.AuditTargetMysteryComment,
		TargetID:   id.String(),
		SubjectID:  commentAuthor,
	}).Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "new"})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_ModeratorEditingOwnComment_WritesNoAuditRow(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(userID, nil)
	m.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "new").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "new"})

	// then
	require.NoError(t, err)
	m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestUpdateComment_Owner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.repo.EXPECT().UpdateComment(mock.Anything, id, userID, "new").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "new"})

	// then
	require.NoError(t, err)
}

func TestDeleteComment_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, repository.MysteryCommentDelete{ID: id, UserID: userID, AsAdmin: true}).Return([]string{"/uploads/mystery/c.png", "/uploads/mystery/c_thumb.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mystery/c.png", "/uploads/mystery/c_thumb.png"})

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteComment_Owner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	m.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, repository.MysteryCommentDelete{ID: id, UserID: userID}).Return([]string{"/uploads/mystery/c.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mystery/c.png"})

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	m.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, repository.MysteryCommentDelete{ID: id, UserID: userID}).Return(nil, errors.New("boom"))

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestLikeComment_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	cid := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, cid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.LikeComment(context.Background(), userID, cid)

	// then
	require.Error(t, err)
}

func TestLikeComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	cid := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, cid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.LikeComment(context.Background(), userID, cid)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikeComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	cid := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, cid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().LikeComment(mock.Anything, userID, cid).Return(nil)

	// when
	err := svc.LikeComment(context.Background(), userID, cid)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	cid := uuid.New()
	m.repo.EXPECT().UnlikeComment(mock.Anything, userID, cid).Return(nil)

	// when
	err := svc.UnlikeComment(context.Background(), userID, cid)

	// then
	require.NoError(t, err)
}

func TestUploadCommentMedia_CommentNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	cid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, cid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), cid, userID, "image/png", "photo.png", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	cid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, cid).Return(uuid.New(), nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), cid, userID, "image/png", "photo.png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}
