package linkpreview

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/media"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService(t *testing.T) *service {
	t.Helper()

	return &service{cache: cache.NewManager(engines.NewInMemory(0)), parse: media.ParseEmbed}
}

func newCountingService(t *testing.T, embed *media.Embed, delay time.Duration) (*service, *atomic.Int64) {
	t.Helper()

	calls := new(atomic.Int64)
	parse := func(rawURL string) *media.Embed {
		calls.Add(1)
		time.Sleep(delay)

		if embed == nil {
			return nil
		}

		copied := *embed
		copied.URL = rawURL

		return &copied
	}

	return &service{cache: cache.NewManager(engines.NewInMemory(0)), parse: parse}, calls
}

func TestResolve_RejectsAnythingThatIsNotAnAbsoluteHTTPURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"relative path", "/gallery/art"},
		{"no host", "https://"},
		{"javascript scheme", "javascript:alert(1)"},
		{"file scheme", "file:///etc/passwd"},
		{"ftp scheme", "ftp://example.com/x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a service whose fetcher must never be reached
			svc, calls := newCountingService(t, nil, 0)

			// when
			_, err := svc.Resolve(context.Background(), tc.url)

			// then
			require.ErrorIs(t, err, ErrInvalidURL)
			assert.Equal(t, int64(0), calls.Load(), "a rejected url must never reach the fetcher")
		})
	}
}

func TestResolve_RejectsAnOverlongURL(t *testing.T) {
	// given
	svc, calls := newCountingService(t, nil, 0)
	long := "https://example.com/" + string(make([]byte, maxURLLength))

	// when
	_, err := svc.Resolve(context.Background(), long)

	// then
	require.ErrorIs(t, err, ErrInvalidURL)
	assert.Equal(t, int64(0), calls.Load())
}

func TestResolve_YouTubeNeedsNoOutboundFetch(t *testing.T) {
	// given the real parser, which recognises youtube from the url shape alone
	svc := newService(t)

	// when
	got, err := svc.Resolve(context.Background(), "https://youtu.be/dQw4w9WgXcQ")

	// then
	require.NoError(t, err)
	assert.Equal(t, media.EmbedTypeYouTube, got.Type)
	assert.Equal(t, "dQw4w9WgXcQ", got.VideoID)
}

func TestResolve_ServesTheSecondCallFromCache(t *testing.T) {
	// given
	svc, calls := newCountingService(t, &media.Embed{Type: media.EmbedTypeLink, Title: "Rokkenjima"}, 0)

	// when the same url is resolved twice
	first, err := svc.Resolve(context.Background(), "https://example.com/a")
	require.NoError(t, err)
	second, err := svc.Resolve(context.Background(), "https://example.com/a")
	require.NoError(t, err)

	// then the origin was contacted once
	assert.Equal(t, "Rokkenjima", first.Title)
	assert.Equal(t, first, second)
	assert.Equal(t, int64(1), calls.Load(), "a cached preview must not refetch")
}

func TestResolve_CollapsesConcurrentCallsForTheSameURL(t *testing.T) {
	// given a slow fetch and many callers arriving at once on a cold cache
	svc, calls := newCountingService(t, &media.Embed{Type: media.EmbedTypeLink, Title: "Beatrice"}, 50*time.Millisecond)

	var (
		wg      sync.WaitGroup
		results = make([]string, 20)
	)

	// when
	for i := range results {
		wg.Add(1)

		go func() {
			defer wg.Done()

			got, err := svc.Resolve(context.Background(), "https://example.com/slow")
			if err == nil {
				results[i] = got.Title
			}
		}()
	}

	wg.Wait()

	// then exactly one fetch left the process and every caller got the same answer
	assert.Equal(t, int64(1), calls.Load(), "singleflight must collapse the stampede")
	for _, title := range results {
		assert.Equal(t, "Beatrice", title)
	}
}

func TestResolve_CachesAMissSoADeadLinkIsNotRefetchedEveryView(t *testing.T) {
	// given a url that yields nothing worth previewing
	svc, calls := newCountingService(t, nil, 0)

	// when
	first, err := svc.Resolve(context.Background(), "https://example.com/dead")
	require.NoError(t, err)
	_, err = svc.Resolve(context.Background(), "https://example.com/dead")
	require.NoError(t, err)

	// then the empty result is remembered rather than refetched on every view
	assert.Empty(t, first.Type)
	assert.Equal(t, "https://example.com/dead", first.URL)
	assert.Equal(t, int64(1), calls.Load(), "a miss must be cached too")
}

func TestResolve_DistinctURLsDoNotShareACacheEntry(t *testing.T) {
	// given
	svc := newService(t)

	// when
	a, err := svc.Resolve(context.Background(), "https://youtu.be/aaaaaaaaaaa")
	require.NoError(t, err)
	b, err := svc.Resolve(context.Background(), "https://youtu.be/bbbbbbbbbbb")
	require.NoError(t, err)

	// then
	assert.Equal(t, "aaaaaaaaaaa", a.VideoID)
	assert.Equal(t, "bbbbbbbbbbb", b.VideoID)
}

func TestResolve_TrimsSurroundingWhitespaceBeforeKeying(t *testing.T) {
	// given
	svc, calls := newCountingService(t, &media.Embed{Type: media.EmbedTypeLink, Title: "Same"}, 0)

	// when the same url arrives padded
	_, err := svc.Resolve(context.Background(), "https://example.com/a")
	require.NoError(t, err)
	_, err = svc.Resolve(context.Background(), "  https://example.com/a  ")
	require.NoError(t, err)

	// then padding must not create a second cache entry
	assert.Equal(t, int64(1), calls.Load())
}
