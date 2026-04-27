package journal

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	repo         *repository.MockJournalRepository
	userRepo     *repository.MockUserRepository
	authz        *authz.MockService
	blockSvc     *block.MockService
	notifService *notification.MockService
	uploadSvc    *upload.MockService
	settingsSvc  *settings.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	repo := repository.NewMockJournalRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	svc := NewService(repo, userRepo, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentfilter.New()).(*service)
	return svc, &testMocks{
		repo:         repo,
		userRepo:     userRepo,
		authz:        authzSvc,
		blockSvc:     blockSvc,
		notifService: notifSvc,
		uploadSvc:    uploadSvc,
		settingsSvc:  settingsSvc,
	}
}

func validCreateReq() dto.CreateJournalRequest {
	return dto.CreateJournalRequest{
		Title: "Title",
		Body:  "Body",
		Work:  "umineko",
	}
}

func TestCreateJournal_EmptyTitle(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Title = "   "

	// when
	_, err := svc.CreateJournal(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateJournal_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Body = "\n\t"

	// when
	_, err := svc.CreateJournal(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateJournal_NoLimit(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	newID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(0)
	m.repo.EXPECT().Create(mock.Anything, userID, validCreateReq()).Return(newID, nil)

	// when
	got, err := svc.CreateJournal(context.Background(), userID, validCreateReq())

	// then
	require.NoError(t, err)
	assert.Equal(t, newID, got)
}

func TestCreateJournal_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
	m.repo.EXPECT().CountUserJournalsToday(mock.Anything, userID).Return(0, errors.New("db down"))

	// when
	_, err := svc.CreateJournal(context.Background(), userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestCreateJournal_RateLimited(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
	m.repo.EXPECT().CountUserJournalsToday(mock.Anything, userID).Return(5, nil)

	// when
	_, err := svc.CreateJournal(context.Background(), userID, validCreateReq())

	// then
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestCreateJournal_UnderLimitOK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	newID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
	m.repo.EXPECT().CountUserJournalsToday(mock.Anything, userID).Return(2, nil)
	m.repo.EXPECT().Create(mock.Anything, userID, validCreateReq()).Return(newID, nil)

	// when
	got, err := svc.CreateJournal(context.Background(), userID, validCreateReq())

	// then
	require.NoError(t, err)
	assert.Equal(t, newID, got)
}

func TestCreateJournal_RepoCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(0)
	m.repo.EXPECT().Create(mock.Anything, userID, validCreateReq()).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.CreateJournal(context.Background(), userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestGetJournalDetail_NotFoundNil(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, id, viewer).Return(nil, nil)

	// when
	_, err := svc.GetJournalDetail(context.Background(), id, viewer)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetJournalDetail_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, id, viewer).Return(nil, errors.New("db down"))

	// when
	_, err := svc.GetJournalDetail(context.Background(), id, viewer)

	// then
	require.Error(t, err)
}

func TestGetJournalDetail_CommentsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	journal := &dto.JournalResponse{ID: id, Author: dto.UserResponse{ID: uuid.New()}}
	m.repo.EXPECT().GetByID(mock.Anything, id, viewer).Return(journal, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().GetComments(mock.Anything, id, viewer, 500, 0, []uuid.UUID(nil)).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.GetJournalDetail(context.Background(), id, viewer)

	// then
	require.Error(t, err)
}

func TestGetJournalDetail_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	authorID := uuid.New()
	journal := &dto.JournalResponse{ID: id, Author: dto.UserResponse{ID: authorID}}
	commentID := uuid.New()
	rows := []repository.JournalCommentRow{{ID: commentID, UserID: authorID, Body: "hi"}}
	m.repo.EXPECT().GetByID(mock.Anything, id, viewer).Return(journal, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return([]uuid.UUID{uuid.New()}, nil)
	m.repo.EXPECT().GetComments(mock.Anything, id, viewer, 500, 0, mock.Anything).Return(rows, 1, nil)
	m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{commentID}).Return(nil, nil)

	// when
	got, err := svc.GetJournalDetail(context.Background(), id, viewer)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got.Comments, 1)
}

func TestListJournals_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	p := params.NewListParams("new", "", uuid.Nil, "", false, 10, 0)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, p, viewer, []uuid.UUID(nil)).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListJournals(context.Background(), p, viewer)

	// then
	require.Error(t, err)
}

func TestListJournals_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	p := params.NewListParams("new", "", uuid.Nil, "", false, 10, 0)
	journals := []dto.JournalResponse{{ID: uuid.New()}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, p, viewer, []uuid.UUID(nil)).Return(journals, 1, nil)

	// when
	got, err := svc.ListJournals(context.Background(), p, viewer)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 0, got.Offset)
	assert.Equal(t, journals, got.Journals)
}

func TestListJournalsByUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	author := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, mock.MatchedBy(func(p params.ListParams) bool {
		return p.AuthorID == author && p.IncludeArchived && p.Sort == "new" && p.Limit == 10 && p.Offset == 5
	}), viewer, []uuid.UUID(nil)).Return([]dto.JournalResponse{}, 0, nil)

	// when
	_, err := svc.ListJournalsByUser(context.Background(), author, viewer, 10, 5)

	// then
	require.NoError(t, err)
}

func TestListFollowedByUser_DefaultsApplied(t *testing.T) {
	cases := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{"zero limit defaults to 20", 0, 0, 20, 0},
		{"negative limit defaults to 20", -5, 0, 20, 0},
		{"limit clamped to 100", 500, 0, 100, 0},
		{"negative offset clamped", 10, -3, 10, 0},
		{"valid values preserved", 25, 10, 25, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			follower := uuid.New()
			viewer := uuid.New()
			m.repo.EXPECT().ListFollowedByUser(mock.Anything, follower, viewer, tc.wantLimit, tc.wantOffset).Return([]dto.JournalResponse{}, 0, nil)

			// when
			got, err := svc.ListFollowedByUser(context.Background(), follower, viewer, tc.limit, tc.offset)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantLimit, got.Limit)
			assert.Equal(t, tc.wantOffset, got.Offset)
		})
	}
}

func TestListFollowedByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	follower := uuid.New()
	viewer := uuid.New()
	m.repo.EXPECT().ListFollowedByUser(mock.Anything, follower, viewer, 20, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListFollowedByUser(context.Background(), follower, viewer, 0, 0)

	// then
	require.Error(t, err)
}

func TestUpdateJournal_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Body = ""

	// when
	err := svc.UpdateJournal(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateJournal_EmptyTitle(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Title = " "

	// when
	err := svc.UpdateJournal(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateJournal_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyJournal).Return(true)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, validCreateReq()).Return(nil)

	// when
	err := svc.UpdateJournal(context.Background(), id, userID, validCreateReq())

	// then
	require.NoError(t, err)
}

func TestUpdateJournal_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyJournal).Return(false)
	m.repo.EXPECT().Update(mock.Anything, id, userID, validCreateReq()).Return(nil)

	// when
	err := svc.UpdateJournal(context.Background(), id, userID, validCreateReq())

	// then
	require.NoError(t, err)
}

func TestUpdateJournal_RepoErrorBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyJournal).Return(false)
	m.repo.EXPECT().Update(mock.Anything, id, userID, validCreateReq()).Return(errors.New("boom"))

	// when
	err := svc.UpdateJournal(context.Background(), id, userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestDeleteJournal_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyJournal).Return(true)
	m.repo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteJournal(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteJournal_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyJournal).Return(false)
	m.repo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeleteJournal(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteJournal_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyJournal).Return(false)
	m.repo.EXPECT().Delete(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteJournal(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func expectBackgroundCommentNotify(m *testMocks) {
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://base").Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("ignored")).Maybe()
	m.repo.EXPECT().GetTitle(mock.Anything, mock.Anything).Return("title", nil).Maybe()
	m.repo.EXPECT().UpdateLastAuthorActivity(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.repo.EXPECT().GetFollowerIDs(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.notifService.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
}

func TestCreateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), nil, "   ")

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_JournalNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(uuid.Nil, errors.New("nope"))

	// when
	_, err := svc.CreateComment(context.Background(), journalID, userID, nil, "hi")

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_IsArchivedError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, journalID).Return(false, errors.New("boom"))

	// when
	_, err := svc.CreateComment(context.Background(), journalID, userID, nil, "hi")

	// then
	require.Error(t, err)
}

func TestCreateComment_Archived(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, journalID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), journalID, userID, nil, "hi")

	// then
	require.ErrorIs(t, err, ErrArchived)
}

func TestCreateComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, journalID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), journalID, userID, nil, "hi")

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateComment_CreateRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, journalID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateComment(mock.Anything, mock.Anything, journalID, (*uuid.UUID)(nil), userID, "hi").Return(errors.New("boom"))

	// when
	_, err := svc.CreateComment(context.Background(), journalID, userID, nil, "hi")

	// then
	require.Error(t, err)
}

func TestCreateComment_OK_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, journalID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateComment(mock.Anything, mock.Anything, journalID, (*uuid.UUID)(nil), userID, "hi").Return(nil)
	expectBackgroundCommentNotify(m)

	// when
	got, err := svc.CreateComment(context.Background(), journalID, userID, nil, "hi")

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, got)
}

