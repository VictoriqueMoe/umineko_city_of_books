package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	contentLengthChunked = -1
	contentLengthNone    = -2
)

func RejectPlainTextBodies() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if !hasBody(ctx) {
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

func LimitRequestBodies() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		contentLength := ctx.Request().Header.ContentLength()
		if contentLength > ctx.App().Server().MaxRequestBodySize {
			return fiber.ErrRequestEntityTooLarge
		}

		if contentLength == contentLengthChunked {
			return fiber.ErrLengthRequired
		}

		return ctx.Next()
	}
}

func hasBody(ctx fiber.Ctx) bool {
	contentLength := ctx.Request().Header.ContentLength()

	return contentLength != 0 && contentLength != contentLengthNone
}
