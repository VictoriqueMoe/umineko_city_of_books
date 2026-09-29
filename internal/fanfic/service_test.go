package fanfic

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
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
	fanficparams "umineko_city_of_books/internal/fanfic/params"
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
		fanficRepo     *repository.MockFanficRepository
		fanficComments *dao.MockCommentDAO[uuid.UUID]
		userRepo       *repository.MockUserRepository
		auditRepo      *repository.MockAuditLogRepository
		authz          *authz.MockService
		blockSvc       *block.MockService
		notifSvc       *notification.MockService
		uploadSvc      *upload.MockService
		settingsSvc    *settings.MockService
	}

	givenFunc func(m *testMocks, id, userID uuid.UUID)

	fanficState int
)

const (
	fanficNotLookedUp fanficState = iota
	fanficMissing
	fanficHiddenDraft
	fanficOwn
	fanficOthers
	fanficOthersBlockLookupFails

	savedImageURL = "/uploads/fanfics/x.png"
)

var (
	errDB            = errors.New("db")
	fanficMissingRow = errors.Join(errors.New("get author"), dao.ErrNotFound)
)

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		fanficRepo:     repository.NewMockFanficRepository(t),
		fanficComments: dao.NewMockCommentDAO[uuid.UUID](t),
		userRepo:       repository.NewMockUserRepository(t),
		auditRepo:      repository.NewMockAuditLogRepository(t),
		authz:          authz.NewMockService(t),
		blockSvc:       block.NewMockService(t),
		notifSvc:       notification.NewMockService(t),
		uploadSvc:      upload.NewMockService(t),
		settingsSvc:    settings.NewMockService(t),
	}
	m.uploadSvc.EXPECT().FullDiskPath(mock.Anything).Return("/tmp/does-not-exist-xyz.png").Maybe()

	mentionSvc := mention.NewService(m.userRepo, m.blockSvc, m.notifSvc, dao.CommentDAOs{
		ByID: map[string]dao.CommentDAO[uuid.UUID]{string(mention.KindFanficComment): m.fanficComments},
	})
	svc := NewService(m.fanficRepo, m.userRepo, m.auditRepo, m.authz, m.blockSvc, m.notifSvc, mentionSvc, m.uploadSvc, media.NewProcessor(1), m.settingsSvc, contentfilter.New(), nil).(*service)

	return svc, m
}

func expectImageSave(m *testMocks, err error) {
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
	m.uploadSvc.EXPECT().
		SaveImage(mock.Anything, "fanfics", mock.Anything, int64(100), int64(1000), mock.Anything).
		Return(savedImageURL, err)
}

func expectListBatches(m *testMocks, ids []uuid.UUID) {
	m.fanficRepo.EXPECT().GetGenresBatch(mock.Anything, ids).Return(nil, nil)
	m.fanficRepo.EXPECT().GetTagsBatch(mock.Anything, ids).Return(nil, nil)
	m.fanficRepo.EXPECT().GetCharactersBatch(mock.Anything, ids).Return(nil, nil)
}

func expectHiddenDraft(m *testMocks, fanficID, viewerID uuid.UUID) {
	m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: fanficID, ViewerID: viewerID}).
		Return(&model.FanficRow{ID: fanficID, UserID: uuid.New(), Status: "draft"}, nil)
	m.authz.EXPECT().Can(mock.Anything, viewerID, authz.PermEditAnyTheory).Return(false)
}

func expectFanficState(m *testMocks, state fanficState, fanficID, userID uuid.UUID, blocked bool) {
	lookup := spec.FanficLookup{ID: fanficID, ViewerID: userID}

	switch state {
	case fanficMissing:
		m.fanficRepo.EXPECT().GetByID(mock.Anything, lookup).Return(nil, nil)
	case fanficHiddenDraft:
		expectHiddenDraft(m, fanficID, userID)
	case fanficOwn, fanficOthers:
		authorID := uuid.New()
		if state == fanficOwn {
			authorID = userID
		}

		m.fanficRepo.EXPECT().GetByID(mock.Anything, lookup).Return(&model.FanficRow{ID: fanficID, UserID: authorID, Status: "in_progress"}, nil)
		m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(blocked, nil)
	case fanficOthersBlockLookupFails:
		authorID := uuid.New()
		m.fanficRepo.EXPECT().GetByID(mock.Anything, lookup).Return(&model.FanficRow{ID: fanficID, UserID: authorID, Status: "in_progress"}, nil)
		m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, errDB)
	}
}

func expectFailingActorLookup(m *testMocks, userID uuid.UUID) <-chan struct{} {
	looked := make(chan struct{})
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).
		RunAndReturn(func(context.Context, uuid.UUID, ...*sql.Tx) (*model.User, error) {
			close(looked)

			return nil, errors.New("stop goroutine")
		})

	return looked
}

func fanficCommentDelete(id, actorID, authorID uuid.UUID, asAdmin bool, action audit.Action) spec.FanficCommentDelete {
	return spec.FanficCommentDelete{
		CommentDeletion: spec.CommentDeletion{CommentID: id, UserID: actorID, AsAdmin: asAdmin},
		Audit:           audit.NewEntry{ActorID: actorID, Action: action, TargetType: audit.TargetFanficComment, TargetID: id.String(), SubjectID: authorID},
	}
}

func TestFanficFieldValidation(t *testing.T) {
	elevenTags := make([]string, 11)
	for i := range elevenTags {
		elevenTags[i] = string(rune('a' + i))
	}

	cases := []struct {
		name    string
		title   string
		genres  []string
		tags    []string
		rating  string
		wantErr error
	}{
		{name: "blank title", title: "   ", wantErr: ErrEmptyTitle},
		{name: "more than two genres", title: "t", genres: []string{"a", "b", "c"}, wantErr: ErrTooManyGenres},
		{name: "more than ten tags", title: "t", tags: elevenTags, wantErr: ErrTooManyTags},
		{name: "a 31-rune tag", title: "t", tags: []string{strings.Repeat("a", 31)}, wantErr: ErrTagTooLong},
		{name: "unknown rating", title: "t", rating: "X", wantErr: ErrInvalidRating},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			createSvc, _ := newTestService(t)
			updateSvc, m := newTestService(t)
			fanficID := uuid.New()
			userID := uuid.New()
			m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(userID, nil)

			// when
			_, createErr := createSvc.CreateFanfic(context.Background(), userID, dto.CreateFanficRequest{Title: tc.title, Genres: tc.genres, Tags: tc.tags, Rating: tc.rating})
			updateErr := updateSvc.UpdateFanfic(context.Background(), fanficID, userID, dto.UpdateFanficRequest{Title: tc.title, Genres: tc.genres, Tags: tc.tags, Rating: tc.rating})

			// then
			require.ErrorIs(t, createErr, tc.wantErr)
			require.ErrorIs(t, updateErr, tc.wantErr)
		})
	}
}

