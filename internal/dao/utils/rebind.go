package utils

import (
	"strconv"
	"strings"
)

func Rebind(query string) string {
	var (
		b strings.Builder
		n int
	)
	b.Grow(len(query) + 16)
	for i := range len(query) {
		c := query[i]
		if c == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
