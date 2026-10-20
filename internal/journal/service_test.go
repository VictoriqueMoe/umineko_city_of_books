package journal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	testMocks struct {
		repo         *repository.MockJournalRepository
		commentSvc   *comment.MockService
		userRepo     *repository.MockUserRepository
		auditRepo    *repository.MockAuditLogRepository
		authz        *authz.MockService
		blockSvc     *block.MockService
		notifService *notification.MockService
		uploadSvc    *upload.MockService
		settingsSvc  *settings.MockService
		activity     *sidebar.MockActivityBumper
	}

	fixture struct {
		journalID uuid.UUID
		entryID   uuid.UUID
		commentID uuid.UUID
		userID    uuid.UUID
		authorID  uuid.UUID
	}
)

var (
	errBoom       = errors.New("boom")
	errRollback   = errors.New("rollback failed")
	errMissingRow = errors.Join(errors.New("no row"), daoerr.ErrNotFound)
)

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		repo:         repository.NewMockJournalRepository(t),
		commentSvc:   comment.NewMockService(t),
		userRepo:     repository.NewMockUserRepository(t),
		auditRepo:    repository.NewMockAuditLogRepository(t),
		authz:        authz.NewMockService(t),
		blockSvc:     block.NewMockService(t),
		notifService: notification.NewMockService(t),
		uploadSvc:    upload.NewMockService(t),
		settingsSvc:  settings.NewMockService(t),
		activity:     sidebar.NewMockActivityBumper(t),
	}

	mentionSvc := mention.NewService(m.userRepo, m.blockSvc, m.notifService)
	svc := NewService(m.repo, m.userRepo, m.auditRepo, m.authz, m.blockSvc, m.notifService, mentionSvc, m.commentSvc, m.uploadSvc, &media.Processor{}, m.settingsSvc, contentfilter.New(), nil, nil, m.activity).(*service)

	return svc, m
}

func newFixture() fixture {
	return fixture{
		journalID: uuid.New(),
		entryID:   uuid.New(),
		commentID: uuid.New(),
		userID:    uuid.New(),
		authorID:  uuid.New(),
	}
}

func expectOpenJournal(m *testMocks, f fixture) {
	m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
	m.repo.EXPECT().IsArchived(mock.Anything, f.journalID).Return(false, nil)
}

func allowBackgroundFanOut(m *testMocks, journalID, actorID uuid.UUID) {
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTitle(mock.Anything, journalID).Return("j", nil).Maybe()
	m.repo.EXPECT().GetFollowerIDs(mock.Anything, journalID).Return(nil, nil).Maybe()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, actorID).Return(nil, nil).Maybe()
	m.notifService.EXPECT().NotifyMany(mock.Anything, mock.Anything).Return().Maybe()
}

func expectMentionOfAlice(m *testMocks, actorID, mentionedID uuid.UUID) {
	m.userRepo.EXPECT().GetByID(mock.Anything, actorID).Return(&model.User{ID: actorID, DisplayName: "Battler"}, nil)
	m.userRepo.EXPECT().GetByUsernames(mock.Anything, []string{"alice"}).Return([]model.User{{ID: mentionedID}}, nil)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, actorID, mentionedID).Return(false, nil)
}

func captureNotifications(m *testMocks, count int) <-chan dto.NotifyParams {
	sent := make(chan dto.NotifyParams, count)
	m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, p dto.NotifyParams) error {
			sent <- p

			return nil
		}).
		Times(count)

	return sent
}

func awaitNotifications(t *testing.T, sent <-chan dto.NotifyParams, count int) map[dto.NotificationType]dto.NotifyParams {
	t.Helper()

	got := make(map[dto.NotificationType]dto.NotifyParams, count)
	for range count {
		select {
		case p := <-sent:
			got[p.Type] = p
		case <-time.After(5 * time.Second):
			require.FailNow(t, "timed out waiting for notifications", "received %d of %d", len(got), count)
		}
	}

	return got
}

func withMediaRecorder(t *testing.T, svc *service, m *testMocks) *upload.MockMediaRecorder {
	t.Helper()

	recorder := upload.NewMockMediaRecorder(t)
	svc.uploader = recorder

	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1024 * 1024).Maybe()
	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()

	return recorder
}

func storedURL(filename string) string {
	return "/uploads/journals/" + filename
}

func storedMedia(targetID uuid.UUID, filename string, sortOrder int, isSpoiler bool) spec.NewMedia {
	return spec.NewMedia{
		TargetID:  targetID,
		MediaURL:  storedURL(filename),
		MediaType: media.MediaTypeImage,
		Filename:  filename,
		SortOrder: sortOrder,
		IsSpoiler: isSpoiler,
	}
}

func expectStored(recorder *upload.MockMediaRecorder, filename string, isSpoiler bool) {
	recorder.EXPECT().
		SaveAndRecord(mock.Anything, "journals", "image/png", filename, mock.Anything, mock.Anything, isSpoiler, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, _ string, name string, _ int64, _ io.Reader, spoiler bool, addFn media.AddFn, _ media.UpdateURLFn, _ media.UpdateURLFn) (*dto.PostMediaResponse, error) {
			id, err := addFn(storedURL(name), media.MediaTypeImage, "", name, 0)
			if err != nil {
				return nil, err
			}

			return &dto.PostMediaResponse{ID: int(id), MediaURL: storedURL(name), MediaType: media.MediaTypeImage, Filename: name, IsSpoiler: spoiler}, nil
		}).
		Once()
}

func expectStoreFailure(recorder *upload.MockMediaRecorder, filename string, isSpoiler bool) {
	recorder.EXPECT().
		SaveAndRecord(mock.Anything, "journals", "image/png", filename, mock.Anything, mock.Anything, isSpoiler, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errBoom).
		Once()
}