func TestCreateFanfic(t *testing.T) {
	characters := []dto.FanficCharacter{
		{CharacterID: "", CharacterName: "  Piece  "},
		{CharacterID: "existing", CharacterName: "Existing"},
	}

	cases := []struct {
		name    string
		req     dto.CreateFanficRequest
		want    spec.NewFanficWithDetails
		repoErr error
	}{
		{
			name: "fields are trimmed, blank and duplicate tags dropped and an unknown status defaulted",
			req: dto.CreateFanficRequest{
				Title:    " Title ",
				Summary:  " sum ",
				Series:   " My Series ",
				Language: " English ",
				Status:   "garbage",
				Tags:     []string{"  a ", "A", "a", " a ", "b", "", "B ", "c"},
			},
			want: spec.NewFanficWithDetails{
				NewFanfic: spec.NewFanfic{Title: "Title", Summary: "sum", Series: "My Series", Rating: "K", Language: "English", Status: "in_progress"},
				Tags:      []string{"a", "b", "c"},
			},
		},
		{
			name: "a body becomes the first chapter with its word count",
			req:  dto.CreateFanficRequest{Title: "Title", Status: "draft", Body: "<p>hello <b>world</b></p>", Characters: characters, Rating: "M"},
			want: spec.NewFanficWithDetails{
				NewFanfic:    spec.NewFanfic{Title: "Title", Series: "Umineko", Rating: "M", Language: "English", Status: "draft"},
				Characters:   characters,
				FirstChapter: &spec.NewChapter{Number: 1, Body: "<p>hello <b>world</b></p>", WordCount: 2},
			},
		},
		{
			name: "the first chapter is sanitised like every other chapter",
			req:  dto.CreateFanficRequest{Title: "Title", Body: `<p onclick="steal()">hello</p><script>alert(1)</script>`},
			want: spec.NewFanficWithDetails{
				NewFanfic:    spec.NewFanfic{Title: "Title", Series: "Umineko", Rating: "K", Language: "English", Status: "in_progress"},
				FirstChapter: &spec.NewChapter{Number: 1, Body: "<p>hello</p>", WordCount: 1},
			},
		},
		{
			name: "a body that sanitises to nothing creates no first chapter",
			req:  dto.CreateFanficRequest{Title: "Title", Body: "<script>alert(1)</script>"},
			want: spec.NewFanficWithDetails{
				NewFanfic: spec.NewFanfic{Title: "Title", Series: "Umineko", Rating: "K", Language: "English", Status: "in_progress"},
			},
		},
		{
			name:    "blank optional fields take their defaults and a repo error propagates",
			req:     dto.CreateFanficRequest{Title: "Title"},
			want:    spec.NewFanficWithDetails{NewFanfic: spec.NewFanfic{Title: "Title", Series: "Umineko", Rating: "K", Language: "English", Status: "in_progress"}},
			repoErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			createdID := uuid.New()
			want := tc.want
			want.UserID = userID

			var created *model.FanficRow
			if tc.repoErr == nil {
				created = &model.FanficRow{ID: createdID}
			}
			m.fanficRepo.EXPECT().CreateWithDetails(mock.Anything, want).Return(created, tc.repoErr)

			// when
			id, err := svc.CreateFanfic(context.Background(), userID, tc.req)

			// then
			if tc.repoErr != nil {
				require.ErrorIs(t, err, tc.repoErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, createdID, id)
		})
	}
}

func TestCreateFanfic_MentionFanOut(t *testing.T) {
	tests := []struct {
		name       string
		summary    string
		body       string
		status     string
		wantFanOut bool
	}{
		{name: "a mention in the summary notifies the named user", summary: "written for @alice", status: "complete", wantFanOut: true},
		{name: "a mention in the first chapter notifies the named user", body: "a gift to @alice", status: "complete", wantFanOut: true},
		{name: "a draft notifies nobody", summary: "written for @alice", status: "draft"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			fanficID := uuid.New()
			mentionedID := uuid.New()

			m.fanficRepo.EXPECT().
				CreateWithDetails(mock.Anything, mock.Anything).
				Return(&model.FanficRow{ID: fanficID}, nil)

			var wg sync.WaitGroup
			var mentioned dto.NotifyParams

			if tt.wantFanOut {
				wg.Add(1)
				m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
				m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)
				m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
					RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
						mentioned = p
						wg.Done()

						return nil
					})
			}

			req := dto.CreateFanficRequest{Title: "Title", Summary: tt.summary, Body: tt.body, Status: tt.status}

			// when
			_, err := svc.CreateFanfic(context.Background(), userID, req)

			// then
			require.NoError(t, err)
			wg.Wait()

			if !tt.wantFanOut {
				m.userRepo.AssertNotCalled(t, "GetByUsernames", mock.Anything, mock.Anything)
				return
			}

			assert.Equal(t, dto.NotifMention, mentioned.Type)
			assert.Equal(t, mentionedID, mentioned.RecipientID)
			assert.Equal(t, fanficID, mentioned.ReferenceID)
			assert.Equal(t, "fanfic", mentioned.ReferenceType)
			assert.Equal(t, "/fanfiction/"+fanficID.String(), mentioned.EmailLink)
		})
	}
}

func TestGetFanfic_Unavailable(t *testing.T) {
	cases := []struct {
		name    string
		given   givenFunc
		wantErr error
	}{
		{
			name: "repo error propagates",
			given: func(m *testMocks, id, viewerID uuid.UUID) {
				m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: id, ViewerID: viewerID}).Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "a missing fanfic is not found",
			given: func(m *testMocks, id, viewerID uuid.UUID) {
				m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: id, ViewerID: viewerID}).Return(nil, nil)
			},
			wantErr: ErrNotFound,
		},
		{name: "someone else's draft is hidden from a reader who cannot edit it", given: expectHiddenDraft, wantErr: ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			viewerID := uuid.New()
			tc.given(m, id, viewerID)

			// when
			_, err := svc.GetFanfic(context.Background(), id, viewerID, "")

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetFanfic_SurfacesAFailedLookup(t *testing.T) {
	cases := []struct {
		name   string
		failAt int
	}{
		{name: "the genres", failAt: 0},
		{name: "the tags", failAt: 1},
		{name: "the characters", failAt: 2},
		{name: "the chapters", failAt: 3},
		{name: "the viewer's block list", failAt: 4},
		{name: "the comments", failAt: 5},
		{name: "the comment media", failAt: 6},
		{name: "the block check against the author", failAt: 7},
		{name: "the reading progress", failAt: 8},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as empty", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			authorID := uuid.New()
			viewerID := uuid.New()
			commentID := uuid.New()
			m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: id, ViewerID: viewerID}).Return(&model.FanficRow{ID: id, UserID: authorID, Status: "complete"}, nil)

			m.fanficRepo.EXPECT().GetGenres(mock.Anything, id).Return(nil, fanficFailAt(tc.failAt, 0))
			if tc.failAt >= 1 {
				m.fanficRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, fanficFailAt(tc.failAt, 1))
			}
			if tc.failAt >= 2 {
				m.fanficRepo.EXPECT().GetCharacters(mock.Anything, id).Return(nil, fanficFailAt(tc.failAt, 2))
			}
			if tc.failAt >= 3 {
				m.fanficRepo.EXPECT().ListChapters(mock.Anything, id).Return(nil, fanficFailAt(tc.failAt, 3))
			}
			if tc.failAt >= 4 {
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, fanficFailAt(tc.failAt, 4))
			}
			if tc.failAt >= 5 {
				m.fanficRepo.EXPECT().GetComments(mock.Anything, mock.Anything).Return([]model.CommentRow{{ID: commentID, UserID: authorID}}, 1, fanficFailAt(tc.failAt, 5))
			}
			if tc.failAt >= 6 {
				m.fanficRepo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{commentID}).Return(nil, fanficFailAt(tc.failAt, 6))
			}
			if tc.failAt >= 7 {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(false, fanficFailAt(tc.failAt, 7))
			}
			if tc.failAt >= 8 {
				m.fanficRepo.EXPECT().GetReadingProgress(mock.Anything, spec.FanficUserRef{UserID: viewerID, FanficID: id}).Return(0, fanficFailAt(tc.failAt, 8))
			}

			// when
			got, err := svc.GetFanfic(context.Background(), id, viewerID, "")

			// then
			require.ErrorIs(t, err, errDB)
			assert.Nil(t, got)
		})
	}
}

