package ship

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/quotefinder"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	testMocks struct {
		shipRepo     *repository.MockShipRepository
		shipComments *dao.MockCommentDAO[uuid.UUID]
		userRepo     *repository.MockUserRepository
		auditRepo    *repository.MockAuditLogRepository
		authz        *authz.MockService
		blockSvc     *block.MockService
		notifSvc     *notification.MockService
		uploadSvc    *upload.MockService
		settingsSvc  *settings.MockService
	}
)

const (
	savedImageURL = "/uploads/ships/x.png"
)

var (
	errNoRow    = errors.Join(errors.New("no row"), dao.ErrNotFound)
	errBoom     = errors.New("boom")
	errDiskFull = errors.New("disk full")
	errNotOwner = errors.New("not owner")
	errStop     = errors.New("stop goroutine")
)

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		shipRepo:     repository.NewMockShipRepository(t),
		shipComments: dao.NewMockCommentDAO[uuid.UUID](t),
		userRepo:     repository.NewMockUserRepository(t),
		auditRepo:    repository.NewMockAuditLogRepository(t),
		authz:        authz.NewMockService(t),
		blockSvc:     block.NewMockService(t),
		notifSvc:     notification.NewMockService(t),
		uploadSvc:    upload.NewMockService(t),
		settingsSvc:  settings.NewMockService(t),
	}
	m.uploadSvc.EXPECT().FullDiskPath(mock.Anything).Return("/tmp/does-not-exist-xyz.png").Maybe()

	mentionSvc := mention.NewService(m.userRepo, m.blockSvc, m.notifSvc, dao.CommentDAOs{
		ByID: map[string]dao.CommentDAO[uuid.UUID]{string(mention.KindShipComment): m.shipComments},
	})
	svc := NewService(m.shipRepo, m.userRepo, m.auditRepo, m.authz, m.blockSvc, m.notifSvc, mentionSvc, m.uploadSvc, media.NewProcessor(1), m.settingsSvc, quotefinder.NewClient(), contentfilter.New(), nil).(*service)

	return svc, m
}

func validCharacters() []dto.ShipCharacter {
	return []dto.ShipCharacter{
		{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
		{Series: "umineko", CharacterID: "beatrice", CharacterName: "Beatrice"},
	}
}

func expectImageSaved(m *testMocks, err error) {
	url := savedImageURL
	if err != nil {
		url = ""
	}

	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "ships", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return(url, err)
}

func expectMentionOfAlice(m *testMocks, actorID uuid.UUID, notifications int) (uuid.UUID, func() dto.NotifyParams) {
	mentionedID := uuid.New()
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "Battler"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, mentionedID).Return(false, nil)

	var wg sync.WaitGroup
	wg.Add(notifications)

	var mentioned dto.NotifyParams
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			if p.Type == dto.NotifMention {
				mentioned = p
			}
			wg.Done()

			return nil
		})

	awaitMention := func() dto.NotifyParams {
		wg.Wait()

		return mentioned
	}

	return mentionedID, awaitMention
}

func TestCreateShipAndUpdateShip_RejectInvalidInput(t *testing.T) {
	cases := []struct {
		name       string
		title      string
		characters []dto.ShipCharacter
		want       error
	}{
		{name: "blank title", title: "   ", characters: validCharacters(), want: ErrEmptyTitle},
		{name: "one character", title: "My Ship", characters: validCharacters()[:1], want: ErrTooFewCharacters},
		{
			name:  "same character twice",
			title: "My Ship",
			characters: []dto.ShipCharacter{
				{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
				{Series: "umineko", CharacterID: "battler", CharacterName: "Battler"},
			},
			want: ErrDuplicateCharacters,
		},
		{
			name:  "same character in a different case",
			title: "My Ship",
			characters: []dto.ShipCharacter{
				{Series: "Umineko", CharacterID: "Battler", CharacterName: "Battler"},
				{Series: "umineko", CharacterID: "battler", CharacterName: "battler"},
			},
			want: ErrDuplicateCharacters,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newTestService(t)

			// when
			_, createErr := svc.CreateShip(context.Background(), uuid.New(), dto.CreateShipRequest{Title: tc.title, Characters: tc.characters})
			updateErr := svc.UpdateShip(context.Background(), uuid.New(), uuid.New(), dto.UpdateShipRequest{Title: tc.title, Characters: tc.characters})

			// then
			require.ErrorIs(t, createErr, tc.want)
			require.ErrorIs(t, updateErr, tc.want)
		})
	}
}

