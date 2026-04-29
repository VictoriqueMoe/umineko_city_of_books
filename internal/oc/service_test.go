package oc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/contentfilter"
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
	ocRepo      *repository.MockOCRepository
	userRepo    *repository.MockUserRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	ocRepo := repository.NewMockOCRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := media.NewProcessor(1)

	svc := NewService(ocRepo, userRepo, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, nil, contentfilter.New()).(*service)
	return svc, &testMocks{
		ocRepo:      ocRepo,
		userRepo:    userRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
	}
}

func TestCreateOC_EmptyNameRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateOCRequest{Name: "   ", Series: "umineko"}

	// when
	_, err := svc.CreateOC(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyName)
}

func TestCreateOC_InvalidSeriesRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateOCRequest{Name: "Linda", Series: "rosegunsdays"}

	// when
	_, err := svc.CreateOC(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidSeries)
}

func TestCreateOC_CustomSeriesRequiresName(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateOCRequest{Name: "Linda", Series: "custom", CustomSeriesName: "  "}

	// when
	_, err := svc.CreateOC(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyCustomSeries)
}

func TestCreateOC_DuplicateNameRejected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Series: "umineko"}
	m.ocRepo.EXPECT().HasOC(mock.Anything, userID, "Linda").Return(true, nil)

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, ErrDuplicateName)
}

func TestCreateOC_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateOCRequest{Name: "  Linda  ", Description: "  bio ", Series: "umineko"}
	m.ocRepo.EXPECT().HasOC(mock.Anything, userID, "Linda").Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, mock.Anything, userID, "Linda", "bio", "umineko", "").
		Return(nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, mock.Anything, userID).Return(nil, nil).Maybe()

	// when
	id, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateOC_CustomSeriesOK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Description: "bio", Series: "custom", CustomSeriesName: " Higanbana "}
	m.ocRepo.EXPECT().HasOC(mock.Anything, userID, "Linda").Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, mock.Anything, userID, "Linda", "bio", "custom", "Higanbana").
		Return(nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, mock.Anything, userID).Return(nil, nil).Maybe()

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.NoError(t, err)
}

func TestCreateOC_RepoErrorBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateOCRequest{Name: "Linda", Series: "umineko"}
	m.ocRepo.EXPECT().HasOC(mock.Anything, userID, "Linda").Return(false, nil)
	m.ocRepo.EXPECT().
		Create(mock.Anything, mock.Anything, userID, "Linda", "", "umineko", "").
		Return(errors.New("db down"))

	// when
	_, err := svc.CreateOC(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestGetOC_NotFoundError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	m.ocRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(nil, nil)

	// when
	_, err := svc.GetOC(context.Background(), id, uuid.Nil)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetOC_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	authorID := uuid.New()
	row := &model.OCRow{ID: id, UserID: authorID, Name: "Linda", Series: "umineko"}
	m.ocRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(row, nil)
	m.ocRepo.EXPECT().GetGallery(mock.Anything, id).Return(nil, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	m.ocRepo.EXPECT().GetComments(mock.Anything, id, uuid.Nil, 500, 0, mock.Anything).Return(nil, 0, nil)
	m.ocRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)

	// when
	got, err := svc.GetOC(context.Background(), id, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Linda", got.Name)
}

func TestUpdateOC_EmptyNameRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.UpdateOCRequest{Name: "  ", Series: "umineko"}

	// when
	err := svc.UpdateOC(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyName)
}

func TestUpdateOC_InvalidSeriesRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.UpdateOCRequest{Name: "Linda", Series: "bad"}

	// when
	err := svc.UpdateOC(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidSeries)
}

func TestUpdateOC_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	req := dto.UpdateOCRequest{Name: "Linda", Description: "bio", Series: "umineko"}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
	m.ocRepo.EXPECT().Update(mock.Anything, id, userID, "Linda", "bio", "umineko", "", false).Return(nil)
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, id, userID).Return(nil, nil).Maybe()

	// when
	err := svc.UpdateOC(context.Background(), id, userID, req)

	// then
	require.NoError(t, err)
}

func TestUpdateOC_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	adminID := uuid.New()
	req := dto.UpdateOCRequest{Name: "Linda", Series: "umineko"}
	m.authz.EXPECT().Can(mock.Anything, adminID, authz.PermEditAnyPost).Return(true)
	m.ocRepo.EXPECT().Update(mock.Anything, id, adminID, "Linda", "", "umineko", "", true).Return(nil)
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, id, mock.Anything).Return(nil, nil).Maybe()

	// when
	err := svc.UpdateOC(context.Background(), id, adminID, req)

	// then
	require.NoError(t, err)
}

func TestDeleteOC_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.ocRepo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeleteOC(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteOC_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	adminID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, adminID, authz.PermDeleteAnyPost).Return(true)
	m.ocRepo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteOC(context.Background(), id, adminID)

	// then
	require.NoError(t, err)
}

func TestVote_BlockedReturnsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	err := svc.Vote(context.Background(), userID, id, 1)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestVote_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.ocRepo.EXPECT().Vote(mock.Anything, userID, id, 1).Return(nil)

	// when
	err := svc.Vote(context.Background(), userID, id, 1)

	// then
	require.NoError(t, err)
}

func TestToggleFavourite_AddsFavourite(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, id, userID).Return(&model.OCRow{ID: id, Name: "Linda", UserFavourited: false}, nil)
	m.ocRepo.EXPECT().Favourite(mock.Anything, userID, id).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("ignored")).Maybe()

	// when
	favourited, err := svc.ToggleFavourite(context.Background(), userID, id)

	// then
	require.NoError(t, err)
	assert.True(t, favourited)
}

func TestToggleFavourite_RemovesFavourite(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.ocRepo.EXPECT().GetByID(mock.Anything, id, userID).Return(&model.OCRow{ID: id, Name: "Linda", UserFavourited: true}, nil)
	m.ocRepo.EXPECT().Unfavourite(mock.Anything, userID, id).Return(nil)

	// when
	favourited, err := svc.ToggleFavourite(context.Background(), userID, id)

	// then
	require.NoError(t, err)
	assert.False(t, favourited)
}

func TestCreateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateCommentRequest{Body: "  "}

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_BlockedReturnsError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	m.ocRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), id, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestUpdateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestValidateSeries_AllVariants(t *testing.T) {
	// given
	cases := []struct {
		input    string
		custom   string
		wantErr  error
		wantSer  string
		wantName string
	}{
		{input: "umineko", wantSer: "umineko"},
		{input: "HIGURASHI", wantSer: "higurashi"},
		{input: " ciconia  ", wantSer: "ciconia"},
		{input: "custom", custom: "Higanbana", wantSer: "custom", wantName: "Higanbana"},
		{input: "custom", custom: "  ", wantErr: ErrEmptyCustomSeries},
		{input: "rosegunsdays", wantErr: ErrInvalidSeries},
	}

	// when / then
	for _, c := range cases {
		gotSer, gotName, err := validateSeries(c.input, c.custom)
		if c.wantErr != nil {
			require.ErrorIs(t, err, c.wantErr, "case %q", c.input)
			continue
		}
		require.NoError(t, err, "case %q", c.input)
		assert.Equal(t, c.wantSer, gotSer)
		assert.Equal(t, c.wantName, gotName)
		assert.Equal(t, strings.ToLower(strings.TrimSpace(c.input)), gotSer)
	}
}
