package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "simple key", key: "avatars/abc_123.webp", valid: true},
		{name: "flat key", key: "file.txt", valid: true},
		{name: "nested key", key: "a/b/c.png", valid: true},
		{name: "empty", key: ""},
		{name: "leading slash", key: "/avatars/a.png"},
		{name: "trailing slash", key: "avatars/"},
		{name: "parent segment", key: "../etc/passwd"},
		{name: "parent segment in the middle", key: "avatars/../../secret"},
		{name: "current segment", key: "./a.png"},
		{name: "double slash", key: "avatars//a.png"},
		{name: "backslash", key: "avatars\\a.png"},
		{name: "windows drive", key: "C:/a.png"},
		{name: "alternate data stream", key: "avatars/a.png:stream"},
		{name: "nul byte", key: "avatars/a\x00.png"},
		{name: "hidden staging segment", key: ".staging/upload-1/a.png"},
		{name: "hidden file", key: "avatars/.a.png.tmp-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := ValidateKey(tc.key)

			// then
			if tc.valid {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrInvalidKey)
		})
	}
}
