package post

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/db/dbtest"
	"umineko_city_of_books/internal/dto"
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

type testMocks struct {
	db          *sql.DB
	postRepo    *repository.MockPostRepository
	userRepo    *repository.MockUserRepository
	roleRepo    *repository.MockRoleRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
	hub         *ws.Hub
}

func newTestService(t *testing.T) (*service, *testMocks) {
	t.Helper()
	db, _ := dbtest.NewEmptyDatabase(t)

	postRepo := repository.NewMockPostRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	roleRepo := repository.NewMockRoleRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()

	svc := NewService(db, postRepo, userRepo, roleRepo, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, hub, contentfilter.New()).(*service)

	return svc, &testMocks{
		db:          db,
		postRepo:    postRepo,
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		hub:         hub,
	}
}

func waitOrFail(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("timed out waiting for goroutine")
	}
}

func validCreatePostReq() dto.CreatePostRequest {
	return dto.CreatePostRequest{
		Corner: "general",
		Body:   "hello",
	}
}

func expectBackgroundSocial(m *testMocks) {
	m.postRepo.EXPECT().AddEmbed(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.postRepo.EXPECT().DeleteEmbeds(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.postRepo.EXPECT().IncrementShareCount(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.postRepo.EXPECT().DecrementShareCount(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.postRepo.EXPECT().GetCommentPostID(mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("ignored")).Maybe()
	m.userRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("ignored")).Maybe()
	m.userRepo.EXPECT().GetByUsername(mock.Anything, mock.Anything).Return(nil, errors.New("ignored")).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).Return("http://base").Maybe()
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Maybe()
	m.roleRepo.EXPECT().GetUsersByRoles(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
}

func TestCreatePost_InvalidShareType(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreatePostReq()
	req.SharedContentID = "abc"
	req.SharedContentType = "bogus"

	// when
	_, err := svc.CreatePost(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidShareType)
}

func TestCreatePost_EmptyBodyNoShare(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreatePostReq()
	req.Body = "   "

	// when
	_, err := svc.CreatePost(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreatePost_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(5)
	m.postRepo.EXPECT().CountUserPostsToday(mock.Anything, userID).Return(0, errors.New("db down"))

	// when
	_, err := svc.CreatePost(context.Background(), userID, validCreatePostReq())

	// then
	require.Error(t, err)
}

func TestCreatePost_RateLimited(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(3)
	m.postRepo.EXPECT().CountUserPostsToday(mock.Anything, userID).Return(3, nil)

	// when
	_, err := svc.CreatePost(context.Background(), userID, validCreatePostReq())

	// then
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestCreatePost_InvalidPollTooFewOptions(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	req := validCreatePostReq()
	req.Poll = &dto.CreatePollInput{
		Options:         []dto.PollOptionInput{{Label: "only"}},
		DurationSeconds: 3600,
	}

	// when
	_, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrInvalidPoll)
}

func TestCreatePost_InvalidPollEmptyLabel(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	req := validCreatePostReq()
	req.Poll = &dto.CreatePollInput{
		Options:         []dto.PollOptionInput{{Label: "a"}, {Label: "   "}},
		DurationSeconds: 3600,
	}

	// when
	_, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrInvalidPoll)
}

func TestCreatePost_InvalidPollDuration(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	req := validCreatePostReq()
	req.Poll = &dto.CreatePollInput{
		Options:         []dto.PollOptionInput{{Label: "a"}, {Label: "b"}},
		DurationSeconds: 123,
	}

	// when
	_, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrInvalidDuration)
}

func TestCreatePost_RepoCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, "general", "hello", (*string)(nil), (*string)(nil)).Return(errors.New("boom"))

	// when
	_, err := svc.CreatePost(context.Background(), userID, validCreatePostReq())

	// then
	require.Error(t, err)
}