func fanficFailAt(failAt int, step int) error {
	if failAt == step {
		return errDB
	}

	return nil
}

func TestGetFanfic_BuildsDetail(t *testing.T) {
	cases := []struct {
		name          string
		anonymous     bool
		viewerHash    string
		withChapter   bool
		withComment   bool
		progress      int
		wantViewCount int
	}{
		{name: "a first view bumps the view count and loads the reader's progress", viewerHash: "hash", progress: 2, wantViewCount: 4},
		{name: "an anonymous reader skips the block check", anonymous: true, wantViewCount: 3},
		{name: "chapters and threaded comments are included", withChapter: true, withComment: true, wantViewCount: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			authorID := uuid.New()
			viewerID := uuid.New()
			if tc.anonymous {
				viewerID = uuid.Nil
			}

			var chapters []model.FanficChapterSummaryRow
			if tc.withChapter {
				chapters = []model.FanficChapterSummaryRow{{ID: uuid.New(), ChapterNum: 1, Title: "Ch1"}}
			}

			var comments []model.CommentRow
			if tc.withComment {
				commentID := uuid.New()
				comments = []model.CommentRow{{ID: commentID, UserID: authorID, Body: "hi"}}
				m.fanficRepo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{commentID}).Return(nil, nil)
			}

			if tc.viewerHash != "" {
				m.fanficRepo.EXPECT().RecordView(mock.Anything, spec.ViewRecord{TargetID: id, ViewerHash: tc.viewerHash}).Return(true, nil)
			}

			if !tc.anonymous {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, viewerID, authorID).Return(false, nil)
			}

			m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: id, ViewerID: viewerID}).
				Return(&model.FanficRow{ID: id, UserID: authorID, Status: "complete", ViewCount: 3}, nil)
			m.fanficRepo.EXPECT().GetGenres(mock.Anything, id).Return(nil, nil)
			m.fanficRepo.EXPECT().GetTags(mock.Anything, id).Return(nil, nil)
			m.fanficRepo.EXPECT().GetCharacters(mock.Anything, id).Return(nil, nil)
			m.fanficRepo.EXPECT().ListChapters(mock.Anything, id).Return(chapters, nil)
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
			m.fanficRepo.EXPECT().
				GetComments(mock.Anything, spec.CommentQuery[uuid.UUID]{TargetID: id, ViewerID: viewerID, Limit: 500, Offset: 0, ExcludeUserIDs: []uuid.UUID(nil)}).
				Return(comments, 0, nil)
			m.fanficRepo.EXPECT().GetReadingProgress(mock.Anything, spec.FanficUserRef{UserID: viewerID, FanficID: id}).Return(tc.progress, nil)

			// when
			got, err := svc.GetFanfic(context.Background(), id, viewerID, tc.viewerHash)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantViewCount, got.ViewCount)
			assert.Equal(t, tc.progress, got.ReadingProgress)
			assert.False(t, got.ViewerBlocked)
			assert.Len(t, got.Chapters, len(chapters))
			assert.Len(t, got.Comments, len(comments))
		})
	}
}

func TestUpdateFanfic(t *testing.T) {
	characters := []dto.FanficCharacter{{CharacterID: "", CharacterName: "Custom"}}

	cases := []struct {
		name    string
		req     dto.UpdateFanficRequest
		given   givenFunc
		wantErr error
	}{
		{
			name: "a missing fanfic is not found",
			req:  dto.UpdateFanficRequest{Title: "T"},
			given: func(m *testMocks, id, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, fanficMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "an author lookup failure is surfaced, not reported as not found",
			req:  dto.UpdateFanficRequest{Title: "T"},
			given: func(m *testMocks, id, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "an admin edit is refused when the before-snapshot for the audit cannot be read",
			req:  dto.UpdateFanficRequest{Title: "T"},
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
				m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: id, ViewerID: userID}).Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "someone else without edit rights is refused",
			req:  dto.UpdateFanficRequest{Title: "T"},
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(false)
			},
			wantErr: ErrNotAuthor,
		},
		{
			name: "an admin edit is audited with the changed fields",
			req:  dto.UpdateFanficRequest{Title: " T ", Summary: " s ", Series: " ser ", Language: " en ", Status: "in_progress"},
			given: func(m *testMocks, id, userID uuid.UUID) {
				authorID := uuid.New()
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(true)
				m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: id, ViewerID: userID}).
					Return(&model.FanficRow{ID: id, UserID: authorID, Title: "old", Series: "ser", Rating: "K", Language: "en", Status: "in_progress"}, nil)
				m.fanficRepo.EXPECT().UpdateWithDetails(mock.Anything, spec.FanficUpdateWithDetails{
					FanficUpdate: spec.FanficUpdate{ID: id, UserID: userID, Title: "T", Summary: "s", Series: "ser", Rating: "K", Language: "en", Status: "in_progress", AsAdmin: true},
				}).Return(nil)
				m.auditRepo.EXPECT().
					Create(mock.Anything, audit.NewEntry{ActorID: userID, Action: audit.ActionFanficUpdateAdmin, TargetType: audit.TargetFanfic, TargetID: id.String(), Details: "changed=title,summary", SubjectID: authorID}).
					Return(nil)
			},
		},
		{
			name: "the author's edit passes characters and defaults into the spec",
			req:  dto.UpdateFanficRequest{Title: "T", Characters: characters},
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
				m.fanficRepo.EXPECT().UpdateWithDetails(mock.Anything, spec.FanficUpdateWithDetails{
					FanficUpdate: spec.FanficUpdate{ID: id, UserID: userID, Title: "T", Series: "Umineko", Rating: "K", Language: "English"},
					Characters:   characters,
				}).Return(nil)
			},
		},
		{
			name: "repo error propagates",
			req:  dto.UpdateFanficRequest{Title: "T"},
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
				m.fanficRepo.EXPECT().UpdateWithDetails(mock.Anything, spec.FanficUpdateWithDetails{
					FanficUpdate: spec.FanficUpdate{ID: id, UserID: userID, Title: "T", Series: "Umineko", Rating: "K", Language: "English"},
				}).Return(errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			tc.given(m, id, userID)

			// when
			err := svc.UpdateFanfic(context.Background(), id, userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDeleteFanfic(t *testing.T) {
	paths := []string{"/uploads/images/cover.png", "/uploads/images/cover_thumb.png", "/uploads/images/comment.png"}

	cases := []struct {
		name    string
		given   givenFunc
		wantErr error
	}{
		{
			name: "a missing fanfic is not found",
			given: func(m *testMocks, id, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, fanficMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "an author lookup failure is surfaced, not reported as not found",
			given: func(m *testMocks, id, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.Nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "an admin delete is audited as an admin action",
			given: func(m *testMocks, id, userID uuid.UUID) {
				authorID := uuid.New()
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(true)
				m.fanficRepo.EXPECT().DeleteFanfic(mock.Anything, spec.FanficDelete{ID: id, UserID: userID, AsAdmin: true}).Return(nil, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: userID, Action: audit.ActionFanficDeleteAdmin, TargetType: audit.TargetFanfic, TargetID: id.String(), SubjectID: authorID}).Return(nil)
				m.uploadSvc.EXPECT().Delete().Once()
			},
		},
		{
			name: "the owner deleting is not an admin action and unlinks the returned paths",
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
				m.fanficRepo.EXPECT().DeleteFanfic(mock.Anything, spec.FanficDelete{ID: id, UserID: userID}).Return(paths, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: userID, Action: audit.ActionFanficDelete, TargetType: audit.TargetFanfic, TargetID: id.String(), SubjectID: userID}).Return(nil)
				m.uploadSvc.EXPECT().Delete(paths).Once()
			},
		},
		{
			name: "a repo error unlinks nothing",
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(userID, nil)
				m.fanficRepo.EXPECT().DeleteFanfic(mock.Anything, spec.FanficDelete{ID: id, UserID: userID}).Return([]string{"/uploads/images/cover.png"}, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "someone else without delete rights gets the repo's refusal",
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(uuid.New(), nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(false)
				m.fanficRepo.EXPECT().DeleteFanfic(mock.Anything, spec.FanficDelete{ID: id, UserID: userID}).Return(nil, errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			tc.given(m, id, userID)

			// when
			err := svc.DeleteFanfic(context.Background(), id, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			m.uploadSvc.AssertExpectations(t)
			if tc.wantErr != nil {
				m.uploadSvc.AssertNotCalled(t, "Delete", mock.Anything)
			}
		})
	}
}

func TestListFanfics(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
	}{
		{name: "a long summary is clipped and the page is echoed"},
		{name: "repo error propagates", repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			id := uuid.New()
			params := fanficparams.ListParams{Limit: 10, Offset: 5}

			var rows []model.FanficRow
			if tc.repoErr == nil {
				rows = []model.FanficRow{{ID: id, UserID: uuid.New(), Title: "A", Summary: strings.Repeat("x", 250)}}
				expectListBatches(m, []uuid.UUID{id})
			}
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, nil)
			m.fanficRepo.EXPECT().List(mock.Anything, spec.FanficListFilter{ViewerID: viewerID, Params: params, ExcludeUserIDs: []uuid.UUID(nil)}).
				Return(rows, len(rows), tc.repoErr)

			// when
			got, err := svc.ListFanfics(context.Background(), viewerID, params)

			// then
			require.ErrorIs(t, err, tc.repoErr)
			if tc.repoErr != nil {
				return
			}
			assert.Equal(t, 1, got.Total)
			assert.Equal(t, 10, got.Limit)
			assert.Equal(t, 5, got.Offset)
			assert.Len(t, got.Fanfics[0].Summary, 203)
		})
	}
}

func TestListFanfics_SurfacesAFailedLookup(t *testing.T) {
	cases := []struct {
		name   string
		failAt int
	}{
		{name: "the viewer's block list", failAt: 0},
		{name: "the genres", failAt: 1},
		{name: "the tags", failAt: 2},
		{name: "the characters", failAt: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name+" failing is surfaced, not rendered as empty", func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			viewerID := uuid.New()
			params := fanficparams.ListParams{Limit: 10}

			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, viewerID).Return(nil, fanficFailAt(tc.failAt, 0))
			if tc.failAt >= 1 {
				m.fanficRepo.EXPECT().List(mock.Anything, mock.Anything).Return([]model.FanficRow{{ID: uuid.New()}}, 1, nil)
				m.fanficRepo.EXPECT().GetGenresBatch(mock.Anything, mock.Anything).Return(nil, fanficFailAt(tc.failAt, 1))
			}
			if tc.failAt >= 2 {
				m.fanficRepo.EXPECT().GetTagsBatch(mock.Anything, mock.Anything).Return(nil, fanficFailAt(tc.failAt, 2))
			}
			if tc.failAt >= 3 {
				m.fanficRepo.EXPECT().GetCharactersBatch(mock.Anything, mock.Anything).Return(nil, fanficFailAt(tc.failAt, 3))
			}

			// when
			got, err := svc.ListFanfics(context.Background(), viewerID, params)

			// then
			require.ErrorIs(t, err, errDB)
			assert.Nil(t, got)
		})
	}
}

