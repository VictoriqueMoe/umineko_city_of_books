package ship

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/quotefinder"
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
	shipRepo    *repository.MockShipRepository
	userRepo    *repository.MockUserRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
	mediaProc   *media.Processor
	quoteClient *quotefinder.Client
}

func newTestService(t *testing.T) (*service, *testMocks) {
	shipRepo := repository.NewMockShipRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := media.NewProcessor(1)
	quoteClient := quotefinder.NewClient()

	svc := NewService(shipRepo, userRepo, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, quoteClient).(*service)
	return svc, &testMocks{
		shipRepo:    shipRepo,
		userRepo:    userRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		mediaProc:   mediaProc,
		quoteClient: quoteClient,
	}
}

func validCharacters() []dto.ShipCharacter {
	return []dto.ShipCharacter{
		{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
		{Series: "umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
	}
}

func TestCreateShip_EmptyTitleRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	userID := uuid.New()
	req := dto.CreateShipRequest{Title: "   ", Characters: validCharacters()}

	// when
	_, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateShip_TooFewCharactersRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	userID := uuid.New()
	req := dto.CreateShipRequest{
		Title: "My Ship",
		Characters: []dto.ShipCharacter{
			{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
		},
	}

	// when
	_, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrTooFewCharacters)
}

func TestCreateShip_DuplicateCharactersRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	userID := uuid.New()
	req := dto.CreateShipRequest{
		Title: "My Ship",
		Characters: []dto.ShipCharacter{
			{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
			{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
		},
	}

	// when
	_, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrDuplicateCharacters)
}

func TestCreateShip_RepoErrorBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateShipRequest{Title: "Ship", Description: "desc", Characters: validCharacters()}
	m.shipRepo.EXPECT().
		CreateWithCharacters(mock.Anything, mock.Anything, userID, "Ship", "desc", req.Characters).
		Return(errors.New("db down"))

	// when
	_, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestCreateShip_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateShipRequest{Title: "  Ship  ", Description: "  desc  ", Characters: validCharacters()}
	m.shipRepo.EXPECT().
		CreateWithCharacters(mock.Anything, mock.Anything, userID, "Ship", "desc", req.Characters).
		Return(nil)

	// when
	id, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestGetShip_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	viewerID := uuid.New()
	m.shipRepo.EXPECT().GetByID(mock.Anything, shipID, viewerID).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetShip(context.Background(), shipID, viewerID)

	// then
	require.Error(t, err)
}

func TestGetShip_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	viewerID := uuid.New()
	m.shipRepo.EXPECT().GetByID(mock.Anything, shipID, viewerID).Return(nil, nil)

	// when
	_, err := svc.GetShip(context.Background(), shipID, viewerID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetShip_OK_WithViewer(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	viewerID := uuid.New()
	authorID := uuid.New()
	row := &model.ShipRow{ID: shipID, UserID: authorID, Title: "T"}
	m.shipRepo.EXPECT().GetByID(mock.Anything, shipID, viewerID).Return(row, nil)
	m.shipRepo.EXPECT().GetCharacters(mock.Anything, shipID).Return(nil, nil)
	blocked := []uuid.UUID{uuid.New()}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(blocked, nil)
	m.shipRepo.EXPECT().GetComments(mock.Anything, shipID, viewerID, 500, 0, blocked).Return(nil, 0, nil)
	m.shipRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(true, nil)

	// when
	got, err := svc.GetShip(context.Background(), shipID, viewerID)

	// then
	require.NoError(t, err)
	assert.True(t, got.ViewerBlocked)
	assert.Equal(t, shipID, got.ID)
}

func TestGetShip_OK_AnonymousViewerSkipsBlockCheck(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	authorID := uuid.New()
	row := &model.ShipRow{ID: shipID, UserID: authorID, Title: "T"}
	m.shipRepo.EXPECT().GetByID(mock.Anything, shipID, uuid.Nil).Return(row, nil)
	m.shipRepo.EXPECT().GetCharacters(mock.Anything, shipID).Return(nil, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	m.shipRepo.EXPECT().GetComments(mock.Anything, shipID, uuid.Nil, 500, 0, []uuid.UUID(nil)).Return(nil, 0, nil)
	m.shipRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetShip(context.Background(), shipID, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.False(t, got.ViewerBlocked)
}

func TestUpdateShip_EmptyTitleRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateShip(context.Background(), uuid.New(), uuid.New(), dto.UpdateShipRequest{Title: "  ", Characters: validCharacters()})

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestUpdateShip_TooFewCharactersRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.UpdateShipRequest{Title: "T", Characters: []dto.ShipCharacter{{Series: "umineko", CharacterID: "b", CharacterName: "B"}}}

	// when
	err := svc.UpdateShip(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrTooFewCharacters)
}

func TestUpdateShip_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	req := dto.UpdateShipRequest{Title: " T ", Description: " d ", Characters: validCharacters()}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(true)
	m.shipRepo.EXPECT().UpdateWithCharacters(mock.Anything, shipID, userID, "T", "d", req.Characters, true).Return(nil)

	// when
	err := svc.UpdateShip(context.Background(), shipID, userID, req)

	// then
	require.NoError(t, err)
}

func TestUpdateShip_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	req := dto.UpdateShipRequest{Title: "T", Characters: validCharacters()}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.shipRepo.EXPECT().UpdateWithCharacters(mock.Anything, shipID, userID, "T", "", req.Characters, false).Return(errors.New("not owner"))

	// when
	err := svc.UpdateShip(context.Background(), shipID, userID, req)

	// then
	require.Error(t, err)
}

