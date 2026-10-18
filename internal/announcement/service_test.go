package announcement_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"umineko_city_of_books/internal/announcement"
	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
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
	auditRepo    *repository.MockAuditLogRepository
	blockSvc     *block.MockService
	notifService *notification.MockService
	commentSvc   *comment.MockService
	uploadSvc    *upload.MockService
	svc          announcement.Service
}

func newHarness(t *testing.T) *harness {
	h := &harness{
		repo:         repository.NewMockAnnouncementRepository(t),
		userRepo:     repository.NewMockUserRepository(t),
		auditRepo:    repository.NewMockAuditLogRepository(t),
		blockSvc:     block.NewMockService(t),
		notifService: notification.NewMockService(t),
		commentSvc:   comment.NewMockService(t),
		uploadSvc:    upload.NewMockService(t),
	}

	mentionSvc := mention.NewService(h.userRepo, h.blockSvc, h.notifService)
	h.svc = announcement.NewService(h.repo, h.auditRepo, mentionSvc, h.commentSvc, ws.NewHub(), h.uploadSvc, nil)

	return h
}

func TestService_List_BuildsResponse(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	authorID := uuid.New()
	h.repo.EXPECT().List(mock.Anything, spec.AnnouncementListQuery{Limit: 20, Offset: 0}).Return([]model.AnnouncementRow{
		{ID: annID, Title: "Welcome", Body: "hi", AuthorID: authorID, AuthorUsername: "beato"},
	}, 1, nil)

	// when
	resp, err := h.svc.List(context.Background(), bounds.NewPage(20, 0))

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

func TestService_GetDetail_CarriesTheAnnouncementCommentThread(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	viewerID := uuid.New()
	thread := []dto.CommentResponse{{ID: uuid.New(), Body: "parent", Replies: []dto.CommentResponse{{ID: uuid.New(), Body: "reply"}}}}
	h.repo.EXPECT().GetByID(mock.Anything, annID).Return(&model.AnnouncementRow{ID: annID, Title: "T"}, nil)
	h.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindAnnouncement, ID: annID}, viewerID).Return(thread, nil).Once()

	// when
	resp, err := h.svc.GetDetail(context.Background(), annID, viewerID)

	// then
	require.NoError(t, err)
	assert.Equal(t, thread, resp.Comments)
}

func TestService_GetDetail_AFailedThreadIsSurfacedInsteadOfHidingComments(t *testing.T) {
	// given
	h := newHarness(t)
	annID := uuid.New()
	boom := errors.New("boom")
	h.repo.EXPECT().GetByID(mock.Anything, annID).Return(&model.AnnouncementRow{ID: annID, Title: "T"}, nil)
	h.commentSvc.EXPECT().Thread(mock.Anything, mock.Anything, mock.Anything).Return(nil, boom).Once()

	// when
	got, err := h.svc.GetDetail(context.Background(), annID, uuid.New())

	// then
	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
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
	h.repo.EXPECT().GetLatest(mock.Anything).Return(&model.AnnouncementRow{ID: annID, Title: "Latest"}, nil)

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
	annID := uuid.New()
	h.repo.EXPECT().Create(mock.Anything, spec.NewAnnouncement{AuthorID: userID, Title: "t", Body: "b"}).Return(&model.AnnouncementRow{ID: annID}, nil)
	h.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionAnnouncementCreate,
		TargetType: audit.TargetAnnouncement,
		TargetID:   annID.String(),
		Details:    "title=t",
		SubjectID:  userID,
	}).Return(nil)

	// when
	id, err := h.svc.Create(context.Background(), userID, "t", "b")

	// then
	require.NoError(t, err)
	assert.Equal(t, annID, id)
}

func TestService_Create_MentionInTheBodyNotifiesTheNamedUser(t *testing.T) {
	// given
	h := newHarness(t)
	userID := uuid.New()
	annID := uuid.New()
	mentionedID := uuid.New()

	h.repo.EXPECT().Create(mock.Anything, spec.NewAnnouncement{AuthorID: userID, Title: "Maintenance", Body: "we are down, ask @alice"}).
		Return(&model.AnnouncementRow{ID: annID}, nil)
	h.auditRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	h.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Beato"}, nil)
	h.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	h.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	var mentioned dto.NotifyParams
	h.notifService.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			mentioned = p
			wg.Done()

			return nil
		})

	// when
	_, err := h.svc.Create(context.Background(), userID, "Maintenance", "we are down, ask @alice")

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, annID, mentioned.ReferenceID)
	assert.Equal(t, "announcement", mentioned.ReferenceType)
	assert.Equal(t, "/announcements/"+annID.String(), mentioned.EmailLink)
}

