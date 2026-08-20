package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	bingShaped   = `{"creationTime":"2026-08-20","prefixes":[{"ipv4Prefix":"52.167.144.0/24"},{"ipv4Prefix":"157.55.39.0/24"}]}`
	googleShaped = `{"creationTime":"2026-08-20","prefixes":[{"ipv6Prefix":"2001:4860:4801:10::/64"},{"ipv4Prefix":"66.249.64.0/27"}]}`
)

func serveFeeds(t *testing.T) *httptest.Server {
	t.Helper()

	bodies := map[string]string{
		"/bing":   bingShaped,
		"/google": googleShaped,
		"/html":   "<html>nope</html>",
		"/empty":  `{"creationTime":"x","prefixes":[]}`,
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

func newService(t *testing.T, server *httptest.Server, feeds string) (*Service, *settings.MockService) {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingCrawlerFeeds).Return(feeds).Maybe()

	svc := New(settingsSvc, cache.NewManager(engines.NewInMemory(0)))
	svc.client = server.Client()

	return svc, settingsSvc
}

func TestParseSources(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []Source
	}{
		{
			name: "one entry per line",
			raw:  "microsoft=https://www.bing.com/toolbox/bingbot.json\ngoogle=https://example.com/googlebot.json",
			want: []Source{
				{Name: "microsoft", URL: "https://www.bing.com/toolbox/bingbot.json"},
				{Name: "google", URL: "https://example.com/googlebot.json"},
			},
		},
		{
			name: "a comma in a query string is part of the url, not a separator",
			raw:  "mine=https://example.com/ranges.json?ids=1,2",
			want: []Source{
				{Name: "mine", URL: "https://example.com/ranges.json?ids=1,2"},
			},
		},
		{
			name: "an equals in a query string survives, because only the first one splits",
			raw:  "mine=https://example.com/r.json?a=b&c=d",
			want: []Source{
				{Name: "mine", URL: "https://example.com/r.json?a=b&c=d"},
			},
		},
		{
			name: "plain http is refused, because an allowlist fetched in the clear can be tampered with",
			raw:  "insecure=http://example.com/ranges.json",
			want: nil,
		},
		{
			name: "an entry with no name is skipped",
			raw:  "=https://example.com/ranges.json",
			want: nil,
		},
		{
			name: "an entry with no url is skipped",
			raw:  "microsoft",
			want: nil,
		},
		{
			name: "blank lines are ignored",
			raw:  "\n\n  \n",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a setting value

			// when
			got := parseSources(tc.raw)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRefresh_MergesEveryFeed(t *testing.T) {
	// given two feeds in the published format, one v4 only and one mixed
	server := serveFeeds(t)
	svc, _ := newService(t, server, "microsoft="+server.URL+"/bing\ngoogle="+server.URL+"/google")

	// when
	count, err := svc.Refresh(context.Background())

	// then every range from both feeds is held
	require.NoError(t, err)
	assert.Equal(t, 4, count)
	assert.Len(t, svc.Ranges(context.Background()), 4)
}

func TestRefresh_ParsesBothPrefixKeys(t *testing.T) {
	// given google's feed, which publishes mostly ipv6
	server := serveFeeds(t)
	svc, _ := newService(t, server, "google="+server.URL+"/google")

	// when
	_, err := svc.Refresh(context.Background())
	require.NoError(t, err)

	// then the ipv6Prefix entry survives, not just the ipv4Prefix one
	assert.True(t, containsAddr(svc.Ranges(context.Background()), "2001:4860:4801:10::1"))
	assert.True(t, containsAddr(svc.Ranges(context.Background()), "66.249.64.5"))
}

func TestRefresh_KeepsWorkingWhenOneFeedIsDown(t *testing.T) {
	// given one healthy feed and one that errors
	server := serveFeeds(t)
	svc, _ := newService(t, server, "good="+server.URL+"/bing\nbad="+server.URL+"/missing")

	// when
	count, err := svc.Refresh(context.Background())

	// then a single broken feed must not discard the ranges from the others
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRefresh_ErrorsWhenEveryFeedFails(t *testing.T) {
	// given
	server := serveFeeds(t)
	svc, _ := newService(t, server, "bad="+server.URL+"/missing")

	// when
	_, err := svc.Refresh(context.Background())

	// then the job reports it rather than silently emptying the allowlist
	require.Error(t, err)
}

func TestRefresh_KeepsTheOldRangesWhenEverythingFails(t *testing.T) {
	// given ranges loaded successfully once
	server := serveFeeds(t)
	svc, settingsSvc := newService(t, server, "good="+server.URL+"/bing")
	_, err := svc.Refresh(context.Background())
	require.NoError(t, err)

	// when every feed later fails
	settingsSvc.ExpectedCalls = nil
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingCrawlerFeeds).Return("bad=" + server.URL + "/missing").Maybe()
	_, err = svc.Refresh(context.Background())

	// then a crawler is not suddenly blocked because a feed had an outage
	require.Error(t, err)
	assert.Len(t, svc.Ranges(context.Background()), 2)
}

func TestRefresh_EmptySettingClearsTheRanges(t *testing.T) {
	// given a feed that was loaded once
	server := serveFeeds(t)
	svc, settingsSvc := newService(t, server, "good="+server.URL+"/bing")
	_, err := svc.Refresh(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, svc.Ranges(context.Background()))

	// when the operator removes every feed
	settingsSvc.ExpectedCalls = nil
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingCrawlerFeeds).Return("").Maybe()
	count, err := svc.Refresh(context.Background())

	// then nothing is allowlisted any more
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, svc.Ranges(context.Background()))
}

func TestRanges_IsEmptyBeforeTheFirstRefresh(t *testing.T) {
	// given a service the job has not run yet
	server := serveFeeds(t)
	svc, _ := newService(t, server, "")

	// then the request path must not blow up
	assert.Empty(t, svc.Ranges(context.Background()))
}

func TestValidator(t *testing.T) {
	server := serveFeeds(t)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "a real feed passes", value: "microsoft=" + server.URL + "/bing"},
		{name: "an empty setting is allowed", value: ""},
		{name: "a malformed entry is rejected", value: "just-some-text", wantErr: true},
		{name: "plain http is rejected", value: "x=http://example.com/a.json", wantErr: true},
		{name: "a url that is not json is rejected", value: "x=" + server.URL + "/html", wantErr: true},
		{name: "json with no usable prefixes is rejected", value: "x=" + server.URL + "/empty", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, _ := newService(t, server, "")

			// when
			err := Validator(svc)(context.Background(), tc.value)

			// then a bad url is caught on save rather than failing quietly in a job at 3am
			if tc.wantErr {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
		})
	}
}

func containsAddr(ranges []netip.Prefix, raw string) bool {
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return false
	}

	for _, prefix := range ranges {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}