func TestListFanficsForUser(t *testing.T) {
	cases := []struct {
		name       string
		favourites bool
		repoErr    error
	}{
		{name: "fanfics by a user"},
		{name: "fanfics by a user repo error propagates", repoErr: errDB},
		{name: "a user's favourites", favourites: true},
		{name: "a user's favourites repo error propagates", favourites: true, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			viewerID := uuid.New()
			id := uuid.New()
			filter := spec.FanficUserListFilter{UserID: userID, ViewerID: viewerID, Limit: 10, Offset: 0}

			var rows []model.FanficRow
			if tc.repoErr == nil {
				rows = []model.FanficRow{{ID: id, UserID: userID, Title: "A"}}
				expectListBatches(m, []uuid.UUID{id})
			}

			list := svc.ListFanficsByUser
			if tc.favourites {
				list = svc.ListFavourites
				m.fanficRepo.EXPECT().ListFavourites(mock.Anything, filter).Return(rows, len(rows), tc.repoErr)
			} else {
				m.fanficRepo.EXPECT().ListByUser(mock.Anything, filter).Return(rows, len(rows), tc.repoErr)
			}

			// when
			got, err := list(context.Background(), userID, viewerID, bounds.NewPage(10, 0))

			// then
			require.ErrorIs(t, err, tc.repoErr)
			if tc.repoErr != nil {
				return
			}
			assert.Equal(t, 1, got.Total)
			assert.Len(t, got.Fanfics, 1)
		})
	}
}

func TestUploadCoverImage(t *testing.T) {
	cases := []struct {
		name         string
		lookupErr    error
		otherAuthor  bool
		saves        bool
		saveErr      error
		updates      bool
		updateErr    error
		cancelledCtx bool
		wantURL      string
		wantErr      error
		wantErrText  string
	}{
		{name: "a missing target is not found", lookupErr: fanficMissingRow, wantErr: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", lookupErr: errDB, wantErr: errDB},
		{name: "someone else without edit rights is refused", otherAuthor: true, wantErrText: "not the fanfic author"},
		{name: "a failed save propagates", saves: true, saveErr: errDB, wantErr: errDB},
		{name: "a failed cover update propagates", saves: true, updates: true, updateErr: errDB, wantErr: errDB},
		{name: "the cover is saved even when the request context is already cancelled", saves: true, updates: true, cancelledCtx: true, wantURL: savedImageURL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelledCtx {
				cancel()
			}

			authorID := userID
			if tc.otherAuthor {
				authorID = uuid.New()
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
			}
			m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, tc.lookupErr)

			if tc.saves {
				expectImageSave(m, tc.saveErr)
			}

			if tc.updates {
				m.fanficRepo.EXPECT().UpdateCoverImage(mock.Anything, spec.FanficCoverUpdate{ID: id, ImageURL: savedImageURL, ThumbnailURL: ""}).Return(tc.updateErr)
			}

			// when
			url, err := svc.UploadCoverImage(ctx, id, userID, "image/png", 100, bytes.NewReader(nil))

			// then
			if tc.wantErrText != "" {
				require.ErrorContains(t, err, tc.wantErrText)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
			assert.Equal(t, tc.wantURL, url)
		})
	}
}