func TestDeleteShip_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.shipRepo.EXPECT().DeleteAsAdmin(mock.Anything, shipID).Return(nil)

	// when
	err := svc.DeleteShip(context.Background(), shipID, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteShip_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.shipRepo.EXPECT().Delete(mock.Anything, shipID, userID).Return(nil)

	// when
	err := svc.DeleteShip(context.Background(), shipID, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteShip_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.shipRepo.EXPECT().Delete(mock.Anything, shipID, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteShip(context.Background(), shipID, userID)

	// then
	require.Error(t, err)
}

func TestListShips_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	m.shipRepo.EXPECT().
		List(mock.Anything, viewerID, "new", false, "umineko", "battler", 20, 0, []uuid.UUID(nil)).
		Return(nil, 0, errors.New("db down"))

	// when
	_, err := svc.ListShips(context.Background(), viewerID, "new", false, "umineko", "battler", 20, 0)

	// then
	require.Error(t, err)
}

func TestListShips_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	shipID := uuid.New()
	rows := []model.ShipRow{{ID: shipID, UserID: uuid.New(), Title: "A"}}
	blocked := []uuid.UUID{uuid.New()}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(blocked, nil)
	m.shipRepo.EXPECT().
		List(mock.Anything, viewerID, "top", true, "", "", 10, 5, blocked).
		Return(rows, 1, nil)
	m.shipRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{shipID}).Return(nil, nil)

	// when
	got, err := svc.ListShips(context.Background(), viewerID, "top", true, "", "", 10, 5)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 5, got.Offset)
	assert.Len(t, got.Ships, 1)
}

func TestListShipsByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	m.shipRepo.EXPECT().
		ListByUser(mock.Anything, userID, viewerID, 10, 0).
		Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListShipsByUser(context.Background(), userID, viewerID, 10, 0)

	// then
	require.Error(t, err)
}

func TestListShipsByUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	shipID := uuid.New()
	rows := []model.ShipRow{{ID: shipID, UserID: userID, Title: "A"}}
	m.shipRepo.EXPECT().ListByUser(mock.Anything, userID, viewerID, 10, 0).Return(rows, 1, nil)
	m.shipRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{shipID}).Return(nil, nil)

	// when
	got, err := svc.ListShipsByUser(context.Background(), userID, viewerID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Ships, 1)
}

func TestUploadShipImage_ShipNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.UploadShipImage(context.Background(), shipID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadShipImage_NotAuthorRejected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)

	// when
	_, err := svc.UploadShipImage(context.Background(), shipID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the ship author")
}

func TestUploadShipImage_UploadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, "image/png", int64(100), int64(1000), mock.Anything).
		Return("", errors.New("disk full"))

	// when
	_, err := svc.UploadShipImage(context.Background(), shipID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadShipImage_UpdateImageError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, "image/png", int64(100), int64(1000), mock.Anything).
		Return("/uploads/ships/x.png", nil)
	m.shipRepo.EXPECT().UpdateImage(mock.Anything, shipID, "/uploads/ships/x.png", "").Return(errors.New("db boom"))

	// when
	_, err := svc.UploadShipImage(context.Background(), shipID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadShipImage_OK_CtxCancelledReturnsOriginalURL(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, "image/png", int64(100), int64(1000), mock.Anything).
		Return("/uploads/ships/x.png", nil)
	m.shipRepo.EXPECT().UpdateImage(mock.Anything, shipID, "/uploads/ships/x.png", "").Return(nil)
	m.uploadSvc.EXPECT().FullDiskPath("/uploads/ships/x.png").Return("/tmp/does-not-exist-xyz.png")

	// when
	url, err := svc.UploadShipImage(ctx, shipID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, "/uploads/ships/x.png", url)
}

