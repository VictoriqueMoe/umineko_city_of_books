package art

import (
	"bytes"
	"context"
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
		artRepo     *repository.MockArtRepository
		artComments *dao.MockCommentDAO[uuid.UUID]
		userRepo    *repository.MockUserRepository
		auditRepo   *repository.MockAuditLogRepository
		authz       *authz.MockService
		blockSvc    *block.MockService
		notifSvc    *notification.MockService
		uploadSvc   *upload.MockService
		settingsSvc *settings.MockService
	}
)

var (
	errDB         = errors.New("db down")
	errStop       = errors.New("stop goroutine")
	errMissingRow = errors.Join(errors.New("no row"), dao.ErrNotFound)
)

func newTestService(t *testing.T) (*service, *testMocks) {
	t.Helper()

	m := &testMocks{
		artRepo:     repository.NewMockArtRepository(t),
		artComments: dao.NewMockCommentDAO[uuid.UUID](t),
		userRepo:    repository.NewMockUserRepository(t),
		auditRepo:   repository.NewMockAuditLogRepository(t),
		authz:       authz.NewMockService(t),
		blockSvc:    block.NewMockService(t),
		notifSvc:    notification.NewMockService(t),
		uploadSvc:   upload.NewMockService(t),
		settingsSvc: settings.NewMockService(t),
	}

	m.uploadSvc.EXPECT().FullDiskPath(mock.Anything).Return("/tmp/does-not-exist-xyz.png").Maybe()

	mentionSvc := mention.NewService(m.userRepo, m.blockSvc, m.notifSvc, dao.CommentDAOs{
		ByID: map[string]dao.CommentDAO[uuid.UUID]{string(mention.KindArtComment): m.artComments},
	})

	svc := NewService(m.artRepo, repository.NewMockPostRepository(t), m.userRepo, m.auditRepo, m.authz, m.blockSvc, m.notifSvc, mentionSvc, m.uploadSvc, media.NewProcessor(1), m.settingsSvc, contentfilter.New(), nil, nil).(*service)

	return svc, m
}

func expectImageSave(m *testMocks, urlPath string, err error) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "art", mock.Anything, int64(10), int64(1000), mock.Anything).
		Return(urlPath, err)
}

func expectArtAuthorNotBlocked(m *testMocks, artID uuid.UUID, userID uuid.UUID, authorID uuid.UUID) {
	m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
}

func expectCommentAuthorNotBlocked(m *testMocks, commentID uuid.UUID, userID uuid.UUID, authorID uuid.UUID) {
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
}

func expectArtDetailReads(m *testMocks, id uuid.UUID, viewerID uuid.UUID, blocked []uuid.UUID) {
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(blocked, nil)
	m.artRepo.EXPECT().GetTags(mock.Anything, id).Return([]string{"tag"}, nil)
	m.artRepo.EXPECT().
		GetComments(mock.Anything, spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: viewerID, Limit: 500, ExcludeUserIDs: blocked}).
		Return(nil, 0, nil)
	m.artRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
	m.artRepo.EXPECT().GetLikedBy(mock.Anything, spec.LikedByQuery{TargetID: id, ExcludeUserIDs: blocked}).Return(nil, nil)
}

func expectMentionOfAlice(m *testMocks, actorID uuid.UUID, aliceID uuid.UUID, notifications int) (*sync.WaitGroup, *dto.NotifyParams) {
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "Battler"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: aliceID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, aliceID).Return(false, nil)

	sent := &sync.WaitGroup{}
	sent.Add(notifications)

	mentioned := &dto.NotifyParams{}
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			if p.Type == dto.NotifMention {
				*mentioned = p
			}
			sent.Done()

			return nil
		})

	return sent, mentioned
}

func TestCreateArt_RejectedBeforeWriting(t *testing.T) {
	userID := uuid.New()

	cases := []struct {
		name    string
		req     dto.CreateArtRequest
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name:    "blank title",
			req:     dto.CreateArtRequest{Title: "   "},
			given:   func(*testMocks) {},
			wantErr: ErrEmptyTitle,
		},
		{
			name: "daily count lookup failure propagates",
			req:  dto.CreateArtRequest{Title: "t"},
			given: func(m *testMocks) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(5)
				m.artRepo.EXPECT().CountUserArtToday(mock.Anything, userID).Return(0, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "daily limit reached",
			req:  dto.CreateArtRequest{Title: "t"},
			given: func(m *testMocks) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(5)
				m.artRepo.EXPECT().CountUserArtToday(mock.Anything, userID).Return(5, nil)
			},
			wantErr: ErrRateLimited,
		},
		{
			name: "image save failure propagates",
			req:  dto.CreateArtRequest{Title: "t"},
			given: func(m *testMocks) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
				expectImageSave(m, "", errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			id, err := svc.CreateArt(context.Background(), userID, tc.req, "image/png", 10, bytes.NewReader(nil))

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, uuid.Nil, id)
		})
	}
}

