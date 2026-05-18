package contentfilter

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func Normalise(s string) string {
	if s == "" {
		return s
	}
	folded := norm.NFKC.String(s)
	var b strings.Builder
	b.Grow(len(folded))
	for _, r := range folded {
		if unicode.Is(unicode.Cf, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func NormaliseLiteral(s string) string {
	n := Normalise(s)
	if n == "" {
		return n
	}
	var b strings.Builder
	b.Grow(len(n))
	for _, r := range n {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
