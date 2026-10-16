package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	RateLimitPolicy struct {
		Namespace cache.Namespace
		Limit     int64
		PerRoute  bool
		Message   string
	}

	RateLimitSubject func(ctx fiber.Ctx) (string, error)

	RateLimiter struct {
		cache *cache.Manager
	}
)

const (
	CredentialAttemptsPerMinute      = 10
	MailAttemptsPerHour              = 5
	ChatSendsPerWindow               = 5
	LinkPreviewsPerMinute            = 120
	StreamViewerTokensPerMinute      = 30
	StreamThumbnailsPerMinute        = 6
	StreamCredentialsResetsPerMinute = 5

	maxSubjectLength    = 192
	hashedSubjectPrefix = "sha256:"

	tooManyRequestsMessage   = "too many requests, please try again later"
	sendingTooQuicklyMessage = "You're sending messages too quickly. Wait a moment and try again."
)

var (
	CredentialAttempts      = RateLimitPolicy{Namespace: cache.RateLimitCredentials, Limit: CredentialAttemptsPerMinute, PerRoute: true, Message: tooManyRequestsMessage}
	MailAttempts            = RateLimitPolicy{Namespace: cache.RateLimitMail, Limit: MailAttemptsPerHour, PerRoute: true, Message: tooManyRequestsMessage}
	ChatSends               = RateLimitPolicy{Namespace: cache.RateLimitChatSend, Limit: ChatSendsPerWindow, Message: sendingTooQuicklyMessage}
	LinkPreviews            = RateLimitPolicy{Namespace: cache.RateLimitLinkPreview, Limit: LinkPreviewsPerMinute, PerRoute: true, Message: tooManyRequestsMessage}
	StreamViewerTokens      = RateLimitPolicy{Namespace: cache.RateLimitStreamViewerToken, Limit: StreamViewerTokensPerMinute, PerRoute: true, Message: tooManyRequestsMessage}
	StreamThumbnails        = RateLimitPolicy{Namespace: cache.RateLimitStreamThumbnail, Limit: StreamThumbnailsPerMinute, PerRoute: true, Message: tooManyRequestsMessage}
	StreamCredentialsResets = RateLimitPolicy{Namespace: cache.RateLimitStreamCredentialsReset, Limit: StreamCredentialsResetsPerMinute, PerRoute: true, Message: tooManyRequestsMessage}
)

func NewRateLimiter(cacheManager *cache.Manager) *RateLimiter {
	return &RateLimiter{cache: cacheManager}
}

func (l *RateLimiter) Limit(policy RateLimitPolicy, subject RateLimitSubject) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		who, err := subject(ctx)
		if err != nil {
			return err
		}

		key := policy.key(ctx, boundSubject(who))

		count, remaining, err := l.cache.IncrementWindow(ctx.Context(), key, policy.Namespace.TTL)
		if err != nil {
			logger.Ctx(ctx.Context()).Warn().
				Err(err).
				Str("policy", policy.Namespace.Prefix).
				Str("route", ctx.Route().Path).
				Msg("rate limit store failed, letting the request through")

			return ctx.Next()
		}

		if count <= policy.Limit {
			return ctx.Next()
		}

		ctx.Set(fiber.HeaderRetryAfter, strconv.FormatInt(retryAfterSeconds(remaining), 10))

		return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": policy.Message,
		})
	}
}

func ByClientIP(ctx fiber.Ctx) (string, error) {
	return "ip:" + clientIP(ctx), nil
}

func ByAccount(field string) RateLimitSubject {
	return func(ctx fiber.Ctx) (string, error) {
		value, ok := normalisedBodyField(ctx, field)
		if !ok {
			return "account:ip:" + clientIP(ctx), nil
		}

		return "account:" + field + ":" + value, nil
	}
}

func ByUser(ctx fiber.Ctx) (string, error) {
	userID, ok := ctx.Locals("userID").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return "", ErrRateLimitWithoutUser
	}

	return "user:" + userID.String(), nil
}

func (p RateLimitPolicy) key(ctx fiber.Ctx, subject string) string {
	if !p.PerRoute {
		return p.Namespace.Key(subject)
	}

	route := ctx.Route()

	return p.Namespace.Key(route.Method, route.Path, subject)
}

func boundSubject(subject string) string {
	if len(subject) <= maxSubjectLength && !strings.HasPrefix(subject, hashedSubjectPrefix) {
		return subject
	}

	sum := sha256.Sum256([]byte(subject))

	return hashedSubjectPrefix + hex.EncodeToString(sum[:])
}

func retryAfterSeconds(remaining time.Duration) int64 {
	seconds := int64((remaining + time.Second - 1) / time.Second)

	return max(seconds, 1)
}

func normalisedBodyField(ctx fiber.Ctx, field string) (string, bool) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return "", false
	}

	var value string
	if err := json.Unmarshal(body[field], &value); err != nil {
		return "", false
	}

	value = strings.ToLower(strings.TrimSpace(value))

	return value, value != ""
}

func clientIP(ctx fiber.Ctx) string {
	if ip, ok := ctx.Locals("client_ip").(string); ok && ip != "" {
		return ip
	}

	return ctx.IP()
}
