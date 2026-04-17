package banlist

import (
	"context"
	"sync"
	"testing"

	"umineko_city_of_books/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T, seed []repository.BannedGiphyRow) (*service, *repository.MockBannedGiphyRepository) {
	repo := repository.NewMockBannedGiphyRepository(t)
	repo.EXPECT().List(mock.Anything).Return(seed, nil).Once()
	svc, err := NewService(context.Background(), repo)
	require.NoError(t, err)
	return svc.(*service), repo
}

func TestNewService_LoadsSeed(t *testing.T) {
	// given
	seed := []repository.BannedGiphyRow{
		{Kind: "gif", Value: "abc123"},
		{Kind: "user", Value: "Larperine"},
	}

	// when
	svc, _ := newTestService(t, seed)

	// then
	assert.True(t, svc.ContainsGif("abc123"))
	assert.True(t, svc.ContainsUser("larperine"))
	assert.True(t, svc.ContainsUser("Larperine"))
	assert.False(t, svc.ContainsGif("other"))
}

func TestContainsGif_Empty(t *testing.T) {
	// given
	svc, _ := newTestService(t, nil)

	// when / then
	assert.False(t, svc.ContainsGif(""))
}

func TestContainsUser_CaseInsensitive(t *testing.T) {
	// given
	svc, _ := newTestService(t, []repository.BannedGiphyRow{{Kind: "user", Value: "Larperine"}})

	// when / then
	assert.True(t, svc.ContainsUser("LARPERINE"))
	assert.True(t, svc.ContainsUser("larperine"))
}

func TestAdd_Gif_UpdatesDBAndCache(t *testing.T) {
	// given
	svc, repo := newTestService(t, nil)
	repo.EXPECT().Add(mock.Anything, "gif", "xyz", "spam", mock.Anything).Return(nil)

	// when
	err := svc.Add(context.Background(), KindGif, "xyz", "spam", nil)

	// then
	require.NoError(t, err)
	assert.True(t, svc.ContainsGif("xyz"))
}

func TestAdd_User_NormalisesCase(t *testing.T) {
	// given
	svc, repo := newTestService(t, nil)
	repo.EXPECT().Add(mock.Anything, "user", "Larperine", "", mock.Anything).Return(nil)

	// when
	err := svc.Add(context.Background(), KindUser, "Larperine", "", nil)

	// then
	require.NoError(t, err)
	assert.True(t, svc.ContainsUser("larperine"))
}

func TestAdd_InvalidKind(t *testing.T) {
	// given
	svc, _ := newTestService(t, nil)

	// when
	err := svc.Add(context.Background(), Kind("other"), "val", "", nil)

	// then
	assert.ErrorIs(t, err, ErrInvalidKind)
}

func TestAdd_EmptyValue(t *testing.T) {
	// given
	svc, _ := newTestService(t, nil)

	// when
	err := svc.Add(context.Background(), KindGif, "   ", "", nil)

	// then
	assert.ErrorIs(t, err, ErrValueRequired)
}

func TestRemove_Gif(t *testing.T) {
	// given
	svc, repo := newTestService(t, []repository.BannedGiphyRow{{Kind: "gif", Value: "abc"}})
	repo.EXPECT().Remove(mock.Anything, "gif", "abc").Return(nil)

	// when
	err := svc.Remove(context.Background(), KindGif, "abc")

	// then
	require.NoError(t, err)
	assert.False(t, svc.ContainsGif("abc"))
}

func TestConcurrentContainsDuringAdd(t *testing.T) {
	// given
	svc, repo := newTestService(t, nil)
	repo.EXPECT().Add(mock.Anything, "gif", mock.Anything, "", mock.Anything).Return(nil).Maybe()

	// when — race the reader against writers
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = svc.Add(context.Background(), KindGif, string(rune('a'+i)), "", nil)
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = svc.ContainsGif("a")
			}
		}()
	}
	wg.Wait()

	// then — no race (run under -race if enabled); sanity check we added something
	assert.True(t, svc.ContainsGif("a"))
}
