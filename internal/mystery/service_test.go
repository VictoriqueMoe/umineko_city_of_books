package mystery

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	repo         *repository.MockMysteryRepository
	userRepo     *repository.MockUserRepository
	authz        *authz.MockService
	blockSvc     *block.MockService
	notifService *notification.MockService
	uploadSvc    *upload.MockService
	settingsSvc  *settings.MockService
}

func newTestService(t *testing.T) (*service, *testMocks) {
	repo := repository.NewMockMysteryRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()
	svc := NewService(repo, userRepo, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub, contentfilter.New()).(*service)
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
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

func validCreateReq() dto.CreateMysteryRequest {
	return dto.CreateMysteryRequest{
		Title:      "Title",
		Body:       "Body",
		Difficulty: "medium",
	}
}

func TestListMysteries_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, "new", (*bool)(nil), 10, 0, []uuid.UUID(nil)).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListMysteries(context.Background(), "new", nil, viewer, 10, 0)

	// then
	require.Error(t, err)
}

func TestListMysteries_OK_TruncatesLongBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	longBody := ""
	for i := 0; i < 250; i++ {
		longBody += "x"
	}
	rows := []repository.MysteryRow{{ID: uuid.New(), Title: "T", Body: longBody}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, "new", (*bool)(nil), 5, 0, []uuid.UUID(nil)).Return(rows, 1, nil)

	// when
	got, err := svc.ListMysteries(context.Background(), "new", nil, viewer, 5, 0)

	// then
	require.NoError(t, err)
	require.Len(t, got.Mysteries, 1)
	assert.Equal(t, 203, len(got.Mysteries[0].Body))
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 5, got.Limit)
}

func TestListMysteries_OK_ShortBodyPreserved(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	rows := []repository.MysteryRow{{ID: uuid.New(), Body: "short"}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, "new", (*bool)(nil), 10, 0, []uuid.UUID(nil)).Return(rows, 1, nil)

	// when
	got, err := svc.ListMysteries(context.Background(), "new", nil, viewer, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, "short", got.Mysteries[0].Body)
}

func TestGetMystery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.Error(t, err)
}

func TestGetMystery_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewer := uuid.New()
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, nil)

	// when
	_, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetMystery_AsGameMasterOwner_SeesAll(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	other := uuid.New()
	playerID := uuid.New()
	row := &repository.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []repository.MysteryAttemptRow{{ID: uuid.New(), UserID: other, Body: "guess"}}
	clues := []dto.MysteryClue{{ID: 1, Body: "c1"}, {ID: 2, Body: "c2", PlayerID: &playerID}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, author).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, author).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)

	// when
	got, err := svc.GetMystery(context.Background(), id, author)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got.Attempts, 1)
	assert.Len(t, got.Clues, 2)
	assert.Equal(t, 1, got.PlayerCount)
}

func TestGetMystery_NonGM_NotSolved_FiltersAttemptsAndClues(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	viewer := uuid.New()
	other := uuid.New()
	row := &repository.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []repository.MysteryAttemptRow{
		{ID: uuid.New(), UserID: viewer, Body: "mine"},
		{ID: uuid.New(), UserID: other, Body: "not mine"},
	}
	otherPlayer := uuid.New()
	clues := []dto.MysteryClue{
		{ID: 1, Body: "public"},
		{ID: 2, Body: "mine", PlayerID: &viewer},
		{ID: 3, Body: "other", PlayerID: &otherPlayer},
	}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)

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
	row := &repository.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: true}
	attempts := []repository.MysteryAttemptRow{
		{ID: uuid.New(), UserID: viewer, Body: "mine"},
		{ID: uuid.New(), UserID: other, Body: "other"},
	}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)

	// when
	got, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.NoError(t, err)
	assert.Len(t, got.Attempts, 2)
}

