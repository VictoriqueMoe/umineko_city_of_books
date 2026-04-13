package art

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	artRepo     *repository.MockArtRepository
	postRepo    *repository.MockPostRepository
	userRepo    *repository.MockUserRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
	mediaProc   *media.Processor
}

func newTestService(t *testing.T) (*service, *testMocks) {
	artRepo := repository.NewMockArtRepository(t)
	postRepo := repository.NewMockPostRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := media.NewProcessor(1)

	svc := NewService(artRepo, postRepo, userRepo, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc).(*service)
	return svc, &testMocks{
		artRepo:     artRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		mediaProc:   mediaProc,
	}
}

func cancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCreateArt_EmptyTitleRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateArt(context.Background(), uuid.New(), dto.CreateArtRequest{Title: "   "}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateArt_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(5)
	m.artRepo.EXPECT().CountUserArtToday(mock.Anything, userID).Return(0, errors.New("db down"))

	// when
	_, err := svc.CreateArt(context.Background(), userID, dto.CreateArtRequest{Title: "t"}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestCreateArt_RateLimited(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(5)
	m.artRepo.EXPECT().CountUserArtToday(mock.Anything, userID).Return(5, nil)

	// when
	_, err := svc.CreateArt(context.Background(), userID, dto.CreateArtRequest{Title: "t"}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestCreateArt_UploadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("", errors.New("disk full"))

	// when
	_, err := svc.CreateArt(context.Background(), userID, dto.CreateArtRequest{Title: "t"}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestCreateArt_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	ctx := cancelledCtx()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("/uploads/art/x.png", nil)
	m.uploadSvc.EXPECT().FullDiskPath("/uploads/art/x.png").Return("/tmp/does-not-exist-xyz.png")
	m.artRepo.EXPECT().
		CreateWithTags(mock.Anything, mock.Anything, userID, "general", "drawing", "t", "d", "/uploads/art/x.png", "", []string{"a"}, false).
		Return(errors.New("db"))

	// when
	_, err := svc.CreateArt(ctx, userID, dto.CreateArtRequest{Title: "  t  ", Description: "  d  ", Tags: []string{"a"}}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestCreateArt_OK_DefaultsAndTagCap(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	ctx := cancelledCtx()
	tags := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	capped := tags[:10]
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("/uploads/art/x.png", nil)
	m.uploadSvc.EXPECT().FullDiskPath("/uploads/art/x.png").Return("/tmp/does-not-exist-xyz.png")
	m.artRepo.EXPECT().
		CreateWithTags(mock.Anything, mock.Anything, userID, "general", "drawing", "t", "", "/uploads/art/x.png", "", capped, true).
		Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).Return("").Maybe()

	// when
	id, err := svc.CreateArt(ctx, userID, dto.CreateArtRequest{Title: "t", Tags: tags, IsSpoiler: true}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateArt_OK_CustomCornerAndType(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	ctx := cancelledCtx()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("/uploads/art/x.png", nil)
	m.uploadSvc.EXPECT().FullDiskPath("/uploads/art/x.png").Return("/tmp/does-not-exist-xyz.png")
	m.artRepo.EXPECT().
		CreateWithTags(mock.Anything, mock.Anything, userID, "umineko", "sketch", "t", "", "/uploads/art/x.png", "", []string(nil), false).
		Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, mock.Anything).Return("").Maybe()

	// when
	_, err := svc.CreateArt(ctx, userID, dto.CreateArtRequest{Title: "t", Corner: "umineko", ArtType: "sketch"}, "image/png", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
}

func TestGetArt_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.artRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetArt(context.Background(), id, uuid.Nil, "")

	// then
	require.Error(t, err)
}

func TestGetArt_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.artRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(nil, nil)

	// when
	_, err := svc.GetArt(context.Background(), id, uuid.Nil, "")

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetArt_OK_WithViewerHashAndBlocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewerID := uuid.New()
	authorID := uuid.New()
	row := &model.ArtRow{ID: id, UserID: authorID, Title: "T", ImageURL: "/u/x.png", ViewCount: 5}
	m.artRepo.EXPECT().GetByID(mock.Anything, id, viewerID).Return(row, nil)
	m.artRepo.EXPECT().RecordView(mock.Anything, id, "hashy").Return(true, nil)
	blocked := []uuid.UUID{uuid.New()}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(blocked, nil)
	m.artRepo.EXPECT().GetTags(mock.Anything, id).Return([]string{"tag"}, nil)
	m.artRepo.EXPECT().GetComments(mock.Anything, id, viewerID, 500, 0, blocked).Return(nil, 0, nil)
	m.artRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "art_comment").Return(nil, nil)
	m.artRepo.EXPECT().GetLikedBy(mock.Anything, id, blocked).Return(nil, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(true, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	got, err := svc.GetArt(context.Background(), id, viewerID, "hashy")

	// then
	require.NoError(t, err)
	assert.Equal(t, 6, got.ViewCount)
	assert.True(t, got.ViewerBlocked)
	assert.NotEmpty(t, got.ThumbnailURL)
}

func TestGetArt_OK_AnonymousNoHash(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	authorID := uuid.New()
	row := &model.ArtRow{ID: id, UserID: authorID, Title: "T", ImageURL: "/u/x.png"}
	m.artRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(row, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	m.artRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, nil)
	m.artRepo.EXPECT().GetComments(mock.Anything, id, uuid.Nil, 500, 0, []uuid.UUID(nil)).Return(nil, 0, nil)
	m.artRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.postRepo.EXPECT().GetEmbedsBatch(mock.Anything, mock.Anything, "art_comment").Return(nil, nil)
	m.artRepo.EXPECT().GetLikedBy(mock.Anything, id, []uuid.UUID(nil)).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("")

	// when
	got, err := svc.GetArt(context.Background(), id, uuid.Nil, "")

	// then
	require.NoError(t, err)
	assert.False(t, got.ViewerBlocked)
}

func TestUpdateArt_EmptyTitleRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateArt(context.Background(), uuid.New(), uuid.New(), dto.UpdateArtRequest{Title: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestUpdateArt_AsOwner_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.artRepo.EXPECT().
		UpdateWithTags(mock.Anything, id, userID, "t", "", []string(nil), false, false).
		Return(errors.New("not owner"))

	// when
	err := svc.UpdateArt(context.Background(), id, userID, dto.UpdateArtRequest{Title: "t"})

	// then
	require.Error(t, err)
}

func TestUpdateArt_AsOwner_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	tags := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.artRepo.EXPECT().
		UpdateWithTags(mock.Anything, id, userID, "t", "d", tags[:10], true, false).
		Return(nil)

	// when
	err := svc.UpdateArt(context.Background(), id, userID, dto.UpdateArtRequest{Title: "  t  ", Description: "  d  ", Tags: tags, IsSpoiler: true})

	// then
	require.NoError(t, err)
}

func TestUpdateArt_AsAdmin_SpawnsNotify(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(true)
	m.artRepo.EXPECT().
		UpdateWithTags(mock.Anything, id, userID, "t", "", []string(nil), false, true).
		Return(nil)
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("stop goroutine")).Maybe()

	// when
	err := svc.UpdateArt(context.Background(), id, userID, dto.UpdateArtRequest{Title: "t"})

	// then
	require.NoError(t, err)
}

func TestDeleteArt_GetImageError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetImageURL(mock.Anything, id).Return("", errors.New("no row"))

	// when
	err := svc.DeleteArt(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestDeleteArt_AsAdmin_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetImageURL(mock.Anything, id).Return("/u/x.png", nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.artRepo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)
	m.uploadSvc.EXPECT().Delete("/u/x.png").Return(nil)

	// when
	err := svc.DeleteArt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteArt_AsAdmin_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetImageURL(mock.Anything, id).Return("/u/x.png", nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.artRepo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(errors.New("boom"))

	// when
	err := svc.DeleteArt(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestDeleteArt_AsOwner_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetImageURL(mock.Anything, id).Return("/u/x.png", nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.artRepo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)
	m.uploadSvc.EXPECT().Delete("/u/x.png").Return(nil)

	// when
	err := svc.DeleteArt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteArt_AsOwner_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetImageURL(mock.Anything, id).Return("/u/x.png", nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.artRepo.EXPECT().Delete(mock.Anything, id, userID).Return(errors.New("not owner"))

	// when
	err := svc.DeleteArt(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestListArt_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	m.artRepo.EXPECT().
		ListAll(mock.Anything, viewerID, "general", "", "", "", "", 10, 0, []uuid.UUID(nil)).
		Return(nil, 0, errors.New("db"))

	// when
	_, err := svc.ListArt(context.Background(), viewerID, "", "", "", "", "", 10, 0)

	// then
	require.Error(t, err)
}

func TestListArt_OK_DefaultsCornerAndThumbnails(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	artID := uuid.New()
	rows := []model.ArtRow{{ID: artID, UserID: uuid.New(), Title: "A", ImageURL: "/u/x.png"}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	m.artRepo.EXPECT().
		ListAll(mock.Anything, viewerID, "general", "drawing", "q", "tag", "new", 10, 5, []uuid.UUID(nil)).
		Return(rows, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{artID}).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	got, err := svc.ListArt(context.Background(), viewerID, "", "drawing", "q", "tag", "new", 10, 5)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 5, got.Offset)
	assert.Len(t, got.Art, 1)
	assert.NotEmpty(t, got.Art[0].ThumbnailURL)
}

func TestListByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	m.artRepo.EXPECT().
		ListByUser(mock.Anything, userID, viewerID, 10, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListByUser(context.Background(), userID, viewerID, 10, 0)

	// then
	require.Error(t, err)
}

func TestListByUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	artID := uuid.New()
	rows := []model.ArtRow{{ID: artID, UserID: userID, Title: "A", ImageURL: "/u/y.png"}}
	m.artRepo.EXPECT().ListByUser(mock.Anything, userID, viewerID, 10, 0).Return(rows, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{artID}).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("")

	// when
	got, err := svc.ListByUser(context.Background(), userID, viewerID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Art, 1)
}

func TestLikeArt_GetAuthorError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.LikeArt(context.Background(), userID, artID)

	// then
	require.Error(t, err)
}

func TestLikeArt_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.LikeArt(context.Background(), userID, artID)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikeArt_LikeRepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().Like(mock.Anything, userID, artID).Return(errors.New("dup"))

	// when
	err := svc.LikeArt(context.Background(), userID, artID)

	// then
	require.Error(t, err)
}

func TestLikeArt_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().Like(mock.Anything, userID, artID).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("stop goroutine")).Maybe()

	// when
	err := svc.LikeArt(context.Background(), userID, artID)

	// then
	require.NoError(t, err)
}

func TestUnlikeArt_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().Unlike(mock.Anything, userID, artID).Return(nil)

	// when
	err := svc.UnlikeArt(context.Background(), userID, artID)

	// then
	require.NoError(t, err)
}

func TestUnlikeArt_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().Unlike(mock.Anything, userID, artID).Return(errors.New("boom"))

	// when
	err := svc.UnlikeArt(context.Background(), userID, artID)

	// then
	require.Error(t, err)
}

func TestGetCornerCounts_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.artRepo.EXPECT().GetCornerCounts(mock.Anything).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetCornerCounts(context.Background())

	// then
	require.Error(t, err)
}

func TestGetCornerCounts_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	want := map[string]int{"general": 2}
	m.artRepo.EXPECT().GetCornerCounts(mock.Anything).Return(want, nil)

	// when
	got, err := svc.GetCornerCounts(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetPopularTags_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.artRepo.EXPECT().GetPopularTags(mock.Anything, "general", 30).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetPopularTags(context.Background(), "general")

	// then
	require.Error(t, err)
}

func TestGetPopularTags_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.artRepo.EXPECT().GetPopularTags(mock.Anything, "umineko", 30).Return([]model.TagCount{{Tag: "a", Count: 4}}, nil)

	// when
	got, err := svc.GetPopularTags(context.Background(), "umineko")

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "a", got[0].Tag)
	assert.Equal(t, 4, got[0].Count)
}

func TestCreateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "  "})

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestCreateComment_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.CreateComment(context.Background(), artID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestCreateComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), artID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, artID, (*uuid.UUID)(nil), userID, "hi").
		Return(errors.New("db"))

	// when
	_, err := svc.CreateComment(context.Background(), artID, userID, dto.CreateCommentRequest{Body: "  hi  "})

	// then
	require.Error(t, err)
}

func TestCreateComment_OK_TopLevel(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, artID, (*uuid.UUID)(nil), userID, "hi").
		Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("stop goroutine")).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), artID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateComment_OK_Reply(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	parentID := uuid.New()
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, artID, &parentID, userID, "hi").
		Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("stop goroutine")).Maybe()

	// when
	_, err := svc.CreateComment(context.Background(), artID, userID, dto.CreateCommentRequest{Body: "hi", ParentID: &parentID})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: "  "})

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestUpdateComment_AsAdmin_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.artRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, commentID, "hi").Return(errors.New("db"))

	// when
	err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestUpdateComment_AsAdmin_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.artRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, commentID, "hi").Return(nil)
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("stop goroutine")).Maybe()
	m.postRepo.EXPECT().DeleteEmbeds(mock.Anything, mock.Anything, "art_comment").Return(nil).Maybe()

	// when
	err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "  hi  "})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AsOwner_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.artRepo.EXPECT().UpdateComment(mock.Anything, commentID, userID, "hi").Return(errors.New("not owner"))

	// when
	err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestUpdateComment_AsOwner_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.artRepo.EXPECT().UpdateComment(mock.Anything, commentID, userID, "hi").Return(nil)
	m.postRepo.EXPECT().DeleteEmbeds(mock.Anything, mock.Anything, "art_comment").Return(nil).Maybe()

	// when
	err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
}

func TestDeleteComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.artRepo.EXPECT().DeleteCommentAsAdmin(mock.Anything, commentID).Return(nil)

	// when
	err := svc.DeleteComment(context.Background(), commentID, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteComment_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	m.artRepo.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("not owner"))

	// when
	err := svc.DeleteComment(context.Background(), commentID, userID)

	// then
	require.Error(t, err)
}

func TestLikeComment_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("no row"))

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
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
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
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(errors.New("db"))

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
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.artRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)
	m.artRepo.EXPECT().GetCommentArtID(mock.Anything, commentID).Return(uuid.Nil, errors.New("stop goroutine")).Maybe()

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
	m.artRepo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

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
	m.artRepo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

	// when
	err := svc.UnlikeComment(context.Background(), userID, commentID)

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_CommentNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the comment author")
}

func TestUploadCommentMedia_UploadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("", errors.New("disk full"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_AddMediaError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("/uploads/art/x.png", nil)
	m.artRepo.EXPECT().
		AddCommentMedia(mock.Anything, commentID, "/uploads/art/x.png", "image", "", 0).
		Return(int64(0), errors.New("db"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_OK_CtxCancelled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	ctx := cancelledCtx()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, "image/png", int64(10), int64(1000), mock.Anything).
		Return("/uploads/art/x.png", nil)
	m.artRepo.EXPECT().
		AddCommentMedia(mock.Anything, commentID, "/uploads/art/x.png", "image", "", 0).
		Return(int64(42), nil)
	m.uploadSvc.EXPECT().FullDiskPath("/uploads/art/x.png").Return("/tmp/does-not-exist-xyz.png")

	// when
	resp, err := svc.UploadCommentMedia(ctx, commentID, userID, "image/png", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, resp.ID)
	assert.Equal(t, "image", resp.MediaType)
}

func TestCreateGallery_EmptyNameRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateGallery(context.Background(), uuid.New(), dto.CreateGalleryRequest{Name: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateGallery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.artRepo.EXPECT().
		CreateGallery(mock.Anything, mock.Anything, userID, "n", "d").
		Return(errors.New("db"))

	// when
	_, err := svc.CreateGallery(context.Background(), userID, dto.CreateGalleryRequest{Name: "  n  ", Description: "  d  "})

	// then
	require.Error(t, err)
}

func TestCreateGallery_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.artRepo.EXPECT().
		CreateGallery(mock.Anything, mock.Anything, userID, "n", "d").
		Return(nil)

	// when
	id, err := svc.CreateGallery(context.Background(), userID, dto.CreateGalleryRequest{Name: "n", Description: "d"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestUpdateGallery_EmptyNameRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateGallery(context.Background(), uuid.New(), uuid.New(), dto.UpdateGalleryRequest{Name: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestUpdateGallery_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().
		UpdateGallery(mock.Anything, id, userID, "n", "d").
		Return(nil)

	// when
	err := svc.UpdateGallery(context.Background(), id, userID, dto.UpdateGalleryRequest{Name: "  n  ", Description: "  d  "})

	// then
	require.NoError(t, err)
}

func TestUpdateGallery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().
		UpdateGallery(mock.Anything, id, userID, "n", "").
		Return(errors.New("not owner"))

	// when
	err := svc.UpdateGallery(context.Background(), id, userID, dto.UpdateGalleryRequest{Name: "n"})

	// then
	require.Error(t, err)
}

func TestSetGalleryCover_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	galleryID := uuid.New()
	userID := uuid.New()
	coverID := uuid.New()
	m.artRepo.EXPECT().SetGalleryCover(mock.Anything, galleryID, userID, &coverID).Return(nil)

	// when
	err := svc.SetGalleryCover(context.Background(), galleryID, userID, &coverID)

	// then
	require.NoError(t, err)
}

func TestSetGalleryCover_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	galleryID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().SetGalleryCover(mock.Anything, galleryID, userID, (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	err := svc.SetGalleryCover(context.Background(), galleryID, userID, nil)

	// then
	require.Error(t, err)
}

func TestDeleteGallery_ListError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().
		ListArtInGallery(mock.Anything, id, uuid.Nil, 10000, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	err := svc.DeleteGallery(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestDeleteGallery_DeleteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().
		ListArtInGallery(mock.Anything, id, uuid.Nil, 10000, 0).
		Return(nil, 0, nil)
	m.artRepo.EXPECT().DeleteGallery(mock.Anything, id, userID).Return(errors.New("not owner"))

	// when
	err := svc.DeleteGallery(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestDeleteGallery_OK_DeletesImages(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	rows := []model.ArtRow{
		{ID: uuid.New(), ImageURL: "/u/a.png"},
		{ID: uuid.New(), ImageURL: "/u/b.png"},
	}
	m.artRepo.EXPECT().
		ListArtInGallery(mock.Anything, id, uuid.Nil, 10000, 0).
		Return(rows, 2, nil)
	m.artRepo.EXPECT().DeleteGallery(mock.Anything, id, userID).Return(nil)
	m.uploadSvc.EXPECT().Delete("/u/a.png").Return(nil)
	m.uploadSvc.EXPECT().Delete("/u/b.png").Return(nil)

	// when
	err := svc.DeleteGallery(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestGetGallery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(nil, errors.New("boom"))

	// when
	_, _, _, err := svc.GetGallery(context.Background(), id, uuid.Nil, 10, 0)

	// then
	require.Error(t, err)
}

func TestGetGallery_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(nil, nil)

	// when
	_, _, _, err := svc.GetGallery(context.Background(), id, uuid.Nil, 10, 0)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetGallery_ListError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewerID := uuid.New()
	g := &model.GalleryRow{ID: id, Name: "G"}
	m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(g, nil)
	m.artRepo.EXPECT().
		ListArtInGallery(mock.Anything, id, viewerID, 10, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	_, _, _, err := svc.GetGallery(context.Background(), id, viewerID, 10, 0)

	// then
	require.Error(t, err)
}

func TestGetGallery_OK_WithCoverThumbnail(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewerID := uuid.New()
	artID := uuid.New()
	g := &model.GalleryRow{ID: id, Name: "G", CoverImageURL: "/u/cover.png"}
	rows := []model.ArtRow{{ID: artID, ImageURL: "/u/x.png"}}
	m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(g, nil)
	m.artRepo.EXPECT().ListArtInGallery(mock.Anything, id, viewerID, 10, 0).Return(rows, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{artID}).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	gallery, arts, total, err := svc.GetGallery(context.Background(), id, viewerID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, arts, 1)
	require.NotNil(t, gallery)
	assert.NotEmpty(t, gallery.CoverThumbnailURL)
}

func TestListUserGalleries_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.artRepo.EXPECT().ListGalleriesByUser(mock.Anything, userID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListUserGalleries(context.Background(), userID)

	// then
	require.Error(t, err)
}

func TestListUserGalleries_OK_WithPreviews(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	galleryID := uuid.New()
	rows := []model.GalleryRow{
		{ID: galleryID, Name: "G", ArtCount: 3, CoverArtID: nil},
	}
	m.artRepo.EXPECT().ListGalleriesByUser(mock.Anything, userID).Return(rows, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com").Maybe()
	m.artRepo.EXPECT().
		GetGalleryPreviewImages(mock.Anything, galleryID, 3).
		Return([]repository.PreviewImage{{ImageURL: "/u/p.png"}}, nil)

	// when
	got, err := svc.ListUserGalleries(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Len(t, got[0].PreviewImages, 1)
}

func TestListUserGalleries_OK_NoPreviewsWhenCoverPresent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	galleryID := uuid.New()
	coverID := uuid.New()
	rows := []model.GalleryRow{
		{ID: galleryID, Name: "G", ArtCount: 3, CoverArtID: &coverID, CoverImageURL: "/u/cover.png"},
	}
	m.artRepo.EXPECT().ListGalleriesByUser(mock.Anything, userID).Return(rows, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	got, err := svc.ListUserGalleries(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Empty(t, got[0].PreviewImages)
	assert.NotEmpty(t, got[0].CoverThumbnailURL)
}

func TestListAllGalleries_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.artRepo.EXPECT().ListAllGalleries(mock.Anything, "general").Return(nil, errors.New("boom"))

	// when
	_, err := svc.ListAllGalleries(context.Background(), "general")

	// then
	require.Error(t, err)
}

func TestListAllGalleries_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	rows := []model.GalleryRow{{ID: uuid.New(), Name: "G", ArtCount: 0}}
	m.artRepo.EXPECT().ListAllGalleries(mock.Anything, "umineko").Return(rows, nil)

	// when
	got, err := svc.ListAllGalleries(context.Background(), "umineko")

	// then
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestSetArtGallery_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	galleryID := uuid.New()
	m.artRepo.EXPECT().SetGallery(mock.Anything, artID, userID, &galleryID).Return(nil)

	// when
	err := svc.SetArtGallery(context.Background(), artID, userID, &galleryID)

	// then
	require.NoError(t, err)
}

func TestSetArtGallery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	m.artRepo.EXPECT().SetGallery(mock.Anything, artID, userID, (*uuid.UUID)(nil)).Return(errors.New("not owner"))

	// when
	err := svc.SetArtGallery(context.Background(), artID, userID, nil)

	// then
	require.Error(t, err)
}
