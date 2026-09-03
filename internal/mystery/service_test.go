package mystery

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
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
	comments     *repository.MockCommentDAO[uuid.UUID]
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
	comments := repository.NewMockCommentDAO[uuid.UUID](t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notifSvc, repository.CommentDAOs{
		ByID: map[string]repository.CommentDAO[uuid.UUID]{string(mention.KindMysteryComment): comments},
	})
	svc := NewService(repo, userRepo, followRepo, auditRepo, authzSvc, blockSvc, notifSvc, mentionSvc, settingsSvc, uploadSvc, mediaProc, hub, contentfilter.New(), nil).(*service)
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
	fanout := make(chan uuid.UUID, 8)
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, mock.Anything).Run(func(_ context.Context, userID uuid.UUID, _ ...*sql.Tx) {
		fanout <- userID
	}).Return(nil, nil).Maybe()
	return svc, &testMocks{
		fanout:       fanout,
		repo:         repo,
		comments:     comments,
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

func TestCreateMystery_MentionInTheBodyNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	mysteryID := uuid.New()
	mentionedID := uuid.New()

	req := validCreateReq()
	req.Body = "solve this with @alice"

	spec := validCreateSpec(userID)
	spec.Body = req.Body

	m.repo.EXPECT().CreateWithClues(mock.Anything, spec).Return(&repository.MysteryRow{ID: mysteryID}, nil)
	stubActor(m, userID, "Battler")
	stubMentionOf(m, userID, mentionedID, "alice")

	var wg sync.WaitGroup
	wg.Add(1)

	var mentioned dto.NotifyParams
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			mentioned = p
			wg.Done()

			return nil
		})

	// when
	_, err := svc.CreateMystery(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, mysteryID, mentioned.ReferenceID)
	assert.Equal(t, "mystery", mentioned.ReferenceType)
	assert.Equal(t, "/mystery/"+mysteryID.String(), mentioned.EmailLink)
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
	synctest.Test(t, testUpdateMysteryAdminChangeSendsNotification)
}

func testUpdateMysteryAdminChangeSendsNotification(t *testing.T) {
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
	wg.Wait()
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