func TestCreateArt_WritesTrimmedSpecWithDefaults(t *testing.T) {
	userID := uuid.New()
	artID := uuid.New()
	uploaded := "/uploads/art/x.png"
	tags := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}

	cases := []struct {
		name    string
		req     dto.CreateArtRequest
		want    spec.NewArtWithTags
		created *model.ArtRow
		repoErr error
		wantID  uuid.UUID
	}{
		{
			name: "trims title and description and propagates a repo failure",
			req:  dto.CreateArtRequest{Title: "  t  ", Description: "  d  ", Tags: []string{"a"}},
			want: spec.NewArtWithTags{
				NewArt: spec.NewArt{UserID: userID, Corner: "general", ArtType: "drawing", Title: "t", Description: "d", ImageURL: uploaded},
				Tags:   []string{"a"},
			},
			repoErr: errDB,
		},
		{
			name: "defaults corner and type and caps tags at ten",
			req:  dto.CreateArtRequest{Title: "t", Tags: tags, IsSpoiler: true},
			want: spec.NewArtWithTags{
				NewArt: spec.NewArt{UserID: userID, Corner: "general", ArtType: "drawing", Title: "t", ImageURL: uploaded, IsSpoiler: true},
				Tags:   tags[:10],
			},
			created: &model.ArtRow{ID: artID},
			wantID:  artID,
		},
		{
			name: "keeps a custom corner and type",
			req:  dto.CreateArtRequest{Title: "t", Corner: "umineko", ArtType: "sketch"},
			want: spec.NewArtWithTags{
				NewArt: spec.NewArt{UserID: userID, Corner: "umineko", ArtType: "sketch", Title: "t", ImageURL: uploaded},
			},
			created: &model.ArtRow{ID: artID},
			wantID:  artID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
			expectImageSave(m, uploaded, nil)
			m.artRepo.EXPECT().CreateWithTags(mock.Anything, tc.want).Return(tc.created, tc.repoErr)

			// when
			id, err := svc.CreateArt(context.Background(), userID, tc.req, "image/png", 10, bytes.NewReader(nil))

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.wantID, id)
		})
	}
}

func TestCreateArt_MentionOnTheDescriptionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	artID := uuid.New()
	mentionedID := uuid.New()

	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxArtPerDay).Return(0)
	expectImageSave(m, "/u/a.png", nil)
	m.artRepo.EXPECT().CreateWithTags(mock.Anything, mock.Anything).Return(&model.ArtRow{ID: artID}, nil)
	sent, mentioned := expectMentionOfAlice(m, userID, mentionedID, 1)

	req := dto.CreateArtRequest{Title: "Golden", Description: "drawn for @alice"}

	// when
	_, err := svc.CreateArt(context.Background(), userID, req, "image/png", 10, bytes.NewReader(nil))

	// then
	require.NoError(t, err)
	sent.Wait()
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, artID, mentioned.ReferenceID)
	assert.Equal(t, "art", mentioned.ReferenceType)
	assert.Equal(t, "/gallery/art/"+artID.String(), mentioned.EmailLink)
}

func TestGetArt_LookupFailure(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "repo failure propagates", repoErr: errDB, wantErr: errDB},
		{name: "missing row maps to not found", wantErr: ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			m.artRepo.EXPECT().GetByID(mock.Anything, spec.ArtLookup{ID: id, ViewerID: uuid.Nil}).Return(nil, tc.repoErr)

			// when
			_, err := svc.GetArt(context.Background(), id, uuid.Nil, "")

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetArt_OK_WithViewerHashAndBlocked(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	viewerID := uuid.New()
	authorID := uuid.New()
	blocked := []uuid.UUID{uuid.New()}

	row := &model.ArtRow{ID: id, UserID: authorID, Title: "T", ImageURL: "/u/x.png", ViewCount: 5}
	m.artRepo.EXPECT().GetByID(mock.Anything, spec.ArtLookup{ID: id, ViewerID: viewerID}).Return(row, nil)
	m.artRepo.EXPECT().RecordView(mock.Anything, spec.ViewRecord{TargetID: id, ViewerHash: "hashy"}).Return(true, nil)
	expectArtDetailReads(m, id, viewerID, blocked)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(true, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	got, err := svc.GetArt(context.Background(), id, viewerID, "hashy")

	// then
	require.NoError(t, err)
	assert.Equal(t, 6, got.ViewCount)
	assert.True(t, got.ViewerBlocked)
	assert.NotEmpty(t, got.ThumbnailURL)
	assert.Equal(t, []string{"tag"}, got.Tags)
}

func TestGetArt_SurfacesAFailedLookup(t *testing.T) {
	cases := []struct {
		name   string
		failAt int
	}{
		{name: "the viewer's block list", failAt: 0},
		{name: "the tags", failAt: 1},
		{name: "the comments", failAt: 2},
		{name: "the comment media", failAt: 3},
		{name: "the likes", failAt: 4},
		{name: "the block check against the author", failAt: 5},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as empty", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			viewerID := uuid.New()
			authorID := uuid.New()
			m.artRepo.EXPECT().GetByID(mock.Anything, spec.ArtLookup{ID: id, ViewerID: viewerID}).Return(&model.ArtRow{ID: id, UserID: authorID}, nil)
			m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("").Maybe()

			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, artFailAt(tc.failAt, 0))
			if tc.failAt >= 1 {
				m.artRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, artFailAt(tc.failAt, 1))
			}
			if tc.failAt >= 2 {
				m.artRepo.EXPECT().GetComments(mock.Anything, mock.Anything).Return(nil, 0, artFailAt(tc.failAt, 2))
			}
			if tc.failAt >= 3 {
				m.artRepo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, artFailAt(tc.failAt, 3))
			}
			if tc.failAt >= 4 {
				m.artRepo.EXPECT().GetLikedBy(mock.Anything, mock.Anything).Return(nil, artFailAt(tc.failAt, 4))
			}
			if tc.failAt >= 5 {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(false, artFailAt(tc.failAt, 5))
			}

			// when
			got, err := svc.GetArt(context.Background(), id, viewerID, "")

			// then
			require.ErrorIs(t, err, errDB)
			assert.Nil(t, got)
		})
	}
}