func TestCreateJournal(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name:    "a blank title is rejected before any lookup",
			title:   "   ",
			given:   func(*testMocks, fixture) {},
			wantErr: ErrEmptyTitle,
		},
		{
			name:  "with no daily limit the journal is created without counting and bumps the journals section",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(0)
				m.repo.EXPECT().Create(mock.Anything, spec.NewJournal{UserID: f.userID, Title: "Title", Work: "umineko"}).Return(&dto.JournalResponse{ID: f.journalID}, nil)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
		},
		{
			name:  "under the daily limit the journal is created and bumps the journals section",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
				m.repo.EXPECT().CountUserJournalsToday(mock.Anything, f.userID).Return(2, nil)
				m.repo.EXPECT().Create(mock.Anything, spec.NewJournal{UserID: f.userID, Title: "Title", Work: "umineko"}).Return(&dto.JournalResponse{ID: f.journalID}, nil)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
		},
		{
			name:  "at the daily limit the author is rate limited",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
				m.repo.EXPECT().CountUserJournalsToday(mock.Anything, f.userID).Return(5, nil)
			},
			wantErr: ErrRateLimited,
		},
		{
			name:  "a count error bubbles up",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
				m.repo.EXPECT().CountUserJournalsToday(mock.Anything, f.userID).Return(0, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name:  "a create error bubbles up",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(0)
				m.repo.EXPECT().Create(mock.Anything, spec.NewJournal{UserID: f.userID, Title: "Title", Work: "umineko"}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			got, err := svc.CreateJournal(context.Background(), f.userID, dto.CreateJournalRequest{Title: tc.title, Work: "umineko"})

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if tc.wantErr == nil {
				assert.Equal(t, f.journalID, got)
			}
		})
	}
}

func TestCreateJournal_MentionInTheTitleNotifiesTheNamedUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	f := newFixture()
	mentionedID := uuid.New()
	req := dto.CreateJournalRequest{Title: "reading along with @alice", Work: "umineko"}

	m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(0)
	m.repo.EXPECT().Create(mock.Anything, spec.NewJournal{UserID: f.userID, Title: req.Title, Work: req.Work}).Return(&dto.JournalResponse{ID: f.journalID}, nil)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
	expectMentionOfAlice(m, f.userID, mentionedID)
	sent := captureNotifications(m, 1)

	// when
	_, err := svc.CreateJournal(context.Background(), f.userID, req)

	// then
	require.NoError(t, err)

	mentioned := awaitNotifications(t, sent, 1)[dto.NotifMention]
	assert.Equal(t, dto.NotifMention, mentioned.Type)
	assert.Equal(t, mentionedID, mentioned.RecipientID)
	assert.Equal(t, f.journalID, mentioned.ReferenceID)
	assert.Equal(t, "journal", mentioned.ReferenceType)
	assert.Equal(t, "/journals/"+f.journalID.String(), mentioned.EmailLink)
}

func TestGetJournalDetail_Failures(t *testing.T) {
	tests := []struct {
		name    string
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name: "a missing journal is not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(nil, nil)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "a lookup error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a comment thread error bubbles up instead of hiding the comments",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}}, nil)
				m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindJournal, ID: f.journalID}, f.userID).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a latest entry media error bubbles up",
			given: func(m *testMocks, f fixture) {
				latest := 1
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}, LatestEntryNumber: &latest}, nil)
				m.commentSvc.EXPECT().Thread(mock.Anything, mock.Anything, f.userID).Return([]dto.CommentResponse{}, nil)
				m.repo.EXPECT().ListEntries(mock.Anything, f.journalID).Return(nil, nil)
				m.repo.EXPECT().GetEntry(mock.Anything, spec.JournalEntryLookup{JournalID: f.journalID, EntryNumber: 1}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				m.repo.EXPECT().GetMediaBatch(mock.Anything, []uuid.UUID{f.entryID}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			_, err := svc.GetJournalDetail(context.Background(), f.journalID, f.userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestGetJournalDetail(t *testing.T) {
	// given
	svc, m := newTestService(t)
	f := newFixture()
	thread := []dto.CommentResponse{{ID: f.commentID, Body: "hi", IsAuthor: true}}
	m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}}, nil)
	m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindJournal, ID: f.journalID}, f.userID).Return(thread, nil).Once()
	m.repo.EXPECT().ListEntries(mock.Anything, f.journalID).Return(nil, nil)

	// when
	got, err := svc.GetJournalDetail(context.Background(), f.journalID, f.userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, thread, got.Comments)
}

func TestListJournals(t *testing.T) {
	tests := []struct {
		name    string
		listErr error
	}{
		{name: "every param and the viewer's blocks reach the query and the page echoes back"},
		{name: "a repo error bubbles up", listErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			blockedIDs := []uuid.UUID{uuid.New()}
			journals := []dto.JournalResponse{{ID: f.journalID}}
			p := params.ListParams{Sort: "old", Work: "umineko", AuthorID: f.authorID, Search: "beatrice", IncludeArchived: true, Limit: 10, Offset: 5}
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(blockedIDs, nil)
			m.repo.EXPECT().List(mock.Anything, spec.JournalQuery{
				Sort:            "old",
				Work:            "umineko",
				AuthorID:        f.authorID,
				Search:          "beatrice",
				IncludeArchived: true,
				Limit:           10,
				Offset:          5,
				ViewerID:        f.userID,
				ExcludeUserIDs:  blockedIDs,
			}).Return(journals, 1, tc.listErr)

			// when
			got, err := svc.ListJournals(context.Background(), p, f.userID)

			// then
			require.ErrorIs(t, err, tc.listErr)

			if tc.listErr != nil {
				assert.Nil(t, got)
				return
			}

			assert.Equal(t, 1, got.Total)
			assert.Equal(t, 10, got.Limit)
			assert.Equal(t, 5, got.Offset)
			assert.Equal(t, journals, got.Journals)
		})
	}
}

