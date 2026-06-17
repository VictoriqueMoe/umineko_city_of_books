package og

import (
	"context"
	"strings"
	"testing"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testBaseHTML = `<head>
<title>When They Cry City of Books</title>
<meta name="description" content="A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms.">
<meta property="og:title" content="When They Cry City of Books">
<meta property="og:site_name" content="When They Cry City of Books">
<meta property="og:description" content="A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms.">
<meta property="og:url" content="https://example.com/">
<meta property="og:image" content="https://example.com/Featherine.jpg">
<meta property="og:image:type" content="image/jpeg">
<meta property="og:image:width" content="2734">
<meta property="og:image:height" content="1533">
<meta name="twitter:title" content="When They Cry City of Books">
<meta name="twitter:description" content="A social platform for fans of Umineko, Higurashi, and the wider When They Cry series. Post theories, solve mysteries, share fan art, chronicle read-throughs, ship pairings, write fanfiction, and chat in live rooms.">
<meta name="twitter:image" content="https://example.com/Featherine.jpg">
<link rel="canonical" href="https://example.com/">
</head>`

func newTestResolver(t *testing.T, ogDefaultImage string) *Resolver {
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return(ogDefaultImage)
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("")
	return &Resolver{settingsSvc: ss, baseHTML: testBaseHTML, baseURL: "https://example.com"}
}

func TestResolver_Resolve_DefaultImage(t *testing.T) {
	tests := []struct {
		name           string
		ogDefaultImage string
		path           string
		wantImage      string
		wantSizeTags   bool
	}{
		{name: "builtin image when unset", ogDefaultImage: "", path: "/mysteries", wantImage: "https://example.com/Featherine.jpg", wantSizeTags: true},
		{name: "custom image on meta page", ogDefaultImage: "/uploads/branding/og_default_1.jpg", path: "/mysteries", wantImage: "https://example.com/uploads/branding/og_default_1.jpg", wantSizeTags: false},
		{name: "custom image on unknown page", ogDefaultImage: "/uploads/branding/og_default_1.jpg", path: "/some/unknown/page", wantImage: "https://example.com/uploads/branding/og_default_1.jpg", wantSizeTags: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := newTestResolver(t, tc.ogDefaultImage)

			// when
			html := r.Resolve(context.Background(), tc.path)

			// then
			assert.Contains(t, html, `property="og:image" content="`+tc.wantImage+`"`)
			assert.Contains(t, html, `name="twitter:image" content="`+tc.wantImage+`"`)
			assert.Equal(t, tc.wantSizeTags, strings.Contains(html, "og:image:width"))
		})
	}
}

func TestResolver_Resolve_SiteName(t *testing.T) {
	// given
	ss := settings.NewMockService(t)
	ss.EXPECT().Get(mock.Anything, config.SettingOGDefaultImage).Return("")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteName).Return("Custom Site")
	ss.EXPECT().Get(mock.Anything, config.SettingSiteDescription).Return("A custom description")
	r := &Resolver{settingsSvc: ss, baseHTML: testBaseHTML, baseURL: "https://example.com"}

	// when
	html := r.Resolve(context.Background(), "/")

	// then
	assert.Contains(t, html, `<title>Custom Site</title>`)
	assert.Contains(t, html, `property="og:title" content="Custom Site"`)
	assert.Contains(t, html, `property="og:site_name" content="Custom Site"`)
	assert.Contains(t, html, `property="og:description" content="A custom description"`)
}

func TestResolver_OGImageURL(t *testing.T) {
	r := &Resolver{baseURL: "https://example.com"}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "webp upload rewritten to jpeg endpoint", in: "https://example.com/uploads/posts/abc.webp", want: "https://example.com/og-image/posts/abc.jpg"},
		{name: "uppercase extension rewritten", in: "https://example.com/uploads/posts/abc.WEBP", want: "https://example.com/og-image/posts/abc.jpg"},
		{name: "non webp upload untouched", in: "https://example.com/uploads/posts/abc.gif", want: "https://example.com/uploads/posts/abc.gif"},
		{name: "external url untouched", in: "https://media.giphy.com/abc.webp", want: "https://media.giphy.com/abc.webp"},
		{name: "default image untouched", in: "https://example.com/Featherine.jpg", want: "https://example.com/Featherine.jpg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := r.ogImageURL(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