func TestCreatePost_EmptyCornerDefaults(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, "general", "hello", (*string)(nil), (*string)(nil)).Return(nil)
	expectBackgroundSocial(m)

	req := dto.CreatePostRequest{Body: "hello"}

	// when
	id, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreatePost_OK_Suggestions(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, "suggestions", "hello", (*string)(nil), (*string)(nil)).Return(nil)
	expectBackgroundSocial(m)

	req := validCreatePostReq()
	req.Corner = "suggestions"

	// when
	id, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreatePost_WithPollCreatesPoll(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, "general", "hello", (*string)(nil), (*string)(nil)).Return(nil)
	m.postRepo.EXPECT().CreatePollWithOptions(mock.Anything, mock.Anything, mock.Anything, 3600, mock.Anything, []string{"a", "b"}).Return(nil)
	expectBackgroundSocial(m)

	req := validCreatePostReq()
	req.Poll = &dto.CreatePollInput{
		Options:         []dto.PollOptionInput{{Label: "a"}, {Label: "b"}},
		DurationSeconds: 3600,
	}

	// when
	id, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreatePost_WithPollRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, "general", "hello", (*string)(nil), (*string)(nil)).Return(nil)
	m.postRepo.EXPECT().CreatePollWithOptions(mock.Anything, mock.Anything, mock.Anything, 3600, mock.Anything, []string{"a", "b"}).Return(errors.New("boom"))
	expectBackgroundSocial(m)

	req := validCreatePostReq()
	req.Poll = &dto.CreatePollInput{
		Options:         []dto.PollOptionInput{{Label: "a"}, {Label: "b"}},
		DurationSeconds: 3600,
	}

	// when
	_, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestCreatePost_ShareOK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxPostsPerDay).Return(0)
	m.postRepo.EXPECT().Create(mock.Anything, mock.Anything, userID, "general", "", mock.Anything, mock.Anything).Return(nil)
	expectBackgroundSocial(m)

	req := dto.CreatePostRequest{
		Body:              "",
		SharedContentID:   uuid.New().String(),
		SharedContentType: "theory",
	}

	// when
	id, err := svc.CreatePost(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestGetPost_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.postRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetPost(context.Background(), id, viewer, "")

	// then
	require.Error(t, err)
}

func TestGetPost_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.postRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(nil, nil)

	// when
	_, err := svc.GetPost(context.Background(), id, viewer, "")

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetPost_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	authorID := uuid.New()
	viewer := uuid.New()
	row := &model.PostRow{ID: id, UserID: authorID}
	m.postRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(row, nil)
	m.postRepo.EXPECT().RecordView(mock.Anything, id, "hash").Return(true, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.postRepo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbeds(mock.Anything, id.String(), "post").Return(nil, nil)
	m.postRepo.EXPECT().GetComments(mock.Anything, id, viewer, 500, 0, mock.Anything).Return(nil, 0, nil)
	m.postRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "comment").Return(nil, nil)
	m.postRepo.EXPECT().GetLikedBy(mock.Anything, id, mock.Anything).Return(nil, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewer, authorID).Return(false, nil)
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, id, viewer).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCount(mock.Anything, id.String(), "post").Return(0, nil)

	// when
	got, err := svc.GetPost(context.Background(), id, viewer, "hash")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.ViewCount)
}

func TestGetPost_AnonymousNoViewHash(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	authorID := uuid.New()
	row := &model.PostRow{ID: id, UserID: authorID}
	m.postRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(row, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	m.postRepo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbeds(mock.Anything, id.String(), "post").Return(nil, nil)
	m.postRepo.EXPECT().GetComments(mock.Anything, id, uuid.Nil, 500, 0, mock.Anything).Return(nil, 0, nil)
	m.postRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "comment").Return(nil, nil)
	m.postRepo.EXPECT().GetLikedBy(mock.Anything, id, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, id, uuid.Nil).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCount(mock.Anything, id.String(), "post").Return(5, nil)

	// when
	got, err := svc.GetPost(context.Background(), id, uuid.Nil, "")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.ViewerBlocked)
	assert.Equal(t, 5, got.ShareCount)
}

func TestUpdatePost_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdatePost(context.Background(), uuid.New(), uuid.New(), dto.UpdatePostRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdatePost_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(true)
	m.postRepo.EXPECT().UpdatePostAsAdmin(mock.Anything, id, "body").Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.UpdatePost(context.Background(), id, userID, dto.UpdatePostRequest{Body: " body "})

	// then
	require.NoError(t, err)
}