func TestListJournalsByUser(t *testing.T) {
	// given
	svc, m := newTestService(t)
	f := newFixture()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(nil, nil)
	m.repo.EXPECT().List(mock.Anything, spec.JournalQuery{
		Sort:            "new",
		AuthorID:        f.authorID,
		IncludeArchived: true,
		Limit:           10,
		Offset:          5,
		ViewerID:        f.userID,
		ExcludeUserIDs:  []uuid.UUID(nil),
	}).Return([]dto.JournalResponse{}, 0, nil)

	// when
	_, err := svc.ListJournalsByUser(context.Background(), f.authorID, f.userID, 10, 5)

	// then
	require.NoError(t, err)
}

func TestListFollowedByUser(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
		listErr    error
	}{
		{name: "a valid page reaches the query and echoes back", limit: 25, offset: 10, wantLimit: 25, wantOffset: 10},
		{name: "a zero page falls back to the default limit", limit: 0, offset: 0, wantLimit: 20, wantOffset: 0},
		{name: "a repo error bubbles up", limit: 0, offset: 0, wantLimit: 20, wantOffset: 0, listErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			follower := uuid.New()
			viewer := uuid.New()
			m.repo.EXPECT().ListFollowedByUser(mock.Anything, spec.JournalFollowedQuery{
				FollowerID: follower,
				ViewerID:   viewer,
				Limit:      tc.wantLimit,
				Offset:     tc.wantOffset,
			}).Return([]dto.JournalResponse{}, 0, tc.listErr)

			// when
			got, err := svc.ListFollowedByUser(context.Background(), follower, viewer, bounds.NewPage(tc.limit, tc.offset))

			// then
			require.ErrorIs(t, err, tc.listErr)

			if tc.listErr != nil {
				return
			}

			assert.Equal(t, tc.wantLimit, got.Limit)
			assert.Equal(t, tc.wantOffset, got.Offset)
		})
	}
}

