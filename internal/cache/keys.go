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

	Key struct {
		name string
		ttl  time.Duration
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
	UserIP   = Namespace{Prefix: "user:ip:", TTL: noExpiry}

	Setting = Namespace{Prefix: "setting:", TTL: noExpiry}

	MysteryTopDetectives = Namespace{Prefix: "mystery:top-detectives", TTL: noExpiry}
	MysteryTopGMs        = Namespace{Prefix: "mystery:top-gms", TTL: noExpiry}
	GameTopWinners       = Namespace{Prefix: "game:top-winners:", TTL: noExpiry}
	VanityAssignments    = Namespace{Prefix: "vanity:assignments", TTL: noExpiry}

	SiteInfo = Namespace{Prefix: "siteinfo", TTL: minute}

	PostCornerCounts = Namespace{Prefix: "post:corner-counts", TTL: noExpiry}
	ArtCornerCounts  = Namespace{Prefix: "art:corner-counts", TTL: noExpiry}

	RolePermissions       = Namespace{Prefix: "authz:role-perms", TTL: minute}
	VanityRolePermissions = Namespace{Prefix: "authz:vanity-perms", TTL: minute}
	UserVanityRoleIDs     = Namespace{Prefix: "authz:user-vanity:", TTL: minute}

	UserBlockedIDs = Namespace{Prefix: "block:ids:", TTL: noExpiry}

	ChatBannedWordsGlobal = Namespace{Prefix: "chat:banned-words:global", TTL: noExpiry}
	ChatBannedWordsRoom   = Namespace{Prefix: "chat:banned-words:room:", TTL: noExpiry}

	ChatbotBasePrompts    = Namespace{Prefix: "chatbot:base-prompts", TTL: noExpiry}
	ChatbotBasePromptByID = Namespace{Prefix: "chatbot:base-prompt:", TTL: noExpiry}
	ChatbotSummonCooldown = Namespace{Prefix: "chatbot:summon-cooldown:"}
	ChatbotRefusalNotice  = Namespace{Prefix: "chatbot:refusal-notice:", TTL: 10 * minute}

	SecretHolders = Namespace{Prefix: "secret:holders:", TTL: noExpiry}
	SecretSolved  = Namespace{Prefix: "secret:solved:", TTL: noExpiry}

	GiphyResponse         = Namespace{Prefix: "giphy:response:", TTL: hour}
	GiphyGifUser          = Namespace{Prefix: "giphy:gif-user:", TTL: week}
	GiphyRateLimitedUntil = Namespace{Prefix: "giphy:rate-limited-until", TTL: noExpiry}

	OpenAIRateLimitedUntil = Namespace{Prefix: "openai:rate-limited-until", TTL: noExpiry}

	StreamThumbnailSlot = Namespace{Prefix: "stream:thumbnail-slot:", TTL: 20 * time.Second}

	RateLimitRegister               = Namespace{Prefix: "ratelimit:register:", TTL: minute}
	RateLimitLogin                  = Namespace{Prefix: "ratelimit:login:", TTL: minute}
	RateLimitResetPassword          = Namespace{Prefix: "ratelimit:reset-password:", TTL: minute}
	RateLimitForgotPassword         = Namespace{Prefix: "ratelimit:forgot-password:", TTL: hour}
	RateLimitSetEmail               = Namespace{Prefix: "ratelimit:set-email:", TTL: hour}
	RateLimitResendVerification     = Namespace{Prefix: "ratelimit:resend-verification:", TTL: hour}
	RateLimitChatSend               = Namespace{Prefix: "ratelimit:chat-send:", TTL: 5 * time.Second}
	RateLimitLinkPreview            = Namespace{Prefix: "ratelimit:link-preview:", TTL: minute}
	RateLimitStreamViewerToken      = Namespace{Prefix: "ratelimit:stream-viewer-token:", TTL: minute}
	RateLimitStreamThumbnail        = Namespace{Prefix: "ratelimit:stream-thumbnail:", TTL: minute}
	RateLimitStreamCredentialsReset = Namespace{Prefix: "ratelimit:stream-credentials-reset:", TTL: minute}
)

func (n Namespace) Key(parts ...string) Key {
	return Key{name: n.Prefix + strings.Join(parts, ":"), ttl: n.TTL}
}

func (k Key) String() string {
	return k.name
}

func (k Key) WithTTL(ttl time.Duration) Key {
	k.ttl = ttl

	return k
}
