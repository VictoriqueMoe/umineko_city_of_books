package middleware

import (
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func RequireAuth(mgr *session.Manager) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		cookie := ctx.Cookies(session.CookieName)
		if cookie == "" {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication required",
			})
		}

		userID, err := mgr.Validate(ctx.Context(), cookie)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired session",
			})
		}

		ctx.Locals("userID", userID)
		return ctx.Next()
	}
}

func OptionalAuth(mgr *session.Manager) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		cookie := ctx.Cookies(session.CookieName)
		if cookie == "" {
			ctx.Locals("userID", uuid.Nil)
			return ctx.Next()
		}

		userID, err := mgr.Validate(ctx.Context(), cookie)
		if err != nil {
			ctx.Locals("userID", uuid.Nil)
			return ctx.Next()
		}

		ctx.Locals("userID", userID)
		return ctx.Next()
	}
}
