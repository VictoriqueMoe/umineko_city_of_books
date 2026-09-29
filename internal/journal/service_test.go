package journal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/audit"
	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dao"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/journal/params"
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
		repo         *repository.MockJournalRepository
		comments     *repository.MockJournalCommentWriter
		userRepo     *repository.MockUserRepository
		auditRepo    *repository.MockAuditLogRepository
		authz        *authz.MockService
		blockSvc     *block.MockService
		notifService *notification.MockService
		uploadSvc    *upload.MockService
		settingsSvc  *settings.MockService
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
	errMissingRow = errors.Join(errors.New("no row"), dao.ErrNotFound)
)

func newTestService(t *testing.T) (*service, *testMocks) {
	m := &testMocks{
		repo:         repository.NewMockJournalRepository(t),
		comments:     repository.NewMockJournalCommentWriter(t),
		userRepo:     repository.NewMockUserRepository(t),
		auditRepo:    repository.NewMockAuditLogRepository(t),
		authz:        authz.NewMockService(t),
		blockSvc:     block.NewMockService(t),
		notifService: notification.NewMockService(t),
		uploadSvc:    upload.NewMockService(t),
		settingsSvc:  settings.NewMockService(t),
	}

	mentionSvc := mention.NewService(m.userRepo, m.blockSvc, m.notifService, dao.CommentDAOs{Journal: m.comments})
	svc := NewService(m.repo, m.userRepo, m.auditRepo, m.authz, m.blockSvc, m.notifService, mentionSvc, m.uploadSvc, &media.Processor{}, m.settingsSvc, contentfilter.New(), nil, nil).(*service)

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

func allowBackgroundFanOut(m *testMocks, f fixture) {
	m.userRepo.EXPECT().GetByID(mock.Anything, f.userID).Return(nil, nil).Maybe()
	m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("j", nil).Maybe()
	m.repo.EXPECT().GetFollowerIDs(mock.Anything, f.journalID).Return(nil, nil).Maybe()
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(nil, nil).Maybe()
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
			name:  "with no daily limit the journal is created without counting",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(0)
				m.repo.EXPECT().Create(mock.Anything, spec.NewJournal{UserID: f.userID, Title: "Title", Work: "umineko"}).Return(&dto.JournalResponse{ID: f.journalID}, nil)
			},
		},
		{
			name:  "under the daily limit the journal is created",
			title: "Title",
			given: func(m *testMocks, f fixture) {
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxJournalsPerDay).Return(5)
				m.repo.EXPECT().CountUserJournalsToday(mock.Anything, f.userID).Return(2, nil)
				m.repo.EXPECT().Create(mock.Anything, spec.NewJournal{UserID: f.userID, Title: "Title", Work: "umineko"}).Return(&dto.JournalResponse{ID: f.journalID}, nil)
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
			name: "a comment load error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}}, nil)
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(nil, nil)
				m.repo.EXPECT().GetComments(mock.Anything, spec.CommentQuery[uuid.UUID]{
					TargetID:       f.journalID,
					ViewerID:       f.userID,
					Limit:          500,
					Offset:         0,
					ExcludeUserIDs: []uuid.UUID(nil),
				}).Return(nil, 0, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a block list error bubbles up instead of showing blocked users' comments",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}}, nil)
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a comment media error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}}, nil)
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(nil, nil)
				m.repo.EXPECT().GetComments(mock.Anything, mock.Anything).Return([]model.CommentRow{{ID: f.commentID}}, 1, nil)
				m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{f.commentID}).Return(nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a latest entry media error bubbles up",
			given: func(m *testMocks, f fixture) {
				latest := 1
				m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}, LatestEntryNumber: &latest}, nil)
				m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(nil, nil)
				m.repo.EXPECT().GetComments(mock.Anything, mock.Anything).Return(nil, 0, nil)
				m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, mock.Anything).Return(nil, nil)
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
	blockedIDs := []uuid.UUID{uuid.New()}
	m.repo.EXPECT().GetByID(mock.Anything, spec.JournalLookup{ID: f.journalID, ViewerID: f.userID}).Return(&dto.JournalResponse{ID: f.journalID, Author: dto.UserResponse{ID: f.authorID}}, nil)
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, f.userID).Return(blockedIDs, nil)
	m.repo.EXPECT().GetComments(mock.Anything, spec.CommentQuery[uuid.UUID]{
		TargetID:       f.journalID,
		ViewerID:       f.userID,
		Limit:          500,
		Offset:         0,
		ExcludeUserIDs: blockedIDs,
	}).Return([]model.CommentRow{{ID: f.commentID, UserID: f.authorID, Body: "hi"}}, 1, nil)
	m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{f.commentID}).Return(nil, nil)
	m.repo.EXPECT().ListEntries(mock.Anything, f.journalID).Return(nil, nil)

	// when
	got, err := svc.GetJournalDetail(context.Background(), f.journalID, f.userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got.Comments, 1)
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
		body    string
		onEntry bool
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name:    "a blank body is rejected before any lookup",
			body:    "   ",
			given:   func(*testMocks, fixture) {},
			wantErr: ErrEmptyBody,
		},
		{
			name: "a missing journal is not found",
			body: "hi",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "a failed author lookup is surfaced, not reported as not found",
			body: "hi",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "an archive check error bubbles up",
			body: "hi",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.repo.EXPECT().IsArchived(mock.Anything, f.journalID).Return(false, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "an archived journal takes no comments",
			body: "hi",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.repo.EXPECT().IsArchived(mock.Anything, f.journalID).Return(true, nil)
			},
			wantErr: ErrArchived,
		},
		{
			name:    "an entry from another journal is a mismatch",
			body:    "body",
			onEntry: true,
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.repo.EXPECT().GetEntryJournalID(mock.Anything, f.entryID).Return(uuid.New(), nil)
			},
			wantErr: ErrEntryMismatch,
		},
		{
			name: "a block between commenter and author is refused",
			body: "hi",
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(true, nil)
			},
			wantErr: block.ErrUserBlocked,
		},
		{
			name: "a write error bubbles up",
			body: "hi",
			given: func(m *testMocks, f fixture) {
				expectOpenJournal(m, f)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
				m.comments.EXPECT().CreateComment(mock.Anything, spec.NewJournalComment{JournalID: f.journalID, UserID: f.userID, Body: "hi"}).Return(nil, errBoom)
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
			_, err := svc.CreateComment(context.Background(), f.journalID, f.userID, entryID, nil, tc.body)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestCreateComment_AuthorReplyWithParent(t *testing.T) {
	// given
	svc, m := newTestService(t)
	f := newFixture()
	f.authorID = f.userID
	parentID := uuid.New()
	expectOpenJournal(m, f)
	m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
	m.comments.EXPECT().CreateComment(mock.Anything, spec.NewJournalComment{
		JournalID:            f.journalID,
		ParentID:             &parentID,
		UserID:               f.userID,
		Body:                 "hi",
		RecordAuthorActivity: true,
	}).Return(&model.CommentRow{ID: f.commentID}, nil)
	allowBackgroundFanOut(m, f)
	m.repo.EXPECT().GetCommentAuthorID(mock.Anything, parentID).Return(uuid.Nil, errBoom).Maybe()

	// when
	got, err := svc.CreateComment(context.Background(), f.journalID, f.userID, nil, &parentID, "hi")

	// then
	require.NoError(t, err)
	assert.Equal(t, f.commentID, got)
}

func TestCreateComment_MentionNotifiesTheNamedUser(t *testing.T) {
	tests := []struct {
		name        string
		onEntry     bool
		entryNumber int
		wantRefType string
		wantLink    string
	}{
		{
			name:        "a comment on the journal itself",
			wantRefType: "journal_comment:%s",
			wantLink:    "/journals/%s#comment-%s",
		},
		{
			name:        "a comment on one entry",
			onEntry:     true,
			entryNumber: 4,
			wantRefType: "journal_entry_comment:4:%s",
			wantLink:    "/journals/%s/entry/4#comment-%s",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			mentionedID := uuid.New()

			wantComment := spec.NewJournalComment{JournalID: f.journalID, UserID: f.userID, Body: "look at this @alice"}
			var entryArg *uuid.UUID
			if tc.onEntry {
				entryArg = &f.entryID
				wantComment.EntryID = &f.entryID
				m.repo.EXPECT().GetEntryJournalID(mock.Anything, f.entryID).Return(f.journalID, nil)
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).
					Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: tc.entryNumber}, nil)
			}

			expectOpenJournal(m, f)
			m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
			m.comments.EXPECT().CreateComment(mock.Anything, wantComment).Return(&model.CommentRow{ID: f.commentID}, nil)
			m.repo.EXPECT().GetTitle(mock.Anything, f.journalID).Return("j", nil)
			expectMentionOfAlice(m, f.userID, mentionedID)
			sent := captureNotifications(m, 2)

			// when
			got, err := svc.CreateComment(context.Background(), f.journalID, f.userID, entryArg, nil, "look at this @alice")

			// then
			require.NoError(t, err)
			assert.Equal(t, f.commentID, got)

			notified := awaitNotifications(t, sent, 2)
			assert.Equal(t, f.authorID, notified[dto.NotifJournalCommented].RecipientID)

			mentioned := notified[dto.NotifMention]
			assert.Equal(t, mentionedID, mentioned.RecipientID)
			assert.Equal(t, fmt.Sprintf(tc.wantRefType, f.commentID), mentioned.ReferenceType)
			assert.Equal(t, fmt.Sprintf(tc.wantLink, f.journalID, f.commentID), mentioned.EmailLink)
		})
	}
}