func TestGetMystery_Solved_LoadsCommentsAndWinner(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	viewer := uuid.New()
	winnerID := uuid.New()
	winnerName := "win"
	winnerDisplay := "Winner"
	winnerAvatar := ""
	winnerRole := "user"
	row := &repository.MysteryRow{
		ID:                id,
		UserID:            author,
		Solved:            true,
		WinnerID:          &winnerID,
		WinnerUsername:    &winnerName,
		WinnerDisplayName: &winnerDisplay,
		WinnerAvatarURL:   &winnerAvatar,
		WinnerRole:        &winnerRole,
	}
	commentID := uuid.New()
	comments := []repository.MysteryCommentRow{{ID: commentID, UserID: author, Body: "post"}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(nil, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().GetComments(mock.Anything, id, viewer, []uuid.UUID(nil)).Return(comments, nil)
	m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{commentID}).Return(nil, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)

	// when
	got, err := svc.GetMystery(context.Background(), id, viewer)

	// then
	require.NoError(t, err)
	require.NotNil(t, got.Winner)
	assert.Equal(t, winnerID, got.Winner.ID)
	assert.Len(t, got.Comments, 1)
}

func TestGetMystery_SuperAdmin_SeesAll(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	author := uuid.New()
	admin := uuid.New()
	other := uuid.New()
	row := &repository.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []repository.MysteryAttemptRow{{ID: uuid.New(), UserID: other, Body: "x"}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, admin).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, admin).Return(authz.RoleSuperAdmin, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)

	// when
	got, err := svc.GetMystery(context.Background(), id, admin)

	// then
	require.NoError(t, err)
	assert.Len(t, got.Attempts, 1)
}

func TestCreateMystery_EmptyTitle(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Title = "   "

	// when
	_, err := svc.CreateMystery(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateMystery_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Body = "\n\t"

	// when
	_, err := svc.CreateMystery(context.Background(), uuid.New(), req)

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateMystery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().Create(mock.Anything, mock.Anything, userID, "Title", "Body", "medium", false).Return(errors.New("boom"))

	// when
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestCreateMystery_DefaultDifficulty(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := validCreateReq()
	req.Difficulty = ""
	m.repo.EXPECT().Create(mock.Anything, mock.Anything, userID, "Title", "Body", "medium", false).Return(nil)

	// when
	_, err := svc.CreateMystery(context.Background(), userID, req)

	// then
	require.NoError(t, err)
}

func TestCreateMystery_WithClues(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := validCreateReq()
	req.Clues = []dto.CreateClueRequest{
		{Body: "clue1"},
		{Body: "  "},
		{Body: "clue2", TruthType: "blue"},
	}
	m.repo.EXPECT().Create(mock.Anything, mock.Anything, userID, "Title", "Body", "medium", false).Return(nil)
	m.repo.EXPECT().AddClue(mock.Anything, mock.Anything, "clue1", "red", 0, (*uuid.UUID)(nil)).Return(nil)
	m.repo.EXPECT().AddClue(mock.Anything, mock.Anything, "clue2", "blue", 2, (*uuid.UUID)(nil)).Return(nil)

	// when
	id, err := svc.CreateMystery(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestCreateMystery_ClueError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := validCreateReq()
	req.Clues = []dto.CreateClueRequest{{Body: "c"}}
	m.repo.EXPECT().Create(mock.Anything, mock.Anything, userID, "Title", "Body", "medium", false).Return(nil)
	m.repo.EXPECT().AddClue(mock.Anything, mock.Anything, "c", "red", 0, (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	_, err := svc.CreateMystery(context.Background(), userID, req)

	// then
	require.Error(t, err)
}

func TestUpdateMystery_NotAuthorised(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.UpdateMystery(context.Background(), id, userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestUpdateMystery_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, nil)

	// when
	err := svc.UpdateMystery(context.Background(), id, userID, validCreateReq())

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateMystery_GetByIDError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(nil, errors.New("boom"))

	// when
	err := svc.UpdateMystery(context.Background(), id, userID, validCreateReq())

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateMystery_UpdateAsAdminError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	old := &repository.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, "Title", "Body", "medium", false).Return(errors.New("boom"))

	// when
	err := svc.UpdateMystery(context.Background(), id, userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestUpdateMystery_OwnerNoChanges_NoNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	old := &repository.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, "Title", "Body", "medium", false).Return(nil)
	m.repo.EXPECT().DeleteClues(mock.Anything, id).Return(nil)

	// when
	err := svc.UpdateMystery(context.Background(), id, userID, validCreateReq())

	// then
	require.NoError(t, err)
}

func TestUpdateMystery_AdminChange_SendsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	admin := uuid.New()
	author := uuid.New()
	old := &repository.MysteryRow{ID: id, UserID: author, Title: "Old Title", Body: "Body", Difficulty: "medium"}
	m.authz.EXPECT().Can(mock.Anything, admin, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, "Title", "Body", "medium", false).Return(nil)
	m.repo.EXPECT().DeleteClues(mock.Anything, id).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == author && p.Type == dto.NotifContentEdited
	})).Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).Return(nil).Maybe()

	// when
	err := svc.UpdateMystery(context.Background(), id, admin, validCreateReq())

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestUpdateMystery_WithClues_Replaces(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	old := &repository.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
	req := validCreateReq()
	req.Clues = []dto.CreateClueRequest{{Body: "new1"}, {Body: "  "}, {Body: "new2", TruthType: "blue"}}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateAsAdmin(mock.Anything, id, "Title", "Body", "medium", false).Return(nil)
	m.repo.EXPECT().DeleteClues(mock.Anything, id).Return(nil)
	m.repo.EXPECT().AddClue(mock.Anything, id, "new1", "red", 0, (*uuid.UUID)(nil)).Return(nil)
	m.repo.EXPECT().AddClue(mock.Anything, id, "new2", "blue", 2, (*uuid.UUID)(nil)).Return(nil)

	// when
	err := svc.UpdateMystery(context.Background(), id, userID, req)

	// then
	require.NoError(t, err)
}