func TestCreateComment_OK_AuthorReplyWithParent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	userID := uuid.New()
	parentID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(userID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, journalID).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	m.repo.EXPECT().CreateComment(mock.Anything, mock.Anything, journalID, &parentID, userID, "hi").Return(nil)
	expectBackgroundCommentNotify(m)

	// when
	got, err := svc.CreateComment(context.Background(), journalID, userID, &parentID, "hi")

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, got)
}

func TestUpdateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), "   ")

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "new body").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, "new body")

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.repo.EXPECT().UpdateComment(mock.Anything, id, userID, "body").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, "body")

	// then
	require.NoError(t, err)
}

func TestUpdateComment_TrimsBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.repo.EXPECT().UpdateComment(mock.Anything, id, userID, "trimmed").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, "  trimmed  ")

	// then
	require.NoError(t, err)
}

func TestUpdateComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.repo.EXPECT().UpdateComment(mock.Anything, id, userID, "body").Return(errors.New("boom"))

	// when
	err := svc.UpdateComment(context.Background(), id, userID, "body")

	// then
	require.Error(t, err)
}

func TestDeleteComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().DeleteCommentAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteComment_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	m.repo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(nil)

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
	m.repo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestLikeComment_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("nope"))

	// when
	err := svc.LikeComment(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestLikeComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.LikeComment(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikeComment_LikeRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().LikeComment(mock.Anything, userID, id).Return(errors.New("boom"))

	// when
	err := svc.LikeComment(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestLikeComment_SelfLikeNoNotify(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(userID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	m.repo.EXPECT().LikeComment(mock.Anything, userID, id).Return(nil)

	// when
	err := svc.LikeComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestLikeComment_OKNotifies(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().LikeComment(mock.Anything, userID, id).Return(nil)
	m.repo.EXPECT().GetCommentJournalID(mock.Anything, id).Return(uuid.New(), nil).Maybe()
	m.repo.EXPECT().GetTitle(mock.Anything, mock.Anything).Return("title", nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://base").Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, nil).Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	err := svc.LikeComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().UnlikeComment(mock.Anything, userID, id).Return(nil)

	// when
	err := svc.UnlikeComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().UnlikeComment(mock.Anything, userID, id).Return(errors.New("boom"))

	// when
	err := svc.UnlikeComment(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_CommentNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("nope"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	otherAuthor := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(otherAuthor, nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestUploadCommentMedia_UploaderError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, "journals", mock.Anything, int64(10), int64(1000), mock.Anything).Return("", errors.New("upload fail"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.Error(t, err)
}

func TestFollowJournal_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("nope"))

	// when
	err := svc.FollowJournal(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestFollowJournal_CannotFollowOwn(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)

	// when
	err := svc.FollowJournal(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, ErrCannotFollowOwn)
}

func TestFollowJournal_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.FollowJournal(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestFollowJournal_FollowRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().Follow(mock.Anything, userID, id).Return(errors.New("boom"))

	// when
	err := svc.FollowJournal(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestFollowJournal_OKNotifies(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().Follow(mock.Anything, userID, id).Return(nil)
	m.repo.EXPECT().GetTitle(mock.Anything, id).Return("title", nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://base").Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, nil).Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	err := svc.FollowJournal(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestUnfollowJournal_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().Unfollow(mock.Anything, userID, id).Return(nil)

	// when
	err := svc.UnfollowJournal(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestUnfollowJournal_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().Unfollow(mock.Anything, userID, id).Return(errors.New("boom"))

	// when
	err := svc.UnfollowJournal(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestArchiveStale_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	// when
	count, err := svc.ArchiveStale(context.Background())

	// then
	require.Error(t, err)
	assert.Zero(t, count)
}

func TestArchiveStale_NoneToArchive(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	count, err := svc.ArchiveStale(context.Background())

	// then
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestArchiveStale_NotifiesAuthors(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id1 := uuid.New()
	id2 := uuid.New()
	author1 := uuid.New()
	m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return([]uuid.UUID{id1, id2}, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://base")
	m.repo.EXPECT().GetAuthorID(mock.Anything, id1).Return(author1, nil)
	m.repo.EXPECT().GetTitle(mock.Anything, id1).Return("title1", nil)
	m.repo.EXPECT().GetAuthorID(mock.Anything, id2).Return(uuid.Nil, errors.New("skip"))
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == author1 && p.Type == dto.NotifJournalArchived && p.ReferenceID == id1
	})).Return(nil)

	// when
	count, err := svc.ArchiveStale(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