func TestUpdateJournal(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		given   func(m *testMocks, f fixture)
		wantErr error
		audited bool
	}{
		{
			name:    "a blank title is rejected before any lookup",
			title:   " ",
			given:   func(*testMocks, fixture) {},
			wantErr: ErrEmptyTitle,
		},
		{
			name:  "a missing journal is not found",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name:  "a failed author lookup is surfaced, not reported as not found",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name:  "an admin edit is refused when the before-snapshot for the audit cannot be read",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(true)
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name:  "the owner's edit is not audited",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().Update(mock.Anything, spec.JournalUpdate{ID: f.journalID, UserID: f.userID, Title: "Title", Work: "umineko"}).Return(nil)
			},
		},
		{
			name:  "an update error bubbles up",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().Update(mock.Anything, spec.JournalUpdate{ID: f.journalID, UserID: f.userID, Title: "Title", Work: "umineko"}).Return(errBoom)
			},
			wantErr: errBoom,
		},
		{
			name:  "an admin edit is audited with the changed fields",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(true)
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Title: "Old", Work: "umineko"}, nil)
				m.repo.EXPECT().Update(mock.Anything, spec.JournalUpdate{ID: f.journalID, UserID: f.userID, Title: "Title", Work: "umineko", AsAdmin: true}).Return(nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalUpdateAdmin,
					TargetType: audit.TargetJournal,
					TargetID:   f.journalID.String(),
					Details:    "changed=title",
					SubjectID:  f.authorID,
				}).Return(nil)
			},
			audited: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			err := svc.UpdateJournal(context.Background(), f.journalID, f.userID, dto.CreateJournalRequest{Title: tc.title, Work: "umineko"})

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if !tc.audited {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteJournal(t *testing.T) {
	tests := []struct {
		name    string
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name: "the owner's delete is audited and its media removed",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("Rokkenjima", nil)
				m.repo.EXPECT().DeleteWithMedia(mock.Anything, spec.JournalDeletion{ID: f.journalID, UserID: f.userID, AsAdmin: false}).Return([]string{"/u/comment.png"}, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalDelete,
					TargetType: audit.TargetJournal,
					TargetID:   f.journalID.String(),
					Details:    "title=Rokkenjima",
					SubjectID:  f.userID,
				}).Return(nil)
				m.uploadSvc.EXPECT().Delete([]string{"/u/comment.png"}).Return()
			},
		},
		{
			name: "an admin's delete is audited as an admin action against the author",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermDeleteAnyJournal).Return(true)
				m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("Rokkenjima", nil)
				m.repo.EXPECT().DeleteWithMedia(mock.Anything, spec.JournalDeletion{ID: f.journalID, UserID: f.userID, AsAdmin: true}).Return([]string{"/u/entry.png", "/u/entry-thumb.png"}, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalDeleteAdmin,
					TargetType: audit.TargetJournal,
					TargetID:   f.journalID.String(),
					Details:    "title=Rokkenjima",
					SubjectID:  f.authorID,
				}).Return(nil)
				m.uploadSvc.EXPECT().Delete([]string{"/u/entry.png", "/u/entry-thumb.png"}).Return()
			},
		},
		{
			name: "a delete error bubbles up before any audit",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("Rokkenjima", nil)
				m.repo.EXPECT().DeleteWithMedia(mock.Anything, spec.JournalDeletion{ID: f.journalID, UserID: f.userID, AsAdmin: false}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			err := svc.DeleteJournal(context.Background(), f.journalID, f.userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if tc.wantErr != nil {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestCreateComment_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		onEntry bool
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name: "a missing journal is not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "a failed author lookup is surfaced, not reported as not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "an archive check error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.repo.EXPECT().IsArchived(mock.Anything, f.journalID).Return(false, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "an archived journal takes no comments",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.repo.EXPECT().IsArchived(mock.Anything, f.journalID).Return(true, nil)
			},
			wantErr: ErrArchived,
		},
		{
			name:    "a failed entry lookup is surfaced",
			onEntry: true,
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name:    "a missing entry is not found",
			onEntry: true,
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(nil, nil)
			},
			wantErr: ErrEntryNotFound,
		},
		{
			name:    "an entry from another journal is a mismatch",
			onEntry: true,
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: uuid.New(), EntryNumber: 2}, nil)
			},
			wantErr: ErrEntryMismatch,
		},
		{
			name: "a comment the comment service refuses bubbles up",
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.commentSvc.EXPECT().Create(mock.Anything, mock.Anything, f.userID, mock.Anything, mock.Anything).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			var entryID *uuid.UUID
			if tc.onEntry {
				entryID = &f.entryID
			}

			// when
			got, err := svc.CreateComment(context.Background(), f.journalID, f.userID, dto.CreateJournalCommentRequest{Body: "hi", EntryID: entryID}, nil)

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, uuid.Nil, got)
			m.repo.AssertNotCalled(t, "UpdateLastAuthorActivity", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateComment_AReaderCommentReachesTheRightThreadWithoutRecordingActivity(t *testing.T) {
	tests := []struct {
		name       string
		onEntry    bool
		wantTarget func(f fixture) spec.CommentTarget
	}{
		{
			name: "a comment on the journal itself",
			wantTarget: func(f fixture) spec.CommentTarget {
				return spec.CommentTarget{Kind: spec.CommentKindJournal, ID: f.journalID}
			},
		},
		{
			name:    "a comment on one entry is addressed by the entry's number",
			onEntry: true,
			wantTarget: func(f fixture) spec.CommentTarget {
				return spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: f.journalID, EntryNumber: 4}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			parentID := uuid.New()
			expectOpenJournal(m, f)

			var entryID *uuid.UUID
			if tc.onEntry {
				entryID = &f.entryID
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 4}, nil)
			}

			want := dto.CreateCommentRequest{Body: "lovely", ParentID: &parentID}
			m.commentSvc.EXPECT().Create(mock.Anything, tc.wantTarget(f), f.userID, want, []upload.FileUpload(nil)).Return(f.commentID, nil).Once()

			// when
			got, err := svc.CreateComment(context.Background(), f.journalID, f.userID, dto.CreateJournalCommentRequest{Body: "lovely", ParentID: &parentID, EntryID: entryID}, nil)

			// then
			require.NoError(t, err)
			assert.Equal(t, f.commentID, got)
			m.repo.AssertNotCalled(t, "UpdateLastAuthorActivity", mock.Anything, mock.Anything)
			m.notifService.AssertNotCalled(t, "NotifyMany", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateComment_TheAuthorsCommentRecordsActivityAndTellsEligibleFollowers(t *testing.T) {
	tests := []struct {
		name        string
		onEntry     bool
		wantRefType string
		wantLink    string
	}{
		{
			name:        "an update on the journal itself",
			wantRefType: "journal_comment:%s",
			wantLink:    "/journals/%s#comment-%s",
		},
		{
			name:        "an update on one entry",
			onEntry:     true,
			wantRefType: "journal_entry_comment:4:%s",
			wantLink:    "/journals/%s/entry/4#comment-%s",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a journal followed by a reader, a blocked reader and its own author
			svc, m := newTestService(t)
			f := newFixture()
			f.authorID = f.userID
			readerID, blockedID := uuid.New(), uuid.New()
			expectOpenJournal(m, f)

			var entryID *uuid.UUID
			if tc.onEntry {
				entryID = &f.entryID
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 4}, nil)
			}

			m.commentSvc.EXPECT().Create(mock.Anything, mock.Anything, f.userID, mock.Anything, mock.Anything).Return(f.commentID, nil).Once()
			m.repo.EXPECT().UpdateLastAuthorActivity(mock.Anything, f.journalID).Return(nil).Once()
			m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("A Cat Box Diary", nil)
			m.repo.EXPECT().GetFollowerIDs(mock.Anything, f.journalID).Return([]uuid.UUID{readerID, blockedID, f.userID}, nil)
			m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return([]uuid.UUID{blockedID}, nil)
			m.userRepo.EXPECT().GetByID(mock.Anything, f.userID).Return(&model.User{ID: f.userID, DisplayName: "Ange"}, nil)

			fanOut := make(chan []dto.NotifyParams, 1)
			m.notifService.EXPECT().NotifyMany(mock.Anything, mock.Anything).
				Run(func(_ context.Context, params []dto.NotifyParams) { fanOut <- params }).
				Return().
				Once()

			// when
			got, err := svc.CreateComment(context.Background(), f.journalID, f.userID, dto.CreateJournalCommentRequest{Body: "today's entry", EntryID: entryID}, nil)

			// then only the unblocked reader hears about the update
			require.NoError(t, err)
			assert.Equal(t, f.commentID, got)

			select {
			case params := <-fanOut:
				require.Len(t, params, 1)
				assert.Equal(t, readerID, params[0].RecipientID)
				assert.Equal(t, dto.NotifJournalUpdate, params[0].Type)
				assert.Equal(t, f.journalID, params[0].ReferenceID)
				assert.Equal(t, fmt.Sprintf(tc.wantRefType, f.commentID), params[0].ReferenceType)
				assert.Equal(t, fmt.Sprintf(tc.wantLink, f.journalID, f.commentID), params[0].EmailLink)
				assert.Equal(t, "A Cat Box Diary", params[0].EmailTitle)
			case <-time.After(5 * time.Second):
				t.Fatal("the follower fan-out never ran")
			}
		})
	}
}