func TestDeleteMystery_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(true)
	m.repo.EXPECT().DeleteAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteMystery(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteMystery_NonAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(false)
	m.repo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeleteMystery(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteMystery_NonAdmin_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyTheory).Return(false)
	m.repo.EXPECT().Delete(mock.Anything, id, userID).Return(errors.New("boom"))

	// when
	err := svc.DeleteMystery(context.Background(), id, userID)

	// then
	require.Error(t, err)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, mock.Anything, mid, userID, (*uuid.UUID)(nil), "body").Return(errors.New("boom"))

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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, mock.Anything, mid, userID, (*uuid.UUID)(nil), "body").Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, Username: "u"}, nil)

	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	id, err := svc.CreateAttempt(context.Background(), mid, userID, dto.CreateAttemptRequest{Body: "body"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
}

func TestDeleteAttempt_Admin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().DeleteAttemptAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteAttempt_NonAdmin(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
	m.repo.EXPECT().DeleteAttempt(mock.Anything, id, userID).Return(nil)

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
	m.repo.EXPECT().VoteAttempt(mock.Anything, userID, aid, 1).Return(errors.New("boom"))

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
	m.repo.EXPECT().VoteAttempt(mock.Anything, userID, aid, 0).Return(nil)

	// when
	err := svc.VoteAttempt(context.Background(), aid, userID, 0)

	// then
	require.NoError(t, err)
}

func TestVoteAttempt_Upvote_SendsNotification(t *testing.T) {
	// given
	svc, m := newTestService(t)
	aid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	mid := uuid.New()
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().VoteAttempt(mock.Anything, userID, aid, 1).Return(nil)

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
	waitOrFail(t, &wg, time.Second)
}

func TestMarkSolved_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSolved_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	aid := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestMarkSolved_AttemptAuthorError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSolved_AttemptMysteryError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSolved_AttemptWrongMystery(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(uuid.New(), nil)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.Error(t, err)
}

func TestMarkSolved_OwnAttempt(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.Error(t, err)
}

func TestMarkSolved_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid).Return(errors.New("boom"))

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.Error(t, err)
}

func TestMarkSolved_OK_Broadcasts(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid).Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, nil).Maybe()

	// when
	err := svc.MarkSolved(context.Background(), mid, userID, aid)

	// then
	require.NoError(t, err)
}

func TestMarkSolved_Admin_CanSolve(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	admin := uuid.New()
	author := uuid.New()
	aid := uuid.New()
	attemptAuthor := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(author, nil)
	m.authz.EXPECT().Can(mock.Anything, admin, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, aid).Return(attemptAuthor, nil)
	m.repo.EXPECT().GetAttemptMysteryID(mock.Anything, aid).Return(mid, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid).Return(nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, nil).Maybe()

	// when
	err := svc.MarkSolved(context.Background(), mid, admin, aid)

	// then
	require.NoError(t, err)
}

func TestAddClue_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.AddClue(context.Background(), uuid.New(), uuid.New(), dto.CreateClueRequest{Body: " "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestAddClue_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAddClue_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestAddClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(0, nil)
	m.repo.EXPECT().AddClue(mock.Anything, mid, "c", "red", 0, (*uuid.UUID)(nil)).Return(errors.New("boom"))

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.Error(t, err)
}

func TestAddClue_OK_DefaultTruthType(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(3, nil)
	m.repo.EXPECT().AddClue(mock.Anything, mid, "c", "red", 3, (*uuid.UUID)(nil)).Return(nil)

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c"})

	// then
	require.NoError(t, err)
}

func TestAddClue_Private_NotifiesPlayer(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	playerID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().CountClues(mock.Anything, mid).Return(0, nil)
	m.repo.EXPECT().AddClue(mock.Anything, mid, "c", "blue", 0, &playerID).Return(nil)

	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.RecipientID == playerID && p.Type == dto.NotifMysteryPrivateClue
	})).Return(nil).Maybe()

	// when
	err := svc.AddClue(context.Background(), mid, userID, dto.CreateClueRequest{Body: "c", TruthType: "blue", PlayerID: &playerID})

	// then
	require.NoError(t, err)
}

