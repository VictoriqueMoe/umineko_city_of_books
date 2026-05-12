package user

import (
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

const MaxDisplayNameRunes = 40

var (
	stripOnce   sync.Once
	stripPolicy *bluemonday.Policy
)

func stripper() *bluemonday.Policy {
	stripOnce.Do(func() {
		stripPolicy = bluemonday.StrictPolicy()
	})
	return stripPolicy
}

func ClampDisplayName(raw string) string {
	stripped := stripper().Sanitize(raw)
	stripped = strings.Join(strings.Fields(stripped), " ")
	runes := []rune(stripped)
	if len(runes) > MaxDisplayNameRunes {
		runes = runes[:MaxDisplayNameRunes]
	}
	return strings.TrimSpace(string(runes))
}
