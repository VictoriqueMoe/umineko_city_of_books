package mystery

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/upload"
	"umineko_city_of_books/internal/upload/uploadtest"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	repo         *repository.MockMysteryRepository
	commentSvc   *comment.MockService
	userRepo     *repository.MockUserRepository
	auditRepo    *repository.MockAuditLogRepository
	authz        *authz.MockService
	blockSvc     *block.MockService
	notifService *notification.MockService
	uploadSvc    *upload.MockService
	settingsSvc  *settings.MockService
	followRepo   *repository.MockFollowRepository
	activity     *sidebar.MockActivityBumper
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
	activity := sidebar.NewMockActivityBumper(t)
	commentSvc := comment.NewMockService(t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notifSvc)
	svc := NewService(repo, userRepo, followRepo, auditRepo, authzSvc, blockSvc, notifSvc, mentionSvc, commentSvc, settingsSvc, uploadSvc, mediaProc, hub, contentfilter.New(), nil, activity).(*service)
	notifSvc.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
	fanout := make(chan uuid.UUID, 8)
	followRepo.EXPECT().GetFollowerIDsToNotify(mock.Anything, mock.Anything).Run(func(_ context.Context, userID uuid.UUID, _ ...*sql.Tx) {
		fanout <- userID
	}).Return(nil, nil).Maybe()
	return svc, &testMocks{
		fanout:       fanout,
		repo:         repo,
		commentSvc:   commentSvc,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		authz:        authzSvc,
		blockSvc:     blockSvc,
		notifService: notifSvc,
		uploadSvc:    uploadSvc,
		settingsSvc:  settingsSvc,
		followRepo:   followRepo,
		activity:     activity,
	}
}

func validCreateReq() dto.CreateMysteryRequest {
	return dto.CreateMysteryRequest{
		Title:      "Title",
		Body:       "Body",
		Difficulty: "medium",
	}
}

func TestAFailedMysteryAuthorLookupIsSurfacedNotReportedAsNotFound(t *testing.T) {
	ctx := context.Background()
	mid := uuid.New()
	userID := uuid.New()
	cases := []struct {
		name string
		call func(*service) error
	}{
		{name: "mark solved", call: func(s *service) error { return s.MarkSolved(ctx, mid, userID, uuid.New()) }},
		{name: "close for good", call: func(s *service) error { return s.MarkPermanentlySolved(ctx, mid, userID) }},
		{name: "add a clue", call: func(s *service) error { return s.AddClue(ctx, mid, userID, dto.CreateClueRequest{Body: "c"}) }},
		{name: "pause", call: func(s *service) error { return s.SetPaused(ctx, mid, userID, true) }},
		{name: "mark the gm away", call: func(s *service) error { return s.SetGmAway(ctx, mid, userID, true) }},
		{name: "delete media", call: func(s *service) error { return s.DeleteMedia(ctx, 1, mid, userID) }},
		{name: "delete an attachment", call: func(s *service) error { return s.DeleteAttachment(ctx, 1, mid, userID) }},
		{
			name: "upload an attachment",
			call: func(s *service) error {
				_, err := s.UploadAttachment(ctx, mid, userID, "f.txt", 10, strings.NewReader("x"))
				return err
			},
		},
		{
			name: "upload media",
			call: func(s *service) error {
				_, err := s.UploadMedia(ctx, mid, userID, "image/png", "p.png", 10, strings.NewReader("x"), false)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			boom := errors.New("boom")
			m.repo.EXPECT().GetAuthorID(mock.Anything, mid).Return(uuid.Nil, boom)

			// when
			err := tc.call(svc)

			// then
			require.ErrorIs(t, err, boom)
		})
	}
}

func TestListMysteries_AFailedBlockLookupIsSurfacedInsteadOfListingBlockedAuthors(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	boom := errors.New("boom")
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, boom)

	// when
	got, err := svc.ListMysteries(context.Background(), "new", nil, viewer, bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, boom)
	assert.Nil(t, got)
	m.repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
}

