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
	RateLimitSubject func(ctx fiber.Ctx) (string, error)

	RateLimiter struct {
		cache *cache.Manager
	}
)

const (
	maxSubjectLength    = 192
	hashedSubjectPrefix = "sha256:"

	tooManyRequestsMessage = "too many requests, please try again later"
)

func NewRateLimiter(cacheManager *cache.Manager) *RateLimiter {
	return &RateLimiter{cache: cacheManager}
}

func (l *RateLimiter) Limit(budget cache.Namespace, limit int, subject RateLimitSubject) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		who, err := subject(ctx)
		if err != nil {
			return err
		}

		allowance, err := l.cache.Allow(ctx.Context(), budget.Key(boundSubject(who)), limit, budget.TTL)
		if err != nil {
			logger.Ctx(ctx.Context()).Warn().
				Err(err).
				Str("budget", budget.Prefix).
				Msg("rate limit store failed, letting the request through")

			return ctx.Next()
		}

		if allowance.Allowed {
			return ctx.Next()
		}

		ctx.Set(fiber.HeaderRetryAfter, strconv.FormatInt(retryAfterSeconds(allowance.RetryAfter), 10))

		return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": tooManyRequestsMessage,
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