func TestRemoveCoverImage(t *testing.T) {
	cases := []struct {
		name        string
		lookupErr   error
		otherAuthor bool
		clears      bool
		wantErr     error
		wantErrText string
	}{
		{name: "an author lookup error propagates", lookupErr: errDB, wantErr: errDB},
		{name: "someone else without edit rights is refused", otherAuthor: true, wantErrText: "not authorised"},
		{name: "the author clears both image URLs", clears: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()

			authorID := userID
			if tc.otherAuthor {
				authorID = uuid.New()
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyPost).Return(false)
			}
			m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, tc.lookupErr)

			if tc.clears {
				m.fanficRepo.EXPECT().UpdateCoverImage(mock.Anything, spec.FanficCoverUpdate{ID: id, ImageURL: "", ThumbnailURL: ""}).Return(nil)
			}

			// when
			err := svc.RemoveCoverImage(context.Background(), id, userID)

			// then
			if tc.wantErrText != "" {
				require.ErrorContains(t, err, tc.wantErrText)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestCreateChapter(t *testing.T) {
	cases := []struct {
		name        string
		req         dto.CreateChapterRequest
		lookupErr   error
		otherAuthor bool
		numbered    bool
		nextNumber  int
		nextErr     error
		chapter     *spec.NewChapter
		createErr   error
		wantErr     error
	}{
		{name: "a missing fanfic is not found", req: dto.CreateChapterRequest{Body: "hi"}, lookupErr: fanficMissingRow, wantErr: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", req: dto.CreateChapterRequest{Body: "hi"}, lookupErr: errDB, wantErr: errDB},
		{name: "someone other than the author is refused", req: dto.CreateChapterRequest{Body: "hi"}, otherAuthor: true, wantErr: ErrNotAuthor},
		{name: "a blank body is rejected", req: dto.CreateChapterRequest{Body: "  "}, wantErr: ErrEmptyBody},
		{name: "a next chapter number error propagates", req: dto.CreateChapterRequest{Body: "hi"}, numbered: true, nextErr: errDB, wantErr: errDB},
		{
			name:       "a create error propagates",
			req:        dto.CreateChapterRequest{Title: " Title ", Body: "body"},
			numbered:   true,
			nextNumber: 2,
			chapter:    &spec.NewChapter{Number: 2, Title: "Title", Body: "body", WordCount: 1},
			createErr:  errDB,
			wantErr:    errDB,
		},
		{
			name:       "the author's chapter takes the next number",
			req:        dto.CreateChapterRequest{Title: "T", Body: "body"},
			numbered:   true,
			nextNumber: 3,
			chapter:    &spec.NewChapter{Number: 3, Title: "T", Body: "body", WordCount: 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			fanficID := uuid.New()
			userID := uuid.New()
			chapterID := uuid.New()

			authorID := userID
			if tc.otherAuthor {
				authorID = uuid.New()
			}
			m.fanficRepo.EXPECT().GetAuthorID(mock.Anything, fanficID).Return(authorID, tc.lookupErr)

			if tc.numbered {
				m.fanficRepo.EXPECT().GetNextChapterNumber(mock.Anything, fanficID).Return(tc.nextNumber, tc.nextErr)
			}

			wantID := uuid.Nil
			if tc.chapter != nil {
				want := *tc.chapter
				want.FanficID = fanficID

				var created *model.FanficChapterRow
				if tc.createErr == nil {
					created = &model.FanficChapterRow{ID: chapterID}
					wantID = chapterID
				}
				m.fanficRepo.EXPECT().CreateChapterWithCount(mock.Anything, want).Return(created, tc.createErr)
			}

			// when
			id, err := svc.CreateChapter(context.Background(), fanficID, userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, wantID, id)
		})
	}
}

func TestGetChapter_NotServed(t *testing.T) {
	cases := []struct {
		name              string
		member            bool
		published         bool
		given             givenFunc
		wantErr           error
		skipsChapterFetch bool
	}{
		{
			name:      "a chapter lookup error propagates",
			published: true,
			given: func(m *testMocks, fanficID, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetChapter(mock.Anything, spec.FanficChapterLookup{FanficID: fanficID, ChapterNumber: 1}).Return(nil, errDB)
			},
			wantErr: errDB,
		},
		{
			name:      "a missing chapter is not found",
			published: true,
			given: func(m *testMocks, fanficID, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetChapter(mock.Anything, spec.FanficChapterLookup{FanficID: fanficID, ChapterNumber: 1}).Return(nil, nil)
			},
			wantErr: ErrNotFound,
		},
		{
			name:      "a chapter count error propagates",
			published: true,
			given: func(m *testMocks, fanficID, _ uuid.UUID) {
				m.fanficRepo.EXPECT().GetChapter(mock.Anything, spec.FanficChapterLookup{FanficID: fanficID, ChapterNumber: 1}).Return(&model.FanficChapterRow{ChapterNum: 1}, nil)
				m.fanficRepo.EXPECT().GetChapterCount(mock.Anything, fanficID).Return(0, errDB)
			},
			wantErr: errDB,
		},
		{
			name: "an unknown fanfic is not found",
			given: func(m *testMocks, fanficID, viewerID uuid.UUID) {
				m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: fanficID, ViewerID: viewerID}).Return(nil, nil)
			},
			wantErr:           ErrNotFound,
			skipsChapterFetch: true,
		},
		{name: "a draft is hidden from an anonymous reader", given: expectHiddenDraft, wantErr: ErrNotFound, skipsChapterFetch: true},
		{name: "a draft is hidden from another member", member: true, given: expectHiddenDraft, wantErr: ErrNotFound, skipsChapterFetch: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			fanficID := uuid.New()
			viewerID := uuid.Nil
			if tc.member {
				viewerID = uuid.New()
			}

			if tc.published {
				m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: fanficID, ViewerID: viewerID}).
					Return(&model.FanficRow{ID: fanficID, UserID: uuid.New(), Status: "in_progress"}, nil)
			}
			tc.given(m, fanficID, viewerID)

			// when
			resp, err := svc.GetChapter(context.Background(), fanficID, 1, viewerID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, resp)
			if tc.skipsChapterFetch {
				m.fanficRepo.AssertNotCalled(t, "GetChapter", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestGetChapter_Serves(t *testing.T) {
	cases := []struct {
		name          string
		viewer        string
		status        string
		chapterNum    int
		chapterCount  int
		body          string
		savesProgress bool
		wantPrev      bool
		wantNext      bool
	}{
		{name: "an anonymous reader gets both neighbours and no saved progress", viewer: "anonymous", status: "in_progress", chapterNum: 2, chapterCount: 3, body: "b", wantPrev: true, wantNext: true},
		{name: "a signed-in reader's progress is saved", viewer: "reader", status: "in_progress", chapterNum: 1, chapterCount: 1, savesProgress: true},
		{name: "a draft chapter is served to its author", viewer: "author", status: "draft", chapterNum: 1, chapterCount: 2, body: "<p>body</p>", savesProgress: true, wantNext: true},
		{name: "a draft chapter is served to an editor", viewer: "editor", status: "draft", chapterNum: 1, chapterCount: 1, body: "b", savesProgress: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			fanficID := uuid.New()
			ownerID := uuid.New()
			viewerID := uuid.New()
			switch tc.viewer {
			case "anonymous":
				viewerID = uuid.Nil
			case "author":
				viewerID = ownerID
			case "editor":
				m.authz.EXPECT().Can(mock.Anything, viewerID, authz.PermEditAnyTheory).Return(true)
			}

			m.fanficRepo.EXPECT().GetByID(mock.Anything, spec.FanficLookup{ID: fanficID, ViewerID: viewerID}).
				Return(&model.FanficRow{ID: fanficID, UserID: ownerID, Status: tc.status}, nil)
			m.fanficRepo.EXPECT().GetChapter(mock.Anything, spec.FanficChapterLookup{FanficID: fanficID, ChapterNumber: tc.chapterNum}).
				Return(&model.FanficChapterRow{ID: uuid.New(), ChapterNum: tc.chapterNum, Body: tc.body}, nil)
			m.fanficRepo.EXPECT().GetChapterCount(mock.Anything, fanficID).Return(tc.chapterCount, nil)
			if tc.savesProgress {
				m.fanficRepo.EXPECT().SetReadingProgress(mock.Anything, spec.FanficReadingProgress{UserID: viewerID, FanficID: fanficID, ChapterNumber: tc.chapterNum}).Return(nil)
			}

			// when
			resp, err := svc.GetChapter(context.Background(), fanficID, tc.chapterNum, viewerID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.body, resp.Body)
			assert.Equal(t, tc.wantPrev, resp.HasPrev)
			assert.Equal(t, tc.wantNext, resp.HasNext)
		})
	}
}

func TestUpdateChapter(t *testing.T) {
	cases := []struct {
		name         string
		req          dto.UpdateChapterRequest
		lookupErr    error
		otherAuthor  bool
		allowed      bool
		update       *spec.ChapterUpdate
		updateErr    error
		auditDetails string
		wantErr      error
	}{
		{name: "a missing chapter is not found", req: dto.UpdateChapterRequest{Body: "b"}, lookupErr: fanficMissingRow, wantErr: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", req: dto.UpdateChapterRequest{Body: "b"}, lookupErr: errDB, wantErr: errDB},
		{name: "someone else without edit rights is refused", req: dto.UpdateChapterRequest{Body: "b"}, otherAuthor: true, wantErr: ErrNotAuthor},
		{name: "a blank body is rejected", req: dto.UpdateChapterRequest{Body: " "}, wantErr: ErrEmptyBody},
		{
			name:      "an update error propagates",
			req:       dto.UpdateChapterRequest{Title: "T", Body: "body"},
			update:    &spec.ChapterUpdate{Title: "T", Body: "body", WordCount: 1},
			updateErr: errDB,
			wantErr:   errDB,
		},
		{
			name:         "an admin edit is audited with the title and word count",
			req:          dto.UpdateChapterRequest{Body: "b"},
			otherAuthor:  true,
			allowed:      true,
			update:       &spec.ChapterUpdate{Body: "b", WordCount: 1},
			auditDetails: "title=,word_count=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			chapterID := uuid.New()
			userID := uuid.New()

			authorID := userID
			if tc.otherAuthor {
				authorID = uuid.New()
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyTheory).Return(tc.allowed)
			}
			m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(authorID, tc.lookupErr)

			if tc.update != nil {
				want := *tc.update
				want.ID = chapterID
				m.fanficRepo.EXPECT().UpdateChapterWithCount(mock.Anything, want).Return(tc.updateErr)
			}

			if tc.auditDetails != "" {
				m.auditRepo.EXPECT().
					Create(mock.Anything, audit.NewEntry{ActorID: userID, Action: audit.ActionFanficChapterUpdateAdmin, TargetType: audit.TargetFanficChapter, TargetID: chapterID.String(), Details: tc.auditDetails, SubjectID: authorID}).
					Return(nil)
			}

			// when
			err := svc.UpdateChapter(context.Background(), chapterID, userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDeleteChapter(t *testing.T) {
	cases := []struct {
		name        string
		lookupErr   error
		otherAuthor bool
		allowed     bool
		deletes     bool
		deleteErr   error
		auditAction audit.Action
		wantErr     error
	}{
		{name: "a missing target is not found", lookupErr: fanficMissingRow, wantErr: ErrNotFound},
		{name: "an author lookup failure is surfaced, not reported as not found", lookupErr: errDB, wantErr: errDB},
		{name: "someone else without delete rights is refused", otherAuthor: true, wantErr: ErrNotAuthor},
		{name: "a delete error propagates", deletes: true, deleteErr: errDB, wantErr: errDB},
		{name: "an admin delete is audited as an admin action", otherAuthor: true, allowed: true, deletes: true, auditAction: audit.ActionFanficChapterDeleteAdmin},
		{name: "the author's delete is audited as a plain delete", deletes: true, auditAction: audit.ActionFanficChapterDelete},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			chapterID := uuid.New()
			userID := uuid.New()

			authorID := userID
			if tc.otherAuthor {
				authorID = uuid.New()
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyPost).Return(tc.allowed)
			}
			m.fanficRepo.EXPECT().GetChapterAuthorID(mock.Anything, chapterID).Return(authorID, tc.lookupErr)

			if tc.deletes {
				m.fanficRepo.EXPECT().DeleteChapterWithCount(mock.Anything, chapterID).Return(tc.deleteErr)
			}

			if tc.auditAction != "" {
				m.auditRepo.EXPECT().
					Create(mock.Anything, audit.NewEntry{ActorID: userID, Action: tc.auditAction, TargetType: audit.TargetFanficChapter, TargetID: chapterID.String(), SubjectID: authorID}).
					Return(nil)
			}

			// when
			err := svc.DeleteChapter(context.Background(), chapterID, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestFavourite(t *testing.T) {
	cases := []struct {
		name     string
		fanfic   fanficState
		blocked  bool
		writes   bool
		writeErr error
		notifies bool
		wantErr  error
	}{
		{name: "a missing fanfic is not found", fanfic: fanficMissing, wantErr: ErrNotFound},
		{name: "someone else's hidden draft is not found", fanfic: fanficHiddenDraft, wantErr: ErrNotFound},
		{name: "a blocked pair is refused", fanfic: fanficOthers, blocked: true, wantErr: block.ErrUserBlocked},
		{name: "a failed block lookup refuses the favourite", fanfic: fanficOthersBlockLookupFails, wantErr: errDB},
		{name: "a repo error propagates", fanfic: fanficOthers, writes: true, writeErr: errDB, wantErr: errDB},
		{name: "favouriting your own fanfic notifies nobody", fanfic: fanficOwn, writes: true},
		{name: "favouriting someone else's fanfic looks up the actor to notify the author", fanfic: fanficOthers, writes: true, notifies: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			fanficID := uuid.New()
			userID := uuid.New()
			expectFanficState(m, tc.fanfic, fanficID, userID, tc.blocked)

			if tc.writes {
				m.fanficRepo.EXPECT().Favourite(mock.Anything, spec.FanficUserRef{UserID: userID, FanficID: fanficID}).Return(tc.writeErr)
			}

			var actorLooked <-chan struct{}
			if tc.notifies {
				actorLooked = expectFailingActorLookup(m, userID)
			}

			// when
			err := svc.Favourite(context.Background(), userID, fanficID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			if !tc.writes {
				m.fanficRepo.AssertNotCalled(t, "Favourite", mock.Anything, mock.Anything)
			}
			if tc.notifies {
				<-actorLooked
			}
		})
	}
}

func TestUnfavouriteAndUnlikeComment(t *testing.T) {
	cases := []struct {
		name    string
		unlike  bool
		repoErr error
	}{
		{name: "unfavourite removes the favourite"},
		{name: "an unfavourite repo error propagates", repoErr: errDB},
		{name: "unlike removes the like", unlike: true},
		{name: "an unlike repo error propagates", unlike: true, repoErr: errDB},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			userID := uuid.New()
			targetID := uuid.New()

			undo := svc.Unfavourite
			if tc.unlike {
				undo = svc.UnlikeComment
				m.fanficRepo.EXPECT().UnlikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: targetID}).Return(tc.repoErr)
			} else {
				m.fanficRepo.EXPECT().Unfavourite(mock.Anything, spec.FanficUserRef{UserID: userID, FanficID: targetID}).Return(tc.repoErr)
			}

			// when
			err := undo(context.Background(), userID, targetID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestLookupsDelegateToTheRepo(t *testing.T) {
	// given
	svc, m := newTestService(t)
	m.fanficRepo.EXPECT().GetLanguages(mock.Anything).Return([]string{"en"}, nil)
	m.fanficRepo.EXPECT().GetSeries(mock.Anything).Return([]string{"s"}, nil)
	m.fanficRepo.EXPECT().SearchOCCharacters(mock.Anything, "").Return([]string{"Alice", "Bob"}, nil)
	m.fanficRepo.EXPECT().SearchOCCharacters(mock.Anything, "alice").Return([]string{"Alice"}, nil)

	// when
	languages, languagesErr := svc.GetLanguages(context.Background())
	series, seriesErr := svc.GetSeries(context.Background())
	everyone, everyoneErr := svc.SearchOCCharacters(context.Background(), "   ")
	alices, alicesErr := svc.SearchOCCharacters(context.Background(), " alice ")

	// then
	require.NoError(t, errors.Join(languagesErr, seriesErr, everyoneErr, alicesErr))
	assert.Equal(t, []string{"en"}, languages)
	assert.Equal(t, []string{"s"}, series)
	assert.Equal(t, []string{"Alice", "Bob"}, everyone)
	assert.Equal(t, []string{"Alice"}, alices)
}

func TestCreateComment(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		fanfic   fanficState
		blocked  bool
		writes   bool
		writeErr error
		notifies bool
		wantErr  error
	}{
		{name: "a blank body is rejected", body: " ", fanfic: fanficNotLookedUp, wantErr: ErrEmptyBody},
		{name: "a missing fanfic is not found", body: "hi", fanfic: fanficMissing, wantErr: ErrNotFound},
		{name: "someone else's hidden draft is not found", body: "hi", fanfic: fanficHiddenDraft, wantErr: ErrNotFound},
		{name: "a blocked pair is refused", body: "hi", fanfic: fanficOthers, blocked: true, wantErr: block.ErrUserBlocked},
		{name: "a failed block lookup refuses the comment", body: "hi", fanfic: fanficOthersBlockLookupFails, wantErr: errDB},
		{name: "a comment write error propagates", body: "hi", fanfic: fanficOthers, writes: true, writeErr: errDB, wantErr: errDB},
		{name: "the new comment's id is returned and the actor looked up to notify the author", body: "hi", fanfic: fanficOthers, writes: true, notifies: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			fanficID := uuid.New()
			userID := uuid.New()
			commentID := uuid.New()
			expectFanficState(m, tc.fanfic, fanficID, userID, tc.blocked)

			if tc.writes {
				var created *model.CommentRow
				if tc.writeErr == nil {
					created = &model.CommentRow{ID: commentID}
				}
				m.fanficComments.EXPECT().
					CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: fanficID, ParentID: nil, UserID: userID, Body: tc.body}).
					Return(created, tc.writeErr)
			}

			var actorLooked <-chan struct{}
			if tc.notifies {
				actorLooked = expectFailingActorLookup(m, userID)
			}

			// when
			id, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: tc.body})

			// then
			require.ErrorIs(t, err, tc.wantErr)
			if !tc.writes {
				m.fanficComments.AssertNotCalled(t, "CreateComment", mock.Anything, mock.Anything)
			}
			if tc.notifies {
				assert.Equal(t, commentID, id)
				<-actorLooked
			}
		})
	}
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	commentID := uuid.New()
	mentionedID := uuid.New()

	expectFanficState(m, fanficOthers, fanficID, userID, false)
	m.fanficComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: fanficID, ParentID: nil, UserID: userID, Body: "look at this @alice"}).
		Return(&model.CommentRow{ID: commentID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, mentionedID).Return(false, nil)

	var wg sync.WaitGroup
	wg.Add(2)

	var mentioned dto.NotifyParams
	var commented dto.NotifyParams
	var mu sync.Mutex
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			mu.Lock()
			if p.Type == dto.NotifMention {
				mentioned = p
			}
			if p.Type == dto.NotifFanficCommented {
				commented = p
			}
			mu.Unlock()
			wg.Done()

			return nil
		})

	// when
	_, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: "look at this @alice"})

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, "fanfic_comment:"+commentID.String(), mentioned.ReferenceType)
	assert.Equal(t, "/fanfiction/"+fanficID.String()+"#comment-"+commentID.String(), mentioned.EmailLink)
	assert.Equal(t, "/fanfiction/"+fanficID.String()+"#comment-"+commentID.String(), commented.EmailLink, "the author's email must open the fanfiction page, not the API route")
}