func TestListMysteries_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewer := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, spec.MysteryListFilter{Sort: "new", Limit: 10, Offset: 0}).Return(nil, 0, errors.New("boom"))

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
	rows := []model.MysteryRow{{ID: uuid.New(), Title: "T", Body: longBody.String()}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, spec.MysteryListFilter{Sort: "new", Limit: 5, Offset: 0}).Return(rows, 1, nil)

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
	rows := []model.MysteryRow{{ID: uuid.New(), Body: "short"}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewer).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, spec.MysteryListFilter{Sort: "new", Limit: 10, Offset: 0}).Return(rows, 1, nil)

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
	row := &model.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []model.MysteryAttemptRow{{ID: uuid.New(), UserID: other, Body: "guess"}}
	clues := []dto.MysteryClue{{ID: 1, Body: "c1"}, {ID: 2, Body: "c2", PlayerID: new(uuid.New())}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(clues, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, spec.MysteryAttemptQuery{MysteryID: id, ViewerID: author}).Return(attempts, nil)
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
	row := &model.MysteryRow{ID: id, UserID: author, Solved: false, FreeForAll: false}
	attempts := []model.MysteryAttemptRow{{ID: uuid.New(), UserID: other, Body: "x"}}
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(row, nil)
	m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, spec.MysterySolverQuery{MysteryID: id, UserID: admin}).Return(false, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().GetAttempts(mock.Anything, spec.MysteryAttemptQuery{MysteryID: id, ViewerID: admin}).Return(attempts, nil)
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
	_, err := svc.CreateMystery(context.Background(), uuid.New(), req, nil, nil)

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func TestCreateMystery_EmptyBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)
	req := validCreateReq()
	req.Body = "\n\t"

	// when
	_, err := svc.CreateMystery(context.Background(), uuid.New(), req, nil, nil)

	// then
	require.ErrorIs(t, err, ErrEmptyTitle)
}

func validCreateSpec(userID uuid.UUID, clues ...spec.NewClue) spec.NewMysteryWithClues {
	if clues == nil {
		clues = []spec.NewClue{}
	}

	return spec.NewMysteryWithClues{
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
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq(), nil, nil)

	// then
	require.Error(t, err)
}

func TestCreateMystery_NotifiesFollowersAndBumpsTheMysteriesSection(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(&model.MysteryRow{ID: uuid.New()}, nil)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyMysteries).Return().Once()

	// when
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq(), nil, nil)

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
	_, err := svc.CreateMystery(context.Background(), userID, validCreateReq(), nil, nil)

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
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(&model.MysteryRow{ID: uuid.New()}, nil)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyMysteries).Return().Once()

	// when
	_, err := svc.CreateMystery(context.Background(), userID, req, nil, nil)

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
	createSpec := validCreateSpec(userID,
		spec.NewClue{Body: "clue1", TruthType: "red", SortOrder: 0},
		spec.NewClue{Body: "clue2", TruthType: "blue", SortOrder: 2},
	)
	m.repo.EXPECT().CreateWithClues(mock.Anything, createSpec).Return(&model.MysteryRow{ID: uuid.New()}, nil)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyMysteries).Return().Once()

	// when
	id, err := svc.CreateMystery(context.Background(), userID, req, nil, nil)

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

	createSpec := validCreateSpec(userID)
	createSpec.Body = req.Body

	m.repo.EXPECT().CreateWithClues(mock.Anything, createSpec).Return(&model.MysteryRow{ID: mysteryID}, nil)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyMysteries).Return().Once()
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
	_, err := svc.CreateMystery(context.Background(), userID, req, nil, nil)

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, mysteryID, mentioned.ReferenceID)
	assert.Equal(t, "mystery", mentioned.ReferenceType)
	assert.Equal(t, "/mystery/"+mysteryID.String(), mentioned.EmailLink)
}

