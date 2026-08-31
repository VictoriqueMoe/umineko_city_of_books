package journal

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/block"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/mention"
	"umineko_city_of_books/internal/notification"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newCacheTestService(t *testing.T, mgr *cache.Manager, echoes homefeed.Service) (*service, *testMocks) {
	repo := repository.NewMockJournalRepository(t)
	userRepo := repository.NewMockUserRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	authzSvc := authz.NewMockService(t)
	blockSvc := block.NewMockService(t)
	notifSvc := notification.NewMockService(t)
	uploadSvc := upload.NewMockService(t)
	settingsSvc := settings.NewMockService(t)
	comments := repository.NewMockJournalCommentWriter(t)
	mentionSvc := mention.NewService(userRepo, blockSvc, notifSvc, repository.CommentDAOs{Journal: comments})

	resolver := og.NewResolver(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		settingsSvc,
		mgr,
		"",
		"",
	)

	svc := NewService(
		repo, userRepo, auditRepo, authzSvc, blockSvc, notifSvc, mentionSvc,
		uploadSvc, &media.Processor{}, settingsSvc, contentfilter.New(), resolver, echoes,
	).(*service)

	return svc, &testMocks{
		repo:         repo,
		comments:     comments,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		authz:        authzSvc,
		blockSvc:     blockSvc,
		notifService: notifSvc,
		uploadSvc:    uploadSvc,
		settingsSvc:  settingsSvc,
	}
}

func TestDeleteJournal_ClearsCachedPages(t *testing.T) {
	// given a deleted journal whose og meta is already cached
	ctx := context.Background()
	mgr := cache.New()
	echoes := homefeed.NewMockService(t)
	svc, m := newCacheTestService(t, mgr, echoes)

	id := uuid.New()
	authorID := uuid.New()
	key := cache.OGMeta.Key("/journals/" + id.String())
	require.NoError(t, mgr.Set(ctx, key, "cached meta", cache.OGMeta.TTL))

	m.repo.EXPECT().GetAuthorID(mock.Anything, id).Return(authorID, nil)
	m.repo.EXPECT().GetTitle(mock.Anything, id).Return("doomed", nil)
	m.repo.EXPECT().Delete(mock.Anything, id, authorID, false).Return(nil, nil)
	m.authz.EXPECT().Can(mock.Anything, mock.Anything, mock.Anything).Return(false).Maybe()
	m.auditRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	m.uploadSvc.EXPECT().Delete(mock.Anything).Return().Maybe()
	echoes.EXPECT().ClearEchoCache(mock.Anything).Return(nil).Once()

	// when the journal is deleted
	require.NoError(t, svc.DeleteJournal(ctx, id, authorID))

	// then the og entry is gone rather than lingering out its ttl
	_, err := mgr.Get[string](ctx, key)
	assert.Error(t, err)

	// and the echoes list is cleared, which the strict mock asserts on cleanup
}
