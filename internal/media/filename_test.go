package media

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeFilename(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  string
	}{
		{name: "an ordinary name is untouched", given: "beatrice theme.mp3", want: "beatrice theme.mp3"},
		{name: "surrounding whitespace goes", given: "  track.mp3  ", want: "track.mp3"},
		{name: "a unix path keeps only the last segment", given: "music/ost/track.mp3", want: "track.mp3"},
		{name: "a windows path keeps only the last segment", given: `C:\Users\ray\track.mp3`, want: "track.mp3"},
		{name: "a traversal attempt keeps only the last segment", given: "../../../etc/passwd", want: "passwd"},
		{name: "control characters are stripped", given: "track\x00\x1b[31m.mp3", want: "track[31m.mp3"},
		{name: "a newline cannot break a log line", given: "track\nDELETE FROM users.mp3", want: "trackDELETE FROM users.mp3"},
		{name: "unicode survives", given: "ベアトリーチェ.mp3", want: "ベアトリーチェ.mp3"},
		{name: "an empty name stays empty", given: "", want: ""},
		{name: "a bare path separator yields nothing", given: "/", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			// when
			got := SafeFilename(tc.given)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSafeFilename_CapsLengthWithoutSplittingARune(t *testing.T) {
	// given a name well past the cap, made of multi-byte runes
	given := strings.Repeat("ま", 400) + ".mp3"

	// when
	got := SafeFilename(given)

	// then it is cut to the rune cap and remains valid utf-8
	assert.Equal(t, maxFilenameRunes, len([]rune(got)))
	assert.Equal(t, strings.Repeat("ま", maxFilenameRunes), got)
}