func TestCreateShip_RepoErrorBubbles(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	req := dto.CreateShipRequest{Title: "Ship", Description: "desc", Characters: validCharacters()}
	m.shipRepo.EXPECT().
		CreateWithCharacters(mock.Anything, spec.NewShipWithCharacters{
			UserID:      userID,
			Title:       "Ship",
			Description: "desc",
			Characters:  req.Characters,
		}).
		Return(nil, errBoom)

	// when
	_, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.ErrorIs(t, err, errBoom)
}

func TestCreateShip_MentionOnTheDescriptionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID, shipID := uuid.New(), uuid.New()
	req := dto.CreateShipRequest{Title: "  Ship  ", Description: "  sailed for @alice  ", Characters: validCharacters()}
	m.shipRepo.EXPECT().
		CreateWithCharacters(mock.Anything, spec.NewShipWithCharacters{
			UserID:      userID,
			Title:       "Ship",
			Description: "sailed for @alice",
			Characters:  req.Characters,
		}).
		Return(&model.ShipRow{ID: shipID}, nil)
	mentionedID, awaitMention := expectMentionOfAlice(m, userID, 1)

	// when
	id, err := svc.CreateShip(context.Background(), userID, req)

	// then
	require.NoError(t, err)
	assert.Equal(t, shipID, id)
	mentioned := awaitMention()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, shipID, mentioned.ReferenceID)
	assert.Equal(t, "ship", mentioned.ReferenceType)
	assert.Equal(t, "/ships/"+shipID.String(), mentioned.EmailLink)
}

func TestGetShip_LookupFailure(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
		want    error
	}{
		{name: "repo error bubbles", repoErr: errBoom, want: errBoom},
		{name: "missing row is not found", want: ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, viewerID := uuid.New(), uuid.New()
			m.shipRepo.EXPECT().GetByID(mock.Anything, spec.ShipLookup{ID: shipID, ViewerID: viewerID}).Return(nil, tc.repoErr)

			// when
			got, err := svc.GetShip(context.Background(), shipID, viewerID)

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, got)
		})
	}
}

func TestGetShip_SurfacesAFailedLookup(t *testing.T) {
	cases := []struct {
		name   string
		failAt int
	}{
		{name: "the characters", failAt: 0},
		{name: "the viewer's block list", failAt: 1},
		{name: "the comments", failAt: 2},
		{name: "the comment media", failAt: 3},
		{name: "the block check against the author", failAt: 4},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as empty", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, authorID, viewerID := uuid.New(), uuid.New(), uuid.New()
			m.shipRepo.EXPECT().GetByID(mock.Anything, spec.ShipLookup{ID: shipID, ViewerID: viewerID}).Return(&model.ShipRow{ID: shipID, UserID: authorID}, nil)

			m.shipRepo.EXPECT().GetCharacters(mock.Anything, shipID).Return(nil, shipFailAt(tc.failAt, 0))
			if tc.failAt >= 1 {
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, shipFailAt(tc.failAt, 1))
			}
			if tc.failAt >= 2 {
				m.shipRepo.EXPECT().GetComments(mock.Anything, mock.Anything).Return(nil, 0, shipFailAt(tc.failAt, 2))
			}
			if tc.failAt >= 3 {
				m.shipRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, shipFailAt(tc.failAt, 3))
			}
			if tc.failAt >= 4 {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(false, shipFailAt(tc.failAt, 4))
			}

			// when
			got, err := svc.GetShip(context.Background(), shipID, viewerID)

			// then
			require.ErrorIs(t, err, errBoom)
			assert.Nil(t, got)
		})
	}
}

func TestListShips_AFailedBlockListIsSurfacedBeforeAnythingIsListed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, errBoom)

	// when
	got, err := svc.ListShips(context.Background(), viewerID, "", false, "", "", bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, errBoom, "listing without the block list would show blocked users' ships")
	assert.Nil(t, got)
}