func TestCreateComment_AFailedActivityRecordRollsTheAuthorsCommentBack(t *testing.T) {
	tests := []struct {
		name      string
		deleteErr error
		wantErrs  []error
	}{
		{name: "the comment is removed and the activity failure reported", wantErrs: []error{errBoom}},
		{name: "a failed rollback is reported alongside the activity failure", deleteErr: errRollback, wantErrs: []error{errBoom, errRollback}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			f.authorID = f.userID
			expectOpenJournal(m, f)
			m.commentSvc.EXPECT().Create(mock.Anything, mock.Anything, f.userID, mock.Anything, mock.Anything).Return(f.commentID, nil).Once()
			m.repo.EXPECT().UpdateLastAuthorActivity(mock.Anything, f.journalID).Return(errBoom).Once()
			m.commentSvc.EXPECT().Delete(mock.Anything, spec.CommentRef{Kind: spec.CommentKindJournal, CommentID: f.commentID}, f.userID).Return(tc.deleteErr).Once()

			// when
			got, err := svc.CreateComment(context.Background(), f.journalID, f.userID, dto.CreateJournalCommentRequest{Body: "today's entry"}, nil)

			// then
			for _, want := range tc.wantErrs {
				require.ErrorIs(t, err, want)
			}
			assert.Equal(t, uuid.Nil, got)
			m.notifService.AssertNotCalled(t, "NotifyMany", mock.Anything, mock.Anything)
		})
	}
}

func TestFollowJournal(t *testing.T) {
	tests := []struct {
		name     string
		given    func(m *testMocks, f fixture)
		wantErr  error
		notifies bool
	}{
		{
			name: "a missing journal is not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "a failed author lookup is surfaced, not reported as not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a failed block lookup refuses the follow",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "the author cannot follow their own journal",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
			},
			wantErr: ErrCannotFollowOwn,
		},
		{
			name: "a block between follower and author is refused",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(true, nil)
			},
			wantErr: block.ErrUserBlocked,
		},
		{
			name: "a follow error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
				m.repo.EXPECT().Follow(mock.Anything, spec.JournalFollow{UserID: f.userID, JournalID: f.journalID}).Return(errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a follow notifies the author",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
				m.repo.EXPECT().Follow(mock.Anything, spec.JournalFollow{UserID: f.userID, JournalID: f.journalID}).Return(nil)
				m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("title", nil)
				m.userRepo.EXPECT().GetByID(mock.Anything, f.userID).Return(nil, nil)
			},
			notifies: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			var sent <-chan dto.NotifyParams
			if tc.notifies {
				sent = captureNotifications(m, 1)
			}

			// when
			err := svc.FollowJournal(context.Background(), f.journalID, f.userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if !tc.notifies {
				return
			}

			followed := awaitNotifications(t, sent, 1)[dto.NotifJournalFollowed]
			assert.Equal(t, f.authorID, followed.RecipientID)
			assert.Equal(t, f.journalID, followed.ReferenceID)
			assert.Equal(t, f.userID, followed.ActorID)
		})
	}
}