func TestUpdateComment(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		given   func(m *testMocks, f fixture)
		wantErr error
		audited bool
	}{
		{
			name:    "a blank body is rejected before any lookup",
			body:    "   ",
			given:   func(*testMocks, fixture) {},
			wantErr: ErrEmptyBody,
		},
		{
			name: "the owner's edit is trimmed and not audited",
			body: "  trimmed  ",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateComment(mock.Anything, spec.CommentUpdate{CommentID: f.commentID, UserID: f.userID, Body: "trimmed"}).Return(nil)
			},
		},
		{
			name: "an update error bubbles up",
			body: "body",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.userID, nil)
				m.repo.EXPECT().UpdateComment(mock.Anything, spec.CommentUpdate{CommentID: f.commentID, UserID: f.userID, Body: "body"}).Return(errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "an admin edit is audited against the author",
			body: "new body",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyComment).Return(true)
				m.repo.EXPECT().UpdateComment(mock.Anything, spec.CommentUpdate{CommentID: f.commentID, UserID: f.userID, Body: "new body", AsAdmin: true}).Return(nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalCommentUpdateAdmin,
					TargetType: audit.TargetJournalComment,
					TargetID:   f.commentID.String(),
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
			err := svc.UpdateComment(context.Background(), f.commentID, f.userID, tc.body)

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if !tc.audited {
				m.auditRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteComment(t *testing.T) {
	tests := []struct {
		name    string
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name: "the owner's delete removes its media",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.userID, nil)
				m.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, spec.CommentDeletion{CommentID: f.commentID, UserID: f.userID, AsAdmin: false}).Return([]string{"/u/reply.png"}, nil)
				m.uploadSvc.EXPECT().Delete([]string{"/u/reply.png"}).Return()
			},
		},
		{
			name: "an admin's delete is flagged as an admin action",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermDeleteAnyComment).Return(true)
				m.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, spec.CommentDeletion{CommentID: f.commentID, UserID: f.userID, AsAdmin: true}).Return([]string{"/u/c.png", "/u/c-thumb.png"}, nil)
				m.uploadSvc.EXPECT().Delete([]string{"/u/c.png", "/u/c-thumb.png"}).Return()
			},
		},
		{
			name: "a delete error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.userID, nil)
				m.repo.EXPECT().DeleteCommentWithAudit(mock.Anything, spec.CommentDeletion{CommentID: f.commentID, UserID: f.userID, AsAdmin: false}).Return(nil, errBoom)
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
			err := svc.DeleteComment(context.Background(), f.commentID, f.userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestLikeComment(t *testing.T) {
	tests := []struct {
		name     string
		given    func(m *testMocks, f fixture)
		wantErr  error
		notifies bool
	}{
		{
			name: "a missing comment is not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(uuid.Nil, errMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "a failed comment author lookup is surfaced, not reported as not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a failed block lookup refuses the like",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "a block between liker and author is refused",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(true, nil)
			},
			wantErr: block.ErrUserBlocked,
		},
		{
			name: "a like error bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
				m.repo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: f.userID, CommentID: f.commentID}).Return(errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "liking your own comment notifies nobody",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.userID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.userID).Return(false, nil)
				m.repo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: f.userID, CommentID: f.commentID}).Return(nil)
			},
		},
		{
			name: "liking someone else's comment notifies its author",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
				m.blockSvc.EXPECT().IsBlockedEither(mock.Anything, f.userID, f.authorID).Return(false, nil)
				m.repo.EXPECT().LikeComment(mock.Anything, spec.CommentLike{UserID: f.userID, CommentID: f.commentID}).Return(nil)
				m.repo.EXPECT().GetCommentEntityID(mock.Anything, f.commentID).Return(f.journalID, nil)
				m.repo.EXPECT().GetCommentEntryNumber(mock.Anything, f.commentID).Return(nil, nil)
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
			err := svc.LikeComment(context.Background(), f.commentID, f.userID)

			// then
			require.ErrorIs(t, err, tc.wantErr)

			if !tc.notifies {
				return
			}

			liked := awaitNotifications(t, sent, 1)[dto.NotifJournalCommentLiked]
			assert.Equal(t, f.authorID, liked.RecipientID)
			assert.Equal(t, f.journalID, liked.ReferenceID)
			assert.Equal(t, f.userID, liked.ActorID)
		})
	}
}

