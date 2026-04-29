package announcement_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/announcement"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type harness struct {
	repo         *repository.MockAnnouncementRepository
	userRepo     *repository.MockUserRepository
	blockSvc     *block.MockService
	notifService *notification.MockService
	settingsSvc  *settings.MockService
	authzSvc     *authz.MockService
	uploadSvc    *upload.MockService
	hub          *ws.Hub
	svc          announcement.Service
}

func newHarness(t *testing.T) *harness {
	repo := repository.NewMockAnnouncementRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	blockSvc := block.NewMockService(t)
	notifService := notification.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	authzSvc := authz.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	hub := ws.NewHub()

	uploader := media.NewUploader(uploadSvc, settingsSvc, nil)

	// goroutine notifications fire with background context; use Maybe so timing isn't asserted
	settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).Return("http://test").Maybe()
	notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	return &harness{
		repo:         repo,
		userRepo:     userRepo,
		blockSvc:     blockSvc,
		notifService: notifService,
		settingsSvc:  settingsSvc,
		authzSvc:     authzSvc,
		uploadSvc:    uploadSvc,
		hub:          hub,
		svc:          announcement.NewService(repo, userRepo, blockSvc, notifService, settingsSvc, authzSvc, hub, uploader),
	}
}

func TestService_List_BuildsResponse(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	authorID := uuid.New()
	h.repo.EXPECT().List(mock.Anything, 20, 0).Return([]repository.AnnouncementRow{
		{ID: annID, Title: "Welcome", Body: "hi", AuthorID: authorID, AuthorUsername: "beato"},
	}, 1, nil)

	// when
	resp, err := h.svc.List(context.Background(), 20, 0)

	// then
	require.NoError(t, err)
	require.Len(t, resp.Announcements, 1)
	assert.Equal(t, annID, resp.Announcements[0].ID)
	assert.Equal(t, 1, resp.Total)
}

func TestService_GetDetail_NotFound(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, annID).Return(nil, nil)

	// when
	_, err := h.svc.GetDetail(context.Background(), annID, uuid.Nil)

	// then
	assert.ErrorIs(t, err, announcement.ErrNotFound)
}

func TestService_GetDetail_BuildsTreeFromComments(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	viewerID := uuid.New()
	c1 := uuid.New()
	c2 := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, annID).Return(&repository.AnnouncementRow{ID: annID, Title: "T"}, nil)
	h.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	h.repo.EXPECT().GetComments(mock.Anything, annID, viewerID, 500, 0, []uuid.UUID(nil)).
		Return([]repository.AnnouncementCommentRow{
			{ID: c1, AnnouncementID: annID, Body: "parent"},
			{ID: c2, AnnouncementID: annID, ParentID: &c1, Body: "reply"},
		}, 2, nil)
	h.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{c1, c2}).
		Return(map[uuid.UUID][]repository.AnnouncementCommentMediaRow{}, nil)

	// when
	resp, err := h.svc.GetDetail(context.Background(), annID, viewerID)

	// then
	require.NoError(t, err)
	require.Len(t, resp.Comments, 1)
	assert.Equal(t, c1, resp.Comments[0].ID)
	require.Len(t, resp.Comments[0].Replies, 1)
	assert.Equal(t, c2, resp.Comments[0].Replies[0].ID)
}

func TestService_GetLatest_NoneReturnsNil(t *testing.T) {
	// given
	h := newHarness(t)
	h.repo.EXPECT().GetLatest(mock.Anything).Return(nil, nil)

	// when
	resp, err := h.svc.GetLatest(context.Background())

	// then
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestService_GetLatest_Returns(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	h.repo.EXPECT().GetLatest(mock.Anything).Return(&repository.AnnouncementRow{ID: annID, Title: "Latest"}, nil)

	// when
	resp, err := h.svc.GetLatest(context.Background())

	// then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, annID, resp.ID)
}

func TestService_Create_RejectsEmpty(t *testing.T) {
	// given
	h := newHarness(t)

	// when
	_, err := h.svc.Create(context.Background(), uuid.New(), "", "")

	// then
	assert.ErrorIs(t, err, announcement.ErrEmptyTitleOrBody)
}

func TestService_Create_OK(t *testing.T) {
	// given
	h := newHarness(t)
	userID := uuid.New()
	h.repo.EXPECT().Create(mock.Anything, mock.AnythingOfType("uuid.UUID"), userID, "t", "b").Return(nil)

	// when
	id, err := h.svc.Create(context.Background(), userID, "t", "b")

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestService_Update_RejectsEmpty(t *testing.T) {
	// given
	h := newHarness(t)

	// when
	err := h.svc.Update(context.Background(), uuid.New(), "", "")

	// then
	assert.ErrorIs(t, err, announcement.ErrEmptyTitleOrBody)
}

func TestService_Update_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	h.repo.EXPECT().Update(mock.Anything, id, "t", "b").Return(nil)

	// when
	err := h.svc.Update(context.Background(), id, "t", "b")

	// then
	assert.NoError(t, err)
}

func TestService_Delete_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	h.repo.EXPECT().Delete(mock.Anything, id).Return(nil)

	// when
	err := h.svc.Delete(context.Background(), id)

	// then
	assert.NoError(t, err)
}

func TestService_SetPinned_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	h.repo.EXPECT().SetPinned(mock.Anything, id, true).Return(nil)

	// when
	err := h.svc.SetPinned(context.Background(), id, true)

	// then
	assert.NoError(t, err)
}