func TestCreateMystery_SavesMediaAndAttachmentsBeforeAnySideEffect(t *testing.T) {
	// given a mystery carrying two images, the second spoilered, and one attachment
	svc, m := newTestService(t)
	recorder := upload.NewMockMediaRecorder(t)
	svc.uploader = recorder
	userID := uuid.New()
	mysteryID := uuid.New()
	files := []upload.FileUpload{uploadtest.PNGUpload(t, "one.png", false), uploadtest.PNGUpload(t, "two.png", true)}
	notes := pdfUpload("notes.pdf")

	var order []string

	uploadtest.ExpectMediaLimits(m.settingsSvc, testMediaLimit)
	expectAttachmentLimit(m, testMediaLimit)
	m.repo.EXPECT().CreateWithClues(mock.Anything, validCreateSpec(userID)).Return(&model.MysteryRow{ID: mysteryID}, nil)
	expectStored(recorder, files[0], "/uploads/mysteries/one.png")
	expectStored(recorder, files[1], "/uploads/mysteries/two.png")
	m.repo.EXPECT().
		AddMedia(mock.Anything, spec.NewMedia{TargetID: mysteryID, MediaURL: "/uploads/mysteries/one.png", MediaType: media.MediaTypeImage, Filename: "one.png", SortOrder: 0}).
		Run(func(_ context.Context, _ spec.NewMedia, _ ...*sql.Tx) { order = append(order, "one.png") }).
		Return(1, nil).
		Once()
	m.repo.EXPECT().
		AddMedia(mock.Anything, spec.NewMedia{TargetID: mysteryID, MediaURL: "/uploads/mysteries/two.png", MediaType: media.MediaTypeImage, Filename: "two.png", SortOrder: 1, IsSpoiler: true}).
		Run(func(_ context.Context, _ spec.NewMedia, _ ...*sql.Tx) { order = append(order, "two.png") }).
		Return(2, nil).
		Once()
	m.uploadSvc.EXPECT().
		SaveAttachment(mock.Anything, "mystery-attachments/"+mysteryID.String(), notes.Size, int64(testMediaLimit), mock.Anything).
		Return("/uploads/mystery-attachments/notes.pdf", nil).
		Once()
	m.repo.EXPECT().
		AddAttachment(mock.Anything, spec.NewMysteryAttachment{MysteryID: mysteryID, FileURL: "/uploads/mystery-attachments/notes.pdf", FileName: "notes.pdf", FileSize: int(notes.Size)}).
		Run(func(_ context.Context, _ spec.NewMysteryAttachment, _ ...*sql.Tx) { order = append(order, "notes.pdf") }).
		Return(7, nil).
		Once()
	m.activity.EXPECT().
		BumpSidebarActivity(sidebar.KeyMysteries).
		Run(func(_ sidebar.Key) { order = append(order, "bump") }).
		Return().
		Once()

	// when
	id, err := svc.CreateMystery(context.Background(), userID, validCreateReq(), files, []upload.FileUpload{notes})

	// then every file is recorded against the mystery in order, and only then is the section bumped
	require.NoError(t, err)
	assert.Equal(t, mysteryID, id)
	assert.Equal(t, []string{"one.png", "two.png", "notes.pdf", "bump"}, order)
}

