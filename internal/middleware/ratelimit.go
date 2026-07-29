package middleware

import (
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

func rateLimitByClientIP(attempts int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          attempts,
		Expiration:   window,
		KeyGenerator: clientIP,
		LimitReached: func(ctx fiber.Ctx) error {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "too many requests, please try again later",
			})
		},
	})
}

func clientIP(ctx fiber.Ctx) string {
	if ip, ok := ctx.Locals("client_ip").(string); ok && ip != "" {
		return ip
	}

	return ctx.IP()
}