func TestUpdatePost_AsAdminRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(true)
	m.postRepo.EXPECT().UpdatePostAsAdmin(mock.Anything, id, "body").Return(errors.New("boom"))

	// when
	err := svc.UpdatePost(context.Background(), id, userID, dto.UpdatePostRequest{Body: "body"})

	// then
	require.Error(t, err)
}

func TestUpdatePost_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.postRepo.EXPECT().UpdatePost(mock.Anything, id, userID, "body").Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.UpdatePost(context.Background(), id, userID, dto.UpdatePostRequest{Body: "body"})

	// then
	require.NoError(t, err)
}

func TestUpdatePost_AsOwnerRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.postRepo.EXPECT().UpdatePost(mock.Anything, id, userID, "body").Return(errors.New("boom"))

	// when
	err := svc.UpdatePost(context.Background(), id, userID, dto.UpdatePostRequest{Body: "body"})

	// then
	require.Error(t, err)
}

func TestDeletePost_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetSharedContentFields(mock.Anything, id).Return(nil, nil, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.postRepo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeletePost(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeletePost_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetSharedContentFields(mock.Anything, id).Return(nil, nil, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.postRepo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeletePost(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeletePost_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetSharedContentFields(mock.Anything, id).Return(nil, nil, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.postRepo.EXPECT().Delete(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeletePost(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestDeletePost_SharedContentDecrements(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	sharedID := "shared-abc"
	sharedType := "theory"
	m.postRepo.EXPECT().GetSharedContentFields(mock.Anything, id).Return(&sharedID, &sharedType, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.postRepo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.postRepo.EXPECT().DecrementShareCount(mock.Anything, sharedID, sharedType).
		Run(func(_ context.Context, _ string, _ string) { wg.Done() }).
		Return(nil)

	// when
	err := svc.DeletePost(context.Background(), id, userID)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestListFeed_FollowingTab(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	rows := []model.PostRow{{ID: uuid.New()}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.postRepo.EXPECT().ListByFollowing(mock.Anything, viewer, "general", "new", 0, 10, 0, mock.Anything).Return(rows, 1, nil)
	m.postRepo.EXPECT().GetMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "post").Return(nil, nil)
	m.postRepo.EXPECT().GetPollsByPostIDs(mock.Anything, mock.Anything, viewer).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCountsBatch(mock.Anything, mock.Anything, "post").Return(nil, nil)

	// when
	got, err := svc.ListFeed(context.Background(), "following", viewer, "", "", "new", 0, 10, 0, "")

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.Total)
}

func TestListFeed_AllTab(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.Nil
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.postRepo.EXPECT().ListAll(mock.Anything, viewer, "general", "q", "new", 0, 5, 0, mock.Anything, "resolved").Return(nil, 0, nil)
	m.postRepo.EXPECT().GetMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "post").Return(nil, nil)
	m.postRepo.EXPECT().GetPollsByPostIDs(mock.Anything, mock.Anything, viewer).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCountsBatch(mock.Anything, mock.Anything, "post").Return(nil, nil)

	// when
	got, err := svc.ListFeed(context.Background(), "all", viewer, "", "q", "new", 0, 5, 0, "resolved")

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, got.Total)
}

func TestListFeed_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.postRepo.EXPECT().ListAll(mock.Anything, viewer, "general", "", "new", 0, 10, 0, mock.Anything, "").Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListFeed(context.Background(), "all", viewer, "", "", "new", 0, 10, 0, "")

	// then
	require.Error(t, err)
}

func TestListUserPosts_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	viewer := uuid.New()
	m.postRepo.EXPECT().ListByUser(mock.Anything, target, viewer, 10, 0).Return(nil, 2, nil)
	m.postRepo.EXPECT().GetMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "post").Return(nil, nil)
	m.postRepo.EXPECT().GetPollsByPostIDs(mock.Anything, mock.Anything, viewer).Return(nil, nil, nil, nil)
	m.postRepo.EXPECT().GetShareCountsBatch(mock.Anything, mock.Anything, "post").Return(nil, nil)

	// when
	got, err := svc.ListUserPosts(context.Background(), target, viewer, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, got.Total)
}

func TestListUserPosts_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	target := uuid.New()
	viewer := uuid.New()
	m.postRepo.EXPECT().ListByUser(mock.Anything, target, viewer, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListUserPosts(context.Background(), target, viewer, 10, 0)

	// then
	require.Error(t, err)
}

func TestUploadPostMedia_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(uuid.Nil, errors.New("nope"))

	// when
	_, err := svc.UploadPostMedia(context.Background(), postID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadPostMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	other := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(other, nil)

	// when
	_, err := svc.UploadPostMedia(context.Background(), postID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the post author")
}

func TestUploadPostMedia_UploaderError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, "posts", mock.Anything, int64(10), int64(1000), mock.Anything).Return("", errors.New("upload fail"))

	// when
	_, err := svc.UploadPostMedia(context.Background(), postID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.Error(t, err)
}

func TestDeletePostMedia_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(uuid.Nil, errors.New("nope"))

	// when
	err := svc.DeletePostMedia(context.Background(), postID, 1, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeletePostMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	other := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(other, nil)

	// when
	err := svc.DeletePostMedia(context.Background(), postID, 1, userID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the post author")
}

func TestDeletePostMedia_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(userID, nil)
	m.postRepo.EXPECT().DeleteMedia(mock.Anything, int64(1), postID).Return("", errors.New("boom"))

	// when
	err := svc.DeletePostMedia(context.Background(), postID, 1, userID)

	// then
	require.Error(t, err)
}

func TestDeletePostMedia_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(userID, nil)
	m.postRepo.EXPECT().DeleteMedia(mock.Anything, int64(1), postID).Return("/uploads/posts/a.webp", nil)
	m.uploadSvc.EXPECT().Delete("/uploads/posts/a.webp").Return(nil)

	// when
	err := svc.DeletePostMedia(context.Background(), postID, 1, userID)

	// then
	require.NoError(t, err)
}

func TestUploadCommentMedia_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("nope"))

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
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.New(), nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the comment author")
}