func TestUnlikeComment(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "the like is removed"},
		{name: "a repo error bubbles up", repoErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			m.repo.EXPECT().UnlikeComment(mock.Anything, spec.CommentLike{UserID: f.userID, CommentID: f.commentID}).Return(tc.repoErr)

			// when
			err := svc.UnlikeComment(context.Background(), f.commentID, f.userID)

			// then
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestUploadCommentMedia(t *testing.T) {
	tests := []struct {
		name    string
		given   func(m *testMocks, f fixture)
		wantErr error
	}{
		{
			name: "a missing comment is not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(uuid.Nil, errMissingRow)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "a failed comment author lookup is surfaced, not reported as not found",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(uuid.Nil, errBoom)
			},
			wantErr: errBoom,
		},
		{
			name: "someone else's comment is refused",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.authorID, nil)
			},
			wantErr: ErrNotAuthor,
		},
		{
			name: "an upload failure bubbles up",
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetCommentAuthorID(mock.Anything, f.commentID).Return(f.userID, nil)
				m.settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(1000)
				m.uploadSvc.EXPECT().SaveImage(mock.Anything, "journals", mock.Anything, int64(10), int64(1000), mock.Anything).Return("", errBoom)
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
			_, err := svc.UploadCommentMedia(context.Background(), f.commentID, f.userID, "image/png", "photo.png", 10, strings.NewReader("x"), false)

			// then
			require.ErrorIs(t, err, tc.wantErr)
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
			name: "an untitled entry takes the next number and counts its words",
			req:  dto.CreateJournalEntryRequest{Body: "an entry"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(7, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
					JournalID:   f.journalID,
					EntryNumber: 7,
					Body:        "an entry",
					WordCount:   2,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				allowBackgroundFanOut(m, f)
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
					JournalID:   f.journalID,
					EntryNumber: 1,
					Title:       new("Day 1"),
					Body:        "the body",
					WordCount:   2,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				allowBackgroundFanOut(m, f)
			},
			wantNumber: 1,
		},
		{
			name: "an admin's entry is audited against the author",
			req:  dto.CreateJournalEntryRequest{Body: "an entry"},
			given: func(m *testMocks, f fixture) {
				m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.authorID, nil)
				m.authz.EXPECT().Can(mock.Anything, f.userID, authz.PermEditAnyJournal).Return(true)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(4, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, spec.NewJournalEntry{
					JournalID:   f.journalID,
					EntryNumber: 4,
					Body:        "an entry",
					WordCount:   2,
				}).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalEntryCreateAdmin,
					TargetType: audit.TargetJournalEntry,
					TargetID:   f.entryID.String(),
					Details:    "journal_id=" + f.journalID.String() + ",entry_number=4,is_draft=false",
					SubjectID:  f.authorID,
				}).Return(nil)
				allowBackgroundFanOut(m, f)
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
			id, number, err := svc.CreateEntry(context.Background(), f.journalID, f.userID, tc.req)

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

