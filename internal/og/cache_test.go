package og

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalMetaPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "plain path is untouched", path: "/theory/abc", want: "/theory/abc"},
		{name: "trailing slash is trimmed", path: "/theory/abc/", want: "/theory/abc"},
		{name: "repeated trailing slashes are trimmed", path: "/theory/abc///", want: "/theory/abc"},
		{name: "root stays root", path: "/", want: "/"},
		{name: "empty becomes root", path: "", want: "/"},
		{name: "casing collapses onto one key", path: "/Featherine/live", want: "/featherine/live"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a request path

			// when it is canonicalised for the cache key
			got := canonicalMetaPath(tc.path)

			// then variants of the same page collapse onto one key
			assert.Equal(t, tc.want, got)
		})
	}
}
