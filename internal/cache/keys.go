package cache

import (
	"strings"
	"time"
)

type (
	Namespace struct {
		Prefix string
		TTL    time.Duration
	}
)

var (
	OGMeta  = Namespace{Prefix: "og:meta:", TTL: 5 * time.Minute}
	OGImage = Namespace{Prefix: "og:image:", TTL: 24 * time.Hour}
)

func (n Namespace) Key(parts ...string) string {
	return n.Prefix + strings.Join(parts, ":")
}
