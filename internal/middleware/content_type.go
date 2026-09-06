package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

func RejectPlainTextBodies() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if len(ctx.Body()) == 0 {
			return ctx.Next()
		}

		contentType := strings.ToLower(strings.TrimSpace(ctx.Get(fiber.HeaderContentType)))
		if strings.HasPrefix(contentType, fiber.MIMETextPlain) {
			return ctx.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
				"error": "unsupported content type",
			})
		}

		return ctx.Next()
	}
}