func TestCreateMystery_AFailedSaveRollsTheMysteryBackAndFiresNothing(t *testing.T) {
	rollbackErr := errors.New("db down")
	rowErr := errors.New("attachment row refused")

	cases := []struct {
		name          string
		expect        func(m *testMocks, recorder *upload.MockMediaRecorder, mysteryID uuid.UUID, cancel context.CancelFunc)
		rollbackPaths []string
		rollbackErr   error
		wantErr       error
	}{
		{
			name: "a media failure deletes the mystery",
			expect: func(_ *testMocks, recorder *upload.MockMediaRecorder, _ uuid.UUID, _ context.CancelFunc) {
				recorder.EXPECT().
					SaveAndRecord(mock.Anything, "mysteries", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errStoreFailed).
					Once()
			},
			wantErr: errStoreFailed,
		},
		{
			name: "an attachment store failure deletes the mystery and its stored media",
			expect: func(m *testMocks, recorder *upload.MockMediaRecorder, mysteryID uuid.UUID, _ context.CancelFunc) {
				recorder.EXPECT().
					SaveAndRecord(mock.Anything, "mysteries", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(storeAs("/uploads/mysteries/one.png")).
					Once()
				m.repo.EXPECT().AddMedia(mock.Anything, mock.Anything).Return(1, nil).Once()
				m.uploadSvc.EXPECT().SaveAttachment(mock.Anything, "mystery-attachments/"+mysteryID.String(), mock.Anything, mock.Anything, mock.Anything).Return("", errStoreFailed).Once()
			},
			rollbackPaths: []string{"/uploads/mysteries/one.png"},
			wantErr:       errStoreFailed,
		},
		{
			name: "a refused attachment row unlinks the stored attachment and deletes the mystery",
			expect: func(m *testMocks, recorder *upload.MockMediaRecorder, _ uuid.UUID, _ context.CancelFunc) {
				recorder.EXPECT().
					SaveAndRecord(mock.Anything, "mysteries", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(storeAs("/uploads/mysteries/one.png")).
					Once()
				m.repo.EXPECT().AddMedia(mock.Anything, mock.Anything).Return(1, nil).Once()
				m.uploadSvc.EXPECT().SaveAttachment(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("/uploads/mystery-attachments/notes.pdf", nil).Once()
				m.repo.EXPECT().AddAttachment(mock.Anything, mock.Anything).Return(0, rowErr).Once()
				m.uploadSvc.EXPECT().Delete([]string{"/uploads/mystery-attachments/notes.pdf"}).Return().Once()
			},
			rollbackPaths: []string{"/uploads/mysteries/one.png"},
			wantErr:       rowErr,
		},
		{
			name: "a failed rollback is reported alongside the cause",
			expect: func(_ *testMocks, recorder *upload.MockMediaRecorder, _ uuid.UUID, _ context.CancelFunc) {
				recorder.EXPECT().
					SaveAndRecord(mock.Anything, "mysteries", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errStoreFailed).
					Once()
			},
			rollbackErr: rollbackErr,
			wantErr:     errStoreFailed,
		},
		{
			name: "a client hanging up mid-upload still rolls the mystery back",
			expect: func(_ *testMocks, recorder *upload.MockMediaRecorder, _ uuid.UUID, cancel context.CancelFunc) {
				recorder.EXPECT().
					SaveAndRecord(mock.Anything, "mysteries", "image/png", "one.png", mock.Anything, mock.Anything, false, mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(context.Context, string, string, string, int64, io.Reader, bool, media.AddFn, media.UpdateURLFn, media.UpdateURLFn) (*dto.PostMediaResponse, error) {
						cancel()

						return nil, errStoreFailed
					}).
					Once()
			},
			wantErr: errStoreFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a mystery that mentions someone, carrying one image and one attachment
			svc, m := newTestService(t)
			recorder := upload.NewMockMediaRecorder(t)
			svc.uploader = recorder
			userID := uuid.New()
			mysteryID := uuid.New()
			req := validCreateReq()
			req.Body = "solve this with @alice"

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			uploadtest.ExpectMediaLimits(m.settingsSvc, testMediaLimit)
			expectAttachmentLimit(m, testMediaLimit)
			m.repo.EXPECT().CreateWithClues(mock.Anything, mock.Anything).Return(&model.MysteryRow{ID: mysteryID}, nil)
			tc.expect(m, recorder, mysteryID, cancel)

			var rollbackCtxErr error
			m.repo.EXPECT().
				DeleteWithMedia(mock.Anything, spec.OwnedDeletion{ID: mysteryID, UserID: userID}).
				Run(func(ctx context.Context, _ spec.OwnedDeletion, _ ...*sql.Tx) { rollbackCtxErr = ctx.Err() }).
				Return(tc.rollbackPaths, tc.rollbackErr).
				Once()
			expectUnlinked(m, tc.rollbackPaths)

			// when
			id, err := svc.CreateMystery(ctx, userID, req, []upload.FileUpload{uploadtest.PNGUpload(t, "one.png", false)}, []upload.FileUpload{pdfUpload("notes.pdf")})

			// then the mystery is gone, and the strict mocks prove no mention, follower notice or section bump fired
			require.ErrorIs(t, err, tc.wantErr)
			if tc.rollbackErr != nil {
				require.ErrorIs(t, err, tc.rollbackErr)
			}
			assert.Equal(t, uuid.Nil, id)
			require.NoError(t, rollbackCtxErr)
			assert.Empty(t, m.fanout)
		})
	}
}

func TestCreateMystery_ARefusedFileWritesNothing(t *testing.T) {
	cases := []struct {
		name          string
		files         []upload.FileUpload
		attachments   []upload.FileUpload
		maxImageSize  int
		maxAttachment int
		wantErr       error
	}{
		{
			name:          "an image over the size limit",
			files:         []upload.FileUpload{uploadtest.PNGUpload(t, "big.png", false)},
			maxImageSize:  1,
			maxAttachment: testMediaLimit,
			wantErr:       upload.ErrFileTooLarge,
		},
		{
			name:          "an image that is not what it claims to be",
			files:         []upload.FileUpload{uploadtest.MemoryUpload("image/png", "fake.png", []byte("not an image at all"), false)},
			maxImageSize:  testMediaLimit,
			maxAttachment: testMediaLimit,
			wantErr:       upload.ErrInvalidFileType,
		},
		{
			name:          "an attachment over the size limit",
			attachments:   []upload.FileUpload{pdfUpload("notes.pdf")},
			maxImageSize:  testMediaLimit,
			maxAttachment: 1,
			wantErr:       upload.ErrFileTooLarge,
		},
		{
			name:          "an attachment of a refused type",
			attachments:   []upload.FileUpload{uploadtest.MemoryUpload("text/html", "page.html", []byte("<html><body>trap</body></html>"), false)},
			maxImageSize:  testMediaLimit,
			maxAttachment: testMediaLimit,
			wantErr:       upload.ErrInvalidAttachmentType,
		},
		{
			name:          "two attachments sharing a name",
			attachments:   []upload.FileUpload{pdfUpload("notes.pdf"), pdfUpload("notes.pdf")},
			maxImageSize:  testMediaLimit,
			maxAttachment: testMediaLimit,
			wantErr:       ErrDuplicateAttachment,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given strict repo, recorder and upload mocks that expect no write at all
			svc, m := newTestService(t)
			svc.uploader = upload.NewMockMediaRecorder(t)
			uploadtest.ExpectMediaLimits(m.settingsSvc, tc.maxImageSize)
			expectAttachmentLimit(m, tc.maxAttachment)

			// when
			id, err := svc.CreateMystery(context.Background(), uuid.New(), validCreateReq(), tc.files, tc.attachments)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, uuid.Nil, id)
			assert.Empty(t, m.fanout)
		})
	}
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
	require.ErrorIs(t, err, ErrNotAuthor)
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

func validUpdateSpec(id uuid.UUID, knox dto.KnoxContract, clues ...spec.NewClue) spec.MysteryUpdateWithClues {
	if clues == nil {
		clues = []spec.NewClue{}
	}

	return spec.MysteryUpdateWithClues{
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
			m.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.MysteryRow{
				ID: id, UserID: userID, Knox: tt.stored, KnoxPublished: tt.published, AttemptCount: tt.attemptCount,
			}, nil)

			m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil).Maybe()

			var written dto.KnoxContract
			if tt.wantErr == nil {
				m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, tt.wantWritten)).
					Run(func(_ context.Context, updateSpec spec.MysteryUpdateWithClues, _ ...*sql.Tx) {
						written = updateSpec.Knox
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

func TestUpdateMystery_AFailedReadIsSurfacedNotReportedAsNotFound(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name     string
		rowErr   error
		cluesErr error
	}{
		{name: "the mystery", rowErr: boom},
		{name: "the clues the audit compares against", cluesErr: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing to load", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
			m.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.MysteryRow{ID: id, UserID: uuid.New()}, tc.rowErr)
			if tc.rowErr == nil {
				m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, tc.cluesErr)
			}

			// when
			err := svc.UpdateMystery(context.Background(), id, userID, validCreateReq())

			// then
			require.ErrorIs(t, err, boom)
			m.repo.AssertNotCalled(t, "UpdateWithClues", mock.Anything, mock.Anything)
		})
	}
}