func TestUploadCommentMedia_UploaderError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().SaveImage(mock.Anything, "posts", mock.Anything, int64(10), int64(1000), mock.Anything).Return("", errors.New("upload fail"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, strings.NewReader("x"))

	// then
	require.Error(t, err)
}

func TestLikePost_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(uuid.Nil, errors.New("nope"))

	// when
	err := svc.LikePost(context.Background(), userID, postID)

	// then
	require.Error(t, err)
}

func TestLikePost_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.LikePost(context.Background(), userID, postID)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikePost_LikeRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().Like(mock.Anything, userID, postID).Return(errors.New("boom"))

	// when
	err := svc.LikePost(context.Background(), userID, postID)

	// then
	require.Error(t, err)
}

func TestLikePost_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().Like(mock.Anything, userID, postID).Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.LikePost(context.Background(), userID, postID)

	// then
	require.NoError(t, err)
}

func TestUnlikePost_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().Unlike(mock.Anything, userID, postID).Return(nil)

	// when
	err := svc.UnlikePost(context.Background(), userID, postID)

	// then
	require.NoError(t, err)
}

func TestUnlikePost_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().Unlike(mock.Anything, userID, postID).Return(errors.New("boom"))

	// when
	err := svc.UnlikePost(context.Background(), userID, postID)

	// then
	require.Error(t, err)
}

func TestCreateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_PostAuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(uuid.Nil, errors.New("nope"))

	// when
	_, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestCreateComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().CreateComment(mock.Anything, mock.Anything, postID, (*uuid.UUID)(nil), userID, "hi").Return(errors.New("boom"))

	// when
	_, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestCreateComment_OKTopLevel(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().CreateComment(mock.Anything, mock.Anything, postID, (*uuid.UUID)(nil), userID, "hi").Return(nil)
	expectBackgroundSocial(m)

	// when
	id, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateComment_OKReply(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	parentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetPostAuthorID(mock.Anything, postID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().CreateComment(mock.Anything, mock.Anything, postID, &parentID, userID, "hi").Return(nil)
	expectBackgroundSocial(m)

	// when
	id, err := svc.CreateComment(context.Background(), postID, userID, dto.CreateCommentRequest{Body: "hi", ParentID: &parentID})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestUpdateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.postRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "body").Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "body"})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AsAdminRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.postRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "body").Return(errors.New("boom"))

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "body"})

	// then
	require.Error(t, err)
}

