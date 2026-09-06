package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

const (
	CredentialAttemptsPerMinute = 10
	MailAttemptsPerHour         = 5

	credentialWindow = time.Minute
	mailWindow       = time.Hour
)

func RateLimitCredentials() fiber.Handler {
	return rateLimitByClientIP(CredentialAttemptsPerMinute, credentialWindow)
}

func RateLimitMail() fiber.Handler {
	return rateLimitByClientIP(MailAttemptsPerHour, mailWindow)
}

func RateLimitCredentialsByAccount(field string) fiber.Handler {
	return rateLimitByKey(CredentialAttemptsPerMinute, credentialWindow, accountKey(field))
}

func RateLimitMailByAccount(field string) fiber.Handler {
	return rateLimitByKey(MailAttemptsPerHour, mailWindow, accountKey(field))
}

func rateLimitByClientIP(attempts int, window time.Duration) fiber.Handler {
	return rateLimitByKey(attempts, window, clientIP)
}

func rateLimitByKey(attempts int, window time.Duration, key func(fiber.Ctx) string) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          attempts,
		Expiration:   window,
		KeyGenerator: key,
		LimitReached: func(ctx fiber.Ctx) error {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "too many requests, please try again later",
			})
		},
	})
}

func accountKey(field string) func(fiber.Ctx) string {
	return func(ctx fiber.Ctx) string {
		var body map[string]json.RawMessage
		if err := json.Unmarshal(ctx.Body(), &body); err != nil {
			return "ip:" + clientIP(ctx)
		}

		var value string
		if err := json.Unmarshal(body[field], &value); err != nil {
			return "ip:" + clientIP(ctx)
		}

		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return "ip:" + clientIP(ctx)
		}

		return field + ":" + value
	}
}

func clientIP(ctx fiber.Ctx) string {
	if ip, ok := ctx.Locals("client_ip").(string); ok && ip != "" {
		return ip
	}

	return ctx.IP()
}
