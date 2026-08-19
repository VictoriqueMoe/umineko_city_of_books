package media

import (
	"net/netip"
	"testing"

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

func TestIsPublicAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{name: "loopback v4", addr: "127.0.0.1", want: false},
		{name: "loopback v4 upper range", addr: "127.255.255.254", want: false},
		{name: "loopback v6", addr: "::1", want: false},
		{name: "rfc1918 ten dot", addr: "10.0.0.5", want: false},
		{name: "rfc1918 172.16 lower bound", addr: "172.16.0.1", want: false},
		{name: "rfc1918 172.31 upper bound", addr: "172.31.255.254", want: false},
		{name: "rfc1918 192.168", addr: "192.168.1.1", want: false},
		{name: "link local v4 metadata endpoint", addr: "169.254.169.254", want: false},
		{name: "link local v6", addr: "fe80::1", want: false},
		{name: "unique local fc00", addr: "fc00::1", want: false},
		{name: "unique local fd00", addr: "fd12:3456:789a::1", want: false},
		{name: "ipv4 mapped private", addr: "::ffff:10.0.0.1", want: false},
		{name: "ipv4 mapped loopback", addr: "::ffff:127.0.0.1", want: false},
		{name: "ipv4 mapped link local", addr: "::ffff:169.254.169.254", want: false},
		{name: "unspecified v4", addr: "0.0.0.0", want: false},
		{name: "unspecified v6", addr: "::", want: false},
		{name: "multicast v4", addr: "224.0.0.1", want: false},
		{name: "multicast v6", addr: "ff02::1", want: false},
		{name: "public v4 cloudflare", addr: "1.1.1.1", want: true},
		{name: "public v4 google", addr: "8.8.8.8", want: true},
		{name: "public v4 just outside rfc1918", addr: "172.32.0.1", want: true},
		{name: "public v6", addr: "2606:4700:4700::1111", want: true},
		{name: "public ipv4 mapped", addr: "::ffff:8.8.8.8", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			addr, err := netip.ParseAddr(tc.addr)
			require.NoError(t, err)

			// when
			got := isPublicAddr(addr)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsPublicAddr_InvalidAddrIsBlocked(t *testing.T) {
	// given
	var addr netip.Addr

	// when
	got := isPublicAddr(addr)

	// then
	assert.False(t, got)
}

func TestBlockNonPublicAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		blocked bool
	}{
		{name: "loopback v4 with port", address: "127.0.0.1:8080", blocked: true},
		{name: "loopback v6 with port", address: "[::1]:8080", blocked: true},
		{name: "docker bridge host", address: "172.17.0.1:5432", blocked: true},
		{name: "metadata endpoint", address: "169.254.169.254:80", blocked: true},
		{name: "ipv4 mapped private with port", address: "[::ffff:192.168.0.10]:443", blocked: true},
		{name: "unparsable address", address: "not-an-address", blocked: true},
		{name: "unresolved hostname", address: "internal-db:5432", blocked: true},
		{name: "public v4 with port", address: "1.1.1.1:443", blocked: false},
		{name: "public v6 with port", address: "[2606:4700:4700::1111]:443", blocked: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			address := tc.address

			// when
			err := blockNonPublicAddress("tcp", address, nil)

			// then
			if tc.blocked {
				assert.ErrorIs(t, err, errBlockedAddress)
				return
			}

			assert.NoError(t, err)
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