func TestUpdateComment_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.postRepo.EXPECT().UpdateComment(mock.Anything, id, userID, "body").Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "body"})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AsOwnerRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.postRepo.EXPECT().UpdateComment(mock.Anything, id, userID, "body").Return(errors.New("boom"))

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "body"})

	// then
	require.Error(t, err)
}

func TestDeleteComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.postRepo.EXPECT().DeleteCommentAsAdmin(mock.Anything, id).Return(nil)

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
	m.postRepo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(nil)

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
	m.postRepo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestLikeComment_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("nope"))

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.Error(t, err)
}

func TestLikeComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikeComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.Error(t, err)
}

func TestLikeComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.postRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.postRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	err := svc.UnlikeComment(context.Background(), userID, commentID)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	err := svc.UnlikeComment(context.Background(), userID, commentID)

	// then
	require.Error(t, err)
}

func TestGetCornerCounts_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	want := map[string]int{"general": 3}
	m.postRepo.EXPECT().GetCornerCounts(mock.Anything).Return(want, nil)

	// when
	got, err := svc.GetCornerCounts(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetCornerCounts_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.postRepo.EXPECT().GetCornerCounts(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetCornerCounts(context.Background())

	// then
	require.Error(t, err)
}

func TestRefreshStaleEmbeds_RepoErrorReturnsZero(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.postRepo.EXPECT().GetStaleEmbeds(mock.Anything, "-1 day", 50).Return(nil, errors.New("boom"))

	// when
	got := svc.RefreshStaleEmbeds(context.Background())

	// then
	assert.Zero(t, got)
}

func TestRefreshStaleEmbeds_UnparsedSkipped(t *testing.T) {
	// given
	svc, m := newTestService(t)
	stale := []model.EmbedRow{{ID: 1, URL: "not-a-url"}}
	m.postRepo.EXPECT().GetStaleEmbeds(mock.Anything, "-1 day", 50).Return(stale, nil)

	// when
	got := svc.RefreshStaleEmbeds(context.Background())

	// then
	assert.Zero(t, got)
}

func TestVotePoll_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(nil, nil, nil, errors.New("boom"))

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.Error(t, err)
}

func TestVotePoll_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(nil, nil, nil, nil)

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestVotePoll_AlreadyVoted(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	pollID := uuid.New().String()
	poll := &model.PollRow{ID: pollID, ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	voted := 1
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, nil, &voted, nil)

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.ErrorIs(t, err, ErrAlreadyVoted)
}

func TestVotePoll_Expired(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	poll := &model.PollRow{ID: uuid.New().String(), ExpiresAt: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, nil, nil, nil)

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.ErrorIs(t, err, ErrPollExpired)
}

func TestVotePoll_InvalidOption(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	poll := &model.PollRow{ID: uuid.New().String(), ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	options := []model.PollOptionRow{{ID: 1}, {ID: 2}}
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, options, nil, nil)

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 999)

	// then
	require.ErrorIs(t, err, ErrInvalidOption)
}

func TestVotePoll_VoteRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	pollID := uuid.New()
	poll := &model.PollRow{ID: pollID.String(), ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	options := []model.PollOptionRow{{ID: 1}}
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, options, nil, nil)
	m.postRepo.EXPECT().VotePoll(mock.Anything, pollID, userID, 1).Return(errors.New("boom"))

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.Error(t, err)
}

func TestVotePoll_VoteAlreadyVotedDBError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	pollID := uuid.New()
	poll := &model.PollRow{ID: pollID.String(), ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	options := []model.PollOptionRow{{ID: 1}}
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, options, nil, nil)
	m.postRepo.EXPECT().VotePoll(mock.Anything, pollID, userID, 1).Return(errors.New("user already voted on this poll"))

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.ErrorIs(t, err, ErrAlreadyVoted)
}

