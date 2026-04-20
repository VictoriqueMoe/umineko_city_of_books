package fanfic

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/config"
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
	fanficRepo  *repository.MockFanficRepository
	userRepo    *repository.MockUserRepository
	authz       *authz.MockService
	blockSvc    *block.MockService
	notifSvc    *notification.MockService
	uploadSvc   *upload.MockService
	settingsSvc *settings.MockService
	mediaProc   *media.Processor
}

func newTestService(t *testing.T) (*service, *testMocks) {
	fanficRepo := repository.NewMockFanficRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	uploadSvc.EXPECT().FullDiskPath(mock.Anything).Return("/tmp/does-not-exist-xyz.png").Maybe()
	settingsSvc := settings.NewMockService(t)
	mediaProc := media.NewProcessor(1)

	svc := NewService(fanficRepo, userRepo, authzSvc, blockSvc, notifSvc, uploadSvc, mediaProc, settingsSvc, contentfilter.New()).(*service)
	return svc, &testMocks{
		fanficRepo:  fanficRepo,
		userRepo:    userRepo,
		authz:       authzSvc,
		blockSvc:    blockSvc,
		notifSvc:    notifSvc,
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		mediaProc:   mediaProc,
	}
}

func TestCreateFanfic_EmptyTitleRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateFanfic(context.Background(), uuid.New(), dto.CreateFanficRequest{Title: "   "})

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateFanfic_TooManyGenres(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateFanficRequest{Title: "t", Genres: []string{"a", "b", "c"}}

	// when
	_, err := svc.CreateFanfic(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrTooManyGenres)
}

func TestCreateFanfic_TooManyTags(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	tags := make([]string, 11)
	for i := range tags {
		tags[i] = string(rune('a' + i))
	}
	req := dto.CreateFanficRequest{Title: "t", Tags: tags}

	// when
	_, err := svc.CreateFanfic(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrTooManyTags)
}

func TestCreateFanfic_TagTooLong(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	long := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	req := dto.CreateFanficRequest{Title: "t", Tags: []string{long}}

	// when
	_, err := svc.CreateFanfic(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrTagTooLong)
}

func TestCreateFanfic_InvalidRating(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := dto.CreateFanficRequest{Title: "t", Rating: "X"}

	// when
	_, err := svc.CreateFanfic(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrInvalidRating)
}

