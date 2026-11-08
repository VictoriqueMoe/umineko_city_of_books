package journal

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/comment"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/sidebar"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newCacheTestService(t *testing.T, echoes homefeed.Service) (*service, *testMocks) {
	repo := repository.NewMockJournalRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	commentSvc := comment.NewMockService(t)
	activity := sidebar.NewMockActivityBumper(t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notifSvc)

	svc := NewService(
		repo, userRepo, auditRepo, authzSvc, blockSvc, notifSvc, mentionSvc, commentSvc,
		uploadSvc, settingsSvc, contentfilter.New(), echoes, activity,
	).(*service)

	return svc, &testMocks{
		repo:         repo,
		commentSvc:   commentSvc,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		authz:        authzSvc,
		blockSvc:     blockSvc,
		notifService: notifSvc,
		uploadSvc:    uploadSvc,
		settingsSvc:  settingsSvc,
		activity:     activity,
	}
}

func TestJournalWrites_ClearTheCachedEchoes(t *testing.T) {
	journalID := uuid.New()
	entryID := uuid.New()
	authorID := uuid.New()

	tests := []struct {
		name  string
		write func(svc *service, m *testMocks) error
	}{
		{
			name: "deleting the journal",
			write: func(svc *service, m *testMocks) error {
				m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
				m.repo.EXPECT().GetTitle(mock.Anything, journalID).Return("doomed", nil)
				m.repo.EXPECT().DeleteWithMedia(mock.Anything, spec.JournalDeletion{ID: journalID, UserID: authorID, AsAdmin: false}).Return(nil, nil)
				m.authz.EXPECT().Can(mock.Anything, mock.Anything, mock.Anything).Return(false).Maybe()
				m.auditRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
				m.uploadSvc.EXPECT().Delete(mock.Anything).Return().Maybe()

				return svc.DeleteJournal(context.Background(), journalID, authorID)
			},
		},
		{
			name: "renaming the journal",
			write: func(svc *service, m *testMocks) error {
				m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
				m.repo.EXPECT().Update(mock.Anything, spec.JournalUpdate{ID: journalID, UserID: authorID, Title: "Renamed", Work: "umineko"}).Return(nil)

				return svc.UpdateJournal(context.Background(), journalID, authorID, dto.CreateJournalRequest{Title: "Renamed", Work: "umineko"})
			},
		},
		{
			name: "writing an entry, which can become the echoed excerpt",
			write: func(svc *service, m *testMocks) error {
				m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
				m.repo.EXPECT().GetNextEntryNumber(mock.Anything, journalID).Return(3, nil)
				m.repo.EXPECT().CreateEntryWithMedia(mock.Anything, mock.Anything).Return(&model.JournalEntryRow{ID: entryID}, nil)

				_, _, err := svc.CreateEntry(context.Background(), journalID, authorID, dto.CreateJournalEntryRequest{Body: "a draft", IsDraft: true})

				return err
			},
		},
		{
			name: "editing an entry",
			write: func(svc *service, m *testMocks) error {
				m.repo.EXPECT().GetEntryByID(mock.Anything, entryID).Return(&model.JournalEntryRow{ID: entryID, JournalID: journalID, IsDraft: true}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
				m.repo.EXPECT().UpdateEntry(mock.Anything, mock.Anything).Return(false, nil)

				return svc.UpdateEntry(context.Background(), entryID, authorID, dto.UpdateJournalEntryRequest{Body: "edited", IsDraft: true})
			},
		},
		{
			name: "deleting an entry",
			write: func(svc *service, m *testMocks) error {
				m.repo.EXPECT().GetEntryAuthorID(mock.Anything, entryID).Return(authorID, nil)
				m.repo.EXPECT().DeleteEntryWithMedia(mock.Anything, entryID).Return(nil, nil)
				m.auditRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
				m.uploadSvc.EXPECT().Delete(mock.Anything).Return().Maybe()

				return svc.DeleteEntry(context.Background(), entryID, authorID)
			},
		},
		{
			name: "archiving stale journals",
			write: func(svc *service, m *testMocks) error {
				m.repo.EXPECT().ArchiveStale(mock.Anything, mock.Anything).Return([]uuid.UUID{journalID}, nil)
				m.repo.EXPECT().GetAuthorID(mock.Anything, journalID).Return(authorID, nil)
				m.repo.EXPECT().GetTitle(mock.Anything, journalID).Return("quiet", nil)
				m.notifService.EXPECT().Notify(mock.Anything, mock.Anything).Return(nil)

				_, err := svc.ArchiveStale(context.Background())

				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			echoes := homefeed.NewMockService(t)
			svc, m := newCacheTestService(t, echoes)
			echoes.EXPECT().ClearEchoCache(mock.Anything).Return(nil).Once()

			// when
			err := tc.write(svc, m)

			// then the echoes list is cleared, which the strict mock asserts on cleanup
			require.NoError(t, err)
		})
	}
}
