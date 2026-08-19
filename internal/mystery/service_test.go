package mystery

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
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
	auditRepo    *repository.MockAuditLogRepository
	authz        *authz.MockService
	blockSvc     *block.MockService
	notifService *notification.MockService
	uploadSvc    *upload.MockService
	settingsSvc  *settings.MockService
	followRepo   *repository.MockFollowRepository
	fanout       chan uuid.UUID
}

func newTestService(t *testing.T) (*service, *testMocks) {
	repo := repository.NewMockMysteryRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	followRepo := repository.NewMockFollowRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	mediaProc := &media.Processor{}
	hub := ws.NewHub()
	svc := NewService(repo, userRepo, followRepo, auditRepo, authzSvc, blockSvc, notifSvc, settingsSvc, uploadSvc, mediaProc, hub, contentfilter.New()).(*service)
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
	fanout := make(chan uuid.UUID, 8)
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, mock.Anything).Run(func(_ context.Context, userID uuid.UUID, _ ...*sql.Tx) {
		fanout <- userID
	}).Return(nil, nil).Maybe()
	return svc, &testMocks{
		fanout:       fanout,
		repo:         repo,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		authz:        authzSvc,
		blockSvc:     blockSvc,
		notifService: notifSvc,
		uploadSvc:    uploadSvc,
		settingsSvc:  settingsSvc,
		followRepo:   followRepo,
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
	_, err := svc.ListMysteries(context.Background(), "new", nil, viewer, bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestListMysteries_OK_TruncatesLongBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	var longBody strings.Builder
	for range 250 {
		longBody.WriteString("x")
	}
	rows := []repository.MysteryRow{{ID: uuid.New(), Title: "T", Body: longBody.String()}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, "new", (*bool)(nil), 5, 0, []uuid.UUID(nil)).Return(rows, 1, nil)

	// when
	got, err := svc.ListMysteries(context.Background(), "new", nil, viewer, bounds.NewPage(5, 0))

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
	got, err := svc.ListMysteries(context.Background(), "new", nil, viewer, bounds.NewPage(10, 0))

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
	row := &repository.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []repository.MysteryAttemptRow{{ID: uuid.New(), UserID: other, Body: "guess"}}
	clues := []dto.MysteryClue{{ID: 1, Body: "c1"}, {ID: 2, Body: "c2", PlayerID: new(uuid.New())}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, author).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, author).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

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
	clues := []dto.MysteryClue{
		{ID: 1, Body: "public"},
		{ID: 2, Body: "mine", PlayerID: &viewer},
		{ID: 3, Body: "other", PlayerID: new(uuid.New())},
	}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, id, viewer).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, id, viewer).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, viewer).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, id, admin).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, id, admin).Return(attempts, nil)
	m.authz.EXPECT().GetRole(mock.Anything, admin).Return(authz.RoleSuperAdmin, nil)
	m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()

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

func validCreateSpec(userID uuid.UUID, clues ...repository.NewClue) repository.NewMystery {
	if clues == nil {
		clues = []repository.NewClue{}
	}

	return repository.NewMystery{
		UserID:     userID,
		Title:      "Title",
		Body:       "Body",
		Difficulty: "medium",
		Knox:       dto.DefaultKnoxContract(),
		Clues:      clues,
	}
}

func TestCreateMystery_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(nil, errors.New("boom"))

	// when
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq())

	// then
	require.Error(t, err)
}

func TestCreateMystery_NotifiesFollowers(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(&repository.MysteryRow{ID: uuid.New()}, nil)

	// when
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq())

	// then
	require.NoError(t, err)
	select {
	case actorID := <-m.fanout:
		assert.Equal(t, userID, actorID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the follower fan-out")
	}
}

func TestCreateMystery_RepoErrorSkipsFollowerFanout(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(nil, errors.New("boom"))

	// when
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq())

	// then
	require.Error(t, err)
	assert.Empty(t, m.fanout)
}