func TestListShipsByUser_AFailedCharacterLookupIsSurfaced(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID, viewerID := uuid.New(), uuid.New()
	m.shipRepo.EXPECT().ListByUser(mock.Anything, mock.Anything).Return([]model.ShipRow{{ID: uuid.New()}}, 1, nil)
	m.shipRepo.EXPECT().GetCharactersBatch(mock.Anything, mock.Anything).Return(nil, errBoom)

	// when
	got, err := svc.ListShipsByUser(context.Background(), userID, viewerID, bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, errBoom)
	assert.Nil(t, got)
}

func shipFailAt(failAt int, step int) error {
	if failAt == step {
		return errBoom
	}

	return nil
}

func TestGetShip_ViewerBlocked(t *testing.T) {
	cases := []struct {
		name        string
		viewerID    uuid.UUID
		blockedIDs  []uuid.UUID
		wantBlocked bool
	}{
		{name: "signed-in viewer blocked either way", viewerID: uuid.New(), blockedIDs: []uuid.UUID{uuid.New()}, wantBlocked: true},
		{name: "anonymous viewer skips the block check", viewerID: uuid.Nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, authorID := uuid.New(), uuid.New()
			row := &model.ShipRow{ID: shipID, UserID: authorID, Title: "T"}
			m.shipRepo.EXPECT().GetByID(mock.Anything, spec.ShipLookup{ID: shipID, ViewerID: tc.viewerID}).Return(row, nil)
			m.shipRepo.EXPECT().GetCharacters(mock.Anything, shipID).Return(nil, nil)
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, tc.viewerID).Return(tc.blockedIDs, nil)
			m.shipRepo.EXPECT().GetComments(mock.Anything, spec.CommentQuery[uuid.UUID]{
				TargetID:       shipID,
				ViewerID:       tc.viewerID,
				Limit:          500,
				Offset:         0,
				ExcludeUserIDs: tc.blockedIDs,
			}).Return(nil, 0, nil)
			m.shipRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
			if tc.viewerID != uuid.Nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, tc.viewerID, authorID).Return(tc.wantBlocked, nil)
			}

			// when
			got, err := svc.GetShip(context.Background(), shipID, tc.viewerID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantBlocked, got.ViewerBlocked)
			assert.Equal(t, shipID, got.ID)
		})
	}
}