func TestGetMystery_AFailedReadIsSurfacedInsteadOfRenderingAnEmptySection(t *testing.T) {
	steps := []string{"clues", "attempts", "viewer role", "comments", "attachments", "media", "viewer solved"}

	for _, failAt := range steps {
		t.Run("the "+failAt+" read failing", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			viewer := uuid.New()
			boom := errors.New("boom")
			errAt := func(step string) error {
				if step == failAt {
					return boom
				}

				return nil
			}
			m.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.MysteryRow{ID: id, UserID: uuid.New(), Solved: true}, nil)
			m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, errAt("clues")).Maybe()
			m.repo.EXPECT().GetAttempts(mock.Anything, mock.Anything).Return(nil, errAt("attempts")).Maybe()
			m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", errAt("viewer role")).Maybe()
			m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindMystery, ID: id}, viewer).Return(nil, errAt("comments")).Maybe()
			m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, errAt("attachments")).Maybe()
			m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, errAt("media")).Maybe()
			m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mock.Anything).Return(false, errAt("viewer solved")).Maybe()

			// when
			got, err := svc.GetMystery(context.Background(), id, viewer)

			// then
			require.ErrorIs(t, err, boom)
			assert.Nil(t, got)
		})
	}
}

func TestUpdateMystery_UpdateWithCluesError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	old := &model.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
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
	old := &model.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
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
	old := &model.MysteryRow{ID: id, UserID: author, Title: "Old Title", Body: "Body", Difficulty: "medium"}
	m.authz.EXPECT().Can(mock.Anything, admin, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, old.Knox)).Return(nil)
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    admin,
		Action:     audit.ActionMysteryUpdateAdmin,
		TargetType: audit.TargetMystery,
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
	old := &model.MysteryRow{ID: id, UserID: userID, Title: "Title", Body: "Body", Difficulty: "medium"}
	req := validCreateReq()
	req.Clues = []dto.CreateClueRequest{{Body: "new1"}, {Body: "  "}, {Body: "new2", TruthType: "blue"}}
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
	m.repo.EXPECT().GetByID(mock.Anything, id).Return(old, nil)
	m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
	m.repo.EXPECT().UpdateWithClues(mock.Anything, validUpdateSpec(id, old.Knox,
		spec.NewClue{Body: "new1", TruthType: "red", SortOrder: 0},
		spec.NewClue{Body: "new2", TruthType: "blue", SortOrder: 2},
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
	m.repo.EXPECT().DeleteWithFiles(mock.Anything, spec.MysteryDelete{ID: id, UserID: userID, AsAdmin: true}).Return([]string{"/uploads/mystery/a.png"}, nil)
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
	m.repo.EXPECT().DeleteWithFiles(mock.Anything, spec.MysteryDelete{ID: id, UserID: userID}).Return([]string{"/uploads/mystery/a.png", "/uploads/mystery/a_thumb.png"}, nil)
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
	m.repo.EXPECT().DeleteWithFiles(mock.Anything, spec.MysteryDelete{ID: id, UserID: userID}).Return(nil, errors.New("boom"))

	// when
	err := svc.DeleteMystery(context.Background(), id, userID)

	// then
	require.Error(t, err)
}

func TestGetMystery_CommentsShowOnlyOnceSolved(t *testing.T) {
	cases := []struct {
		name   string
		solved bool
	}{
		{name: "a solved mystery carries its comment thread", solved: true},
		{name: "an unsolved mystery shows no comments and never loads them"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id, author, viewer := uuid.New(), uuid.New(), uuid.New()
			thread := []dto.CommentResponse{{ID: uuid.New(), Body: "the butler did it"}}
			m.repo.EXPECT().GetByID(mock.Anything, id).Return(&model.MysteryRow{ID: id, UserID: author, Solved: tc.solved}, nil)
			m.repo.EXPECT().UserHasWinningAttempt(mock.Anything, mock.Anything).Return(false, nil).Maybe()
			m.repo.EXPECT().GetClues(mock.Anything, id).Return(nil, nil)
			m.repo.EXPECT().GetAttempts(mock.Anything, mock.Anything).Return(nil, nil)
			m.authz.EXPECT().GetRole(mock.Anything, viewer).Return("", nil).Maybe()
			m.repo.EXPECT().GetAttachments(mock.Anything, id).Return(nil, nil)
			m.repo.EXPECT().GetMedia(mock.Anything, id).Return(nil, nil).Maybe()
			if tc.solved {
				m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindMystery, ID: id}, viewer).Return(thread, nil).Once()
			}

			// when
			got, err := svc.GetMystery(context.Background(), id, viewer)

			// then
			require.NoError(t, err)
			if tc.solved {
				assert.Equal(t, thread, got.Comments)
				return
			}

			assert.NotNil(t, got.Comments, "an unsolved mystery renders an empty list, not null")
			assert.Empty(t, got.Comments)
		})
	}
}

func TestCreateComment(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name      string
		solved    bool
		solvedErr error
		createErr error
		want      error
	}{
		{name: "a missing mystery is not found", solvedErr: errMissingRow, want: ErrNotFound},
		{name: "a failed solved check is surfaced, not reported as not found", solvedErr: boom, want: boom},
		{name: "an unsolved mystery refuses comments", want: ErrNotSolved},
		{name: "a solved mystery hands the comment to the comment service", solved: true},
		{name: "a refused comment bubbles", solved: true, createErr: boom, want: boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			mid, userID, commentID := uuid.New(), uuid.New(), uuid.New()
			req := dto.CreateCommentRequest{Body: "hi"}
			m.repo.EXPECT().IsSolved(mock.Anything, mid).Return(tc.solved, tc.solvedErr)

			wantID := uuid.Nil
			if tc.solved {
				returned := commentID
				if tc.createErr != nil {
					returned = uuid.Nil
				}
				m.commentSvc.EXPECT().Create(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindMystery, ID: mid}, userID, req, []upload.FileUpload(nil)).Return(returned, tc.createErr).Once()
				wantID = returned
			}

			// when
			id, err := svc.CreateComment(context.Background(), mid, userID, req, nil)

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, wantID, id)
		})
	}
}