func TestCreateMystery_DefaultDifficulty(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := validCreateReq()
	req.Difficulty = ""
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(&repository.MysteryRow{ID: uuid.New()}, nil)

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
	spec := validCreateSpec(userID,
		repository.NewClue{Body: "clue1", TruthType: "red", SortOrder: 0},
		repository.NewClue{Body: "clue2", TruthType: "blue", SortOrder: 2},
	)
	m.repo.EXPECT().CreateWithClues(mock.Anything, spec).Return(&repository.MysteryRow{ID: uuid.New()}, nil)

	// when
	id, err := svc.CreateMystery(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
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

func validUpdateSpec(id uuid.UUID, knox dto.KnoxContract, clues ...repository.NewClue) repository.MysteryUpdate {
	if clues == nil {
		clues = []repository.NewClue{}
	}

	return repository.MysteryUpdate{
		ID:         id,
		Title:      "Title",
		Body:       "Body",
		Difficulty: "medium",
		Knox:       knox,
		Clues:      clues,
	}
}

func TestUpdateMystery_KnoxContract(t *testing.T) {
	waived := dto.DefaultKnoxContract()
	waived.NoSupernatural = false

	tests := []struct {
		name         string
		stored       dto.KnoxContract
		published    bool
		attemptCount int
		requested    *dto.KnoxContract
		wantErr      error
		wantWritten  dto.KnoxContract
	}{
		{
			name:         "a mystery that predates the contract may still declare one mid game",
			stored:       dto.DefaultKnoxContract(),
			published:    false,
			attemptCount: 9,
			requested:    &waived,
			wantWritten:  waived,
		},
		{
			name:         "changing the contract before any attempt is allowed",
			stored:       dto.DefaultKnoxContract(),
			published:    true,
			attemptCount: 0,
			requested:    &waived,
			wantWritten:  waived,
		},
		{
			name:         "changing the contract after an attempt is refused",
			stored:       dto.DefaultKnoxContract(),
			published:    true,
			attemptCount: 1,
			requested:    &waived,
			wantErr:      ErrContractLocked,
		},
		{
			name:         "an unchanged contract still saves after an attempt",
			stored:       waived,
			published:    true,
			attemptCount: 3,
			requested:    &waived,
			wantWritten:  waived,
		},
		{
			name:         "omitting the contract keeps the stored one",
			stored:       waived,
			published:    true,
			attemptCount: 5,
			requested:    nil,
			wantWritten:  waived,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			req := validCreateReq()
			req.KnoxContract = tt.requested
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
			m.repo.EXPECT().GetByID(mock.Anything, id).Return(&repository.MysteryRow{
				ID: id, UserID: userID, Knox: tt.stored, KnoxPublished: tt.published, AttemptCount: tt.attemptCount,
			}, nil)

			m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil).Maybe()

			var written dto.KnoxContract
			if tt.wantErr == nil {
				m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, tt.wantWritten)).
					Run(func(_ context.Context, spec repository.MysteryUpdate, _ ...*sql.Tx) {
						written = spec.Knox
					}).Return(nil)
			}

			// when
			err := svc.UpdateMystery(context.Background(), id, userID, req)

			// then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantWritten, written)
		})
	}
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

func TestUpdateMystery_UpdateWithCluesError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	old := &repository.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, old.Knox)).Return(errors.New("boom"))

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
	m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, old.Knox)).Return(nil)

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
	m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, old.Knox)).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    admin,
		Action:     repository.AuditActionMysteryUpdateAdmin,
		TargetType: repository.AuditTargetMystery,
		TargetID:   id.String(),
		Details:    `title="Old Title" changed="title" clues=rewritten`,
		SubjectID:  author,
	}).Return(nil)

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
	m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, old.Knox,
		repository.NewClue{Body: "new1", TruthType: "red", SortOrder: 0},
		repository.NewClue{Body: "new2", TruthType: "blue", SortOrder: 2},
	)).Return(nil)

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
	m.repo.EXPECT().DeleteWithFiles(mock.Anything, repository.MysteryDelete{ID: id, UserID: userID, AsAdmin: true}).Return([]string{"/uploads/mystery/a.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mystery/a.png"})

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
	m.repo.EXPECT().DeleteWithFiles(mock.Anything, repository.MysteryDelete{ID: id, UserID: userID}).Return([]string{"/uploads/mystery/a.png", "/uploads/mystery/a_thumb.png"}, nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mystery/a.png", "/uploads/mystery/a_thumb.png"})

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
	m.repo.EXPECT().DeleteWithFiles(mock.Anything, repository.MysteryDelete{ID: id, UserID: userID}).Return(nil, errors.New("boom"))

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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, userID).Return(false, nil)
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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, userID).Return(false, nil)
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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, userID).Return(false, nil)
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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, userID).Return(false, nil)
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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, userID).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, mid, userID, (*uuid.UUID)(nil), "body").Return(nil, errors.New("boom"))

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
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, userID).Return(false, nil)
	m.repo.EXPECT().IsPaused(mock.Anything, mid).Return(false, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.repo.EXPECT().CreateAttempt(mock.Anything, mid, userID, (*uuid.UUID)(nil), "body").Return(&repository.MysteryAttemptRow{ID: uuid.New(), AuthorUsername: "u"}, nil)

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
	attemptAuthor := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, id).Return(attemptAuthor, nil)
	m.repo.EXPECT().DeleteAttemptAsAdmin(mock.Anything, id).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryAttemptDeleteAdmin,
		TargetType: repository.AuditTargetMysteryAttempt,
		TargetID:   id.String(),
		SubjectID:  attemptAuthor,
	}).Return(nil)

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestDeleteAttempt_ModeratorDeletingOwnAttempt_WritesNoAuditRow(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, id).Return(userID, nil)
	m.repo.EXPECT().DeleteAttemptAsAdmin(mock.Anything, id).Return(nil)

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.NoError(t, err)
	m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestDeleteAttempt_AttemptNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
	m.repo.EXPECT().GetAttemptAuthorID(mock.Anything, id).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteAttempt(context.Background(), id, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
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
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&repository.MysteryRow{ID: mid, UserID: userID}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, attemptAuthor).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid, true).Return(errors.New("boom"))

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
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&repository.MysteryRow{ID: mid, UserID: userID}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, attemptAuthor).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid, true).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysterySolved,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "attempt=" + aid.String(),
		SubjectID:  attemptAuthor,
	}).Return(nil)
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
	m.repo.EXPECT().GetByID(mock.Anything, mid).Return(&repository.MysteryRow{ID: mid, UserID: author}, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mid, attemptAuthor).Return(false, nil)
	m.repo.EXPECT().MarkSolved(mock.Anything, mid, aid, true).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    admin,
		Action:     repository.AuditActionMysterySolved,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "attempt=" + aid.String(),
		SubjectID:  attemptAuthor,
	}).Return(nil)
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

