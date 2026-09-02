package text

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestClampBytes(t *testing.T) {
	// given
	cases := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{name: "empty string", in: "", limit: 8, want: ""},
		{name: "under the limit", in: "abc", limit: 8, want: "abc"},
		{name: "exactly at the limit", in: "abcdefgh", limit: 8, want: "abcdefgh"},
		{name: "ascii cut", in: "abcdefghij", limit: 4, want: "abcd"},
		{name: "cut mid two-byte rune", in: "aé", limit: 2, want: "a"},
		{name: "cut mid three-byte rune", in: "aあ", limit: 3, want: "a"},
		{name: "cut mid four-byte rune", in: "a🎉", limit: 4, want: "a"},
		{name: "cut lands exactly on a rune boundary", in: "あい", limit: 3, want: "あ"},
		{name: "backs up past every byte of a lone rune", in: "あ", limit: 2, want: ""},
		{name: "multibyte string exactly at the limit", in: "あい", limit: 6, want: "あい"},
		{name: "limit of zero", in: "abc", limit: 0, want: ""},
		{name: "negative limit", in: "abc", limit: -5, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := ClampBytes(tc.in, tc.limit)

			// then
			assert.Equal(t, tc.want, got)
			assert.True(t, utf8.ValidString(got), "ClampBytes must never return invalid UTF-8")
			assert.LessOrEqual(t, len(got), max(tc.limit, 0))
		})
	}
}

func TestClampRunes(t *testing.T) {
	// given
	cases := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{name: "empty string", in: "", limit: 4, want: ""},
		{name: "under the limit", in: "abc", limit: 8, want: "abc"},
		{name: "exactly at the limit", in: "abcd", limit: 4, want: "abcd"},
		{name: "ascii cut", in: "abcdefghij", limit: 4, want: "abcd"},
		{name: "limit of one on ascii", in: "abc", limit: 1, want: "a"},
		{name: "limit of one on a multibyte-first string", in: "あいう", limit: 1, want: "あ"},
		{name: "cut after a two-byte rune", in: "aéb", limit: 2, want: "aé"},
		{name: "cut after a three-byte rune", in: "aあb", limit: 2, want: "aあ"},
		{name: "cut after a four-byte rune", in: "a🎉b", limit: 2, want: "a🎉"},
		{name: "cut lands exactly on a rune boundary", in: "あいう", limit: 2, want: "あい"},
		{name: "multibyte string exactly at the limit", in: "あたくし", limit: 4, want: "あたくし"},
		{name: "byte length far above the limit is kept whole", in: "🎉🎉", limit: 8, want: "🎉🎉"},
		{name: "limit of zero", in: "abc", limit: 0, want: ""},
		{name: "negative limit", in: "abc", limit: -5, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := ClampRunes(tc.in, tc.limit)

			// then
			assert.Equal(t, tc.want, got)
			assert.True(t, utf8.ValidString(got), "ClampRunes must never return invalid UTF-8 for valid input")
			assert.LessOrEqual(t, utf8.RuneCountInString(got), max(tc.limit, 0))
		})
	}
}

func TestClampRunes_InvalidUTF8(t *testing.T) {
	// given
	in := "a\xffb"

	// when
	got := ClampRunes(in, 2)

	// then
	assert.Equal(t, "a\xff", got)
	assert.Equal(t, 2, utf8.RuneCountInString(got))
}