func TestUpdateShip(t *testing.T) {
	cases := []struct {
		name       string
		stranger   bool
		lookupErr  error
		canEditAny bool
		updateErr  error
		want       error
	}{
		{name: "missing ship is not found", lookupErr: errNoRow, want: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", lookupErr: errBoom, want: errBoom},
		{name: "admin editing another user's ship writes an admin audit row", stranger: true, canEditAny: true},
		{name: "moderator editing own ship writes no admin row", canEditAny: true},
		{name: "owner update refused by the repo bubbles", updateErr: errNotOwner, want: errNotOwner},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, userID := uuid.New(), uuid.New()
			authorID := userID
			if tc.stranger {
				authorID = uuid.New()
			}
			req := dto.UpdateShipRequest{Title: " T ", Description: " d ", Characters: validCharacters()}
			m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil {
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(tc.canEditAny)
				m.shipRepo.EXPECT().
					UpdateWithCharacters(mock.Anything, spec.ShipUpdate{
						ID:          shipID,
						UserID:      userID,
						Title:       "T",
						Description: "d",
						AsAdmin:     tc.canEditAny,
						Characters:  req.Characters,
					}).
					Return(tc.updateErr)
			}
			if tc.stranger {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    userID,
					Action:     audit.ActionShipUpdateAdmin,
					TargetType: audit.TargetShip,
					TargetID:   shipID.String(),
					Details:    "title=T",
					SubjectID:  authorID,
				}).Return(nil)
			}

			// when
			err := svc.UpdateShip(context.Background(), shipID, userID, req)

			// then
			require.ErrorIs(t, err, tc.want)
			if !tc.stranger {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteShip_AuditsTheDeletionAndRemovesItsFiles(t *testing.T) {
	cases := []struct {
		name         string
		stranger     bool
		canDeleteAny bool
		wantAction   audit.Action
	}{
		{name: "admin deleting another user's ship records an admin action", stranger: true, canDeleteAny: true, wantAction: audit.ActionShipDeleteAdmin},
		{name: "moderator deleting own ship records owner action", canDeleteAny: true, wantAction: audit.ActionShipDelete},
		{name: "owner deleting own ship records owner action", wantAction: audit.ActionShipDelete},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, userID := uuid.New(), uuid.New()
			authorID := userID
			if tc.stranger {
				authorID = uuid.New()
			}
			paths := []string{"/uploads/ships/x.png", "/uploads/ships/x-thumb.png"}
			m.shipRepo.EXPECT().
				GetByID(mock.Anything, spec.ShipLookup{ID: shipID, ViewerID: userID}).
				Return(&model.ShipRow{ID: shipID, UserID: authorID, Title: "Doomed", VoteScore: 7, CommentCount: 3}, nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(tc.canDeleteAny)
			m.shipRepo.EXPECT().
				DeleteShip(mock.Anything, spec.ShipDeletion{ID: shipID, UserID: userID, AsAdmin: tc.canDeleteAny}).
				Return(paths, nil)
			m.uploadSvc.EXPECT().Delete(paths).Return()
			m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
				ActorID:    userID,
				Action:     tc.wantAction,
				TargetType: audit.TargetShip,
				TargetID:   shipID.String(),
				Details:    "title=Doomed vote_score=7 comments=3",
				SubjectID:  authorID,
			}).Return(nil)

			// when
			err := svc.DeleteShip(context.Background(), shipID, userID)

			// then
			require.NoError(t, err)
		})
	}
}

func TestDeleteShip_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID, userID := uuid.New(), uuid.New()
	m.shipRepo.EXPECT().GetByID(mock.Anything, spec.ShipLookup{ID: shipID, ViewerID: userID}).Return(nil, nil)

	// when
	err := svc.DeleteShip(context.Background(), shipID, userID)

	// then
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteShip_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID, userID := uuid.New(), uuid.New()
	row := &model.ShipRow{ID: shipID, UserID: userID, Title: "Owned"}
	m.shipRepo.EXPECT().GetByID(mock.Anything, spec.ShipLookup{ID: shipID, ViewerID: userID}).Return(row, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
	m.shipRepo.EXPECT().
		DeleteShip(mock.Anything, spec.ShipDeletion{ID: shipID, UserID: userID}).
		Return(nil, errBoom)

	// when
	err := svc.DeleteShip(context.Background(), shipID, userID)

	// then
	require.ErrorIs(t, err, errBoom)
	m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestListShips_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	m.shipRepo.EXPECT().
		List(mock.Anything, spec.ShipListing{
			ViewerID:       viewerID,
			Sort:           "new",
			CrackshipsOnly: false,
			Series:         "umineko",
			CharacterID:    "battler",
			Limit:          20,
			Offset:         0,
			ExcludeUserIDs: nil,
		}).
		Return(nil, 0, errBoom)

	// when
	_, err := svc.ListShips(context.Background(), viewerID, "new", false, "umineko", "battler", bounds.NewPage(20, 0))

	// then
	require.ErrorIs(t, err, errBoom)
}

func TestListShips_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID, shipID := uuid.New(), uuid.New()
	blocked := []uuid.UUID{uuid.New()}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(blocked, nil)
	m.shipRepo.EXPECT().
		List(mock.Anything, spec.ShipListing{
			ViewerID:       viewerID,
			Sort:           "top",
			CrackshipsOnly: true,
			Series:         "",
			CharacterID:    "",
			Limit:          10,
			Offset:         5,
			ExcludeUserIDs: blocked,
		}).
		Return([]model.ShipRow{{ID: shipID, UserID: uuid.New(), Title: "A"}}, 1, nil)
	m.shipRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{shipID}).Return(nil, nil)

	// when
	got, err := svc.ListShips(context.Background(), viewerID, "top", true, "", "", bounds.NewPage(10, 5))

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 5, got.Offset)
	assert.Len(t, got.Ships, 1)
}