func TestMarkPermanentlySolved_AuditsTheClose(t *testing.T) {
	tests := []struct {
		name        string
		actorIsGM   bool
		wantDetails string
	}{
		{name: "the game master closes their own board", actorIsGM: true, wantDetails: "by=author"},
		{name: "staff closes someone else's board", actorIsGM: false, wantDetails: "by=staff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid := uuid.New()
			authorID := uuid.New()
			actorID := authorID
			if !tt.actorIsGM {
				actorID = uuid.New()
				m.authz.EXPECT().Can(mock.Anything, actorID, authz.PermEditAnyTheory).Return(true)
			}
			m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(authorID, nil)
			m.repo.EXPECT().MarkPermanentlySolved(mock.Anything, mid).Return(nil)
			m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
				ActorID:    actorID,
				Action:     repository.AuditActionMysteryClosed,
				TargetType: repository.AuditTargetMystery,
				TargetID:   mid.String(),
				Details:    tt.wantDetails,
				SubjectID:  authorID,
			}).Return(nil)
			m.repo.EXPECT().GetSolverIDs(mock.Anything, mid).Return(nil, nil).Maybe()
			m.repo.EXPECT().GetPlayerIDs(mock.Anything, mid).Return(nil, nil).Maybe()
			m.repo.EXPECT().GetTopGMIDs(mock.Anything).Return(nil, nil).Maybe()

			// when
			err := svc.MarkPermanentlySolved(context.Background(), mid, actorID)

			// then
			require.NoError(t, err)
		})
	}
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
	m.repo.EXPECT().AddClue(mock.Anything, mid, repository.NewClue{Body: "c", TruthType: "red", SortOrder: 0}).Return(nil, errors.New("boom"))

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
	m.repo.EXPECT().AddClue(mock.Anything, mid, repository.NewClue{Body: "c", TruthType: "red", SortOrder: 3}).Return(&dto.MysteryClue{ID: 1}, nil)

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
	m.repo.EXPECT().AddClue(mock.Anything, mid, repository.NewClue{Body: "c", TruthType: "blue", SortOrder: 0, PlayerID: &playerID}).Return(&dto.MysteryClue{ID: 1}, nil)

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
	_, err := svc.GetLeaderboard(context.Background(), bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestGetLeaderboard_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	entries := []repository.LeaderboardEntry{{UserID: uuid.New(), Username: "u", Score: 5}}
	m.repo.EXPECT().GetLeaderboard(mock.Anything, 10).Return(entries, nil)

	// when
	got, err := svc.GetLeaderboard(context.Background(), bounds.NewPage(10, 0))

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
	_, err := svc.GetGMLeaderboard(context.Background(), bounds.NewPage(5, 0))

	// then
	require.Error(t, err)
}

func TestGetGMLeaderboard_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	entries := []repository.GMLeaderboardEntry{{UserID: uuid.New(), Score: 7, MysteryCount: 2, PlayerCount: 4}}
	m.repo.EXPECT().GetGMLeaderboard(mock.Anything, 5).Return(entries, nil)

	// when
	got, err := svc.GetGMLeaderboard(context.Background(), bounds.NewPage(5, 0))

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
	_, err := svc.ListByUser(context.Background(), userID, bounds.NewPage(10, 0))

	// then
	require.Error(t, err)
}