func TestFavourite_EmailLinkOpensTheFanfictionPage(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	expectFanficState(m, fanficOthers, fanficID, userID, false)
	m.fanficRepo.EXPECT().Favourite(mock.Anything, spec.FanficUserRef{UserID: userID, FanficID: fanficID}).Return(nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
	sent := make(chan dto.NotifyParams, 1)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			sent <- p

			return nil
		})

	// when
	err := svc.Favourite(context.Background(), userID, fanficID)

	// then
	require.NoError(t, err)
	got := <-sent
	assert.Equal(t, dto.NotifFanficFavourited, got.Type)
	assert.Equal(t, "/fanfiction/"+fanficID.String(), got.EmailLink)
}

func TestLikeComment_EmailLinkOpensTheFanfictionPage(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	authorID := uuid.New()
	commentID := uuid.New()
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(false, nil)
	m.fanficRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(nil)
	m.fanficRepo.EXPECT().GetCommentEntityID(mock.Anything, commentID).Return(fanficID, nil)
	sent := make(chan dto.NotifyParams, 1)
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			sent <- p

			return nil
		})

	// when
	err := svc.LikeComment(context.Background(), userID, commentID)

	// then
	require.NoError(t, err)
	got := <-sent
	assert.Equal(t, dto.NotifFanficCommentLiked, got.Type)
	assert.Equal(t, "/fanfiction/"+fanficID.String()+"#comment-"+commentID.String(), got.EmailLink)
}