func TestUnfollowJournal(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "the follow is removed"},
		{name: "a repo error bubbles up", repoErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			m.repo.EXPECT().Unfollow(mock.Anything, spec.JournalFollow{UserID: f.userID, JournalID: f.journalID}).Return(tc.repoErr)

			// when
			err := svc.UnfollowJournal(context.Background(), f.journalID, f.userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestArchiveStale(t *testing.T) {
	tests := []struct {
		name      string
		given     func(m *testMocks, f fixture)
		wantCount int
		wantErr   error
	}{
		{
			name: "a repo error archives nothing",
			given: func(m *testMocks, _ fixture) {
				m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "nothing stale archives nothing",
			given: func(m *testMocks, _ fixture) {
				m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return(nil, nil)
			},
		},
		{
			name: "each archived journal's author is notified and an unknown author is skipped",
			given: func(m *testMocks, f fixture) {
				skippedID := uuid.New()
				m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return([]uuid.UUID{f.journalID, skippedID}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("title1", nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, skippedID).Return(uuid.Nil, errBoom)
				m.notifService.EXPECT().Notify(mock.Anything, mock.MatchedBy(func(p dto.NotifyParams) bool {
					return p.RecipientID == f.authorID && p.Type == dto.NotifJournalArchived && p.ReferenceID == f.journalID
				})).Return(nil)
			},
			wantCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			tc.given(m, newFixture())

			// when
			count, err := svc.ArchiveStale(context.Background())

			// then
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.wantCount, count)
		})
	}
}

func TestCreateEntry(t *testing.T) {
	tests := []struct {
		name       string
		req        dto.CreateJournalEntryRequest
		given      func(m *testMocks, f fixture)
		wantErr    error
		wantNumber int
	}{
		{
			name:    "a blank body is rejected before any lookup",
			req:     dto.CreateJournalEntryRequest{Title: "x", Body: "  "},
			given:   func(*testMocks, fixture) {},
			wantErr: ErrEmptyBody,
		},
		{
			name: "someone else without the permission is refused",
			req:  dto.CreateJournalEntryRequest{Body: "body"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(false)
			},
			wantErr: ErrNotAuthor,
		},
		{
			name: "an untitled entry takes the next number, counts its words and bumps the journals section",
			req:  dto.CreateJournalEntryRequest{Body: "an entry"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(7, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
					JournalID:            f.journalID,
					EntryNumber:          7,
					Body:                 "an entry",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				allowBackgroundFanOut(m, f.journalID, f.userID)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
			wantNumber: 7,
		},
		{
			name: "a titled entry stores its title",
			req:  dto.CreateJournalEntryRequest{Title: "Day 1", Body: "the body"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(1, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
					JournalID:            f.journalID,
					EntryNumber:          1,
					Title:                new("Day 1"),
					Body:                 "the body",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				allowBackgroundFanOut(m, f.journalID, f.userID)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
			wantNumber: 1,
		},
		{
			name: "a draft entry is stored without bumping the sidebar",
			req:  dto.CreateJournalEntryRequest{Body: "a draft", IsDraft: true},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(2, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
					JournalID:            f.journalID,
					EntryNumber:          2,
					Body:                 "a draft",
					WordCount:            2,
					IsDraft:              true,
					RecordAuthorActivity: true,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
			},
			wantNumber: 2,
		},
		{
			name: "an admin's entry is audited against the author",
			req:  dto.CreateJournalEntryRequest{Body: "an entry"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(true)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(4, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
					JournalID:            f.journalID,
					EntryNumber:          4,
					Body:                 "an entry",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalEntryCreateAdmin,
					TargetType: audit.TargetJournalEntry,
					TargetID:   f.entryID.String(),
					Details:    "journal_id=" + f.journalID.String() + ",entry_number=4,is_draft=false",
					SubjectID:  f.authorID,
				}).Return(nil)
				allowBackgroundFanOut(m, f.journalID, f.authorID)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
			wantNumber: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			id, number, err := svc.CreateEntry(context.Background(), f.journalID, f.userID, tc.req, nil)

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if tc.wantErr != nil {
				return
			}

			assert.Equal(t, f.entryID, id)
			assert.Equal(t, tc.wantNumber, number)
		})
	}
}

func TestCreateEntry_WithMedia(t *testing.T) {
	picture := uploadtest.PNG(t)
	expectEntryWritten := func(m *testMocks, f fixture) {
		m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(3, nil).Once()
		m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
			JournalID:   f.journalID,
			EntryNumber: 3,
			Body:        "an entry",
			WordCount:   2,
		}).Return(&model.JournalEntryRow{ID: f.entryID}, nil).Once()
	}

	tests := []struct {
		name     string
		files    func() []upload.FileUpload
		given    func(m *testMocks, f fixture, recorder *upload.MockMediaRecorder)
		wantErrs []error
	}{
		{
			name: "a file that is not what it claims is refused before the entry is written",
			files: func() []upload.FileUpload {
				return []upload.FileUpload{uploadtest.MemoryUpload("image/png", "fake.png", []byte("plainly not an image"), false)}
			},
			given:    func(*testMocks, fixture, *upload.MockMediaRecorder) {},
			wantErrs: []error{upload.ErrInvalidFileType},
		},
		{
			name: "every file is kept in display order with its spoiler flag before the activity is recorded and the entry announced",
			files: func() []upload.FileUpload {
				return []upload.FileUpload{uploadtest.MemoryUpload("image/png", "one.png", picture, false), uploadtest.MemoryUpload("image/png", "two.png", picture, true)}
			},
			given: func(m *testMocks, f fixture, recorder *upload.MockMediaRecorder) {
				expectEntryWritten(m, f)
				expectStored(recorder, "one.png", false)
				expectStored(recorder, "two.png", true)
				m.repo.EXPECT().AddMedia(mock.Anything, storedMedia(f.entryID, "one.png", 0, false)).Return(int64(1), nil).Once()
				m.repo.EXPECT().AddMedia(mock.Anything, storedMedia(f.entryID, "two.png", 1, true)).Return(int64(2), nil).Once()
				m.repo.EXPECT().UpdateLastAuthorActivity(mock.Anything, f.journalID).Return(nil).Once()
				allowBackgroundFanOut(m, f.journalID, f.userID)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
		},
		{
			name: "a store failure deletes the entry with the files already stored and announces nothing",
			files: func() []upload.FileUpload {
				return []upload.FileUpload{uploadtest.MemoryUpload("image/png", "one.png", picture, false), uploadtest.MemoryUpload("image/png", "two.png", picture, true)}
			},
			given: func(m *testMocks, f fixture, recorder *upload.MockMediaRecorder) {
				expectEntryWritten(m, f)
				expectStored(recorder, "one.png", false)
				expectStoreFailure(recorder, "two.png", true)
				m.repo.EXPECT().AddMedia(mock.Anything, storedMedia(f.entryID, "one.png", 0, false)).Return(int64(1), nil).Once()
				m.repo.EXPECT().DeleteEntryWithMedia(mock.Anything, f.entryID).Return([]string{storedURL("one.png")}, nil).Once()
				m.uploadSvc.EXPECT().Delete([]string{storedURL("one.png")}).Return().Twice()
			},
			wantErrs: []error{errBoom},
		},
		{
			name: "an activity failure after the files are stored deletes the entry and its files and announces nothing",
			files: func() []upload.FileUpload {
				return []upload.FileUpload{uploadtest.MemoryUpload("image/png", "one.png", picture, false), uploadtest.MemoryUpload("image/png", "two.png", picture, true)}
			},
			given: func(m *testMocks, f fixture, recorder *upload.MockMediaRecorder) {
				expectEntryWritten(m, f)
				expectStored(recorder, "one.png", false)
				expectStored(recorder, "two.png", true)
				m.repo.EXPECT().AddMedia(mock.Anything, storedMedia(f.entryID, "one.png", 0, false)).Return(int64(1), nil).Once()
				m.repo.EXPECT().AddMedia(mock.Anything, storedMedia(f.entryID, "two.png", 1, true)).Return(int64(2), nil).Once()
				m.repo.EXPECT().UpdateLastAuthorActivity(mock.Anything, f.journalID).Return(errBoom).Once()
				m.repo.EXPECT().DeleteEntryWithMedia(mock.Anything, f.entryID).Return([]string{storedURL("one.png"), storedURL("two.png")}, nil).Once()
				m.uploadSvc.EXPECT().Delete([]string{storedURL("one.png"), storedURL("two.png")}).Return().Once()
			},
			wantErrs: []error{errBoom},
		},
		{
			name: "a rollback that fails is reported alongside the store failure",
			files: func() []upload.FileUpload {
				return []upload.FileUpload{uploadtest.MemoryUpload("image/png", "one.png", picture, false)}
			},
			given: func(m *testMocks, f fixture, recorder *upload.MockMediaRecorder) {
				expectEntryWritten(m, f)
				expectStoreFailure(recorder, "one.png", false)
				m.repo.EXPECT().DeleteEntryWithMedia(mock.Anything, f.entryID).Return(nil, errRollback).Once()
			},
			wantErrs: []error{errBoom, errRollback},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			recorder := withMediaRecorder(t, svc, m)
			f := newFixture()
			m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
			tc.given(m, f, recorder)

			// when
			id, number, err := svc.CreateEntry(context.Background(), f.journalID, f.userID, dto.CreateJournalEntryRequest{Body: "an entry"}, tc.files())

			// then
			if len(tc.wantErrs) > 0 {
				for _, wantErr := range tc.wantErrs {
					require.ErrorIs(t, err, wantErr)
				}
				assert.Equal(t, uuid.Nil, id)
				assert.Zero(t, number)
				m.activity.AssertNotCalled(t, "BumpSidebarActivity", sidebar.KeyJournals)
				m.repo.AssertNotCalled(t, "GetTitle", mock.Anything, f.journalID)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, f.entryID, id)
			assert.Equal(t, 3, number)
		})
	}
}

func TestCreateEntry_FilesWithoutText(t *testing.T) {
	// given an entry with no words but one picture
	svc, m := newTestService(t)
	recorder := withMediaRecorder(t, svc, m)
	f := newFixture()
	m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
	m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(5, nil).Once()
	m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
		JournalID:   f.journalID,
		EntryNumber: 5,
	}).Return(&model.JournalEntryRow{ID: f.entryID}, nil).Once()
	expectStored(recorder, "one.png", false)
	m.repo.EXPECT().AddMedia(mock.Anything, storedMedia(f.entryID, "one.png", 0, false)).Return(int64(1), nil).Once()
	m.repo.EXPECT().UpdateLastAuthorActivity(mock.Anything, f.journalID).Return(nil).Once()
	allowBackgroundFanOut(m, f.journalID, f.userID)
	m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()

	// when
	id, number, err := svc.CreateEntry(context.Background(), f.journalID, f.userID, dto.CreateJournalEntryRequest{Body: "  "}, []upload.FileUpload{uploadtest.MemoryUpload("image/png", "one.png", uploadtest.PNG(t), false)})

	// then
	require.NoError(t, err)
	assert.Equal(t, f.entryID, id)
	assert.Equal(t, 5, number)
}

func TestJournalEntryPublish_NotifiesMentionsAndFollowersOnlyOnTheFirstPublish(t *testing.T) {
	tests := []struct {
		name           string
		create         bool
		draft          bool
		asAdmin        bool
		firstPublished bool
		wantNotified   bool
	}{
		{
			name:         "an entry created published notifies the named user and the followers",
			create:       true,
			wantNotified: true,
		},
		{
			name:   "an entry created as a draft notifies nobody",
			create: true,
			draft:  true,
		},
		{
			name:         "an admin creating a published entry notifies in the author's name",
			create:       true,
			asAdmin:      true,
			wantNotified: true,
		},
		{
			name:           "publishing a draft for the first time notifies the named user and the followers",
			firstPublished: true,
			wantNotified:   true,
		},
		{
			name: "republishing an entry that was unpublished notifies nobody again",
		},
		{
			name:           "an admin publishing a draft for the first time notifies in the author's name",
			asAdmin:        true,
			firstPublished: true,
			wantNotified:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			mentionedID := uuid.New()
			followerID := uuid.New()
			body := "thoughts for @alice"

			authorID := f.userID
			if tc.asAdmin {
				authorID = f.authorID
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(true)
				m.auditRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			}

			m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(authorID, nil)

			if tc.create {
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(3, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, mock.Anything).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
			} else {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).
					Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 3, IsDraft: true}, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, mock.Anything).Return(tc.firstPublished, nil)
			}

			var sent <-chan dto.NotifyParams
			followerNotices := make(chan []dto.NotifyParams, 1)
			if tc.wantNotified {
				expectMentionOfAlice(m, authorID, mentionedID)
				sent = captureNotifications(m, 1)
				m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("Diary", nil)
				m.repo.EXPECT().GetFollowerIDs(mock.Anything, f.journalID).Return([]uuid.UUID{followerID, authorID}, nil)
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, authorID).Return(nil, nil)
				m.notifService.EXPECT().NotifyMany(mock.Anything, mock.Anything).
					Run(func(_ context.Context, params []dto.NotifyParams) {
						followerNotices <- params
					}).
					Return().
					Once()
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			}

			// when
			var err error
			if tc.create {
				_, _, err = svc.CreateEntry(context.Background(), f.journalID, f.userID, dto.CreateJournalEntryRequest{Body: body, IsDraft: tc.draft}, nil)
			} else {
				err = svc.UpdateEntry(context.Background(), f.entryID, f.userID, dto.UpdateJournalEntryRequest{Body: body})
			}

			// then
			require.NoError(t, err)

			if !tc.wantNotified {
				m.userRepo.AssertNotCalled(t, "GetByUsernames", mock.Anything, mock.Anything)
				return
			}

			mentioned := awaitNotifications(t, sent, 1)[dto.NotifMention]
			assert.Equal(t, dto.NotifMention, mentioned.Type)
			assert.Equal(t, mentionedID, mentioned.RecipientID)
			assert.Equal(t, authorID, mentioned.ActorID)
			assert.Equal(t, f.journalID, mentioned.ReferenceID)
			assert.Equal(t, "journal_entry:3", mentioned.ReferenceType)
			assert.Equal(t, "/journals/"+f.journalID.String()+"/entry/3", mentioned.EmailLink)

			var notices []dto.NotifyParams
			select {
			case notices = <-followerNotices:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "timed out waiting for the follower fan-out")
			}
			require.Len(t, notices, 1)
			assert.Equal(t, followerID, notices[0].RecipientID)
			assert.Equal(t, dto.NotifJournalUpdate, notices[0].Type)
			assert.Equal(t, authorID, notices[0].ActorID)
			assert.Equal(t, "journal_entry:3", notices[0].ReferenceType)
		})
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	// given
	svc, m := newTestService(t)
	journalID := uuid.New()
	m.repo.EXPECT().GetEntry(mock.Anything, spec.JournalEntryLookup{JournalID: journalID, EntryNumber: 5}).Return(nil, nil)

	// when
	_, _, err := svc.GetEntry(context.Background(), journalID, 5, uuid.Nil)

	// then
	require.ErrorIs(t, err, ErrEntryNotFound)
}