func TestVotePoll_RefreshError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	pollID := uuid.New()
	poll := &model.PollRow{ID: pollID.String(), ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	options := []model.PollOptionRow{{ID: 1}}
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, options, nil, nil).Once()
	m.postRepo.EXPECT().VotePoll(mock.Anything, pollID, userID, 1).Return(nil)
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(nil, nil, nil, errors.New("boom")).Once()

	// when
	_, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.Error(t, err)
}

func TestVotePoll_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	pollID := uuid.New()
	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	poll := &model.PollRow{ID: pollID.String(), ExpiresAt: expiresAt}
	options := []model.PollOptionRow{{ID: 1}}
	voted := 1
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, options, nil, nil).Once()
	m.postRepo.EXPECT().VotePoll(mock.Anything, pollID, userID, 1).Return(nil)
	m.postRepo.EXPECT().GetPollByPostID(mock.Anything, postID, userID).Return(poll, options, &voted, nil).Once()

	// when
	got, err := svc.VotePoll(context.Background(), postID, userID, 1)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestResolveSuggestion_Unauthorised(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(false)

	// when
	err := svc.ResolveSuggestion(context.Background(), postID, userID, "done")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authorised")
}

func TestResolveSuggestion_InvalidStatusNormalised(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(true)
	m.postRepo.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, "done").Return(nil)
	expectBackgroundSocial(m)

	// when
	err := svc.ResolveSuggestion(context.Background(), postID, userID, "not-a-status")

	// then
	require.NoError(t, err)
}

func TestResolveSuggestion_Archived(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(true)
	m.postRepo.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, "archived").Return(nil)

	// when
	err := svc.ResolveSuggestion(context.Background(), postID, userID, "archived")

	// then
	require.NoError(t, err)
}

func TestResolveSuggestion_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(true)
	m.postRepo.EXPECT().ResolveSuggestion(mock.Anything, postID, userID, "done").Return(errors.New("boom"))

	// when
	err := svc.ResolveSuggestion(context.Background(), postID, userID, "done")

	// then
	require.Error(t, err)
}

func TestUnresolveSuggestion_Unauthorised(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(false)

	// when
	err := svc.UnresolveSuggestion(context.Background(), postID, userID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authorised")
}

func TestUnresolveSuggestion_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(true)
	m.postRepo.EXPECT().UnresolveSuggestion(mock.Anything, postID).Return(nil)

	// when
	err := svc.UnresolveSuggestion(context.Background(), postID, userID)

	// then
	require.NoError(t, err)
}

func TestUnresolveSuggestion_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	postID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermResolveSuggestion).Return(true)
	m.postRepo.EXPECT().UnresolveSuggestion(mock.Anything, postID).Return(errors.New("boom"))

	// when
	err := svc.UnresolveSuggestion(context.Background(), postID, userID)

	// then
	require.Error(t, err)
}

func TestGetShareCount_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.postRepo.EXPECT().GetShareCount(mock.Anything, "abc", "post").Return(7, nil)

	// when
	got, err := svc.GetShareCount(context.Background(), "abc", "post")

	// then
	require.NoError(t, err)
	assert.Equal(t, 7, got)
}

func TestGetShareCount_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.postRepo.EXPECT().GetShareCount(mock.Anything, "abc", "post").Return(0, errors.New("boom"))

	// when
	_, err := svc.GetShareCount(context.Background(), "abc", "post")

	// then
	require.Error(t, err)
}

type alwaysRejectRule struct{}

func (alwaysRejectRule) Name() contentfilter.RuleName { return "test_reject" }
func (alwaysRejectRule) Check(_ context.Context, _ []string) (*contentfilter.Rejection, error) {
	return &contentfilter.Rejection{Rule: "test_reject", Reason: "nope", Detail: "xyz"}, nil
}

func TestCreatePost_RejectedByContentFilter(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	svc.contentFilter = contentfilter.New(alwaysRejectRule{})
	req := validCreatePostReq()
	req.Body = "https://giphy.com/gifs/abc123"

	// when
	_, err := svc.CreatePost(context.Background(), uuid.New(), req)

	// then
	var rej *contentfilter.RejectedError
	require.ErrorAs(t, err, &rej)
	assert.Equal(t, "xyz", rej.Rejection.Detail)
}
