package homefeed_test

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/homefeed"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_ClearEchoCache(t *testing.T) {
	t.Run("a rebuilt list no longer carries the deleted entry", func(t *testing.T) {
		// given a homepage whose echoes are cached from a first read
		ctx := context.Background()
		repo := repository.NewMockHomeFeedRepository(t)
		svc := homefeed.NewService(repo, ws.NewHub(), cache.New())

		repo.EXPECT().ListRecentActivity(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
		repo.EXPECT().ListRecentMembers(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
		repo.EXPECT().ListPublicRooms(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
		repo.EXPECT().ListCornerActivity24h(mock.Anything).Return(nil, nil).Maybe()

		doomed := []repository.HomeEchoRow{{Kind: "theory", ID: uuid.New(), Title: "doomed", AuthorID: uuid.New()}}
		repo.EXPECT().ListEchoes(mock.Anything, mock.Anything, mock.Anything).Return(doomed, nil).Once()

		first, err := svc.HomeActivity(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, first.Echoes)

		// and a second read served from cache, proving the entry is sticky
		second, err := svc.HomeActivity(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, second.Echoes)

		// when the theory is deleted and the cache is cleared
		require.NoError(t, svc.ClearEchoCache(ctx))
		repo.EXPECT().ListEchoes(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

		// then the next read rebuilds from the database instead of serving the deleted title
		third, err := svc.HomeActivity(ctx)
		require.NoError(t, err)
		assert.Empty(t, third.Echoes)
	})

	t.Run("a nil cache is tolerated", func(t *testing.T) {
		// given a service with no cache wired in, as in a unit test
		repo := repository.NewMockHomeFeedRepository(t)
		svc := homefeed.NewService(repo, ws.NewHub(), nil)

		// when the cache is cleared
		err := svc.ClearEchoCache(context.Background())

		// then it is a no-op rather than a panic
		assert.NoError(t, err)
	})
}