func TestCreateComment_ReplyEmailLinkOpensTheFanfictionPage(t *testing.T) {
	// given
	svc, m := newTestService(t)
	fanficID := uuid.New()
	userID := uuid.New()
	parentID := uuid.New()
	parentAuthorID := uuid.New()
	commentID := uuid.New()
	expectFanficState(m, fanficOthers, fanficID, userID, false)
	m.fanficComments.EXPECT().
		CreateComment(mock.Anything, spec.NewComment[uuid.UUID]{TargetID: fanficID, ParentID: &parentID, UserID: userID, Body: "hi"}).
		Return(&model.CommentRow{ID: commentID}, nil)
	m.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID, DisplayName: "Battler"}, nil)
	m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, parentID).Return(parentAuthorID, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	var replied dto.NotifyParams
	var mu sync.Mutex
	m.notifSvc.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			mu.Lock()
			if p.Type == dto.NotifFanficCommentReply {
				replied = p
			}
			mu.Unlock()
			wg.Done()

			return nil
		})

	// when
	_, err := svc.CreateComment(context.Background(), fanficID, userID, dto.CreateCommentRequest{Body: "hi", ParentID: &parentID})

	// then
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, parentAuthorID, replied.RecipientID)
	assert.Equal(t, "/fanfiction/"+fanficID.String()+"#comment-"+commentID.String(), replied.EmailLink)
}