func TestListShipsByUser(t *testing.T) {
	cases := []struct {
		name    string
		listErr error
	}{
		{name: "repo error bubbles", listErr: errBoom},
		{name: "builds the page from the rows"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID, viewerID, shipID := uuid.New(), uuid.New(), uuid.New()
			m.shipRepo.EXPECT().
				ListByUser(mock.Anything, spec.ShipUserListing{
					UserID:   userID,
					ViewerID: viewerID,
					Limit:    10,
					Offset:   0,
				}).
				Return([]model.ShipRow{{ID: shipID, UserID: userID, Title: "A"}}, 1, tc.listErr)
			if tc.listErr == nil {
				m.shipRepo.EXPECT().GetCharactersBatch(mock.Anything, []uuid.UUID{shipID}).Return(nil, nil)
			}

			// when
			got, err := svc.ListShipsByUser(context.Background(), userID, viewerID, bounds.NewPage(10, 0))

			// then
			require.ErrorIs(t, err, tc.listErr)
			if tc.listErr == nil {
				assert.Equal(t, 1, got.Total)
				assert.Len(t, got.Ships, 1)
			}
		})
	}
}

func TestUploadShipImage_NotAuthorRejected(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID, userID, authorID := uuid.New(), uuid.New(), uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)

	// when
	_, err := svc.UploadShipImage(context.Background(), shipID, userID, "image/png", 100, bytes.NewReader(nil))

	// then
	require.ErrorContains(t, err, "not the ship author")
}

func TestUploadShipImage(t *testing.T) {
	cases := []struct {
		name      string
		cancelled bool
		lookupErr error
		saveErr   error
		updateErr error
		want      error
		wantURL   string
	}{
		{name: "missing ship is not found", lookupErr: errNoRow, want: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", lookupErr: errBoom, want: errBoom},
		{name: "upload error bubbles", saveErr: errDiskFull, want: errDiskFull},
		{name: "image update error bubbles", updateErr: errBoom, want: errBoom},
		{name: "cancelled request still returns the saved URL", cancelled: true, wantURL: savedImageURL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, userID := uuid.New(), uuid.New()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelled {
				cancel()
			}
			m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(userID, tc.lookupErr)
			if tc.lookupErr == nil {
				expectImageSaved(m, tc.saveErr)
			}
			if tc.lookupErr == nil && tc.saveErr == nil {
				m.shipRepo.EXPECT().
					UpdateImage(mock.Anything, spec.ShipImageUpdate{
						ID:           shipID,
						ImageURL:     savedImageURL,
						ThumbnailURL: "",
					}).
					Return(tc.updateErr)
			}

			// when
			url, err := svc.UploadShipImage(ctx, shipID, userID, "image/png", 100, bytes.NewReader(nil))

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, tc.wantURL, url)
		})
	}
}

func TestVote(t *testing.T) {
	cases := []struct {
		name      string
		lookupErr error
		blocked   bool
		blockErr  error
		value     int
		voteErr   error
		want      error
	}{
		{name: "missing ship is not found", lookupErr: errNoRow, value: 1, want: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", lookupErr: errBoom, value: 1, want: errBoom},
		{name: "a block either way rejects the vote", blocked: true, value: 1, want: block.ErrUserBlocked},
		{name: "a failed block lookup rejects the vote", blockErr: errBoom, value: 1, want: errBoom},
		{name: "upvote is recorded", value: 1},
		{name: "downvote repo error bubbles", value: -1, voteErr: errBoom, want: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, userID, authorID := uuid.New(), uuid.New(), uuid.New()
			m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}
			if tc.lookupErr == nil && !tc.blocked && tc.blockErr == nil {
				m.shipRepo.EXPECT().Vote(mock.Anything, spec.Vote{UserID: userID, TargetID: shipID, Value: tc.value}).Return(tc.voteErr)
			}

			// when
			err := svc.Vote(context.Background(), userID, shipID, tc.value)

			// then
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCreateCommentAndUpdateComment_RejectBlankBody(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, createErr := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "   "})
	updateErr := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: "  "})

	// then
	require.ErrorIs(t, createErr, ErrEmptyBody)
	require.ErrorIs(t, updateErr, ErrEmptyBody)
}

