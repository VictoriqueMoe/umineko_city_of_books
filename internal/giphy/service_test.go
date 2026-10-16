package giphy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engine"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/httpclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T, handler http.HandlerFunc) *service {
	t.Helper()

	return newTestServiceWithCache(t, cache.NewManager(engines.NewInMemory(0)), handler)
}

func newTestServiceWithCache(t *testing.T, cacheMgr *cache.Manager, handler http.HandlerFunc) *service {
	t.Helper()
	srv := httptest.NewTestServer(t, handler)
	transport := srv.Client().Transport

	return &service{
		apiKey:     "test-key",
		baseURL:    srv.URL,
		httpClient: httpclient.New(requestTimeout, httpclient.WithTransport(transport)),
		cache:      cacheMgr,
	}
}

func rateLimitedHandler(calls *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
	}
}

func TestService_Enabled(t *testing.T) {
	empty := &service{}
	assert.False(t, empty.Enabled())
	withKey := &service{apiKey: "x"}
	assert.True(t, withKey.Enabled())
}

func TestService_Search_DisabledShortCircuits(t *testing.T) {
	s := &service{}
	_, err := s.Search(context.Background(), "cats", 0, 0)
	assert.ErrorIs(t, err, ErrDisabled)
}

func TestService_Search_SendsExpectedParams(t *testing.T) {
	var seenPath, seenQuery string
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":[{"id":"abc","title":"cat","images":{"original":{"url":"https://media.giphy.com/media/abc/giphy.gif"}}}],"pagination":{"total_count":1,"count":1,"offset":0}}`))
	})

	resp, err := s.Search(context.Background(), "cats", 25, 15)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "abc", resp.Data[0].ID)
	assert.Equal(t, "/gifs/search", seenPath)
	assert.Contains(t, seenQuery, "q=cats")
	assert.Contains(t, seenQuery, "api_key=test-key")
	assert.Contains(t, seenQuery, "limit=15")
	assert.Contains(t, seenQuery, "offset=25")
	assert.Contains(t, seenQuery, "rating=pg-13")
	assert.Contains(t, seenQuery, "bundle=messaging_non_clips")
}

func TestService_Search_ClampsLimit(t *testing.T) {
	var seenQuery string
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{}`))
	})

	_, err := s.Search(context.Background(), "cats", 0, 999)
	require.NoError(t, err)
	assert.Contains(t, seenQuery, "limit=24")

	_, err = s.Search(context.Background(), "cats", 0, -1)
	require.NoError(t, err)
	assert.Contains(t, seenQuery, "limit=24")
}

func TestService_Search_NegativeOffsetClamped(t *testing.T) {
	var seenQuery string
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{}`))
	})

	_, err := s.Search(context.Background(), "cats", -10, 10)
	require.NoError(t, err)
	assert.Contains(t, seenQuery, "offset=0")
}

func TestService_Search_UpstreamError(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`nope`))
	})

	_, err := s.Search(context.Background(), "cats", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "418")
	assert.Contains(t, err.Error(), "nope")
}

func TestService_Search_BadJSON(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	})

	_, err := s.Search(context.Background(), "cats", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestService_Search_ContextCancelled(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Search(ctx, "cats", 0, 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context"))
}

func TestService_Trending_SendsExpectedParams(t *testing.T) {
	var seenPath, seenQuery string
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":[{"id":"t1"}]}`))
	})

	resp, err := s.Trending(context.Background(), 7, 30)
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "t1", resp.Data[0].ID)
	assert.Equal(t, "/gifs/trending", seenPath)
	assert.Contains(t, seenQuery, "api_key=test-key")
	assert.Contains(t, seenQuery, "limit=30")
	assert.Contains(t, seenQuery, "offset=7")
	assert.Contains(t, seenQuery, "rating=pg-13")
	assert.NotContains(t, seenQuery, "q=")
}

func TestService_Trending_Disabled(t *testing.T) {
	s := &service{}
	_, err := s.Trending(context.Background(), 0, 0)
	assert.ErrorIs(t, err, ErrDisabled)
}

func TestService_Search_CachesResultsPerQuery(t *testing.T) {
	var calls int
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"data":[{"id":"a"}]}`))
	})

	for range 3 {
		_, err := s.Search(context.Background(), "cats", 0, 24)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, calls, "same query should hit upstream only once")

	_, err := s.Search(context.Background(), "dogs", 0, 24)
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "different query should miss cache")

	_, err = s.Search(context.Background(), "cats", 25, 24)
	require.NoError(t, err)
	assert.Equal(t, 3, calls, "different offset should miss cache")
}

func TestService_Trending_CachesAcrossCalls(t *testing.T) {
	var calls int
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"data":[{"id":"t"}]}`))
	})

	for range 5 {
		_, err := s.Trending(context.Background(), 0, 24)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, calls)
}

func TestService_Search_Returns429AsRateLimitError(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := s.Search(context.Background(), "cats", 0, 24)
	var rl *RateLimitError
	require.ErrorAs(t, err, &rl)
	assert.True(t, rl.ResetAt.After(time.Now()))
	assert.True(t, rl.ResetAt.Before(time.Now().Add(2*time.Minute)))
}