func TestGetLeaderboard_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().GetLeaderboard(mock.Anything, 10).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetLeaderboard(context.Background(), 10)

	// then
	require.Error(t, err)
}

func TestGetLeaderboard_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	entries := []repository.LeaderboardEntry{{UserID: uuid.New(), Username: "u", Score: 5}}
	m.repo.EXPECT().GetLeaderboard(mock.Anything, 10).Return(entries, nil)

	// when
	got, err := svc.GetLeaderboard(context.Background(), 10)

	// then
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, 5, got.Entries[0].Score)
}

func TestGetTopDetectiveIDs_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	expected := []string{"a", "b"}
	m.repo.EXPECT().GetTopDetectiveIDs(mock.Anything).Return(expected, nil)

	// when
	got, err := svc.GetTopDetectiveIDs(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestGetGMLeaderboard_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().GetGMLeaderboard(mock.Anything, 5).Return(nil, errors.New("boom"))

	// when
	_, err := svc.GetGMLeaderboard(context.Background(), 5)

	// then
	require.Error(t, err)
}

func TestGetGMLeaderboard_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	entries := []repository.GMLeaderboardEntry{{UserID: uuid.New(), Score: 7, MysteryCount: 2, PlayerCount: 4}}
	m.repo.EXPECT().GetGMLeaderboard(mock.Anything, 5).Return(entries, nil)

	// when
	got, err := svc.GetGMLeaderboard(context.Background(), 5)

	// then
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, 7, got.Entries[0].Score)
}

func TestGetTopGMIDs_Delegates(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return([]string{"x"}, nil)

	// when
	got, err := svc.GetTopGMIDs(context.Background())

	// then
	require.NoError(t, err)
	assert.Equal(t, []string{"x"}, got)
}

func TestListByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(nil, 0, errors.New("boom"))

	// when
	_, err := svc.ListByUser(context.Background(), userID, 10, 0)

	// then
	require.Error(t, err)
}

func TestListByUser_TruncatesLongBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	body := ""
	for i := 0; i < 300; i++ {
		body += "x"
	}
	rows := []repository.MysteryRow{{ID: uuid.New(), Body: body}}
	m.repo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(rows, 1, nil)

	// when
	got, err := svc.ListByUser(context.Background(), userID, 10, 0)

	// then
	require.NoError(t, err)
	assert.Equal(t, 203, len(got.Mysteries[0].Body))
	assert.Equal(t, 1, got.Total)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateComment(mock.Anything, mock.Anything, mid, (*uuid.UUID)(nil), userID, "hi").Return(errors.New("boom"))

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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateComment(mock.Anything, mock.Anything, mid, (*uuid.UUID)(nil), userID, "hi").Return(nil)

	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "D"}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil).Maybe()

	// when
	id, err := svc.CreateComment(context.Background(), mid, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
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
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateComment(mock.Anything, mock.Anything, mid, &parentID, userID, "hi").Return(nil)

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
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
	m.repo.EXPECT().UpdateCommentAsAdmin(mock.Anything, id, "new").Return(nil)

	// when
	err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: "new"})

	// then
	require.NoError(t, err)
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
	m.repo.EXPECT().DeleteCommentAsAdmin(mock.Anything, id).Return(nil)

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
	m.repo.EXPECT().DeleteComment(mock.Anything, id, userID).Return(nil)

	// when
	err := svc.DeleteComment(context.Background(), id, userID)

	// then
	require.NoError(t, err)
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
	_, err := svc.UploadCommentMedia(context.Background(), cid, userID, "image/png", 10, bytes.NewReader(nil))

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
	_, err := svc.UploadCommentMedia(context.Background(), cid, userID, "image/png", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadAttachment_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestUploadAttachment_FileTooBig(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(5)

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 999, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_DuplicateName(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return([]dto.MysteryAttachment{{FileName: "f.txt"}}, nil)

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_SaveFileError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.uploadSvc.EXPECT().SaveFile(mock.Anything, "f.txt", mock.Anything).Return("", errors.New("boom"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_AddAttachmentError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.uploadSvc.EXPECT().SaveFile(mock.Anything, "f.txt", mock.Anything).Return("/uploads/x", nil)
	m.repo.EXPECT().AddAttachment(mock.Anything, mid, "/uploads/x", "f.txt", 10).Return(int64(0), errors.New("boom"))

	// when
	_, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.Error(t, err)
}

func TestUploadAttachment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxGeneralSize).Return(1024 * 1024)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.uploadSvc.EXPECT().SaveFile(mock.Anything, "f.txt", mock.Anything).Return("/uploads/x", nil)
	m.repo.EXPECT().AddAttachment(mock.Anything, mid, "/uploads/x", "f.txt", 10).Return(int64(42), nil)

	// when
	got, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, got.ID)
	assert.Equal(t, "/uploads/x", got.FileURL)
}

func TestDeleteAttachment_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteAttachment_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteAttachment_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(nil, nil)
	m.repo.EXPECT().DeleteAttachment(mock.Anything, int64(1), mid).Return(errors.New("boom"))

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.Error(t, err)
}

