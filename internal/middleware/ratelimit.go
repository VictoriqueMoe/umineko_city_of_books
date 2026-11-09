package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	RateLimitSubject func(ctx fiber.Ctx) (string, error)

	RateLimiter struct {
		cache       *cache.Manager
		settingsSvc settings.Service
	}
)

const (
	maxSubjectLength    = 192
	hashedSubjectPrefix = "sha256:"
	ipv6SubnetBits      = 64

	tooManyRequestsMessage = "too many requests, please try again later"
)

func NewRateLimiter(cacheManager *cache.Manager, settingsSvc settings.Service) *RateLimiter {
	return &RateLimiter{cache: cacheManager, settingsSvc: settingsSvc}
}

func (l *RateLimiter) Limit(budget cache.Namespace, setting *config.SiteSettingDef, subject RateLimitSubject) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		limit := l.settingsSvc.GetInt(ctx.Context(), setting)
		if limit <= 0 {
			return ctx.Next()
		}

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
	return "ip:" + clientNetwork(ctx), nil
}

func ByAccount(field string) RateLimitSubject {
	return func(ctx fiber.Ctx) (string, error) {
		value, ok := normalisedBodyField(ctx, field)
		if !ok {
			return "account:ip:" + clientNetwork(ctx), nil
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
	if utils.IsMultipart(ctx) {
		return "", false
	}

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

func clientNetwork(ctx fiber.Ctx) string {
	ip := clientIP(ctx)

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}

	addr = addr.Unmap()
	if !addr.Is6() {
		return addr.String()
	}

	network, err := addr.Prefix(ipv6SubnetBits)
	if err != nil {
		return ip
	}

	return network.String()
}

func clientIP(ctx fiber.Ctx) string {
	if ip, ok := ctx.Locals("client_ip").(string); ok && ip != "" {
		return ip
	}

	return ctx.IP()
}