func TestService_Update_RejectsEmpty(t *testing.T) {
	// given
	h := newHarness(t)

	// when
	err := h.svc.Update(context.Background(), uuid.New(), uuid.New(), "", "")

	// then
	assert.ErrorIs(t, err, announcement.ErrEmptyTitleOrBody)
}

func TestService_Update_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	actor := uuid.New()
	author := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.AnnouncementRow{ID: id, Title: "old", AuthorID: author}, nil)
	h.repo.EXPECT().Update(mock.Anything, spec.AnnouncementUpdate{ID: id, Title: "t", Body: "b"}).Return(nil)
	h.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionAnnouncementUpdate,
		TargetType: audit.TargetAnnouncement,
		TargetID:   id.String(),
		Details:    "title=old -> t",
		SubjectID:  author,
	}).Return(nil)

	// when
	err := h.svc.Update(context.Background(), actor, id, "t", "b")

	// then
	assert.NoError(t, err)
}

func TestService_Update_MissingAnnouncement(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, nil)

	// when
	err := h.svc.Update(context.Background(), uuid.New(), id, "t", "b")

	// then
	assert.ErrorIs(t, err, announcement.ErrNotFound)
}

func TestService_Delete_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	actor := uuid.New()
	author := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.AnnouncementRow{ID: id, Title: "Doomed", AuthorID: author}, nil)
	h.repo.EXPECT().DeleteWithMedia(mock.Anything, id).Return(nil, nil)
	h.uploadSvc.EXPECT().Delete().Return()
	h.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionAnnouncementDelete,
		TargetType: audit.TargetAnnouncement,
		TargetID:   id.String(),
		Details:    "title=Doomed",
		SubjectID:  author,
	}).Return(nil)

	// when
	err := h.svc.Delete(context.Background(), actor, id)

	// then
	assert.NoError(t, err)
}

func TestService_Delete_UnlinksCommentMediaAfterCommit(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.AnnouncementRow{ID: id, Title: "Doomed"}, nil)
	h.repo.EXPECT().DeleteWithMedia(mock.Anything, id).Return([]string{"/uploads/announcements/a.png", "/uploads/announcements/a-thumb.png"}, nil)
	h.uploadSvc.EXPECT().Delete([]string{"/uploads/announcements/a.png", "/uploads/announcements/a-thumb.png"}).Return()
	h.auditRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	// when
	err := h.svc.Delete(context.Background(), uuid.New(), id)

	// then
	assert.NoError(t, err)
}

func TestService_Delete_KeepsFilesWhenTransactionFails(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.AnnouncementRow{ID: id, Title: "Doomed"}, nil)
	h.repo.EXPECT().DeleteWithMedia(mock.Anything, id).Return(nil, errors.New("rolled back"))

	// when
	err := h.svc.Delete(context.Background(), uuid.New(), id)

	// then
	assert.Error(t, err)
}

func TestService_SetPinned_Delegates(t *testing.T) {
	// given
	h := newHarness(t)
	id := uuid.New()
	actor := uuid.New()
	author := uuid.New()
	h.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.AnnouncementRow{ID: id, Title: "Welcome", AuthorID: author}, nil)
	h.repo.EXPECT().SetPinned(mock.Anything, spec.AnnouncementPinUpdate{ID: id, Pinned: true}).Return(nil)
	h.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    actor,
		Action:     audit.ActionAnnouncementPin,
		TargetType: audit.TargetAnnouncement,
		TargetID:   id.String(),
		Details:    "title=Welcome pinned=true",
		SubjectID:  author,
	}).Return(nil)

	// when
	err := h.svc.SetPinned(context.Background(), actor, id, true)

	// then
	assert.NoError(t, err)
}
