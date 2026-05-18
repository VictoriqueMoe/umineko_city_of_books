package contentfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalise_StripsFormatChars(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"soft hyphen", "hel\u00adlo", "hello"},
		{"zero-width space", "f\u200buck", "fuck"},
		{"zwnj and zwj", "a\u200cb\u200dc", "abc"},
		{"bom", "\uFEFFleading", "leading"},
		{"word joiner", "trail\u2060ing", "trailing"},
		{"plain ascii unchanged", "plain", "plain"},
		{"empty", "", ""},
		{"fullwidth letters folded", "ｆｕｃk", "fuck"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := Normalise(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormaliseLiteral_StripsWhitespace(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"spaced letters", "f u c k", "fuck"},
		{"tab and newline", "f\tu\nc k", "fuck"},
		{"leading and trailing spaces", "  hello  world ", "helloworld"},
		{"mixed invisible and spaces", "f\u00adu c\u200bk", "fuck"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := NormaliseLiteral(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}