func TestCreateComment_Errors(t *testing.T) {
	cases := []struct {
		name      string
		lookupErr error
		blocked   bool
		blockErr  error
		createErr error
		want      error
	}{
		{name: "missing ship is not found", lookupErr: errNoRow, want: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", lookupErr: errBoom, want: errBoom},
		{name: "a block either way rejects the comment", blocked: true, want: block.ErrUserBlocked},
		{name: "a failed block lookup rejects the comment", blockErr: errBoom, want: errBoom},
		{name: "repo error bubbles after trimming the body", createErr: errBoom, want: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			shipID, userID, authorID := uuid.New(), uuid.New(), uuid.New()
			m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}
			if tc.lookupErr == nil && !tc.blocked && tc.blockErr == nil {
				m.shipComments.EXPECT().
					CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{
						TargetID: shipID,
						ParentID: nil,
						UserID:   userID,
						Body:     "hi",
					}).
					Return(nil, tc.createErr)
			}

			// when
			id, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "  hi  "})

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, uuid.Nil, id)
		})
	}
}

func TestCreateComment_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID, userID, authorID, commentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{
			TargetID: shipID,
			ParentID: nil,
			UserID:   userID,
			Body:     "hi",
		}).
		Return(&model.CommentRow{ID: commentID}, nil)

	var actorLoaded sync.WaitGroup
	actorLoaded.Add(1)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).
		RunAndReturn(func(context.Context, uuid.UUID, ...*sql.Tx) (*model.User, error) {
			actorLoaded.Done()

			return nil, errStop
		})

	// when
	id, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "hi"})

	// then
	require.NoError(t, err)
	assert.Equal(t, commentID, id)
	actorLoaded.Wait()
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	shipID, userID, authorID, commentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	m.shipRepo.EXPECT().GetAuthorID(mock.Anything, shipID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.shipComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{
			TargetID: shipID,
			ParentID: nil,
			UserID:   userID,
			Body:     "look at this @alice",
		}).
		Return(&model.CommentRow{ID: commentID}, nil)
	mentionedID, awaitMention := expectMentionOfAlice(m, userID, 2)

	// when
	_, err := svc.CreateComment(context.Background(), shipID, userID, dto.CreateCommentRequest{Body: "look at this @alice"})

	// then
	require.NoError(t, err)
	mentioned := awaitMention()
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, "ship_comment:"+commentID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/ships/"+shipID.String()+"#comment-"+commentID.String(), mentioned.EmailLink)
}