func artFailAt(failAt int, step int) error {
	if failAt == step {
		return errDB
	}

	return nil
}

func TestGetArt_OK_AnonymousNoHash(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()

	row := &model.ArtRow{ID: id, UserID: uuid.New(), Title: "T", ImageURL: "/u/x.png"}
	m.artRepo.EXPECT().GetByID(mock.Anything, spec.ArtLookup{ID: id, ViewerID: uuid.Nil}).Return(row, nil)
	expectArtDetailReads(m, id, uuid.Nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("")

	// when
	got, err := svc.GetArt(context.Background(), id, uuid.Nil, "")

	// then
	require.NoError(t, err)
	assert.False(t, got.ViewerBlocked)
}

func TestUpdateArt(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	tags := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}

	cases := []struct {
		name    string
		req     dto.UpdateArtRequest
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name:    "blank title is rejected before the permission check",
			req:     dto.UpdateArtRequest{Title: "  "},
			given:   func(*testMocks) {},
			wantErr: ErrEmptyTitle,
		},
		{
			name: "owner update failure propagates",
			req:  dto.UpdateArtRequest{Title: "t"},
			given: func(m *testMocks) {
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
				m.artRepo.EXPECT().
					UpdateWithTags(mock.Anything, spec.ArtUpdateWithTags{ArtUpdate: spec.ArtUpdate{ID: id, UserID: userID, Title: "t"}}).
					Return(errDB)
			},
			wantErr: errDB,
		},
		{
			name: "owner update trims fields and caps tags at ten",
			req:  dto.UpdateArtRequest{Title: "  t  ", Description: "  d  ", Tags: tags, IsSpoiler: true},
			given: func(m *testMocks) {
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
				m.artRepo.EXPECT().
					UpdateWithTags(mock.Anything, spec.ArtUpdateWithTags{
						ArtUpdate: spec.ArtUpdate{ID: id, UserID: userID, Title: "t", Description: "d", IsSpoiler: true},
						Tags:      tags[:10],
					}).
					Return(nil)
			},
		},
		{
			name: "admin update is flagged and spawns the edit notification",
			req:  dto.UpdateArtRequest{Title: "t"},
			given: func(m *testMocks) {
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(true)
				m.artRepo.EXPECT().
					UpdateWithTags(mock.Anything, spec.ArtUpdateWithTags{ArtUpdate: spec.ArtUpdate{ID: id, UserID: userID, Title: "t", AsAdmin: true}}).
					Return(nil)
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, id).Return(uuid.Nil, errStop).Maybe()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			err := svc.UpdateArt(context.Background(), id, userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDeleteArt(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	otherID := uuid.New()

	cases := []struct {
		name         string
		authorID     uuid.UUID
		canDeleteAny bool
		wantAction   audit.Action
		paths        []string
		repoErr      error
	}{
		{name: "admin deletes someone else's art", authorID: otherID, canDeleteAny: true, wantAction: audit.ActionArtDeleteAdmin, paths: []string{"/u/x.png", "/u/x-thumb.png"}},
		{name: "moderator deleting own art is not an admin action", authorID: userID, canDeleteAny: true, wantAction: audit.ActionArtDelete, paths: []string{"/u/x.png"}},
		{name: "admin delete failure propagates", authorID: otherID, canDeleteAny: true, wantAction: audit.ActionArtDeleteAdmin, repoErr: errDB},
		{name: "owner deletes own art", authorID: userID, wantAction: audit.ActionArtDelete, paths: []string{"/u/x.png", "/u/c.png"}},
		{name: "owner delete failure propagates", authorID: userID, wantAction: audit.ActionArtDelete, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, id).Return(tc.authorID, nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(tc.canDeleteAny)
			m.artRepo.EXPECT().
				DeleteWithImage(mock.Anything, spec.ArtDelete{
					ID:      id,
					UserID:  userID,
					AsAdmin: tc.canDeleteAny,
					Audit: audit.NewEntry{
						ActorID:    userID,
						Action:     tc.wantAction,
						TargetType: audit.TargetArt,
						TargetID:   id.String(),
						SubjectID:  tc.authorID,
					},
				}).
				Return(tc.paths, tc.repoErr)

			if tc.repoErr == nil {
				m.uploadSvc.EXPECT().Delete(tc.paths).Return()
			}

			// when
			err := svc.DeleteArt(context.Background(), id, userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestArtAuthorLookupFailures(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		call func(s *service, id uuid.UUID) error
	}{
		{name: "delete", call: func(s *service, id uuid.UUID) error { return s.DeleteArt(ctx, id, uuid.New()) }},
		{name: "like", call: func(s *service, id uuid.UUID) error { return s.LikeArt(ctx, uuid.New(), id) }},
		{name: "comment", call: func(s *service, id uuid.UUID) error {
			_, err := s.CreateComment(ctx, id, uuid.New(), dto.CreateCommentRequest{Body: "hi"})
			return err
		}},
	}
	outcomes := []struct {
		name      string
		lookupErr error
		wantErr   error
	}{
		{name: "a missing art is not found", lookupErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a failed lookup is surfaced", lookupErr: errDB, wantErr: errDB},
	}

	for _, tc := range cases {
		for _, outcome := range outcomes {
			t.Run(tc.name+": "+outcome.name, func(t *testing.T) {
				// given
				svc, m := newTestService(t)
				id := uuid.New()
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, id).Return(uuid.Nil, outcome.lookupErr)

				// when
				err := tc.call(svc, id)

				// then
				require.ErrorIs(t, err, outcome.wantErr)
			})
		}
	}
}

func TestListArt_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	m.artRepo.EXPECT().
		ListAll(mock.Anything, spec.ArtFilter{ViewerID: viewerID, Corner: "general", Limit: 10}).
		Return(nil, 0, errDB)

	// when
	_, err := svc.ListArt(context.Background(), viewerID, "", "", "", "", "", bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, errDB)
}

func TestListArt_OK_DefaultsCornerAndThumbnails(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	artID := uuid.New()

	rows := []model.ArtRow{{ID: artID, UserID: uuid.New(), Title: "A", ImageURL: "/u/x.png"}}
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
	m.artRepo.EXPECT().
		ListAll(mock.Anything, spec.ArtFilter{ViewerID: viewerID, Corner: "general", ArtType: "drawing", Search: "q", Tag: "tag", Sort: "new", Limit: 10, Offset: 5}).
		Return(rows, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{artID}).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	got, err := svc.ListArt(context.Background(), viewerID, "", "drawing", "q", "tag", "new", bounds.NewPage(10, 5))

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 5, got.Offset)
	assert.Len(t, got.Art, 1)
	assert.NotEmpty(t, got.Art[0].ThumbnailURL)
}

func TestListArt_AFailedBlockListIsSurfacedBeforeAnythingIsListed(t *testing.T) {
	// given
	svc, m := newTestService(t)
	viewerID := uuid.New()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, errDB)

	// when
	got, err := svc.ListArt(context.Background(), viewerID, "", "", "", "", "", bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, errDB, "listing without the block list would show blocked users' art")
	assert.Nil(t, got)
}

func TestListByUser_AFailedTagLookupIsSurfaced(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	m.artRepo.EXPECT().ListByUser(mock.Anything, mock.Anything).Return([]model.ArtRow{{ID: uuid.New()}}, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, mock.Anything).Return(nil, errDB)

	// when
	got, err := svc.ListByUser(context.Background(), userID, viewerID, bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, errDB)
	assert.Nil(t, got)
}

func TestListByUser_RepoError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	m.artRepo.EXPECT().
		ListByUser(mock.Anything, spec.ArtUserFilter{UserID: userID, ViewerID: viewerID, Limit: 10}).
		Return(nil, 0, errDB)

	// when
	_, err := svc.ListByUser(context.Background(), userID, viewerID, bounds.NewPage(10, 0))

	// then
	require.ErrorIs(t, err, errDB)
}

func TestListByUser_OK(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	viewerID := uuid.New()
	artID := uuid.New()

	rows := []model.ArtRow{{ID: artID, UserID: userID, Title: "A", ImageURL: "/u/y.png"}}
	m.artRepo.EXPECT().ListByUser(mock.Anything, spec.ArtUserFilter{UserID: userID, ViewerID: viewerID, Limit: 10}).Return(rows, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{artID}).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("")

	// when
	got, err := svc.ListByUser(context.Background(), userID, viewerID, bounds.NewPage(10, 0))

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Art, 1)
}

func TestLikeArt(t *testing.T) {
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()

	cases := []struct {
		name    string
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name: "author lookup failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(uuid.Nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "a block in either direction refuses the like",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)
			},
			wantErr: block.ErrUserBlocked,
		},
		{
			name: "a failed block lookup refuses the like",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "like write failure propagates",
			given: func(m *testMocks) {
				expectArtAuthorNotBlocked(m, artID, userID, authorID)
				m.artRepo.EXPECT().Like(mock.Anything, spec.Like{UserID: userID, TargetID: artID}).Return(errDB)
			},
			wantErr: errDB,
		},
		{
			name: "records the like",
			given: func(m *testMocks) {
				expectArtAuthorNotBlocked(m, artID, userID, authorID)
				m.artRepo.EXPECT().Like(mock.Anything, spec.Like{UserID: userID, TargetID: artID}).Return(nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errStop).Maybe()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			err := svc.LikeArt(context.Background(), userID, artID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestUnlikeArt(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
	}{
		{name: "removes the like"},
		{name: "repo failure propagates", repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			artID := uuid.New()
			userID := uuid.New()
			m.artRepo.EXPECT().Unlike(mock.Anything, spec.Like{UserID: userID, TargetID: artID}).Return(tc.repoErr)

			// when
			err := svc.UnlikeArt(context.Background(), userID, artID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestGetCornerCounts(t *testing.T) {
	cases := []struct {
		name    string
		counts  map[string]int
		repoErr error
	}{
		{name: "returns the per corner counts", counts: map[string]int{"general": 2}},
		{name: "repo failure propagates", repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.artRepo.EXPECT().GetCornerCounts(mock.Anything).Return(tc.counts, tc.repoErr)

			// when
			got, err := svc.GetCornerCounts(context.Background())

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.counts, got)
		})
	}
}

func TestGetPopularTags(t *testing.T) {
	cases := []struct {
		name    string
		corner  string
		rows    []model.TagCount
		repoErr error
		want    []dto.TagCountResponse
	}{
		{
			name:   "maps the corner's tag counts",
			corner: "umineko",
			rows:   []model.TagCount{{Tag: "a", Count: 4}},
			want:   []dto.TagCountResponse{{Tag: "a", Count: 4}},
		},
		{name: "repo failure propagates", corner: "general", repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.artRepo.EXPECT().GetPopularTags(mock.Anything, spec.PopularTagFilter{Corner: tc.corner, Limit: 30}).Return(tc.rows, tc.repoErr)

			// when
			got, err := svc.GetPopularTags(context.Background(), tc.corner)

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCreateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), dto.CreateCommentRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestCreateComment(t *testing.T) {
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	parentID := uuid.New()
	commentID := uuid.New()

	cases := []struct {
		name    string
		req     dto.CreateCommentRequest
		given   func(m *testMocks)
		wantID  uuid.UUID
		wantErr error
	}{
		{
			name: "art author lookup failure propagates",
			req:  dto.CreateCommentRequest{Body: "hi"},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(uuid.Nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "a block in either direction refuses the comment",
			req:  dto.CreateCommentRequest{Body: "hi"},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)
			},
			wantErr: block.ErrUserBlocked,
		},
		{
			name: "a failed block lookup refuses the comment",
			req:  dto.CreateCommentRequest{Body: "hi"},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetArtAuthorID(mock.Anything, artID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "trims the body and propagates a write failure",
			req:  dto.CreateCommentRequest{Body: "  hi  "},
			given: func(m *testMocks) {
				expectArtAuthorNotBlocked(m, artID, userID, authorID)
				m.artComments.EXPECT().
					CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: artID, ParentID: nil, UserID: userID, Body: "hi"}).
					Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "reply carries the parent id",
			req:  dto.CreateCommentRequest{Body: "hi", ParentID: &parentID},
			given: func(m *testMocks) {
				expectArtAuthorNotBlocked(m, artID, userID, authorID)
				m.artComments.EXPECT().
					CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: artID, ParentID: &parentID, UserID: userID, Body: "hi"}).
					Return(&model.CommentRow{ID: commentID}, nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, errStop).Maybe()
			},
			wantID: commentID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			id, err := svc.CreateComment(context.Background(), artID, userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.wantID, id)
		})
	}
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	artID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	commentID := uuid.New()
	mentionedID := uuid.New()

	expectArtAuthorNotBlocked(m, artID, userID, authorID)
	m.artComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: artID, ParentID: nil, UserID: userID, Body: "look at this @alice"}).
		Return(&model.CommentRow{ID: commentID}, nil)
	sent, mentioned := expectMentionOfAlice(m, userID, mentionedID, 2)

	// when
	id, err := svc.CreateComment(context.Background(), artID, userID, dto.CreateCommentRequest{Body: "look at this @alice"})

	// then
	require.NoError(t, err)
	assert.Equal(t, commentID, id)
	sent.Wait()
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, artID, mentioned.ReferenceID)
	assert.Equal(t, "art_comment:"+commentID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/gallery/art/"+artID.String()+"#comment-"+commentID.String(), mentioned.EmailLink)
}

func TestUpdateComment_EmptyBodyRejected(t *testing.T) {
	// given
	svc, _ := newTestService(t)

	// when
	err := svc.UpdateComment(context.Background(), uuid.New(), uuid.New(), dto.UpdateCommentRequest{Body: "  "})

	// then
	require.ErrorIs(t, err, ErrEmptyBody)
}

func TestUpdateComment_AuthorLookupFailures(t *testing.T) {
	cases := []struct {
		name      string
		lookupErr error
		wantErr   error
	}{
		{name: "a missing comment is not found", lookupErr: errMissingRow, wantErr: ErrNotFound},
		{name: "a failed lookup is surfaced", lookupErr: errDB, wantErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID := uuid.New()
			m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, tc.lookupErr)

			// when
			err := svc.UpdateComment(context.Background(), commentID, uuid.New(), dto.UpdateCommentRequest{Body: "hi"})

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestUpdateComment(t *testing.T) {
	commentID := uuid.New()
	userID := uuid.New()
	otherID := uuid.New()

	cases := []struct {
		name       string
		authorID   uuid.UUID
		canEditAny bool
		repoErr    error
		audited    bool
		notifies   bool
	}{
		{name: "admin editing someone else's comment is audited", authorID: otherID, canEditAny: true, audited: true, notifies: true},
		{name: "moderator editing own comment is not audited", authorID: userID, canEditAny: true, notifies: true},
		{name: "admin edit failure propagates", authorID: otherID, canEditAny: true, repoErr: errDB},
		{name: "owner edit", authorID: userID},
		{name: "owner edit failure propagates", authorID: userID, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(tc.authorID, nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(tc.canEditAny)
			m.artRepo.EXPECT().
				UpdateCommentWithDetails(mock.Anything, spec.CommentUpdate{CommentID: commentID, UserID: userID, Body: "hi", AsAdmin: tc.canEditAny}).
				Return(tc.repoErr)

			if tc.audited {
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    userID,
					Action:     audit.ActionArtCommentUpdateAdmin,
					TargetType: audit.TargetArtComment,
					TargetID:   commentID.String(),
					SubjectID:  otherID,
				}).Return(nil)
			}

			if tc.notifies {
				m.artRepo.EXPECT().GetCommentEntityID(mock.Anything, commentID).Return(uuid.Nil, errStop).Maybe()
			}

			// when
			err := svc.UpdateComment(context.Background(), commentID, userID, dto.UpdateCommentRequest{Body: "  hi  "})

			// then
			require.ErrorIs(t, err, tc.repoErr)

			if !tc.audited {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteComment(t *testing.T) {
	commentID := uuid.New()
	userID := uuid.New()
	otherID := uuid.New()

	cases := []struct {
		name         string
		authorID     uuid.UUID
		canDeleteAny bool
		wantAction   audit.Action
		paths        []string
		repoErr      error
	}{
		{name: "admin deletes someone else's comment", authorID: otherID, canDeleteAny: true, wantAction: audit.ActionArtCommentDeleteAdmin, paths: []string{"/u/c.png", "/u/c-thumb.png"}},
		{name: "moderator deleting own comment is not an admin action", authorID: userID, canDeleteAny: true, wantAction: audit.ActionArtCommentDelete, paths: []string{"/u/c.png"}},
		{name: "owner delete failure propagates", authorID: userID, wantAction: audit.ActionArtCommentDelete, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(tc.authorID, nil)
			m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(tc.canDeleteAny)
			m.artRepo.EXPECT().
				DeleteCommentWithAudit(mock.Anything, spec.ArtCommentDeletion{
					CommentDeletion: spec.CommentDeletion{CommentID: commentID, UserID: userID, AsAdmin: tc.canDeleteAny},
					Audit: audit.NewEntry{
						ActorID:    userID,
						Action:     tc.wantAction,
						TargetType: audit.TargetArtComment,
						TargetID:   commentID.String(),
						SubjectID:  tc.authorID,
					},
				}).
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

func TestDeleteComment_AuthorLookupError(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errDB)

	// when
	err := svc.DeleteComment(context.Background(), commentID, uuid.New())

	// then
	require.ErrorIs(t, err, errDB)
}

func TestLikeComment(t *testing.T) {
	commentID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()

	cases := []struct {
		name    string
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name: "comment author lookup failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "a block in either direction refuses the like",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(true, nil)
			},
			wantErr: block.ErrUserBlocked,
		},
		{
			name: "a failed block lookup refuses the like",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "like write failure propagates",
			given: func(m *testMocks) {
				expectCommentAuthorNotBlocked(m, commentID, userID, authorID)
				m.artRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(errDB)
			},
			wantErr: errDB,
		},
		{
			name: "records the like",
			given: func(m *testMocks) {
				expectCommentAuthorNotBlocked(m, commentID, userID, authorID)
				m.artRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(nil)
				m.artRepo.EXPECT().GetCommentEntityID(mock.Anything, commentID).Return(uuid.Nil, errStop).Maybe()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			err := svc.LikeComment(context.Background(), userID, commentID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestUnlikeComment(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
	}{
		{name: "removes the like"},
		{name: "repo failure propagates", repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID := uuid.New()
			userID := uuid.New()
			m.artRepo.EXPECT().UnlikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(tc.repoErr)

			// when
			err := svc.UnlikeComment(context.Background(), userID, commentID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestUploadCommentMedia(t *testing.T) {
	commentID := uuid.New()
	userID := uuid.New()
	uploaded := "/uploads/art/x.png"
	mediaRow := spec.NewMedia{TargetID: commentID, MediaURL: uploaded, MediaType: "image", Filename: "photo.png"}

	cases := []struct {
		name     string
		given    func(m *testMocks)
		wantResp *dto.PostMediaResponse
		wantErr  error
	}{
		{
			name: "comment lookup failure maps to not found",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.Nil, errDB)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "image save failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
				expectImageSave(m, "", errDB)
			},
			wantErr: errDB,
		},
		{
			name: "media row failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
				expectImageSave(m, uploaded, nil)
				m.artRepo.EXPECT().AddCommentMedia(mock.Anything, mediaRow).Return(int64(0), errDB)
			},
			wantErr: errDB,
		},
		{
			name: "records the media row",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(userID, nil)
				expectImageSave(m, uploaded, nil)
				m.artRepo.EXPECT().AddCommentMedia(mock.Anything, mediaRow).Return(int64(42), nil)
			},
			wantResp: &dto.PostMediaResponse{ID: 42, MediaURL: uploaded, MediaType: "image", Filename: "photo.png"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			resp, err := svc.UploadCommentMedia(context.Background(), commentID, userID, "image/png", "photo.png", 10, bytes.NewReader(nil), false)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.wantResp, resp)
		})
	}
}

func TestUploadCommentMedia_NotAuthor(t *testing.T) {
	// given
	svc, m := newTestService(t)
	commentID := uuid.New()
	m.artRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(uuid.New(), nil)

	// when
	_, err := svc.UploadCommentMedia(context.Background(), commentID, uuid.New(), "image/png", "photo.png", 10, bytes.NewReader(nil), false)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the comment author")
}

func TestCreateGallery(t *testing.T) {
	userID := uuid.New()
	galleryID := uuid.New()

	cases := []struct {
		name    string
		req     dto.CreateGalleryRequest
		given   func(m *testMocks)
		wantID  uuid.UUID
		wantErr error
	}{
		{
			name:    "blank name is rejected",
			req:     dto.CreateGalleryRequest{Name: "  "},
			given:   func(*testMocks) {},
			wantErr: ErrEmptyTitle,
		},
		{
			name: "trims name and description and propagates a repo failure",
			req:  dto.CreateGalleryRequest{Name: "  n  ", Description: "  d  "},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().CreateGallery(mock.Anything, spec.NewGallery{UserID: userID, Name: "n", Description: "d"}).Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "creates the gallery",
			req:  dto.CreateGalleryRequest{Name: "n", Description: "d"},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().
					CreateGallery(mock.Anything, spec.NewGallery{UserID: userID, Name: "n", Description: "d"}).
					Return(&model.GalleryRow{ID: galleryID}, nil)
			},
			wantID: galleryID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			id, err := svc.CreateGallery(context.Background(), userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.wantID, id)
		})
	}
}

func TestUpdateGallery(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()

	cases := []struct {
		name    string
		req     dto.UpdateGalleryRequest
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name:    "blank name is rejected",
			req:     dto.UpdateGalleryRequest{Name: "  "},
			given:   func(*testMocks) {},
			wantErr: ErrEmptyTitle,
		},
		{
			name: "trims name and description",
			req:  dto.UpdateGalleryRequest{Name: "  n  ", Description: "  d  "},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().UpdateGallery(mock.Anything, spec.GalleryUpdate{ID: id, UserID: userID, Name: "n", Description: "d"}).Return(nil)
			},
		},
		{
			name: "repo failure propagates",
			req:  dto.UpdateGalleryRequest{Name: "n"},
			given: func(m *testMocks) {
				m.artRepo.EXPECT().UpdateGallery(mock.Anything, spec.GalleryUpdate{ID: id, UserID: userID, Name: "n"}).Return(errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			err := svc.UpdateGallery(context.Background(), id, userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestSetGalleryCover(t *testing.T) {
	coverID := uuid.New()

	cases := []struct {
		name       string
		coverArtID *uuid.UUID
		repoErr    error
	}{
		{name: "sets the cover", coverArtID: &coverID},
		{name: "clearing the cover propagates a repo failure", coverArtID: nil, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			galleryID := uuid.New()
			userID := uuid.New()
			m.artRepo.EXPECT().
				SetGalleryCover(mock.Anything, spec.GalleryCoverUpdate{GalleryID: galleryID, UserID: userID, CoverArtID: tc.coverArtID}).
				Return(tc.repoErr)

			// when
			err := svc.SetGalleryCover(context.Background(), galleryID, userID, tc.coverArtID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestDeleteGallery_FailuresWriteNoAudit(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()

	cases := []struct {
		name    string
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name: "missing gallery maps to not found",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(nil, nil)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "delete failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(&model.GalleryRow{ID: id, UserID: userID, Name: "sketches", ArtCount: 3}, nil)
				m.artRepo.EXPECT().DeleteGallery(mock.Anything, spec.GalleryRef{GalleryID: id, UserID: userID}).Return(nil, errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			err := svc.DeleteGallery(context.Background(), id, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestDeleteGallery_OK_DeletesImages(t *testing.T) {
	// given
	svc, m := newTestService(t)
	id := uuid.New()
	userID := uuid.New()
	paths := []string{"/u/a.png", "/u/a-thumb.png", "/u/b.png", "/u/comment.png"}

	m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(&model.GalleryRow{ID: id, UserID: userID, Name: "sketches", ArtCount: 2}, nil)
	m.artRepo.EXPECT().DeleteGallery(mock.Anything, spec.GalleryRef{GalleryID: id, UserID: userID}).Return(paths, nil)
	m.uploadSvc.EXPECT().Delete(paths).Return()
	m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
		ActorID:    userID,
		Action:     audit.ActionGalleryDelete,
		TargetType: audit.TargetGallery,
		TargetID:   id.String(),
		Details:    `name="sketches" art=2 files=4`,
		SubjectID:  userID,
	}).Return(nil)

	// when
	err := svc.DeleteGallery(context.Background(), id, userID)

	// then
	require.NoError(t, err)
}

func TestGetGallery_Failures(t *testing.T) {
	id := uuid.New()
	viewerID := uuid.New()

	cases := []struct {
		name    string
		given   func(m *testMocks)
		wantErr error
	}{
		{
			name: "gallery lookup failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "missing gallery maps to not found",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(nil, nil)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "art listing failure propagates",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(&model.GalleryRow{ID: id, Name: "G"}, nil)
				m.artRepo.EXPECT().
					ListArtInGallery(mock.Anything, spec.GalleryArtFilter{GalleryID: id, ViewerID: viewerID, Limit: 10}).
					Return(nil, 0, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "tag lookup failure propagates instead of rendering untagged art",
			given: func(m *testMocks) {
				m.artRepo.EXPECT().GetGalleryByID(mock.Anything, id).Return(&model.GalleryRow{ID: id, Name: "G"}, nil)
				m.artRepo.EXPECT().
					ListArtInGallery(mock.Anything, spec.GalleryArtFilter{GalleryID: id, ViewerID: viewerID, Limit: 10}).
					Return([]model.ArtRow{{ID: uuid.New()}}, 1, nil)
				m.artRepo.EXPECT().GetTagsBatch(mock.Anything, mock.Anything).Return(nil, errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m)

			// when
			_, _, _, err := svc.GetGallery(context.Background(), id, viewerID, bounds.NewPage(10, 0))

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
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
	m.artRepo.EXPECT().ListArtInGallery(mock.Anything, spec.GalleryArtFilter{GalleryID: id, ViewerID: viewerID, Limit: 10}).Return(rows, 1, nil)
	m.artRepo.EXPECT().GetTagsBatch(mock.Anything, []uuid.UUID{artID}).Return(nil, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")

	// when
	gallery, arts, total, err := svc.GetGallery(context.Background(), id, viewerID, bounds.NewPage(10, 0))

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
	m.artRepo.EXPECT().ListGalleriesByUser(mock.Anything, userID).Return(nil, errDB)

	// when
	_, err := svc.ListUserGalleries(context.Background(), userID)

	// then
	require.ErrorIs(t, err, errDB)
}

func TestListUserGalleries_PreviewsOnlyWithoutACover(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	bareID := uuid.New()

	rows := []model.GalleryRow{
		{ID: bareID, Name: "G", ArtCount: 3, CoverArtID: nil},
		{ID: uuid.New(), Name: "C", ArtCount: 3, CoverArtID: new(uuid.New()), CoverImageURL: "/u/cover.png"},
	}
	m.artRepo.EXPECT().ListGalleriesByUser(mock.Anything, userID).Return(rows, nil)
	m.settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://example.com")
	m.artRepo.EXPECT().
		GetGalleryPreviewImages(mock.Anything, spec.GalleryPreviewFilter{GalleryID: bareID, Limit: 3}).
		Return([]model.PreviewImage{{ImageURL: "/u/p.png"}}, nil)

	// when
	got, err := svc.ListUserGalleries(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Len(t, got[0].PreviewImages, 1)
	assert.Empty(t, got[1].PreviewImages)
	assert.NotEmpty(t, got[1].CoverThumbnailURL)
}

func TestListUserGalleries_AFailedPreviewLookupIsSurfaced(t *testing.T) {
	// given
	svc, m := newTestService(t)
	userID := uuid.New()
	m.artRepo.EXPECT().ListGalleriesByUser(mock.Anything, userID).Return([]model.GalleryRow{{ID: uuid.New(), Name: "G", ArtCount: 3}}, nil)
	m.artRepo.EXPECT().GetGalleryPreviewImages(mock.Anything, mock.Anything).Return(nil, errDB)

	// when
	got, err := svc.ListUserGalleries(context.Background(), userID)

	// then
	require.ErrorIs(t, err, errDB)
	assert.Nil(t, got)
}

func TestListAllGalleries(t *testing.T) {
	cases := []struct {
		name    string
		corner  string
		rows    []model.GalleryRow
		repoErr error
	}{
		{name: "lists the corner's galleries", corner: "umineko", rows: []model.GalleryRow{{ID: uuid.New(), Name: "G", ArtCount: 0}}},
		{name: "repo failure propagates", corner: "general", repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			m.artRepo.EXPECT().ListAllGalleries(mock.Anything, tc.corner).Return(tc.rows, tc.repoErr)

			// when
			got, err := svc.ListAllGalleries(context.Background(), tc.corner)

			// then
			require.ErrorIs(t, err, tc.repoErr)
			assert.Len(t, got, len(tc.rows))
		})
	}
}

func TestSetArtGallery(t *testing.T) {
	galleryID := uuid.New()

	cases := []struct {
		name      string
		galleryID *uuid.UUID
		repoErr   error
	}{
		{name: "assigns the gallery", galleryID: &galleryID},
		{name: "unassigning propagates a repo failure", galleryID: nil, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			artID := uuid.New()
			userID := uuid.New()
			m.artRepo.EXPECT().SetGallery(mock.Anything, spec.ArtGalleryAssignment{ArtID: artID, UserID: userID, GalleryID: tc.galleryID}).Return(tc.repoErr)

			// when
			err := svc.SetArtGallery(context.Background(), artID, userID, tc.galleryID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}
