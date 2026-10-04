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

const (
	noExpiry time.Duration = 0
	minute                 = time.Minute
	hour                   = time.Hour
	day                    = 24 * hour
	week                   = 7 * day
)

var (
	OGMeta  = Namespace{Prefix: "og:meta:", TTL: 5 * minute}
	OGImage = Namespace{Prefix: "og:image:", TTL: day}

	HomeEchoes = Namespace{Prefix: "home:echoes:", TTL: day}

	LinkPreview = Namespace{Prefix: "linkpreview:", TTL: day}

	QuoteCharacters = Namespace{Prefix: "quote:characters:", TTL: week}
	QuoteByAudioID  = Namespace{Prefix: "quote:audio:", TTL: week}
	QuoteByIndex    = Namespace{Prefix: "quote:index:", TTL: week}

	DroneBL = Namespace{Prefix: "dronebl:", TTL: day}

	// CrawlerRanges refilled by a job rather than invalidated by a write, to keep third party fetches off the request path
	CrawlerRanges = Namespace{Prefix: "dronebl:crawler-ranges", TTL: noExpiry}

	UserRole = Namespace{Prefix: "role:", TTL: noExpiry}

	Setting = Namespace{Prefix: "setting:", TTL: noExpiry}

	MysteryTopDetectives = Namespace{Prefix: "mystery:top-detectives", TTL: noExpiry}
	MysteryTopGMs        = Namespace{Prefix: "mystery:top-gms", TTL: noExpiry}
	GameTopWinners       = Namespace{Prefix: "game:top-winners:", TTL: noExpiry}
	VanityAssignments    = Namespace{Prefix: "vanity:assignments", TTL: noExpiry}

	RolePermissions       = Namespace{Prefix: "authz:role-perms", TTL: minute}
	VanityRolePermissions = Namespace{Prefix: "authz:vanity-perms", TTL: minute}
	UserVanityRoleIDs     = Namespace{Prefix: "authz:user-vanity:", TTL: minute}

	ChatbotBasePrompts    = Namespace{Prefix: "chatbot:base-prompts", TTL: noExpiry}
	ChatbotBasePromptByID = Namespace{Prefix: "chatbot:base-prompt:", TTL: noExpiry}

	SecretHolders = Namespace{Prefix: "secret:holders:", TTL: noExpiry}
	SecretSolved  = Namespace{Prefix: "secret:solved:", TTL: noExpiry}

	GiphyResponse = Namespace{Prefix: "giphy:response:", TTL: noExpiry}
	GiphyGifUser  = Namespace{Prefix: "giphy:gif-user:", TTL: week}
)

func (n Namespace) Key(parts ...string) string {
	return n.Prefix + strings.Join(parts, ":")
}