func TestService_CreateComment_RejectsEmptyBody(t *testing.T) {
	// given
	h := newHarness(t)

	// when
	_, err := h.svc.CreateComment(context.Background(), uuid.New(), uuid.New(), nil, "")

	// then
	assert.ErrorIs(t, err, announcement.ErrEmptyBody)
}

func TestService_CreateComment_AnnouncementNotFound(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, annID).Return(nil, nil)

	// when
	_, err := h.svc.CreateComment(context.Background(), annID, uuid.New(), nil, "hello")

	// then
	assert.ErrorIs(t, err, announcement.ErrNotFound)
}

func TestService_CreateComment_BlockedAuthor(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, annID).
		Return(&repository.AnnouncementRow{ID: annID, AuthorID: authorID}, nil)
	h.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := h.svc.CreateComment(context.Background(), annID, userID, nil, "hi")

	// then
	assert.ErrorIs(t, err, announcement.ErrBlocked)
}

func TestService_CreateComment_OK(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, annID).
		Return(&repository.AnnouncementRow{ID: annID, AuthorID: authorID, Title: "Welcome"}, nil)
	h.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	h.repo.EXPECT().CreateComment(mock.Anything, mock.AnythingOfType("uuid.UUID"), annID, (*uuid.UUID)(nil), userID, "hello").Return(nil)
	h.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Beato"}, nil).Maybe()

	// when
	id, err := h.svc.CreateComment(context.Background(), annID, userID, nil, "hello")

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestService_UpdateComment_AsAuthor(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	userID := uuid.New()
	h.authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	h.repo.EXPECT().UpdateComment(mock.Anything, id, userID, "x").Return(nil)

	// when
	err := h.svc.UpdateComment(context.Background(), id, userID, "x")

	// then
	assert.NoError(t, err)
}

func TestService_UpdateComment_AsAdmin(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	userID := uuid.New()
	h.authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	h.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "x").Return(nil)

	// when
	err := h.svc.UpdateComment(context.Background(), id, userID, "x")

	// then
	assert.NoError(t, err)
}

func TestService_UpdateComment_RejectsEmpty(t *testing.T) {
	// given
	h := newHarness(t)

	// when
	err := h.svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), "")

	// then
	assert.ErrorIs(t, err, announcement.ErrEmptyBody)
}

func TestService_UpdateComment_NonAuthorIsForbidden(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	userID := uuid.New()
	h.authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	h.repo.EXPECT().UpdateComment(mock.Anything, id, userID, "x").Return(errors.New("not yours"))

	// when
	err := h.svc.UpdateComment(context.Background(), id, userID, "x")

	// then
	assert.ErrorIs(t, err, announcement.ErrForbidden)
}

func TestService_DeleteComment_AsAuthor(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	userID := uuid.New()
	h.authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	h.repo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(nil)

	// when
	err := h.svc.DeleteComment(context.Background(), id, userID)

	// then
	assert.NoError(t, err)
}

func TestService_DeleteComment_AsAdmin(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	userID := uuid.New()
	h.authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	h.repo.EXPECT().DeleteCommentAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := h.svc.DeleteComment(context.Background(), id, userID)

	// then
	assert.NoError(t, err)
}

func TestService_DeleteComment_NonAuthorIsForbidden(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	userID := uuid.New()
	h.authzSvc.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	h.repo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(errors.New("not yours"))

	// when
	err := h.svc.DeleteComment(context.Background(), id, userID)

	// then
	assert.ErrorIs(t, err, announcement.ErrForbidden)
}

func TestService_LikeComment_NotFound(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	h.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("not found"))

	// when
	err := h.svc.LikeComment(context.Background(), uuid.New(), commentID)

	// then
	assert.ErrorIs(t, err, announcement.ErrCommentNotFound)
}

func TestService_LikeComment_Blocked(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	h.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	h.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := h.svc.LikeComment(context.Background(), userID, commentID)

	// then
	assert.ErrorIs(t, err, announcement.ErrBlocked)
}

func TestService_LikeComment_RepoBlockedSentinelMaps(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	h.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	h.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	h.repo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(block.ErrUserBlocked)

	// when
	err := h.svc.LikeComment(context.Background(), userID, commentID)

	// then
	assert.ErrorIs(t, err, announcement.ErrBlocked)
}

func TestService_LikeComment_OK(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	authorID := uuid.New()
	userID := uuid.New()
	h.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	h.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	h.repo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)
	h.repo.EXPECT().GetCommentAnnouncementID(mock.Anything, commentID).Return(uuid.New(), nil).Maybe()
	h.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "X"}, nil).Maybe()

	// when
	err := h.svc.LikeComment(context.Background(), userID, commentID)

	// then
	assert.NoError(t, err)
}

func TestService_UnlikeComment_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	userID := uuid.New()
	h.repo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	err := h.svc.UnlikeComment(context.Background(), userID, commentID)

	// then
	assert.NoError(t, err)
}

func TestService_UploadCommentMedia_NotFound(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	h.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("nope"))

	// when
	_, err := h.svc.UploadCommentMedia(context.Background(), commentID, uuid.New(), "image/png", 10, strings.NewReader("x"))

	// then
	assert.ErrorIs(t, err, announcement.ErrCommentNotFound)
}

func TestService_UploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	h := newHarness(t)
	commentID := uuid.New()
	otherID := uuid.New()
	h.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(otherID, nil)

	// when
	_, err := h.svc.UploadCommentMedia(context.Background(), commentID, uuid.New(), "image/png", 10, strings.NewReader("x"))

	// then
	assert.ErrorIs(t, err, announcement.ErrForbidden)
}