func TestDeleteAttachment_OK_DeletesFile(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	attachments := []dto.MysteryAttachment{{ID: 1, FileURL: "/uploads/mystery-attachments/abc/f.txt"}}
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, mid).Return(attachments, nil)
	m.repo.EXPECT().DeleteAttachment(mock.Anything, int64(1), mid).Return(nil)
	m.uploadSvc.EXPECT().GetUploadDir().Return("/tmp/nonexistent-dir")

	// when
	err := svc.DeleteAttachment(context.Background(), 1, mid, userID)

	// then
	require.NoError(t, err)
}

func TestSetPaused_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSetPaused_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestSetPaused_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().SetPaused(mock.Anything, mid, true).Return(errors.New("boom"))

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.Error(t, err)
}

func TestSetPaused_OK_Pause(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().SetPaused(mock.Anything, mid, true).Return(nil)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()

	// when
	err := svc.SetPaused(context.Background(), mid, userID, true)

	// then
	require.NoError(t, err)
}

func TestSetPaused_OK_Unpause_NotifiesPlayers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	player := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().SetPaused(mock.Anything, mid, false).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return([]uuid.UUID{player}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifMysteryUnpaused && p.RecipientID == player
	})).Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).Return(nil).Maybe()

	// when
	err := svc.SetPaused(context.Background(), mid, userID, false)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestSetGmAway_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSetGmAway_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestSetGmAway_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().SetGmAway(mock.Anything, mid, true).Return(errors.New("boom"))

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.Error(t, err)
}

func TestSetGmAway_OK_Away(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().SetGmAway(mock.Anything, mid, true).Return(nil)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, true)

	// then
	require.NoError(t, err)
}

func TestSetGmAway_OK_Back_NotifiesPlayers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	player := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().SetGmAway(mock.Anything, mid, false).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return([]uuid.UUID{player}, nil).Maybe()
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("http://e.test").Maybe()
	m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
		return p.Type == dto.NotifMysteryGmBack && p.RecipientID == player
	})).Run(func(_ context.Context, _ dto.NotifyParams) { wg.Done() }).Return(nil).Maybe()

	// when
	err := svc.SetGmAway(context.Background(), mid, userID, false)

	// then
	require.NoError(t, err)
	waitOrFail(t, &wg, time.Second)
}

func TestDeleteClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().DeleteClue(mock.Anything, 7).Return(errors.New("boom"))

	// when
	err := svc.DeleteClue(context.Background(), mid, 7, userID)

	// then
	require.Error(t, err)
}

func TestDeleteClue_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().DeleteClue(mock.Anything, 7).Return(nil)

	// when
	err := svc.DeleteClue(context.Background(), mid, 7, userID)

	// then
	require.NoError(t, err)
}

func TestUpdateClue_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateClue(context.Background(), uuid.New(), 1, uuid.New(), " ")

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateClue_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().UpdateClue(mock.Anything, 3, "new").Return(errors.New("boom"))

	// when
	err := svc.UpdateClue(context.Background(), uuid.New(), 3, uuid.New(), "new")

	// then
	require.Error(t, err)
}

func TestUpdateClue_OK_Trims(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.repo.EXPECT().UpdateClue(mock.Anything, 3, "new").Return(nil)

	// when
	err := svc.UpdateClue(context.Background(), uuid.New(), 3, uuid.New(), "  new  ")

	// then
	require.NoError(t, err)
}
