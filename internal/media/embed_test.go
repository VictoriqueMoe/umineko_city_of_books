package media

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractYouTubeID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "watch url", url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "short url", url: "https://youtu.be/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "embed url", url: "https://youtube.com/embed/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "shorts url", url: "https://www.youtube.com/shorts/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "mobile subdomain", url: "https://m.youtube.com/watch?v=dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "spoofed host is rejected", url: "https://notyoutube.com/watch?v=dQw4w9WgXcQ", want: ""},
		{name: "host in path is rejected", url: "https://evil.com/youtube.com/watch?v=dQw4w9WgXcQ", want: ""},
		{name: "non youtube url", url: "https://example.com/page", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			rawURL := tc.url

			// when
			got := extractYouTubeID(rawURL)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLinkEmbed_DropsAResultWithNothingWorthShowing(t *testing.T) {
	cases := []struct {
		name string
		og   map[string]string
		want bool
	}{
		{"no og tags at all", map[string]string{}, false},
		{"only a site name is not worth a card", map[string]string{"og:site_name": "Example"}, false},
		{"a title alone is enough", map[string]string{"og:title": "Rokkenjima"}, true},
		{"a description alone is enough", map[string]string{"og:description": "the golden witch"}, true},
		{"an image alone is enough", map[string]string{"og:image": "https://example.com/a.png"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given

			// when
			got := linkEmbed("https://example.com", tc.og)

			// then an empty card must never be stored or rendered
			if !tc.want {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, EmbedTypeLink, got.Type)
			assert.Equal(t, "https://example.com", got.URL)
		})
	}
}

func TestParseOGFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want map[string]string
	}{
		{
			name: "tags in the head",
			html: `<html><head><meta property="og:title" content="Rokkenjima"/></head><body></body></html>`,
			want: map[string]string{"og:title": "Rokkenjima"},
		},
		{
			name: "tags streamed into the body",
			html: `<html><head></head><body><div hidden id="S:1"><title>Rokkenjima</title><meta property="og:title" content="Rokkenjima"/><meta property="og:image" content="https://example.com/a.png"/></div></body></html>`,
			want: map[string]string{"og:title": "Rokkenjima", "og:image": "https://example.com/a.png"},
		},
		{
			name: "the head wins over a later duplicate in the body",
			html: `<html><head><meta property="og:title" content="Rokkenjima"/></head><body><meta property="og:title" content="Kinzo"/></body></html>`,
			want: map[string]string{"og:title": "Rokkenjima"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := strings.NewReader(tc.html)

			// when
			got, err := parseOGFromHTML(r)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseOGFromHTML_AConnectionThatDiesMidPageIsAnErrorNotAShortPage(t *testing.T) {
	// given a page whose title arrives before the connection resets
	reset := errors.New("connection reset by peer")
	r := io.MultiReader(strings.NewReader(`<html><head><meta property="og:title" content="Rokkenjima"/>`), iotest.ErrReader(reset))

	// when
	got, err := parseOGFromHTML(r)

	// then nothing half-read is handed on as a finished preview
	require.ErrorIs(t, err, reset)
	assert.Nil(t, got)
}

func TestLinkEmbed_CarriesEveryOpenGraphField(t *testing.T) {
	// given
	og := map[string]string{
		"og:title":       "Rokkenjima",
		"og:description": "an island",
		"og:image":       "https://example.com/a.png",
		"og:site_name":   "Example",
	}

	// when
	got := linkEmbed("https://example.com/x", og)

	// then
	require.NotNil(t, got)
	assert.Equal(t, "Rokkenjima", got.Title)
	assert.Equal(t, "an island", got.Desc)
	assert.Equal(t, "https://example.com/a.png", got.Image)
	assert.Equal(t, "Example", got.SiteName)
}