func TestUpdateComment(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		given   givenFunc
		audited bool
		wantErr error
	}{
		{name: "a blank body is rejected", body: "  ", wantErr: ErrEmptyBody},
		{
			name: "an admin edit is audited",
			body: "hi",
			given: func(m *testMocks, id, userID uuid.UUID) {
				authorID := uuid.New()
				m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(true)
				m.fanficRepo.EXPECT().UpdateCommentBody(mock.Anything, spec.CommentUpdate{CommentID: id, UserID: userID, Body: "hi", AsAdmin: true}).Return(nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{ActorID: userID, Action: audit.ActionFanficCommentUpdateAdmin, TargetType: audit.TargetFanficComment, TargetID: id.String(), SubjectID: authorID}).Return(nil)
			},
			audited: true,
		},
		{
			name: "the author's edit writes no audit row",
			body: "hi",
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(userID, nil)
				m.fanficRepo.EXPECT().UpdateCommentBody(mock.Anything, spec.CommentUpdate{CommentID: id, UserID: userID, Body: "hi"}).Return(nil)
			},
		},
		{
			name: "someone else without edit rights gets the repo's refusal",
			body: "hi",
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(uuid.New(), nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermEditAnyComment).Return(false)
				m.fanficRepo.EXPECT().UpdateCommentBody(mock.Anything, spec.CommentUpdate{CommentID: id, UserID: userID, Body: "hi"}).Return(errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			if tc.given != nil {
				tc.given(m, id, userID)
			}

			// when
			err := svc.UpdateComment(context.Background(), id, userID, dto.UpdateCommentRequest{Body: tc.body})

			// then
			require.ErrorIs(t, err, tc.wantErr)
			if !tc.audited {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteComment(t *testing.T) {
	paths := []string{"/uploads/images/comment.png", "/uploads/images/comment_thumb.png"}

	cases := []struct {
		name    string
		given   givenFunc
		wantErr error
	}{
		{
			name: "an admin delete is recorded as an admin action",
			given: func(m *testMocks, id, userID uuid.UUID) {
				authorID := uuid.New()
				m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(true)
				m.fanficRepo.EXPECT().DeleteCommentWithAudit(mock.Anything, fanficCommentDelete(id, userID, authorID, true, audit.ActionFanficCommentDeleteAdmin)).Return(nil, nil)
				m.uploadSvc.EXPECT().Delete().Once()
			},
		},
		{
			name: "deleting your own comment is not an admin action and unlinks the returned paths",
			given: func(m *testMocks, id, userID uuid.UUID) {
				m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(userID, nil)
				m.fanficRepo.EXPECT().DeleteCommentWithAudit(mock.Anything, fanficCommentDelete(id, userID, userID, false, audit.ActionFanficCommentDelete)).Return(paths, nil)
				m.uploadSvc.EXPECT().Delete(paths).Once()
			},
		},
		{
			name: "someone else without delete rights gets the repo's refusal",
			given: func(m *testMocks, id, userID uuid.UUID) {
				authorID := uuid.New()
				m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, id).Return(authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, userID, authz.PermDeleteAnyComment).Return(false)
				m.fanficRepo.EXPECT().DeleteCommentWithAudit(mock.Anything, fanficCommentDelete(id, userID, authorID, false, audit.ActionFanficCommentDelete)).Return(nil, errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			id := uuid.New()
			userID := uuid.New()
			tc.given(m, id, userID)

			// when
			err := svc.DeleteComment(context.Background(), id, userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			m.uploadSvc.AssertExpectations(t)
			if tc.wantErr != nil {
				m.uploadSvc.AssertNotCalled(t, "Delete", mock.Anything)
			}
		})
	}
}

func TestLikeComment(t *testing.T) {
	cases := []struct {
		name       string
		lookupErr  error
		ownComment bool
		blocked    bool
		blockErr   error
		likes      bool
		likeErr    error
		notifies   bool
		wantErr    error
	}{
		{name: "an author lookup error propagates", lookupErr: errDB, wantErr: errDB},
		{name: "a blocked pair is refused", blocked: true, wantErr: block.ErrUserBlocked},
		{name: "a failed block lookup refuses the like", blockErr: errDB, wantErr: errDB},
		{name: "a like error propagates", likes: true, likeErr: errDB, wantErr: errDB},
		{name: "liking your own comment notifies nobody", ownComment: true, likes: true},
		{name: "liking someone else's comment looks up the fanfic to notify the author", likes: true, notifies: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID := uuid.New()
			userID := uuid.New()

			authorID := uuid.New()
			if tc.ownComment {
				authorID = userID
			}
			m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, tc.lookupErr)

			if tc.lookupErr == nil {
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, userID, authorID).Return(tc.blocked, tc.blockErr)
			}

			if tc.likes {
				m.fanficRepo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: userID, CommentID: commentID}).Return(tc.likeErr)
			}

			fanficLooked := make(chan struct{})
			if tc.notifies {
				m.fanficRepo.EXPECT().GetCommentEntityID(mock.Anything, commentID).
					RunAndReturn(func(context.Context, uuid.UUID, ...*sql.Tx) (uuid.UUID, error) {
						close(fanficLooked)

						return uuid.Nil, errors.New("stop goroutine")
					})
			}

			// when
			err := svc.LikeComment(context.Background(), userID, commentID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			if tc.notifies {
				<-fanficLooked
			}
		})
	}
}

func TestUploadCommentMedia(t *testing.T) {
	cases := []struct {
		name         string
		lookupErr    error
		otherAuthor  bool
		saves        bool
		saveErr      error
		records      bool
		recordErr    error
		cancelledCtx bool
		wantResp     *dto.PostMediaResponse
		wantErr      error
		wantErrText  string
	}{
		{name: "a missing comment is not found", lookupErr: fanficMissingRow, wantErr: ErrNotFound},
		{name: "a comment author lookup failure is surfaced, not reported as not found", lookupErr: errDB, wantErr: errDB},
		{name: "someone other than the comment author is refused", otherAuthor: true, wantErrText: "not the comment author"},
		{name: "a failed save propagates", saves: true, saveErr: errDB, wantErr: errDB},
		{name: "a failed media row insert propagates", saves: true, records: true, recordErr: errDB, wantErr: errDB},
		{
			name:         "the media is recorded even when the request context is already cancelled",
			saves:        true,
			records:      true,
			cancelledCtx: true,
			wantResp:     &dto.PostMediaResponse{ID: 42, MediaURL: savedImageURL, MediaType: "image", Filename: "photo.png"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			commentID := uuid.New()
			userID := uuid.New()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelledCtx {
				cancel()
			}

			authorID := userID
			if tc.otherAuthor {
				authorID = uuid.New()
			}
			m.fanficRepo.EXPECT().GetCommentAuthorID(mock.Anything, commentID).Return(authorID, tc.lookupErr)

			if tc.saves {
				expectImageSave(m, tc.saveErr)
			}

			if tc.records {
				m.fanficRepo.EXPECT().
					AddCommentMedia(mock.Anything, spec.NewMedia{TargetID: commentID, MediaURL: savedImageURL, MediaType: "image", Filename: "photo.png"}).
					Return(int64(42), tc.recordErr)
			}

			// when
			resp, err := svc.UploadCommentMedia(ctx, commentID, userID, "image/png", "photo.png", 100, bytes.NewReader(nil), false)

			// then
			if tc.wantErrText != "" {
				require.ErrorContains(t, err, tc.wantErrText)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
			assert.Equal(t, tc.wantResp, resp)
		})
	}
}
