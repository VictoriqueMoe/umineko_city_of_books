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

	UserRole = Namespace{Prefix: "role:", TTL: 0}

	Setting = Namespace{Prefix: "setting:", TTL: 0}

	MysteryTopDetectives = Namespace{Prefix: "mystery:top-detectives", TTL: 0}
	MysteryTopGMs        = Namespace{Prefix: "mystery:top-gms", TTL: 0}
	GameTopWinners       = Namespace{Prefix: "game:top-winners:", TTL: 0}
	VanityAssignments    = Namespace{Prefix: "vanity:assignments", TTL: 0}

	SecretHolders = Namespace{Prefix: "secret:holders:", TTL: 0}
	SecretSolved  = Namespace{Prefix: "secret:solved:", TTL: 0}
)

func (n Namespace) Key(parts ...string) string {
	return n.Prefix + strings.Join(parts, ":")
}