func TestCreateFanfic_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateFanficRequest{Title: "Title"}
	m.fanficRepo.EXPECT().
		CreateWithDetails(mock.Anything, mock.Anything, userID, "Title", "", "", "K", "", "in_progress", false, false, []string(nil), []string(nil), []dto.FanficCharacter(nil), false).
		Return(errors.New("db"))

	// when
	_, err := svc.CreateFanfic(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestCreateFanfic_OK_DefaultsApplied(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateFanficRequest{
		Title:    " Title ",
		Summary:  " sum ",
		Series:   " My Series ",
		Language: " English ",
		Status:   "garbage",
		Tags:     []string{"a", "A", " a ", "b"},
	}
	m.fanficRepo.EXPECT().RegisterSeries(mock.Anything, "My Series").Return(nil)
	m.fanficRepo.EXPECT().RegisterLanguage(mock.Anything, "English").Return(nil)
	m.fanficRepo.EXPECT().
		CreateWithDetails(mock.Anything, mock.Anything, userID, "Title", "sum", "My Series", "K", "English", "in_progress", false, false, []string(nil), []string{"a", "b"}, []dto.FanficCharacter(nil), false).
		Return(nil)

	// when
	id, err := svc.CreateFanfic(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateFanfic_OK_WithBodyAndOCCharacter(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateFanficRequest{
		Title:  "Title",
		Status: "draft",
		Body:   "<p>hello world</p>",
		Characters: []dto.FanficCharacter{
			{CharacterID: "", CharacterName: "  Piece  "},
			{CharacterID: "existing", CharacterName: "Existing"},
		},
		Rating: "M",
	}
	m.fanficRepo.EXPECT().
		CreateWithDetails(mock.Anything, mock.Anything, userID, "Title", "", "", "M", "", "draft", false, false, []string(nil), []string(nil), req.Characters, false).
		Return(nil)
	m.fanficRepo.EXPECT().CreateChapter(mock.Anything, mock.Anything, mock.Anything, 1, "", "<p>hello world</p>", 2).Return(nil)
	m.fanficRepo.EXPECT().UpdateWordCount(mock.Anything, mock.Anything).Return(nil)
	m.fanficRepo.EXPECT().RegisterOCCharacter(mock.Anything, "Piece", userID).Return(nil)

	// when
	id, err := svc.CreateFanfic(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateFanfic_ChapterCreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateFanficRequest{Title: "T", Body: "hi"}
	m.fanficRepo.EXPECT().
		CreateWithDetails(mock.Anything, mock.Anything, userID, "T", "", "", "K", "", "in_progress", false, false, []string(nil), []string(nil), []dto.FanficCharacter(nil), false).
		Return(nil)
	m.fanficRepo.EXPECT().CreateChapter(mock.Anything, mock.Anything, mock.Anything, 1, "", "hi", 1).Return(errors.New("db"))

	// when
	_, err := svc.CreateFanfic(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestGetFanfic_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.fanficRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetFanfic(context.Background(), id, viewer, "")

	// then
	require.Error(t, err)
}

func TestGetFanfic_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.fanficRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(nil, nil)

	// when
	_, err := svc.GetFanfic(context.Background(), id, viewer, "")

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetFanfic_DraftNotAuthorNotAdmin_Hidden(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	author := uuid.New()
	row := &model.FanficRow{ID: id, UserID: author, Status: "draft"}
	m.fanficRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(row, nil)
	m.authz.EXPECT().Can(mock.Anything, viewer, authz.PermEditAnyTheory).Return(false)

	// when
	_, err := svc.GetFanfic(context.Background(), id, viewer, "")

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetFanfic_OK_IncrementsViewCount(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	author := uuid.New()
	row := &model.FanficRow{ID: id, UserID: author, Status: "complete", ViewCount: 3}
	m.fanficRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(row, nil)
	m.fanficRepo.EXPECT().RecordView(mock.Anything, id, "hash").Return(true, nil)
	m.fanficRepo.EXPECT().GetGenres(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharacters(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().ListChapters(mock.Anything, id).Return(nil, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.fanficRepo.EXPECT().GetComments(mock.Anything, id, viewer, []uuid.UUID(nil)).Return(nil, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewer, author).Return(false, nil)
	m.fanficRepo.EXPECT().GetReadingProgress(mock.Anything, viewer, id).Return(2, nil)

	// when
	got, err := svc.GetFanfic(context.Background(), id, viewer, "hash")

	// then
	require.NoError(t, err)
	assert.Equal(t, 4, got.ViewCount)
	assert.Equal(t, 2, got.ReadingProgress)
}

func TestGetFanfic_OK_AnonymousSkipsBlockCheck(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	row := &model.FanficRow{ID: id, UserID: author, Status: "complete"}
	m.fanficRepo.EXPECT().GetByID(mock.Anything, id, uuid.Nil).Return(row, nil)
	m.fanficRepo.EXPECT().GetGenres(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharacters(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().ListChapters(mock.Anything, id).Return(nil, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	m.fanficRepo.EXPECT().GetComments(mock.Anything, id, uuid.Nil, []uuid.UUID(nil)).Return(nil, nil)
	m.fanficRepo.EXPECT().GetReadingProgress(mock.Anything, uuid.Nil, id).Return(0, nil)

	// when
	got, err := svc.GetFanfic(context.Background(), id, uuid.Nil, "")

	// then
	require.NoError(t, err)
	assert.False(t, got.ViewerBlocked)
}

func TestGetFanfic_OK_WithCommentsThreaded(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	author := uuid.New()
	commentID := uuid.New()
	row := &model.FanficRow{ID: id, UserID: author, Status: "complete"}
	comments := []model.FanficCommentRow{{ID: commentID, UserID: author, Body: "hi"}}
	m.fanficRepo.EXPECT().GetByID(mock.Anything, id, viewer).Return(row, nil)
	m.fanficRepo.EXPECT().GetGenres(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharacters(mock.Anything, id).Return(nil, nil)
	m.fanficRepo.EXPECT().ListChapters(mock.Anything, id).Return([]model.FanficChapterSummaryRow{{ID: uuid.New(), ChapterNum: 1, Title: "Ch1"}}, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.fanficRepo.EXPECT().GetComments(mock.Anything, id, viewer, []uuid.UUID(nil)).Return(comments, nil)
	m.fanficRepo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{commentID}).Return(nil, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewer, author).Return(false, nil)
	m.fanficRepo.EXPECT().GetReadingProgress(mock.Anything, viewer, id).Return(0, nil)

	// when
	got, err := svc.GetFanfic(context.Background(), id, viewer, "")

	// then
	require.NoError(t, err)
	assert.Len(t, got.Comments, 1)
	assert.Len(t, got.Chapters, 1)
}

func TestUpdateFanfic_NotFoundIfAuthorLookupFails(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateFanfic_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(author, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T"})

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestUpdateFanfic_EmptyTitle(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestUpdateFanfic_TooManyGenres(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T", Genres: []string{"a", "b", "c"}})

	// then
	require.ErrorIs(t, err, ErrTooManyGenres)
}

func TestUpdateFanfic_TooManyTags(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)
	tags := make([]string, 11)
	for i := range tags {
		tags[i] = string(rune('a' + i))
	}

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T", Tags: tags})

	// then
	require.ErrorIs(t, err, ErrTooManyTags)
}

func TestUpdateFanfic_TagTooLong(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T", Tags: []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}})

	// then
	require.ErrorIs(t, err, ErrTagTooLong)
}

func TestUpdateFanfic_InvalidRating(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T", Rating: "X"})

	// then
	require.ErrorIs(t, err, ErrInvalidRating)
}

func TestUpdateFanfic_AsAdmin_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	req := dto.UpdateFanficRequest{
		Title:    " T ",
		Summary:  " s ",
		Series:   " ser ",
		Language: " en ",
		Status:   "in_progress",
	}
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(author, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.fanficRepo.EXPECT().RegisterSeries(mock.Anything, "ser").Return(nil)
	m.fanficRepo.EXPECT().RegisterLanguage(mock.Anything, "en").Return(nil)
	m.fanficRepo.EXPECT().
		UpdateWithDetails(mock.Anything, id, userID, "T", "s", "ser", "K", "en", "in_progress", false, false, []string(nil), []string(nil), []dto.FanficCharacter(nil), false, true).
		Return(nil)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, req)

	// then
	require.NoError(t, err)
}

func TestUpdateFanfic_RegistersOCCharacter(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	req := dto.UpdateFanficRequest{
		Title: "T",
		Characters: []dto.FanficCharacter{
			{CharacterID: "", CharacterName: "Custom"},
		},
	}
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)
	m.fanficRepo.EXPECT().
		UpdateWithDetails(mock.Anything, id, userID, "T", "", "", "K", "", "", false, false, []string(nil), []string(nil), req.Characters, false, false).
		Return(nil)
	m.fanficRepo.EXPECT().RegisterOCCharacter(mock.Anything, "Custom", userID).Return(nil)

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, req)

	// then
	require.NoError(t, err)
}

func TestUpdateFanfic_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)
	m.fanficRepo.EXPECT().
		UpdateWithDetails(mock.Anything, id, userID, "T", "", "", "K", "", "", false, false, []string(nil), []string(nil), []dto.FanficCharacter(nil), false, false).
		Return(errors.New("db"))

	// when
	err := svc.UpdateFanfic(context.Background(), id, userID, dto.UpdateFanficRequest{Title: "T"})

	// then
	require.Error(t, err)
}

func TestDeleteFanfic_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.fanficRepo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteFanfic(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteFanfic_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.fanficRepo.EXPECT().Delete(mock.Anything, id, userID).Return(errors.New("not owner"))

	// when
	err := svc.DeleteFanfic(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestListFanfics_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	params := repository.FanficListParams{Limit: 10, Offset: 0}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.fanficRepo.EXPECT().List(mock.Anything, viewer, params, []uuid.UUID(nil)).Return(nil, 0, errors.New("db"))

	// when
	_, err := svc.ListFanfics(context.Background(), viewer, params)

	// then
	require.Error(t, err)
}

func TestListFanfics_OK_TruncatesLongSummary(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	id := uuid.New()
	longSummary := ""
	for i := 0; i < 250; i++ {
		longSummary += "x"
	}
	rows := []model.FanficRow{{ID: id, UserID: uuid.New(), Title: "A", Summary: longSummary}}
	params := repository.FanficListParams{Limit: 10, Offset: 5}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.fanficRepo.EXPECT().List(mock.Anything, viewer, params, []uuid.UUID(nil)).Return(rows, 1, nil)
	m.fanficRepo.EXPECT().GetGenresBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)

	// when
	got, err := svc.ListFanfics(context.Background(), viewer, params)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 5, got.Offset)
	assert.Len(t, got.Fanfics[0].Summary, 203)
}

func TestListFanficsByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewer := uuid.New()
	m.fanficRepo.EXPECT().ListByUser(mock.Anything, userID, viewer, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListFanficsByUser(context.Background(), userID, viewer, 10, 0)

	// then
	require.Error(t, err)
}

func TestListFanficsByUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewer := uuid.New()
	id := uuid.New()
	rows := []model.FanficRow{{ID: id, UserID: userID, Title: "A"}}
	m.fanficRepo.EXPECT().ListByUser(mock.Anything, userID, viewer, 10, 0).Return(rows, 1, nil)
	m.fanficRepo.EXPECT().GetGenresBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)

	// when
	got, err := svc.ListFanficsByUser(context.Background(), userID, viewer, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Fanfics, 1)
}

func TestListFavourites_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewer := uuid.New()
	m.fanficRepo.EXPECT().ListFavourites(mock.Anything, userID, viewer, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListFavourites(context.Background(), userID, viewer, 10, 0)

	// then
	require.Error(t, err)
}

func TestListFavourites_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewer := uuid.New()
	id := uuid.New()
	rows := []model.FanficRow{{ID: id, UserID: userID, Title: "A"}}
	m.fanficRepo.EXPECT().ListFavourites(mock.Anything, userID, viewer, 10, 0).Return(rows, 1, nil)
	m.fanficRepo.EXPECT().GetGenresBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{id}).Return(nil, nil)

	// when
	got, err := svc.ListFavourites(context.Background(), userID, viewer, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
}

func TestUploadCoverImage_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.UploadCoverImage(context.Background(), id, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadCoverImage_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(author, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)

	// when
	_, err := svc.UploadCoverImage(context.Background(), id, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the fanfic author")
}

func TestUploadCoverImage_UploadError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return("", errors.New("disk full"))

	// when
	_, err := svc.UploadCoverImage(context.Background(), id, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadCoverImage_UpdateDBError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return("/uploads/fanfics/x.png", nil)
	m.fanficRepo.EXPECT().UpdateCoverImage(mock.Anything, id, "/uploads/fanfics/x.png", "").Return(errors.New("db boom"))

	// when
	_, err := svc.UploadCoverImage(context.Background(), id, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadCoverImage_OK_CtxCancelled(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return("/uploads/fanfics/x.png", nil)
	m.fanficRepo.EXPECT().UpdateCoverImage(mock.Anything, id, "/uploads/fanfics/x.png", "").Return(nil)

	// when
	url, err := svc.UploadCoverImage(ctx, id, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, "/uploads/fanfics/x.png", url)
}

func TestRemoveCoverImage_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.RemoveCoverImage(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestRemoveCoverImage_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(author, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)

	// when
	err := svc.RemoveCoverImage(context.Background(), id, userID)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authorised")
}

func TestRemoveCoverImage_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
	m.fanficRepo.EXPECT().UpdateCoverImage(mock.Anything, id, "", "").Return(nil)

	// when
	err := svc.RemoveCoverImage(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestCreateChapter_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.CreateChapter(context.Background(), fanficID, userID, dto.CreateChapterRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateChapter_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(uuid.New(), nil)

	// when
	_, err := svc.CreateChapter(context.Background(), fanficID, userID, dto.CreateChapterRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestCreateChapter_EmptyBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(userID, nil)

	// when
	_, err := svc.CreateChapter(context.Background(), fanficID, userID, dto.CreateChapterRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateChapter_NextNumberError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(userID, nil)
	m.fanficRepo.EXPECT().GetNextChapterNumber(mock.Anything, fanficID).Return(0, errors.New("boom"))

	// when
	_, err := svc.CreateChapter(context.Background(), fanficID, userID, dto.CreateChapterRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestCreateChapter_CreateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(userID, nil)
	m.fanficRepo.EXPECT().GetNextChapterNumber(mock.Anything, fanficID).Return(2, nil)
	m.fanficRepo.EXPECT().CreateChapter(mock.Anything, mock.Anything, fanficID, 2, "Title", "body", 1).Return(errors.New("db"))

	// when
	_, err := svc.CreateChapter(context.Background(), fanficID, userID, dto.CreateChapterRequest{Title: " Title ", Body: "body"})

	// then
	require.Error(t, err)
}

func TestCreateChapter_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(userID, nil)
	m.fanficRepo.EXPECT().GetNextChapterNumber(mock.Anything, fanficID).Return(3, nil)
	m.fanficRepo.EXPECT().CreateChapter(mock.Anything, mock.Anything, fanficID, 3, "T", "body", 1).Return(nil)
	m.fanficRepo.EXPECT().UpdateWordCount(mock.Anything, fanficID).Return(nil)

	// when
	id, err := svc.CreateChapter(context.Background(), fanficID, userID, dto.CreateChapterRequest{Title: "T", Body: "body"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestGetChapter_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapter(mock.Anything, fanficID, 1).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetChapter(context.Background(), fanficID, 1, uuid.Nil)

	// then
	require.Error(t, err)
}

func TestGetChapter_NilNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapter(mock.Anything, fanficID, 1).Return(nil, nil)

	// when
	_, err := svc.GetChapter(context.Background(), fanficID, 1, uuid.Nil)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetChapter_CountError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapter(mock.Anything, fanficID, 1).Return(&model.FanficChapterRow{ChapterNum: 1}, nil)
	m.fanficRepo.EXPECT().GetChapterCount(mock.Anything, fanficID).Return(0, errors.New("db"))

	// when
	_, err := svc.GetChapter(context.Background(), fanficID, 1, uuid.Nil)

	// then
	require.Error(t, err)
}

func TestGetChapter_OK_AnonSkipsProgress(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapter(mock.Anything, fanficID, 2).Return(&model.FanficChapterRow{ChapterNum: 2, Title: "t", Body: "b"}, nil)
	m.fanficRepo.EXPECT().GetChapterCount(mock.Anything, fanficID).Return(3, nil)

	// when
	got, err := svc.GetChapter(context.Background(), fanficID, 2, uuid.Nil)

	// then
	require.NoError(t, err)
	assert.True(t, got.HasPrev)
	assert.True(t, got.HasNext)
}

func TestGetChapter_OK_SetsProgress(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	viewer := uuid.New()
	m.fanficRepo.EXPECT().GetChapter(mock.Anything, fanficID, 1).Return(&model.FanficChapterRow{ChapterNum: 1}, nil)
	m.fanficRepo.EXPECT().GetChapterCount(mock.Anything, fanficID).Return(1, nil)
	m.fanficRepo.EXPECT().SetReadingProgress(mock.Anything, viewer, fanficID, 1).Return(nil)

	// when
	got, err := svc.GetChapter(context.Background(), fanficID, 1, viewer)

	// then
	require.NoError(t, err)
	assert.False(t, got.HasPrev)
	assert.False(t, got.HasNext)
}

func TestUpdateChapter_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.UpdateChapter(context.Background(), chapterID, userID, dto.UpdateChapterRequest{Body: "b"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateChapter_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateChapter(context.Background(), chapterID, userID, dto.UpdateChapterRequest{Body: "b"})

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestUpdateChapter_EmptyBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(userID, nil)

	// when
	err := svc.UpdateChapter(context.Background(), chapterID, userID, dto.UpdateChapterRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateChapter_UpdateError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(userID, nil)
	m.fanficRepo.EXPECT().UpdateChapter(mock.Anything, chapterID, "T", "body", 1).Return(errors.New("db"))

	// when
	err := svc.UpdateChapter(context.Background(), chapterID, userID, dto.UpdateChapterRequest{Title: "T", Body: "body"})

	// then
	require.Error(t, err)
}

func TestUpdateChapter_FanficIDError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(userID, nil)
	m.fanficRepo.EXPECT().UpdateChapter(mock.Anything, chapterID, "", "b", 1).Return(nil)
	m.fanficRepo.EXPECT().GetChapterFanficID(mock.Anything, chapterID).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.UpdateChapter(context.Background(), chapterID, userID, dto.UpdateChapterRequest{Body: "b"})

	// then
	require.Error(t, err)
}

func TestUpdateChapter_AsAdmin_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.fanficRepo.EXPECT().UpdateChapter(mock.Anything, chapterID, "", "b", 1).Return(nil)
	m.fanficRepo.EXPECT().GetChapterFanficID(mock.Anything, chapterID).Return(fanficID, nil)
	m.fanficRepo.EXPECT().UpdateWordCount(mock.Anything, fanficID).Return(nil)

	// when
	err := svc.UpdateChapter(context.Background(), chapterID, userID, dto.UpdateChapterRequest{Body: "b"})

	// then
	require.NoError(t, err)
}

func TestDeleteChapter_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.DeleteChapter(context.Background(), chapterID, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteChapter_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)

	// when
	err := svc.DeleteChapter(context.Background(), chapterID, userID)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteChapter_FanficIDError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(userID, nil)
	m.fanficRepo.EXPECT().GetChapterFanficID(mock.Anything, chapterID).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteChapter(context.Background(), chapterID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChapter_DeleteError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(userID, nil)
	m.fanficRepo.EXPECT().GetChapterFanficID(mock.Anything, chapterID).Return(fanficID, nil)
	m.fanficRepo.EXPECT().DeleteChapter(mock.Anything, chapterID).Return(errors.New("db"))

	// when
	err := svc.DeleteChapter(context.Background(), chapterID, userID)

	// then
	require.Error(t, err)
}

func TestDeleteChapter_AsAdmin_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	chapterID := uuid.New()
	userID := uuid.New()
	fanficID := uuid.New()
	m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
	m.fanficRepo.EXPECT().GetChapterFanficID(mock.Anything, chapterID).Return(fanficID, nil)
	m.fanficRepo.EXPECT().DeleteChapter(mock.Anything, chapterID).Return(nil)
	m.fanficRepo.EXPECT().UpdateWordCount(mock.Anything, fanficID).Return(nil)

	// when
	err := svc.DeleteChapter(context.Background(), chapterID, userID)

	// then
	require.NoError(t, err)
}

func TestFavourite_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.Favourite(context.Background(), userID, fanficID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestFavourite_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(true, nil)

	// when
	err := svc.Favourite(context.Background(), userID, fanficID)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestFavourite_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(false, nil)
	m.fanficRepo.EXPECT().Favourite(mock.Anything, userID, fanficID).Return(errors.New("db"))

	// when
	err := svc.Favourite(context.Background(), userID, fanficID)

	// then
	require.Error(t, err)
}

func TestFavourite_OK_SelfSkipsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(userID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	m.fanficRepo.EXPECT().Favourite(mock.Anything, userID, fanficID).Return(nil)

	// when
	err := svc.Favourite(context.Background(), userID, fanficID)

	// then
	require.NoError(t, err)
}

func TestFavourite_OK_OtherAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(false, nil)
	m.fanficRepo.EXPECT().Favourite(mock.Anything, userID, fanficID).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("stop goroutine")).Maybe()

	// when
	err := svc.Favourite(context.Background(), userID, fanficID)

	// then
	require.NoError(t, err)
}

func TestUnfavourite_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().Unfavourite(mock.Anything, userID, fanficID).Return(nil)

	// when
	err := svc.Unfavourite(context.Background(), userID, fanficID)

	// then
	require.NoError(t, err)
}

func TestUnfavourite_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().Unfavourite(mock.Anything, userID, fanficID).Return(errors.New("boom"))

	// when
	err := svc.Unfavourite(context.Background(), userID, fanficID)

	// then
	require.Error(t, err)
}

func TestGetLanguages_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.fanficRepo.EXPECT().GetLanguages(mock.Anything).Return([]string{"en"}, nil)

	// when
	got, err := svc.GetLanguages(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"en"}, got)
}

func TestGetSeries_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.fanficRepo.EXPECT().GetSeries(mock.Anything).Return([]string{"s"}, nil)

	// when
	got, err := svc.GetSeries(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"s"}, got)
}

func TestSearchOCCharacters_EmptyQueryDelegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.fanficRepo.EXPECT().SearchOCCharacters(mock.Anything, "").Return([]string{"Alice", "Bob"}, nil)

	// when
	got, err := svc.SearchOCCharacters(context.Background(), "   ")

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "Bob"}, got)
}

func TestSearchOCCharacters_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.fanficRepo.EXPECT().SearchOCCharacters(mock.Anything, "alice").Return([]string{"Alice"}, nil)

	// when
	got, err := svc.SearchOCCharacters(context.Background(), " alice ")

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice"}, got)
}

