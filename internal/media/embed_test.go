package media

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