func TestService_ShortCircuitsDuringRateLimit(t *testing.T) {
	var calls int
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := s.Search(context.Background(), "cats", 0, 24)
	var rl *RateLimitError
	require.ErrorAs(t, err, &rl)
	assert.Equal(t, 1, calls)

	for range 5 {
		_, err = s.Search(context.Background(), "dogs", 0, 24)
		require.ErrorAs(t, err, &rl)
	}
	assert.Equal(t, 1, calls, "subsequent calls during blackout must not hit upstream")
}

func TestService_CachedResultsStillServedDuringRateLimit(t *testing.T) {
	callCount := 0
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"a"}]}`))
			return
		}
		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := s.Search(context.Background(), "cats", 0, 24)
	require.NoError(t, err)

	_, err = s.Search(context.Background(), "dogs", 0, 24)
	var rl *RateLimitError
	require.ErrorAs(t, err, &rl)

	resp, err := s.Search(context.Background(), "cats", 0, 24)
	require.NoError(t, err, "cached 'cats' result should still be served during blackout")
	assert.Equal(t, "a", resp.Data[0].ID)
}

func TestService_RateLimitIsSharedThroughTheCache(t *testing.T) {
	// given two instances sharing one cache, as two processes share valkey
	shared := cache.NewManager(engines.NewInMemory(0))

	var firstCalls, secondCalls int
	first := newTestServiceWithCache(t, shared, rateLimitedHandler(&firstCalls))
	second := newTestServiceWithCache(t, shared, rateLimitedHandler(&secondCalls))

	_, err := first.Search(context.Background(), "cats", 0, 24)
	var tripped *RateLimitError
	require.ErrorAs(t, err, &tripped)

	// when
	_, err = second.Trending(context.Background(), 0, 24)

	// then
	var blocked *RateLimitError
	require.ErrorAs(t, err, &blocked)
	assert.Equal(t, 1, firstCalls)
	assert.Zero(t, secondCalls, "a ban tripped by one instance must hold back the other")
	assert.True(t, tripped.ResetAt.Equal(blocked.ResetAt))
}

func TestService_TripRateLimitNeverShortensABan(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		first       time.Time
		second      time.Time
		wantInForce time.Time
	}{
		{name: "a later reset extends the ban", first: now.Add(time.Minute), second: now.Add(5 * time.Minute), wantInForce: now.Add(5 * time.Minute)},
		{name: "an earlier reset leaves the ban alone", first: now.Add(5 * time.Minute), second: now.Add(time.Minute), wantInForce: now.Add(5 * time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
				t.Error("tripping the breaker must not call giphy")
			})
			ctx := context.Background()
			s.tripRateLimit(ctx, tt.first)

			// when
			inForce := s.tripRateLimit(ctx, tt.second)

			// then
			want := time.UnixMilli(tt.wantInForce.UnixMilli())
			assert.True(t, want.Equal(inForce))

			resetAt, blocked := s.rateLimitResetAt(ctx)
			assert.True(t, blocked)
			assert.True(t, want.Equal(resetAt))
		})
	}
}

func TestService_RateLimitFailsOpenWhenTheStoreFails(t *testing.T) {
	// given a cache whose store errors on every call
	storeErr := errors.New("connection refused")

	failing := engine.NewMockEngine(t)
	failing.EXPECT().Enabled().Return(true).Maybe()
	failing.EXPECT().Get(mock.Anything, mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, cache.GiphyResponse.Prefix)
	})).Return(nil, storeErr).Times(2)
	failing.EXPECT().Get(mock.Anything, cache.GiphyRateLimitedUntil.Key()).Return(nil, storeErr).Times(2)
	failing.EXPECT().ExtendDeadline(mock.Anything, cache.GiphyRateLimitedUntil.Key(), mock.Anything).Return(time.Time{}, storeErr).Times(2)

	var calls int
	s := newTestServiceWithCache(t, cache.NewManager(failing), rateLimitedHandler(&calls))

	// when
	for range 2 {
		_, err := s.Search(context.Background(), "cats", 0, 24)
		var rl *RateLimitError
		require.ErrorAs(t, err, &rl)
	}

	// then the breaker cannot hold, so every call reaches giphy
	assert.Equal(t, 2, calls)
}

func TestParseRateLimitReset(t *testing.T) {
	now := time.Unix(1_000_000, 0)

	h := http.Header{}
	h.Set("Retry-After", "120")
	assert.Equal(t, now.Add(2*time.Minute), parseRateLimitReset(h, now))

	h = http.Header{}
	h.Set("X-RateLimit-Reset", "1000500")
	assert.Equal(t, time.Unix(1_000_500, 0), parseRateLimitReset(h, now))

	h = http.Header{}
	assert.Equal(t, now.Add(defaultRateLimitHold), parseRateLimitReset(h, now))
}

func TestService_Search_DoesNotCacheErrors(t *testing.T) {
	var calls int
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`oops`))
	})

	for range 3 {
		_, err := s.Search(context.Background(), "cats", 0, 24)
		require.Error(t, err)
	}
	assert.Equal(t, 3, calls, "upstream errors must not be cached")
}