func TestListByUser_TruncatesLongBody(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	var body strings.Builder
	for range 300 {
		body.WriteString("x")
	}
	rows := []repository.MysteryRow{{ID: uuid.New(), Body: body.String()}}
	m.repo.EXPECT().ListByUser(mock.Anything, userID, 10, 0).Return(rows, 1, nil)

	// when
	got, err := svc.ListByUser(context.Background(), userID, bounds.NewPage(10, 0))

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
	m.repo.EXPECT().CreateComment(mock.Anything, mid, (*uuid.UUID)(nil), userID, "hi").Return(nil, errors.New("boom"))

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
	m.repo.EXPECT().CreateComment(mock.Anything, mid, (*uuid.UUID)(nil), userID, "hi").Return(&repository.CommentRow{ID: uuid.New()}, nil)

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
	m.repo.EXPECT().CreateComment(mock.Anything, mid, &parentID, userID, "hi").Return(&repository.CommentRow{ID: uuid.New()}, nil)

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
	m.uploadSvc.EXPECT().SaveFile(mock.Anything, mock.MatchedBy(isServerGeneratedTxtName), mock.Anything).Return("", errors.New("boom"))

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
	m.uploadSvc.EXPECT().SaveFile(mock.Anything, mock.MatchedBy(isServerGeneratedTxtName), mock.Anything).Return("/uploads/x", nil)
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
	m.uploadSvc.EXPECT().SaveFile(mock.Anything, mock.MatchedBy(isServerGeneratedTxtName), mock.Anything).Return("/uploads/x", nil)
	m.repo.EXPECT().AddAttachment(mock.Anything, mid, "/uploads/x", "f.txt", 10).Return(int64(42), nil)

	// when
	got, err := svc.UploadAttachment(context.Background(), mid, userID, "f.txt", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	assert.Equal(t, 42, got.ID)
	assert.Equal(t, "/uploads/x", got.FileURL)
}

func isServerGeneratedTxtName(name string) bool {
	if !strings.HasSuffix(name, ".txt") {
		return false
	}

	_, err := uuid.Parse(strings.TrimSuffix(name, ".txt"))

	return err == nil
}

func TestAttachmentDiskName(t *testing.T) {
	// given
	tests := []struct {
		name        string
		sniffedType string
		wantExt     string
		wantErr     bool
	}{
		{name: "pdf keeps its extension", sniffedType: "application/pdf", wantExt: ".pdf"},
		{name: "plain text becomes txt", sniffedType: "text/plain", wantExt: ".txt"},
		{name: "docx arrives sniffed as zip", sniffedType: "application/zip", wantExt: ".docx"},
		{name: "html is rejected", sniffedType: "text/html", wantErr: true},
		{name: "svg is rejected", sniffedType: "image/svg+xml", wantErr: true},
		{name: "unrecognised binary is rejected", sniffedType: "application/octet-stream", wantErr: true},
		{name: "empty type is rejected", sniffedType: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got, err := attachmentDiskName(tt.sniffedType)

			// then
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrAttachmentType)
				assert.Empty(t, got)

				return
			}

			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(got, tt.wantExt), "expected %q to end in %q", got, tt.wantExt)

			_, parseErr := uuid.Parse(strings.TrimSuffix(got, tt.wantExt))
			assert.NoError(t, parseErr, "expected %q to be a uuid plus extension", got)
		})
	}
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
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryClueDelete,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "clue=7",
	}).Return(nil)

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
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().UpdateClue(mock.Anything, 3, "new").Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, repository.NewAuditEntry{
		ActorID:    userID,
		Action:     repository.AuditActionMysteryClueUpdate,
		TargetType: repository.AuditTargetMystery,
		TargetID:   mid.String(),
		Details:    "clue=3",
	}).Return(nil)

	// when
	err := svc.UpdateClue(context.Background(), mid, 3, userID, "  new  ")

	// then
	require.NoError(t, err)
}

func TestUploadMedia_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	_, err := svc.UploadMedia(context.Background(), mid, userID, "image/png", "photo.png", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUploadMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	_, err := svc.UploadMedia(context.Background(), mid, userID, "image/png", "photo.png", 10, bytes.NewReader(nil))

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteMedia_MysteryNotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, errors.New("boom"))

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.New(), nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.ErrorIs(t, err, ErrNotAuthor)
}

func TestDeleteMedia_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().DeleteMedia(mock.Anything, int64(1), mid).Return("", errors.New("boom"))

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.Error(t, err)
}

func TestDeleteMedia_OK_DeletesFile(t *testing.T) {
	// given
	svc, m := newTestService(t)
	mid := uuid.New()
	userID := uuid.New()
	m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(userID, nil)
	m.repo.EXPECT().DeleteMedia(mock.Anything, int64(1), mid).Return("/uploads/mysteries/x.png", nil)
	m.uploadSvc.EXPECT().Delete([]string{"/uploads/mysteries/x.png"}).Return()

	// when
	err := svc.DeleteMedia(context.Background(), 1, mid, userID)

	// then
	require.NoError(t, err)
}