func TestVote_ShipNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.Vote(context.Background(), userID, shipID, 1)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestVote_BlockedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.Vote(context.Background(), userID, shipID, 1)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestVote_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipRepo.EXPECT().Vote(mock.Anything, userID, shipID, 1).Return(nil)

	// when
	err := svc.Vote(context.Background(), userID, shipID, 1)

	// then
	require.NoError(t, err)
}

func TestVote_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipRepo.EXPECT().Vote(mock.Anything, userID, shipID, -1).Return(errors.New("boom"))

	// when
	err := svc.Vote(context.Background(), userID, shipID, -1)

	// then
	require.Error(t, err)
}

func TestCreateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "   "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_ShipNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, shipID, (*uuid.UUID)(nil), userID, "hi").
		Return(errors.New("db down"))

	// when
	_, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "  hi  "})

	// then
	require.Error(t, err)
}

func TestCreateComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, shipID, (*uuid.UUID)(nil), userID, "hi").
		Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("stop goroutine")).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestUpdateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.shipRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, commentID, "hi").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.shipRepo.EXPECT().UpdateComment(mock.Anything, commentID, userID, "hi").Return(errors.New("not owner"))

	// when
	err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestDeleteComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.shipRepo.EXPECT().DeleteCommentAsAdmin(mock.Anything, commentID).Return(nil)

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
	m.shipRepo.EXPECT().DeleteComment(mock.Anything, commentID, userID).Return(errors.New("not owner"))

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
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("no row"))

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
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
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
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(errors.New("db"))

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.Error(t, err)
}

func TestLikeComment_OK_SelfLikeSkipsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	m.shipRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.NoError(t, err)
}

func TestLikeComment_OK_OtherAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipRepo.EXPECT().LikeComment(mock.Anything, userID, commentID).Return(nil)
	m.shipRepo.EXPECT().GetCommentShipID(mock.Anything, commentID).Return(uuid.Nil, errors.New("stop goroutine")).Maybe()

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
	m.shipRepo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(nil)

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
	m.shipRepo.EXPECT().UnlikeComment(mock.Anything, userID, commentID).Return(errors.New("boom"))

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
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the comment author")
}

func TestUploadCommentMedia_UploadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, "image/png", int64(100), int64(1000), mock.Anything).
		Return("", errors.New("disk full"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_AddMediaError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, "image/png", int64(100), int64(1000), mock.Anything).
		Return("/uploads/ships/x.png", nil)
	m.shipRepo.EXPECT().
		AddCommentMedia(mock.Anything, commentID, "/uploads/ships/x.png", "image", "", 0).
		Return(int64(0), errors.New("db"))

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_OK_CtxCancelled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, "image/png", int64(100), int64(1000), mock.Anything).
		Return("/uploads/ships/x.png", nil)
	m.shipRepo.EXPECT().
		AddCommentMedia(mock.Anything, commentID, "/uploads/ships/x.png", "image", "", 0).
		Return(int64(42), nil)
	m.uploadSvc.EXPECT().FullDiskPath("/uploads/ships/x.png").Return("/tmp/does-not-exist-xyz.png")

	// when
	resp, err := svc.UploadCommentMedia(ctx, commentID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, resp.ID)
	assert.Equal(t, "image", resp.MediaType)
}

func TestListCharacters_InvalidSeries(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.ListCharacters(quotefinder.Series("nonsense"))

	// then
	require.Error(t, err)
}

func TestValidateCharacters_CaseInsensitiveDuplicate(t *testing.T) {
	// given
	chars := []dto.ShipCharacter{
		{Series: "Umineko", CharacterID: "Battler", CharacterName: "Battler"},
		{Series: "umineko", CharacterID: "battler", CharacterName: "battler"},
	}

	// when
	err := validateCharacters(chars)

	// then
	require.ErrorIs(t, err, ErrDuplicateCharacters)
}

func TestValidateCharacters_OK(t *testing.T) {
	// given
	chars := validCharacters()

	// when
	err := validateCharacters(chars)

	// then
	require.NoError(t, err)
}

var _ io.Reader = (*bytes.Reader)(nil)