func TestCreateComment_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment_FanficNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(uuid.Nil, errors.New("no row"))

	// when
	_, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(true, nil)

	// when
	_, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestCreateComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(false, nil)
	m.fanficRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, fanficID, (*uuid.UUID)(nil), userID, "hi").
		Return(errors.New("db"))

	// when
	_, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestCreateComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(false, nil)
	m.fanficRepo.EXPECT().
		CreateComment(mock.Anything, mock.Anything, fanficID, (*uuid.UUID)(nil), userID, "hi").
		Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errors.New("stop goroutine")).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestUpdateComment_EmptyBody(t *testing.T) {
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
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.fanficRepo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "hi").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
}

func TestUpdateComment_AsOwner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
	m.fanficRepo.EXPECT().UpdateComment(mock.Anything, id, userID, "hi").Return(errors.New("not owner"))

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "hi"})

	// then
	require.Error(t, err)
}

func TestDeleteComment_AsAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.fanficRepo.EXPECT().DeleteCommentAsAdmin(mock.Anything, id).Return(nil)

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
	m.fanficRepo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(errors.New("not owner"))

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestLikeComment_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("no row"))

	// when
	err := svc.LikeComment(context.Background(), userID, id)

	// then
	require.Error(t, err)
}

func TestLikeComment_Blocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(true, nil)

	// when
	err := svc.LikeComment(context.Background(), userID, id)

	// then
	require.ErrorIs(t, err, block.ErrUserBlocked)
}

func TestLikeComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(false, nil)
	m.fanficRepo.EXPECT().LikeComment(mock.Anything, userID, id).Return(errors.New("db"))

	// when
	err := svc.LikeComment(context.Background(), userID, id)

	// then
	require.Error(t, err)
}

func TestLikeComment_SelfLikeSkipsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(userID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, userID).Return(false, nil)
	m.fanficRepo.EXPECT().LikeComment(mock.Anything, userID, id).Return(nil)

	// when
	err := svc.LikeComment(context.Background(), userID, id)

	// then
	require.NoError(t, err)
}

func TestLikeComment_OK_OtherAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	author := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(author, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, author).Return(false, nil)
	m.fanficRepo.EXPECT().LikeComment(mock.Anything, userID, id).Return(nil)
	m.fanficRepo.EXPECT().GetCommentFanficID(mock.Anything, id).Return(uuid.Nil, errors.New("stop goroutine")).Maybe()

	// when
	err := svc.LikeComment(context.Background(), userID, id)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().UnlikeComment(mock.Anything, userID, id).Return(nil)

	// when
	err := svc.UnlikeComment(context.Background(), userID, id)

	// then
	require.NoError(t, err)
}

func TestUnlikeComment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().UnlikeComment(mock.Anything, userID, id).Return(errors.New("boom"))

	// when
	err := svc.UnlikeComment(context.Background(), userID, id)

	// then
	require.Error(t, err)
}

func TestUploadCommentMedia_CommentNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	userID := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errors.New("no row"))

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
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.New(), nil)

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
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
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
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return("/uploads/fanfics/x.png", nil)
	m.fanficRepo.EXPECT().
		AddCommentMedia(mock.Anything, commentID, "/uploads/fanfics/x.png", "image", "", 0).
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
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return("/uploads/fanfics/x.png", nil)
	m.fanficRepo.EXPECT().
		AddCommentMedia(mock.Anything, commentID, "/uploads/fanfics/x.png", "image", "", 0).
		Return(int64(42), nil)

	// when
	resp, err := svc.UploadCommentMedia(ctx, commentID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, resp.ID)
	assert.Equal(t, "image", resp.MediaType)
}

func TestCountWords(t *testing.T) {
	// given
	html := "<p>hello <b>world</b></p>"

	// when
	got := countWords(html)

	// then
	assert.Equal(t, 2, got)
}

func TestSanitiseTags_DedupAndTrim(t *testing.T) {
	// given
	in := []string{"  a ", "A", "b", "", "B ", "c"}

	// when
	got := sanitiseTags(in)

	// then
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

var _ io.Reader = (*bytes.Reader)(nil)