func TestUpdateComment(t *testing.T) {
	cases := []struct {
		name       string
		canEditAny bool
		repoErr    error
	}{
		{name: "admin edits as admin", canEditAny: true},
		{name: "owner edit refused by the repo bubbles", repoErr: errNotOwner},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID, userID := uuid.New(), uuid.New()
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(tc.canEditAny)
			m.shipRepo.EXPECT().
				UpdateCommentBody(mock.Anything, spec.CommentUpdate{
					CommentID: commentID,
					UserID:    userID,
					Body:      "hi",
					AsAdmin:   tc.canEditAny,
				}).
				Return(tc.repoErr)

			// when
			err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "  hi  "})

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestDeleteComment(t *testing.T) {
	cases := []struct {
		name         string
		canDeleteAny bool
		paths        []string
		repoErr      error
	}{
		{name: "admin delete removes the comment's files", canDeleteAny: true, paths: []string{"/uploads/ships/c.png", "/uploads/ships/c-thumb.png"}},
		{name: "owner delete refused by the repo bubbles", repoErr: errNotOwner},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID, userID := uuid.New(), uuid.New()
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(tc.canDeleteAny)
			m.shipRepo.EXPECT().
				DeleteCommentWithAudit(mock.Anything, spec.CommentDeletion{CommentID: commentID, UserID: userID, AsAdmin: tc.canDeleteAny}).
				Return(tc.paths, tc.repoErr)
			if tc.repoErr == nil {
				m.uploadSvc.EXPECT().Delete(tc.paths).Return()
			}

			// when
			err := svc.DeleteComment(context.Background(), commentID, userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestLikeComment(t *testing.T) {
	cases := []struct {
		name      string
		selfLike  bool
		lookupErr error
		blocked   bool
		blockErr  error
		likeErr   error
		notifies  bool
		want      error
	}{
		{name: "author lookup error bubbles", lookupErr: errNoRow, want: errNoRow},
		{name: "a block either way rejects the like", blocked: true, want: block.ErrUserBlocked},
		{name: "a failed block lookup rejects the like", blockErr: errBoom, want: errBoom},
		{name: "repo error bubbles", likeErr: errBoom, want: errBoom},
		{name: "self like skips the notification", selfLike: true},
		{name: "liking another author's comment looks up the ship to notify them", notifies: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID, userID, authorID := uuid.New(), uuid.New(), uuid.New()
			if tc.selfLike {
				authorID = userID
			}
			m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, tc.lookupErr)
			if tc.lookupErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}
			if tc.lookupErr == nil && !tc.blocked && tc.blockErr == nil {
				m.shipRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(tc.likeErr)
			}

			var notifying sync.WaitGroup
			if tc.notifies {
				notifying.Add(1)
				m.shipRepo.EXPECT().GetCommentEntityID(mock.Anything, commentID).
					RunAndReturn(func(context.Context, uuid.UUID, ...*sql.Tx) (uuid.UUID, error) {
						notifying.Done()

						return uuid.Nil, errStop
					})
			}

			// when
			err := svc.LikeComment(context.Background(), userID, commentID)

			// then
			require.ErrorIs(t, err, tc.want)
			notifying.Wait()
		})
	}
}

func TestUnlikeComment(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
	}{
		{name: "unlike succeeds"},
		{name: "repo error bubbles", repoErr: errBoom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID, userID := uuid.New(), uuid.New()
			m.shipRepo.EXPECT().UnlikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(tc.repoErr)

			// when
			err := svc.UnlikeComment(context.Background(), userID, commentID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestUploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID, userID, authorID := uuid.New(), uuid.New(), uuid.New()
	m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", "photo.png", 100, bytes.NewReader(nil), false)

	// then
	require.ErrorContains(t, err, "not the comment author")
}

func TestUploadCommentMedia(t *testing.T) {
	cases := []struct {
		name      string
		cancelled bool
		lookupErr error
		saveErr   error
		mediaID   int64
		addErr    error
		want      error
		wantResp  *dto.PostMediaResponse
	}{
		{name: "missing comment is not found", lookupErr: errNoRow, want: ErrNotFound},
		{name: "a comment author lookup failure is surfaced, not reported as not found", lookupErr: errBoom, want: errBoom},
		{name: "upload error bubbles", saveErr: errDiskFull, want: errDiskFull},
		{name: "media row error bubbles", addErr: errBoom, want: errBoom},
		{
			name:      "cancelled request still records the media",
			cancelled: true,
			mediaID:   42,
			wantResp:  &dto.PostMediaResponse{ID: 42, MediaURL: savedImageURL, MediaType: "image", Filename: "photo.png"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID, userID := uuid.New(), uuid.New()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelled {
				cancel()
			}
			m.shipRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, tc.lookupErr)
			if tc.lookupErr == nil {
				expectImageSaved(m, tc.saveErr)
			}
			if tc.lookupErr == nil && tc.saveErr == nil {
				m.shipRepo.EXPECT().
					AddCommentMedia(mock.Anything, spec.NewMedia{
						TargetID:  commentID,
						MediaURL:  savedImageURL,
						MediaType: "image",
						Filename:  "photo.png",
					}).
					Return(tc.mediaID, tc.addErr)
			}

			// when
			resp, err := svc.UploadCommentMedia(ctx, commentID, userID, "image/png", "photo.png", 100, bytes.NewReader(nil), false)

			// then
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, tc.wantResp, resp)
		})
	}
}

func TestListCharacters_InvalidSeries(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.ListCharacters("nonsense")

	// then
	require.Error(t, err)
}
