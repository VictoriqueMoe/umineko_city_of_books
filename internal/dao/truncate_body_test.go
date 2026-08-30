package dao

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncateBody_CutsPreviewsByRunesAndNeverHalvesOne(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		maxLen int
		want   string
	}{
		{
			name:   "an empty body is untouched",
			body:   "",
			maxLen: 200,
			want:   "",
		},
		{
			name:   "a short ascii body is untouched",
			body:   "a short post",
			maxLen: 200,
			want:   "a short post",
		},
		{
			name:   "a body exactly on the limit is untouched",
			body:   strings.Repeat("a", 200),
			maxLen: 200,
			want:   strings.Repeat("a", 200),
		},
		{
			name:   "an ascii body over the limit keeps the ellipsis",
			body:   strings.Repeat("a", 201),
			maxLen: 200,
			want:   strings.Repeat("a", 200) + "...",
		},
		{
			name:   "a japanese body the byte cut would have chopped is untouched",
			body:   strings.Repeat("さ", 150),
			maxLen: 200,
			want:   strings.Repeat("さ", 150),
		},
		{
			name:   "a japanese body over the limit keeps two hundred characters",
			body:   strings.Repeat("う", 300),
			maxLen: 200,
			want:   strings.Repeat("う", 200) + "...",
		},
		{
			name:   "an emoji body over the limit is never halved",
			body:   strings.Repeat("🎉", 300),
			maxLen: 200,
			want:   strings.Repeat("🎉", 200) + "...",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := truncateBody(tc.body, tc.maxLen)

			// then
			assert.Equal(t, tc.want, got)
			assert.True(t, utf8.ValidString(got))
			assert.LessOrEqual(t, utf8.RuneCountInString(strings.TrimSuffix(got, "...")), tc.maxLen)
		})
	}
}