func TestGetEntry(t *testing.T) {
	// given
	svc, m := newTestService(t)
	f := newFixture()
	row := &model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 3, Body: "b", HasPrev: true}
	m.repo.EXPECT().GetEntry(mock.Anything, spec.JournalEntryLookup{JournalID: f.journalID, EntryNumber: 3}).Return(row, nil)
	m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
	thread := []dto.CommentResponse{{ID: f.commentID, EntryID: &f.entryID, Body: "loved this one"}}
	m.commentSvc.EXPECT().Thread(mock.Anything, spec.CommentTarget{Kind: spec.CommentKindJournalEntry, ID: f.journalID, EntryNumber: 3}, uuid.Nil).Return(thread, nil).Once()
	m.repo.EXPECT().GetMediaBatch(mock.Anything, []uuid.UUID{f.entryID}).Return(nil, nil)

	// when
	entry, comments, err := svc.GetEntry(context.Background(), f.journalID, 3, uuid.Nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, 3, entry.EntryNumber)
	assert.True(t, entry.HasPrev)
	assert.Equal(t, thread, comments)
}

func TestUpdateEntry(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.UpdateJournalEntryRequest
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name: "someone else without the permission is refused",
			req:  dto.UpdateJournalEntryRequest{Body: "x"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(false)
			},
			wantErr: ErrNotAuthor,
		},
		{
			name: "a draft saved as a draft records no activity and does not bump the sidebar",
			req:  dto.UpdateJournalEntryRequest{Body: "still drafting", IsDraft: true},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, IsDraft: true}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, spec.JournalEntryUpdate{
					ID:        f.entryID,
					JournalID: f.journalID,
					Body:      "still drafting",
					WordCount: 2,
					IsDraft:   true,
				}).Return(false, nil)
			},
		},
		{
			name: "publishing a draft records activity and bumps the journals section",
			req:  dto.UpdateJournalEntryRequest{Title: "Day 2", Body: "published now"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 2, IsDraft: true}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, spec.JournalEntryUpdate{
					ID:                   f.entryID,
					JournalID:            f.journalID,
					Title:                new("Day 2"),
					Body:                 "published now",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(true, nil)
				allowBackgroundFanOut(m, f.journalID, f.userID)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
		},
		{
			name: "republishing an entry that was published before records activity but neither notifies nor bumps the sidebar",
			req:  dto.UpdateJournalEntryRequest{Body: "back again"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 2, IsDraft: true}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, spec.JournalEntryUpdate{
					ID:                   f.entryID,
					JournalID:            f.journalID,
					Body:                 "back again",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(false, nil)
			},
		},
		{
			name: "editing an already published entry does not bump the sidebar",
			req:  dto.UpdateJournalEntryRequest{Body: "fixed a typo"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 2}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, spec.JournalEntryUpdate{
					ID:        f.entryID,
					JournalID: f.journalID,
					Body:      "fixed a typo",
					WordCount: 3,
				}).Return(false, nil)
			},
		},
		{
			name: "a failed publish neither notifies nor bumps the sidebar",
			req:  dto.UpdateJournalEntryRequest{Body: "published now"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 2, IsDraft: true}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, spec.JournalEntryUpdate{
					ID:                   f.entryID,
					JournalID:            f.journalID,
					Body:                 "published now",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(false, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "an admin edit is audited with the changed fields",
			req:  dto.UpdateJournalEntryRequest{Body: "new body"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 2, Body: "old body", IsDraft: true}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(true)
				m.repo.EXPECT().UpdateEntry(mock.Anything, spec.JournalEntryUpdate{
					ID:                   f.entryID,
					JournalID:            f.journalID,
					Body:                 "new body",
					WordCount:            2,
					RecordAuthorActivity: true,
				}).Return(true, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalEntryUpdateAdmin,
					TargetType: audit.TargetJournalEntry,
					TargetID:   f.entryID.String(),
					Details:    "changed=body,is_draft",
					SubjectID:  f.authorID,
				}).Return(nil)
				allowBackgroundFanOut(m, f.journalID, f.authorID)
				m.activity.EXPECT().BumpSidebarActivity(sidebar.KeyJournals).Return().Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			err := svc.UpdateEntry(context.Background(), f.entryID, f.userID, tc.req)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDeleteEntry(t *testing.T) {
	tests := []struct {
		name  string
		given func(m *testMocks, f fixture)
	}{
		{
			name: "the owner's delete is audited and its media removed",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryAuthorID(mock.Anything, f.entryID).Return(f.userID, nil)
				m.repo.EXPECT().DeleteEntryWithMedia(mock.Anything, f.entryID).Return([]string{"/u/e.png", "/u/e-thumb.png"}, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalEntryDelete,
					TargetType: audit.TargetJournalEntry,
					TargetID:   f.entryID.String(),
					SubjectID:  f.userID,
				}).Return(nil)
				m.uploadSvc.EXPECT().Delete([]string{"/u/e.png", "/u/e-thumb.png"}).Return()
			},
		},
		{
			name: "an admin's delete is audited as an admin action against the author",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetEntryAuthorID(mock.Anything, f.entryID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermDeleteAnyJournal).Return(true)
				m.repo.EXPECT().DeleteEntryWithMedia(mock.Anything, f.entryID).Return(nil, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalEntryDeleteAdmin,
					TargetType: audit.TargetJournalEntry,
					TargetID:   f.entryID.String(),
					SubjectID:  f.authorID,
				}).Return(nil)
				m.uploadSvc.EXPECT().Delete().Return()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			tc.given(m, f)

			// when
			err := svc.DeleteEntry(context.Background(), f.entryID, f.userID)

			// then
			require.NoError(t, err)
		})
	}
}