func TestJournalEntry_MentionFanOut(t *testing.T) {
	tests := []struct {
		name       string
		draft      bool
		publishing bool
		wantFanOut bool
	}{
		{
			name:       "a published new entry notifies the named user",
			wantFanOut: true,
		},
		{
			name:  "a draft entry notifies nobody",
			draft: true,
		},
		{
			name:       "publishing a draft entry notifies the named user",
			publishing: true,
			wantFanOut: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, m := newTestService(t)
			f := newFixture()
			mentionedID := uuid.New()
			body := "thoughts for @alice"

			m.repo.EXPECT().GetAuthorID(mock.Anything, f.journalID).Return(f.userID, nil)

			if tc.publishing {
				m.repo.EXPECT().GetEntryByID(mock.Anything, f.entryID).
					Return(&model.JournalEntryRow{ID: f.entryID, JournalID: f.journalID, EntryNumber: 3, IsDraft: true}, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, mock.Anything).Return(nil)
			} else {
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, f.journalID).Return(3, nil)
				m.repo.EXPECT().CreateEntry(mock.Anything, mock.Anything).Return(&model.JournalEntryRow{ID: f.entryID}, nil)
			}

			var sent <-chan dto.NotifyParams
			if tc.wantFanOut {
				expectMentionOfAlice(m, f.userID, mentionedID)
				sent = captureNotifications(m, 1)
				allowBackgroundFanOut(m, f)
			}

			// when
			var err error
			if tc.publishing {
				err = svc.UpdateEntry(context.Background(), f.entryID, f.userID, dto.UpdateJournalEntryRequest{Body: body})
			} else {
				_, _, err = svc.CreateEntry(context.Background(), f.journalID, f.userID, dto.CreateJournalEntryRequest{Body: body, IsDraft: tc.draft})
			}

			// then
			require.NoError(t, err)

			if !tc.wantFanOut {
				m.userRepo.AssertNotCalled(t, "GetByUsernames", mock.Anything, mock.Anything)
				return
			}

			mentioned := awaitNotifications(t, sent, 1)[dto.NotifMention]
			assert.Equal(t, dto.NotifMention, mentioned.Type)
			assert.Equal(t, mentionedID, mentioned.RecipientID)
			assert.Equal(t, f.journalID, mentioned.ReferenceID)
			assert.Equal(t, "journal_entry:3", mentioned.ReferenceType)
			assert.Equal(t, "/journals/"+f.journalID.String()+"/entry/3", mentioned.EmailLink)
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
	m.blockSvc.EXPECT().GetBlockedIDs(mock.Anything, uuid.Nil).Return(nil, nil)
	m.repo.EXPECT().GetEntryComments(mock.Anything, spec.CommentQuery[uuid.UUID]{
		TargetID:       f.entryID,
		ViewerID:       uuid.Nil,
		Limit:          500,
		Offset:         0,
		ExcludeUserIDs: []uuid.UUID(nil),
	}).Return(nil, 0, nil)
	m.repo.EXPECT().GetCommentMediaBatch(mock.Anything, []uuid.UUID{}).Return(nil, nil)
	m.repo.EXPECT().GetMediaBatch(mock.Anything, []uuid.UUID{f.entryID}).Return(nil, nil)

	// when
	entry, comments, err := svc.GetEntry(context.Background(), f.journalID, 3, uuid.Nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, 3, entry.EntryNumber)
	assert.True(t, entry.HasPrev)
	assert.Empty(t, comments)
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
			name: "a draft saved as a draft does not record activity",
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
				}).Return(nil)
			},
		},
		{
			name: "publishing a draft records activity",
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
				}).Return(nil)
				allowBackgroundFanOut(m, f)
			},
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
				}).Return(nil)
				m.auditRepo.EXPECT().Create(mock.Anything, audit.NewEntry{
					ActorID:    f.userID,
					Action:     audit.ActionJournalEntryUpdateAdmin,
					TargetType: audit.TargetJournalEntry,
					TargetID:   f.entryID.String(),
					Details:    "changed=body,is_draft",
					SubjectID:  f.authorID,
				}).Return(nil)
				allowBackgroundFanOut(m, f)
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
